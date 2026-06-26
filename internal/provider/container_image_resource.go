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
	_ resource.Resource                = &containerImageResource{}
	_ resource.ResourceWithConfigure   = &containerImageResource{}
	_ resource.ResourceWithImportState = &containerImageResource{}
)

// NewContainerImageResource is a resource.Resource factory.
func NewContainerImageResource() resource.Resource {
	return &containerImageResource{}
}

type containerImageResource struct {
	client *bpclient.Client
}

// containerImageResourceModel is the Terraform state for a managed container
// image. Its natural key is environment + image_name.
type containerImageResourceModel struct {
	Environment          types.String `tfsdk:"environment"`
	Name                 types.String `tfsdk:"name"`
	ImageName            types.String `tfsdk:"image_name"`
	TagFilter            types.String `tfsdk:"tag_filter"`
	RegistryConnectionID types.String `tfsdk:"registry_connection_id"`
	AutoUpdate           types.Bool   `tfsdk:"auto_update"`
	ID                   types.String `tfsdk:"id"`
	EnvironmentID        types.String `tfsdk:"environment_id"`
	CurrentTag           types.String `tfsdk:"current_tag"`
	LatestTag            types.String `tfsdk:"latest_tag"`
	UpdateAvailable      types.Bool   `tfsdk:"update_available"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
}

func (r *containerImageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_image"
}

func (r *containerImageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a container image tracked in an environment, optionally linked to a " +
			"`bridgeport_registry_connection`.",
		Attributes: map[string]schema.Attribute{
			"environment": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the environment the image belongs to. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Display name for the tracked image.",
			},
			"image_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The image repository name (e.g. `nginx`), the natural key within the environment. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"tag_filter": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Tag (or filter) to track; defaults to `latest`.",
			},
			"registry_connection_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "ID of the `bridgeport_registry_connection` the image is pulled from.",
			},
			"auto_update": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether BridgePort auto-updates the tracked tag.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Opaque server-assigned identifier for the image.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Opaque identifier of the environment the image belongs to.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"current_tag": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The currently-resolved tag (runtime, reference only).",
			},
			"latest_tag": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The latest available tag, if known (runtime, reference only).",
			},
			"update_available": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether a newer tag is available (runtime, reference only).",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp of when the image was created.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp of when the image was last updated.",
			},
		},
	}
}

func (r *containerImageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *containerImageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan containerImageResourceModel
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

	createReq := bpclient.CreateContainerImageRequest{
		Name:                 plan.Name.ValueString(),
		ImageName:            plan.ImageName.ValueString(),
		RegistryConnectionID: plan.RegistryConnectionID.ValueStringPointer(),
	}
	if !plan.TagFilter.IsNull() && !plan.TagFilter.IsUnknown() {
		createReq.TagFilter = plan.TagFilter.ValueStringPointer()
	}

	img, err := r.client.CreateContainerImage(env.ID, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create BridgePort container image",
			fmt.Sprintf("Could not create image %q in environment %q: %s", plan.ImageName.ValueString(), plan.Environment.ValueString(), err.Error()),
		)
		return
	}

	// auto_update is not settable on create; apply it with a follow-up update.
	if !plan.AutoUpdate.IsNull() && !plan.AutoUpdate.IsUnknown() && plan.AutoUpdate.ValueBool() != img.AutoUpdate {
		img, err = r.client.UpdateContainerImage(img.ID, bpclient.UpdateContainerImageRequest{
			AutoUpdate: plan.AutoUpdate.ValueBoolPointer(),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to set auto_update on BridgePort container image",
				err.Error(),
			)
			return
		}
	}

	r.applyComputed(&plan, env.ID, img)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *containerImageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state containerImageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if id == "" {
		// Import: resolve the ID from environment + image_name.
		env, err := r.client.GetEnvironmentByName(state.Environment.ValueString())
		if err != nil {
			if isNotFound(err) {
				resp.State.RemoveResource(ctx)
				return
			}
			resp.Diagnostics.AddError("Unable to resolve BridgePort environment", err.Error())
			return
		}
		imgs, err := r.client.ListContainerImages(env.ID)
		if err != nil {
			resp.Diagnostics.AddError("Unable to list BridgePort container images", err.Error())
			return
		}
		for i := range imgs {
			if imgs[i].ImageName == state.ImageName.ValueString() {
				id = imgs[i].ID
				break
			}
		}
		if id == "" {
			resp.State.RemoveResource(ctx)
			return
		}
	}

	img, err := r.client.GetContainerImage(id)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read BridgePort container image", err.Error())
		return
	}

	state.Name = types.StringValue(img.Name)
	state.ImageName = types.StringValue(img.ImageName)
	state.TagFilter = types.StringValue(img.TagFilter)
	state.RegistryConnectionID = stringOrNull(img.RegistryConnectionID)
	r.applyComputed(&state, img.EnvironmentID, img)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *containerImageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state containerImageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := bpclient.UpdateContainerImageRequest{
		Name:                 plan.Name.ValueStringPointer(),
		TagFilter:            plan.TagFilter.ValueStringPointer(),
		RegistryConnectionID: plan.RegistryConnectionID.ValueStringPointer(),
	}
	if !plan.AutoUpdate.IsNull() && !plan.AutoUpdate.IsUnknown() {
		updateReq.AutoUpdate = plan.AutoUpdate.ValueBoolPointer()
	}

	img, err := r.client.UpdateContainerImage(state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update BridgePort container image",
			fmt.Sprintf("Could not update image %q: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	r.applyComputed(&plan, state.EnvironmentID.ValueString(), img)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *containerImageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state containerImageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteContainerImage(state.ID.ValueString()); err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to delete BridgePort container image",
			fmt.Sprintf("Could not delete image %q: %s", state.ID.ValueString(), err.Error()),
		)
	}
}

// ImportState accepts the natural key "environment/image_name".
func (r *containerImageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"environment/image_name\", got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("image_name"), parts[1])...)
}

func (r *containerImageResource) applyComputed(m *containerImageResourceModel, envID string, img *bpclient.ContainerImage) {
	m.ID = types.StringValue(img.ID)
	m.EnvironmentID = types.StringValue(envID)
	m.TagFilter = types.StringValue(img.TagFilter)
	m.AutoUpdate = types.BoolValue(img.AutoUpdate)
	m.CurrentTag = types.StringValue(img.CurrentTag)
	m.LatestTag = stringOrNull(img.LatestTag)
	m.UpdateAvailable = types.BoolValue(img.UpdateAvailable)
	m.CreatedAt = types.StringValue(img.CreatedAt)
	m.UpdatedAt = types.StringValue(img.UpdatedAt)
}
