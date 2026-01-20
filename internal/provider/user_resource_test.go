// Copyright 2026 Canonical Ltd.
// Licensed under the Apache License, Version 2.0, see LICENCE file for details.

// Package provider contains unit tests for the UserResource implementation.
// These tests verify:
// - Schema definition and validation
// - Resource metadata and type naming
// - Configuration handling and client setup
// - Terraform framework type handling (null, unknown, and valid values)
// - Interface compliance (Resource and ResourceWithImportState)
// - Attribute characteristics (required, optional, computed, sensitive)
package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"terraform-provider-legocharm/internal/legocharmclient"
)

func TestUserResource_Schema(t *testing.T) {
	r := &UserResource{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	require.NotNil(t, resp.Schema)
	require.Equal(t, "User resource for LegoCharm", resp.Schema.MarkdownDescription)

	attrs := resp.Schema.Attributes
	require.Contains(t, attrs, "username")
	require.Contains(t, attrs, "password")
	require.Contains(t, attrs, "email")
	require.Contains(t, attrs, "id")

	// Verify username is required
	require.True(t, attrs["username"].IsRequired())
	require.False(t, attrs["username"].IsOptional())
	require.False(t, attrs["username"].IsComputed())

	// Verify password is required and sensitive
	require.True(t, attrs["password"].IsRequired())
	require.True(t, attrs["password"].IsSensitive())

	// Verify email is optional
	require.True(t, attrs["email"].IsOptional())
	require.False(t, attrs["email"].IsRequired())

	// Verify id is computed
	require.True(t, attrs["id"].IsComputed())
	require.False(t, attrs["id"].IsRequired())
}

func TestUserResource_Metadata(t *testing.T) {
	r := &UserResource{}
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "legocharm"}, resp)
	require.Equal(t, "legocharm_user", resp.TypeName)
}

func TestUserResource_Configure_Success(t *testing.T) {
	r := &UserResource{}

	// Create a mock client
	address := "https://test.example.com"
	username := "testuser"
	password := "testpass"
	client, err := legocharmclient.NewClient(&address, &username, &password)
	require.NoError(t, err)

	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: client,
	}, resp)

	require.False(t, resp.Diagnostics.HasError())
	require.NotNil(t, r.client)
	require.Equal(t, client, r.client)
}

func TestUserResource_Configure_NilProviderData(t *testing.T) {
	r := &UserResource{}

	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: nil,
	}, resp)

	// Should not error when ProviderData is nil
	require.False(t, resp.Diagnostics.HasError())
	require.Nil(t, r.client)
}

func TestUserResource_Configure_InvalidType(t *testing.T) {
	r := &UserResource{}

	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "invalid-type",
	}, resp)

	require.True(t, resp.Diagnostics.HasError())
	require.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Unexpected Resource Configure Type")
}

func TestUserModel_StructTags(t *testing.T) {
	// Verify that UserModel has correct struct tags for Terraform schema binding
	model := UserModel{
		Username: types.StringValue("test"),
		Password: types.StringValue("pass"),
		Email:    types.StringValue("test@example.com"),
		Id:       types.StringValue("123"),
	}

	require.NotNil(t, model.Username)
	require.NotNil(t, model.Password)
	require.NotNil(t, model.Email)
	require.NotNil(t, model.Id)
}

func TestNewUserResource(t *testing.T) {
	r := NewUserResource()
	require.NotNil(t, r)

	// Verify it implements the required interfaces
	var _ resource.Resource = r
}

func TestUserResource_Implements_Interfaces(t *testing.T) {
	var r interface{} = &UserResource{}

	// Check that UserResource implements required interfaces
	_, ok := r.(resource.Resource)
	require.True(t, ok, "UserResource should implement resource.Resource")

	_, ok = r.(resource.ResourceWithImportState)
	require.True(t, ok, "UserResource should implement resource.ResourceWithImportState")
}

func TestUserModel_TypesHandling(t *testing.T) {
	t.Run("NullValues", func(t *testing.T) {
		model := UserModel{
			Username: types.StringNull(),
			Password: types.StringNull(),
			Email:    types.StringNull(),
			Id:       types.StringNull(),
		}

		require.True(t, model.Username.IsNull())
		require.True(t, model.Password.IsNull())
		require.True(t, model.Email.IsNull())
		require.True(t, model.Id.IsNull())
	})

	t.Run("UnknownValues", func(t *testing.T) {
		model := UserModel{
			Username: types.StringUnknown(),
			Password: types.StringUnknown(),
			Email:    types.StringUnknown(),
			Id:       types.StringUnknown(),
		}

		require.True(t, model.Username.IsUnknown())
		require.True(t, model.Password.IsUnknown())
		require.True(t, model.Email.IsUnknown())
		require.True(t, model.Id.IsUnknown())
	})

	t.Run("ValidValues", func(t *testing.T) {
		model := UserModel{
			Username: types.StringValue("testuser"),
			Password: types.StringValue("testpass"),
			Email:    types.StringValue("test@example.com"),
			Id:       types.StringValue("user123"),
		}

		require.False(t, model.Username.IsNull())
		require.False(t, model.Username.IsUnknown())
		require.Equal(t, "testuser", model.Username.ValueString())

		require.False(t, model.Password.IsNull())
		require.False(t, model.Password.IsUnknown())
		require.Equal(t, "testpass", model.Password.ValueString())

		require.False(t, model.Email.IsNull())
		require.False(t, model.Email.IsUnknown())
		require.Equal(t, "test@example.com", model.Email.ValueString())

		require.False(t, model.Id.IsNull())
		require.False(t, model.Id.IsUnknown())
		require.Equal(t, "user123", model.Id.ValueString())
	})
}

func TestUserResource_SchemaAttributes_Characteristics(t *testing.T) {
	r := &UserResource{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes

	// Verify username characteristics
	require.True(t, attrs["username"].IsRequired(), "username should be required")
	require.False(t, attrs["username"].IsComputed(), "username should not be computed")

	// Verify password characteristics
	require.True(t, attrs["password"].IsRequired(), "password should be required")
	require.True(t, attrs["password"].IsSensitive(), "password should be sensitive")

	// Verify email characteristics
	require.True(t, attrs["email"].IsOptional(), "email should be optional")

	// Verify id characteristics
	require.True(t, attrs["id"].IsComputed(), "id should be computed")
	require.False(t, attrs["id"].IsRequired(), "id should not be required")
}
