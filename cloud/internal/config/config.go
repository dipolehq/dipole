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

	// DEPRECATED — remove once the MCP Identity Gateway resolves its tenant
	// per request (progress.md Fix 11). These are the last consumers.
	//
	// Everything else now resolves the tenant from the request hostname. The
	// Gateway still binds one tenant at construction (its downstream MCP client
	// map, event sink, and token audience are all built once at startup), so it
	// cannot be wired up without a tenant to bind to. Do NOT add new readers of
	// these fields — a startup-bound tenant means every request is served as
	// that tenant regardless of which hostname it arrived on.
	GatewayTenantID        string
	GatewayTenantSubdomain string

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

		// DEPRECATED — see struct comment. Pending Fix 11.
		GatewayTenantID:        os.Getenv("DIPOLE_TENANT_ID"),
		GatewayTenantSubdomain: envStr("DIPOLE_TENANT_SUBDOMAIN", ""),
		GRPCTLSCert:            os.Getenv("DIPOLE_GRPC_TLS_CERT"),
		GRPCTLSKey:             os.Getenv("DIPOLE_GRPC_TLS_KEY"),
		GeoIPDBPath:            os.Getenv("DIPOLE_GEOIP_DB_PATH"),
		ResendAPIKey:           os.Getenv("DIPOLE_RESEND_API_KEY"),
		EmailFrom:              envStr("DIPOLE_EMAIL_FROM", "Dipole <noreply@dipolehq.com>"),
		CORSAllowedOrigins:     os.Getenv("DIPOLE_CORS_ALLOWED_ORIGINS"),
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
