# Managed resources — implementation blueprint (#5)

> **Status: blocked on the SDK.** The BridgePort Go SDK (`github.com/bridgeinpt/bridgeport/client`)
> is read-only as of `client/v0.1.0`. Verified against platform `v3.0.0`: the `client/`
> package is byte-identical to `v0.1.0` — no `Create*/Update*/Delete*` methods, no
> health/version method. This blueprint is staged so that, the moment a `client/vX.Y.Z`
> tag with write methods lands, implementation is mechanical.
>
> This doc is **developer documentation**, deliberately kept out of `docs/` (which is
> tfplugindocs-generated registry content).

## 0. Unblock sequence

1. Upstream (`bridgeinpt/bridgeport`, `client/`): add the write methods in the
   [SDK contract](#2-sdk-contract-what-upstream-must-add) below, release as a new
   `client/vX.Y.Z` tag.
2. Here: `go get github.com/bridgeinpt/bridgeport/client@vX.Y.Z` → `make bootstrap`
   → commit the `go.mod`/`go.sum` diff (one prep PR, or folded into the first resource PR).
3. Implement resources in [dependency order](#5-dependency-order--per-resource-notes),
   **one PR per resource**, each: schema + CRUD + `ImportState` + acceptance tests +
   `make generate`, registered in `provider.go`'s `Resources()`.

## 1. Non-negotiable design tenets (from `CLAUDE.md`)

These shape every resource below — they are the reason the provider is safe to adopt:

1. **Configuration only.** Never model deploys/restarts/rollbacks/live status as managed
   state. Runtime fields (`status`, `container_status`, `health_status`, ports, discovery)
   are `Computed`, never Required/Optional inputs.
2. **Secrets never enter state.** Secret *values* are **write-only** arguments paired with
   a `version`/rotation attribute; never read a secret value back into state.
3. **`plan` works offline.** No live API calls at plan/validate; diff against submitted
   config, not live runtime.
4. **Natural-key import.** `terraform import` keys on `environment` + `name`/`key`.

## 2. SDK contract (what upstream must add)

Signatures below are **proposed** — match them to whatever the released SDK actually exposes;
the provider-side pattern in §3–4 is stable regardless of exact names. Each resource needs the
CRUD quartet (Create/Read/Update/Delete); reads (`Get*`) already exist for most.

| Resource | Create | Update | Delete | Read (exists?) |
|---|---|---|---|---|
| `server` | `CreateServer(envID, ServerInput) (*Server, error)` | `UpdateServer(id, ServerInput) (*Server, error)` | `DeleteServer(id) error` | ✅ `GetServer`, `GetServerByEnvAndName` |
| `var` | `CreateVar(envID, VarInput) (*Var, error)` | `UpdateVar(id, VarInput) (*Var, error)` | `DeleteVar(id) error` | ⚠️ no `Var` type yet |
| `secret` | `CreateSecret(envID, SecretInput) (*Secret, error)` | `UpdateSecret(id, SecretInput) (*Secret, error)` | `DeleteSecret(id) error` | ✅ `ListSecrets` (metadata only — never value) |
| `config_file` | `CreateConfigFile(envID, ConfigFileInput) (*ConfigFile, error)` | `UpdateConfigFile(id, …)` | `DeleteConfigFile(id) error` | ✅ `ListConfigFiles` |
| `registry_connection` | `CreateRegistry(envID, RegistryInput) (*Registry, error)` | `UpdateRegistry(id, …)` | `DeleteRegistry(id) error` | ✅ `ListRegistries` |
| `container_image` | `CreateContainerImage(envID, …)` | `UpdateContainerImage(id, …)` | `DeleteContainerImage(id) error` | ✅ `ListContainerImages` |
| `service` | `CreateService(serverID, ServiceInput) (*Service, error)` | `UpdateService(id, …)` | `DeleteService(id) error` | ✅ `GetService`, `GetServiceByName`, `ListServices` |

`*Input` request structs must expose **only** configuration fields. The platform's
`READONLY_FIELD` registry maps 1:1 to the config/computed split — keep them aligned.

## 3. Provider resource pattern

Mirror the data-source files (`internal/provider/*_data_source.go`). Each resource:

- Satisfies `resource.Resource`, `resource.ResourceWithConfigure`, `resource.ResourceWithImportState`.
- Asserts `req.ProviderData.(*client.Client)` in `Configure` with a clear diagnostic (same as data sources).
- Splits the schema:
  - **Required/Optional** = configuration only. Natural-key parts (`environment`, `name`/`key`)
    use `RequiresReplace` plan modifiers (changing them is a recreate, not an in-place update).
  - **Computed** = server-assigned (`id`, `created_at`) and all runtime fields.
- `Create` → SDK create → map response to state. `Read` → SDK get by id (or natural key) →
  refresh state; if 404, `resp.State.RemoveResource(ctx)`. `Update` → SDK patch. `Delete` → SDK delete.
- `ImportState` parses `environment/name` (or `environment/server/name` for services) and seeds
  the natural-key attributes, then `Read` hydrates the rest.
- Reuse the existing `*ToModel` mappers where the data-source model matches.

## 4. Reference implementation — `bridgeport_server` resource

Drop-in skeleton (illustrative; wire to the real SDK method names when the tag lands).
Schema fields are a starting point — confirm which are truly configurable vs runtime against
the released API.

```go
// internal/provider/server_resource.go
package provider

import (
	"context"
	"fmt"
	"strings"

	bpclient "github.com/bridgeinpt/bridgeport/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &serverResource{}
	_ resource.ResourceWithConfigure   = &serverResource{}
	_ resource.ResourceWithImportState = &serverResource{}
)

func NewServerResource() resource.Resource { return &serverResource{} }

type serverResource struct{ client *bpclient.Client }

// Reuses serverDataSourceModel's shape; a managed resource adds plan modifiers.
type serverResourceModel struct {
	Environment   types.String `tfsdk:"environment"`    // natural key, RequiresReplace
	Name          types.String `tfsdk:"name"`           // natural key, RequiresReplace
	PrivateIP     types.String `tfsdk:"private_ip"`     // config (confirm vs API)
	PublicIP      types.String `tfsdk:"public_ip"`      // config/optional
	ID            types.String `tfsdk:"id"`             // computed
	EnvironmentID types.String `tfsdk:"environment_id"` // computed
	Status        types.String `tfsdk:"status"`         // computed (runtime)
	CreatedAt     types.String `tfsdk:"created_at"`     // computed
}

func (r *serverResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (r *serverResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a BridgePort server within an environment.",
		Attributes: map[string]schema.Attribute{
			"environment": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Environment (by `name`) the server belongs to.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Unique name of the server within its environment.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"private_ip": schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Private IP address."},
			"public_ip":  schema.StringAttribute{Optional: true, MarkdownDescription: "Public IP address, if any."},
			"id":             schema.StringAttribute{Computed: true, MarkdownDescription: "Server identifier."},
			"environment_id": schema.StringAttribute{Computed: true, MarkdownDescription: "Environment identifier."},
			"status":         schema.StringAttribute{Computed: true, MarkdownDescription: "Runtime status (reference only)."},
			"created_at":     schema.StringAttribute{Computed: true, MarkdownDescription: "RFC 3339 creation timestamp."},
		},
	}
}

func (r *serverResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*bpclient.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData))
		return
	}
	r.client = c
}

func (r *serverResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serverResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	env, err := r.client.GetEnvironmentByName(plan.Environment.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to resolve environment", err.Error())
		return
	}
	// TODO(sdk): replace with the real create method + input struct.
	srv, err := r.client.CreateServer(env.ID, bpclient.ServerInput{
		Name:      plan.Name.ValueString(),
		PrivateIP: plan.PrivateIP.ValueString(),
		PublicIP:  plan.PublicIP.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create BridgePort server", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, applyServer(plan, *srv))...)
}

func (r *serverResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serverResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	srv, err := r.client.GetServer(state.ID.ValueString())
	if err != nil {
		if isNotFound(err) { // map APIError{StatusCode:404}
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read BridgePort server", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, applyServer(state, *srv))...)
}

func (r *serverResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serverResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// TODO(sdk): UpdateServer(id, ServerInput).
	srv, err := r.client.UpdateServer(plan.ID.ValueString(), bpclient.ServerInput{
		Name:      plan.Name.ValueString(),
		PrivateIP: plan.PrivateIP.ValueString(),
		PublicIP:  plan.PublicIP.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to update BridgePort server", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, applyServer(plan, *srv))...)
}

func (r *serverResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serverResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteServer(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to delete BridgePort server", err.Error())
	}
}

// ImportState: "environment/name" natural key.
func (r *serverResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", `Expected "environment/name".`)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
	// Read() then hydrates id + computed fields. (Resolve id from the natural key
	// in Read when ID is null — or look it up here via GetServerByEnvAndName.)
}

func applyServer(m serverResourceModel, s bpclient.Server) serverResourceModel {
	m.ID = types.StringValue(s.ID)
	m.Name = types.StringValue(s.Name)
	m.EnvironmentID = types.StringValue(s.EnvironmentID)
	m.PrivateIP = types.StringValue(s.PrivateIP)
	m.PublicIP = types.StringPointerValue(s.PublicIP)
	m.Status = types.StringValue(s.Status)
	m.CreatedAt = types.StringValue(s.CreatedAt)
	return m
}
```

Plus, in `provider.go`:

```go
func (p *bridgeportProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewServerResource,
		// …added one per PR, in dependency order.
	}
}
```

And an acceptance test (`server_resource_test.go`) using `resource.Test` with create → import →
update → destroy steps, gated on `TF_ACC` against the disposable harness.

## 5. Dependency order & per-resource notes

Land in this order (each builds on the prior), per the README roadmap and epic #197:

1. **`server`** — foundation; reference impl above. Confirm configurable vs runtime fields.
2. **`var` / `secret`** — `secret` is the secrets-never-in-state case: a write-only `value`
   argument + a `version` (or `rotation`) attribute that triggers update; `Read` refreshes only
   metadata (`ListSecrets`), never the value. `var` is the plain (non-secret) analogue.
3. **`config_file` / `config_fragment`** (+ attachments) — content-bearing; watch for large bodies
   and normalization (trailing newline) to avoid perpetual diffs.
4. **`registry_connection` / `container_image`** — registry creds are secret-bearing (apply the
   write-only pattern); `container_image` references a registry + tag.
5. **`service` / `service_deployment`** — top of the stack; `service` is server-scoped
   (`environment` + `server` + `name`). Deployments are runtime — model only the *configuration*
   of a deployment, never its live status.

### Per-resource PR checklist

- [ ] SDK bump committed (`go.mod`/`go.sum`), if not already.
- [ ] `internal/provider/<noun>_resource.go` — schema (config/computed split), CRUD, `ImportState`.
- [ ] Registered in `provider.go` `Resources()`.
- [ ] `examples/resources/bridgeport_<noun>/resource.tf` (+ `import.sh` for the natural key).
- [ ] Acceptance test (`TF_ACC`): create → import → update → destroy.
- [ ] `make generate` to refresh `docs/`.
- [ ] Secrets: no value ever read back into state (for secret-bearing resources).
