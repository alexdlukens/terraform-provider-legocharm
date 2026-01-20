// Copyright 2026 Canonical Ltd.
// Licensed under the Apache License, Version 2.0, see LICENCE file for details.

package legocharmclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// LastPathSegment returns the last non-empty segment of a URL path.
func LastPathSegment(u string) string {
	u = strings.TrimSuffix(u, "/")
	parts := strings.Split(u, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// Client is a lightweight HTTP client for the LegoCharm API. It stores the
// base URL and credentials and exposes helpers to build and dispatch requests.
// All methods preserve the original API interactions while following Go conventions.
type Client struct {
	BaseURL    string
	Username   string
	Password   string
	HTTPClient *http.Client
}

// NewClient constructs a new LegoCharm API client.
// The provider code passes pointers to strings, so this function accepts
// pointer arguments and validates them.
func NewClient(address, username, password *string) (*Client, error) {
	if address == nil || *address == "" {
		return nil, errors.New("address is required")
	}
	if username == nil || *username == "" {
		return nil, errors.New("username is required")
	}
	if password == nil || *password == "" {
		return nil, errors.New("password is required")
	}

	u := *address
	// If no scheme was provided, default to https.
	parsed, err := url.Parse(u)
	if err != nil || !parsed.IsAbs() {
		if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
			u = "https://" + u
			parsed, err = url.Parse(u)
		}
		if err != nil || !parsed.IsAbs() {
			return nil, fmt.Errorf("invalid address %q: %w", *address, err)
		}
	}

	// Determine HTTP client timeout from environment variable LEGOCHARM_API_TIMEOUT.
	// Accepts either a duration string (e.g. "30s") or an integer number of seconds (e.g. "30").
	// Defaults to 120 seconds when unset.
	timeout := 120 * time.Second
	if v := os.Getenv("LEGOCHARM_API_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			timeout = d
		} else if s, err2 := strconv.Atoi(v); err2 == nil {
			timeout = time.Duration(s) * time.Second
		} else {
			return nil, fmt.Errorf("invalid LEGOCHARM_API_TIMEOUT %q: %w", v, err)
		}
	}

	return &Client{
		BaseURL:    strings.TrimRight(u, "/"),
		Username:   *username,
		Password:   *password,
		HTTPClient: &http.Client{Timeout: timeout},
	}, nil
}

// NewRequest creates an HTTP request for the LegoCharm API, setting basic
// authentication and reasonable default headers.
func (c *Client) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	if c == nil {
		return nil, errors.New("client is nil")
	}

	rel := strings.TrimLeft(path, "/")
	full := c.BaseURL + "/" + rel
	req, err := http.NewRequest(method, full, body)
	if err != nil {
		return nil, err
	}

	// Use basic auth for now.
	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("User-Agent", "terraform-provider-legocharm")
	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// Do sends the HTTP request using the client's underlying HTTP client.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if c == nil {
		return nil, errors.New("client is nil")
	}
	return c.HTTPClient.Do(req)
}

// ErrNotFound is returned when an API lookup yields no results.
var ErrNotFound = errors.New("not found")

// GetUserById queries the API for a user by user ID and returns the user data.
// Returns ErrNotFound if the user does not exist.
func (c *Client) GetUserById(userId string) (*UserData, error) {

	req, err := c.NewRequest("GET", "/api/v1/users/"+url.PathEscape(userId)+"/", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var userData UserData
	if err := json.Unmarshal(body, &userData); err != nil {
		return nil, fmt.Errorf("failed to parse user response: %w (body: %s)", err, string(body))
	}

	return &userData, nil
}

// GetUserByUsername queries the API for a user by username and returns the
// first matching user record or ErrNotFound if none exist.
func (c *Client) GetUserByUsername(username string) (*UserData, error) {
	req, err := c.NewRequest("GET", "/api/v1/users/?username="+url.QueryEscape(username), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Try to decode an array response first.
	var list []UserData
	if err := json.Unmarshal(body, &list); err == nil {
		if len(list) == 0 {
			return nil, ErrNotFound
		}
		return &list[0], nil
	}

	// Fallback to single-object decode.
	var single UserData
	if err := json.Unmarshal(body, &single); err == nil {
		return &single, nil
	}

	return nil, fmt.Errorf("failed to parse user response: %s", string(body))
}

// CreateUser creates a new user by POSTing the provided user object
// as JSON and returns the created user.
func (c *Client) CreateUser(user UserCreateData) (*UserData, error) {
	b, err := json.Marshal(user)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user data: %w", err)
	}

	req, err := c.NewRequest("POST", "/api/v1/users/", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// if we got a non-2xx response, return an error
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to create user: status %d, body: %s", resp.StatusCode, string(body))
	}

	var userData UserData
	if err := json.Unmarshal(body, &userData); err != nil {
		return nil, fmt.Errorf("failed to parse user response: %w (body: %s)", err, string(body))
	}

	return &userData, nil
}

// DeleteUserById deletes a user by their ID.
// Returns the HTTP response from the API.
func (c *Client) DeleteUserById(id string) (*http.Response, error) {
	req, err := c.NewRequest("DELETE", "/api/v1/users/"+url.PathEscape(id)+"/", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	return resp, nil
}

// HasValidUserPassword verifies if a username and password combination is valid
// by attempting to authenticate with the API using those credentials.
func (c *Client) HasValidUserPassword(username, password string) (bool, error) {
	// create a new client with the user credentials
	userClient, err := NewClient(&c.BaseURL, &username, &password)
	if err != nil {
		return false, fmt.Errorf("failed to create client: %w", err)
	}
	req, err := userClient.NewRequest("GET", "/api/v1/users/?username="+url.QueryEscape(username), nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := userClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// if result is 401 Unauthorized, the password is incorrect (return false)
	if resp.StatusCode == http.StatusUnauthorized {
		return false, nil
	}
	// if result is 403 Forbidden, the password is correct (return true)
	if resp.StatusCode == http.StatusForbidden {
		return true, nil
	}

	// For other status codes, return an error
	return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)

}

// GetDomainAccess retrieves domain access permissions for a user and domain.
// Returns ErrNotFound if no matching permission exists.
func (c *Client) GetDomainAccess(userId, domain string) (*DomainUserPermissionData, error) {
	// get user to fetch username
	user, err := c.GetUserById(userId)
	if err != nil {
		return nil, fmt.Errorf("failed to get user data: %w", err)
	}

	username := user.Username

	req, err := c.NewRequest("GET", "/api/v1/domain-user-permissions/?username="+url.QueryEscape(username)+"&fqdn="+url.QueryEscape(domain), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Try to decode an array response first.
	var list []DomainUserPermissionData
	if err := json.Unmarshal(body, &list); err == nil {
		if len(list) == 0 {
			return nil, ErrNotFound
		}
		return &list[0], nil
	}

	// Fallback to single-object decode.
	var single DomainUserPermissionData
	if err := json.Unmarshal(body, &single); err == nil {
		return &single, nil
	}

	return nil, fmt.Errorf("failed to parse domain access response: %s", string(body))
}

// GetDomain retrieves domain information by FQDN.
// Returns ErrNotFound if the domain does not exist.
func (c *Client) GetDomain(fqdn string) (DomainData, error) {
	req, err := c.NewRequest("GET", "/api/v1/domains/?fqdn="+url.QueryEscape(fqdn), nil)
	if err != nil {
		return DomainData{}, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return DomainData{}, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return DomainData{}, ErrNotFound
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return DomainData{}, fmt.Errorf("failed to read response body: %w", err)
	}

	// Try to decode an array response first.
	var list []DomainData
	if err := json.Unmarshal(body, &list); err == nil {
		if len(list) == 0 {
			return DomainData{}, ErrNotFound
		}
		return list[0], nil
	}

	// Fallback to single-object decode.
	var single DomainData
	if err := json.Unmarshal(body, &single); err == nil {
		return single, nil
	}

	return DomainData{}, fmt.Errorf("failed to parse domain response: %s", string(body))
}

// CreateDomain creates a new domain in the LegoCharm API.
func (c *Client) CreateDomain(domain DomainData) (*DomainData, error) {
	b, err := json.Marshal(domain)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal domain data: %w", err)
	}

	req, err := c.NewRequest("POST", "/api/v1/domains/", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// if we got a non-2xx response, return an error
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to create domain: status %d, body: %s", resp.StatusCode, string(body))
	}

	var domainData DomainData
	if err := json.Unmarshal(body, &domainData); err != nil {
		return nil, fmt.Errorf("failed to parse domain response: %w (body: %s)", err, string(body))
	}
	return &domainData, nil
}

// CreateDomainAccess creates a new domain access permission.
// If the domain does not exist, it will be created automatically.
func (c *Client) CreateDomainAccess(access DomainUserPermissionCreateData) (*DomainUserPermissionData, error) {
	// get domain by fqdn
	domainData, err := c.GetDomain(access.Domain)
	if err != nil && err != ErrNotFound {
		return nil, fmt.Errorf("failed to get domain data: %w", err)
	}
	if err == ErrNotFound {
		// create the domain here
		newDomainData, err := c.CreateDomain(DomainData{Fqdn: access.Domain})
		if err != nil {
			return nil, fmt.Errorf("failed to create domain: %w", err)
		}
		domainData = *newDomainData
	}

	payloadData := DomainUserPermissionCreatePayloadData{
		UserID:      access.UserID,
		Domain:      domainData.ID,
		AccessLevel: access.AccessLevel,
	}

	b, err := json.Marshal(payloadData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload data: %w", err)
	}

	req, err := c.NewRequest("POST", "/api/v1/domain-user-permissions/", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// if we got a non-2xx response, return an error
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to create domain access: status %d, body: %s", resp.StatusCode, string(body))
	}

	var accessData DomainUserPermissionData
	if err := json.Unmarshal(body, &accessData); err != nil {
		return nil, fmt.Errorf("failed to parse domain access response: %w (body: %s)", err, string(body))
	}

	return &accessData, nil
}

// DeleteDomainAccess deletes a domain access permission using the provided ID.
func (c *Client) DeleteDomainAccess(id int) (*http.Response, error) {
	path := fmt.Sprintf("/api/v1/domain-user-permissions/%d/", id)
	req, err := c.NewRequest("DELETE", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	return resp, nil
}

// UserData represents a user returned from the LegoCharm API.
type UserData struct {
	Username string   `json:"username"`
	Url      string   `json:"url"`
	Email    string   `json:"email"`
	Groups   []string `json:"groups"`
}

// UserCreateData represents the data needed to create a new user.
type UserCreateData struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	Email    string   `json:"email"`
	Groups   []string `json:"groups"`
}

// DomainUserPermissionCreateData represents the input data for creating a user's access permission to a domain.
type DomainUserPermissionCreateData struct {
	UserID      string `json:"user"`
	Domain      string `json:"domain"`
	AccessLevel string `json:"access_level"`
}

// DomainUserPermissionCreatePayloadData represents the API payload for creating a domain access permission.
type DomainUserPermissionCreatePayloadData struct {
	UserID      string `json:"user"`
	Domain      int    `json:"domain"`
	AccessLevel string `json:"access_level"`
}

// DomainUserPermissionData represents a user's access permission to a domain as returned from the API.
type DomainUserPermissionData struct {
	UserID      int    `json:"user"`
	Domain      int    `json:"domain"`
	AccessLevel string `json:"access_level"`
	ID          int    `json:"id"`
}

// DomainData represents domain information from the LegoCharm API.
type DomainData struct {
	Fqdn string `json:"fqdn"`
	ID   int    `json:"id"`
}
