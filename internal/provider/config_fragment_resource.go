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
	_ resource.Resource                = &configFragmentResource{}
	_ resource.ResourceWithConfigure   = &configFragmentResource{}
	_ resource.ResourceWithImportState = &configFragmentResource{}
)

// NewConfigFragmentResource is a resource.Resource factory.
func NewConfigFragmentResource() resource.Resource {
	return &configFragmentResource{}
}

type configFragmentResource struct {
	client *bpclient.Client
}

// configFragmentResourceModel is the Terraform state for a managed config
// fragment: reusable config text included by config files.
type configFragmentResourceModel struct {
	Environment   types.String `tfsdk:"environment"`
	Name          types.String `tfsdk:"name"`
	Content       types.String `tfsdk:"content"`
	Description   types.String `tfsdk:"description"`
	ID            types.String `tfsdk:"id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func (r *configFragmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config_fragment"
}

func (r *configFragmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a reusable config fragment — config text that can be included by one or " +
			"more `bridgeport_config_file` resources.",
		Attributes: map[string]schema.Attribute{
			"environment": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the environment the fragment belongs to. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The unique name of the fragment within its environment (its natural key).",
			},
			"content": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The fragment's text content.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Human-friendly description of the fragment.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Opaque server-assigned identifier for the fragment.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Opaque identifier of the environment the fragment belongs to.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp of when the fragment was created.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp of when the fragment was last updated.",
			},
		},
	}
}

func (r *configFragmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *configFragmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan configFragmentResourceModel
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

	frag, err := r.client.CreateConfigFragment(env.ID, bpclient.CreateConfigFragmentRequest{
		Name:        plan.Name.ValueString(),
		Content:     plan.Content.ValueString(),
		Description: plan.Description.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create BridgePort config fragment",
			fmt.Sprintf("Could not create fragment %q in environment %q: %s", plan.Name.ValueString(), plan.Environment.ValueString(), err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(frag.ID)
	plan.EnvironmentID = types.StringValue(env.ID)
	plan.CreatedAt = types.StringValue(frag.CreatedAt)
	plan.UpdatedAt = types.StringValue(frag.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *configFragmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state configFragmentResourceModel
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

	// After import only the natural key is set; resolve the ID from the name.
	id := state.ID.ValueString()
	if id == "" {
		frags, err := r.client.ListConfigFragments(env.ID)
		if err != nil {
			resp.Diagnostics.AddError("Unable to list BridgePort config fragments", err.Error())
			return
		}
		for i := range frags {
			if frags[i].Name == state.Name.ValueString() {
				id = frags[i].ID
				break
			}
		}
		if id == "" {
			resp.State.RemoveResource(ctx)
			return
		}
	}

	frag, err := r.client.GetConfigFragment(id)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read BridgePort config fragment", err.Error())
		return
	}

	state.ID = types.StringValue(frag.ID)
	state.Name = types.StringValue(frag.Name)
	state.Content = types.StringValue(frag.Content)
	state.Description = stringOrNull(frag.Description)
	state.EnvironmentID = types.StringValue(env.ID)
	state.CreatedAt = types.StringValue(frag.CreatedAt)
	state.UpdatedAt = types.StringValue(frag.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *configFragmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state configFragmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	desc := plan.Description.ValueString()
	frag, err := r.client.UpdateConfigFragment(state.ID.ValueString(), bpclient.UpdateConfigFragmentRequest{
		Name:        plan.Name.ValueStringPointer(),
		Content:     plan.Content.ValueStringPointer(),
		Description: &desc,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update BridgePort config fragment",
			fmt.Sprintf("Could not update fragment %q: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(frag.ID)
	plan.EnvironmentID = types.StringValue(state.EnvironmentID.ValueString())
	plan.CreatedAt = types.StringValue(frag.CreatedAt)
	plan.UpdatedAt = types.StringValue(frag.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *configFragmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state configFragmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteConfigFragment(state.ID.ValueString()); err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to delete BridgePort config fragment",
			fmt.Sprintf("Could not delete fragment %q: %s", state.ID.ValueString(), err.Error()),
		)
	}
}

// ImportState accepts the natural key "environment/name".
func (r *configFragmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
