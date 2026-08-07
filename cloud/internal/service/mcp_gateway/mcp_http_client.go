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

type HTTPMCPClient struct {
	baseURL    string
	httpClient *http.Client
	reqID      atomic.Int64
}

func NewHTTPMCPClient(host string, port int, path string, tls bool) *HTTPMCPClient {
	scheme := "http"
	if tls {
		scheme = "https"
	}

	portStr := ""
	if (tls && port != 443) || (!tls && port != 80) {
		portStr = ":" + strconv.Itoa(port)
	}

	baseURL := scheme + "://" + host + portStr
	if path != "" {
		baseURL += path
	}

	return &HTTPMCPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
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
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request to %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

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
