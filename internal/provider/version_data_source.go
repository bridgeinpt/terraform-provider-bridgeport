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
	_ datasource.DataSource              = &versionDataSource{}
	_ datasource.DataSourceWithConfigure = &versionDataSource{}
)

// NewVersionDataSource is a datasource.DataSource factory.
func NewVersionDataSource() datasource.DataSource {
	return &versionDataSource{}
}

type versionDataSource struct {
	client *bpclient.Client
}

// versionDataSourceModel is the state of the singleton bridgeport_version data
// source: the instance's status and version triple from GET /health. It takes
// no inputs — the endpoint identifies the instance — so every attribute is
// Computed.
type versionDataSourceModel struct {
	Status              types.String `tfsdk:"status"`
	Version             types.String `tfsdk:"version"`
	BundledAgentVersion types.String `tfsdk:"bundled_agent_version"`
	CLIVersion          types.String `tfsdk:"cli_version"`
	Timestamp           types.String `tfsdk:"timestamp"`
}

func (d *versionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_version"
}

func (d *versionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reports the targeted BridgePort instance's status and version, read from its " +
			"unauthenticated `GET /health` endpoint (the canonical version source — there is deliberately no " +
			"`/api/version` route). Use it for **provider ↔ instance version negotiation**: surface the running " +
			"platform version in outputs, or assert a minimum BridgePort version with a `precondition` before a " +
			"module applies.",
		Attributes: map[string]schema.Attribute{
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Instance health status, e.g. `ok`.",
			},
			"version": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Running BridgePort application version, e.g. `1.4.2`.",
			},
			"bundled_agent_version": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Version of the agent bundled with this instance.",
			},
			"cli_version": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Version of the CLI bundled with this instance.",
			},
			"timestamp": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp of when the instance produced this health response (i.e. read time).",
			},
		},
	}
}

func (d *versionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *versionDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	health, err := d.client.GetHealth()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read BridgePort version",
			fmt.Sprintf("Could not read instance health from GET /health: %s", err.Error()),
		)
		return
	}

	model := healthToVersionModel(*health)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

// healthToVersionModel maps an SDK Health to the Terraform model.
func healthToVersionModel(h bpclient.Health) versionDataSourceModel {
	return versionDataSourceModel{
		Status:              types.StringValue(h.Status),
		Version:             types.StringValue(h.Version),
		BundledAgentVersion: types.StringValue(h.BundledAgentVersion),
		CLIVersion:          types.StringValue(h.CliVersion),
		Timestamp:           types.StringValue(h.Timestamp),
	}
}
