/* Configuration loading and utility helpers for the agent controller. */
package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type ControllerConfig struct {
	TenantID         string
	DeviceCode       string
	ControllerID     string
	CloudEndpoint    string
	HTTPPort         int
	HTTPPortDisabled bool
	LogLevel         string
	AgentTrustPath   string
	TLSCertPath      string
	TLSKeyPath       string
}

func loadControllerConfig() *ControllerConfig {
	tenantID := os.Getenv("DIPOLE_TENANT_ID")
	if tenantID == "" {
		fmt.Fprintln(os.Stderr, "DIPOLE_TENANT_ID is required")
		os.Exit(1)
	}

	cloudEndpoint := os.Getenv("DIPOLE_CLOUD_ENDPOINT")
	if cloudEndpoint == "" {
		cloudEndpoint = fmt.Sprintf("https://%s.dipolehq.com", tenantID)
	}

	port := 80
	if p := os.Getenv("DIPOLE_HTTP_PORT"); p != "" {
		fmt.Sscanf(p, "%d", &port)
	}

	return &ControllerConfig{
		TenantID:         tenantID,
		DeviceCode:       os.Getenv("DIPOLE_DEVICE_CODE"),
		ControllerID:     os.Getenv("DIPOLE_AGENT_CONTROLLER_ID"),
		CloudEndpoint:    cloudEndpoint,
		HTTPPort:         port,
		HTTPPortDisabled: os.Getenv("DIPOLE_HTTP_PORT_DISABLED") == "true",
		LogLevel:         envOr("DIPOLE_LOG_LEVEL", "information"),
		AgentTrustPath:   os.Getenv("AGENT_TRUST_PATH"),
		TLSCertPath:      os.Getenv("TLS_PEM_PATH"),
		TLSKeyPath:       os.Getenv("TLS_KEY_PATH"),
	}
}

func buildCloudHTTPClient(trustPath string) *http.Client {
	transport := &http.Transport{}

	if trustPath != "" {
		pem, err := os.ReadFile(trustPath)
		if err == nil {
			pool := x509.NewCertPool()
			if pool.AppendCertsFromPEM(pem) {
				transport.TLSClientConfig = &tls.Config{RootCAs: pool}
			}
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func newControllerLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "fatal", "error":
		lvl = slog.LevelError
	case "warning":
		lvl = slog.LevelWarn
	case "debug", "verbose":
		lvl = slog.LevelDebug
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}
