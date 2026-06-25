---
page_title: "bridgeport_environment Data Source - terraform-provider-bridgeport"
description: |-
  Look up a single BridgePort environment by its natural key (name).
---

# bridgeport_environment (Data Source)

Look up a single BridgePort environment by its natural key (`name`).

## Example Usage

```terraform
# Look up a single environment by its natural key (name).
data "bridgeport_environment" "production" {
  name = "production"
}

output "production_environment_id" {
  value = data.bridgeport_environment.production.id
}
```

## Schema

### Required

- `name` (String) The unique slug of the environment (its natural key), e.g. `production`.

### Read-Only

- `id` (String) Opaque server-assigned identifier for the environment.
- `display_name` (String) Human-friendly name shown in the UI.
- `ssh_configured` (Boolean) Whether an SSH key is configured for this environment.
- `created_at` (String) RFC 3339 timestamp of when the environment was created.
