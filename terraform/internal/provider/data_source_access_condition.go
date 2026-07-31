/*
data_source_access_condition.go — Data source for dipole_access_condition.

Read-only lookup of an existing Access Condition by ID or name.
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

type accessConditionDataSource struct {
	client *client.Client
}

type accessConditionDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IsActive    types.Bool   `tfsdk:"is_active"`
	Type        types.String `tfsdk:"type"`
}

func NewAccessConditionDataSource() datasource.DataSource {
	return &accessConditionDataSource{}
}

func (d *accessConditionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_condition"
}

func (d *accessConditionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Dipole Access Condition by ID or name.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Optional: true, Computed: true, Description: "UUID of the Access Condition."},
			"name":        schema.StringAttribute{Optional: true, Computed: true, Description: "Name of the Access Condition."},
			"description": schema.StringAttribute{Computed: true, Description: "Description."},
			"is_active":   schema.BoolAttribute{Computed: true, Description: "Active status."},
			"type":        schema.StringAttribute{Computed: true, Description: "Access Condition type (e.g., time, geoip)."},
		},
	}
}

func (d *accessConditionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *accessConditionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config accessConditionDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var respBody json.RawMessage
	var err error

	if !config.ID.IsNull() && config.ID.ValueString() != "" {
		respBody, err = d.client.Read(ctx, accessConditionResourceType, config.ID.ValueString())
	} else if !config.Name.IsNull() && config.Name.ValueString() != "" {
		respBody, err = d.client.List(ctx, accessConditionResourceType, map[string]string{"search": config.Name.ValueString()})
		if err == nil {
			respBody, err = extractFirstItem(respBody, config.Name.ValueString(), "Access Condition")
		}
	} else {
		resp.Diagnostics.AddError("Missing filter", "Either id or name must be specified.")
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Error reading Access Condition", err.Error())
		return
	}

	var result apiAccessCondition
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	state := accessConditionDataSourceModel{
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
