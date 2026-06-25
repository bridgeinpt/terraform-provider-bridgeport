// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"

	bpclient "github.com/bridgeinpt/bridgeport/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &serviceDataSource{}
	_ datasource.DataSourceWithConfigure = &serviceDataSource{}
)

// NewServiceDataSource is a datasource.DataSource factory.
func NewServiceDataSource() datasource.DataSource {
	return &serviceDataSource{}
}

type serviceDataSource struct {
	client *bpclient.Client
}

// serviceModel holds the attributes of a BridgePort service. It is reused as
// the element type of the bridgeport_services list. Per the provider's
// configuration-only tenet, the runtime fields (status, container_status,
// health_status) are surfaced read-only for reference, never as inputs.
type serviceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	ImageTag         types.String `tfsdk:"image_tag"`
	EnvironmentID    types.String `tfsdk:"environment_id"`
	ContainerImageID types.String `tfsdk:"container_image_id"`
	ServiceTypeID    types.String `tfsdk:"service_type_id"`
	ServerID         types.String `tfsdk:"server_id"`
	ContainerName    types.String `tfsdk:"container_name"`
	Status           types.String `tfsdk:"status"`
	ContainerStatus  types.String `tfsdk:"container_status"`
	HealthStatus     types.String `tfsdk:"health_status"`
	CreatedAt        types.String `tfsdk:"created_at"`
}

// serviceDataSourceModel is the singular data source's state: the natural-key
// inputs (environment + server + name) plus the computed service attributes.
type serviceDataSourceModel struct {
	Environment      types.String `tfsdk:"environment"`
	Server           types.String `tfsdk:"server"`
	Name             types.String `tfsdk:"name"`
	ID               types.String `tfsdk:"id"`
	ImageTag         types.String `tfsdk:"image_tag"`
	EnvironmentID    types.String `tfsdk:"environment_id"`
	ContainerImageID types.String `tfsdk:"container_image_id"`
	ServiceTypeID    types.String `tfsdk:"service_type_id"`
	ServerID         types.String `tfsdk:"server_id"`
	ContainerName    types.String `tfsdk:"container_name"`
	Status           types.String `tfsdk:"status"`
	ContainerStatus  types.String `tfsdk:"container_status"`
	HealthStatus     types.String `tfsdk:"health_status"`
	CreatedAt        types.String `tfsdk:"created_at"`
}

func (d *serviceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (d *serviceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a single BridgePort service by its natural key (`environment` + `server` + `name`).",
		Attributes: map[string]schema.Attribute{
			"environment": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the environment that hosts the service's server, e.g. `production`.",
			},
			"server": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the server the service runs on.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The unique name of the service on the server (its natural key).",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Opaque server-assigned identifier for the service.",
			},
			"image_tag": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The container image tag the service is configured to run.",
			},
			"environment_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Opaque identifier of the environment the service belongs to.",
			},
			"container_image_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Opaque identifier of the container image the service deploys.",
			},
			"service_type_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Opaque identifier of the service type, or null if the service has none.",
			},
			"server_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Opaque identifier of the server the service runs on.",
			},
			"container_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Name of the running container (runtime, reference only).",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current runtime status of the service as reported by the platform (reference only).",
			},
			"container_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current container status (runtime, reference only).",
			},
			"health_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current health status (runtime, reference only).",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp of when the service was created.",
			},
		},
	}
}

func (d *serviceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*bpclient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = client
}

func (d *serviceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serviceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The SDK looks services up by server ID, so resolve the server first.
	server, err := d.client.GetServerByEnvAndName(data.Environment.ValueString(), data.Server.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read BridgePort server",
			fmt.Sprintf("Could not look up server %q in environment %q: %s", data.Server.ValueString(), data.Environment.ValueString(), err.Error()),
		)
		return
	}

	service, err := d.client.GetServiceByName(server.ID, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read BridgePort service",
			fmt.Sprintf("Could not look up service %q on server %q: %s", data.Name.ValueString(), data.Server.ValueString(), err.Error()),
		)
		return
	}

	m := serviceToModel(*service)
	// Preserve the lookup keys (environment + server names); the SDK Service
	// carries IDs, which m populates.
	data.Name = m.Name
	data.ID = m.ID
	data.ImageTag = m.ImageTag
	data.EnvironmentID = m.EnvironmentID
	data.ContainerImageID = m.ContainerImageID
	data.ServiceTypeID = m.ServiceTypeID
	data.ServerID = m.ServerID
	data.ContainerName = m.ContainerName
	data.Status = m.Status
	data.ContainerStatus = m.ContainerStatus
	data.HealthStatus = m.HealthStatus
	data.CreatedAt = m.CreatedAt

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// serviceToModel maps an SDK Service to the Terraform model.
func serviceToModel(s bpclient.Service) serviceModel {
	return serviceModel{
		ID:               types.StringValue(s.ID),
		Name:             types.StringValue(s.Name),
		ImageTag:         types.StringValue(s.ImageTag),
		EnvironmentID:    types.StringValue(s.EnvironmentID),
		ContainerImageID: types.StringValue(s.ContainerImageID),
		ServiceTypeID:    types.StringPointerValue(s.ServiceTypeID),
		ServerID:         types.StringValue(s.ServerID),
		ContainerName:    types.StringValue(s.ContainerName),
		Status:           types.StringValue(s.Status),
		ContainerStatus:  types.StringValue(s.ContainerStatus),
		HealthStatus:     types.StringValue(s.HealthStatus),
		CreatedAt:        types.StringValue(s.CreatedAt),
	}
}
