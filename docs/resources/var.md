---
page_title: "bridgeport_var Resource - terraform-provider-bridgeport"
description: |-
  Manages a non-secret environment variable (a key/value). The value is stored in plaintext and readable via the API — use bridgeport_secret for sensitive values.
---

# bridgeport_var (Resource)

Manages a non-secret environment variable (a key/value). The value is stored in plaintext and readable via the API — use `bridgeport_secret` for sensitive values.

## Example Usage

```terraform
# Manage a non-secret environment variable.
resource "bridgeport_var" "log_level" {
  environment = "production"
  key         = "LOG_LEVEL"
  value       = "info"
  description = "Application log verbosity"
}
```

## Schema

### Required

- `environment` (String) The name of the environment the variable belongs to. Changing this forces a new resource.
- `key` (String) The variable key (its natural key). Must match `^[A-Z][A-Z0-9_]*$`. Changing this forces a new resource.
- `value` (String) The variable value (plaintext).

### Optional

- `description` (String) Human-friendly description of the variable.

### Read-Only

- `id` (String) Opaque server-assigned identifier for the variable.
- `created_at` (String) RFC 3339 timestamp of when the variable was created.
- `updated_at` (String) RFC 3339 timestamp of when the variable was last updated.

## Import

Import is supported using the following syntax:

```shell
# Variables are imported by their natural key: "environment/key".
terraform import bridgeport_var.log_level production/LOG_LEVEL
```
