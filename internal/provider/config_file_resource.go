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
	_ resource.Resource                = &configFileResource{}
	_ resource.ResourceWithConfigure   = &configFileResource{}
	_ resource.ResourceWithImportState = &configFileResource{}
)

// NewConfigFileResource is a resource.Resource factory.
func NewConfigFileResource() resource.Resource {
	return &configFileResource{}
}

type configFileResource struct {
	client *bpclient.Client
}

// configFileResourceModel is the Terraform state for a managed (text) config
// file. fragment_ids are write-through: the API does not return them on read, so
// they are tracked from configuration and not refreshed for drift.
type configFileResourceModel struct {
	Environment   types.String `tfsdk:"environment"`
	Name          types.String `tfsdk:"name"`
	Filename      types.String `tfsdk:"filename"`
	Content       types.String `tfsdk:"content"`
	Description   types.String `tfsdk:"description"`
	Language      types.String `tfsdk:"language"`
	FragmentIDs   types.List   `tfsdk:"fragment_ids"`
	ID            types.String `tfsdk:"id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	SyncStatus    types.String `tfsdk:"sync_status"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func (r *configFileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config_file"
}

func (r *configFileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a text config file in an environment, optionally composed from " +
			"`bridgeport_config_fragment` fragments. Binary files are not managed by this resource.",
		Attributes: map[string]schema.Attribute{
			"environment": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the environment the config file belongs to. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The unique name of the config file within its environment (its natural key).",
			},
			"filename": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The on-disk filename the content is written to.",
			},
			"content": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The file's text content.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Human-friendly description of the config file.",
			},
			"language": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Syntax/language hint for the content (e.g. `yaml`, `json`).",
			},
			"fragment_ids": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				MarkdownDescription: "Ordered list of `bridgeport_config_fragment` IDs to include in the file. " +
					"Note: the API does not return fragment associations on read, so changes made outside Terraform are not detected.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Opaque server-assigned identifier for the config file.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Opaque identifier of the environment the config file belongs to.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sync_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current sync status of the file across servers (runtime, reference only).",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp of when the config file was created.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp of when the config file was last updated.",
			},
		},
	}
}

func (r *configFileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *configFileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan configFileResourceModel
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

	fragmentIDs, diags := listToStrings(ctx, plan.FragmentIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := bpclient.CreateConfigFileRequest{
		Name:        plan.Name.ValueString(),
		Filename:    plan.Filename.ValueString(),
		Content:     plan.Content.ValueString(),
		Description: plan.Description.ValueStringPointer(),
		FragmentIDs: fragmentIDs,
	}
	if !plan.Language.IsNull() && !plan.Language.IsUnknown() {
		createReq.Language = plan.Language.ValueStringPointer()
	}

	file, err := r.client.CreateConfigFile(env.ID, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create BridgePort config file",
			fmt.Sprintf("Could not create config file %q in environment %q: %s", plan.Name.ValueString(), plan.Environment.ValueString(), err.Error()),
		)
		return
	}

	plan.Language = types.StringValue(file.Language)
	plan.ID = types.StringValue(file.ID)
	plan.EnvironmentID = types.StringValue(env.ID)
	plan.SyncStatus = types.StringValue(file.SyncStatus)
	plan.CreatedAt = types.StringValue(file.CreatedAt)
	plan.UpdatedAt = types.StringValue(file.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *configFileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state configFileResourceModel
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
		files, err := r.client.ListConfigFiles(env.ID)
		if err != nil {
			resp.Diagnostics.AddError("Unable to list BridgePort config files", err.Error())
			return
		}
		for i := range files {
			if files[i].Name == state.Name.ValueString() {
				id = files[i].ID
				break
			}
		}
		if id == "" {
			resp.State.RemoveResource(ctx)
			return
		}
	}

	file, err := r.client.GetConfigFile(id)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read BridgePort config file", err.Error())
		return
	}

	state.ID = types.StringValue(file.ID)
	state.Name = types.StringValue(file.Name)
	state.Filename = types.StringValue(file.Filename)
	state.Content = types.StringValue(file.Content)
	state.Description = stringOrNull(file.Description)
	state.Language = types.StringValue(file.Language)
	state.EnvironmentID = types.StringValue(env.ID)
	state.SyncStatus = types.StringValue(file.SyncStatus)
	state.CreatedAt = types.StringValue(file.CreatedAt)
	state.UpdatedAt = types.StringValue(file.UpdatedAt)
	// fragment_ids are intentionally left as-is: the API does not return them.

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *configFileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state configFileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fragmentIDs, diags := listToStrings(ctx, plan.FragmentIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	desc := plan.Description.ValueString()
	updateReq := bpclient.UpdateConfigFileRequest{
		Name:        plan.Name.ValueStringPointer(),
		Filename:    plan.Filename.ValueStringPointer(),
		Content:     plan.Content.ValueStringPointer(),
		Description: &desc,
		Language:    plan.Language.ValueStringPointer(),
		FragmentIDs: fragmentIDs,
	}

	file, err := r.client.UpdateConfigFile(state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update BridgePort config file",
			fmt.Sprintf("Could not update config file %q: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	plan.Language = types.StringValue(file.Language)
	plan.ID = types.StringValue(file.ID)
	plan.EnvironmentID = types.StringValue(state.EnvironmentID.ValueString())
	plan.SyncStatus = types.StringValue(file.SyncStatus)
	plan.CreatedAt = types.StringValue(file.CreatedAt)
	plan.UpdatedAt = types.StringValue(file.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *configFileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state configFileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteConfigFile(state.ID.ValueString()); err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to delete BridgePort config file",
			fmt.Sprintf("Could not delete config file %q: %s", state.ID.ValueString(), err.Error()),
		)
	}
}

// ImportState accepts the natural key "environment/name". fragment_ids cannot be
// recovered on import; re-declare them in configuration afterward.
func (r *configFileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
