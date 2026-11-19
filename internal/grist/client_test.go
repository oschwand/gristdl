package grist

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewValidation(t *testing.T) {
	if _, err := New("", "tok"); err == nil || !strings.Contains(err.Error(), "no Grist server") {
		t.Errorf("want server error, got %v", err)
	}
	if _, err := New("https://grist.example.com", ""); err == nil || !strings.Contains(err.Error(), "no API token") {
		t.Errorf("want token error, got %v", err)
	}
	if _, err := New("grist.example.com", "tok"); err == nil || !strings.Contains(err.Error(), "invalid server URL") {
		t.Errorf("want URL error, got %v", err)
	}
	c, err := New("https://grist.example.com/", "tok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Server != "https://grist.example.com" {
		t.Errorf("trailing slash not stripped: %q", c.Server)
	}
}

func TestOrgs(t *testing.T) {
	var gotAuth, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":42,"name":"Grist Labs","domain":"gristlabs","access":"owners","createdAt":"2019-09-13T15:42:35.000Z","updatedAt":"2019-09-13T15:42:35.000Z"}]`))
	}))
	defer srv.Close()

	c, err := New(srv.URL, "secret-token")
	if err != nil {
		t.Fatal(err)
	}
	orgs, err := c.Orgs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret-token" {
		t.Errorf("auth header = %q", gotAuth)
	}
	if gotPath != "/api/orgs" {
		t.Errorf("path = %q", gotPath)
	}
	if len(orgs) != 1 || orgs[0].ID != 42 || orgs[0].Name != "Grist Labs" || *orgs[0].Domain != "gristlabs" {
		t.Errorf("unexpected orgs: %+v", orgs)
	}
}

func TestAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"no user found for api key"}`))
	}))
	defer srv.Close()

	c, err := New(srv.URL, "bad-token")
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Orgs(context.Background())
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("want *APIError, got %T: %v", err, err)
	}
	if apiErr.Status != 401 {
		t.Errorf("status = %d", apiErr.Status)
	}
	if !strings.Contains(apiErr.Error(), "no user found for api key") {
		t.Errorf("error message = %q", apiErr.Error())
	}
}

func TestWorkspaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/orgs/7/workspaces" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(`[{"id":97,"name":"Secret Plans","access":"owners","docs":[{"id":"145","name":"Project Lollipop","urlId":null,"isPinned":true,"access":"owners"}]}]`))
	}))
	defer srv.Close()

	c, err := New(srv.URL, "tok")
	if err != nil {
		t.Fatal(err)
	}
	ws, err := c.Workspaces(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(ws) != 1 || ws[0].ID != 97 || len(ws[0].Docs) != 1 || ws[0].Docs[0].Name != "Project Lollipop" {
		t.Errorf("unexpected workspaces: %+v", ws)
	}
}
