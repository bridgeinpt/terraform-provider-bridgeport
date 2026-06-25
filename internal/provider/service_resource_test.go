// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccServiceResource exercises create -> import -> update -> destroy against
// the seeded "management" environment. Runs only with TF_ACC=1.
func TestAccServiceResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "bridgeport_container_image" "img" {
  environment = "management"
  name        = "tf-acc-svc-img"
  image_name  = "library/nginx"
}

resource "bridgeport_service" "test" {
  environment        = "management"
  name               = "tf-acc-service"
  container_image_id = bridgeport_container_image.img.id
  image_tag          = "1.27"
  deploy_strategy    = "sequential"
  base_env = {
    LOG_LEVEL = "info"
  }
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("bridgeport_service.test", "id"),
					resource.TestCheckResourceAttr("bridgeport_service.test", "name", "tf-acc-service"),
					resource.TestCheckResourceAttr("bridgeport_service.test", "image_tag", "1.27"),
					resource.TestCheckResourceAttr("bridgeport_service.test", "base_env.LOG_LEVEL", "info"),
				),
			},
			{
				ResourceName:      "bridgeport_service.test",
				ImportState:       true,
				ImportStateId:     "management/tf-acc-service",
				ImportStateVerify: true,
			},
			{
				Config: providerConfig + `
resource "bridgeport_container_image" "img" {
  environment = "management"
  name        = "tf-acc-svc-img"
  image_name  = "library/nginx"
}

resource "bridgeport_service" "test" {
  environment        = "management"
  name               = "tf-acc-service"
  container_image_id = bridgeport_container_image.img.id
  image_tag          = "1.28"
  deploy_strategy    = "parallel"
  base_env = {
    LOG_LEVEL = "debug"
  }
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bridgeport_service.test", "image_tag", "1.28"),
					resource.TestCheckResourceAttr("bridgeport_service.test", "deploy_strategy", "parallel"),
					resource.TestCheckResourceAttr("bridgeport_service.test", "base_env.LOG_LEVEL", "debug"),
				),
			},
		},
	})
}
