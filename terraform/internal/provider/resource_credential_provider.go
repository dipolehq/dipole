/*
resource_credential_provider.go — Terraform resource for dipole_credential_provider.

Manages Credential Provider entities via the REST API. Credential Providers
define how credentials are retrieved (API keys, OAuth tokens, AWS STS, etc).
The config_json field is Sensitive — Terraform masks it in plan/apply output.
The API redacts config on GET, so we preserve config from state on read.
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

const credentialProviderResourceType = "credential-providers"

type credentialProviderResource struct {
	client      *client.Client
	defaultTags map[string]string
}

type credentialProviderModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	IsActive      types.Bool   `tfsdk:"is_active"`
	Type          types.String `tfsdk:"type"`
	ConfigJSON    types.String `tfsdk:"config_json"`
	ResourceSetID types.String `tfsdk:"resource_set_id"`
	Tags          types.Map    `tfsdk:"tags"`
	TagsAll       types.Map    `tfsdk:"tags_all"`
}

func NewCredentialProviderResource() resource.Resource {
	return &credentialProviderResource{}
}

func (r *credentialProviderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_credential_provider"
}

func (r *credentialProviderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Dipole Credential Provider.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID identifier of the Credential Provider.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name of the Credential Provider.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the Credential Provider.",
			},
			"is_active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Active status of the Credential Provider.",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "Credential Provider type (api_key, username_password, oauth_client_credentials, aws_sts_federation, oidc_id_token, json_web_token, vault_client_token, etc).",
			},
			"config_json": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Credential Provider configuration as JSON. Contains secrets — marked sensitive. Use jsonencode() to provide.",
			},
			"resource_set_id": schema.StringAttribute{
				Optional:    true,
				Description: "Resource Set ID to assign this Credential Provider to.",
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

func (r *credentialProviderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *credentialProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan credentialProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.modelToAPI(ctx, plan)
	respBody, err := r.client.Create(ctx, credentialProviderResourceType, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Credential Provider", err.Error())
		return
	}

	var result apiCredentialProvider
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	state := r.apiToModel(ctx, result)
	// Preserve config_json from plan since API redacts it on response
	state.ConfigJSON = plan.ConfigJSON
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *credentialProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state credentialProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	respBody, err := r.client.Read(ctx, credentialProviderResourceType, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Credential Provider", err.Error())
		return
	}

	var result apiCredentialProvider
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	newState := r.apiToModel(ctx, result)
	// Preserve config_json from existing state since API redacts secrets
	newState.ConfigJSON = state.ConfigJSON
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *credentialProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan credentialProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.modelToAPI(ctx, plan)
	respBody, err := r.client.Update(ctx, credentialProviderResourceType, plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Credential Provider", err.Error())
		return
	}

	var result apiCredentialProvider
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	state := r.apiToModel(ctx, result)
	// Preserve config_json from plan since API redacts it
	state.ConfigJSON = plan.ConfigJSON
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *credentialProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state credentialProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, credentialProviderResourceType, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting Credential Provider", err.Error())
	}
}

func (r *credentialProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type apiCredentialProvider struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	IsActive      bool     `json:"isActive"`
	Type          string   `json:"type"`
	ConfigJSON    string   `json:"configJSON,omitempty"`
	Tags          []apiTag `json:"tags"`
	ResourceSetID *string  `json:"resourceSetID"`
}

func (r *credentialProviderResource) modelToAPI(ctx context.Context, m credentialProviderModel) apiCredentialProvider {
	api := apiCredentialProvider{
		Name:        m.Name.ValueString(),
		Description: m.Description.ValueString(),
		IsActive:    m.IsActive.ValueBool(),
		Type:        m.Type.ValueString(),
		ConfigJSON:  m.ConfigJSON.ValueString(),
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

func (r *credentialProviderResource) apiToModel(ctx context.Context, api apiCredentialProvider) *credentialProviderModel {
	m := &credentialProviderModel{
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

	// config_json is NOT set here — it's preserved from plan/state in CRUD methods

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

func (r *credentialProviderResource) mergeTags(ctx context.Context, userTags types.Map) map[string]string {
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
