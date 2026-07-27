package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL     string
	EncryptionKey   string
	GRPCPort        int
	HTTPPort        int
	LogLevel        string
	Region          string
	TenantID        string
	TenantSubdomain string
	GRPCTLSCert     string
	GRPCTLSKey      string
}

func LoadFromEnv() (*Config, error) {
	cfg := &Config{
		DatabaseURL:     os.Getenv("DIPOLE_DATABASE_URL"),
		EncryptionKey:   os.Getenv("DIPOLE_ENCRYPTION_KEY"),
		GRPCPort:        envInt("DIPOLE_GRPC_PORT", 50051),
		HTTPPort:        envInt("DIPOLE_HTTP_PORT", 8080),
		LogLevel:        envStr("DIPOLE_LOG_LEVEL", "info"),
		Region:          envStr("DIPOLE_REGION", "ap-south-1"),
		TenantID:        os.Getenv("DIPOLE_TENANT_ID"),
		TenantSubdomain: envStr("DIPOLE_TENANT_SUBDOMAIN", ""),
		GRPCTLSCert:     os.Getenv("DIPOLE_GRPC_TLS_CERT"),
		GRPCTLSKey:      os.Getenv("DIPOLE_GRPC_TLS_KEY"),
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
