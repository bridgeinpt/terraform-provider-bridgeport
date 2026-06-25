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
	_ datasource.DataSource              = &serverDataSource{}
	_ datasource.DataSourceWithConfigure = &serverDataSource{}
)

// NewServerDataSource is a datasource.DataSource factory.
func NewServerDataSource() datasource.DataSource {
	return &serverDataSource{}
}

type serverDataSource struct {
	client *bpclient.Client
}

// serverModel holds the server-assigned attributes of a BridgePort server.
// It is reused as the element type of the bridgeport_servers list. The
// environment *name* is not carried on the SDK Server (only environment_id is),
// so it appears only as the lookup key on the singular data source.
type serverModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	PrivateIP     types.String `tfsdk:"private_ip"`
	PublicIP      types.String `tfsdk:"public_ip"`
	Status        types.String `tfsdk:"status"`
	CreatedAt     types.String `tfsdk:"created_at"`
}

// serverDataSourceModel is the singular data source's state: the natural-key
// inputs (environment + name) plus the computed server attributes.
type serverDataSourceModel struct {
	Environment   types.String `tfsdk:"environment"`
	Name          types.String `tfsdk:"name"`
	ID            types.String `tfsdk:"id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	PrivateIP     types.String `tfsdk:"private_ip"`
	PublicIP      types.String `tfsdk:"public_ip"`
	Status        types.String `tfsdk:"status"`
	CreatedAt     types.String `tfsdk:"created_at"`
}

func (d *serverDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (d *serverDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a single BridgePort server by its natural key (`environment` + `name`).",
		Attributes: map[string]schema.Attribute{
			"environment": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the environment the server belongs to, e.g. `production`.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The unique name of the server within its environment (its natural key).",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Opaque server-assigned identifier for the server.",
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
	}
}

func (d *serverDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return // provider has not been configured yet
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

func (d *serverDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serverDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := d.client.GetServerByEnvAndName(data.Environment.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read BridgePort server",
			fmt.Sprintf("Could not look up server %q in environment %q: %s", data.Name.ValueString(), data.Environment.ValueString(), err.Error()),
		)
		return
	}

	m := serverToModel(*server)
	// Preserve the environment name (the lookup key); the SDK Server carries
	// only environment_id, which m populates.
	data.Name = m.Name
	data.ID = m.ID
	data.EnvironmentID = m.EnvironmentID
	data.PrivateIP = m.PrivateIP
	data.PublicIP = m.PublicIP
	data.Status = m.Status
	data.CreatedAt = m.CreatedAt

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// serverToModel maps an SDK Server to the Terraform model.
func serverToModel(s bpclient.Server) serverModel {
	return serverModel{
		ID:            types.StringValue(s.ID),
		Name:          types.StringValue(s.Name),
		EnvironmentID: types.StringValue(s.EnvironmentID),
		PrivateIP:     types.StringValue(s.PrivateIP),
		PublicIP:      types.StringPointerValue(s.PublicIP),
		Status:        types.StringValue(s.Status),
		CreatedAt:     types.StringValue(s.CreatedAt),
	}
}
