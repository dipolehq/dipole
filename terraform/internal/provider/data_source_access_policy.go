/*
data_source_access_policy.go — Data source for dipole_access_policy.

Read-only lookup of an existing Access Policy by ID or name.
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

type accessPolicyDataSource struct {
	client *client.Client
}

type accessPolicyDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	IsActive         types.Bool   `tfsdk:"is_active"`
	ClientWorkloadID types.String `tfsdk:"client_workload_id"`
	ServerWorkloadID types.String `tfsdk:"server_workload_id"`
}

func NewAccessPolicyDataSource() datasource.DataSource {
	return &accessPolicyDataSource{}
}

func (d *accessPolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_policy"
}

func (d *accessPolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Dipole Access Policy by ID or name.",
		Attributes: map[string]schema.Attribute{
			"id":                 schema.StringAttribute{Optional: true, Computed: true, Description: "UUID of the Access Policy."},
			"name":               schema.StringAttribute{Optional: true, Computed: true, Description: "Name of the Access Policy."},
			"description":        schema.StringAttribute{Computed: true, Description: "Description."},
			"is_active":          schema.BoolAttribute{Computed: true, Description: "Active status."},
			"client_workload_id": schema.StringAttribute{Computed: true, Description: "Client Workload ID."},
			"server_workload_id": schema.StringAttribute{Computed: true, Description: "Server Workload ID."},
		},
	}
}

func (d *accessPolicyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *accessPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config accessPolicyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var respBody json.RawMessage
	var err error

	if !config.ID.IsNull() && config.ID.ValueString() != "" {
		respBody, err = d.client.Read(ctx, accessPolicyResourceType, config.ID.ValueString())
	} else if !config.Name.IsNull() && config.Name.ValueString() != "" {
		respBody, err = d.client.List(ctx, accessPolicyResourceType, map[string]string{"search": config.Name.ValueString()})
		if err == nil {
			respBody, err = extractFirstItem(respBody, config.Name.ValueString(), "Access Policy")
		}
	} else {
		resp.Diagnostics.AddError("Missing filter", "Either id or name must be specified.")
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Error reading Access Policy", err.Error())
		return
	}

	var result apiAccessPolicy
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	state := accessPolicyDataSourceModel{
		ID:               types.StringValue(result.ID),
		Name:             types.StringValue(result.Name),
		IsActive:         types.BoolValue(result.IsActive),
		ClientWorkloadID: types.StringValue(result.ClientWorkloadID),
		ServerWorkloadID: types.StringValue(result.ServerWorkloadID),
	}
	if result.Description != "" {
		state.Description = types.StringValue(result.Description)
	} else {
		state.Description = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
