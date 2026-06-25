// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccVarResource exercises the full managed lifecycle (create -> import ->
// update -> destroy) against a real instance, targeting the seeded "management"
// environment. Runs only with TF_ACC=1.
func TestAccVarResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "bridgeport_var" "test" {
  environment = "management"
  key         = "TF_ACC_LOG_LEVEL"
  value       = "info"
  description = "acc test"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("bridgeport_var.test", "id"),
					resource.TestCheckResourceAttr("bridgeport_var.test", "key", "TF_ACC_LOG_LEVEL"),
					resource.TestCheckResourceAttr("bridgeport_var.test", "value", "info"),
				),
			},
			{
				ResourceName:      "bridgeport_var.test",
				ImportState:       true,
				ImportStateId:     "management/TF_ACC_LOG_LEVEL",
				ImportStateVerify: true,
			},
			{
				Config: providerConfig + `
resource "bridgeport_var" "test" {
  environment = "management"
  key         = "TF_ACC_LOG_LEVEL"
  value       = "debug"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bridgeport_var.test", "value", "debug"),
				),
			},
		},
	})
}
