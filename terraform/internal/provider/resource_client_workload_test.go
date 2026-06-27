/*
resource_client_workload_test.go — Acceptance tests for dipole_client_workload.

Tests CRUD lifecycle, update, import, and destroy against a live API.
*/
package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccClientWorkload_basic(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "dipole_client_workload" "test" {
  name = "Test Client Workload"
  identifiers = [
    { type = "k8s_namespace", value = "default" }
  ]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("dipole_client_workload.test", "id"),
					resource.TestCheckResourceAttr("dipole_client_workload.test", "name", "Test Client Workload"),
					resource.TestCheckResourceAttr("dipole_client_workload.test", "is_active", "true"),
				),
			},
			{
				Config: `
resource "dipole_client_workload" "test" {
  name = "Updated Client Workload"
  identifiers = [
    { type = "k8s_namespace", value = "production" }
  ]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("dipole_client_workload.test", "name", "Updated Client Workload"),
				),
			},
			{
				ResourceName:      "dipole_client_workload.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
