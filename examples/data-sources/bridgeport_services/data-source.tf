# List every service visible to the configured token.
data "bridgeport_services" "all" {}

# Or narrow to a single environment...
data "bridgeport_services" "production" {
  environment = "production"
}

# ...or to a single server within an environment.
data "bridgeport_services" "web_1" {
  environment = "production"
  server      = "web-1"
}

output "production_service_names" {
  value = [for s in data.bridgeport_services.production.services : s.name]
}
