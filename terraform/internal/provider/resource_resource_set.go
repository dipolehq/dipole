/*
resource_resource_set.go — Terraform resource for dipole_resource_set.

Manages Resource Set entities via the REST API. Resource Sets provide
multi-tenancy within a tenant — segment workloads by team, environment, or region.
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const resourceSetResourceType = "resource-sets"

type resourceSetResource struct {
	client *client.Client
}

type resourceSetModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Roles       types.Set    `tfsdk:"roles"`
}

func NewResourceSetResource() resource.Resource {
	return &resourceSetResource{}
}

func (r *resourceSetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_set"
}

func (r *resourceSetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Dipole Resource Set.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID identifier of the Resource Set.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name of the Resource Set.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the Resource Set.",
			},
			"roles": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Role IDs associated with this Resource Set (read-only, managed from dipole_role resource).",
			},
		},
	}
}

func (r *resourceSetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *resourceSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceSetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]string{
		"name":        plan.Name.ValueString(),
		"description": plan.Description.ValueString(),
	}

	respBody, err := r.client.Create(ctx, resourceSetResourceType, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Resource Set", err.Error())
		return
	}

	var result apiResourceSet
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, resourceSetFromAPI(result))...)
}

func (r *resourceSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceSetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	respBody, err := r.client.Read(ctx, resourceSetResourceType, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Resource Set", err.Error())
		return
	}

	var result apiResourceSet
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, resourceSetFromAPI(result))...)
}

func (r *resourceSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceSetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]string{
		"name":        plan.Name.ValueString(),
		"description": plan.Description.ValueString(),
	}

	respBody, err := r.client.Update(ctx, resourceSetResourceType, plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Resource Set", err.Error())
		return
	}

	var result apiResourceSet
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, resourceSetFromAPI(result))...)
}

func (r *resourceSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceSetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, resourceSetResourceType, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting Resource Set", err.Error())
	}
}

func (r *resourceSetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type apiResourceSet struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Roles       []string `json:"roles"`
}

func resourceSetFromAPI(api apiResourceSet) *resourceSetModel {
	m := &resourceSetModel{
		ID:   types.StringValue(api.ID),
		Name: types.StringValue(api.Name),
	}
	if api.Description != "" {
		m.Description = types.StringValue(api.Description)
	} else {
		m.Description = types.StringNull()
	}
	if len(api.Roles) > 0 {
		elements := make([]attr.Value, 0, len(api.Roles))
		for _, id := range api.Roles {
			elements = append(elements, types.StringValue(id))
		}
		m.Roles, _ = types.SetValue(types.StringType, elements)
	} else {
		m.Roles, _ = types.SetValue(types.StringType, []attr.Value{})
	}
	return m
}
