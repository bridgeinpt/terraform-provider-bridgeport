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

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &serverResource{}
	_ resource.ResourceWithConfigure   = &serverResource{}
	_ resource.ResourceWithImportState = &serverResource{}
)

// NewServerResource is a resource.Resource factory.
func NewServerResource() resource.Resource {
	return &serverResource{}
}

type serverResource struct {
	client *bpclient.Client
}

// serverResourceModel is the Terraform state for a managed server. Configuration
// fields are Required/Optional; server-assigned and runtime fields are Computed
// (per the configuration-only tenet, runtime is never an input).
type serverResourceModel struct {
	Environment   types.String `tfsdk:"environment"`
	Name          types.String `tfsdk:"name"`
	Hostname      types.String `tfsdk:"hostname"`
	PublicIP      types.String `tfsdk:"public_ip"`
	Tags          types.List   `tfsdk:"tags"`
	ID            types.String `tfsdk:"id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	PrivateIP     types.String `tfsdk:"private_ip"`
	Status        types.String `tfsdk:"status"`
	CreatedAt     types.String `tfsdk:"created_at"`
}

func (r *serverResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (r *serverResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a BridgePort server within an environment. " +
			"This manages the server's *configuration* only; runtime operations (deploys, " +
			"restarts) and live status remain imperative and are surfaced read-only.",
		Attributes: map[string]schema.Attribute{
			"environment": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the environment the server belongs to. Changing this forces a new server.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The unique name of the server within its environment (its natural key).",
			},
			"hostname": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The hostname or IP address BridgePort uses to reach the server.",
			},
			"public_ip": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The server's public IP address, if any.",
			},
			"tags": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Free-form tags applied to the server.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Opaque server-assigned identifier for the server.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Opaque identifier of the environment the server belongs to.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"private_ip": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The server's private IP address, assigned by the platform.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current runtime status of the server as reported by the platform.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp of when the server was created.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *serverResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *serverResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serverResourceModel
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

	tags, diags := listToStrings(ctx, plan.Tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := bpclient.CreateServerRequest{
		Name:     plan.Name.ValueString(),
		Hostname: plan.Hostname.ValueString(),
		Tags:     tags,
	}
	if !plan.PublicIP.IsNull() {
		createReq.PublicIP = plan.PublicIP.ValueStringPointer()
	}

	server, err := r.client.CreateServer(env.ID, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create BridgePort server",
			fmt.Sprintf("Could not create server %q in environment %q: %s", plan.Name.ValueString(), plan.Environment.ValueString(), err.Error()),
		)
		return
	}

	// Computed fields come from the API; configuration fields are echoed from
	// the plan so state matches the submitted configuration exactly.
	plan.ID = types.StringValue(server.ID)
	plan.EnvironmentID = types.StringValue(server.EnvironmentID)
	plan.PrivateIP = types.StringValue(server.PrivateIP)
	plan.Status = types.StringValue(server.Status)
	plan.CreatedAt = types.StringValue(server.CreatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serverResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serverResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Look up by natural key (environment + name). The single-server detail
	// endpoint is read via the list path here because it carries the full
	// object; it also makes import (which only has the natural key) work
	// without a separate code path.
	server, err := r.client.GetServerByEnvAndName(state.Environment.ValueString(), state.Name.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read BridgePort server", err.Error())
		return
	}

	state.ID = types.StringValue(server.ID)
	state.Name = types.StringValue(server.Name)
	state.Hostname = types.StringValue(server.Hostname)
	state.PublicIP = stringOrNull(server.PublicIP)
	state.EnvironmentID = types.StringValue(server.EnvironmentID)
	state.PrivateIP = types.StringValue(server.PrivateIP)
	state.Status = types.StringValue(server.Status)
	state.CreatedAt = types.StringValue(server.CreatedAt)

	tags, diags := parseTags(ctx, server.Tags, state.Tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Tags = tags

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serverResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state serverResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags, diags := listToStrings(ctx, plan.Tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := bpclient.UpdateServerRequest{
		Name:     plan.Name.ValueStringPointer(),
		Hostname: plan.Hostname.ValueStringPointer(),
		Tags:     tags,
	}
	// public_ip: send the new value, or an empty string to clear a previously set one.
	switch {
	case !plan.PublicIP.IsNull():
		updateReq.PublicIP = plan.PublicIP.ValueStringPointer()
	case !state.PublicIP.IsNull():
		empty := ""
		updateReq.PublicIP = &empty
	}

	server, err := r.client.UpdateServer(state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update BridgePort server",
			fmt.Sprintf("Could not update server %q: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(server.ID)
	plan.EnvironmentID = types.StringValue(server.EnvironmentID)
	plan.PrivateIP = types.StringValue(server.PrivateIP)
	plan.Status = types.StringValue(server.Status)
	plan.CreatedAt = types.StringValue(server.CreatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serverResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serverResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteServer(state.ID.ValueString()); err != nil {
		if isNotFound(err) {
			return // already gone
		}
		resp.Diagnostics.AddError(
			"Unable to delete BridgePort server",
			fmt.Sprintf("Could not delete server %q: %s", state.ID.ValueString(), err.Error()),
		)
	}
}

// ImportState accepts the natural key "environment/name".
func (r *serverResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
