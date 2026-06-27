/*
data_source_server_workload.go — Data source for dipole_server_workload.

Read-only lookup of an existing Server Workload by ID or name.
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

type serverWorkloadDataSource struct {
	client *client.Client
}

type serverWorkloadDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IsActive    types.Bool   `tfsdk:"is_active"`
	Host        types.String `tfsdk:"host"`
	Port        types.Int64  `tfsdk:"port"`
	AppProtocol types.String `tfsdk:"app_protocol"`
}

func NewServerWorkloadDataSource() datasource.DataSource {
	return &serverWorkloadDataSource{}
}

func (d *serverWorkloadDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_workload"
}

func (d *serverWorkloadDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Dipole Server Workload by ID or name.",
		Attributes: map[string]schema.Attribute{
			"id":           schema.StringAttribute{Optional: true, Computed: true, Description: "UUID of the Server Workload."},
			"name":         schema.StringAttribute{Optional: true, Computed: true, Description: "Name of the Server Workload."},
			"description":  schema.StringAttribute{Computed: true, Description: "Description."},
			"is_active":    schema.BoolAttribute{Computed: true, Description: "Active status."},
			"host":         schema.StringAttribute{Computed: true, Description: "Service endpoint hostname."},
			"port":         schema.Int64Attribute{Computed: true, Description: "Service endpoint port."},
			"app_protocol": schema.StringAttribute{Computed: true, Description: "Application protocol."},
		},
	}
}

func (d *serverWorkloadDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *serverWorkloadDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config serverWorkloadDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var respBody json.RawMessage
	var err error

	if !config.ID.IsNull() && config.ID.ValueString() != "" {
		respBody, err = d.client.Read(ctx, serverWorkloadResourceType, config.ID.ValueString())
	} else if !config.Name.IsNull() && config.Name.ValueString() != "" {
		respBody, err = d.client.List(ctx, serverWorkloadResourceType, map[string]string{"search": config.Name.ValueString()})
		if err == nil {
			respBody, err = extractFirstItem(respBody, config.Name.ValueString(), "Server Workload")
		}
	} else {
		resp.Diagnostics.AddError("Missing filter", "Either id or name must be specified.")
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Error reading Server Workload", err.Error())
		return
	}

	var result apiServerWorkload
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	state := serverWorkloadDataSourceModel{
		ID:          types.StringValue(result.ID),
		Name:        types.StringValue(result.Name),
		IsActive:    types.BoolValue(result.IsActive),
		Host:        types.StringValue(result.Host),
		Port:        types.Int64Value(int64(result.Port)),
		AppProtocol: types.StringValue(result.AppProtocol),
	}
	if result.Description != "" {
		state.Description = types.StringValue(result.Description)
	} else {
		state.Description = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
