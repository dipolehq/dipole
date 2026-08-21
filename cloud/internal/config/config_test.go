package config

import (
	"os"
	"testing"
)

func TestLoadFromEnv_MissingDatabaseURL_ReturnsError(t *testing.T) {
	os.Clearenv()
	os.Setenv("DIPOLE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")

	_, err := LoadFromEnv()
	if err == nil {
		t.Fatal("expected error when DIPOLE_DATABASE_URL missing")
	}
}

func TestLoadFromEnv_MissingEncryptionKey_ReturnsError(t *testing.T) {
	os.Clearenv()
	os.Setenv("DIPOLE_DATABASE_URL", "postgres://localhost/test")

	_, err := LoadFromEnv()
	if err == nil {
		t.Fatal("expected error when DIPOLE_ENCRYPTION_KEY missing")
	}
}

func TestLoadFromEnv_InvalidEncryptionKeyLength_ReturnsError(t *testing.T) {
	os.Clearenv()
	os.Setenv("DIPOLE_DATABASE_URL", "postgres://localhost/test")
	os.Setenv("DIPOLE_ENCRYPTION_KEY", "tooshort")

	_, err := LoadFromEnv()
	if err == nil {
		t.Fatal("expected error when DIPOLE_ENCRYPTION_KEY has wrong length")
	}
}

func TestLoadFromEnv_InvalidEnv_ReturnsError(t *testing.T) {
	os.Clearenv()
	os.Setenv("DIPOLE_DATABASE_URL", "postgres://localhost/test")
	os.Setenv("DIPOLE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	os.Setenv("DIPOLE_ENV", "development")

	_, err := LoadFromEnv()
	if err == nil {
		t.Fatal("expected error when DIPOLE_ENV is not production or staging")
	}
}

func TestLoadFromEnv_InvalidEmailFrom_ReturnsError(t *testing.T) {
	os.Clearenv()
	os.Setenv("DIPOLE_DATABASE_URL", "postgres://localhost/test")
	os.Setenv("DIPOLE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	os.Setenv("DIPOLE_EMAIL_FROM", "not-an-email")

	_, err := LoadFromEnv()
	if err == nil {
		t.Fatal("expected error when DIPOLE_EMAIL_FROM is invalid")
	}
}

func TestLoadFromEnv_ValidConfig_Succeeds(t *testing.T) {
	os.Clearenv()
	os.Setenv("DIPOLE_DATABASE_URL", "postgres://localhost/test")
	os.Setenv("DIPOLE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	os.Setenv("DIPOLE_ENV", "production")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.DatabaseURL != "postgres://localhost/test" {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, "postgres://localhost/test")
	}
	if cfg.Env != "production" {
		t.Errorf("Env = %q, want %q", cfg.Env, "production")
	}
}

func TestLoadFromEnv_DefaultFallbacks_Applied(t *testing.T) {
	os.Clearenv()
	os.Setenv("DIPOLE_DATABASE_URL", "postgres://localhost/test")
	os.Setenv("DIPOLE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.HTTPPort != 8080 {
		t.Errorf("HTTPPort = %d, want 8080", cfg.HTTPPort)
	}
	if cfg.GRPCPort != 50051 {
		t.Errorf("GRPCPort = %d, want 50051", cfg.GRPCPort)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.Env != "production" {
		t.Errorf("Env = %q, want production", cfg.Env)
	}
}

func TestLoadFromEnv_CustomPorts_Override(t *testing.T) {
	os.Clearenv()
	os.Setenv("DIPOLE_DATABASE_URL", "postgres://localhost/test")
	os.Setenv("DIPOLE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	os.Setenv("DIPOLE_HTTP_PORT", "9000")
	os.Setenv("DIPOLE_GRPC_PORT", "60000")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.HTTPPort != 9000 {
		t.Errorf("HTTPPort = %d, want 9000", cfg.HTTPPort)
	}
	if cfg.GRPCPort != 60000 {
		t.Errorf("GRPCPort = %d, want 60000", cfg.GRPCPort)
	}
}

func TestLoadFromEnv_DevBootstrap_ParsesBool(t *testing.T) {
	os.Clearenv()
	os.Setenv("DIPOLE_DATABASE_URL", "postgres://localhost/test")
	os.Setenv("DIPOLE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	os.Setenv("DIPOLE_DEV_BOOTSTRAP", "true")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !cfg.DevBootstrap {
		t.Error("expected DevBootstrap to be true")
	}
}

func TestIsValidEmail_ValidEmails_ReturnsTrue(t *testing.T) {
	tests := []string{
		"user@example.com",
		"test@test.co",
		"a@b.c",
		"Dipole <noreply@dipolehq.com>",
	}
	for _, email := range tests {
		if !isValidEmail(email) {
			t.Errorf("isValidEmail(%q) = false, want true", email)
		}
	}
}

func TestIsValidEmail_InvalidEmails_ReturnsFalse(t *testing.T) {
	tests := []string{
		"notanemail",
		"@example.com",
		"user@",
		"",
		"no-at-sign",
	}
	for _, email := range tests {
		if isValidEmail(email) {
			t.Errorf("isValidEmail(%q) = true, want false", email)
		}
	}
}
