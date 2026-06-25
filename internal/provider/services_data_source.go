// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"

	bpclient "github.com/bridgeinpt/bridgeport/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &servicesDataSource{}
	_ datasource.DataSourceWithConfigure = &servicesDataSource{}
)

// NewServicesDataSource is a datasource.DataSource factory.
func NewServicesDataSource() datasource.DataSource {
	return &servicesDataSource{}
}

type servicesDataSource struct {
	client *bpclient.Client
}

// servicesModel is the top-level state for the list data source. The optional
// environment / server filters are echoed back so they remain in state.
type servicesModel struct {
	Environment types.String   `tfsdk:"environment"`
	Server      types.String   `tfsdk:"server"`
	Services    []serviceModel `tfsdk:"services"`
}

func (d *servicesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_services"
}

func (d *servicesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List services visible to the configured token, optionally narrowed to an environment or a single server.",
		Attributes: map[string]schema.Attribute{
			"environment": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "If set, only return services in this environment (by `name`). Required when `server` is set, so the server can be resolved by name.",
			},
			"server": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "If set, only return services on this server (by `name`). Requires `environment` to also be set. Omit to list every service in the environment, or across all environments when `environment` is also omitted.",
			},
			"services": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The matching services.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Opaque server-assigned identifier for the service.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The unique name of the service on its server.",
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
							MarkdownDescription: "Current runtime status of the service (reference only).",
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
				},
			},
		},
	}
}

func (d *servicesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *servicesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data servicesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasEnv := !data.Environment.IsNull() && data.Environment.ValueString() != ""
	hasServer := !data.Server.IsNull() && data.Server.ValueString() != ""

	if hasServer && !hasEnv {
		resp.Diagnostics.AddAttributeError(
			path.Root("environment"),
			"Missing environment",
			"`environment` is required when `server` is set, so the server can be resolved by name.",
		)
		return
	}

	var services []bpclient.Service
	switch {
	case hasServer:
		// A specific server within an environment.
		server, err := d.client.GetServerByEnvAndName(data.Environment.ValueString(), data.Server.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to look up BridgePort server",
				fmt.Sprintf("Could not resolve server %q in environment %q: %s", data.Server.ValueString(), data.Environment.ValueString(), err.Error()),
			)
			return
		}
		services, err = d.client.ListServices(server.ID)
		if err != nil {
			resp.Diagnostics.AddError("Unable to list BridgePort services", err.Error())
			return
		}
	case hasEnv:
		// Every server in the environment. The SDK lists services by server,
		// so aggregate across the environment's servers.
		env, err := d.client.GetEnvironmentByName(data.Environment.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to look up BridgePort environment",
				fmt.Sprintf("Could not resolve environment %q: %s", data.Environment.ValueString(), err.Error()),
			)
			return
		}
		servers, err := d.client.ListServers(env.ID)
		if err != nil {
			resp.Diagnostics.AddError("Unable to list BridgePort servers", err.Error())
			return
		}
		for _, s := range servers {
			svcs, err := d.client.ListServices(s.ID)
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to list BridgePort services",
					fmt.Sprintf("Could not list services for server %q: %s", s.Name, err.Error()),
				)
				return
			}
			services = append(services, svcs...)
		}
	default:
		// Every service the token can see.
		var err error
		services, err = d.client.ListServices("")
		if err != nil {
			resp.Diagnostics.AddError("Unable to list BridgePort services", err.Error())
			return
		}
	}

	data.Services = make([]serviceModel, 0, len(services))
	for _, s := range services {
		data.Services = append(data.Services, serviceToModel(s))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
