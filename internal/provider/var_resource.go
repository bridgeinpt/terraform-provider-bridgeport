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
	_ resource.Resource                = &varResource{}
	_ resource.ResourceWithConfigure   = &varResource{}
	_ resource.ResourceWithImportState = &varResource{}
)

// NewVarResource is a resource.Resource factory.
func NewVarResource() resource.Resource {
	return &varResource{}
}

type varResource struct {
	client *bpclient.Client
}

// varResourceModel is the Terraform state for a managed environment variable.
// Unlike a secret, a var's value is non-secret: the API stores and returns it.
type varResourceModel struct {
	Environment types.String `tfsdk:"environment"`
	Key         types.String `tfsdk:"key"`
	Value       types.String `tfsdk:"value"`
	Description types.String `tfsdk:"description"`
	ID          types.String `tfsdk:"id"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (r *varResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_var"
}

func (r *varResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a non-secret environment variable (a key/value). The value is stored " +
			"in plaintext and readable via the API — use `bridgeport_secret` for sensitive values.",
		Attributes: map[string]schema.Attribute{
			"environment": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the environment the variable belongs to. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The variable key (its natural key). Must match `^[A-Z][A-Z0-9_]*$`. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"value": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The variable value (plaintext).",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Human-friendly description of the variable.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Opaque server-assigned identifier for the variable.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp of when the variable was created.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp of when the variable was last updated.",
			},
		},
	}
}

func (r *varResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *varResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan varResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
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

	v, err := r.client.CreateVar(env.ID, bpclient.CreateVarRequest{
		Key:         plan.Key.ValueString(),
		Value:       plan.Value.ValueString(),
		Description: plan.Description.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create BridgePort variable",
			fmt.Sprintf("Could not create variable %q in environment %q: %s", plan.Key.ValueString(), plan.Environment.ValueString(), err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(v.ID)
	plan.CreatedAt = types.StringValue(v.CreatedAt)
	plan.UpdatedAt = types.StringValue(v.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *varResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state varResourceModel
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

	v, err := r.client.GetVarByKey(env.ID, state.Key.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read BridgePort variable", err.Error())
		return
	}

	state.ID = types.StringValue(v.ID)
	state.Key = types.StringValue(v.Key)
	state.Value = types.StringValue(v.Value)
	state.Description = stringOrNull(v.Description)
	state.CreatedAt = types.StringValue(v.CreatedAt)
	state.UpdatedAt = types.StringValue(v.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *varResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state varResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Always send description (empty string clears it) so removing it from
	// config takes effect rather than leaving the previous value in place.
	desc := plan.Description.ValueString()
	v, err := r.client.UpdateVar(state.ID.ValueString(), bpclient.UpdateVarRequest{
		Value:       plan.Value.ValueStringPointer(),
		Description: &desc,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update BridgePort variable",
			fmt.Sprintf("Could not update variable %q: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(v.ID)
	plan.CreatedAt = types.StringValue(v.CreatedAt)
	plan.UpdatedAt = types.StringValue(v.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *varResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state varResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteVar(state.ID.ValueString()); err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to delete BridgePort variable",
			fmt.Sprintf("Could not delete variable %q: %s", state.ID.ValueString(), err.Error()),
		)
	}
}

// ImportState accepts the natural key "environment/key".
func (r *varResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
