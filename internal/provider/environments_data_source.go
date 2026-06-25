// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"

	bpclient "github.com/bridgeinpt/bridgeport/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &environmentsDataSource{}
	_ datasource.DataSourceWithConfigure = &environmentsDataSource{}
)

// NewEnvironmentsDataSource is a datasource.DataSource factory.
func NewEnvironmentsDataSource() datasource.DataSource {
	return &environmentsDataSource{}
}

type environmentsDataSource struct {
	client *bpclient.Client
}

// environmentsModel is the top-level state for the list data source.
type environmentsModel struct {
	Environments []environmentModel `tfsdk:"environments"`
}

func (d *environmentsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environments"
}

func (d *environmentsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List all environments visible to the configured token.",
		Attributes: map[string]schema.Attribute{
			"environments": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "All environments the token can see.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Opaque server-assigned identifier for the environment.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The unique slug of the environment (its natural key).",
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
				},
			},
		},
	}
}

func (d *environmentsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *environmentsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	envs, err := d.client.ListEnvironments()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to list BridgePort environments",
			err.Error(),
		)
		return
	}

	state := environmentsModel{Environments: make([]environmentModel, 0, len(envs))}
	for _, env := range envs {
		state.Environments = append(state.Environments, environmentToModel(env))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
