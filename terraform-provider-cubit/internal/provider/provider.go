package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure CubitProvider satisfies the provider.Provider interface.
var _ provider.Provider = &CubitProvider{}

// CubitProvider defines the provider implementation.
type CubitProvider struct {
	version string
}

// CubitProviderModel describes the provider data model.
type CubitProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Token    types.String `tfsdk:"token"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CubitProvider{
			version: version,
		}
	}
}

func (p *CubitProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cubit"
	resp.Version = p.version
}

func (p *CubitProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with Cubit Control Plane to manage bare-metal edge nodes, Cloudflare Worker applications, and domains.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Description: "The URL of the Cubit Control Plane API. Defaults to CUBIT_ENDPOINT or http://localhost:8000.",
				Optional:    true,
			},
			"token": schema.StringAttribute{
				Description: "API token for authentication. Can be provided via CUBIT_TOKEN environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *CubitProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data CubitProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := os.Getenv("CUBIT_ENDPOINT")
	if !data.Endpoint.IsNull() && data.Endpoint.ValueString() != "" {
		endpoint = data.Endpoint.ValueString()
	}
	if endpoint == "" {
		endpoint = "http://localhost:8000"
	}

	token := os.Getenv("CUBIT_TOKEN")
	if !data.Token.IsNull() && data.Token.ValueString() != "" {
		token = data.Token.ValueString()
	}

	client := NewCubitClient(endpoint, token)
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *CubitProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewNodeResource,
		NewApplicationResource,
		NewDomainResource,
	}
}

func (p *CubitProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
