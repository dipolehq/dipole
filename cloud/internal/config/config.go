// Package config loads application configuration from environment variables.
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
	ProxySecret        string
	ConsoleURL         string
	DevBootstrap       bool
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
		ProxySecret:        os.Getenv("DIPOLE_PROXY_SECRET"),
		ConsoleURL:         envStr("DIPOLE_CONSOLE_URL", "https://console.dipolehq.com"),
		DevBootstrap:       envBool("DIPOLE_DEV_BOOTSTRAP", false),
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
	if c.EmailFrom != "" {
		if !isValidEmail(c.EmailFrom) {
			return fmt.Errorf("DIPOLE_EMAIL_FROM must be a valid email address, got %q", c.EmailFrom)
		}
	}
	return nil
}

func isValidEmail(s string) bool {
	parts := splitEmail(s)
	if len(parts) != 2 {
		return false
	}
	if parts[0] == "" || parts[1] == "" {
		return false
	}
	return true
}

func splitEmail(email string) []string {
	if idx := lastIndex(email, '@'); idx != -1 {
		return []string{email[:idx], email[idx+1:]}
	}
	return []string{email}
}

func lastIndex(s string, ch byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ch {
			return i
		}
	}
	return -1
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
