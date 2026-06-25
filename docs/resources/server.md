---
page_title: "bridgeport_server Resource - terraform-provider-bridgeport"
description: |-
  Manages a BridgePort server within an environment. This manages the server's configuration only; runtime operations (deploys, restarts) and live status remain imperative and are surfaced read-only.
---

# bridgeport_server (Resource)

Manages a BridgePort server within an environment. This manages the server's *configuration* only; runtime operations (deploys, restarts) and live status remain imperative and are surfaced read-only.

## Example Usage

```terraform
# Manage a server within an existing environment.
resource "bridgeport_server" "web" {
  environment = "production"
  name        = "web-1"
  hostname    = "10.0.0.10"
  public_ip   = "203.0.113.10"
  tags        = ["web", "edge"]
}

output "web_server_id" {
  value = bridgeport_server.web.id
}
```

## Schema

### Required

- `environment` (String) The name of the environment the server belongs to. Changing this forces a new server.
- `name` (String) The unique name of the server within its environment (its natural key).
- `hostname` (String) The hostname or IP address BridgePort uses to reach the server.

### Optional

- `public_ip` (String) The server's public IP address, if any.
- `tags` (List of String) Free-form tags applied to the server.

### Read-Only

- `id` (String) Opaque server-assigned identifier for the server.
- `environment_id` (String) Opaque identifier of the environment the server belongs to.
- `private_ip` (String) The server's private IP address, assigned by the platform.
- `status` (String) Current runtime status of the server as reported by the platform.
- `created_at` (String) RFC 3339 timestamp of when the server was created.

## Import

Import is supported using the following syntax:

```shell
# Servers are imported by their natural key: "environment/name".
terraform import bridgeport_server.web production/web-1
```
