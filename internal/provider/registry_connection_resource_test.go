// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRegistryConnectionResource exercises create -> import -> update ->
// destroy against the seeded "management" environment. Runs only with TF_ACC=1.
func TestAccRegistryConnectionResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "bridgeport_registry_connection" "test" {
  environment      = "management"
  name             = "tf-acc-registry"
  type             = "generic"
  registry_url     = "registry.example.com"
  username         = "robot"
  password_wo      = "s3cr3t"
  password_wo_version = "1"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("bridgeport_registry_connection.test", "id"),
					resource.TestCheckResourceAttr("bridgeport_registry_connection.test", "name", "tf-acc-registry"),
					resource.TestCheckResourceAttr("bridgeport_registry_connection.test", "type", "generic"),
					resource.TestCheckResourceAttr("bridgeport_registry_connection.test", "has_password", "true"),
					resource.TestCheckNoResourceAttr("bridgeport_registry_connection.test", "password_wo"),
				),
			},
			{
				ResourceName:      "bridgeport_registry_connection.test",
				ImportState:       true,
				ImportStateId:     "management/tf-acc-registry",
				ImportStateVerify: true,
				// Only the write-only credentials can't be recovered on import.
				ImportStateVerifyIgnore: []string{
					"token_wo", "token_wo_version", "password_wo", "password_wo_version",
				},
			},
			{
				Config: providerConfig + `
resource "bridgeport_registry_connection" "test" {
  environment      = "management"
  name             = "tf-acc-registry"
  type             = "generic"
  registry_url     = "registry2.example.com"
  username         = "robot"
  password_wo      = "s3cr3t"
  password_wo_version = "1"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bridgeport_registry_connection.test", "registry_url", "registry2.example.com"),
				),
			},
		},
	})
}
