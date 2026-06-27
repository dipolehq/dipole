/*
resource_trust_provider.go — Terraform resource for dipole_trust_provider.

Manages Trust Provider entities via the REST API. Trust Providers verify
workload identity through cryptographic attestation (K8s SA tokens, AWS metadata,
GitHub/GitLab OIDC, etc). Config varies by type so we use a config_json string
attribute. Computed fields: edge_sdk_client_id, edge_sdk_audience.
*/
package provider

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/dipole-labs/terraform-provider-dipole/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const trustProviderResourceType = "trust-providers"

type trustProviderResource struct {
	client      *client.Client
	defaultTags map[string]string
}

type trustProviderModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	IsActive        types.Bool   `tfsdk:"is_active"`
	Type            types.String `tfsdk:"type"`
	ConfigJSON      types.String `tfsdk:"config_json"`
	EdgeSDKClientID types.String `tfsdk:"edge_sdk_client_id"`
	EdgeSDKAudience types.String `tfsdk:"edge_sdk_audience"`
	ResourceSetID   types.String `tfsdk:"resource_set_id"`
	Tags            types.Map    `tfsdk:"tags"`
	TagsAll         types.Map    `tfsdk:"tags_all"`
}

func NewTrustProviderResource() resource.Resource {
	return &trustProviderResource{}
}

func (r *trustProviderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_trust_provider"
}

func (r *trustProviderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Dipole Trust Provider.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID identifier of the Trust Provider.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name of the Trust Provider.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the Trust Provider.",
			},
			"is_active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Active status of the Trust Provider.",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "Trust Provider type (kubernetes_service_account, github_action, gitlab_job, aws_metadata_service, aws_role, azure_metadata, gcp_identity_token, oidc_id_token, terraform_cloud, kerberos, certificate_signed_attestation, samlv2_response).",
			},
			"config_json": schema.StringAttribute{
				Required:    true,
				Description: "Trust Provider configuration as JSON. Structure varies by type. Use jsonencode() to provide.",
			},
			"edge_sdk_client_id": schema.StringAttribute{
				Computed:    true,
				Description: "Auto-generated Edge SDK Client ID for this Trust Provider.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"edge_sdk_audience": schema.StringAttribute{
				Computed:    true,
				Description: "Auto-generated Edge SDK Audience for this Trust Provider.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_set_id": schema.StringAttribute{
				Optional:    true,
				Description: "Resource Set ID to assign this Trust Provider to.",
			},
			"tags": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Tags are key-value pairs.",
			},
			"tags_all": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "All tags including provider-level default tags.",
			},
		},
	}
}

func (r *trustProviderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	providerData, ok := req.ProviderData.(*DipoleProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Data", "Expected *DipoleProviderData")
		return
	}
	r.client = providerData.Client
	r.defaultTags = providerData.DefaultTags
}

func (r *trustProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan trustProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.modelToAPI(ctx, plan)
	respBody, err := r.client.Create(ctx, trustProviderResourceType, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Trust Provider", err.Error())
		return
	}

	var result apiTrustProvider
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.apiToModel(ctx, result))...)
}

func (r *trustProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state trustProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	respBody, err := r.client.Read(ctx, trustProviderResourceType, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Trust Provider", err.Error())
		return
	}

	var result apiTrustProvider
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.apiToModel(ctx, result))...)
}

func (r *trustProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan trustProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.modelToAPI(ctx, plan)
	respBody, err := r.client.Update(ctx, trustProviderResourceType, plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Trust Provider", err.Error())
		return
	}

	var result apiTrustProvider
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.apiToModel(ctx, result))...)
}

func (r *trustProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state trustProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, trustProviderResourceType, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting Trust Provider", err.Error())
	}
}

func (r *trustProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type apiTrustProvider struct {
	ID              string          `json:"ID"`
	Name            string          `json:"Name"`
	Description     string          `json:"Description"`
	IsActive        bool            `json:"IsActive"`
	Type            string          `json:"Type"`
	Config          json.RawMessage `json:"Config"`
	MatchRules      json.RawMessage `json:"MatchRules"`
	EdgeSDKClientID string          `json:"EdgeSDKClientID"`
	EdgeSDKAudience string          `json:"EdgeSDKAudience"`
	Tags            []apiTag        `json:"Tags"`
	ResourceSetID   *string         `json:"ResourceSetID"`
}

func (r *trustProviderResource) modelToAPI(ctx context.Context, m trustProviderModel) apiTrustProvider {
	api := apiTrustProvider{
		Name:        m.Name.ValueString(),
		Description: m.Description.ValueString(),
		IsActive:    m.IsActive.ValueBool(),
		Type:        m.Type.ValueString(),
	}

	if !m.ConfigJSON.IsNull() && !m.ConfigJSON.IsUnknown() {
		rawConfig := []byte(m.ConfigJSON.ValueString())

		// Extract match_rules from config_json if present, send as separate field
		var parsed map[string]json.RawMessage
		if json.Unmarshal(rawConfig, &parsed) == nil {
			if mr, ok := parsed["match_rules"]; ok {
				api.MatchRules = mr
				delete(parsed, "match_rules")
				remaining, _ := json.Marshal(parsed)
				api.Config = remaining
			} else {
				api.Config = rawConfig
			}
		} else {
			api.Config = rawConfig
		}
	}

	merged := r.mergeTags(ctx, m.Tags)
	for k, v := range merged {
		api.Tags = append(api.Tags, apiTag{Key: k, Value: v})
	}

	if !m.ResourceSetID.IsNull() && !m.ResourceSetID.IsUnknown() {
		v := m.ResourceSetID.ValueString()
		api.ResourceSetID = &v
	}
	return api
}

func (r *trustProviderResource) apiToModel(ctx context.Context, api apiTrustProvider) *trustProviderModel {
	m := &trustProviderModel{
		ID:       types.StringValue(api.ID),
		Name:     types.StringValue(api.Name),
		IsActive: types.BoolValue(api.IsActive),
		Type:     types.StringValue(api.Type),
	}

	if api.Description != "" {
		m.Description = types.StringValue(api.Description)
	} else {
		m.Description = types.StringNull()
	}

	if len(api.Config) > 0 && string(api.Config) != "null" {
		// Merge match_rules back into config_json for round-trip consistency
		var parsed map[string]json.RawMessage
		if json.Unmarshal(api.Config, &parsed) == nil {
			if len(api.MatchRules) > 0 && string(api.MatchRules) != "null" {
				parsed["match_rules"] = api.MatchRules
			}
			merged, _ := json.Marshal(parsed)
			m.ConfigJSON = types.StringValue(string(merged))
		} else {
			m.ConfigJSON = types.StringValue(string(api.Config))
		}
	} else if len(api.MatchRules) > 0 && string(api.MatchRules) != "null" {
		cfg, _ := json.Marshal(map[string]json.RawMessage{"match_rules": api.MatchRules})
		m.ConfigJSON = types.StringValue(string(cfg))
	} else {
		m.ConfigJSON = types.StringValue("{}")
	}

	if api.EdgeSDKClientID != "" {
		m.EdgeSDKClientID = types.StringValue(api.EdgeSDKClientID)
	} else {
		m.EdgeSDKClientID = types.StringValue("")
	}

	if api.EdgeSDKAudience != "" {
		m.EdgeSDKAudience = types.StringValue(api.EdgeSDKAudience)
	} else {
		m.EdgeSDKAudience = types.StringValue("")
	}

	// Tags
	allTags := make(map[string]string, len(api.Tags))
	for _, t := range api.Tags {
		allTags[t.Key] = t.Value
	}

	userTags := make(map[string]string)
	for k, v := range allTags {
		if _, isDefault := r.defaultTags[k]; !isDefault {
			userTags[k] = v
		}
	}

	if len(userTags) > 0 {
		tagElements := make(map[string]attr.Value, len(userTags))
		for k, v := range userTags {
			tagElements[k] = types.StringValue(v)
		}
		m.Tags, _ = types.MapValue(types.StringType, tagElements)
	} else {
		m.Tags = types.MapNull(types.StringType)
	}

	if len(allTags) > 0 {
		allElements := make(map[string]attr.Value, len(allTags))
		for k, v := range allTags {
			allElements[k] = types.StringValue(v)
		}
		m.TagsAll, _ = types.MapValue(types.StringType, allElements)
	} else {
		m.TagsAll, _ = types.MapValue(types.StringType, map[string]attr.Value{})
	}

	if api.ResourceSetID != nil && *api.ResourceSetID != "" {
		m.ResourceSetID = types.StringValue(*api.ResourceSetID)
	} else {
		m.ResourceSetID = types.StringNull()
	}

	return m
}

func (r *trustProviderResource) mergeTags(ctx context.Context, userTags types.Map) map[string]string {
	merged := make(map[string]string)
	for k, v := range r.defaultTags {
		merged[k] = v
	}
	if !userTags.IsNull() && !userTags.IsUnknown() {
		var tags map[string]string
		userTags.ElementsAs(ctx, &tags, false)
		for k, v := range tags {
			merged[k] = v
		}
	}
	return merged
}
