// Copyright 2026 Canonical Ltd.
// Licensed under the Apache License, Version 2.0, see LICENCE file for details.

package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-legocharm/internal/legocharmclient"
)

var _ resource.Resource = &UserResource{}
var _ resource.ResourceWithImportState = &UserResource{}

// NewUserResource creates a new user resource.
func NewUserResource() resource.Resource { return &UserResource{} }

// UserResource is the resource implementation for LegoCharm users.
// It manages the lifecycle of user resources in the LegoCharm API.
type UserResource struct {
	client *legocharmclient.Client
}

// UserModel maps Terraform schema to Go types for user resources.
type UserModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Email    types.String `tfsdk:"email"`
	Id       types.String `tfsdk:"id"`
}

func (r *UserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "User resource for LegoCharm",
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				MarkdownDescription: "Username",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Email address",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *UserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If client is not configured, return an error diagnostic.
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The LegoCharm API client is not configured for this resource")
		return
	}

	// Check for conflict: ensure username does not already exist
	if existingUser, err := r.client.GetUserByUsername(data.Username.ValueString()); err == nil {
		existingUserId := legocharmclient.LastPathSegment(existingUser.Url)
		resp.Diagnostics.AddError("User Exists", fmt.Sprintf("A user with username '%s' already exists (id=%s).", data.Username.ValueString(), existingUserId))
		return
	} else if err != legocharmclient.ErrNotFound {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to check for existing user: %s", err))
		return
	}

	create := legocharmclient.UserCreateData{
		Username: data.Username.ValueString(),
		Password: data.Password.ValueString(),
		Email:    data.Email.ValueString(),
		Groups:   []string{},
	}

	_, err := r.client.CreateUser(create)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create user, got error: %s", err))
		return
	}

	time.Sleep(500 * time.Millisecond)

	// Fetch created user to populate state
	user, err := r.client.GetUserByUsername(data.Username.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("User created but failed to read back: %s", err))
		return
	}

	data.Id = types.StringValue(legocharmclient.LastPathSegment(user.Url))
	data.Email = types.StringValue(user.Email)
	data.Password = types.StringValue(data.Password.ValueString())

	// Write logs
	tflog.Trace(ctx, "created user")

	// Save state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The LegoCharm API client is not configured for this resource")
		return
	}

	// Attempt to look up by id (URL) first, then by username
	var user *legocharmclient.UserData
	var err error
	if !data.Id.IsNull() && data.Id.ValueString() != "" {
		// Try delete or fetch by URL: the API may not support fetch by URL, so
		// fall back to username lookup.
		user, err = r.client.GetUserByUsername(data.Username.ValueString())
	} else {
		user, err = r.client.GetUserByUsername(data.Username.ValueString())
	}
	if err != nil {
		if err == legocharmclient.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read user: %s", err))
		return
	}

	data.Email = types.StringValue(user.Email)
	data.Id = types.StringValue(legocharmclient.LastPathSegment(user.Url))

	// ensure the password is valid
	valid, err := r.client.HasValidUserPassword(data.Username.ValueString(), data.Password.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to validate user password: %s", err))
		return
	}
	if !valid {
		resp.Diagnostics.AddWarning("Invalid Password", "The stored password is no longer valid")
		// require replacement on next apply
		data.Password = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Refresh state from the API. Password is generated at creation and retained
	// in state; it is not user-supplied and will be preserved across refreshes.
	var plan UserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The LegoCharm API client is not configured for this resource")
		return
	}

	user, err := r.client.GetUserByUsername(plan.Username.ValueString())
	if err != nil {
		if err == legocharmclient.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read user: %s", err))
		return
	}

	plan.Email = types.StringValue(user.Email)
	plan.Id = types.StringValue(legocharmclient.LastPathSegment(user.Url))

	// Preserve generated password from prior state (if present)
	var state UserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if !state.Password.IsNull() && !state.Password.IsUnknown() {
		plan.Password = state.Password
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The LegoCharm API client is not configured for this resource")
		return
	}

	// Use ID (URL) if set, otherwise fetch user to get a URL and delete by that.
	if !data.Id.IsNull() && data.Id.ValueString() != "" {
		_, err := r.client.DeleteUserById(data.Id.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete user: %s", err))
			return
		}
		return
	}

	user, err := r.client.GetUserByUsername(data.Username.ValueString())
	if err != nil {
		if err == legocharmclient.ErrNotFound {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to locate user for deletion: %s", err))
		return
	}

	_, err = r.client.DeleteUserById(legocharmclient.LastPathSegment(user.Url))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete user: %s", err))
		return
	}
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// id is of format "username:password"
	parts := strings.Split(req.ID, ":")

	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid Import ID", "Import ID must be in the format 'username:password'")
		return
	}

	username := parts[0]
	password := parts[1]

	var data UserModel
	data.Username = types.StringValue(username)
	data.Password = types.StringValue(password)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
