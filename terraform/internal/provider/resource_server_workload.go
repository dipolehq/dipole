/*
resource_server_workload.go — Terraform resource for dipole_server_workload.

Manages Server Workload entities via the REST API. Server Workloads represent
the target services (APIs, databases, SaaS) that Client Workloads access.
Identified by host + port + protocol + authentication method.
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const serverWorkloadResourceType = "server-workloads"

type serverWorkloadResource struct {
	client      *client.Client
	defaultTags map[string]string
}

type serverWorkloadModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	IsActive          types.Bool   `tfsdk:"is_active"`
	Host              types.String `tfsdk:"host"`
	Port              types.Int64  `tfsdk:"port"`
	AppProtocol       types.String `tfsdk:"app_protocol"`
	TransportProtocol types.String `tfsdk:"transport_protocol"`
	TLS               types.Bool   `tfsdk:"tls"`
	TLSVerification   types.String `tfsdk:"tls_verification"`
	RequestedPort     types.Int64  `tfsdk:"requested_port"`
	RequestedTLS      types.Bool   `tfsdk:"requested_tls"`
	AuthMethod        types.String `tfsdk:"auth_method"`
	AuthScheme        types.String `tfsdk:"auth_scheme"`
	AuthConfig        types.Map    `tfsdk:"auth_config"`
	HTTPHeaders       types.Map    `tfsdk:"http_headers"`
	URLPath           types.String `tfsdk:"url_path"`
	ResourceSetID     types.String `tfsdk:"resource_set_id"`
	Tags              types.Map    `tfsdk:"tags"`
	TagsAll           types.Map    `tfsdk:"tags_all"`
}

func NewServerWorkloadResource() resource.Resource {
	return &serverWorkloadResource{}
}

func (r *serverWorkloadResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_workload"
}

func (r *serverWorkloadResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Dipole Server Workload.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID identifier of the Server Workload.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name of the Server Workload.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the Server Workload.",
			},
			"is_active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Active status of the Server Workload.",
			},
			"host": schema.StringAttribute{
				Required:    true,
				Description: "Hostname of the service endpoint. Supports wildcards (e.g. *.amazonaws.com).",
			},
			"port": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(443),
				Description: "Port of the service endpoint (default: 443).",
			},
			"app_protocol": schema.StringAttribute{
				Required:    true,
				Description: "Application protocol (HTTP, PostgreSQL, MySQL, Redis, Snowflake, OracleDatabase, Redshift, MCP, TCP).",
			},
			"transport_protocol": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Transport protocol (default: TCP).",
			},
			"tls": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether TLS is enabled on the service endpoint.",
			},
			"tls_verification": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "TLS verification mode: full (default) or none.",
			},
			"requested_port": schema.Int64Attribute{
				Optional:    true,
				Description: "Forward-to port (actual server port if different from client-facing port).",
			},
			"requested_tls": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to use TLS on the proxy-to-server connection.",
			},
			"auth_method": schema.StringAttribute{
				Optional:    true,
				Description: "Authentication method (http_bearer, http_basic, http_header, api_key_header, api_key_query, password_auth, sigv4, snowflake_jwt, mtls_x509).",
			},
			"auth_scheme": schema.StringAttribute{
				Optional:    true,
				Description: "Authentication scheme (Bearer, Basic, Header, etc).",
			},
			"auth_config": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Additional authentication configuration (e.g. header_name for custom headers).",
			},
			"http_headers": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Static HTTP headers to add to every request to this server workload.",
			},
			"url_path": schema.StringAttribute{
				Optional:    true,
				Description: "URL path of the service endpoint.",
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

func (r *serverWorkloadResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *serverWorkloadResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serverWorkloadModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.modelToAPI(ctx, plan)
	respBody, err := r.client.Create(ctx, serverWorkloadResourceType, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Server Workload", err.Error())
		return
	}

	var result apiServerWorkload
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.apiToModel(ctx, result))...)
}

func (r *serverWorkloadResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serverWorkloadModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	respBody, err := r.client.Read(ctx, serverWorkloadResourceType, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Server Workload", err.Error())
		return
	}

	var result apiServerWorkload
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.apiToModel(ctx, result))...)
}

func (r *serverWorkloadResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serverWorkloadModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.modelToAPI(ctx, plan)
	respBody, err := r.client.Update(ctx, serverWorkloadResourceType, plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Server Workload", err.Error())
		return
	}

	var result apiServerWorkload
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.apiToModel(ctx, result))...)
}

func (r *serverWorkloadResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serverWorkloadModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, serverWorkloadResourceType, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting Server Workload", err.Error())
	}
}

func (r *serverWorkloadResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type apiServerWorkload struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	IsActive          bool              `json:"isActive"`
	Host              string            `json:"host"`
	Port              int               `json:"port"`
	AppProtocol       string            `json:"appProtocol"`
	TransportProtocol string            `json:"transportProtocol"`
	TLS               bool              `json:"tls"`
	TLSVerification   string            `json:"tlsVerification"`
	RequestedPort     int               `json:"requestedPort"`
	RequestedTLS      bool              `json:"requestedTls"`
	AuthMethod        string            `json:"authMethod"`
	AuthScheme        string            `json:"authScheme"`
	AuthConfig        map[string]string `json:"authConfig"`
	HTTPHeaders       map[string]string `json:"httpHeaders"`
	URLPath           string            `json:"urlPath"`
	Tags              []apiTag          `json:"tags"`
	ResourceSetID     *string           `json:"resourceSetId"`
}

func (r *serverWorkloadResource) modelToAPI(ctx context.Context, m serverWorkloadModel) apiServerWorkload {
	transportProtocol := "TCP"
	if !m.TransportProtocol.IsNull() && !m.TransportProtocol.IsUnknown() && m.TransportProtocol.ValueString() != "" {
		transportProtocol = m.TransportProtocol.ValueString()
	}

	tlsVerification := "full"
	if !m.TLSVerification.IsNull() && !m.TLSVerification.IsUnknown() && m.TLSVerification.ValueString() != "" {
		tlsVerification = m.TLSVerification.ValueString()
	}

	api := apiServerWorkload{
		Name:              m.Name.ValueString(),
		Description:       m.Description.ValueString(),
		IsActive:          m.IsActive.ValueBool(),
		Host:              m.Host.ValueString(),
		Port:              int(m.Port.ValueInt64()),
		AppProtocol:       m.AppProtocol.ValueString(),
		TransportProtocol: transportProtocol,
		TLS:               m.TLS.ValueBool(),
		TLSVerification:   tlsVerification,
		RequestedPort:     int(m.RequestedPort.ValueInt64()),
		RequestedTLS:      m.RequestedTLS.ValueBool(),
		AuthMethod:        m.AuthMethod.ValueString(),
		AuthScheme:        m.AuthScheme.ValueString(),
		URLPath:           m.URLPath.ValueString(),
	}

	if !m.AuthConfig.IsNull() && !m.AuthConfig.IsUnknown() {
		api.AuthConfig = make(map[string]string)
		m.AuthConfig.ElementsAs(ctx, &api.AuthConfig, false)
	}

	if !m.HTTPHeaders.IsNull() && !m.HTTPHeaders.IsUnknown() {
		api.HTTPHeaders = make(map[string]string)
		m.HTTPHeaders.ElementsAs(ctx, &api.HTTPHeaders, false)
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

func (r *serverWorkloadResource) apiToModel(ctx context.Context, api apiServerWorkload) *serverWorkloadModel {
	m := &serverWorkloadModel{
		ID:           types.StringValue(api.ID),
		Name:         types.StringValue(api.Name),
		IsActive:     types.BoolValue(api.IsActive),
		Host:         types.StringValue(api.Host),
		Port:         types.Int64Value(int64(api.Port)),
		AppProtocol:  types.StringValue(api.AppProtocol),
		TLS:          types.BoolValue(api.TLS),
		RequestedTLS: types.BoolValue(api.RequestedTLS),
	}

	if api.Description != "" {
		m.Description = types.StringValue(api.Description)
	} else {
		m.Description = types.StringNull()
	}

	if api.TransportProtocol != "" {
		m.TransportProtocol = types.StringValue(api.TransportProtocol)
	} else {
		m.TransportProtocol = types.StringValue("TCP")
	}

	if api.TLSVerification != "" {
		m.TLSVerification = types.StringValue(api.TLSVerification)
	} else {
		m.TLSVerification = types.StringValue("full")
	}

	if api.RequestedPort > 0 {
		m.RequestedPort = types.Int64Value(int64(api.RequestedPort))
	} else {
		m.RequestedPort = types.Int64Null()
	}

	if api.AuthMethod != "" {
		m.AuthMethod = types.StringValue(api.AuthMethod)
	} else {
		m.AuthMethod = types.StringNull()
	}

	if api.AuthScheme != "" {
		m.AuthScheme = types.StringValue(api.AuthScheme)
	} else {
		m.AuthScheme = types.StringNull()
	}

	if len(api.AuthConfig) > 0 {
		elements := make(map[string]attr.Value, len(api.AuthConfig))
		for k, v := range api.AuthConfig {
			elements[k] = types.StringValue(v)
		}
		m.AuthConfig, _ = types.MapValue(types.StringType, elements)
	} else {
		m.AuthConfig = types.MapNull(types.StringType)
	}

	if len(api.HTTPHeaders) > 0 {
		elements := make(map[string]attr.Value, len(api.HTTPHeaders))
		for k, v := range api.HTTPHeaders {
			elements[k] = types.StringValue(v)
		}
		m.HTTPHeaders, _ = types.MapValue(types.StringType, elements)
	} else {
		m.HTTPHeaders = types.MapNull(types.StringType)
	}

	if api.URLPath != "" {
		m.URLPath = types.StringValue(api.URLPath)
	} else {
		m.URLPath = types.StringNull()
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

func (r *serverWorkloadResource) mergeTags(ctx context.Context, userTags types.Map) map[string]string {
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
