/*
Package proxy implements the Agent Proxy's network listeners.

ExplicitListener accepts HTTP CONNECT (for HTTPS tunneling) and plain HTTP
proxy requests on a configurable port. Each connection gets a UUID, is subject
to an idle timeout and a global connection limit, and is dispatched to a
ConnectionHandler callback for protocol-specific handling.
*/
package proxy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/edge/internal/config"
)

type ConnectionHandler func(clientConn net.Conn, targetHost string, targetPort int, connID uuid.UUID)

type ExplicitListener struct {
	config   *config.ProxyConfig
	logger   *slog.Logger
	handler  ConnectionHandler
	listener net.Listener

	activeConns atomic.Int64
	maxConns    int64

	shutdownOnce sync.Once
	shutdownCh   chan struct{}
	wg           sync.WaitGroup
}

func NewExplicitListener(cfg *config.ProxyConfig, logger *slog.Logger, handler ConnectionHandler) *ExplicitListener {
	return &ExplicitListener{
		config:     cfg,
		logger:     logger,
		handler:    handler,
		maxConns:   10000,
		shutdownCh: make(chan struct{}),
	}
}

func (l *ExplicitListener) Start(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", l.config.HTTPServerPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	l.listener = ln
	l.logger.Info("explicit listener started", "addr", addr)

	go func() {
		<-ctx.Done()
		l.Shutdown()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-l.shutdownCh:
				return nil
			default:
				l.logger.Error("accept failed", "error", err)
				continue
			}
		}

		if l.activeConns.Load() >= l.maxConns {
			conn.Write([]byte("HTTP/1.1 503 Service Unavailable\r\n\r\n"))
			conn.Close()
			continue
		}

		l.activeConns.Add(1)
		l.wg.Add(1)
		go l.handleConn(conn)
	}
}

func (l *ExplicitListener) Shutdown() {
	l.shutdownOnce.Do(func() {
		close(l.shutdownCh)
		if l.listener != nil {
			l.listener.Close()
		}

		done := make(chan struct{})
		go func() {
			l.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(30 * time.Second):
			l.logger.Warn("shutdown deadline exceeded, forcing close")
		}
	})
}

func (l *ExplicitListener) ActiveConnections() int64 {
	return l.activeConns.Load()
}

func (l *ExplicitListener) handleConn(conn net.Conn) {
	defer func() {
		l.activeConns.Add(-1)
		l.wg.Done()
	}()

	connID := uuid.New()
	idleTimeout := time.Duration(l.config.HTTPIdleTimeoutSecs) * time.Second
	conn.SetDeadline(time.Now().Add(idleTimeout))

	reader := bufio.NewReader(conn)
	reqLine, err := reader.ReadString('\n')
	if err != nil {
		conn.Close()
		return
	}

	parts := strings.Fields(reqLine)
	if len(parts) < 3 {
		conn.Close()
		return
	}

	method := parts[0]
	target := parts[1]

	if method == "CONNECT" {
		l.handleConnect(conn, reader, target, connID)
	} else {
		l.handlePlainHTTP(conn, reader, method, target, parts[2], connID)
	}
}

func (l *ExplicitListener) handleConnect(conn net.Conn, reader *bufio.Reader, target string, connID uuid.UUID) {
	host, port, err := parseHostPort(target, 443)
	if err != nil {
		l.logger.Warn("invalid CONNECT target", "target", target, "error", err, "connection_id", connID)
		conn.Close()
		return
	}

	// Consume remaining headers until empty line
	for {
		line, err := reader.ReadString('\n')
		if err != nil || strings.TrimSpace(line) == "" {
			break
		}
	}

	_, err = conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		conn.Close()
		return
	}

	l.logger.Info("CONNECT tunnel established", "host", host, "port", port, "connection_id", connID)
	l.handler(conn, host, port, connID)
}

func (l *ExplicitListener) handlePlainHTTP(conn net.Conn, reader *bufio.Reader, method, rawURL, proto string, connID uuid.UUID) {
	defer conn.Close()

	host, port, path, err := parseAbsoluteURL(rawURL)
	if err != nil {
		l.logger.Warn("invalid proxy request URL", "url", rawURL, "error", err, "connection_id", connID)
		return
	}

	serverConn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), 10*time.Second)
	if err != nil {
		l.logger.Error("failed to connect to target", "host", host, "port", port, "error", err, "connection_id", connID)
		return
	}
	defer serverConn.Close()

	// Write rewritten request line
	reqLine := fmt.Sprintf("%s %s %s\r\n", method, path, proto)
	if _, err := serverConn.Write([]byte(reqLine)); err != nil {
		return
	}

	// Forward remaining headers and body from the buffered reader
	// The reader has everything after the first line already buffered
	done := make(chan struct{}, 2)
	go func() {
		io.Copy(serverConn, reader)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(conn, serverConn)
		done <- struct{}{}
	}()

	l.logger.Info("plain HTTP forwarded", "method", method, "host", host, "path", path, "connection_id", connID)
	<-done
}

func parseHostPort(target string, defaultPort int) (string, int, error) {
	host, portStr, err := net.SplitHostPort(target)
	if err != nil {
		// No port specified, use default
		return target, defaultPort, nil
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port %q: %w", portStr, err)
	}
	return host, port, nil
}

func parseAbsoluteURL(rawURL string) (host string, port int, path string, err error) {
	req, err := http.ReadRequest(bufio.NewReader(strings.NewReader("GET " + rawURL + " HTTP/1.1\r\nHost: x\r\n\r\n")))
	if err != nil {
		return "", 0, "", fmt.Errorf("parse URL %q: %w", rawURL, err)
	}

	host = req.URL.Hostname()
	path = req.URL.RequestURI()
	if host == "" {
		return "", 0, "", fmt.Errorf("no host in URL %q", rawURL)
	}

	port = 80
	if req.URL.Scheme == "https" {
		port = 443
	}
	if p := req.URL.Port(); p != "" {
		port, _ = strconv.Atoi(p)
	}

	return host, port, path, nil
}
