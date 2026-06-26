// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccContainerImageResource exercises create -> import -> update -> destroy
// against the seeded "management" environment. Runs only with TF_ACC=1.
func TestAccContainerImageResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "bridgeport_container_image" "test" {
  environment = "management"
  name        = "tf-acc-image"
  image_name  = "library/nginx"
  tag_filter  = "1.27"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("bridgeport_container_image.test", "id"),
					resource.TestCheckResourceAttr("bridgeport_container_image.test", "name", "tf-acc-image"),
					resource.TestCheckResourceAttr("bridgeport_container_image.test", "image_name", "library/nginx"),
					resource.TestCheckResourceAttr("bridgeport_container_image.test", "tag_filter", "1.27"),
				),
			},
			{
				ResourceName:      "bridgeport_container_image.test",
				ImportState:       true,
				ImportStateId:     "management/library/nginx",
				ImportStateVerify: true,
			},
			{
				Config: providerConfig + `
resource "bridgeport_container_image" "test" {
  environment = "management"
  name        = "tf-acc-image-renamed"
  image_name  = "library/nginx"
  tag_filter  = "1.28"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bridgeport_container_image.test", "name", "tf-acc-image-renamed"),
					resource.TestCheckResourceAttr("bridgeport_container_image.test", "tag_filter", "1.28"),
				),
			},
		},
	})
}
