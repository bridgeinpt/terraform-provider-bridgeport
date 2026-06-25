<h1 align="center">Terraform Provider for BridgePort</h1>

<p align="center">
  Manage your <a href="https://github.com/bridgeinpt/bridgeport">BridgePort</a> configuration as code — environments, servers, secrets, config files, registries, and services — with <code>terraform plan</code> surfacing drift before it bites.
</p>

<p align="center">
  <a href="LICENSE"><img alt="License: Apache-2.0" src="https://img.shields.io/badge/License-Apache--2.0-blue.svg" /></a>
  <a href="https://registry.terraform.io/providers/bridgeinpt/bridgeport/latest"><img alt="Terraform Registry" src="https://img.shields.io/badge/registry-bridgeinpt%2Fbridgeport-7B42BC?logo=terraform&logoColor=white" /></a>
  <a href="https://github.com/bridgeinpt/terraform-provider-bridgeport/actions/workflows/test.yml"><img alt="Tests" src="https://github.com/bridgeinpt/terraform-provider-bridgeport/actions/workflows/test.yml/badge.svg" /></a>
  <img alt="Go" src="https://img.shields.io/badge/Go-1.25-CC0000?logo=go&logoColor=white" />
</p>

<p align="center">
  <sub>Created by the Engineering Team at <a href="https://bridgein.pt">BRIDGE IN</a>.</sub>
</p>

---

> [!WARNING]
> **Pre-1.0 and under active development.** The provider currently ships **data sources** only. Managed (CRUD) resources are on the [roadmap](#roadmap) and the schema may change before `v1.0.0`. Pin a version and read the [CHANGELOG](CHANGELOG.md) before upgrading.

## Why

BridgePort sits between infrastructure that's typically provisioned with IaC and the services running on it. Today that handoff is imperative — the UI, ad-hoc scripts, and the one-way import endpoint. This provider closes the loop: desired state lives in version control, changes ship through review, and configuration that drifts out-of-band shows up in `plan` instead of as a 2 AM surprise.

**Design tenets** (see the [platform epic](https://github.com/bridgeinpt/bridgeport/issues/197)):

- **Configuration only — runtime stays imperative.** Deploys, restarts, and rollbacks remain UI/API/CLI operations. Runtime fields (status, ports, health, discovery) are read-only/`Computed`.
- **Secrets never enter Terraform state.** Secret values use write-only arguments with a version attribute to trigger rotation.
- **`plan` works offline** — no live API calls at plan/validate time — and diffs against your submitted configuration, not against live runtime state.
- **Acceptance tests run against the real image.** SQLite + the admin-bootstrap make disposable instances cheap.

## Using the provider

```hcl
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

# Look up an environment by its natural key
data "bridgeport_environment" "prod" {
  name = "production"
}

output "prod_id" {
  value = data.bridgeport_environment.prod.id
}
```

Configure credentials via environment variables so the token never lands in state:

```bash
export BRIDGEPORT_ENDPOINT="https://bridgeport.example.com"
export BRIDGEPORT_TOKEN="<service-account-token>"   # admin/operator/viewer scoped as needed
```

Generate a token in BridgePort under **Service Accounts** (recommended for automation) or via `bridgeport login`.

### Available data sources

| Data source | Purpose |
|---|---|
| `bridgeport_environment` | Look up a single environment by `name` |
| `bridgeport_environments` | List all environments visible to the token |
| `bridgeport_server` | Look up a single server by `environment` + `name` |
| `bridgeport_servers` | List servers, optionally filtered by `environment` |
| `bridgeport_service` | Look up a single service by `environment` + `server` + `name` |
| `bridgeport_services` | List services, optionally narrowed by `environment` / `server` |

Full reference: the [`docs/`](docs/) directory (rendered on the [Terraform Registry](https://registry.terraform.io/providers/bridgeinpt/bridgeport/latest/docs)).

## Roadmap

Resources land in dependency order, tracked in the [platform epic #197](https://github.com/bridgeinpt/bridgeport/issues/197):

1. `bridgeport_server`
2. `bridgeport_var` / `bridgeport_secret` (write-only values)
3. `bridgeport_config_file` / `bridgeport_config_fragment` (+ attachments)
4. `bridgeport_registry_connection` / `bridgeport_container_image`
5. `bridgeport_service` / `bridgeport_service_deployment`

`terraform import` will work off the existing natural keys (`environment` + `name`/`key`).

## Compatibility

The provider talks to BridgePort's HTTP API, whose wire format is governed by the [API Stability & Deprecation Policy](https://github.com/bridgeinpt/bridgeport/blob/master/docs/api-stability.md). A provider release supports a *range* of BridgePort versions and is versioned on its own semver line, independent of platform releases. The canonical contract is the checked-in [OpenAPI snapshot](https://github.com/bridgeinpt/bridgeport/blob/master/openapi.json).

## Development

Requires Go (see `go.mod`), Terraform/OpenTofu, and Docker (for acceptance tests).

```bash
make build       # compile (go.mod + go.sum are committed — builds out of the box)
make bootstrap   # go mod tidy — run after bumping the SDK or a dependency
make test        # unit tests
make lint        # golangci-lint
make generate    # regenerate docs/ from schema + examples/ (tfplugindocs)
make testacc     # acceptance suite against a disposable BridgePort image
```

`make testacc` uses [`scripts/acc-harness.sh`](scripts/acc-harness.sh): it `docker run`s a throwaway BridgePort, mints a token via the first-boot admin bootstrap, runs the `TF_ACC` suite, and tears the instance down. Override the image with `BRIDGEPORT_IMAGE`.

To try an unreleased build against a real Terraform config, use a [dev override](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides) pointing `bridgeinpt/bridgeport` at your `$GOPATH/bin`.

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full workflow and [CLAUDE.md](CLAUDE.md) for repository conventions.

## Releasing

Releases are cut by pushing a `vX.Y.Z` tag. [`.github/workflows/release.yml`](.github/workflows/release.yml) runs GoReleaser to cross-compile, build the registry zips + `SHA256SUMS`, and GPG-sign the checksums. The Terraform and OpenTofu registries ingest the resulting GitHub Release. See [CONTRIBUTING.md](CONTRIBUTING.md#releasing) for the GPG/secret prerequisites.

## Community and support

- **Bugs / features**: [Issues](https://github.com/bridgeinpt/terraform-provider-bridgeport/issues)
- **Questions**: [BridgePort Discussions](https://github.com/bridgeinpt/bridgeport/discussions)
- **Security**: [SECURITY.md](SECURITY.md) — please use private advisories, not public issues

## License

Licensed under the [Apache License 2.0](LICENSE). Copyright 2024-2026 BRIDGE IN.
