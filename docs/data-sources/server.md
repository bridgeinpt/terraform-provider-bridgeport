---
page_title: "bridgeport_server Data Source - terraform-provider-bridgeport"
description: |-
  Look up a single BridgePort server by its natural key (environment + name).
---

# bridgeport_server (Data Source)

Look up a single BridgePort server by its natural key (`environment` + `name`).

## Example Usage

```terraform
# Look up a single server by its natural key (environment + name).
data "bridgeport_server" "web" {
  environment = "production"
  name        = "web-1"
}

output "web_server_private_ip" {
  value = data.bridgeport_server.web.private_ip
}
```

## Schema

### Required

- `environment` (String) The name of the environment the server belongs to, e.g. `production`.
- `name` (String) The unique name of the server within its environment (its natural key).

### Read-Only

- `id` (String) Opaque server-assigned identifier for the server.
- `environment_id` (String) Opaque identifier of the environment the server belongs to.
- `private_ip` (String) The server's private IP address.
- `public_ip` (String) The server's public IP address, or null if it has none.
- `status` (String) Current runtime status of the server as reported by the platform.
- `created_at` (String) RFC 3339 timestamp of when the server was created.
