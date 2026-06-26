# Deploy a service onto a specific server.
resource "bridgeport_service_deployment" "app_web1" {
  service_id     = bridgeport_service.app.id
  server_id      = bridgeport_server.web1.id
  container_name = "app"
  env_overrides = {
    NODE_ENV = "production"
  }
}
