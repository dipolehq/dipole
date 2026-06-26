/*
resource_client_workload.go — Terraform resource for dipole_client_workload.

Manages Client Workload entities via the REST API. Supports full CRUD lifecycle
and import by ID. Client Workloads represent software that initiates access
requests, identified by runtime environment attributes (K8s namespace, pod name, etc).
Uses Set for identifiers (unordered, deduplicated) and supports tags with
provider-level default_tags merging via tags_all.
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const clientWorkloadResourceType = "client-workloads"

type clientWorkloadResource struct {
	client      *client.Client
	defaultTags map[string]string
}

type clientWorkloadModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	IsActive      types.Bool   `tfsdk:"is_active"`
	Identifiers   types.Set    `tfsdk:"identifiers"`
	ResourceSetID types.String `tfsdk:"resource_set_id"`
	Tags          types.Map    `tfsdk:"tags"`
	TagsAll       types.Map    `tfsdk:"tags_all"`
}

var identifierAttrTypes = map[string]attr.Type{
	"type":  types.StringType,
	"value": types.StringType,
}

func NewClientWorkloadResource() resource.Resource {
	return &clientWorkloadResource{}
}

func (r *clientWorkloadResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_client_workload"
}

func (r *clientWorkloadResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Dipole Client Workload.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID identifier of the Client Workload.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name of the Client Workload.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the Client Workload.",
			},
			"is_active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Active status of the Client Workload.",
			},
			"identifiers": schema.SetNestedAttribute{
				Required:    true,
				Description: "Set of Client Workload identifiers. All must match (AND logic).",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:    true,
							Description: "Identifier type (e.g. k8s_namespace, k8s_pod_name_prefix, aws_account_id).",
						},
						"value": schema.StringAttribute{
							Required:    true,
							Description: "Identifier value to match against.",
						},
					},
				},
			},
			"resource_set_id": schema.StringAttribute{
				Optional:    true,
				Description: "Resource Set ID to assign this workload to.",
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

func (r *clientWorkloadResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *clientWorkloadResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clientWorkloadModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.modelToAPI(ctx, plan)
	respBody, err := r.client.Create(ctx, clientWorkloadResourceType, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Client Workload", err.Error())
		return
	}

	var result apiClientWorkload
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.apiToModel(ctx, result))...)
}

func (r *clientWorkloadResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clientWorkloadModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	respBody, err := r.client.Read(ctx, clientWorkloadResourceType, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Client Workload", err.Error())
		return
	}

	var result apiClientWorkload
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.apiToModel(ctx, result))...)
}

func (r *clientWorkloadResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan clientWorkloadModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.modelToAPI(ctx, plan)
	respBody, err := r.client.Update(ctx, clientWorkloadResourceType, plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Client Workload", err.Error())
		return
	}

	var result apiClientWorkload
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.apiToModel(ctx, result))...)
}

func (r *clientWorkloadResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clientWorkloadModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, clientWorkloadResourceType, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting Client Workload", err.Error())
	}
}

func (r *clientWorkloadResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type apiClientWorkload struct {
	ID            string          `json:"ID"`
	Name          string          `json:"Name"`
	Description   string          `json:"Description"`
	IsActive      bool            `json:"IsActive"`
	Identifiers   []apiIdentifier `json:"Identifiers"`
	Tags          []apiTag        `json:"Tags"`
	ResourceSetID *string         `json:"ResourceSetID"`
}

type apiIdentifier struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type apiTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (r *clientWorkloadResource) modelToAPI(ctx context.Context, m clientWorkloadModel) apiClientWorkload {
	isActive := true
	if !m.IsActive.IsNull() && !m.IsActive.IsUnknown() {
		isActive = m.IsActive.ValueBool()
	}

	api := apiClientWorkload{
		Name:        m.Name.ValueString(),
		Description: m.Description.ValueString(),
		IsActive:    isActive,
	}

	var identifiers []struct {
		Type  string `tfsdk:"type"`
		Value string `tfsdk:"value"`
	}
	m.Identifiers.ElementsAs(ctx, &identifiers, false)
	for _, id := range identifiers {
		api.Identifiers = append(api.Identifiers, apiIdentifier{Type: id.Type, Value: id.Value})
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

func (r *clientWorkloadResource) apiToModel(ctx context.Context, api apiClientWorkload) *clientWorkloadModel {
	m := &clientWorkloadModel{
		ID:       types.StringValue(api.ID),
		Name:     types.StringValue(api.Name),
		IsActive: types.BoolValue(api.IsActive),
	}

	if api.Description != "" {
		m.Description = types.StringValue(api.Description)
	} else {
		m.Description = types.StringNull()
	}

	// Identifiers → Set
	idElements := make([]attr.Value, 0, len(api.Identifiers))
	for _, id := range api.Identifiers {
		obj, _ := types.ObjectValue(identifierAttrTypes, map[string]attr.Value{
			"type":  types.StringValue(id.Type),
			"value": types.StringValue(id.Value),
		})
		idElements = append(idElements, obj)
	}
	m.Identifiers, _ = types.SetValue(types.ObjectType{AttrTypes: identifierAttrTypes}, idElements)

	// Split tags into user tags and tags_all
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

func (r *clientWorkloadResource) mergeTags(ctx context.Context, userTags types.Map) map[string]string {
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
