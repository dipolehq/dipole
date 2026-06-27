/*
resource_role.go — Terraform resource for dipole_role.

Manages Role entities via the REST API. Roles define RBAC permissions for
users — granular read/write access to each resource type within the tenant.
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

const roleResourceType = "roles"

type roleResource struct {
	client *client.Client
}

type roleModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	IsActive        types.Bool   `tfsdk:"is_active"`
	PermissionsJSON types.String `tfsdk:"permissions_json"`
	ResourceSetIDs  types.Set    `tfsdk:"resource_set_ids"`
}

func NewRoleResource() resource.Resource {
	return &roleResource{}
}

func (r *roleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *roleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Dipole Role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID identifier of the Role.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name of the Role.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the Role.",
			},
			"is_active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Active status of the Role.",
			},
			"permissions_json": schema.StringAttribute{
				Required:    true,
				Description: "Role permissions as JSON. Defines read/write access per resource type. Use jsonencode() to provide.",
			},
			"resource_set_ids": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Set of Resource Set IDs this role is assigned to.",
			},
		},
	}
}

func (r *roleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := roleToAPI(plan)
	respBody, err := r.client.Create(ctx, roleResourceType, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Role", err.Error())
		return
	}

	var result apiRole
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, roleFromAPI(result))...)
}

func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state roleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	respBody, err := r.client.Read(ctx, roleResourceType, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Role", err.Error())
		return
	}

	var result apiRole
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, roleFromAPI(result))...)
}

func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan roleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := roleToAPI(plan)
	respBody, err := r.client.Update(ctx, roleResourceType, plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Role", err.Error())
		return
	}

	var result apiRole
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, roleFromAPI(result))...)
}

func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, roleResourceType, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting Role", err.Error())
	}
}

func (r *roleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type apiRole struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	IsActive       bool            `json:"isActive"`
	Permissions    json.RawMessage `json:"permissions"`
	ResourceSetIDs []string        `json:"resourceSetIds"`
}

func roleToAPI(m roleModel) apiRole {
	api := apiRole{
		Name:        m.Name.ValueString(),
		Description: m.Description.ValueString(),
		IsActive:    m.IsActive.ValueBool(),
	}
	if !m.PermissionsJSON.IsNull() && !m.PermissionsJSON.IsUnknown() {
		api.Permissions = json.RawMessage(m.PermissionsJSON.ValueString())
	}
	if !m.ResourceSetIDs.IsNull() && !m.ResourceSetIDs.IsUnknown() {
		var ids []string
		m.ResourceSetIDs.ElementsAs(context.Background(), &ids, false)
		api.ResourceSetIDs = ids
	}
	return api
}

func roleFromAPI(api apiRole) *roleModel {
	m := &roleModel{
		ID:       types.StringValue(api.ID),
		Name:     types.StringValue(api.Name),
		IsActive: types.BoolValue(api.IsActive),
	}

	if api.Description != "" {
		m.Description = types.StringValue(api.Description)
	} else {
		m.Description = types.StringNull()
	}

	if len(api.Permissions) > 0 && string(api.Permissions) != "null" {
		m.PermissionsJSON = types.StringValue(string(api.Permissions))
	} else {
		m.PermissionsJSON = types.StringValue("{}")
	}

	if len(api.ResourceSetIDs) > 0 {
		elements := make([]attr.Value, 0, len(api.ResourceSetIDs))
		for _, id := range api.ResourceSetIDs {
			elements = append(elements, types.StringValue(id))
		}
		m.ResourceSetIDs, _ = types.SetValue(types.StringType, elements)
	} else {
		m.ResourceSetIDs = types.SetNull(types.StringType)
	}

	return m
}
