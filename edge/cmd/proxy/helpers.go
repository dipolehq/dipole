/* Utility functions for the agent proxy. */
package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/dipolehq/dipole/edge/internal/cloudclient"
	"github.com/dipolehq/dipole/edge/internal/config"
	"github.com/dipolehq/dipole/edge/internal/metrics"
	"github.com/dipolehq/dipole/edge/internal/proxy"
	dipoletls "github.com/dipolehq/dipole/edge/internal/tls"
)

// buildTrustAnchor parses the tenant's Root CA public certificate returned
// by Cloud's Register RPC. The private key never leaves Cloud: leaf
// certificates are signed remotely per-hostname via the SignLeafCertificate
// RPC (see dipoletls.LeafSigner). There is no local/dev fallback CA — if
// registration did not succeed or Cloud did not return a CA certificate,
// TLS Decrypt is unavailable and this fails closed, exactly like every
// other proxy capability that requires a live Cloud connection (policy
// evaluation, credential injection).
func buildTrustAnchor(regResult *cloudclient.RegistrationResult) (*dipoletls.TrustAnchor, error) {
	if regResult == nil || !regResult.Success || regResult.CACertPEM == "" {
		return nil, fmt.Errorf("no tenant CA certificate available: cloud registration did not succeed")
	}
	return dipoletls.NewTrustAnchor([]byte(regResult.CACertPEM))
}

// writeCAToSharedVolume writes the tenant Root CA's public certificate to a
// path other containers/processes on the host can mount and trust (e.g. a
// sidecar's system trust store). If this fails, TLS Decrypt still signs
// leaf certs server-side, but clients on this host were never given the CA
// to trust them, so every intercepted connection will fail certificate
// validation. That failure mode is silent from the proxy's own perspective
// (it doesn't touch any connection path), so it is escalated to Error and
// counted in a metric rather than only logged at Warn — otherwise an
// operator has no signal to distinguish "TLS Decrypt disabled" from
// "TLS Decrypt enabled but broken for every client."
func writeCAToSharedVolume(anchor *dipoletls.TrustAnchor, logger *slog.Logger) {
	const caPath = "/dipole/ca/ca.pem"

	if err := os.MkdirAll("/dipole/ca", 0755); err != nil {
		logger.Error("cannot create CA directory for trust injection: TLS Decrypt clients will fail certificate validation", "path", caPath, "error", err)
		metrics.TLSCATrustInjectionFailures.Inc()
		return
	}

	if err := os.WriteFile(caPath, anchor.CACertPEM(), 0644); err != nil {
		logger.Error("cannot write CA cert for trust injection: TLS Decrypt clients will fail certificate validation", "path", caPath, "error", err)
		metrics.TLSCATrustInjectionFailures.Inc()
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
