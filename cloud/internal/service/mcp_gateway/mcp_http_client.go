/*
mcp_http_client.go — MCP Streamable HTTP client for downstream MCP servers.
Handles JSON-RPC over HTTP with SSE streaming, session management, Bearer auth,
and one transient-failure retry on initialize (covers serverless cold-starts).
*/
package mcp_gateway

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const mcpUpstreamTimeout = 30 * time.Second
const maxSSELineSize = 8 << 20 // 8 MiB

type MCPClientFactory interface {
	ClientFor(server ServerInfo) MCPServerClient
}

type HTTPMCPClientFactory struct {
	httpClient *http.Client
}

func NewHTTPMCPClientFactory() *HTTPMCPClientFactory {
	return &HTTPMCPClientFactory{
		httpClient: &http.Client{},
	}
}

func (f *HTTPMCPClientFactory) ClientFor(server ServerInfo) MCPServerClient {
	return &HTTPMCPClient{
		baseURL:    mcpBaseURL(server),
		httpClient: f.httpClient,
	}
}

type sessionState struct {
	mu          sync.Mutex
	id          string
	sessionless bool
}

type HTTPMCPClient struct {
	baseURL    string
	httpClient *http.Client
	reqID      atomic.Int64
	session    sessionState
}

func NewHTTPMCPClient(host string, port int, path string, tls bool) *HTTPMCPClient {
	return &HTTPMCPClient{
		baseURL:    mcpBaseURL(ServerInfo{Host: host, Port: port, Path: path, TLS: tls}),
		httpClient: &http.Client{},
	}
}

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
	if method != "initialize" {
		if err := c.ensureSession(ctx, token); err != nil {
			return nil, fmt.Errorf("session initialization failed: %w", err)
		}
	}

	res, err := c.doRequest(ctx, token, method, params, c.currentSessionID())
	if err != nil {
		return nil, err
	}
	c.recordSessionID(res.receivedSessionID)

	if res.staleSession {
		if err := c.reinitialize(ctx, token); err != nil {
			return nil, fmt.Errorf("session expired and re-initialization failed: %w", err)
		}
		res, err = c.doRequest(ctx, token, method, params, c.currentSessionID())
		if err != nil {
			return nil, err
		}
		c.recordSessionID(res.receivedSessionID)
		if res.staleSession {
			return nil, fmt.Errorf("MCP server rejected session immediately after re-initialization")
		}
	}
	return res.result, nil
}

type requestOutcome struct {
	result            any
	receivedSessionID string
	staleSession      bool
}

type transientUpstreamError struct {
	err error
}

func (e *transientUpstreamError) Error() string { return e.err.Error() }
func (e *transientUpstreamError) Unwrap() error { return e.err }

func (c *HTTPMCPClient) doRequest(ctx context.Context, token string, method string, params any, sentSessionID string) (requestOutcome, error) {
	reqCtx, cancel := context.WithTimeout(ctx, mcpUpstreamTimeout)
	defer cancel()

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
		return requestOutcome{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return requestOutcome{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if sentSessionID != "" {
		req.Header.Set("Mcp-Session-Id", sentSessionID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return requestOutcome{}, &transientUpstreamError{fmt.Errorf("send request to %s: %w", c.baseURL, err)}
	}
	defer resp.Body.Close()

	receivedSessionID := resp.Header.Get("Mcp-Session-Id")

	if resp.StatusCode == http.StatusNotFound && sentSessionID != "" {
		io.Copy(io.Discard, resp.Body)
		return requestOutcome{staleSession: true}, nil
	}

	if resp.StatusCode == http.StatusAccepted {
		io.Copy(io.Discard, resp.Body)
		return requestOutcome{}, fmt.Errorf("MCP server returned 202 for a request carrying an id")
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		err := fmt.Errorf("MCP server returned status %d: %s", resp.StatusCode, truncate(string(respBody), 200))
		if resp.StatusCode >= 500 {
			return requestOutcome{}, &transientUpstreamError{err}
		}
		return requestOutcome{}, err
	}

	jsonBody, err := readJSONRPCResponse(resp.Body, resp.Header.Get("Content-Type"), id)
	if err != nil {
		return requestOutcome{}, &transientUpstreamError{err}
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
	if err := json.Unmarshal(jsonBody, &rpcResp); err != nil {
		return requestOutcome{}, fmt.Errorf("parse JSON-RPC response: %w", err)
	}

	if rpcResp.Error != nil {
		return requestOutcome{}, fmt.Errorf("MCP server error (code %d): %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return requestOutcome{result: rpcResp.Result, receivedSessionID: receivedSessionID}, nil
}

func (c *HTTPMCPClient) currentSessionID() string {
	c.session.mu.Lock()
	defer c.session.mu.Unlock()
	return c.session.id
}

func (c *HTTPMCPClient) recordSessionID(sid string) {
	if sid == "" {
		return
	}
	c.session.mu.Lock()
	c.session.id = sid
	c.session.mu.Unlock()
}

func (c *HTTPMCPClient) ensureSession(ctx context.Context, token string) error {
	c.session.mu.Lock()
	defer c.session.mu.Unlock()

	if c.session.id != "" || c.session.sessionless {
		return nil
	}
	return c.initializeLocked(ctx, token)
}

func (c *HTTPMCPClient) reinitialize(ctx context.Context, token string) error {
	c.session.mu.Lock()
	defer c.session.mu.Unlock()

	c.session.id = ""
	c.session.sessionless = false
	return c.initializeLocked(ctx, token)
}

func (c *HTTPMCPClient) initializeLocked(ctx context.Context, token string) error {
	params := map[string]any{
		"protocolVersion": "2025-03-26",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "Dipole MCP Identity Gateway",
			"version": "0.1.0",
		},
	}

	const retryDelay = 2 * time.Second

	res, err := c.doRequest(ctx, token, "initialize", params, "")
	if err != nil {
		var transient *transientUpstreamError
		if !errors.As(err, &transient) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryDelay):
		}
		res, err = c.doRequest(ctx, token, "initialize", params, "")
		if err != nil {
			return err
		}
	}

	if res.staleSession {
		return fmt.Errorf("unexpected: initialize reported a stale session")
	}

	c.session.id = res.receivedSessionID
	c.session.sessionless = c.session.id == ""

	c.sendNotification(ctx, token, "notifications/initialized", nil)
	return nil
}

func (c *HTTPMCPClient) sendNotification(ctx context.Context, token string, method string, params any) {
	notifCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if params != nil {
		reqBody["params"] = params
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return
	}

	req, err := http.NewRequestWithContext(notifCtx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if sid := c.session.id; sid != "" {
		req.Header.Set("Mcp-Session-Id", sid)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

func remarshal(src any, dst any) error {
	raw, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dst)
}

func readJSONRPCResponse(body io.Reader, contentType string, requestID int64) ([]byte, error) {
	if !isSSE(contentType) {
		return io.ReadAll(body)
	}

	wantID := strconv.FormatInt(requestID, 10)
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), maxSSELineSize)

	var currentData []byte
	haveData := false
	sawAnyEvent := false

	flush := func() ([]byte, bool) {
		if !haveData {
			return nil, false
		}
		event := currentData
		currentData = nil
		haveData = false
		sawAnyEvent = true
		return event, true
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		switch {
		case bytes.HasPrefix(line, []byte("data:")):
			payload := bytes.TrimPrefix(line, []byte("data:"))
			payload = bytes.TrimPrefix(payload, []byte(" "))
			if haveData {
				currentData = append(currentData, '\n')
				currentData = append(currentData, payload...)
			} else {
				currentData = append([]byte(nil), payload...)
				haveData = true
			}
		case len(bytes.TrimSpace(line)) == 0:
			if event, ok := flush(); ok && sseEventMatchesID(event, wantID) {
				return event, nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read SSE response: %w", err)
	}
	if event, ok := flush(); ok && sseEventMatchesID(event, wantID) {
		return event, nil
	}

	if !sawAnyEvent {
		return nil, fmt.Errorf("SSE response contained no data: events")
	}
	return nil, fmt.Errorf("SSE response closed without a reply matching request id %d", requestID)
}

func sseEventMatchesID(event []byte, wantID string) bool {
	var peek struct {
		ID json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(event, &peek); err != nil {
		return false
	}
	if peek.ID == nil {
		return false
	}
	return string(bytes.TrimSpace(peek.ID)) == wantID
}

func isSSE(contentType string) bool {
	for _, part := range splitContentType(contentType) {
		if part == "text/event-stream" {
			return true
		}
	}
	return false
}

func splitContentType(ct string) []string {
	parts := make([]string, 0, 2)
	for _, p := range bytes.Split([]byte(ct), []byte{';'}) {
		parts = append(parts, string(bytes.TrimSpace(p)))
	}
	return parts
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
