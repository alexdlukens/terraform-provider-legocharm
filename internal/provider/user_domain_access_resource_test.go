// Copyright 2026 Canonical Ltd.
// Licensed under the Apache License, Version 2.0, see LICENCE file for details.

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/require"
)

func TestUserDomainAccessResource_Schema(t *testing.T) {
	r := &UserDomainAccessResource{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	require.NotNil(t, resp.Schema)
	attrs := resp.Schema.Attributes
	require.Contains(t, attrs, "user_id")
	require.Contains(t, attrs, "domain")
	require.Contains(t, attrs, "access_level")
	require.Contains(t, attrs, "id")
}

func TestUserDomainAccessResource_Metadata(t *testing.T) {
	r := &UserDomainAccessResource{}
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "legocharm"}, resp)
	require.Equal(t, "legocharm_domain_access", resp.TypeName)
}
