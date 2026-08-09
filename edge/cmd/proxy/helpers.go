/* Utility functions for the agent proxy. */
package main

import (
	"bufio"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/dipolehq/dipole/edge/internal/cloudclient"
	"github.com/dipolehq/dipole/edge/internal/config"
	"github.com/dipolehq/dipole/edge/internal/proxy"
	dipoletls "github.com/dipolehq/dipole/edge/internal/tls"
)

func buildCertificateAuthority(regResult *cloudclient.RegistrationResult, logger *slog.Logger) (*dipoletls.CertificateAuthority, error) {
	if regResult != nil && regResult.CACertPEM != "" && regResult.CAKeyPEM != "" {
		ca, err := dipoletls.LoadCertificateAuthority([]byte(regResult.CACertPEM), []byte(regResult.CAKeyPEM))
		if err != nil {
			logger.Error("failed to load tenant CA from cloud, falling back to local CA", "error", err)
			return dipoletls.NewCertificateAuthority()
		}
		logger.Info("using tenant CA from cloud")
		return ca, nil
	}

	logger.Info("generating local CA (no tenant CA from cloud)")
	return dipoletls.NewCertificateAuthority()
}

func writeCAToSharedVolume(ca *dipoletls.CertificateAuthority, logger *slog.Logger) {
	const caPath = "/dipole/ca/ca.pem"

	if err := os.MkdirAll("/dipole/ca", 0755); err != nil {
		logger.Warn("cannot create CA directory for trust injection", "path", caPath, "error", err)
		return
	}

	if err := os.WriteFile(caPath, ca.CACertPEM(), 0644); err != nil {
		logger.Warn("cannot write CA cert for trust injection", "path", caPath, "error", err)
		return
	}

	logger.Info("CA certificate written for trust injection", "path", caPath)
}

func waitForShutdown(cancel func(), cfg *config.ProxyConfig, logger *slog.Logger, listener *proxy.ExplicitListener) {
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

type bufferedConn struct {
	r *bufio.Reader
	net.Conn
}

func newBufferedConn(r *bufio.Reader, conn net.Conn) *bufferedConn {
	return &bufferedConn{r: r, Conn: conn}
}

func (bc *bufferedConn) Read(b []byte) (int, error) {
	return bc.r.Read(b)
}
