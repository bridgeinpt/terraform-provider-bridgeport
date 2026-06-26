# Manage a service template (deployed onto servers via service_deployment).
resource "bridgeport_container_image" "app" {
  environment = "production"
  name        = "app"
  image_name  = "myorg/app"
}

resource "bridgeport_service" "app" {
  environment        = "production"
  name               = "app"
  container_image_id = bridgeport_container_image.app.id
  image_tag          = "1.4.0"
  deploy_strategy    = "sequential"
  base_env = {
    LOG_LEVEL = "info"
  }
}
