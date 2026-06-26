// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	bpclient "github.com/bridgeinpt/bridgeport/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &serviceResource{}
	_ resource.ResourceWithConfigure   = &serviceResource{}
	_ resource.ResourceWithImportState = &serviceResource{}
)

// NewServiceResource is a resource.Resource factory.
func NewServiceResource() resource.Resource {
	return &serviceResource{}
}

type serviceResource struct {
	client *bpclient.Client
}

// serviceResourceModel is the Terraform state for a managed service template
// (environment-scoped). Per-server placement is a separate
// bridgeport_service_deployment resource. Only configuration is modeled; runtime
// (status/health/ports) belongs to deployments and stays imperative.
type serviceResourceModel struct {
	Environment      types.String `tfsdk:"environment"`
	Name             types.String `tfsdk:"name"`
	ContainerImageID types.String `tfsdk:"container_image_id"`
	ImageTag         types.String `tfsdk:"image_tag"`
	ComposeTemplate  types.String `tfsdk:"compose_template"`
	HealthCheckURL   types.String `tfsdk:"health_check_url"`
	BaseEnv          types.Map    `tfsdk:"base_env"`
	DeployStrategy   types.String `tfsdk:"deploy_strategy"`
	ServiceTypeID    types.String `tfsdk:"service_type_id"`
	HealthWaitMs     types.Int64  `tfsdk:"health_wait_ms"`
	HealthRetries    types.Int64  `tfsdk:"health_retries"`
	HealthIntervalMs types.Int64  `tfsdk:"health_interval_ms"`
	ID               types.String `tfsdk:"id"`
	EnvironmentID    types.String `tfsdk:"environment_id"`
	CreatedAt        types.String `tfsdk:"created_at"`
}

func (r *serviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *serviceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a service template in an environment. A service is the deployable definition; " +
			"placing it on servers is done with `bridgeport_service_deployment`. Runtime status is not managed here.",
		Attributes: map[string]schema.Attribute{
			"environment": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the environment the service belongs to. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The unique name of the service within its environment (its natural key).",
			},
			"container_image_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the `bridgeport_container_image` the service runs.",
			},
			"image_tag": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Image tag to run; defaults to `latest`.",
			},
			"compose_template": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional Docker Compose template for the service.",
			},
			"health_check_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional health-check URL.",
			},
			"base_env": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Base environment variables applied to the service (key/value).",
			},
			"deploy_strategy": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Deployment strategy: `sequential` or `parallel`.",
			},
			"service_type_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional service-type ID.",
			},
			"health_wait_ms": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Milliseconds to wait before health checks.",
			},
			"health_retries": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Number of health-check retries.",
			},
			"health_interval_ms": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Milliseconds between health-check retries.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Opaque server-assigned identifier for the service.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Opaque identifier of the environment the service belongs to.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp of when the service was created.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *serviceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *serviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceResourceModel
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

	baseEnv, diags := mapToStringMap(ctx, plan.BaseEnv)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := bpclient.CreateServiceRequest{
		Name:             plan.Name.ValueString(),
		ContainerImageID: plan.ContainerImageID.ValueString(),
		ComposeTemplate:  plan.ComposeTemplate.ValueStringPointer(),
		HealthCheckURL:   plan.HealthCheckURL.ValueStringPointer(),
		BaseEnv:          baseEnv,
	}
	// image_tag and deploy_strategy are Optional+Computed: when unset they are
	// unknown (not null), so only send them when the user provided a value —
	// otherwise the server defaults them (sending "" is rejected).
	if !plan.ImageTag.IsNull() && !plan.ImageTag.IsUnknown() {
		createReq.ImageTag = plan.ImageTag.ValueStringPointer()
	}
	if !plan.DeployStrategy.IsNull() && !plan.DeployStrategy.IsUnknown() {
		createReq.DeployStrategy = plan.DeployStrategy.ValueStringPointer()
	}

	svc, err := r.client.CreateService(env.ID, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create BridgePort service",
			fmt.Sprintf("Could not create service %q in environment %q: %s", plan.Name.ValueString(), plan.Environment.ValueString(), err.Error()),
		)
		return
	}

	// service_type_id and the health_* tunables are update-only; apply them with
	// a follow-up update when set.
	if r.needsPostCreateUpdate(&plan) {
		updateReq := bpclient.UpdateServiceRequest{
			ServiceTypeID: plan.ServiceTypeID.ValueStringPointer(),
		}
		if !plan.HealthWaitMs.IsNull() && !plan.HealthWaitMs.IsUnknown() {
			v := int(plan.HealthWaitMs.ValueInt64())
			updateReq.HealthWaitMs = &v
		}
		if !plan.HealthRetries.IsNull() && !plan.HealthRetries.IsUnknown() {
			v := int(plan.HealthRetries.ValueInt64())
			updateReq.HealthRetries = &v
		}
		if !plan.HealthIntervalMs.IsNull() && !plan.HealthIntervalMs.IsUnknown() {
			v := int(plan.HealthIntervalMs.ValueInt64())
			updateReq.HealthIntervalMs = &v
		}
		svc, err = r.client.UpdateService(svc.ID, updateReq)
		if err != nil {
			resp.Diagnostics.AddError("Unable to apply BridgePort service settings", err.Error())
			return
		}
	}

	r.applyComputed(ctx, &plan, svc, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.getService(state)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read BridgePort service", err.Error())
		return
	}
	if svc == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(svc.Name)
	state.ContainerImageID = types.StringValue(svc.ContainerImageID)
	state.ImageTag = types.StringValue(svc.ImageTag)
	state.ComposeTemplate = stringOrNull(svc.ComposeTemplate)
	state.HealthCheckURL = stringOrNull(svc.HealthCheckURL)
	state.ServiceTypeID = stringOrNull(svc.ServiceTypeID)
	r.applyComputed(ctx, &state, svc, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state serviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	baseEnv, diags := mapToStringMap(ctx, plan.BaseEnv)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := bpclient.UpdateServiceRequest{
		Name:             plan.Name.ValueStringPointer(),
		ContainerImageID: plan.ContainerImageID.ValueStringPointer(),
		ComposeTemplate:  plan.ComposeTemplate.ValueStringPointer(),
		HealthCheckURL:   plan.HealthCheckURL.ValueStringPointer(),
		BaseEnv:          baseEnv,
		ServiceTypeID:    plan.ServiceTypeID.ValueStringPointer(),
	}
	if !plan.ImageTag.IsNull() && !plan.ImageTag.IsUnknown() {
		updateReq.ImageTag = plan.ImageTag.ValueStringPointer()
	}
	if !plan.DeployStrategy.IsNull() && !plan.DeployStrategy.IsUnknown() {
		updateReq.DeployStrategy = plan.DeployStrategy.ValueStringPointer()
	}
	if !plan.HealthWaitMs.IsNull() && !plan.HealthWaitMs.IsUnknown() {
		v := int(plan.HealthWaitMs.ValueInt64())
		updateReq.HealthWaitMs = &v
	}
	if !plan.HealthRetries.IsNull() && !plan.HealthRetries.IsUnknown() {
		v := int(plan.HealthRetries.ValueInt64())
		updateReq.HealthRetries = &v
	}
	if !plan.HealthIntervalMs.IsNull() && !plan.HealthIntervalMs.IsUnknown() {
		v := int(plan.HealthIntervalMs.ValueInt64())
		updateReq.HealthIntervalMs = &v
	}

	svc, err := r.client.UpdateService(state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update BridgePort service",
			fmt.Sprintf("Could not update service %q: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	r.applyComputed(ctx, &plan, svc, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteService(state.ID.ValueString()); err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to delete BridgePort service",
			fmt.Sprintf("Could not delete service %q: %s", state.ID.ValueString(), err.Error()),
		)
	}
}

// ImportState accepts the natural key "environment/name".
func (r *serviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

func (r *serviceResource) needsPostCreateUpdate(m *serviceResourceModel) bool {
	known := func(s types.String) bool { return !s.IsNull() && !s.IsUnknown() }
	knownI := func(i types.Int64) bool { return !i.IsNull() && !i.IsUnknown() }
	return known(m.ServiceTypeID) || knownI(m.HealthWaitMs) || knownI(m.HealthRetries) || knownI(m.HealthIntervalMs)
}

// getService fetches a service by ID (normal refresh) or resolves it from the
// environment-scoped service list by name (import). The SDK has no env-scoped
// service getter (GetServiceByName is server-scoped and would miss a template
// with no deployments), so the import path lists via client.Get.
func (r *serviceResource) getService(state serviceResourceModel) (*bpclient.Service, error) {
	if id := state.ID.ValueString(); id != "" {
		return r.client.GetService(id)
	}

	env, err := r.client.GetEnvironmentByName(state.Environment.ValueString())
	if err != nil {
		return nil, err
	}
	var resp struct {
		Services []bpclient.Service `json:"services"`
	}
	if err := r.client.Get(fmt.Sprintf("/api/environments/%s/services", env.ID), &resp); err != nil {
		return nil, err
	}
	for i := range resp.Services {
		if resp.Services[i].Name == state.Name.ValueString() {
			return &resp.Services[i], nil
		}
	}
	return nil, nil
}

// applyComputed maps server-assigned/normalized fields onto the model, including
// base_env (returned as a JSON-encoded string).
func (r *serviceResource) applyComputed(ctx context.Context, m *serviceResourceModel, svc *bpclient.Service, diags *diag.Diagnostics) {
	m.ID = types.StringValue(svc.ID)
	m.EnvironmentID = types.StringValue(svc.EnvironmentID)
	m.ImageTag = types.StringValue(svc.ImageTag)
	m.DeployStrategy = types.StringValue(svc.DeployStrategy)
	m.HealthWaitMs = types.Int64Value(int64(svc.HealthWaitMs))
	m.HealthRetries = types.Int64Value(int64(svc.HealthRetries))
	m.HealthIntervalMs = types.Int64Value(int64(svc.HealthIntervalMs))
	m.CreatedAt = types.StringValue(svc.CreatedAt)

	baseEnv, d := parseJSONMap(ctx, svc.BaseEnv, m.BaseEnv)
	diags.Append(d...)
	m.BaseEnv = baseEnv
}
