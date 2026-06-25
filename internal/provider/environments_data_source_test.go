// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccEnvironmentsDataSource exercises the full plugin path against a real
// instance: provider Configure -> SDK client -> GET /api/environments -> state.
// It runs only with TF_ACC=1 and a reachable BridgePort (see the Makefile's
// `testacc` target and scripts/acc-harness.sh).
func TestAccEnvironmentsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `data "bridgeport_environments" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// The list attribute is always populated (possibly empty),
					// which proves the provider authenticated and the read path
					// round-tripped against the live API.
					resource.TestCheckResourceAttrSet("data.bridgeport_environments.all", "environments.#"),
				),
			},
		},
	})
}
