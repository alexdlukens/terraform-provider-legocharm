package legocharmclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

	return &Client{
		BaseURL:    strings.TrimRight(u, "/"),
		Username:   *username,
		Password:   *password,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
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

// GetUser queries the API for a user by username and returns the http response.
func (c *Client) GetUser(username string) (*http.Response, error) {
	req, err := c.NewRequest("GET", "/api/v1/users/?username="+url.QueryEscape(username), nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// GetUserByUsername queries the API for a user by username and returns the
// first matching user record or ErrNotFound if none exist.
func (c *Client) GetUserByUsername(username string) (*UserData, error) {
	resp, err := c.GetUser(username)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
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

	return nil, fmt.Errorf("unable to parse user response: %s", string(body))
}

// CreateUser creates a new user by POSTing the provided user object
// as JSON and returns the created user.
func (c *Client) CreateUser(user UserCreateData) (*UserData, error) {
	b, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}

	req, err := c.NewRequest("POST", "/api/v1/users/", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// if we got a non-2xx response, return an error
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("error creating user: status %d, body: %s", resp.StatusCode, string(body))
	}

	var userData UserData
	if err := json.Unmarshal(body, &userData); err != nil {
		return nil, fmt.Errorf("unable to parse user response: %w (body: %s)", err, string(body))
	}

	return &userData, nil
}

// DeleteUser deletes a user. If the provided url is an absolute URL it will be
// used directly; otherwise it will be treated as a path relative to the
// configured BaseURL.
func (c *Client) DeleteUser(urlStr string) (*http.Response, error) {
	if strings.HasPrefix(urlStr, "http://") || strings.HasPrefix(urlStr, "https://") {
		req, err := http.NewRequest("DELETE", urlStr, nil)
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(c.Username, c.Password)
		req.Header.Set("User-Agent", "terraform-provider-legocharm")
		return c.Do(req)
	}

	// Otherwise treat as relative path.
	req, err := c.NewRequest("DELETE", urlStr, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *Client) HasValidUserPassword(username string, password string) (bool, error) {
	// create a new client with the user credentials
	userClient, err := NewClient(&c.BaseURL, &username, &password)
	if err != nil {
		return false, err
	}
	req, err := userClient.NewRequest("GET", "/api/v1/users/?username="+url.QueryEscape(username), nil)
	if err != nil {
		return false, err
	}

	resp, err := userClient.Do(req)
	if err != nil {
		return false, err
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

// User data types
type UserData struct {
	Username string   `json:"username"`
	Url      string   `json:"url"`
	Email    string   `json:"email"`
	Groups   []string `json:"groups"`
}

type UserCreateData struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	Email    string   `json:"email"`
	Groups   []string `json:"groups"`
}
