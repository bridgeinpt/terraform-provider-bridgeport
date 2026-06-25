---
page_title: "bridgeport_container_image Resource - terraform-provider-bridgeport"
description: |-
  Manages a container image tracked in an environment, optionally linked to a bridgeport_registry_connection.
---

# bridgeport_container_image (Resource)

Manages a container image tracked in an environment, optionally linked to a `bridgeport_registry_connection`.

## Example Usage

```terraform
# Track a container image, optionally from a registry connection.
resource "bridgeport_container_image" "app" {
  environment            = "production"
  name                   = "app"
  image_name             = "myorg/app"
  tag_filter             = "stable"
  registry_connection_id = bridgeport_registry_connection.do.id
  auto_update            = true
}
```

## Schema

### Required

- `environment` (String) The name of the environment the image belongs to. Changing this forces a new resource.
- `name` (String) Display name for the tracked image.
- `image_name` (String) The image repository name (e.g. `nginx`), the natural key within the environment. Changing this forces a new resource.

### Optional

- `tag_filter` (String) Tag (or filter) to track; defaults to `latest`.
- `registry_connection_id` (String) ID of the `bridgeport_registry_connection` the image is pulled from.
- `auto_update` (Boolean) Whether BridgePort auto-updates the tracked tag.

### Read-Only

- `id` (String) Opaque server-assigned identifier for the image.
- `environment_id` (String) Opaque identifier of the environment the image belongs to.
- `current_tag` (String) The currently-resolved tag (runtime, reference only).
- `latest_tag` (String) The latest available tag, if known (runtime, reference only).
- `update_available` (Boolean) Whether a newer tag is available (runtime, reference only).
- `created_at` (String) RFC 3339 timestamp of when the image was created.
- `updated_at` (String) RFC 3339 timestamp of when the image was last updated.

## Import

Import is supported using the following syntax:

```shell
# Container images are imported by their natural key: "environment/image_name".
terraform import bridgeport_container_image.app production/myorg/app
```
