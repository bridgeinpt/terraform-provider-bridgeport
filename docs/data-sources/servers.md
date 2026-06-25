---
page_title: "bridgeport_servers Data Source - terraform-provider-bridgeport"
description: |-
  List servers visible to the configured token, optionally filtered to a single environment.
---

# bridgeport_servers (Data Source)

List servers visible to the configured token, optionally filtered to a single environment.

## Example Usage

```terraform
# List every server visible to the configured token.
data "bridgeport_servers" "all" {}

# Or narrow to a single environment.
data "bridgeport_servers" "production" {
  environment = "production"
}

output "production_server_names" {
  value = [for s in data.bridgeport_servers.production.servers : s.name]
}
```

## Schema

### Optional

- `environment` (String) If set, only return servers in this environment (by `name`). Omit to list servers across every environment the token can see.

### Read-Only

- `servers` (Attributes List) The matching servers. (see [below for nested schema](#nestedatt--servers))

<a id="nestedatt--servers"></a>
### Nested Schema for `servers`

Read-Only:

- `id` (String) Opaque server-assigned identifier for the server.
- `name` (String) The unique name of the server within its environment.
- `environment_id` (String) Opaque identifier of the environment the server belongs to.
- `private_ip` (String) The server's private IP address.
- `public_ip` (String) The server's public IP address, or null if it has none.
- `status` (String) Current runtime status of the server as reported by the platform.
- `created_at` (String) RFC 3339 timestamp of when the server was created.
