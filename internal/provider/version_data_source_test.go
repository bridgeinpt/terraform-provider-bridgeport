// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccVersionDataSource exercises the full plugin path against a real
// instance: provider Configure -> SDK client -> GET /health -> state. It runs
// only with TF_ACC=1 and a reachable BridgePort (see the Makefile's `testacc`
// target and scripts/acc-harness.sh). /health is unauthenticated and always
// reports a version, so this passes against a freshly booted instance.
func TestAccVersionDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `data "bridgeport_version" "this" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.bridgeport_version.this", "version"),
					resource.TestCheckResourceAttr("data.bridgeport_version.this", "status", "ok"),
				),
			},
		},
	})
}
