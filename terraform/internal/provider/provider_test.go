/*
provider_test.go — Acceptance test helpers and provider factory for testing.

Tests require a running Dipole Cloud API. Set DIPOLE_TENANT_ID and DIPOLE_API_TOKEN
env vars to run. Tests are skipped if these are not set.
*/
package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"dipole": providerserver.NewProtocol6WithError(New()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("DIPOLE_TENANT_ID") == "" {
		t.Skip("DIPOLE_TENANT_ID not set, skipping acceptance test")
	}
	if os.Getenv("DIPOLE_API_TOKEN") == "" {
		t.Skip("DIPOLE_API_TOKEN not set, skipping acceptance test")
	}
}
