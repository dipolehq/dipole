/*
data_source_trust_provider.go — Data source for dipole_trust_provider.

Read-only lookup of an existing Trust Provider by ID or name.
*/
package provider

import (
	"context"
	"encoding/json"

	"github.com/dipole-labs/terraform-provider-dipole/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type trustProviderDataSource struct {
	client *client.Client
}

type trustProviderDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	IsActive        types.Bool   `tfsdk:"is_active"`
	Type            types.String `tfsdk:"type"`
	EdgeSDKClientID types.String `tfsdk:"edge_sdk_client_id"`
	EdgeSDKAudience types.String `tfsdk:"edge_sdk_audience"`
}

func NewTrustProviderDataSource() datasource.DataSource {
	return &trustProviderDataSource{}
}

func (d *trustProviderDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_trust_provider"
}

func (d *trustProviderDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Dipole Trust Provider by ID or name.",
		Attributes: map[string]schema.Attribute{
			"id":                 schema.StringAttribute{Optional: true, Computed: true, Description: "UUID of the Trust Provider."},
			"name":               schema.StringAttribute{Optional: true, Computed: true, Description: "Name of the Trust Provider."},
			"description":        schema.StringAttribute{Computed: true, Description: "Description."},
			"is_active":          schema.BoolAttribute{Computed: true, Description: "Active status."},
			"type":               schema.StringAttribute{Computed: true, Description: "Trust Provider type."},
			"edge_sdk_client_id": schema.StringAttribute{Computed: true, Description: "Edge SDK Client ID."},
			"edge_sdk_audience":  schema.StringAttribute{Computed: true, Description: "Edge SDK Audience."},
		},
	}
}

func (d *trustProviderDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *trustProviderDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config trustProviderDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var respBody json.RawMessage
	var err error

	if !config.ID.IsNull() && config.ID.ValueString() != "" {
		respBody, err = d.client.Read(ctx, trustProviderResourceType, config.ID.ValueString())
	} else if !config.Name.IsNull() && config.Name.ValueString() != "" {
		respBody, err = d.client.List(ctx, trustProviderResourceType, map[string]string{"search": config.Name.ValueString()})
		if err == nil {
			respBody, err = extractFirstItem(respBody, config.Name.ValueString(), "Trust Provider")
		}
	} else {
		resp.Diagnostics.AddError("Missing filter", "Either id or name must be specified.")
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Error reading Trust Provider", err.Error())
		return
	}

	var result apiTrustProvider
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	state := trustProviderDataSourceModel{
		ID:              types.StringValue(result.ID),
		Name:            types.StringValue(result.Name),
		IsActive:        types.BoolValue(result.IsActive),
		Type:            types.StringValue(result.Type),
		EdgeSDKClientID: types.StringValue(result.EdgeSDKClientID),
		EdgeSDKAudience: types.StringValue(result.EdgeSDKAudience),
	}
	if result.Description != "" {
		state.Description = types.StringValue(result.Description)
	} else {
		state.Description = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
