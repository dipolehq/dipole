/*
Terraform Provider for Dipole — manages workload IAM resources as code.

Registers the provider with terraform-plugin-framework and starts the gRPC server
that Terraform communicates with during plan/apply.
*/
package main

import (
	"context"
	"log"

	"github.com/dipolehq/terraform-provider-dipole/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/dipolehq/dipole",
	}

	err := providerserver.Serve(context.Background(), provider.New, opts)
	if err != nil {
		log.Fatal(err)
	}
}
