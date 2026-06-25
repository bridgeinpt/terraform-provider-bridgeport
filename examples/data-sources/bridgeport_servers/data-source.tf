# List every server visible to the configured token.
data "bridgeport_servers" "all" {}

# Or narrow to a single environment.
data "bridgeport_servers" "production" {
  environment = "production"
}

output "production_server_names" {
  value = [for s in data.bridgeport_servers.production.servers : s.name]
}
