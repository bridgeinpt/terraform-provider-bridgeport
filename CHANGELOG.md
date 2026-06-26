# Changelog

All notable changes to the BridgePort Terraform provider are documented here.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/) on its own
release line (independent of BridgePort platform releases).

## [Unreleased]

### Added

- Managed resource `bridgeport_server` — create, update, and delete servers in
  an environment, with natural-key import (`terraform import … environment/name`).
  First of the managed (CRUD) resources tracked in
  [bridgeinpt/bridgeport#197](https://github.com/bridgeinpt/bridgeport/issues/197).
- Managed resource `bridgeport_var` — manage non-secret environment variables,
  with natural-key import (`environment/key`).
- Managed resource `bridgeport_secret` — manage secrets with a **write-only**
  value (`value_wo` + `value_wo_version` rotation trigger), so secret values
  never enter Terraform state. Requires Terraform 1.11+.
- Managed resources `bridgeport_config_fragment` and `bridgeport_config_file` —
  manage reusable config fragments and text config files (which can include
  fragments via `fragment_ids`), with natural-key import (`environment/name`).
- Managed resources `bridgeport_registry_connection` (write-only `token_wo` /
  `password_wo` credentials) and `bridgeport_container_image`, with natural-key
  import.
- Managed resources `bridgeport_service` (environment-scoped service template,
  with `base_env`) and `bridgeport_service_deployment` (places a service onto a
  server, with `env_overrides`). Completes the managed-resource set for
  [bridgeinpt/bridgeport#197](https://github.com/bridgeinpt/bridgeport/issues/197).
- Data source `bridgeport_version` — reports the targeted instance's status and
  version (`version`, `bundled_agent_version`, `cli_version`) from `GET /health`,
  for provider ↔ instance version negotiation. Completes the data-source set
  from [bridgeinpt/terraform-provider-bridgeport#6](https://github.com/bridgeinpt/terraform-provider-bridgeport/issues/6);
  gated on the SDK gaining a typed health getter
  ([bridgeinpt/bridgeport#304](https://github.com/bridgeinpt/bridgeport/issues/304)).

### Changed

- Bump the BridgePort Go SDK to `client/v0.3.0`, which adds the resource write
  methods plus complete read-back (detail getters, full registry fields), letting
  the resources read by ID/natural key without list-based workarounds.
- Acceptance CI now uses Terraform 1.15.7 (write-only arguments need 1.11+).

## [0.1.0] - 2026-06-25

First tagged release, published to the [Terraform Registry](https://registry.terraform.io/providers/bridgeinpt/bridgeport/latest)
as `bridgeinpt/bridgeport`. Read-only: data sources only (managed resources are
on the roadmap, gated on the BridgePort Go SDK gaining write methods).

### Added

- Initial provider scaffold (terraform-plugin-framework, protocol v6).
- Provider configuration: `endpoint` and `token` (with `BRIDGEPORT_ENDPOINT` /
  `BRIDGEPORT_TOKEN` environment-variable fallbacks); `Configure` validates the
  token against the live instance.
- Data sources `bridgeport_environment` / `bridgeport_environments` — look up a
  single environment by name, or list all environments.
- Data sources `bridgeport_server` / `bridgeport_servers` — look up a server by
  its natural key (`environment` + `name`), or list servers (optionally filtered
  by environment).
- Data sources `bridgeport_service` / `bridgeport_services` — look up a service
  by its natural key (`environment` + `server` + `name`), or list services
  (optionally narrowed by environment / server).
- Acceptance-test harness (`scripts/acc-harness.sh`) that runs the `TF_ACC`
  suite against a disposable BridgePort image.
- GoReleaser + GPG-signed release pipeline for the Terraform/OpenTofu registries.

### Roadmap

- Managed resources in dependency order (`server` → `var`/`secret` →
  `config_file`/`fragment` → `registry_connection`/`container_image` →
  `service`/`service_deployment`), tracked in
  [bridgeinpt/bridgeport#197](https://github.com/bridgeinpt/bridgeport/issues/197).
  Gated on the BridgePort Go SDK (`client/`) gaining write methods.

[Unreleased]: https://github.com/bridgeinpt/terraform-provider-bridgeport/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/bridgeinpt/terraform-provider-bridgeport/releases/tag/v0.1.0
