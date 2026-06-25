// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccServersDataSource exercises the full plugin path against a real
// instance: provider Configure -> SDK client -> GET servers -> state. It runs
// only with TF_ACC=1 and a reachable BridgePort (see the Makefile's `testacc`
// target and scripts/acc-harness.sh). The list is always populated (possibly
// empty), so this passes against a freshly booted instance with no servers.
func TestAccServersDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `data "bridgeport_servers" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.bridgeport_servers.all", "servers.#"),
				),
			},
		},
	})
}
