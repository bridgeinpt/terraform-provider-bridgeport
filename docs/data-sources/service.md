---
page_title: "bridgeport_service Data Source - terraform-provider-bridgeport"
description: |-
  Look up a single BridgePort service by its natural key (environment + server + name).
---

# bridgeport_service (Data Source)

Look up a single BridgePort service by its natural key (`environment` + `server` + `name`).

## Example Usage

```terraform
# Look up a single service by its natural key (environment + server + name).
data "bridgeport_service" "api" {
  environment = "production"
  server      = "web-1"
  name        = "api"
}

output "api_image_tag" {
  value = data.bridgeport_service.api.image_tag
}
```

## Schema

### Required

- `environment` (String) The name of the environment that hosts the service's server, e.g. `production`.
- `server` (String) The name of the server the service runs on.
- `name` (String) The unique name of the service on the server (its natural key).

### Read-Only

- `id` (String) Opaque server-assigned identifier for the service.
- `image_tag` (String) The container image tag the service is configured to run.
- `environment_id` (String) Opaque identifier of the environment the service belongs to.
- `container_image_id` (String) Opaque identifier of the container image the service deploys.
- `service_type_id` (String) Opaque identifier of the service type, or null if the service has none.
- `server_id` (String) Opaque identifier of the server the service runs on.
- `container_name` (String) Name of the running container (runtime, reference only).
- `status` (String) Current runtime status of the service as reported by the platform (reference only).
- `container_status` (String) Current container status (runtime, reference only).
- `health_status` (String) Current health status (runtime, reference only).
- `created_at` (String) RFC 3339 timestamp of when the service was created.
