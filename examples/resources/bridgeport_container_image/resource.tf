# Track a container image, optionally from a registry connection.
resource "bridgeport_container_image" "app" {
  environment            = "production"
  name                   = "app"
  image_name             = "myorg/app"
  tag_filter             = "stable"
  registry_connection_id = bridgeport_registry_connection.do.id
  auto_update            = true
}
