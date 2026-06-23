/*
transparent.go implements the iptables-redirect transparent proxy listener.

TransparentListener accepts connections on port 15001 (redirected by iptables),
resolves the original destination via SO_ORIGINAL_DST, and dispatches to the
same ConnectionHandler as the explicit listener. Fail-closed: if the original
destination cannot be determined, the connection is dropped. Linux-only.
*/
package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/edge/internal/config"
)

const transparentPort = 15001

type TransparentListener struct {
	config   *config.ProxyConfig
	logger   *slog.Logger
	handler  ConnectionHandler
	listener net.Listener

	activeConns atomic.Int64
	maxConns    int64

	shutdownOnce sync.Once
	shutdownCh   chan struct{}
	wg           sync.WaitGroup

	allowedHosts map[string]bool
}

func NewTransparentListener(cfg *config.ProxyConfig, logger *slog.Logger, handler ConnectionHandler) *TransparentListener {
	allowed := make(map[string]bool)
	for _, h := range cfg.SteeringAllowedHosts {
		host := strings.TrimSpace(h)
		if host != "" {
			allowed[host] = true
		}
	}

	return &TransparentListener{
		config:       cfg,
		logger:       logger,
		handler:      handler,
		maxConns:     10000,
		shutdownCh:   make(chan struct{}),
		allowedHosts: allowed,
	}
}

func (t *TransparentListener) Start(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", transparentPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("transparent listen on %s: %w", addr, err)
	}
	t.listener = ln
	t.logger.Info("transparent listener started", "addr", addr)

	go func() {
		<-ctx.Done()
		t.Shutdown()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-t.shutdownCh:
				return nil
			default:
				t.logger.Error("transparent accept failed", "error", err)
				continue
			}
		}

		if t.activeConns.Load() >= t.maxConns {
			conn.Close()
			continue
		}

		t.activeConns.Add(1)
		t.wg.Add(1)
		go t.handleConn(conn)
	}
}

func (t *TransparentListener) Shutdown() {
	t.shutdownOnce.Do(func() {
		close(t.shutdownCh)
		if t.listener != nil {
			t.listener.Close()
		}

		done := make(chan struct{})
		go func() {
			t.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(30 * time.Second):
			t.logger.Warn("transparent shutdown deadline exceeded")
		}
	})
}

func (t *TransparentListener) handleConn(conn net.Conn) {
	defer func() {
		t.activeConns.Add(-1)
		t.wg.Done()
	}()

	connID := uuid.New()

	host, port, err := getOriginalDst(conn)
	if err != nil {
		t.logger.Error("cannot determine original destination", "error", err, "connection_id", connID)
		conn.Close()
		return
	}

	// Selective steering: if allowed hosts configured, check if this host is in the list
	if len(t.allowedHosts) > 0 && !t.allowedHosts[host] {
		// Not in allowed list — passthrough without inspection
		t.passthroughDirect(conn, host, port, connID)
		return
	}

	t.logger.Info("transparent connection intercepted", "host", host, "port", port, "connection_id", connID)
	t.handler(conn, host, port, connID)
}

func (t *TransparentListener) passthroughDirect(clientConn net.Conn, host string, port int, connID uuid.UUID) {
	serverConn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 10*time.Second)
	if err != nil {
		t.logger.Debug("passthrough dial failed", "host", host, "port", port, "error", err, "connection_id", connID)
		clientConn.Close()
		return
	}

	done := make(chan struct{}, 2)
	go func() { copyStream(serverConn, clientConn); done <- struct{}{} }()
	go func() { copyStream(clientConn, serverConn); done <- struct{}{} }()
	<-done

	clientConn.Close()
	serverConn.Close()
}

func copyStream(dst, src net.Conn) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			dst.Write(buf[:n])
		}
		if err != nil {
			return
		}
	}
}
