---
page_title: "bridgeport_service Resource - terraform-provider-bridgeport"
description: |-
  Manages a service template in an environment. A service is the deployable definition; placing it on servers is done with bridgeport_service_deployment. Runtime status is not managed here.
---

# bridgeport_service (Resource)

Manages a service template in an environment. A service is the deployable definition; placing it on servers is done with `bridgeport_service_deployment`. Runtime status is not managed here.

## Example Usage

```terraform
# Manage a service template (deployed onto servers via service_deployment).
resource "bridgeport_container_image" "app" {
  environment = "production"
  name        = "app"
  image_name  = "myorg/app"
}

resource "bridgeport_service" "app" {
  environment        = "production"
  name               = "app"
  container_image_id = bridgeport_container_image.app.id
  image_tag          = "1.4.0"
  deploy_strategy    = "sequential"
  base_env = {
    LOG_LEVEL = "info"
  }
}
```

## Schema

### Required

- `environment` (String) The name of the environment the service belongs to. Changing this forces a new resource.
- `name` (String) The unique name of the service within its environment (its natural key).
- `container_image_id` (String) ID of the `bridgeport_container_image` the service runs.

### Optional

- `image_tag` (String) Image tag to run; defaults to `latest`.
- `compose_template` (String) Optional Docker Compose template for the service.
- `health_check_url` (String) Optional health-check URL.
- `base_env` (Map of String) Base environment variables applied to the service (key/value).
- `deploy_strategy` (String) Deployment strategy: `sequential` or `parallel`.
- `service_type_id` (String) Optional service-type ID.
- `health_wait_ms` (Number) Milliseconds to wait before health checks.
- `health_retries` (Number) Number of health-check retries.
- `health_interval_ms` (Number) Milliseconds between health-check retries.

### Read-Only

- `id` (String) Opaque server-assigned identifier for the service.
- `environment_id` (String) Opaque identifier of the environment the service belongs to.
- `created_at` (String) RFC 3339 timestamp of when the service was created.

## Import

Import is supported using the following syntax:

```shell
# Services are imported by their natural key: "environment/name".
terraform import bridgeport_service.app production/app
```
