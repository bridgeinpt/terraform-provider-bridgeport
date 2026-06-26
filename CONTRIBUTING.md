# Contributing to terraform-provider-bridgeport

Thanks for your interest in contributing! This is the official Terraform/OpenTofu provider for [BridgePort](https://github.com/bridgeinpt/bridgeport). Participation is governed by our [Code of Conduct](CODE_OF_CONDUCT.md).

> Repository conventions and architecture live in [CLAUDE.md](CLAUDE.md). Read it before your first change — it's short and authoritative.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Setup](#setup)
- [Development workflow](#development-workflow)
- [Adding a data source or resource](#adding-a-data-source-or-resource)
- [Testing](#testing)
- [Docs](#docs)
- [Design tenets](#design-tenets)
- [Releasing](#releasing)
- [Getting help](#getting-help)

## Prerequisites

| Tool | Version | Used for |
|------|---------|----------|
| Go | see `go.mod` | Building the provider |
| Terraform or OpenTofu | 1.6+ | Running / acceptance-testing |
| Docker | 20+ | Acceptance tests (disposable BridgePort) |
| golangci-lint | v2.x | Linting (`make lint`) |

## Setup

```bash
git clone https://github.com/bridgeinpt/terraform-provider-bridgeport.git
cd terraform-provider-bridgeport

# go.mod + go.sum are committed — builds out of the box
make build

# after bumping the SDK or a dependency, re-tidy and commit the diff
make bootstrap
```

To exercise an unreleased build against real `.tf`, use a Terraform [dev override](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides):

```hcl
# ~/.terraformrc
provider_installation {
  dev_overrides {
    "bridgeinpt/bridgeport" = "/path/to/your/$GOPATH/bin"
  }
  direct {}
}
```

Then `make install` and run Terraform — it will use your local binary (skip `terraform init`).

## Development workflow

Branch off `master` with a descriptive prefix (`feature/`, `fix/`, `docs/`), make small focused commits, and open a PR. Before pushing:

```bash
make fmt      # gofmt
make vet
make lint
make test     # unit tests
make testacc  # acceptance (needs Docker)
make generate # if you changed schema/examples — commit the docs/ diff
```

## Adding a data source or resource

See the step-by-step in [CLAUDE.md](CLAUDE.md#adding-a-data-source-or-resource). In short: implement under `internal/provider/`, register the factory in `provider.go`, add an `examples/` snippet, add a `TF_ACC` test, and run `make generate`.

If your change needs BridgePort API surface the Go SDK doesn't expose yet, **add it to the SDK first** — the SDK is `client/` in the [platform repo](https://github.com/bridgeinpt/bridgeport) — release it as a `client/vX.Y.Z` tag, and bump the dependency here.

## Testing

| Kind | Command | Notes |
|------|---------|-------|
| Unit | `make test` | Fast, no live instance |
| Acceptance | `make testacc` | `TF_ACC=1`; spins up a disposable BridgePort via `scripts/acc-harness.sh`, mints a token, runs the suite, tears down |

Every new data source/resource needs an acceptance test. Use `testAccProtoV6ProviderFactories` and `testAccPreCheck` from `provider_test.go`.

## Docs

Registry docs in `docs/` are **generated** by [`tfplugindocs`](https://github.com/hashicorp/terraform-plugin-docs) from the schema `MarkdownDescription`s and the `examples/` directory. Don't hand-edit `docs/` — edit the schema/examples and run `make generate`. The tfplugindocs version is pinned in `main.go`'s `//go:generate` directive so the output is reproducible, and CI (`.github/workflows/test.yml`) fails if the committed `docs/` don't match `make generate`.

## Design tenets

The provider manages **configuration only** — runtime operations (deploys, restarts, rollbacks) stay imperative. Secrets never enter state. `plan` makes no live API calls. These are non-negotiable; see [CLAUDE.md](CLAUDE.md#design-tenets-do-not-violate) and the [platform epic](https://github.com/bridgeinpt/bridgeport/issues/197).

## Releasing

Releases are GPG-signed and published in the layout the Terraform/OpenTofu registries ingest.

1. Ensure the repo secrets `GPG_PRIVATE_KEY` and `PASSPHRASE` are set, and the release actions are allowlisted (see the repo's manual-setup notes).
2. Update [CHANGELOG.md](CHANGELOG.md).
3. Tag and push: `git tag v0.1.0 && git push origin v0.1.0`.
4. `.github/workflows/release.yml` runs GoReleaser and creates the GitHub Release. The registry picks it up once the repo is published and the signing key is registered there.

## Getting help

- **Provider questions / design**: [BridgePort Discussions](https://github.com/bridgeinpt/bridgeport/discussions)
- **Bugs**: [Issues](https://github.com/bridgeinpt/terraform-provider-bridgeport/issues)
- **Security**: [SECURITY.md](SECURITY.md) — private advisories only

Thank you for contributing!
