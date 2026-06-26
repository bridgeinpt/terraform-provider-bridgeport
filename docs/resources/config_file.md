---
page_title: "bridgeport_config_file Resource - terraform-provider-bridgeport"
description: |-
  Manages a text config file in an environment, optionally composed from bridgeport_config_fragment fragments. Binary files are not managed by this resource.
---

# bridgeport_config_file (Resource)

Manages a text config file in an environment, optionally composed from `bridgeport_config_fragment` fragments. Binary files are not managed by this resource.

## Example Usage

```terraform
# Manage a text config file, optionally composed from fragments.
resource "bridgeport_config_fragment" "common_headers" {
  environment = "production"
  name        = "common-headers"
  content     = "X-Frame-Options: DENY\n"
}

resource "bridgeport_config_file" "nginx" {
  environment  = "production"
  name         = "nginx-conf"
  filename     = "nginx.conf"
  language     = "nginx"
  content      = "server { listen 80; }\n"
  fragment_ids = [bridgeport_config_fragment.common_headers.id]
}
```

## Schema

### Required

- `environment` (String) The name of the environment the config file belongs to. Changing this forces a new resource.
- `name` (String) The unique name of the config file within its environment (its natural key).
- `filename` (String) The on-disk filename the content is written to.
- `content` (String) The file's text content.

### Optional

- `description` (String) Human-friendly description of the config file.
- `language` (String) Syntax/language hint for the content (e.g. `yaml`, `json`).
- `fragment_ids` (List of String) Ordered list of `bridgeport_config_fragment` IDs to include in the file. Note: the API does not return fragment associations on read, so changes made outside Terraform are not detected.

### Read-Only

- `id` (String) Opaque server-assigned identifier for the config file.
- `environment_id` (String) Opaque identifier of the environment the config file belongs to.
- `sync_status` (String) Current sync status of the file across servers (runtime, reference only).
- `created_at` (String) RFC 3339 timestamp of when the config file was created.
- `updated_at` (String) RFC 3339 timestamp of when the config file was last updated.

## Import

Import is supported using the following syntax:

```shell
# Config files are imported by their natural key: "environment/name".
# fragment_ids cannot be recovered on import — re-declare them in config.
terraform import bridgeport_config_file.nginx production/nginx-conf
```
