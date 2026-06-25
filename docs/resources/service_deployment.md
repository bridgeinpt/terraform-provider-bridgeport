---
page_title: "bridgeport_service_deployment Resource - terraform-provider-bridgeport"
description: |-
  Manages a service deployment: the placement of a bridgeport_service onto a bridgeport_server. Manages configuration only; runtime status/health are surfaced read-only.
---

# bridgeport_service_deployment (Resource)

Manages a service deployment: the placement of a `bridgeport_service` onto a `bridgeport_server`. Manages configuration only; runtime status/health are surfaced read-only.

## Example Usage

```terraform
# Deploy a service onto a specific server.
resource "bridgeport_service_deployment" "app_web1" {
  service_id     = bridgeport_service.app.id
  server_id      = bridgeport_server.web1.id
  container_name = "app"
  env_overrides = {
    NODE_ENV = "production"
  }
}
```

## Schema

### Required

- `service_id` (String) ID of the `bridgeport_service` being deployed. Changing this forces a new resource.
- `server_id` (String) ID of the `bridgeport_server` to deploy onto. Changing this forces a new resource.
- `container_name` (String) Name of the container for this deployment.

### Optional

- `compose_path` (String) Optional path to a compose file for this deployment.
- `env_overrides` (Map of String) Per-deployment environment variable overrides (key/value).

### Read-Only

- `id` (String) Opaque server-assigned identifier for the deployment.
- `status` (String) Current deployment status (runtime, reference only).
- `container_status` (String) Current container status (runtime, reference only).
- `health_status` (String) Current health status (runtime, reference only).
- `discovery_status` (String) Current discovery status (runtime, reference only).
- `last_deployed_at` (String) RFC 3339 timestamp of the last deploy, if any (runtime, reference only).

## Import

Import is supported using the following syntax:

```shell
# Service deployments are imported as "service_id/server_id".
terraform import bridgeport_service_deployment.app_web1 svc-abc123/srv-def456
```
