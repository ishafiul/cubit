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
	_ resource.Resource              = &ApplicationResource{}
	_ resource.ResourceWithConfigure = &ApplicationResource{}
)

func NewApplicationResource() resource.Resource {
	return &ApplicationResource{}
}

type ApplicationResource struct {
	client *CubitClient
}

type ApplicationResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	GitRepo            types.String `tfsdk:"git_repo"`
	Status             types.String `tfsdk:"status"`
	ActiveDeploymentID types.String `tfsdk:"active_deployment_id"`
}

func (r *ApplicationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (r *ApplicationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Cloudflare Worker application and its runtime configuration in Cubit.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the application.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Human-readable name of the application.",
				Required:    true,
			},
			"git_repo": schema.StringAttribute{
				Description: "Git repository containing the Cloudflare Worker source code.",
				Required:    true,
			},
			"status": schema.StringAttribute{
				Description: "Current status of the application (e.g. active, building).",
				Computed:    true,
			},
			"active_deployment_id": schema.StringAttribute{
				Description: "ID of the currently active deployment.",
				Computed:    true,
			},
		},
	}
}

func (r *ApplicationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app := &ClientApplication{
		Name:    plan.Name.ValueString(),
		GitRepo: plan.GitRepo.ValueString(),
	}

	created, err := r.client.CreateApplication(ctx, app)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Cubit Application", err.Error())
		return
	}

	plan.ID = types.StringValue(created.ID)
	plan.Status = types.StringValue(created.Status)
	plan.ActiveDeploymentID = types.StringValue(created.ActiveDeployment)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := r.client.GetApplication(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Cubit Application", err.Error())
		return
	}

	state.Name = types.StringValue(app.Name)
	state.GitRepo = types.StringValue(app.GitRepo)
	state.Status = types.StringValue(app.Status)
	state.ActiveDeploymentID = types.StringValue(app.ActiveDeployment)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app := &ClientApplication{
		Name:    plan.Name.ValueString(),
		GitRepo: plan.GitRepo.ValueString(),
	}

	updated, err := r.client.UpdateApplication(ctx, plan.ID.ValueString(), app)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Cubit Application", err.Error())
		return
	}

	plan.Status = types.StringValue(updated.Status)
	plan.ActiveDeploymentID = types.StringValue(updated.ActiveDeployment)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteApplication(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error Deleting Cubit Application", err.Error())
		return
	}
}
