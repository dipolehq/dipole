/*
resource_access_policy_test.go — Acceptance test for full policy lifecycle.

Creates CW + SW + TP + CP + Policy, verifies all linked correctly, then destroys.
*/
package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAccessPolicy_fullStack(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "dipole_client_workload" "test" {
  name = "Policy Test Client"
  identifiers = [
    { type = "k8s_namespace", value = "test-ns" }
  ]
}

resource "dipole_server_workload" "test" {
  name         = "Policy Test Server"
  host         = "api.test.com"
  port         = 443
  app_protocol = "HTTP"
}

resource "dipole_trust_provider" "test" {
  name = "Policy Test TP"
  type = "kubernetes_service_account"
  config_json = jsonencode({
    oidc_endpoint = "https://oidc.example.com"
  })
}

resource "dipole_credential_provider" "test" {
  name = "Policy Test CP"
  type = "api_key"
  config_json = jsonencode({
    api_key = "test-key-12345"
  })
}

resource "dipole_access_policy" "test" {
  name                   = "Full Stack Policy"
  is_active              = true
  client_workload_id     = dipole_client_workload.test.id
  server_workload_id     = dipole_server_workload.test.id
  trust_provider_ids     = [dipole_trust_provider.test.id]
  credential_provider_id = dipole_credential_provider.test.id
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("dipole_access_policy.test", "id"),
					resource.TestCheckResourceAttr("dipole_access_policy.test", "name", "Full Stack Policy"),
					resource.TestCheckResourceAttr("dipole_access_policy.test", "is_active", "true"),
					resource.TestCheckResourceAttrPair("dipole_access_policy.test", "client_workload_id", "dipole_client_workload.test", "id"),
					resource.TestCheckResourceAttrPair("dipole_access_policy.test", "server_workload_id", "dipole_server_workload.test", "id"),
					resource.TestCheckResourceAttrPair("dipole_access_policy.test", "credential_provider_id", "dipole_credential_provider.test", "id"),
				),
			},
		},
	})
}
