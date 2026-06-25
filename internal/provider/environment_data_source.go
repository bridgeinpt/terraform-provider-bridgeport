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
	_ datasource.DataSource              = &environmentDataSource{}
	_ datasource.DataSourceWithConfigure = &environmentDataSource{}
)

// NewEnvironmentDataSource is a datasource.DataSource factory.
func NewEnvironmentDataSource() datasource.DataSource {
	return &environmentDataSource{}
}

type environmentDataSource struct {
	client *bpclient.Client
}

// environmentModel is the Terraform representation of a BridgePort environment.
// It is shared by the singular and plural (list) data sources.
type environmentModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	DisplayName   types.String `tfsdk:"display_name"`
	SSHConfigured types.Bool   `tfsdk:"ssh_configured"`
	CreatedAt     types.String `tfsdk:"created_at"`
}

func (d *environmentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (d *environmentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a single BridgePort environment by its natural key (`name`).",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The unique slug of the environment (its natural key), e.g. `production`.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Opaque server-assigned identifier for the environment.",
			},
			"display_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Human-friendly name shown in the UI.",
			},
			"ssh_configured": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether an SSH key is configured for this environment.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp of when the environment was created.",
			},
		},
	}
}

func (d *environmentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *environmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data environmentModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	env, err := d.client.GetEnvironmentByName(data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read BridgePort environment",
			fmt.Sprintf("Could not look up environment %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	model := environmentToModel(*env)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

// environmentToModel maps an SDK Environment to the Terraform model.
func environmentToModel(env bpclient.Environment) environmentModel {
	return environmentModel{
		ID:            types.StringValue(env.ID),
		Name:          types.StringValue(env.Name),
		DisplayName:   types.StringValue(env.DisplayName),
		SSHConfigured: types.BoolValue(env.SSHConfigured),
		CreatedAt:     types.StringValue(env.CreatedAt),
	}
}
