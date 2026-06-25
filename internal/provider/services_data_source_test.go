// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccServicesDataSource exercises the full plugin path against a real
// instance: provider Configure -> SDK client -> GET services -> state. It runs
// only with TF_ACC=1 and a reachable BridgePort (see the Makefile's `testacc`
// target and scripts/acc-harness.sh). The unfiltered list is always populated
// (possibly empty), so this passes against a freshly booted instance with no
// services.
func TestAccServicesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `data "bridgeport_services" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.bridgeport_services.all", "services.#"),
				),
			},
		},
	})
}
