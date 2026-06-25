// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccServerResource exercises the full managed lifecycle (create -> import ->
// update -> destroy) against a real instance. It runs only with TF_ACC=1 and a
// reachable BridgePort (see the Makefile's `testacc` target). It targets the
// seeded "management" environment that a fresh instance provides.
func TestAccServerResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create + Read.
			{
				Config: providerConfig + `
resource "bridgeport_server" "test" {
  environment = "management"
  name        = "tf-acc-server"
  hostname    = "10.20.0.1"
  tags        = ["acc", "web"]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("bridgeport_server.test", "id"),
					resource.TestCheckResourceAttrSet("bridgeport_server.test", "environment_id"),
					resource.TestCheckResourceAttr("bridgeport_server.test", "name", "tf-acc-server"),
					resource.TestCheckResourceAttr("bridgeport_server.test", "hostname", "10.20.0.1"),
					resource.TestCheckResourceAttr("bridgeport_server.test", "tags.#", "2"),
					resource.TestCheckResourceAttr("bridgeport_server.test", "tags.0", "acc"),
				),
			},
			// Import by natural key.
			{
				ResourceName:      "bridgeport_server.test",
				ImportState:       true,
				ImportStateId:     "management/tf-acc-server",
				ImportStateVerify: true,
			},
			// Update: change hostname and shrink tags.
			{
				Config: providerConfig + `
resource "bridgeport_server" "test" {
  environment = "management"
  name        = "tf-acc-server"
  hostname    = "10.20.0.2"
  tags        = ["acc"]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bridgeport_server.test", "hostname", "10.20.0.2"),
					resource.TestCheckResourceAttr("bridgeport_server.test", "tags.#", "1"),
				),
			},
		},
	})
}
