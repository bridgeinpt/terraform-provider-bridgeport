# Manage a container registry connection. Credentials are write-only.
variable "registry_token" {
  type      = string
  sensitive = true
}

resource "bridgeport_registry_connection" "do" {
  environment      = "production"
  name             = "digitalocean"
  type             = "digitalocean"
  registry_url     = "registry.digitalocean.com"
  token_wo         = var.registry_token
  token_wo_version = "1"
}
