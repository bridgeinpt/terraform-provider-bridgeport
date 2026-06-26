// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccConfigFragmentResource exercises create -> import -> update -> destroy
// against the seeded "management" environment. Runs only with TF_ACC=1.
func TestAccConfigFragmentResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "bridgeport_config_fragment" "test" {
  environment = "management"
  name        = "tf-acc-fragment"
  content     = "hello = world\n"
  description = "acc test"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("bridgeport_config_fragment.test", "id"),
					resource.TestCheckResourceAttr("bridgeport_config_fragment.test", "name", "tf-acc-fragment"),
					resource.TestCheckResourceAttr("bridgeport_config_fragment.test", "content", "hello = world\n"),
				),
			},
			{
				ResourceName:      "bridgeport_config_fragment.test",
				ImportState:       true,
				ImportStateId:     "management/tf-acc-fragment",
				ImportStateVerify: true,
			},
			{
				Config: providerConfig + `
resource "bridgeport_config_fragment" "test" {
  environment = "management"
  name        = "tf-acc-fragment"
  content     = "hello = mars\n"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bridgeport_config_fragment.test", "content", "hello = mars\n"),
				),
			},
		},
	})
}
