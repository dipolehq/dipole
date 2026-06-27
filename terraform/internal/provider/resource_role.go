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

	"github.com/dipole-labs/terraform-provider-dipole/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	PermissionsJSON types.String `tfsdk:"permissions_json"`
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
			"permissions_json": schema.StringAttribute{
				Required:    true,
				Description: "Role permissions as JSON. Defines read/write access per resource type. Use jsonencode() to provide.",
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
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Permissions json.RawMessage `json:"permissions"`
}

func roleToAPI(m roleModel) apiRole {
	api := apiRole{
		Name: m.Name.ValueString(),
	}
	if !m.PermissionsJSON.IsNull() && !m.PermissionsJSON.IsUnknown() {
		api.Permissions = json.RawMessage(m.PermissionsJSON.ValueString())
	}
	return api
}

func roleFromAPI(api apiRole) *roleModel {
	m := &roleModel{
		ID:   types.StringValue(api.ID),
		Name: types.StringValue(api.Name),
	}
	if len(api.Permissions) > 0 && string(api.Permissions) != "null" {
		m.PermissionsJSON = types.StringValue(string(api.Permissions))
	} else {
		m.PermissionsJSON = types.StringValue("{}")
	}
	return m
}
