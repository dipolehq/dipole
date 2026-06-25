package conditions

import (
	"testing"
)

func TestGeoIP_NoDB_FailsClosed(t *testing.T) {
	checker := NewGeoIPConditionChecker("")
	cfg := &GeoIPConditionConfig{Countries: []string{"IN"}}

	passed, reason, err := checker.Check("1.2.3.4", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if passed {
		t.Error("expected fail-closed when DB not loaded")
	}
	if reason == "" {
		t.Error("expected failure reason")
	}
}

func TestGeoIP_InvalidIP_Fails(t *testing.T) {
	// Even with a valid DB, invalid IP should fail
	checker := NewGeoIPConditionChecker("")
	cfg := &GeoIPConditionConfig{Countries: []string{"US"}}

	passed, _, err := checker.Check("not-an-ip", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if passed {
		t.Error("expected fail for invalid IP")
	}
}

func TestGeoIP_EmptyConfig_Passes(t *testing.T) {
	// No countries or subdivisions configured = no restriction
	// But DB is nil → fail-closed
	checker := NewGeoIPConditionChecker("")
	cfg := &GeoIPConditionConfig{}

	passed, _, _ := checker.Check("8.8.8.8", cfg)
	// With nil DB, always fails closed
	if passed {
		t.Error("expected fail-closed with nil DB")
	}
}
