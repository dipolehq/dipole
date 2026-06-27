/*
resource_access_condition.go — Terraform resource for dipole_access_condition.

Manages Access Condition entities via the REST API. Access Conditions enforce
contextual security requirements (time-based, GeoIP, CrowdStrike, Wiz) that
must all pass (AND logic) before access is granted.
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

const accessConditionResourceType = "access-conditions"

type accessConditionResource struct {
	client      *client.Client
	defaultTags map[string]string
}

type accessConditionModel struct {
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

func NewAccessConditionResource() resource.Resource {
	return &accessConditionResource{}
}

func (r *accessConditionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_condition"
}

func (r *accessConditionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Dipole Access Condition.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID identifier of the Access Condition.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name of the Access Condition.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the Access Condition.",
			},
			"is_active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Active status of the Access Condition.",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "Access Condition type (time_condition, geoip_condition).",
			},
			"config_json": schema.StringAttribute{
				Required:    true,
				Description: "Access Condition configuration as JSON. Structure varies by type. Use jsonencode() to provide.",
			},
			"resource_set_id": schema.StringAttribute{
				Optional:    true,
				Description: "Resource Set ID to assign this Access Condition to.",
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

func (r *accessConditionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *accessConditionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan accessConditionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.modelToAPI(ctx, plan)
	respBody, err := r.client.Create(ctx, accessConditionResourceType, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Access Condition", err.Error())
		return
	}

	var result apiAccessCondition
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.apiToModel(ctx, result))...)
}

func (r *accessConditionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state accessConditionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	respBody, err := r.client.Read(ctx, accessConditionResourceType, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Access Condition", err.Error())
		return
	}

	var result apiAccessCondition
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.apiToModel(ctx, result))...)
}

func (r *accessConditionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan accessConditionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.modelToAPI(ctx, plan)
	respBody, err := r.client.Update(ctx, accessConditionResourceType, plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Access Condition", err.Error())
		return
	}

	var result apiAccessCondition
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.apiToModel(ctx, result))...)
}

func (r *accessConditionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state accessConditionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, accessConditionResourceType, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting Access Condition", err.Error())
	}
}

func (r *accessConditionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type apiAccessCondition struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	IsActive      bool            `json:"isActive"`
	Type          string          `json:"type"`
	Config        json.RawMessage `json:"config"`
	Tags          []apiTag        `json:"tags"`
	ResourceSetID *string         `json:"resourceSetId"`
}

func (r *accessConditionResource) modelToAPI(ctx context.Context, m accessConditionModel) apiAccessCondition {
	api := apiAccessCondition{
		Name:        m.Name.ValueString(),
		Description: m.Description.ValueString(),
		IsActive:    m.IsActive.ValueBool(),
		Type:        m.Type.ValueString(),
	}

	if !m.ConfigJSON.IsNull() && !m.ConfigJSON.IsUnknown() {
		api.Config = json.RawMessage(m.ConfigJSON.ValueString())
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

func (r *accessConditionResource) apiToModel(ctx context.Context, api apiAccessCondition) *accessConditionModel {
	m := &accessConditionModel{
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
		m.ConfigJSON = types.StringValue(string(api.Config))
	} else {
		m.ConfigJSON = types.StringValue("{}")
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

func (r *accessConditionResource) mergeTags(ctx context.Context, userTags types.Map) map[string]string {
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
