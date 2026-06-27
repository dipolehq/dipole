/*
data_source_client_workload.go — Data source for dipole_client_workload.

Read-only lookup of an existing Client Workload by ID or name. Used when
referencing resources not managed by this Terraform configuration.
*/
package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dipole-labs/terraform-provider-dipole/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type clientWorkloadDataSource struct {
	client *client.Client
}

type clientWorkloadDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IsActive    types.Bool   `tfsdk:"is_active"`
	Identifiers types.Set    `tfsdk:"identifiers"`
}

func NewClientWorkloadDataSource() datasource.DataSource {
	return &clientWorkloadDataSource{}
}

func (d *clientWorkloadDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_client_workload"
}

func (d *clientWorkloadDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Dipole Client Workload by ID or name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "UUID of the Client Workload to look up.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Name of the Client Workload to look up.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Description of the Client Workload.",
			},
			"is_active": schema.BoolAttribute{
				Computed:    true,
				Description: "Active status of the Client Workload.",
			},
			"identifiers": schema.SetNestedAttribute{
				Computed:    true,
				Description: "Set of Client Workload identifiers.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type":  schema.StringAttribute{Computed: true},
						"value": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *clientWorkloadDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *clientWorkloadDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config clientWorkloadDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var respBody json.RawMessage
	var err error

	if !config.ID.IsNull() && config.ID.ValueString() != "" {
		respBody, err = d.client.Read(ctx, clientWorkloadResourceType, config.ID.ValueString())
	} else if !config.Name.IsNull() && config.Name.ValueString() != "" {
		respBody, err = d.client.List(ctx, clientWorkloadResourceType, map[string]string{"search": config.Name.ValueString()})
		if err == nil {
			respBody, err = extractFirstItem(respBody, config.Name.ValueString(), "Client Workload")
		}
	} else {
		resp.Diagnostics.AddError("Missing filter", "Either id or name must be specified.")
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Error reading Client Workload", err.Error())
		return
	}

	var result apiClientWorkload
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	state := clientWorkloadDataSourceModel{
		ID:       types.StringValue(result.ID),
		Name:     types.StringValue(result.Name),
		IsActive: types.BoolValue(result.IsActive),
	}

	if result.Description != "" {
		state.Description = types.StringValue(result.Description)
	} else {
		state.Description = types.StringNull()
	}

	idElements := make([]attr.Value, 0, len(result.Identifiers))
	for _, id := range result.Identifiers {
		obj, _ := types.ObjectValue(identifierAttrTypes, map[string]attr.Value{
			"type":  types.StringValue(id.Type),
			"value": types.StringValue(id.Value),
		})
		idElements = append(idElements, obj)
	}
	state.Identifiers, _ = types.SetValue(types.ObjectType{AttrTypes: identifierAttrTypes}, idElements)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func extractFirstItem(listResponse json.RawMessage, name, resourceType string) (json.RawMessage, error) {
	var paginated struct {
		Items []json.RawMessage `json:"items"`
		Total int               `json:"total"`
	}
	if err := json.Unmarshal(listResponse, &paginated); err != nil {
		return nil, fmt.Errorf("parsing list response: %w", err)
	}
	if len(paginated.Items) == 0 {
		return nil, fmt.Errorf("%s with name %q not found", resourceType, name)
	}
	return paginated.Items[0], nil
}
