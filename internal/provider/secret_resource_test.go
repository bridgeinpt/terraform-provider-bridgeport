// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccSecretResource exercises the full managed lifecycle for a write-only
// secret (create -> import -> rotate -> destroy) against the seeded "management"
// environment. Runs only with TF_ACC=1. The write-only value requires the test
// harness to use Terraform 1.11+.
func TestAccSecretResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "bridgeport_secret" "test" {
  environment      = "management"
  key              = "TF_ACC_SECRET"
  value_wo         = "s3cr3t-v1"
  value_wo_version = "1"
  description      = "acc test"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("bridgeport_secret.test", "id"),
					resource.TestCheckResourceAttr("bridgeport_secret.test", "key", "TF_ACC_SECRET"),
					resource.TestCheckResourceAttr("bridgeport_secret.test", "never_reveal", "false"),
					// The value is write-only: it must never be stored in state.
					resource.TestCheckNoResourceAttr("bridgeport_secret.test", "value_wo"),
				),
			},
			{
				ResourceName:      "bridgeport_secret.test",
				ImportState:       true,
				ImportStateId:     "management/TF_ACC_SECRET",
				ImportStateVerify: true,
				// The value and its version trigger can't be recovered on import.
				ImportStateVerifyIgnore: []string{"value_wo", "value_wo_version"},
			},
			{
				// Rotate: new value + bumped version.
				Config: providerConfig + `
resource "bridgeport_secret" "test" {
  environment      = "management"
  key              = "TF_ACC_SECRET"
  value_wo         = "s3cr3t-v2"
  value_wo_version = "2"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bridgeport_secret.test", "value_wo_version", "2"),
					resource.TestCheckNoResourceAttr("bridgeport_secret.test", "value_wo"),
				),
			},
		},
	})
}
