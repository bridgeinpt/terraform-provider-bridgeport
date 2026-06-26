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
	_ resource.Resource                = &serviceDeploymentResource{}
	_ resource.ResourceWithConfigure   = &serviceDeploymentResource{}
	_ resource.ResourceWithImportState = &serviceDeploymentResource{}
)

// NewServiceDeploymentResource is a resource.Resource factory.
func NewServiceDeploymentResource() resource.Resource {
	return &serviceDeploymentResource{}
}

type serviceDeploymentResource struct {
	client *bpclient.Client
}

// serviceDeploymentResourceModel is the Terraform state for a service deployment:
// the placement of a service template onto a specific server. Only configuration
// is modeled; the runtime fields (status/health/discovery/ports) are Computed.
type serviceDeploymentResourceModel struct {
	ServiceID       types.String `tfsdk:"service_id"`
	ServerID        types.String `tfsdk:"server_id"`
	ContainerName   types.String `tfsdk:"container_name"`
	ComposePath     types.String `tfsdk:"compose_path"`
	EnvOverrides    types.Map    `tfsdk:"env_overrides"`
	ID              types.String `tfsdk:"id"`
	Status          types.String `tfsdk:"status"`
	ContainerStatus types.String `tfsdk:"container_status"`
	HealthStatus    types.String `tfsdk:"health_status"`
	DiscoveryStatus types.String `tfsdk:"discovery_status"`
	LastDeployedAt  types.String `tfsdk:"last_deployed_at"`
}

func (r *serviceDeploymentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_deployment"
}

func (r *serviceDeploymentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a service deployment: the placement of a `bridgeport_service` onto a " +
			"`bridgeport_server`. Manages configuration only; runtime status/health are surfaced read-only.",
		Attributes: map[string]schema.Attribute{
			"service_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the `bridgeport_service` being deployed. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"server_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the `bridgeport_server` to deploy onto. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"container_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the container for this deployment.",
			},
			"compose_path": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional path to a compose file for this deployment.",
			},
			"env_overrides": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Per-deployment environment variable overrides (key/value).",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Opaque server-assigned identifier for the deployment.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current deployment status (runtime, reference only).",
			},
			"container_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current container status (runtime, reference only).",
			},
			"health_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current health status (runtime, reference only).",
			},
			"discovery_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current discovery status (runtime, reference only).",
			},
			"last_deployed_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp of the last deploy, if any (runtime, reference only).",
			},
		},
	}
}

func (r *serviceDeploymentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *serviceDeploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceDeploymentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	envOverrides, diags := mapToStringMap(ctx, plan.EnvOverrides)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dep, err := r.client.CreateDeployment(plan.ServiceID.ValueString(), bpclient.CreateDeploymentRequest{
		ServerID:      plan.ServerID.ValueString(),
		ContainerName: plan.ContainerName.ValueString(),
		ComposePath:   plan.ComposePath.ValueStringPointer(),
		EnvOverrides:  envOverrides,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create BridgePort service deployment",
			fmt.Sprintf("Could not deploy service %q to server %q: %s", plan.ServiceID.ValueString(), plan.ServerID.ValueString(), err.Error()),
		)
		return
	}

	r.applyComputed(&plan, dep)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serviceDeploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceDeploymentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dep, err := r.findDeployment(state)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read BridgePort service deployment", err.Error())
		return
	}
	if dep == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ServerID = types.StringValue(dep.ServerID)
	state.ContainerName = types.StringValue(dep.ContainerName)
	state.ComposePath = stringOrNull(dep.ComposePath)
	envOverrides, diags := parseJSONMap(ctx, dep.EnvOverrides, state.EnvOverrides)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.EnvOverrides = envOverrides
	r.applyComputed(&state, dep)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serviceDeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state serviceDeploymentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	envOverrides, diags := mapToStringMap(ctx, plan.EnvOverrides)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dep, err := r.client.UpdateDeployment(state.ServiceID.ValueString(), state.ID.ValueString(), bpclient.UpdateDeploymentRequest{
		ContainerName: plan.ContainerName.ValueStringPointer(),
		ComposePath:   plan.ComposePath.ValueStringPointer(),
		EnvOverrides:  envOverrides,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update BridgePort service deployment",
			fmt.Sprintf("Could not update deployment %q: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	r.applyComputed(&plan, dep)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serviceDeploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceDeploymentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteDeployment(state.ServiceID.ValueString(), state.ID.ValueString()); err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to delete BridgePort service deployment",
			fmt.Sprintf("Could not delete deployment %q: %s", state.ID.ValueString(), err.Error()),
		)
	}
}

// ImportState accepts "service_id/server_id".
func (r *serviceDeploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"service_id/server_id\", got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), parts[1])...)
}

// findDeployment loads the parent service (whose detail includes its
// deployments, since there is no per-deployment getter) and returns the matching
// deployment by ID, or by server ID after import.
func (r *serviceDeploymentResource) findDeployment(state serviceDeploymentResourceModel) (*bpclient.ServiceDeployment, error) {
	svc, err := r.client.GetService(state.ServiceID.ValueString())
	if err != nil {
		return nil, err
	}
	id := state.ID.ValueString()
	for i := range svc.ServiceDeployments {
		d := &svc.ServiceDeployments[i]
		if (id != "" && d.ID == id) || (id == "" && d.ServerID == state.ServerID.ValueString()) {
			return d, nil
		}
	}
	return nil, nil
}

func (r *serviceDeploymentResource) applyComputed(m *serviceDeploymentResourceModel, dep *bpclient.ServiceDeployment) {
	m.ID = types.StringValue(dep.ID)
	m.ServiceID = types.StringValue(dep.ServiceID)
	m.Status = types.StringValue(dep.Status)
	m.ContainerStatus = types.StringValue(dep.ContainerStatus)
	m.HealthStatus = types.StringValue(dep.HealthStatus)
	m.DiscoveryStatus = types.StringValue(dep.DiscoveryStatus)
	m.LastDeployedAt = stringOrNull(dep.LastDeployedAt)
}
