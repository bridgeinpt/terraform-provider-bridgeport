// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccServiceDeploymentResource exercises create -> import -> update ->
// destroy against the seeded "management" environment, wiring up a server, an
// image, and a service. Runs only with TF_ACC=1.
func TestAccServiceDeploymentResource(t *testing.T) {
	base := providerConfig + `
resource "bridgeport_server" "s" {
  environment = "management"
  name        = "tf-acc-dep-server"
  hostname    = "10.30.0.1"
}

resource "bridgeport_container_image" "img" {
  environment = "management"
  name        = "tf-acc-dep-img"
  image_name  = "library/nginx"
}

resource "bridgeport_service" "svc" {
  environment        = "management"
  name               = "tf-acc-dep-service"
  container_image_id = bridgeport_container_image.img.id
  image_tag          = "1.27"
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: base + `
resource "bridgeport_service_deployment" "test" {
  service_id     = bridgeport_service.svc.id
  server_id      = bridgeport_server.s.id
  container_name = "tf-acc-dep"
  env_overrides = {
    NODE_ENV = "production"
  }
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("bridgeport_service_deployment.test", "id"),
					resource.TestCheckResourceAttr("bridgeport_service_deployment.test", "container_name", "tf-acc-dep"),
					resource.TestCheckResourceAttr("bridgeport_service_deployment.test", "env_overrides.NODE_ENV", "production"),
				),
			},
			{
				Config: base + `
resource "bridgeport_service_deployment" "test" {
  service_id     = bridgeport_service.svc.id
  server_id      = bridgeport_server.s.id
  container_name = "tf-acc-dep2"
  env_overrides = {
    NODE_ENV = "staging"
  }
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bridgeport_service_deployment.test", "container_name", "tf-acc-dep2"),
					resource.TestCheckResourceAttr("bridgeport_service_deployment.test", "env_overrides.NODE_ENV", "staging"),
				),
			},
		},
	})
}
