---
page_title: "BridgePort Provider"
description: |-
  Manage BridgePort configuration as code — environments, servers, and the
  resources layered on top of them.
---

# BridgePort Provider

The BridgePort provider manages configuration on a [BridgePort](https://github.com/bridgeinpt/bridgeport) instance — environments, servers, and the resources layered on top of them — declaratively via its HTTP API. Runtime operations (deploys, restarts, rollbacks) remain imperative; the provider only manages desired configuration.

~> **Pre-1.0.** The provider currently ships **data sources** only. Managed (CRUD) resources are on the roadmap and the schema may change before `v1.0.0`.

## Example Usage

```terraform
terraform {
  required_providers {
    bridgeport = {
      source  = "bridgeinpt/bridgeport"
      version = "~> 0.1"
    }
  }
}

provider "bridgeport" {
  endpoint = "https://bridgeport.example.com"
  # token is read from BRIDGEPORT_TOKEN — keep it out of config and state
}
```

## Authentication

Supply the endpoint and token via environment variables so the token never lands in configuration or state:

```bash
export BRIDGEPORT_ENDPOINT="https://bridgeport.example.com"
export BRIDGEPORT_TOKEN="<service-account-token>"
```

Generate a token in BridgePort under **Service Accounts** (recommended for automation). Scope it to the minimum role and environments the run needs.

## Schema

### Optional

- `endpoint` (String) Base URL of the BridgePort instance, e.g. `https://bridgeport.example.com` (no trailing slash). May also be set with the `BRIDGEPORT_ENDPOINT` environment variable.
- `token` (String, Sensitive) API bearer token used to authenticate. A scoped service-account token is recommended. May also be set with the `BRIDGEPORT_TOKEN` environment variable.
