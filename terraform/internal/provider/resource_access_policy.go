/*
resource_access_policy.go — Terraform resource for dipole_access_policy.

Manages Access Policy entities via the REST API. An Access Policy links a Client
Workload to a Server Workload with Trust Providers, Credential Providers, and
Access Conditions. Supports both singular credential_provider_id (simple case)
and plural credential_providers (with mapping) for multi-credential policies.
*/
package provider

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/dipolehq/terraform-provider-dipole/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const accessPolicyResourceType = "access-policies"

type accessPolicyResource struct {
	client *client.Client
}

type accessPolicyModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	IsActive             types.Bool   `tfsdk:"is_active"`
	ClientWorkloadID     types.String `tfsdk:"client_workload_id"`
	ServerWorkloadID     types.String `tfsdk:"server_workload_id"`
	TrustProviderIDs     types.Set    `tfsdk:"trust_provider_ids"`
	CredentialProviderID types.String `tfsdk:"credential_provider_id"`
	CredentialProviders  types.Set    `tfsdk:"credential_providers"`
	AccessConditionIDs   types.Set    `tfsdk:"access_condition_ids"`
	ResourceSetID        types.String `tfsdk:"resource_set_id"`
}

var credentialProviderMappingAttrTypes = map[string]attr.Type{
	"credential_provider_id": types.StringType,
	"mapping_type":           types.StringType,
	"mapping_value":          types.StringType,
}

func NewAccessPolicyResource() resource.Resource {
	return &accessPolicyResource{}
}

func (r *accessPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_policy"
}

func (r *accessPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Dipole Access Policy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID identifier of the Access Policy.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name of the Access Policy.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the Access Policy.",
			},
			"is_active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Active status of the Access Policy.",
			},
			"client_workload_id": schema.StringAttribute{
				Required:    true,
				Description: "Client Workload ID configured in the Access Policy.",
			},
			"server_workload_id": schema.StringAttribute{
				Required:    true,
				Description: "Server Workload ID configured in the Access Policy.",
			},
			"trust_provider_ids": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Set of Trust Provider IDs to enforce on the Access Policy.",
			},
			"credential_provider_id": schema.StringAttribute{
				Optional:    true,
				Description: "Credential Provider ID configured in the Access Policy (simple single-credential case).",
			},
			"credential_providers": schema.SetNestedAttribute{
				Optional:    true,
				Description: "Set of Credential Providers with mapping configuration (for multi-credential policies).",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"credential_provider_id": schema.StringAttribute{
							Required:    true,
							Description: "ID of the associated Credential Provider.",
						},
						"mapping_type": schema.StringAttribute{
							Required:    true,
							Description: "Mapping type for the credential provider.",
						},
						"mapping_value": schema.StringAttribute{
							Optional:    true,
							Description: "Mapping value (e.g. header name, access key ID).",
						},
					},
				},
			},
			"access_condition_ids": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Set of Access Condition IDs to enforce on the Access Policy.",
			},
			"resource_set_id": schema.StringAttribute{
				Optional:    true,
				Description: "Resource Set ID to assign this Access Policy to.",
			},
		},
	}
}

func (r *accessPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	providerData, ok := req.ProviderData.(*DipoleProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Data", "Expected *DipoleProviderData")
		return
	}
	r.client = providerData.Client
}

func (r *accessPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan accessPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.modelToAPI(ctx, plan)
	respBody, err := r.client.Create(ctx, accessPolicyResourceType, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Access Policy", err.Error())
		return
	}

	var result apiAccessPolicy
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.apiToModel(ctx, result))...)
}

func (r *accessPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state accessPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	respBody, err := r.client.Read(ctx, accessPolicyResourceType, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Access Policy", err.Error())
		return
	}

	var result apiAccessPolicy
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.apiToModel(ctx, result))...)
}

func (r *accessPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan accessPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.modelToAPI(ctx, plan)
	respBody, err := r.client.Update(ctx, accessPolicyResourceType, plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Access Policy", err.Error())
		return
	}

	var result apiAccessPolicy
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.apiToModel(ctx, result))...)
}

func (r *accessPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state accessPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, accessPolicyResourceType, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting Access Policy", err.Error())
	}
}

func (r *accessPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type apiAccessPolicy struct {
	ID                    string                        `json:"id"`
	Name                  string                        `json:"name"`
	Description           string                        `json:"description"`
	IsActive              bool                          `json:"isActive"`
	ClientWorkloadID      string                        `json:"clientWorkloadId"`
	ServerWorkloadID      string                        `json:"serverWorkloadId"`
	TrustProviderIDs      []string                      `json:"trustProviderIds"`
	CredentialProviderIDs []apiPolicyCredentialProvider `json:"credentialProviderIds"`
	AccessConditionIDs    []string                      `json:"accessConditionIds"`
	ResourceSetID         *string                       `json:"resourceSetId"`
}

type apiPolicyCredentialProvider struct {
	CredentialProviderID string `json:"credentialProviderId"`
	MappingType          string `json:"mappingType"`
	MappingValue         string `json:"mappingValue"`
}

func (r *accessPolicyResource) modelToAPI(ctx context.Context, m accessPolicyModel) apiAccessPolicy {
	api := apiAccessPolicy{
		Name:             m.Name.ValueString(),
		Description:      m.Description.ValueString(),
		IsActive:         m.IsActive.ValueBool(),
		ClientWorkloadID: m.ClientWorkloadID.ValueString(),
		ServerWorkloadID: m.ServerWorkloadID.ValueString(),
	}

	if !m.TrustProviderIDs.IsNull() && !m.TrustProviderIDs.IsUnknown() {
		var ids []string
		m.TrustProviderIDs.ElementsAs(ctx, &ids, false)
		api.TrustProviderIDs = ids
	}

	if !m.CredentialProviderID.IsNull() && !m.CredentialProviderID.IsUnknown() {
		api.CredentialProviderIDs = []apiPolicyCredentialProvider{
			{CredentialProviderID: m.CredentialProviderID.ValueString(), MappingType: "default"},
		}
	} else if !m.CredentialProviders.IsNull() && !m.CredentialProviders.IsUnknown() {
		var cps []struct {
			CredentialProviderID string `tfsdk:"credential_provider_id"`
			MappingType          string `tfsdk:"mapping_type"`
			MappingValue         string `tfsdk:"mapping_value"`
		}
		m.CredentialProviders.ElementsAs(ctx, &cps, false)
		for _, cp := range cps {
			api.CredentialProviderIDs = append(api.CredentialProviderIDs, apiPolicyCredentialProvider{
				CredentialProviderID: cp.CredentialProviderID,
				MappingType:          cp.MappingType,
				MappingValue:         cp.MappingValue,
			})
		}
	}

	if !m.AccessConditionIDs.IsNull() && !m.AccessConditionIDs.IsUnknown() {
		var ids []string
		m.AccessConditionIDs.ElementsAs(ctx, &ids, false)
		api.AccessConditionIDs = ids
	}

	if !m.ResourceSetID.IsNull() && !m.ResourceSetID.IsUnknown() {
		v := m.ResourceSetID.ValueString()
		api.ResourceSetID = &v
	}
	return api
}

func (r *accessPolicyResource) apiToModel(ctx context.Context, api apiAccessPolicy) *accessPolicyModel {
	m := &accessPolicyModel{
		ID:               types.StringValue(api.ID),
		Name:             types.StringValue(api.Name),
		IsActive:         types.BoolValue(api.IsActive),
		ClientWorkloadID: types.StringValue(api.ClientWorkloadID),
		ServerWorkloadID: types.StringValue(api.ServerWorkloadID),
	}

	if api.Description != "" {
		m.Description = types.StringValue(api.Description)
	} else {
		m.Description = types.StringNull()
	}

	// Trust Provider IDs
	if len(api.TrustProviderIDs) > 0 {
		elements := make([]attr.Value, 0, len(api.TrustProviderIDs))
		for _, id := range api.TrustProviderIDs {
			elements = append(elements, types.StringValue(id))
		}
		m.TrustProviderIDs, _ = types.SetValue(types.StringType, elements)
	} else {
		m.TrustProviderIDs = types.SetNull(types.StringType)
	}

	// Credential Provider ID (first one, simple case) + plural credential_providers
	if len(api.CredentialProviderIDs) == 1 && api.CredentialProviderIDs[0].MappingType == "default" {
		m.CredentialProviderID = types.StringValue(api.CredentialProviderIDs[0].CredentialProviderID)
		m.CredentialProviders = types.SetNull(types.ObjectType{AttrTypes: credentialProviderMappingAttrTypes})
	} else if len(api.CredentialProviderIDs) > 0 {
		m.CredentialProviderID = types.StringNull()
		elements := make([]attr.Value, 0, len(api.CredentialProviderIDs))
		for _, cp := range api.CredentialProviderIDs {
			obj, _ := types.ObjectValue(credentialProviderMappingAttrTypes, map[string]attr.Value{
				"credential_provider_id": types.StringValue(cp.CredentialProviderID),
				"mapping_type":           types.StringValue(cp.MappingType),
				"mapping_value":          types.StringValue(cp.MappingValue),
			})
			elements = append(elements, obj)
		}
		m.CredentialProviders, _ = types.SetValue(types.ObjectType{AttrTypes: credentialProviderMappingAttrTypes}, elements)
	} else {
		m.CredentialProviderID = types.StringNull()
		m.CredentialProviders = types.SetNull(types.ObjectType{AttrTypes: credentialProviderMappingAttrTypes})
	}

	// Access Condition IDs
	if len(api.AccessConditionIDs) > 0 {
		elements := make([]attr.Value, 0, len(api.AccessConditionIDs))
		for _, id := range api.AccessConditionIDs {
			elements = append(elements, types.StringValue(id))
		}
		m.AccessConditionIDs, _ = types.SetValue(types.StringType, elements)
	} else {
		m.AccessConditionIDs = types.SetNull(types.StringType)
	}

	if api.ResourceSetID != nil && *api.ResourceSetID != "" {
		m.ResourceSetID = types.StringValue(*api.ResourceSetID)
	} else {
		m.ResourceSetID = types.StringNull()
	}

	return m
}
