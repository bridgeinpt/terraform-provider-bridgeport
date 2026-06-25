terraform {
  required_providers {
    bridgeport = {
      source  = "bridgeinpt/bridgeport"
      version = "~> 0.1"
    }
  }
}

# Endpoint and token are best supplied via environment variables so the token
# never lands in configuration or state:
#
#   export BRIDGEPORT_ENDPOINT="https://bridgeport.example.com"
#   export BRIDGEPORT_TOKEN="$(cat ~/.bridgeport-token)"
#
provider "bridgeport" {
  endpoint = "https://bridgeport.example.com"
  # token  = var.bridgeport_token   # or rely on BRIDGEPORT_TOKEN
}
