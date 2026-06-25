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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &secretResource{}
	_ resource.ResourceWithConfigure   = &secretResource{}
	_ resource.ResourceWithImportState = &secretResource{}
)

// NewSecretResource is a resource.Resource factory.
func NewSecretResource() resource.Resource {
	return &secretResource{}
}

type secretResource struct {
	client *bpclient.Client
}

// secretResourceModel is the Terraform state for a managed secret. The secret
// value is a write-only argument (`value_wo`) — it is sent to the API but never
// stored in state. `value_wo_version` is the rotation trigger: bump it (with a
// new `value_wo`) to rotate the secret, since Terraform cannot diff a write-only
// value on its own.
type secretResourceModel struct {
	Environment    types.String `tfsdk:"environment"`
	Key            types.String `tfsdk:"key"`
	ValueWO        types.String `tfsdk:"value_wo"`
	ValueWOVersion types.String `tfsdk:"value_wo_version"`
	Description    types.String `tfsdk:"description"`
	NeverReveal    types.Bool   `tfsdk:"never_reveal"`
	ID             types.String `tfsdk:"id"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (r *secretResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

func (r *secretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a secret in an environment. The secret value is a **write-only** argument " +
			"(`value_wo`) — it is sent to BridgePort but never stored in Terraform state. To rotate the value, " +
			"change `value_wo` and bump `value_wo_version`.",
		Attributes: map[string]schema.Attribute{
			"environment": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the environment the secret belongs to. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The secret key (its natural key). Must match `^[A-Z][A-Z0-9_]*$`. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"value_wo": schema.StringAttribute{
				Required:            true,
				WriteOnly:           true,
				Sensitive:           true,
				MarkdownDescription: "The secret value. Write-only: it is sent to the API but never persisted in state. Requires Terraform 1.11+.",
			},
			"value_wo_version": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "An arbitrary version string for the secret value. Change it together with `value_wo` to rotate the secret.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Human-friendly description of the secret.",
			},
			"never_reveal": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "If true, the secret value can never be revealed through the UI/API after creation.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Opaque server-assigned identifier for the secret.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp of when the secret was created.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp of when the secret was last updated.",
			},
		},
	}
}

func (r *secretResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *secretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan secretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Write-only values live only in the configuration, never in plan/state.
	var valueWO types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("value_wo"), &valueWO)...)
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

	secret, err := r.client.CreateSecret(env.ID, bpclient.CreateSecretRequest{
		Key:         plan.Key.ValueString(),
		Value:       valueWO.ValueString(),
		Description: plan.Description.ValueStringPointer(),
		NeverReveal: plan.NeverReveal.ValueBoolPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create BridgePort secret",
			fmt.Sprintf("Could not create secret %q in environment %q: %s", plan.Key.ValueString(), plan.Environment.ValueString(), err.Error()),
		)
		return
	}

	plan.NeverReveal = types.BoolValue(secret.NeverReveal)
	plan.ID = types.StringValue(secret.ID)
	plan.CreatedAt = types.StringValue(secret.CreatedAt)
	plan.UpdatedAt = types.StringValue(secret.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *secretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state secretResourceModel
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

	secret, err := r.client.GetSecretByKey(env.ID, state.Key.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read BridgePort secret", err.Error())
		return
	}

	// The value is never read back (write-only); value_wo stays null and
	// value_wo_version is left as the user set it.
	state.ID = types.StringValue(secret.ID)
	state.Key = types.StringValue(secret.Key)
	state.Description = stringOrNull(secret.Description)
	state.NeverReveal = types.BoolValue(secret.NeverReveal)
	state.CreatedAt = types.StringValue(secret.CreatedAt)
	state.UpdatedAt = types.StringValue(secret.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *secretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state secretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	desc := plan.Description.ValueString()
	updateReq := bpclient.UpdateSecretRequest{
		Description: &desc,
		NeverReveal: plan.NeverReveal.ValueBoolPointer(),
	}

	// Rotate the value only when the version trigger changes.
	if !plan.ValueWOVersion.Equal(state.ValueWOVersion) {
		var valueWO types.String
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("value_wo"), &valueWO)...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.Value = valueWO.ValueStringPointer()
	}

	secret, err := r.client.UpdateSecret(state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update BridgePort secret",
			fmt.Sprintf("Could not update secret %q: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	plan.NeverReveal = types.BoolValue(secret.NeverReveal)
	plan.ID = types.StringValue(secret.ID)
	plan.CreatedAt = types.StringValue(secret.CreatedAt)
	plan.UpdatedAt = types.StringValue(secret.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *secretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state secretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteSecret(state.ID.ValueString()); err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to delete BridgePort secret",
			fmt.Sprintf("Could not delete secret %q: %s", state.ID.ValueString(), err.Error()),
		)
	}
}

// ImportState accepts the natural key "environment/key". The secret value cannot
// be recovered, so after import you must add `value_wo` and `value_wo_version`
// to the configuration.
func (r *secretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"environment/key\", got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key"), parts[1])...)
}
