/*
data_source_credential_provider.go — Data source for dipole_credential_provider.

Read-only lookup of an existing Credential Provider by ID or name.
*/
package provider

import (
	"context"
	"encoding/json"

	"github.com/dipolehq/terraform-provider-dipole/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type credentialProviderDataSource struct {
	client *client.Client
}

type credentialProviderDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IsActive    types.Bool   `tfsdk:"is_active"`
	Type        types.String `tfsdk:"type"`
}

func NewCredentialProviderDataSource() datasource.DataSource {
	return &credentialProviderDataSource{}
}

func (d *credentialProviderDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_credential_provider"
}

func (d *credentialProviderDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Dipole Credential Provider by ID or name.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Optional: true, Computed: true, Description: "UUID of the Credential Provider."},
			"name":        schema.StringAttribute{Optional: true, Computed: true, Description: "Name of the Credential Provider."},
			"description": schema.StringAttribute{Computed: true, Description: "Description."},
			"is_active":   schema.BoolAttribute{Computed: true, Description: "Active status."},
			"type":        schema.StringAttribute{Computed: true, Description: "Credential Provider type (e.g., api_key, oauth_client_credentials, aws_sts)."},
		},
	}
}

func (d *credentialProviderDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	providerData, ok := req.ProviderData.(*DipoleProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Data", "Expected *DipoleProviderData")
		return
	}
	d.client = providerData.Client
}

func (d *credentialProviderDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config credentialProviderDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var respBody json.RawMessage
	var err error

	if !config.ID.IsNull() && config.ID.ValueString() != "" {
		respBody, err = d.client.Read(ctx, credentialProviderResourceType, config.ID.ValueString())
	} else if !config.Name.IsNull() && config.Name.ValueString() != "" {
		respBody, err = d.client.List(ctx, credentialProviderResourceType, map[string]string{"search": config.Name.ValueString()})
		if err == nil {
			respBody, err = extractFirstItem(respBody, config.Name.ValueString(), "Credential Provider")
		}
	} else {
		resp.Diagnostics.AddError("Missing filter", "Either id or name must be specified.")
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Error reading Credential Provider", err.Error())
		return
	}

	var result apiCredentialProvider
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	state := credentialProviderDataSourceModel{
		ID:       types.StringValue(result.ID),
		Name:     types.StringValue(result.Name),
		IsActive: types.BoolValue(result.IsActive),
		Type:     types.StringValue(result.Type),
	}
	if result.Description != "" {
		state.Description = types.StringValue(result.Description)
	} else {
		state.Description = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
