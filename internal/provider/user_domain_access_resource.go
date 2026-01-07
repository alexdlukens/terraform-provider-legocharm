// Copyright 2026 Canonical Ltd.
// Licensed under the Apache License, Version 2.0, see LICENCE file for details.

package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-legocharm/internal/legocharmclient"
)

var _ resource.Resource = &UserDomainAccessResource{}
var _ resource.ResourceWithImportState = &UserDomainAccessResource{}

func NewUserDomainAccessResource() resource.Resource { return &UserDomainAccessResource{} }

type UserDomainAccessResource struct {
	client *legocharmclient.Client
}

type UserDomainAccessModel struct {
	UserId      types.String `tfsdk:"user_id"`
	Domain      types.String `tfsdk:"domain"`
	AccessLevel types.String `tfsdk:"access_level"`
	Id          types.String `tfsdk:"id"`
	DatabaseID  types.Int64  `tfsdk:"database_id"`
}

func (r *UserDomainAccessResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_domain_access"
}

func (r *UserDomainAccessResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "User domain access resource for httprequest-lego-provider.",
		Attributes: map[string]schema.Attribute{
			"user_id": schema.StringAttribute{
				MarkdownDescription: "ID of user to grant domain access to",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "FQDN of the domain to grant access to",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"access_level": schema.StringAttribute{
				MarkdownDescription: "Access level. Possible values: 'domain' 'subdomain'",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the user domain access resource, in format 'user_id:domain:access_level'",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"database_id": schema.Int64Attribute{
				MarkdownDescription: "Internal database ID for the domain access permission",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *UserDomainAccessResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UserDomainAccessModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...) // Unmarshal plan
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The LegoCharm API client is not configured for this resource")
		return
	}

	// check if a domain access already exists for this user+domain
	existing, err := r.client.GetDomainAccess(data.UserId.ValueString(), data.Domain.ValueString())
	if err == nil && existing != nil {
		resp.Diagnostics.AddError("Domain Access Already Exists", "A domain access permission already exists for this user and domain combination.")
		return
	}

	createData := &legocharmclient.DomainUserPermissionCreateData{UserID: data.UserId.ValueString(), Domain: data.Domain.ValueString(), AccessLevel: data.AccessLevel.ValueString()}
	domain, err := r.client.CreateDomainAccess(*createData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create user domain access: %s", err))
		return
	}

	// Placeholder: set a fake ID for now
	data.Id = types.StringValue(data.UserId.ValueString() + ":" + data.Domain.ValueString() + ":" + data.AccessLevel.ValueString())
	data.DatabaseID = types.Int64Value(int64(domain.ID))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...) // Save state
}

func (r *UserDomainAccessResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UserDomainAccessModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...) // Unmarshal state
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The LegoCharm API client is not configured for this resource")
		return
	}

	if data.UserId.IsNull() || data.Domain.IsNull() {
		resp.Diagnostics.AddError("Invalid State", "User ID or Domain is null in state")
		return
	}

	found, err := r.client.GetDomainAccess(data.UserId.ValueString(), data.Domain.ValueString())
	// If not found, resp.State.RemoveResource(ctx)
	if err != nil {
		if errors.Is(err, legocharmclient.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read user domain access: %s", err))
		return
	}
	data.AccessLevel = types.StringValue(found.AccessLevel)
	data.DatabaseID = types.Int64Value(int64(found.ID))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...) // Save state
}

// Update implements resource updating for UserDomainAccessResource.
func (r *UserDomainAccessResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data UserDomainAccessModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...) // Unmarshal plan
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The LegoCharm API client is not configured for this resource")
		return
	}

	if data.UserId.IsNull() || data.Domain.IsNull() {
		resp.Diagnostics.AddError("Invalid State", "User ID or Domain is null in state")
		return
	}
	if data.DatabaseID.IsNull() || data.DatabaseID.ValueInt64() == 0 {
		resp.Diagnostics.AddError("Invalid State", "Database ID is null or zero in state")
		return
	}

	_, err := r.client.DeleteDomainAccess(int(data.DatabaseID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete user domain access: %s", err))
		return
	}

	// recreate with new access level
	createData := &legocharmclient.DomainUserPermissionCreateData{UserID: data.UserId.ValueString(), Domain: data.Domain.ValueString(), AccessLevel: data.AccessLevel.ValueString()}
	domain, err := r.client.CreateDomainAccess(*createData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update user domain access: %s", err))
		return
	}
	data.DatabaseID = types.Int64Value(int64(domain.ID))
	data.Id = types.StringValue(data.UserId.ValueString() + ":" + data.Domain.ValueString() + ":" + data.AccessLevel.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...) // Save state
}

// Delete implements resource deletion for UserDomainAccessResource.
func (r *UserDomainAccessResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UserDomainAccessModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...) // Unmarshal state
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The LegoCharm API client is not configured for this resource")
		return
	}

	if data.DatabaseID.IsNull() || data.DatabaseID.ValueInt64() == 0 {
		resp.Diagnostics.AddError("Invalid State", "Database ID is null or zero in state")
		return
	}

	// TODO: Call client to delete domain access resource
	_, err := r.client.DeleteDomainAccess(int(data.DatabaseID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete user domain access: %s", err))
		return
	}

	// Remove resource from state
	resp.State.RemoveResource(ctx)
}

// ImportState implements resource import for UserDomainAccessResource.
func (r *UserDomainAccessResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// id is of format "username:domain:access_level"
	parts := strings.Split(req.ID, ":")
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid Import ID", "Import ID must be in the format 'username:domain:access_level'")
		return
	}
	var data UserDomainAccessModel
	data.UserId = types.StringValue(parts[0])
	data.Domain = types.StringValue(parts[1])
	data.AccessLevel = types.StringValue(parts[2])
	data.Id = types.StringValue(req.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...) // Save state
}

func (r *UserDomainAccessResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*legocharmclient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *legocharmclient.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}
