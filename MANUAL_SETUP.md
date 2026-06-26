# Manual setup checklist

This repo was bootstrapped with everything that could be configured via the GitHub API/CLI already applied. This file lists what still needs a **human with org-admin rights** (or external accounts / secret material) to finish. Items are grouped by when you need them.

> Token note: the bootstrap ran with a `repo` + `read:org` token. Anything org-scoped (org actions allowlist, granting a team access) or needing secret material (GPG key) could **not** be automated and is listed below.

---

## ✅ Already configured (no action needed)

- Repo created **public** in `bridgeinpt/`, default branch `master`, wiki/projects/discussions off, issues on.
- Merge policy: **squash-only**, auto-merge on, "update branch" on, delete branch on merge.
- Commit sign-off: enforced (inherited from the org).
- Security: **secret scanning + push protection** on, **Dependabot alerts + security updates** on, **private vulnerability reporting** on.
- Topics set; description set.
- `.github/`: Dependabot (gomod + actions), CODEOWNERS, issue templates, PR template.
- CI `test.yml` (build/vet/lint/unit + acceptance) uses **GitHub-owned actions only**, so it runs without touching the org actions allowlist.
- **`go.mod` + `go.sum` committed and locked** (BridgePort Go SDK `client/v0.1.0`); `go build`/`go vet`/unit tests verified locally with Go 1.25.11. CI is green.

---

## 1. Before merging PRs

### 1a. ✅ Grant the engineering team write access — *done*
`@bridgeinpt/engineering` has been granted access, so CODEOWNERS reviews enforce.

### 1b. ✅ Enable branch protection on `master` — *done*
Applied: block direct pushes, require PR + green CI (`build & vet`, `golangci-lint`, `acceptance …`) + code-owner review, linear history, conversation resolution. Reapply/adjust with:
Mirrors the platform repo (block direct pushes, require PR + green CI + code-owner review). Run after CI is green so you don't lock yourself out mid-iteration:

```bash
gh api -X PUT repos/bridgeinpt/terraform-provider-bridgeport/branches/master/protection \
  --input - <<'JSON'
{
  "required_status_checks": {
    "strict": true,
    "contexts": ["build & vet", "golangci-lint", "acceptance (TF_ACC) against built image"]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": {
    "required_approving_review_count": 1,
    "require_code_owner_reviews": true
  },
  "restrictions": null,
  "required_linear_history": true,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "required_conversation_resolution": true
}
JSON
```
(Adjust the check names if you rename the CI jobs. Drop `acceptance …` from `contexts` if you later make acceptance nightly instead of PR-blocking.)

---

## 2. ✅ Code hygiene — *done*

`go.mod` + `go.sum` are committed and locked (BridgePort Go SDK `client/v0.1.0`); `go build`/`go vet`/unit tests pass locally and in CI. CI also enforces `go mod tidy` produces no diff.

Going forward, run `make generate` if you change a schema or example to refresh the committed `docs/`. tfplugindocs (pinned in `main.go`'s `//go:generate` directive) is the source of truth, and CI fails if the committed `docs/` drift from its output.

---

## 3. ✅ First release prerequisites — *done*

`release.yml` runs GoReleaser to build GPG-signed assets in the layout the registries ingest. All three prerequisites are in place (verified by the signed `v0.1.0` release); the commands are kept below for key rotation / re-bootstrapping.

### 3a. ✅ GPG signing key — *done*
A GPG signing key (key ID `03DF9BCABE6A2FAC`) was generated; its public key is registered with the Terraform Registry (step 4). To rotate/regenerate:
```bash
gpg --full-generate-key                       # RSA 4096, no expiry or a long one
gpg --armor --export-secret-keys <KEY_ID> > private.asc
gpg --armor --export <KEY_ID> > public.asc     # used in step 4
```
(`*.asc` / `*.gpg` are git-ignored, so exported key files never land in the repo.)

### 3b. ✅ Repo secrets — *done*
`GPG_PRIVATE_KEY` and `PASSPHRASE` are set on the repo. To reapply (e.g. after rotation):
```bash
gh secret set GPG_PRIVATE_KEY < private.asc --repo bridgeinpt/terraform-provider-bridgeport
gh secret set PASSPHRASE       --repo bridgeinpt/terraform-provider-bridgeport   # paste the passphrase
rm -f private.asc              # don't leave the private key on disk
```

### 3c. ✅ Allowlist the release workflow's third-party actions — *done*
`goreleaser/goreleaser-action@*` and `crazy-max/ghaction-import-gpg@*` are allowlisted in the org Actions policy (Org → Settings → Actions → General → "Allow select actions"). `actions/checkout` and `actions/setup-go` are covered by "Allow actions created by GitHub".

---

## 4. Registry publication

A public repo does **not** auto-publish; this is done per registry once a signed release exists.

- **Terraform Registry**: ✅ *done* — the `bridgeinpt` public namespace is claimed, the provider `bridgeinpt/bridgeport` is published, and the GPG public key is uploaded. **v0.1.0 is live** and verified end-to-end (`terraform init` installs it and validates the signature): <https://registry.terraform.io/providers/bridgeinpt/bridgeport/latest>
- **OpenTofu Registry**: ⬜ *pending* — submit the provider + GPG key via a PR to <https://github.com/opentofu/registry>.

The `Address` in `main.go` (`registry.terraform.io/bridgeinpt/bridgeport`) and the `source = "bridgeinpt/bridgeport"` in the docs/examples assume the `bridgeinpt` namespace. On the public registry the namespace **is** the GitHub account/org that owns the repo — it can't be chosen arbitrarily, so `bridgeinpt` is correct here.

> **Note on the HCP Terraform organization:** publishing now goes through an HCP Terraform org (app.terraform.io). That org's name is arbitrary and unrelated to the public namespace — the namespace `bridgeinpt` is claimed by linking the `bridgeinpt` **GitHub** account to the org.

---

## 5. Optional / nice-to-have

- **CodeQL code scanning** (Go): Repo → Settings → Security → Code scanning → Set up → Default. (The platform repo uses CodeQL.) Confirm the CodeQL action is allowlisted if the org policy is restrictive.
- **Social preview image**: Repo → Settings → upload a banner (no API for this).
- **Custom labels**: Dependabot auto-creates `dependencies`/`go`/`ci`; `bug`/`enhancement` exist by default. Create others only if you want them pre-seeded.
- **Separate Discussions**: currently routed to the platform repo's Discussions (see `.github/ISSUE_TEMPLATE/config.yml`). Enable Discussions here if you want a provider-specific forum.

---

## What this unblocks

This repo is the prerequisite the platform issue [bridgeinpt/bridgeport#202](https://github.com/bridgeinpt/bridgeport/issues/202) was blocked on. With the provider + its acceptance suite now existing, the platform repo can add the **provider-compat CI job** (check out this repo at its latest release tag, run `TF_ACC=1` against the PR-built image) and attach the OpenAPI snapshot to releases. That work stays in the platform repo.
