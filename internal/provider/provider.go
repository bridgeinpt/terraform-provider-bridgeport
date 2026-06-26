// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"os"
	"sync"
	"time"

	bpclient "github.com/bridgeinpt/bridgeport/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure bridgeportProvider satisfies the provider.Provider interface.
var _ provider.Provider = &bridgeportProvider{}

// validatedTokens records (endpoint, token) pairs whose token this process has
// already validated, so Configure doesn't re-probe GET /api/auth/me every time.
var validatedTokens sync.Map

// bridgeportProvider is the provider implementation.
type bridgeportProvider struct {
	// version is set to the released provider version (or "dev"/"test") and is
	// surfaced to Terraform for diagnostics and user-agent reporting.
	version string
}

// bridgeportProviderModel maps the provider-block configuration to Go values.
type bridgeportProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Token    types.String `tfsdk:"token"`
}

// New returns a function that constructs the provider, as expected by
// providerserver.Serve. version is injected at build time.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &bridgeportProvider{version: version}
	}
}

func (p *bridgeportProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "bridgeport"
	resp.Version = p.version
}

func (p *bridgeportProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The BridgePort provider manages configuration on a [BridgePort](https://github.com/bridgeinpt/bridgeport) " +
			"instance — environments, servers, and the resources layered on top of them — declaratively via its HTTP API. " +
			"Runtime operations (deploys, restarts, rollbacks) remain imperative; the provider only manages desired configuration.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Base URL of the BridgePort instance, e.g. `https://bridgeport.example.com` (no trailing slash). " +
					"May also be set with the `BRIDGEPORT_ENDPOINT` environment variable.",
			},
			"token": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				MarkdownDescription: "API bearer token used to authenticate. A scoped **service-account** token is recommended for automation " +
					"(see the BridgePort docs on Service Accounts). May also be set with the `BRIDGEPORT_TOKEN` environment variable. " +
					"Prefer the environment variable so the token never lands in Terraform state or configuration.",
			},
		},
	}
}

func (p *bridgeportProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config bridgeportProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Values must be known at configure time — a value derived from another
	// resource isn't available yet and can't be used to build the client.
	if config.Endpoint.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Unknown BridgePort endpoint",
			"The provider cannot create the API client because the endpoint is unknown at configure time. "+
				"Set it to a static value or use the BRIDGEPORT_ENDPOINT environment variable.",
		)
	}
	if config.Token.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Unknown BridgePort token",
			"The provider cannot create the API client because the token is unknown at configure time. "+
				"Set it to a static value or use the BRIDGEPORT_TOKEN environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration takes precedence over environment variables.
	endpoint := os.Getenv("BRIDGEPORT_ENDPOINT")
	token := os.Getenv("BRIDGEPORT_TOKEN")
	if !config.Endpoint.IsNull() {
		endpoint = config.Endpoint.ValueString()
	}
	if !config.Token.IsNull() {
		token = config.Token.ValueString()
	}

	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Missing BridgePort endpoint",
			"Set the `endpoint` argument or the BRIDGEPORT_ENDPOINT environment variable.",
		)
	}
	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Missing BridgePort token",
			"Set the `token` argument or the BRIDGEPORT_TOKEN environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	client := bpclient.NewClient(endpoint, token)
	// Make read calls resilient to transient server errors (the API returns
	// retryable 503s under brief database contention; older instances surface a
	// 500). Retries idempotent GET/HEAD only. The client Timeout bounds the whole
	// call including retries, so widen it to fit the retry budget.
	client.HTTPClient.Timeout = 90 * time.Second
	client.HTTPClient.Transport = newRetryTransport(client.HTTPClient.Transport)

	// Fail fast with a clear diagnostic if the credentials are wrong, rather
	// than surfacing an opaque 401 on the first data-source read. This probe is
	// done once per (endpoint, token) per process: the server authenticates
	// every real request anyway, and the probe itself triggers a server-side
	// write (lastActiveAt) — re-running it on every Configure needlessly loads
	// the instance during acceptance (Terraform calls Configure once per run in
	// normal use, so this is a no-op there).
	validationKey := endpoint + "\x00" + token
	if _, done := validatedTokens.Load(validationKey); !done {
		if _, err := client.GetCurrentUser(); err != nil {
			resp.Diagnostics.AddError(
				"Unable to authenticate with BridgePort",
				"The provider could not validate the configured token against "+endpoint+". "+
					"Verify the endpoint is reachable and the token is valid.\n\nError: "+err.Error(),
			)
			return
		}
		validatedTokens.Store(validationKey, struct{}{})
	}

	// The client is shared with every data source and resource.
	resp.DataSourceData = client
	resp.ResourceData = client
}

// Resources returns the managed (CRUD) resources. They land in dependency order
// (see the README roadmap and platform epic bridgeinpt/bridgeport#197) as the
// Go SDK gains the corresponding write methods.
func (p *bridgeportProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewServerResource,
		NewVarResource,
	}
}

func (p *bridgeportProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewEnvironmentDataSource,
		NewEnvironmentsDataSource,
		NewServerDataSource,
		NewServersDataSource,
		NewServiceDataSource,
		NewServicesDataSource,
	}
}
