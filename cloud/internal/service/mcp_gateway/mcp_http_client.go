/*
mcp_http_client.go implements MCPServerClient using JSON-RPC over HTTP.

HTTPMCPClient calls downstream MCP servers by sending JSON-RPC 2.0 requests
with Bearer token authorization. It handles tools/list, tools/call,
resources/list, and resources/read methods.
*/
package mcp_gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

const mcpUpstreamTimeout = 30 * time.Second

// MCPClientFactory builds a client for one downstream MCP server.
//
// Implementations must always return a usable client, never nil: callers treat
// the result as ready to use, and an unreachable server is expected to surface as
// an error from a call rather than as a missing client.
//
// The Gateway resolves which servers a request may reach from that tenant's
// Access Policies on every request, then asks the factory for a client per
// server. Clients are therefore not held anywhere: there is no per-tenant
// registry to keep in sync, no cache to expire, and no map that could be keyed
// by one tenant and read by another.
type MCPClientFactory interface {
	ClientFor(server ServerInfo) MCPServerClient
}

// HTTPMCPClientFactory produces JSON-RPC-over-HTTP clients that share one
// http.Client, so connection pooling and keep-alive still apply even though a
// client value is constructed per request. Pooling lives in the shared
// Transport and is keyed by target host, so per-request client values cost
// nothing.
type HTTPMCPClientFactory struct {
	httpClient *http.Client
}

func NewHTTPMCPClientFactory() *HTTPMCPClientFactory {
	return &HTTPMCPClientFactory{
		httpClient: &http.Client{Timeout: mcpUpstreamTimeout},
	}
}

func (f *HTTPMCPClientFactory) ClientFor(server ServerInfo) MCPServerClient {
	return &HTTPMCPClient{
		baseURL:    mcpBaseURL(server),
		httpClient: f.httpClient,
	}
}

type HTTPMCPClient struct {
	baseURL    string
	httpClient *http.Client
	reqID      atomic.Int64
	sessionID  atomic.Value // stores string; empty until initialized
}

// NewHTTPMCPClient builds a standalone client with its own connection pool.
// Production goes through HTTPMCPClientFactory so the pool is shared.
func NewHTTPMCPClient(host string, port int, path string, tls bool) *HTTPMCPClient {
	return &HTTPMCPClient{
		baseURL:    mcpBaseURL(ServerInfo{Host: host, Port: port, Path: path, TLS: tls}),
		httpClient: &http.Client{Timeout: mcpUpstreamTimeout},
	}
}

// mcpBaseURL renders a server's endpoint, omitting the port when it is the
// default for the scheme.
func mcpBaseURL(server ServerInfo) string {
	scheme := "http"
	if server.TLS {
		scheme = "https"
	}

	portStr := ""
	if (server.TLS && server.Port != 443) || (!server.TLS && server.Port != 80) {
		portStr = ":" + strconv.Itoa(server.Port)
	}

	baseURL := scheme + "://" + server.Host + portStr
	if server.Path != "" {
		baseURL += server.Path
	}
	return baseURL
}

func (c *HTTPMCPClient) ListTools(ctx context.Context, token string) ([]MCPTool, error) {
	result, err := c.call(ctx, token, "tools/list", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Tools []MCPTool `json:"tools"`
	}
	if err := remarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("parse tools/list response: %w", err)
	}
	return resp.Tools, nil
}

func (c *HTTPMCPClient) CallTool(ctx context.Context, token string, toolName string, args json.RawMessage) (any, error) {
	params := map[string]any{
		"name": toolName,
	}
	if args != nil {
		params["arguments"] = json.RawMessage(args)
	}
	return c.call(ctx, token, "tools/call", params)
}

func (c *HTTPMCPClient) ListResources(ctx context.Context, token string) ([]MCPResource, error) {
	result, err := c.call(ctx, token, "resources/list", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Resources []MCPResource `json:"resources"`
	}
	if err := remarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("parse resources/list response: %w", err)
	}
	return resp.Resources, nil
}

func (c *HTTPMCPClient) ReadResource(ctx context.Context, token string, uri string) (*MCPResourceContent, error) {
	params := map[string]any{
		"uri": uri,
	}
	result, err := c.call(ctx, token, "resources/read", params)
	if err != nil {
		return nil, err
	}

	var content MCPResourceContent
	if err := remarshal(result, &content); err != nil {
		return nil, fmt.Errorf("parse resources/read response: %w", err)
	}
	return &content, nil
}

func (c *HTTPMCPClient) call(ctx context.Context, token string, method string, params any) (any, error) {
	// Ensure session is initialized before any non-initialize call.
	if method != "initialize" {
		if err := c.ensureSession(ctx, token); err != nil {
			return nil, fmt.Errorf("session initialization failed: %w", err)
		}
	}

	id := c.reqID.Add(1)

	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}
	if params != nil {
		reqBody["params"] = params
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if sid, ok := c.sessionID.Load().(string); ok && sid != "" {
		req.Header.Set("Mcp-Session-Id", sid)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request to %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	// Capture session ID from response if present.
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		c.sessionID.Store(sid)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MCP server returned status %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var rpcResp struct {
		JSONRPC string `json:"jsonrpc"`
		ID      any    `json:"id"`
		Result  any    `json:"result"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("parse JSON-RPC response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("MCP server error (code %d): %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

// ensureSession sends an initialize request if no session ID has been acquired yet.
func (c *HTTPMCPClient) ensureSession(ctx context.Context, token string) error {
	if sid, ok := c.sessionID.Load().(string); ok && sid != "" {
		return nil
	}

	_, err := c.call(ctx, token, "initialize", map[string]any{
		"protocolVersion": "2025-03-26",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "Dipole MCP Identity Gateway",
			"version": "0.1.0",
		},
	})
	return err
}

func remarshal(src any, dst any) error {
	raw, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dst)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
