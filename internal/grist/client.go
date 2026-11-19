package grist

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	Server string
	Token  string
	HTTP   *http.Client
}

type Org struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Domain    *string `json:"domain"`
	Access    string  `json:"access"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
}

type Doc struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	URLID    *string `json:"urlId"`
	IsPinned bool    `json:"isPinned"`
	Access   string  `json:"access"`
}

type Workspace struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Access string `json:"access"`
	Docs   []Doc  `json:"docs"`
}

type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	var payload struct {
		Error string `json:"error"`
	}
	if json.Unmarshal([]byte(e.Body), &payload) == nil && payload.Error != "" {
		return fmt.Sprintf("grist API error (HTTP %d): %s", e.Status, payload.Error)
	}
	if e.Body != "" {
		return fmt.Sprintf("grist API error (HTTP %d): %s", e.Status, e.Body)
	}
	return fmt.Sprintf("grist API error (HTTP %d)", e.Status)
}

func New(server, token string) (*Client, error) {
	if server == "" {
		return nil, fmt.Errorf("no Grist server configured: use --server, GRISTDL_SERVER, or server in the config file")
	}
	if token == "" {
		return nil, fmt.Errorf("no API token configured: use --token, GRISTDL_TOKEN, token in the config file, or a .env file")
	}
	u, err := url.Parse(server)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid server URL %q", server)
	}
	return &Client{
		Server: strings.TrimRight(u.String(), "/"),
		Token:  token,
		HTTP:   &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *Client) do(ctx context.Context, method, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.Server+path, nil)
	if err != nil {
		return fmt.Errorf("build request %s %s: %w", method, path, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read response of %s %s: %w", method, path, err)
	}
	if resp.StatusCode >= 400 {
		return &APIError{Status: resp.StatusCode, Body: strings.TrimSpace(string(body))}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode response of %s %s: %w", method, path, err)
	}
	return nil
}

func (c *Client) Orgs(ctx context.Context) ([]Org, error) {
	var orgs []Org
	if err := c.do(ctx, http.MethodGet, "/api/orgs", &orgs); err != nil {
		return nil, err
	}
	return orgs, nil
}

func (c *Client) Workspaces(ctx context.Context, orgID int64) ([]Workspace, error) {
	var ws []Workspace
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/orgs/%d/workspaces", orgID), &ws); err != nil {
		return nil, err
	}
	return ws, nil
}
