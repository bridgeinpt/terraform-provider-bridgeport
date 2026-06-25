---
page_title: "bridgeport_config_fragment Resource - terraform-provider-bridgeport"
description: |-
  Manages a reusable config fragment — config text that can be included by one or more bridgeport_config_file resources.
---

# bridgeport_config_fragment (Resource)

Manages a reusable config fragment — config text that can be included by one or more `bridgeport_config_file` resources.

## Example Usage

```terraform
# Manage a reusable config fragment.
resource "bridgeport_config_fragment" "common_headers" {
  environment = "production"
  name        = "common-headers"
  content     = <<-EOT
    X-Frame-Options: DENY
    X-Content-Type-Options: nosniff
  EOT
  description = "Shared security headers"
}
```

## Schema

### Required

- `environment` (String) The name of the environment the fragment belongs to. Changing this forces a new resource.
- `name` (String) The unique name of the fragment within its environment (its natural key).
- `content` (String) The fragment's text content.

### Optional

- `description` (String) Human-friendly description of the fragment.

### Read-Only

- `id` (String) Opaque server-assigned identifier for the fragment.
- `environment_id` (String) Opaque identifier of the environment the fragment belongs to.
- `created_at` (String) RFC 3339 timestamp of when the fragment was created.
- `updated_at` (String) RFC 3339 timestamp of when the fragment was last updated.

## Import

Import is supported using the following syntax:

```shell
# Config fragments are imported by their natural key: "environment/name".
terraform import bridgeport_config_fragment.common_headers production/common-headers
```
