# Changelog

All notable changes to the BridgePort Terraform provider are documented here.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/) on its own
release line (independent of BridgePort platform releases).

## [Unreleased]

### Added

- Initial provider scaffold (terraform-plugin-framework, protocol v6).
- Provider configuration: `endpoint` and `token` (with `BRIDGEPORT_ENDPOINT` /
  `BRIDGEPORT_TOKEN` environment-variable fallbacks); `Configure` validates the
  token against the live instance.
- Data source `bridgeport_environment` — look up a single environment by name.
- Data source `bridgeport_environments` — list all environments.
- Acceptance-test harness (`scripts/acc-harness.sh`) that runs the `TF_ACC`
  suite against a disposable BridgePort image.
- GoReleaser + GPG-signed release pipeline for the Terraform/OpenTofu registries.

### Roadmap

- Managed resources in dependency order (`server` → `var`/`secret` →
  `config_file`/`fragment` → `registry_connection`/`container_image` →
  `service`/`service_deployment`), tracked in
  [bridgeinpt/bridgeport#197](https://github.com/bridgeinpt/bridgeport/issues/197).
