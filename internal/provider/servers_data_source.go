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
	_ datasource.DataSource              = &serversDataSource{}
	_ datasource.DataSourceWithConfigure = &serversDataSource{}
)

// NewServersDataSource is a datasource.DataSource factory.
func NewServersDataSource() datasource.DataSource {
	return &serversDataSource{}
}

type serversDataSource struct {
	client *bpclient.Client
}

// serversModel is the top-level state for the list data source. The optional
// environment filter is echoed back so it remains in state.
type serversModel struct {
	Environment types.String  `tfsdk:"environment"`
	Servers     []serverModel `tfsdk:"servers"`
}

func (d *serversDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_servers"
}

func (d *serversDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List servers visible to the configured token, optionally filtered to a single environment.",
		Attributes: map[string]schema.Attribute{
			"environment": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "If set, only return servers in this environment (by `name`). Omit to list servers across every environment the token can see.",
			},
			"servers": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The matching servers.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Opaque server-assigned identifier for the server.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The unique name of the server within its environment.",
						},
						"environment_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Opaque identifier of the environment the server belongs to.",
						},
						"private_ip": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The server's private IP address.",
						},
						"public_ip": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The server's public IP address, or null if it has none.",
						},
						"status": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Current runtime status of the server as reported by the platform.",
						},
						"created_at": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "RFC 3339 timestamp of when the server was created.",
						},
					},
				},
			},
		},
	}
}

func (d *serversDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *serversDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serversModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve the optional environment filter (by name) to its ID. An empty ID
	// makes the SDK aggregate servers across all environments.
	var envID string
	if !data.Environment.IsNull() && data.Environment.ValueString() != "" {
		env, err := d.client.GetEnvironmentByName(data.Environment.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to look up BridgePort environment",
				fmt.Sprintf("Could not resolve environment %q: %s", data.Environment.ValueString(), err.Error()),
			)
			return
		}
		envID = env.ID
	}

	servers, err := d.client.ListServers(envID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to list BridgePort servers",
			err.Error(),
		)
		return
	}

	data.Servers = make([]serverModel, 0, len(servers))
	for _, s := range servers {
		data.Servers = append(data.Servers, serverToModel(s))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
