package legocharmclient

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientValidation(t *testing.T) {
	_, err := NewClient(nil, ptr("u"), ptr("p"))
	if err == nil {
		t.Fatal("expected error for nil address")
	}

	_, err = NewClient(ptr(""), ptr("u"), ptr("p"))
	if err == nil {
		t.Fatal("expected error for empty address")
	}

	_, err = NewClient(ptr("https://example.com"), ptr(""), ptr("p"))
	if err == nil {
		t.Fatal("expected error for empty username")
	}

	_, err = NewClient(ptr("https://example.com"), ptr("u"), ptr(""))
	if err == nil {
		t.Fatal("expected error for empty password")
	}
}

func TestNewRequestSetsBasicAuth(t *testing.T) {
	client, err := NewClient(ptr("https://example.com"), ptr("user"), ptr("pass"))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	req, err := client.NewRequest("GET", "/api/v1/thing", nil)
	if err != nil {
		t.Fatalf("unexpected error creating request: %v", err)
	}

	auth := req.Header.Get("Authorization")
	if auth == "" {
		t.Fatalf("expected Authorization header to be set; got empty")
	}
}

func TestDo_Succeeds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok")) // nolint:errcheck
	}))
	defer srv.Close()

	client, err := NewClient(ptr(srv.URL), ptr("u"), ptr("p"))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	req, err := client.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("unexpected error creating request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error doing request: %v", err)
	}
	defer resp.Body.Close() // nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK; got %d", resp.StatusCode)
	}
}

func TestDeleteUser_AbsoluteURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/api/v1/users/1004/" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client, err := NewClient(ptr(srv.URL), ptr("u"), ptr("p"))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	resp, err := client.DeleteUserById("1004")
	if err != nil {
		t.Fatalf("unexpected error deleting user: %v", err)
	}
	defer resp.Body.Close() // nolint:errcheck

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 No Content; got %d", resp.StatusCode)
	}
}

func TestDeleteUser_RelativePath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/api/v1/users/1004/" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client, err := NewClient(ptr(srv.URL), ptr("u"), ptr("p"))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	resp, err := client.DeleteUserById("1004")
	if err != nil {
		t.Fatalf("unexpected error deleting user: %v", err)
	}
	defer resp.Body.Close() // nolint:errcheck

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 No Content; got %d", resp.StatusCode)
	}
}

func ptr(s string) *string {
	return &s
}
