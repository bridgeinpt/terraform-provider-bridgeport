---
page_title: "bridgeport_environments Data Source - terraform-provider-bridgeport"
description: |-
  List all environments visible to the configured token.
---

# bridgeport_environments (Data Source)

List all environments visible to the configured token.

## Example Usage

```terraform
# List every environment visible to the configured token.
data "bridgeport_environments" "all" {}

output "environment_names" {
  value = [for e in data.bridgeport_environments.all.environments : e.name]
}
```

## Schema

### Read-Only

- `environments` (Attributes List) All environments the token can see. (see [below for nested schema](#nestedatt--environments))

<a id="nestedatt--environments"></a>
### Nested Schema for `environments`

Read-Only:

- `id` (String) Opaque server-assigned identifier for the environment.
- `name` (String) The unique slug of the environment (its natural key).
- `display_name` (String) Human-friendly name shown in the UI.
- `ssh_configured` (Boolean) Whether an SSH key is configured for this environment.
- `created_at` (String) RFC 3339 timestamp of when the environment was created.
