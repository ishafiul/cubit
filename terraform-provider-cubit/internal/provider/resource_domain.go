package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &DomainResource{}
	_ resource.ResourceWithConfigure = &DomainResource{}
)

func NewDomainResource() resource.Resource {
	return &DomainResource{}
}

type DomainResource struct {
	client *CubitClient
}

type DomainResourceModel struct {
	ID            types.String `tfsdk:"id"`
	DomainName    types.String `tfsdk:"domain_name"`
	ApplicationID types.String `tfsdk:"application_id"`
	PathPrefix    types.String `tfsdk:"path_prefix"`
	TLSStatus     types.String `tfsdk:"tls_status"`
}

func (r *DomainResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *DomainResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Binds a custom hostname and Traefik routing rule to a Cubit application.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the domain binding.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_name": schema.StringAttribute{
				Description: "Fully qualified domain name (FQDN), e.g. api.example.com.",
				Required:    true,
			},
			"application_id": schema.StringAttribute{
				Description: "ID of the target Cubit application to route traffic to.",
				Required:    true,
			},
			"path_prefix": schema.StringAttribute{
				Description: "Optional URL path prefix rule (e.g. /v1).",
				Optional:    true,
			},
			"tls_status": schema.StringAttribute{
				Description: "Let's Encrypt TLS status (ready, pending, failed).",
				Computed:    true,
			},
		},
	}
}

func (r *DomainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*CubitClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *CubitClient, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *DomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := &ClientDomain{
		DomainName:    plan.DomainName.ValueString(),
		ApplicationID: plan.ApplicationID.ValueString(),
		PathPrefix:    plan.PathPrefix.ValueString(),
	}

	created, err := r.client.CreateDomain(ctx, domain)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Cubit Domain", err.Error())
		return
	}

	plan.ID = types.StringValue(created.ID)
	plan.TLSStatus = types.StringValue(created.TLSStatus)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := r.client.GetDomain(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Cubit Domain", err.Error())
		return
	}

	state.DomainName = types.StringValue(domain.DomainName)
	state.ApplicationID = types.StringValue(domain.ApplicationID)
	state.PathPrefix = types.StringValue(domain.PathPrefix)
	state.TLSStatus = types.StringValue(domain.TLSStatus)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Re-creates on update
	var plan DomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteDomain(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error Deleting Cubit Domain", err.Error())
		return
	}
}
