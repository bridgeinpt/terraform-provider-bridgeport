# List every environment visible to the configured token.
data "bridgeport_environments" "all" {}

output "environment_names" {
  value = [for e in data.bridgeport_environments.all.environments : e.name]
}
