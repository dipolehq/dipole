/*
Agent Proxy entry point. Parses configuration, initializes the logger,
starts the explicit proxy listener, and handles graceful shutdown on SIGTERM/SIGINT.
*/
package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/edge/internal/config"
	"github.com/dipolehq/dipole/edge/internal/proxy"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := newLogger(cfg.LogLevel)
	logger.Info("starting agent proxy",
		"tenant_id", cfg.TenantID,
		"port", cfg.HTTPServerPort,
		"sigterm_strategy", cfg.SIGTERMStrategy,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := func(clientConn net.Conn, targetHost string, targetPort int, connID uuid.UUID) {
		// Placeholder: will be replaced by TLS MITM engine in Task 6
		logger.Info("connection received", "host", targetHost, "port", targetPort, "connection_id", connID)
		clientConn.Close()
	}

	listener := proxy.NewExplicitListener(cfg, logger, handler)

	go func() {
		if err := listener.Start(ctx); err != nil {
			logger.Error("listener failed", "error", err)
			cancel()
		}
	}()

	waitForShutdown(cancel, cfg, logger, listener)
}

func waitForShutdown(cancel context.CancelFunc, cfg *config.ProxyConfig, logger *slog.Logger, listener *proxy.ExplicitListener) {
	sigCh := make(chan os.Signal, 1)

	if cfg.SIGTERMStrategy == "sigkill" {
		logger.Info("sigkill strategy: ignoring SIGTERM, awaiting SIGKILL")
		signal.Ignore(syscall.SIGTERM)
		signal.Notify(sigCh, syscall.SIGINT)
	} else {
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	}

	sig := <-sigCh
	logger.Info("received signal, shutting down", "signal", sig.String())
	cancel()
	listener.Shutdown()
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "error":
		lvl = slog.LevelError
	case "warn":
		lvl = slog.LevelWarn
	case "debug", "trace":
		lvl = slog.LevelDebug
	default:
		lvl = slog.LevelInfo
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}
