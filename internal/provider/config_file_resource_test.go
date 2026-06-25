// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccConfigFileResource exercises create -> import -> update -> destroy
// against the seeded "management" environment, including a fragment include.
// Runs only with TF_ACC=1.
func TestAccConfigFileResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "bridgeport_config_fragment" "inc" {
  environment = "management"
  name        = "tf-acc-file-fragment"
  content     = "# header\n"
}

resource "bridgeport_config_file" "test" {
  environment  = "management"
  name         = "tf-acc-file"
  filename     = "app.conf"
  language     = "ini"
  content      = "key = value\n"
  fragment_ids = [bridgeport_config_fragment.inc.id]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("bridgeport_config_file.test", "id"),
					resource.TestCheckResourceAttr("bridgeport_config_file.test", "name", "tf-acc-file"),
					resource.TestCheckResourceAttr("bridgeport_config_file.test", "filename", "app.conf"),
					resource.TestCheckResourceAttr("bridgeport_config_file.test", "content", "key = value\n"),
					resource.TestCheckResourceAttr("bridgeport_config_file.test", "fragment_ids.#", "1"),
				),
			},
			{
				ResourceName:      "bridgeport_config_file.test",
				ImportState:       true,
				ImportStateId:     "management/tf-acc-file",
				ImportStateVerify: true,
				// Fragment associations are not returned by the API on read.
				ImportStateVerifyIgnore: []string{"fragment_ids"},
			},
			{
				Config: providerConfig + `
resource "bridgeport_config_fragment" "inc" {
  environment = "management"
  name        = "tf-acc-file-fragment"
  content     = "# header\n"
}

resource "bridgeport_config_file" "test" {
  environment  = "management"
  name         = "tf-acc-file"
  filename     = "app.conf"
  language     = "ini"
  content      = "key = changed\n"
  fragment_ids = [bridgeport_config_fragment.inc.id]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bridgeport_config_file.test", "content", "key = changed\n"),
				),
			},
		},
	})
}
