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
	_ resource.Resource              = &NodeResource{}
	_ resource.ResourceWithConfigure = &NodeResource{}
)

func NewNodeResource() resource.Resource {
	return &NodeResource{}
}

type NodeResource struct {
	client *CubitClient
}

type NodeResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	IPAddress types.String `tfsdk:"ip_address"`
	CPUCores  types.Int64  `tfsdk:"cpu_cores"`
	MemoryMB  types.Int64  `tfsdk:"memory_mb"`
	Status    types.String `tfsdk:"status"`
}

func (r *NodeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_node"
}

func (r *NodeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a bare-metal server node in the Cubit cluster.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the node.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Human-readable hostname or server label.",
				Required:    true,
			},
			"ip_address": schema.StringAttribute{
				Description: "IPv4 or IPv6 address of the bare-metal server.",
				Required:    true,
			},
			"cpu_cores": schema.Int64Attribute{
				Description: "Number of available CPU cores on this node.",
				Optional:    true,
				Computed:    true,
			},
			"memory_mb": schema.Int64Attribute{
				Description: "RAM capacity in megabytes.",
				Optional:    true,
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Current node status (online, draining, offline).",
				Computed:    true,
			},
		},
	}
}

func (r *NodeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NodeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NodeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	node := &ClientNode{
		Name:      plan.Name.ValueString(),
		IPAddress: plan.IPAddress.ValueString(),
		CPUCores:  int(plan.CPUCores.ValueInt64()),
		MemoryMB:  int(plan.MemoryMB.ValueInt64()),
	}

	created, err := r.client.CreateNode(ctx, node)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Cubit Node", err.Error())
		return
	}

	plan.ID = types.StringValue(created.ID)
	plan.Status = types.StringValue(created.Status)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NodeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NodeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	node, err := r.client.GetNode(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Cubit Node", err.Error())
		return
	}

	state.Name = types.StringValue(node.Name)
	state.IPAddress = types.StringValue(node.IPAddress)
	state.Status = types.StringValue(node.Status)
	state.CPUCores = types.Int64Value(int64(node.CPUCores))
	state.MemoryMB = types.Int64Value(int64(node.MemoryMB))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NodeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// For nodes, updating properties is handled via re-registration or state refresh
	var plan NodeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NodeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NodeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteNode(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error Deleting Cubit Node", err.Error())
		return
	}
}
