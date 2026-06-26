/*
client.go — HTTP client for the Dipole Cloud REST API.

Provides generic CRUD and List operations against /api/v1/<resourceType> endpoints.
All requests include Bearer token auth and JSON content type. Handles 404 as
ErrNotFound, 4xx as descriptive errors, and retries 5xx once before failing.
*/
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

var ErrNotFound = fmt.Errorf("resource not found")

type Client struct {
	baseURL       string
	token         string
	resourceSetID string
	httpClient    *http.Client
}

func New(baseURL, token, resourceSetID string) *Client {
	return &Client{
		baseURL:       baseURL,
		token:         token,
		resourceSetID: resourceSetID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Create(ctx context.Context, resourceType string, body any) (json.RawMessage, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshalling request body: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, c.resourceURL(resourceType, ""), data)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) Read(ctx context.Context, resourceType, id string) (json.RawMessage, error) {
	resp, err := c.do(ctx, http.MethodGet, c.resourceURL(resourceType, id), nil)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) Update(ctx context.Context, resourceType, id string, body any) (json.RawMessage, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshalling request body: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPut, c.resourceURL(resourceType, id), data)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) Delete(ctx context.Context, resourceType, id string) error {
	_, err := c.do(ctx, http.MethodDelete, c.resourceURL(resourceType, id), nil)
	return err
}

func (c *Client) List(ctx context.Context, resourceType string, params map[string]string) (json.RawMessage, error) {
	u := c.resourceURL(resourceType, "")
	if len(params) > 0 {
		q := url.Values{}
		for k, v := range params {
			q.Set(k, v)
		}
		u += "?" + q.Encode()
	}

	resp, err := c.do(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) resourceURL(resourceType, id string) string {
	u := c.baseURL + "/api/v1/" + resourceType
	if id != "" {
		u += "/" + id
	}
	return u
}

func (c *Client) do(ctx context.Context, method, endpoint string, body []byte) (json.RawMessage, error) {
	resp, err := c.doRequest(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}

	// Retry once on 5xx
	if resp.StatusCode >= 500 {
		resp.Body.Close()
		resp, err = c.doRequest(ctx, method, endpoint, body)
		if err != nil {
			return nil, err
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error != "" {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, apiErr.Error)
		}
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	return json.RawMessage(respBody), nil
}

func (c *Client) doRequest(ctx context.Context, method, reqURL string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "terraform-provider-dipole/0.1.0")

	if c.resourceSetID != "" {
		req.Header.Set("X-Dipole-ResourceSet", c.resourceSetID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	return resp, nil
}
