package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/dipolehq/dipole/cloud/internal/platform"
)

type Config struct {
	DatabaseURL   string
	EncryptionKey string
	GRPCPort      int
	HTTPPort      int
	LogLevel      string
	Region        string

	// PlatformDomain is the apex domain that per-tenant service hostnames hang
	// off (<tenant>-mcp.<domain>, <tenant>-id.<domain>, ...). Both URL
	// generation and request-hostname parsing derive from it.
	PlatformDomain string

	// ConsoleURL is the admin console base URL for links we put in outbound
	// email (password reset). Must differ per environment — a hardcoded value
	// sends staging users to production.
	ConsoleURL string

	// DevBootstrap enables local-development conveniences that must NEVER be on
	// in a deployed environment:
	//   - auto-creating a tenant + admin user on first boot
	//   - honouring the X-Dipole-Tenant-Subdomain header to select a tenant
	// Both would otherwise let requests be served as an arbitrary tenant, since
	// the MCP/OAuth routes are unauthenticated by design.
	DevBootstrap bool

	// AdminSSOTenantID binds admin-console SSO login to one tenant. It is the
	// LAST startup-bound tenant in the process — every other component now
	// resolves its tenant from the request hostname.
	//
	// It survives because admin SSO genuinely cannot use either mechanism: the
	// SSO routes run BEFORE authentication so there is no JWT to read a tenant
	// from, and the console is served from one shared hostname rather than a
	// per-tenant subdomain, so the hostname cannot supply one either. Resolving
	// that needs a product decision (identifier-first login, or a per-tenant
	// console hostname) — see the admin-console SSO entry in later.md.
	//
	// Do NOT add new readers. A startup-bound tenant means every request is
	// served as that tenant regardless of which hostname it arrived on. Admin SSO
	// is not currently offered to customers; the console uses email + password.
	AdminSSOTenantID string

	GRPCTLSCert        string
	GRPCTLSKey         string
	GeoIPDBPath        string
	ResendAPIKey       string
	EmailFrom          string
	CORSAllowedOrigins string
}

func LoadFromEnv() (*Config, error) {
	cfg := &Config{
		DatabaseURL:    os.Getenv("DIPOLE_DATABASE_URL"),
		EncryptionKey:  os.Getenv("DIPOLE_ENCRYPTION_KEY"),
		GRPCPort:       envInt("DIPOLE_GRPC_PORT", 50051),
		HTTPPort:       envInt("DIPOLE_HTTP_PORT", 8080),
		LogLevel:       envStr("DIPOLE_LOG_LEVEL", "info"),
		Region:         envStr("DIPOLE_REGION", "ap-south-1"),
		PlatformDomain: envStr("DIPOLE_PLATFORM_DOMAIN", platform.DefaultDomain),
		ConsoleURL:     envStr("DIPOLE_CONSOLE_URL", "https://console.dipolehq.com"),
		DevBootstrap:   envBool("DIPOLE_DEV_BOOTSTRAP", false),

		// Admin SSO only — see struct comment. The env var name is unchanged so
		// deployments don't need editing, but it no longer affects anything
		// except whether admin-console SSO routes are registered.
		// DIPOLE_TENANT_SUBDOMAIN is no longer read by anything.
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
