// geoip_condition_test.go tests GeoIP condition fail-closed behavior when the database is unavailable or the IP is invalid.
package conditions

import (
	"testing"
)

func TestGeoIP_NoDB_FailsClosed(t *testing.T) {
	checker := NewGeoIPConditionChecker("", nil)
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
	checker := NewGeoIPConditionChecker("", nil)
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
	checker := NewGeoIPConditionChecker("", nil)
	cfg := &GeoIPConditionConfig{}

	passed, _, _ := checker.Check("8.8.8.8", cfg)
	if passed {
		t.Error("expected fail-closed with nil DB")
	}
}
