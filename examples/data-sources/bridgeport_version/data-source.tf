# Read the targeted BridgePort instance's version (from GET /health) for
# provider/version negotiation against the target instance.
data "bridgeport_version" "this" {}

output "bridgeport_version" {
  value = data.bridgeport_version.this.version
}

# Assert a minimum platform version before this configuration applies.
check "bridgeport_supported" {
  assert {
    condition     = data.bridgeport_version.this.status == "ok"
    error_message = "BridgePort instance is not healthy (status: ${data.bridgeport_version.this.status})."
  }
}
