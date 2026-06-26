---
page_title: "bridgeport_registry_connection Resource - terraform-provider-bridgeport"
description: |-
  Manages a container registry connection in an environment. Credentials (token_wo, password_wo) are write-only — sent to BridgePort but never stored in Terraform state.
---

# bridgeport_registry_connection (Resource)

Manages a container registry connection in an environment. Credentials (`token_wo`, `password_wo`) are **write-only** — sent to BridgePort but never stored in Terraform state. Rotate a credential by changing it together with its `*_version`.

~> **Terraform 1.11+ required** when using `token_wo` / `password_wo` (write-only arguments).

## Example Usage

```terraform
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
```

## Schema

### Required

- `environment` (String) The name of the environment the registry belongs to. Changing this forces a new resource.
- `name` (String) The unique name of the registry connection within its environment (its natural key).
- `type` (String) Registry type: `digitalocean`, `dockerhub`, or `generic`.
- `registry_url` (String) Base URL of the registry.

### Optional

- `repository_prefix` (String) Optional repository prefix applied to image names.
- `username` (String) Username for registries that authenticate with username + password.
- `token_wo` (String, Write-only, Sensitive) Write-only access token (for token-based registries). Requires Terraform 1.11+.
- `token_wo_version` (String) Version string for `token_wo`; change it together with `token_wo` to rotate the token.
- `password_wo` (String, Write-only, Sensitive) Write-only password (for username/password registries). Requires Terraform 1.11+.
- `password_wo_version` (String) Version string for `password_wo`; change it together with `password_wo` to rotate the password.
- `is_default` (Boolean) Whether this is the environment's default registry.
- `refresh_interval_minutes` (Number) How often (minutes) BridgePort refreshes tags from the registry.
- `auto_link_pattern` (String) Optional pattern for auto-linking discovered images.

### Read-Only

- `id` (String) Opaque server-assigned identifier for the registry connection.
- `environment_id` (String) Opaque identifier of the environment the registry belongs to.
- `has_token` (Boolean) Whether an access token is configured.
- `has_password` (Boolean) Whether a password is configured.
- `image_count` (Number) Number of container images linked to this registry.
- `created_at` (String) RFC 3339 timestamp of when the registry connection was created.
- `updated_at` (String) RFC 3339 timestamp of when the registry connection was last updated.

## Import

Import is supported using the following syntax:

```shell
# Registry connections are imported by their natural key: "environment/name".
# Credentials can't be recovered — re-declare token_wo/password_wo afterward.
terraform import bridgeport_registry_connection.do production/digitalocean
```
