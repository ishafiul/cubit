package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// CubitClient handles HTTP communication with the Cubit Control Plane API.
type CubitClient struct {
	endpoint   string
	token      string
	httpClient *http.Client
}

// NewCubitClient creates a new configured CubitClient.
func NewCubitClient(endpoint, token string) *CubitClient {
	endpoint = strings.TrimRight(endpoint, "/")
	if !strings.HasSuffix(endpoint, "/api/v1") {
		endpoint = endpoint + "/api/v1"
	}
	return &CubitClient{
		endpoint:   endpoint,
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// ClientNode represents a registered server in Cubit.
type ClientNode struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	IPAddress string `json:"ipAddress"`
	Status    string `json:"status,omitempty"`
	CPUCores  int    `json:"cpuCores,omitempty"`
	MemoryMB  int    `json:"memoryMb,omitempty"`
}

// ClientApplication represents a worker application.
type ClientApplication struct {
	ID               string            `json:"id,omitempty"`
	Name             string            `json:"name"`
	SourceType       string            `json:"sourceType,omitempty"`
	GitRepo          string            `json:"gitRepo,omitempty"`
	InlineCode       string            `json:"inlineCode,omitempty"`
	ActiveDeployment string            `json:"activeDeploymentId,omitempty"`
	Status           string            `json:"status,omitempty"`
	Env              map[string]string `json:"env,omitempty"`
	Bindings         []ClientBinding   `json:"bindings,omitempty"`
}

// ClientBinding represents a KV, D1, or Queue resource binding.
type ClientBinding struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	TargetResource string `json:"targetResource"`
}

// ClientDomain represents a domain routing rule.
type ClientDomain struct {
	ID            string `json:"id,omitempty"`
	DomainName    string `json:"hostname"`
	ApplicationID string `json:"applicationId"`
	PathPrefix    string `json:"pathPrefix,omitempty"`
	TLSStatus     string `json:"tlsStatus,omitempty"`
}

func (c *CubitClient) doRequest(ctx context.Context, method, path string, in, out interface{}) error {
	url := fmt.Sprintf("%s%s", c.endpoint, path)
	var body io.Reader
	if in != nil {
		buf, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("resource not found (404)")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("api error %d: %s", resp.StatusCode, string(respBytes))
	}

	if out != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// Node API
func (c *CubitClient) CreateNode(ctx context.Context, node *ClientNode) (*ClientNode, error) {
	var result ClientNode
	req := map[string]interface{}{
		"name":      node.Name,
		"ipAddress": node.IPAddress,
		"specs": map[string]interface{}{
			"cpuCores": node.CPUCores,
			"memoryMb": node.MemoryMB,
		},
	}
	if err := c.doRequest(ctx, http.MethodPost, "/nodes", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *CubitClient) GetNode(ctx context.Context, id string) (*ClientNode, error) {
	var result ClientNode
	if err := c.doRequest(ctx, http.MethodGet, "/nodes/"+id, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *CubitClient) DeleteNode(ctx context.Context, id string) error {
	return c.doRequest(ctx, http.MethodDelete, "/nodes/"+id, nil, nil)
}

// Application API
func (c *CubitClient) CreateApplication(ctx context.Context, app *ClientApplication) (*ClientApplication, error) {
	var result ClientApplication
	req := map[string]interface{}{
		"name": app.Name,
	}
	if app.SourceType != "" {
		req["sourceType"] = app.SourceType
	}
	if app.GitRepo != "" {
		req["gitRepo"] = app.GitRepo
	}
	if app.InlineCode != "" {
		req["inlineCode"] = app.InlineCode
	}
	if err := c.doRequest(ctx, http.MethodPost, "/applications", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *CubitClient) GetApplication(ctx context.Context, id string) (*ClientApplication, error) {
	var result ClientApplication
	if err := c.doRequest(ctx, http.MethodGet, "/applications/"+id, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *CubitClient) UpdateApplication(ctx context.Context, id string, app *ClientApplication) (*ClientApplication, error) {
	var result ClientApplication
	req := map[string]interface{}{}
	if app.GitRepo != "" {
		req["gitRepo"] = app.GitRepo
	}
	if app.InlineCode != "" {
		req["inlineCode"] = app.InlineCode
	}
	if err := c.doRequest(ctx, http.MethodPut, "/applications/"+id, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *CubitClient) DeleteApplication(ctx context.Context, id string) error {
	return c.doRequest(ctx, http.MethodDelete, "/applications/"+id, nil, nil)
}

// Domain API
func (c *CubitClient) CreateDomain(ctx context.Context, domain *ClientDomain) (*ClientDomain, error) {
	var result ClientDomain
	req := map[string]interface{}{
		"hostname":      domain.DomainName,
		"applicationId": domain.ApplicationID,
		"pathPrefix":    domain.PathPrefix,
	}
	if err := c.doRequest(ctx, http.MethodPost, "/domains", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *CubitClient) GetDomain(ctx context.Context, id string) (*ClientDomain, error) {
	var domains []ClientDomain
	if err := c.doRequest(ctx, http.MethodGet, "/domains", nil, &domains); err != nil {
		return nil, err
	}
	for _, d := range domains {
		if d.ID == id || d.DomainName == id {
			return &d, nil
		}
	}
	return nil, fmt.Errorf("resource not found (404)")
}

func (c *CubitClient) DeleteDomain(ctx context.Context, id string) error {
	return c.doRequest(ctx, http.MethodDelete, "/domains/"+id, nil, nil)
}
