# Look up a single environment by its natural key (name).
data "bridgeport_environment" "production" {
  name = "production"
}

output "production_environment_id" {
  value = data.bridgeport_environment.production.id
}
