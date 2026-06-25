# Look up a single service by its natural key (environment + server + name).
data "bridgeport_service" "api" {
  environment = "production"
  server      = "web-1"
  name        = "api"
}

output "api_image_tag" {
  value = data.bridgeport_service.api.image_tag
}
