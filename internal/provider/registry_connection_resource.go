// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

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
	_ resource.Resource                = &registryConnectionResource{}
	_ resource.ResourceWithConfigure   = &registryConnectionResource{}
	_ resource.ResourceWithImportState = &registryConnectionResource{}
)

// NewRegistryConnectionResource is a resource.Resource factory.
func NewRegistryConnectionResource() resource.Resource {
	return &registryConnectionResource{}
}

type registryConnectionResource struct {
	client *bpclient.Client
}

// registryConnectionResourceModel is the Terraform state for a managed registry
// connection. Credentials (token, password) are write-only: they are sent to the
// API but never stored in state; the API exposes only has_token/has_password.
type registryConnectionResourceModel struct {
	Environment            types.String `tfsdk:"environment"`
	Name                   types.String `tfsdk:"name"`
	Type                   types.String `tfsdk:"type"`
	RegistryURL            types.String `tfsdk:"registry_url"`
	RepositoryPrefix       types.String `tfsdk:"repository_prefix"`
	Username               types.String `tfsdk:"username"`
	TokenWO                types.String `tfsdk:"token_wo"`
	TokenWOVersion         types.String `tfsdk:"token_wo_version"`
	PasswordWO             types.String `tfsdk:"password_wo"`
	PasswordWOVersion      types.String `tfsdk:"password_wo_version"`
	IsDefault              types.Bool   `tfsdk:"is_default"`
	RefreshIntervalMinutes types.Int64  `tfsdk:"refresh_interval_minutes"`
	AutoLinkPattern        types.String `tfsdk:"auto_link_pattern"`
	ID                     types.String `tfsdk:"id"`
	EnvironmentID          types.String `tfsdk:"environment_id"`
	HasToken               types.Bool   `tfsdk:"has_token"`
	HasPassword            types.Bool   `tfsdk:"has_password"`
	ImageCount             types.Int64  `tfsdk:"image_count"`
	CreatedAt              types.String `tfsdk:"created_at"`
	UpdatedAt              types.String `tfsdk:"updated_at"`
}

func (r *registryConnectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_registry_connection"
}

func (r *registryConnectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a container registry connection in an environment. Credentials (`token_wo`, " +
			"`password_wo`) are **write-only** — sent to BridgePort but never stored in Terraform state. " +
			"Rotate a credential by changing it together with its `*_version`.",
		Attributes: map[string]schema.Attribute{
			"environment": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the environment the registry belongs to. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The unique name of the registry connection within its environment (its natural key).",
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Registry type: `digitalocean`, `dockerhub`, or `generic`.",
			},
			"registry_url": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Base URL of the registry.",
			},
			"repository_prefix": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional repository prefix applied to image names.",
			},
			"username": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Username for registries that authenticate with username + password.",
			},
			"token_wo": schema.StringAttribute{
				Optional:            true,
				WriteOnly:           true,
				Sensitive:           true,
				MarkdownDescription: "Write-only access token (for token-based registries). Requires Terraform 1.11+.",
			},
			"token_wo_version": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Version string for `token_wo`; change it together with `token_wo` to rotate the token.",
			},
			"password_wo": schema.StringAttribute{
				Optional:            true,
				WriteOnly:           true,
				Sensitive:           true,
				MarkdownDescription: "Write-only password (for username/password registries). Requires Terraform 1.11+.",
			},
			"password_wo_version": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Version string for `password_wo`; change it together with `password_wo` to rotate the password.",
			},
			"is_default": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether this is the environment's default registry.",
			},
			"refresh_interval_minutes": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "How often (minutes) BridgePort refreshes tags from the registry.",
			},
			"auto_link_pattern": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional pattern for auto-linking discovered images.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Opaque server-assigned identifier for the registry connection.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Opaque identifier of the environment the registry belongs to.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"has_token": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether an access token is configured.",
			},
			"has_password": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether a password is configured.",
			},
			"image_count": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of container images linked to this registry.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp of when the registry connection was created.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp of when the registry connection was last updated.",
			},
		},
	}
}

func (r *registryConnectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*bpclient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *registryConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan registryConnectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var tokenWO, passwordWO types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("token_wo"), &tokenWO)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("password_wo"), &passwordWO)...)
	if resp.Diagnostics.HasError() {
		return
	}

	env, err := r.client.GetEnvironmentByName(plan.Environment.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to resolve BridgePort environment",
			fmt.Sprintf("Could not resolve environment %q: %s", plan.Environment.ValueString(), err.Error()),
		)
		return
	}

	createReq := bpclient.CreateRegistryRequest{
		Name:             plan.Name.ValueString(),
		Type:             plan.Type.ValueString(),
		RegistryURL:      plan.RegistryURL.ValueString(),
		RepositoryPrefix: plan.RepositoryPrefix.ValueStringPointer(),
		Username:         plan.Username.ValueStringPointer(),
		Token:            tokenWO.ValueStringPointer(),
		Password:         passwordWO.ValueStringPointer(),
		AutoLinkPattern:  plan.AutoLinkPattern.ValueStringPointer(),
	}
	if !plan.IsDefault.IsNull() && !plan.IsDefault.IsUnknown() {
		createReq.IsDefault = plan.IsDefault.ValueBoolPointer()
	}
	if !plan.RefreshIntervalMinutes.IsNull() && !plan.RefreshIntervalMinutes.IsUnknown() {
		v := int(plan.RefreshIntervalMinutes.ValueInt64())
		createReq.RefreshIntervalMinutes = &v
	}

	reg, err := r.client.CreateRegistry(env.ID, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create BridgePort registry connection",
			fmt.Sprintf("Could not create registry %q in environment %q: %s", plan.Name.ValueString(), plan.Environment.ValueString(), err.Error()),
		)
		return
	}

	r.applyComputed(&plan, env.ID, reg)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *registryConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state registryConnectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	env, err := r.client.GetEnvironmentByName(state.Environment.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to resolve BridgePort environment", err.Error())
		return
	}

	reg, err := r.client.GetRegistryByName(env.ID, state.Name.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read BridgePort registry connection", err.Error())
		return
	}

	// GetRegistryByName returns the full record, so every field round-trips and
	// out-of-band drift is detected. Credentials remain write-only (only
	// has_token/has_password are exposed).
	state.Name = types.StringValue(reg.Name)
	state.Type = types.StringValue(reg.Type)
	state.RegistryURL = types.StringValue(reg.RegistryURL)
	state.RepositoryPrefix = stringOrNull(reg.RepositoryPrefix)
	state.Username = stringOrNull(reg.Username)
	state.AutoLinkPattern = stringOrNull(reg.AutoLinkPattern)
	r.applyComputed(&state, env.ID, reg)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *registryConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state registryConnectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := bpclient.UpdateRegistryRequest{
		Name:             plan.Name.ValueStringPointer(),
		Type:             plan.Type.ValueStringPointer(),
		RegistryURL:      plan.RegistryURL.ValueStringPointer(),
		RepositoryPrefix: plan.RepositoryPrefix.ValueStringPointer(),
		Username:         plan.Username.ValueStringPointer(),
		AutoLinkPattern:  plan.AutoLinkPattern.ValueStringPointer(),
	}
	if !plan.IsDefault.IsNull() && !plan.IsDefault.IsUnknown() {
		updateReq.IsDefault = plan.IsDefault.ValueBoolPointer()
	}
	if !plan.RefreshIntervalMinutes.IsNull() && !plan.RefreshIntervalMinutes.IsUnknown() {
		v := int(plan.RefreshIntervalMinutes.ValueInt64())
		updateReq.RefreshIntervalMinutes = &v
	}
	// Rotate credentials only when their version trigger changes.
	if !plan.TokenWOVersion.Equal(state.TokenWOVersion) {
		var tokenWO types.String
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("token_wo"), &tokenWO)...)
		updateReq.Token = tokenWO.ValueStringPointer()
	}
	if !plan.PasswordWOVersion.Equal(state.PasswordWOVersion) {
		var passwordWO types.String
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("password_wo"), &passwordWO)...)
		updateReq.Password = passwordWO.ValueStringPointer()
	}
	if resp.Diagnostics.HasError() {
		return
	}

	reg, err := r.client.UpdateRegistry(state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update BridgePort registry connection",
			fmt.Sprintf("Could not update registry %q: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	r.applyComputed(&plan, state.EnvironmentID.ValueString(), reg)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *registryConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state registryConnectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteRegistry(state.ID.ValueString()); err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to delete BridgePort registry connection",
			fmt.Sprintf("Could not delete registry %q: %s", state.ID.ValueString(), err.Error()),
		)
	}
}

// ImportState accepts the natural key "environment/name". Credentials cannot be
// recovered; re-declare token_wo/password_wo (and their versions) after import.
func (r *registryConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"environment/name\", got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
}

// applyComputed maps the server-assigned and computed fields onto the model.
func (r *registryConnectionResource) applyComputed(m *registryConnectionResourceModel, envID string, reg *bpclient.Registry) {
	m.ID = types.StringValue(reg.ID)
	m.EnvironmentID = types.StringValue(envID)
	m.IsDefault = types.BoolValue(reg.IsDefault)
	m.RefreshIntervalMinutes = types.Int64Value(int64(reg.RefreshIntervalMinutes))
	m.HasToken = types.BoolValue(reg.HasToken)
	m.HasPassword = types.BoolValue(reg.HasPassword)
	m.ImageCount = types.Int64Value(int64(reg.ImageCount))
	m.CreatedAt = types.StringValue(reg.CreatedAt)
	m.UpdatedAt = types.StringValue(reg.UpdatedAt)
}
