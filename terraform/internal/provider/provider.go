/*
provider.go — Dipole Terraform Provider.

Implements provider configuration: client_id (native OIDC auth for CI/CD),
tenant_id + token (direct auth for dev), region, resource_set_id (default
Resource Set), and default_tags. All fields Optional with env var fallback.
Auth requires either client_id OR tenant_id+token.
*/
package provider

import (
	"context"
	"os"
	"strings"

	"github.com/dipole-labs/terraform-provider-dipole/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DipoleProvider struct {
	version string
}

type DipoleProviderModel struct {
	ClientID      types.String `tfsdk:"client_id"`
	TenantID      types.String `tfsdk:"tenant_id"`
	Token         types.String `tfsdk:"token"`
	Region        types.String `tfsdk:"region"`
	ResourceSetID types.String `tfsdk:"resource_set_id"`
	DefaultTags   types.Map    `tfsdk:"default_tags"`
}

type DipoleProviderData struct {
	Client        *client.Client
	ClientID      string
	TenantID      string
	Token         string
	BaseURL       string
	Region        string
	ResourceSetID string
	DefaultTags   map[string]string
}

func New() provider.Provider {
	return &DipoleProvider{version: "0.1.0"}
}

func (p *DipoleProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "dipole"
	resp.Version = p.version
}

func (p *DipoleProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Dipole workload IAM resources as infrastructure-as-code.",
		Attributes: map[string]schema.Attribute{
			"client_id": schema.StringAttribute{
				Optional:    true,
				Description: "Dipole Trust Provider Client ID for native OIDC authentication (recommended for CI/CD). Can also be set via DIPOLE_CLIENT_ID env var.",
			},
			"tenant_id": schema.StringAttribute{
				Optional:    true,
				Description: "Dipole Tenant ID (subdomain). Can also be set via DIPOLE_TENANT_ID env var.",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "API Token for authentication. Can also be set via DIPOLE_API_TOKEN env var.",
			},
			"region": schema.StringAttribute{
				Optional:    true,
				Description: "Region (default: ap-south-1). Can also be set via DIPOLE_REGION env var.",
			},
			"resource_set_id": schema.StringAttribute{
				Optional:    true,
				Description: "Default Resource Set ID for all resources managed by this provider. Can also be set via DIPOLE_RESOURCE_SET_ID env var.",
			},
			"default_tags": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Default tags to apply to all resources created by this provider.",
			},
		},
	}
}

func (p *DipoleProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config DipoleProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientID := stringValueOrEnv(config.ClientID, "DIPOLE_CLIENT_ID")
	tenantID := stringValueOrEnv(config.TenantID, "DIPOLE_TENANT_ID")
	token := stringValueOrEnv(config.Token, "DIPOLE_API_TOKEN")

	// Auth: either client_id (native OIDC) or tenant_id+token (direct)
	if clientID == "" && (tenantID == "" || token == "") {
		resp.Diagnostics.AddError(
			"Missing Authentication Configuration",
			"Either client_id (for native OIDC authentication) or both tenant_id and token "+
				"must be set in the provider configuration or via environment variables "+
				"(DIPOLE_CLIENT_ID, DIPOLE_TENANT_ID, DIPOLE_API_TOKEN).",
		)
		return
	}

	region := stringValueOrEnv(config.Region, "DIPOLE_REGION")
	if region == "" {
		region = "ap-south-1"
	}

	resourceSetID := stringValueOrEnv(config.ResourceSetID, "DIPOLE_RESOURCE_SET_ID")

	// Derive base URL from tenant_id, or parse it from client_id
	var baseURL string
	if tenantID != "" {
		baseURL = "https://" + tenantID + ".dipole.io"
	} else if clientID != "" {
		// client_id format: dipole:<region>:<tenantSubdomain>:identity:<type>:<uuid>
		parts := strings.Split(clientID, ":")
		if len(parts) >= 3 {
			tenantID = parts[2]
			baseURL = "https://" + tenantID + ".dipole.io"
		}
	}

	// Parse default tags
	var defaultTags map[string]string
	if !config.DefaultTags.IsNull() && !config.DefaultTags.IsUnknown() {
		defaultTags = make(map[string]string, len(config.DefaultTags.Elements()))
		diags := config.DefaultTags.ElementsAs(ctx, &defaultTags, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	providerData := &DipoleProviderData{
		Client:        client.New(baseURL, token, resourceSetID),
		ClientID:      clientID,
		TenantID:      tenantID,
		Token:         token,
		BaseURL:       baseURL,
		Region:        region,
		ResourceSetID: resourceSetID,
		DefaultTags:   defaultTags,
	}

	resp.DataSourceData = providerData
	resp.ResourceData = providerData
}

func (p *DipoleProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewClientWorkloadResource,
		NewServerWorkloadResource,
		NewTrustProviderResource,
		NewCredentialProviderResource,
		NewAccessPolicyResource,
		NewAccessConditionResource,
		NewResourceSetResource,
		NewRoleResource,
	}
}

func (p *DipoleProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewClientWorkloadDataSource,
		NewServerWorkloadDataSource,
		NewTrustProviderDataSource,
		NewAccessPolicyDataSource,
	}
}

func stringValueOrEnv(val types.String, envKey string) string {
	if !val.IsNull() && !val.IsUnknown() && val.ValueString() != "" {
		return val.ValueString()
	}
	return os.Getenv(envKey)
}
