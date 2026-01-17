// Copyright (c) Arthur Diniz <arthurbdiniz@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/arthurbdiniz/terraform-provider-n8n/internal/pkg/n8n-client-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &workflowResource{}
	_ resource.ResourceWithConfigure   = &workflowResource{}
	_ resource.ResourceWithImportState = &workflowResource{}
	_ resource.ResourceWithModifyPlan  = &workflowResource{}
)

// NewWorkflowResource returns a new resource.
func NewWorkflowResource() resource.Resource {
	return &workflowResource{}
}

type workflowResource struct {
	client *n8n.Client
}

// workflowResourceModel maps the resource schema data.
type workflowResourceModel struct {
	ID           types.String          `tfsdk:"id"`
	Name         types.String          `tfsdk:"name"`
	Active       types.Bool            `tfsdk:"active"`
	Nodes        types.String          `tfsdk:"nodes"`
	Connections  types.String          `tfsdk:"connections"`
	Settings     *settingsResourceModel `tfsdk:"settings"`
	VersionId    types.String          `tfsdk:"version_id"`
	CreatedAt    types.String          `tfsdk:"created_at"`
	UpdatedAt    types.String          `tfsdk:"updated_at"`
}

type settingsResourceModel struct {
	SaveExecutionProgress    types.Bool   `tfsdk:"save_execution_progress"`
	SaveManualExecutions     types.Bool   `tfsdk:"save_manual_executions"`
	SaveDataErrorExecution   types.String `tfsdk:"save_data_error_execution"`
	SaveDataSuccessExecution types.String `tfsdk:"save_data_success_execution"`
	ExecutionTimeout         types.Int64  `tfsdk:"execution_timeout"`
	ErrorWorkflow            types.String `tfsdk:"error_workflow"`
	Timezone                 types.String `tfsdk:"timezone"`
	ExecutionOrder           types.String `tfsdk:"execution_order"`
}

func (r *workflowResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*n8n.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *n8n.Client, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *workflowResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow"
}

func (r *workflowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an n8n workflow.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Workflow ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the workflow.",
			},
			"active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the workflow is active.",
			},
			"nodes": schema.StringAttribute{
				Required:    true,
				Description: "JSON-encoded array of workflow nodes.",
				PlanModifiers: []planmodifier.String{
					JSONSemanticEquality(),
				},
			},
			"connections": schema.StringAttribute{
				Required:    true,
				Description: "JSON-encoded connections between nodes.",
				PlanModifiers: []planmodifier.String{
					JSONSemanticEquality(),
				},
			},
			"settings": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Workflow execution settings.",
				Attributes: map[string]schema.Attribute{
					"save_execution_progress": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(true),
						Description: "Whether to save execution progress.",
					},
					"save_manual_executions": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(true),
						Description: "Whether to save manual executions.",
					},
					"save_data_error_execution": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("all"),
						Description: "Save behavior for error executions: 'all' or 'none'.",
					},
					"save_data_success_execution": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("all"),
						Description: "Save behavior for successful executions: 'all' or 'none'.",
					},
					"execution_timeout": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(3600),
						Description: "Execution timeout in seconds (max 3600).",
					},
					"error_workflow": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString(""),
						Description: "ID of the error handler workflow.",
					},
					"timezone": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("America/New_York"),
						Description: "Timezone for the workflow.",
					},
					"execution_order": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("v1"),
						Description: "Execution order version.",
					},
				},
			},
			"version_id": schema.StringAttribute{
				Computed:    true,
				Description: "Workflow version ID. Changes on every workflow update.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the workflow was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the workflow was last updated. Changes on every workflow update.",
			},
		},
	}
}

func (r *workflowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan workflowResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse nodes from JSON
	var nodes []n8n.Node
	if err := json.Unmarshal([]byte(plan.Nodes.ValueString()), &nodes); err != nil {
		resp.Diagnostics.AddError("Invalid nodes JSON", err.Error())
		return
	}

	// Parse connections from JSON
	var connections map[string]n8n.Connection
	if err := json.Unmarshal([]byte(plan.Connections.ValueString()), &connections); err != nil {
		resp.Diagnostics.AddError("Invalid connections JSON", err.Error())
		return
	}

	// Build settings
	settings := n8n.Settings{
		SaveExecutionProgress:    true,
		SaveManualExecutions:     true,
		SaveDataErrorExecution:   "all",
		SaveDataSuccessExecution: "all",
		ExecutionTimeout:         3600,
		ErrorWorkflow:            "",
		Timezone:                 "America/New_York",
		ExecutionOrder:           "v1",
	}
	if plan.Settings != nil {
		settings.SaveExecutionProgress = plan.Settings.SaveExecutionProgress.ValueBool()
		settings.SaveManualExecutions = plan.Settings.SaveManualExecutions.ValueBool()
		settings.SaveDataErrorExecution = plan.Settings.SaveDataErrorExecution.ValueString()
		settings.SaveDataSuccessExecution = plan.Settings.SaveDataSuccessExecution.ValueString()
		settings.ExecutionTimeout = int(plan.Settings.ExecutionTimeout.ValueInt64())
		settings.ErrorWorkflow = plan.Settings.ErrorWorkflow.ValueString()
		settings.Timezone = plan.Settings.Timezone.ValueString()
		settings.ExecutionOrder = plan.Settings.ExecutionOrder.ValueString()
	}

	// Create workflow
	createReq := &n8n.CreateWorkflowRequest{
		Name:        plan.Name.ValueString(),
		Nodes:       nodes,
		Connections: connections,
		Settings:    settings,
	}

	tflog.Debug(ctx, "Creating workflow", map[string]any{"name": plan.Name.ValueString()})

	workflow, err := r.client.CreateWorkflow(createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating workflow", err.Error())
		return
	}

	// Activate if requested
	if plan.Active.ValueBool() {
		workflow, err = r.client.ActivateWorkflow(workflow.ID)
		if err != nil {
			resp.Diagnostics.AddError("Error activating workflow", err.Error())
			return
		}
	}

	// Map response to state
	plan.ID = types.StringValue(workflow.ID)
	plan.VersionId = types.StringValue(workflow.VersionId)
	plan.CreatedAt = types.StringValue(workflow.CreatedAt)
	plan.UpdatedAt = types.StringValue(workflow.UpdatedAt)
	plan.Active = types.BoolValue(workflow.Active)
	plan.Settings = &settingsResourceModel{
		SaveExecutionProgress:    types.BoolValue(workflow.Settings.SaveExecutionProgress),
		SaveManualExecutions:     types.BoolValue(workflow.Settings.SaveManualExecutions),
		SaveDataErrorExecution:   types.StringValue(workflow.Settings.SaveDataErrorExecution),
		SaveDataSuccessExecution: types.StringValue(workflow.Settings.SaveDataSuccessExecution),
		ExecutionTimeout:         types.Int64Value(int64(workflow.Settings.ExecutionTimeout)),
		ErrorWorkflow:            types.StringValue(workflow.Settings.ErrorWorkflow),
		Timezone:                 types.StringValue(workflow.Settings.Timezone),
		ExecutionOrder:           types.StringValue(workflow.Settings.ExecutionOrder),
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *workflowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state workflowResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	workflow, err := r.client.GetWorkflow(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading workflow", err.Error())
		return
	}

	// Convert nodes back to JSON
	nodesJSON, err := json.Marshal(workflow.Nodes)
	if err != nil {
		resp.Diagnostics.AddError("Error serializing nodes", err.Error())
		return
	}

	// Convert connections back to JSON
	connectionsJSON, err := json.Marshal(workflow.Connections)
	if err != nil {
		resp.Diagnostics.AddError("Error serializing connections", err.Error())
		return
	}

	state.ID = types.StringValue(workflow.ID)
	state.Name = types.StringValue(workflow.Name)
	state.Active = types.BoolValue(workflow.Active)
	state.Nodes = types.StringValue(string(nodesJSON))
	state.Connections = types.StringValue(string(connectionsJSON))
	state.VersionId = types.StringValue(workflow.VersionId)
	state.CreatedAt = types.StringValue(workflow.CreatedAt)
	state.UpdatedAt = types.StringValue(workflow.UpdatedAt)
	state.Settings = &settingsResourceModel{
		SaveExecutionProgress:    types.BoolValue(workflow.Settings.SaveExecutionProgress),
		SaveManualExecutions:     types.BoolValue(workflow.Settings.SaveManualExecutions),
		SaveDataErrorExecution:   types.StringValue(workflow.Settings.SaveDataErrorExecution),
		SaveDataSuccessExecution: types.StringValue(workflow.Settings.SaveDataSuccessExecution),
		ExecutionTimeout:         types.Int64Value(int64(workflow.Settings.ExecutionTimeout)),
		ErrorWorkflow:            types.StringValue(workflow.Settings.ErrorWorkflow),
		Timezone:                 types.StringValue(workflow.Settings.Timezone),
		ExecutionOrder:           types.StringValue(workflow.Settings.ExecutionOrder),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *workflowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan workflowResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state workflowResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse nodes from JSON
	var nodes []n8n.Node
	if err := json.Unmarshal([]byte(plan.Nodes.ValueString()), &nodes); err != nil {
		resp.Diagnostics.AddError("Invalid nodes JSON", err.Error())
		return
	}

	// Parse connections from JSON
	var connections map[string]n8n.Connection
	if err := json.Unmarshal([]byte(plan.Connections.ValueString()), &connections); err != nil {
		resp.Diagnostics.AddError("Invalid connections JSON", err.Error())
		return
	}

	// Build settings
	settings := n8n.Settings{}
	if plan.Settings != nil {
		settings.SaveExecutionProgress = plan.Settings.SaveExecutionProgress.ValueBool()
		settings.SaveManualExecutions = plan.Settings.SaveManualExecutions.ValueBool()
		settings.SaveDataErrorExecution = plan.Settings.SaveDataErrorExecution.ValueString()
		settings.SaveDataSuccessExecution = plan.Settings.SaveDataSuccessExecution.ValueString()
		settings.ExecutionTimeout = int(plan.Settings.ExecutionTimeout.ValueInt64())
		settings.ErrorWorkflow = plan.Settings.ErrorWorkflow.ValueString()
		settings.Timezone = plan.Settings.Timezone.ValueString()
		settings.ExecutionOrder = plan.Settings.ExecutionOrder.ValueString()
	}

	updateReq := &n8n.UpdateWorkflowRequest{
		Name:        plan.Name.ValueString(),
		Nodes:       nodes,
		Connections: connections,
		Settings:    settings,
	}

	tflog.Debug(ctx, "Updating workflow", map[string]any{"id": state.ID.ValueString()})

	workflow, err := r.client.UpdateWorkflow(state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating workflow", err.Error())
		return
	}

	// Handle activation state change
	if plan.Active.ValueBool() != state.Active.ValueBool() {
		if plan.Active.ValueBool() {
			workflow, err = r.client.ActivateWorkflow(workflow.ID)
		} else {
			workflow, err = r.client.DeactivateWorkflow(workflow.ID)
		}
		if err != nil {
			resp.Diagnostics.AddError("Error changing workflow activation state", err.Error())
			return
		}
	}

	// Map response to state
	plan.ID = types.StringValue(workflow.ID)
	plan.VersionId = types.StringValue(workflow.VersionId)
	plan.CreatedAt = types.StringValue(workflow.CreatedAt)
	plan.UpdatedAt = types.StringValue(workflow.UpdatedAt)
	plan.Active = types.BoolValue(workflow.Active)
	plan.Settings = &settingsResourceModel{
		SaveExecutionProgress:    types.BoolValue(workflow.Settings.SaveExecutionProgress),
		SaveManualExecutions:     types.BoolValue(workflow.Settings.SaveManualExecutions),
		SaveDataErrorExecution:   types.StringValue(workflow.Settings.SaveDataErrorExecution),
		SaveDataSuccessExecution: types.StringValue(workflow.Settings.SaveDataSuccessExecution),
		ExecutionTimeout:         types.Int64Value(int64(workflow.Settings.ExecutionTimeout)),
		ErrorWorkflow:            types.StringValue(workflow.Settings.ErrorWorkflow),
		Timezone:                 types.StringValue(workflow.Settings.Timezone),
		ExecutionOrder:           types.StringValue(workflow.Settings.ExecutionOrder),
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *workflowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state workflowResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting workflow", map[string]any{"id": state.ID.ValueString()})

	_, err := r.client.DeleteWorkflow(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting workflow", err.Error())
		return
	}
}

func (r *workflowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ModifyPlan implements resource-level plan modification to prevent unnecessary updates.
// When only computed fields (updated_at, version_id) differ, we preserve state values
// to avoid triggering an update that would only change timestamps.
func (r *workflowResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Skip during create (no state) or destroy (no plan)
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var plan, state workflowResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if any content fields actually changed
	contentChanged := false

	// Compare name
	if !plan.Name.Equal(state.Name) {
		contentChanged = true
	}

	// Compare active
	if !plan.Active.Equal(state.Active) {
		contentChanged = true
	}

	// Compare nodes (using semantic equality)
	if !plan.Nodes.IsUnknown() && !state.Nodes.IsUnknown() {
		if !jsonSemanticEqual(plan.Nodes.ValueString(), state.Nodes.ValueString()) {
			contentChanged = true
		}
	}

	// Compare connections (using semantic equality)
	if !plan.Connections.IsUnknown() && !state.Connections.IsUnknown() {
		if !jsonSemanticEqual(plan.Connections.ValueString(), state.Connections.ValueString()) {
			contentChanged = true
		}
	}

	// Compare settings
	if plan.Settings != nil && state.Settings != nil {
		if !plan.Settings.SaveExecutionProgress.Equal(state.Settings.SaveExecutionProgress) ||
			!plan.Settings.SaveManualExecutions.Equal(state.Settings.SaveManualExecutions) ||
			!plan.Settings.SaveDataErrorExecution.Equal(state.Settings.SaveDataErrorExecution) ||
			!plan.Settings.SaveDataSuccessExecution.Equal(state.Settings.SaveDataSuccessExecution) ||
			!plan.Settings.ExecutionTimeout.Equal(state.Settings.ExecutionTimeout) ||
			!plan.Settings.ErrorWorkflow.Equal(state.Settings.ErrorWorkflow) ||
			!plan.Settings.Timezone.Equal(state.Settings.Timezone) ||
			!plan.Settings.ExecutionOrder.Equal(state.Settings.ExecutionOrder) {
			contentChanged = true
		}
	} else if (plan.Settings == nil) != (state.Settings == nil) {
		contentChanged = true
	}

	tflog.Debug(ctx, "ModifyPlan content comparison", map[string]any{
		"contentChanged": contentChanged,
		"workflowId":     state.ID.ValueString(),
	})

	// If no content changed, preserve computed field values from state
	// This prevents unnecessary updates that would only change timestamps
	if !contentChanged {
		// Preserve version_id from state
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("version_id"), state.VersionId)...)
		// Preserve updated_at from state
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("updated_at"), state.UpdatedAt)...)
		// Also preserve nodes and connections with state values to ensure no diff
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("nodes"), state.Nodes)...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("connections"), state.Connections)...)

		tflog.Debug(ctx, "No content changes detected, preserving state values for computed fields", map[string]any{
			"workflowId": state.ID.ValueString(),
		})
	}
}
