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

---

## 1. Before merging PRs (recommended)

### 1a. Grant the engineering team write access  — *needs org admin*
CODEOWNERS (`@bridgeinpt/engineering`) only enforces reviews if the team has **Write** (or Maintain) access. The team exists but is **not** on this repo yet.

- UI: Repo → Settings → Collaborators and teams → Add teams → `engineering` → **Write**.
- CLI (needs `admin:org` / team-maintainer token):
  ```bash
  gh api -X PUT orgs/bridgeinpt/teams/engineering/repos/bridgeinpt/terraform-provider-bridgeport -f permission=push
  ```

### 1b. Enable branch protection on `master`  — *can be run with the bootstrap token*
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

## 2. One-time code hygiene  — *needs a local Go toolchain*

`go.sum` is intentionally **not** committed (the bootstrap environment had no Go). CI regenerates it each run via `go mod tidy`, so it's green without this — but commit it for reproducibility and so Dependabot tracks exact versions:

```bash
make bootstrap                       # == go mod tidy: resolves deps + the BridgePort Go SDK pseudo-version
git add go.mod go.sum
git commit -s -m "chore: lock go modules"
git push
```

While you're there, run `make generate` to confirm the committed `docs/` match the schema (the scaffold's docs are hand-written to match; tfplugindocs is the source of truth going forward).

---

## 3. Before the first release (`vX.Y.Z` tag)  — *needs secret material + org admin*

`release.yml` runs GoReleaser to build GPG-signed assets in the layout the registries ingest. It needs:

### 3a. A GPG signing key
Generate (or reuse an org) signing key and export the **private** key + note the passphrase:
```bash
gpg --full-generate-key                       # RSA 4096, no expiry or a long one
gpg --armor --export-secret-keys <KEY_ID> > private.asc
gpg --armor --export <KEY_ID> > public.asc     # used in step 4
```

### 3b. Repo secrets  — *can be run with the bootstrap token*
```bash
gh secret set GPG_PRIVATE_KEY < private.asc --repo bridgeinpt/terraform-provider-bridgeport
gh secret set PASSPHRASE       --repo bridgeinpt/terraform-provider-bridgeport   # paste the passphrase
rm -f private.asc              # don't leave the private key on disk
```

### 3c. Allowlist the release workflow's third-party actions  — *needs org admin*
`test.yml` is fine, but `release.yml` uses two non-GitHub actions. Add them under
Org → Settings → Actions → General → "Allow select actions" (or repo-level if the org delegates):
- `goreleaser/goreleaser-action@*`
- `crazy-max/ghaction-import-gpg@*`
- (also `actions/checkout`, `actions/setup-go` — usually already covered by "Allow actions created by GitHub").

If you'd rather not allowlist them, the alternative is to install GoReleaser + import the key via plain run-steps; ask and I'll rewrite `release.yml` that way.

---

## 4. Registry publication  — *manual web flows, after the first signed release*

A public repo does **not** auto-publish. After step 3 produces a signed `vX.Y.Z` release:

- **Terraform Registry**: sign in at <https://registry.terraform.io> with GitHub → Publish → Providers → select `terraform-provider-bridgeport` → upload the **GPG public key** (`public.asc`). The registry then ingests existing and future releases.
- **OpenTofu Registry**: submit the provider + GPG key via the process in <https://github.com/opentofu/registry> (a PR to their registry repo).

The `Address` in `main.go` (`registry.terraform.io/bridgeinpt/bridgeport`) and the `source = "bridgeinpt/bridgeport"` in the docs/examples already assume the `bridgeinpt` namespace — make sure your registry account owns it.

---

## 5. Optional / nice-to-have

- **CodeQL code scanning** (Go): Repo → Settings → Security → Code scanning → Set up → Default. (The platform repo uses CodeQL.) Confirm the CodeQL action is allowlisted if the org policy is restrictive.
- **Social preview image**: Repo → Settings → upload a banner (no API for this).
- **Custom labels**: Dependabot auto-creates `dependencies`/`go`/`ci`; `bug`/`enhancement` exist by default. Create others only if you want them pre-seeded.
- **Separate Discussions**: currently routed to the platform repo's Discussions (see `.github/ISSUE_TEMPLATE/config.yml`). Enable Discussions here if you want a provider-specific forum.

---

## What this unblocks

This repo is the prerequisite the platform issue [bridgeinpt/bridgeport#202](https://github.com/bridgeinpt/bridgeport/issues/202) was blocked on. With the provider + its acceptance suite now existing, the platform repo can add the **provider-compat CI job** (check out this repo at its latest release tag, run `TF_ACC=1` against the PR-built image) and attach the OpenAPI snapshot to releases. That work stays in the platform repo.
