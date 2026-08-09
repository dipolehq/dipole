/*
Package config loads application configuration from environment variables.

Key fields:
  - Env: "staging" or "production". Drives per-tenant URL generation so staging
    emits <tenant>-<svc>-staging hostnames and clients never leave the staging
    boundary.
  - PlatformDomain: apex domain for per-tenant service hostnames.
  - DevBootstrap: enables auto-tenant creation and the X-Dipole-Tenant-Subdomain
    header for local development. Must be false in deployed environments.
  - AdminSSOTenantID: binds admin SSO login to one tenant. Last remaining
    startup-bound tenant — exists because SSO routes run before authentication
    and the console uses a shared hostname. Do not add new readers.
*/
package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/dipolehq/dipole/cloud/internal/platform"
)

type Config struct {
	DatabaseURL        string
	EncryptionKey      string
	GRPCPort           int
	HTTPPort           int
	LogLevel           string
	Region             string
	Env                string
	PlatformDomain     string
	ConsoleURL         string
	DevBootstrap       bool
	AdminSSOTenantID   string
	GRPCTLSCert        string
	GRPCTLSKey         string
	GeoIPDBPath        string
	ResendAPIKey       string
	EmailFrom          string
	CORSAllowedOrigins string
}

func LoadFromEnv() (*Config, error) {
	cfg := &Config{
		DatabaseURL:        os.Getenv("DIPOLE_DATABASE_URL"),
		EncryptionKey:      os.Getenv("DIPOLE_ENCRYPTION_KEY"),
		GRPCPort:           envInt("DIPOLE_GRPC_PORT", 50051),
		HTTPPort:           envInt("DIPOLE_HTTP_PORT", 8080),
		LogLevel:           envStr("DIPOLE_LOG_LEVEL", "info"),
		Region:             envStr("DIPOLE_REGION", "ap-south-1"),
		Env:                envStr("DIPOLE_ENV", "production"),
		PlatformDomain:     envStr("DIPOLE_PLATFORM_DOMAIN", platform.DefaultDomain),
		ConsoleURL:         envStr("DIPOLE_CONSOLE_URL", "https://console.dipolehq.com"),
		DevBootstrap:       envBool("DIPOLE_DEV_BOOTSTRAP", false),
		AdminSSOTenantID:   os.Getenv("DIPOLE_TENANT_ID"),
		GRPCTLSCert:        os.Getenv("DIPOLE_GRPC_TLS_CERT"),
		GRPCTLSKey:         os.Getenv("DIPOLE_GRPC_TLS_KEY"),
		GeoIPDBPath:        os.Getenv("DIPOLE_GEOIP_DB_PATH"),
		ResendAPIKey:       os.Getenv("DIPOLE_RESEND_API_KEY"),
		EmailFrom:          envStr("DIPOLE_EMAIL_FROM", "Dipole <noreply@dipolehq.com>"),
		CORSAllowedOrigins: os.Getenv("DIPOLE_CORS_ALLOWED_ORIGINS"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DIPOLE_DATABASE_URL is required")
	}
	if c.EncryptionKey == "" {
		return fmt.Errorf("DIPOLE_ENCRYPTION_KEY is required")
	}
	if len(c.EncryptionKey) != 64 {
		return fmt.Errorf("DIPOLE_ENCRYPTION_KEY must be 64 hex characters (32 bytes)")
	}
	if c.Env != "production" && c.Env != "staging" {
		return fmt.Errorf("DIPOLE_ENV must be \"production\" or \"staging\", got %q", c.Env)
	}
	return nil
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
