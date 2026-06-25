# Security Policy

This policy covers the **Terraform provider** (`terraform-provider-bridgeport`). For vulnerabilities in the BridgePort server or its HTTP API, report against the [platform repo](https://github.com/bridgeinpt/bridgeport/blob/master/docs/SECURITY.md) instead.

## Reporting a Vulnerability

**Please do not open public GitHub issues for security vulnerabilities.**

Instead, either:

- Open a private advisory via [GitHub Security Advisories](https://github.com/bridgeinpt/terraform-provider-bridgeport/security/advisories/new), or
- Email the maintainers at **security@bridgein.pt**

Include a description, reproduction steps (or a proof-of-concept), and the impact as you understand it.

### What to report (provider-specific)

- **Secret leakage into state or logs** — a token or secret value written to Terraform state, plan output, or `TF_LOG` diagnostics.
- **Credential handling flaws** — the `token` not marked sensitive, or endpoints/credentials exposed in error messages.
- **Supply-chain integrity** — issues with the release pipeline, GPG signing, checksums, or a dependency that could let a malicious build reach the registry.
- **Provider logic that weakens the platform's guarantees** — e.g. a code path that mutates runtime state the provider is supposed to treat as read-only.

### Response timeline

| Step | Timeline |
|------|----------|
| Acknowledgment | Within 2 business days |
| Initial assessment & severity triage | Within 5 business days |
| Fix released | Depends on severity, alongside a patch release |
| Public disclosure | After the fix ships, coordinated with the reporter |

## Supported versions

The provider follows a rolling release model on its own semver line. The **latest release** receives security patches; older minors are best-effort for critical issues only.

## Security model

- **Secrets stay out of state.** The `token` argument is marked `Sensitive`; prefer the `BRIDGEPORT_TOKEN` environment variable so it never lands in configuration or state. Secret *values* (when secret resources land) use write-only arguments by design.
- **Least privilege.** Use a scoped BridgePort **service-account** token with only the role and environments the run needs.
- **Signed releases.** Release artifacts are GPG-signed and published with `SHA256SUMS`; the registry verifies them against the published signing key.

## Contact

- **Security reports**: security@bridgein.pt
- **General questions**: [BridgePort Discussions](https://github.com/bridgeinpt/bridgeport/discussions)
