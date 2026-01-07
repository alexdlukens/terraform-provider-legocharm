package provider

import (
	"context"
	"os"

	"terraform-provider-legocharm/internal/legocharmclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &legocharmProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &legocharmProvider{
			version: version,
		}
	}
}

// legocharmProvider is the provider implementation.
type legocharmProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// legocharmProviderModel maps provider schema data to a Go type.
type legocharmProviderModel struct {
	Address  types.String `tfsdk:"address"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

// Metadata returns the provider type name.
func (p *legocharmProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "legocharm"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *legocharmProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"address": schema.StringAttribute{
			Optional: true,
		},
		"username": schema.StringAttribute{
			Optional: true,
		},
		"password": schema.StringAttribute{
			Optional:  true,
			Sensitive: true,
		},
	},
	}
}

// Configure prepares a LegoCharm API client for data sources and resources.
func (p *legocharmProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config legocharmProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Address.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("address"),
			"Unknown LegoCharm API Address",
			"The provider cannot create the LegoCharm API client as there is an unknown configuration value for the LegoCharm API address. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the LEGOCHARM_ADDRESS environment variable.",
		)
	}

	if config.Username.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Unknown LegoCharm API Username",
			"The provider cannot create the LegoCharm API client as there is an unknown configuration value for the LegoCharm API username. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the LEGOCHARM_USERNAME environment variable.",
		)
	}

	if config.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Unknown LegoCharm API Password",
			"The provider cannot create the LegoCharm API client as there is an unknown configuration value for the LegoCharm API password. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the LEGOCHARM_PASSWORD environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	address := os.Getenv("LEGOCHARM_ADDRESS")
	username := os.Getenv("LEGOCHARM_USERNAME")
	password := os.Getenv("LEGOCHARM_PASSWORD")

	if !config.Address.IsNull() {
		address = config.Address.ValueString()
	}

	if !config.Username.IsNull() {
		username = config.Username.ValueString()
	}

	if !config.Password.IsNull() {
		password = config.Password.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if address == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("address"),
			"LegoCharm API Address Not Set",
			"The provider cannot create the LegoCharm API client as there is no configured address. "+
				"Set the address value in the provider configuration or use the LEGOCHARM_ADDRESS environment variable.",
		)
	}

	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"LegoCharm API Username Not Set",
			"The provider cannot create the LegoCharm API client as there is no configured username. "+
				"Set the username value in the provider configuration or use the LEGOCHARM_USERNAME environment variable.",
		)
	}

	if password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"LegoCharm API Password Not Set",
			"The provider cannot create the LegoCharm API client as there is no configured password. "+
				"Set the password value in the provider configuration or use the LEGOCHARM_PASSWORD environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Create a new LegoCharm client using the configuration values
	client, err := legocharmclient.NewClient(&address, &username, &password)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create LegoCharm API Client",
			"An unexpected error occurred when creating the LegoCharm API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"LegoCharm Client Error: "+err.Error(),
		)
		return
	}

	// Make the LegoCharm client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
}

// DataSources defines the data sources implemented in the provider.
func (p *legocharmProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

// Resources defines the resources implemented in the provider.
func (p *legocharmProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewUserResource,
	}
}
