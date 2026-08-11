/*
mcp_http_client.go implements MCPServerClient using JSON-RPC over HTTP,
against the MCP Streamable HTTP transport.

HTTPMCPClient calls downstream MCP servers for tools/list, tools/call,
resources/list, and resources/read, with Bearer token authorization.

Design notes:
  - Each call is bounded by mcpUpstreamTimeout via a per-call context deadline
    (not http.Client.Timeout), so the bound covers reading a slow/streaming SSE
    body too, without capping connection reuse across unrelated calls.
  - Session state (sessionState) is guarded by a mutex, not atomics: acquiring
    a session is check-then-act (read, then maybe initialize and write), which
    needs to be atomic as a whole so concurrent callers can't both initialize.
  - doRequest never touches session state directly (session id is passed in,
    received id is returned out) so it can be called safely both with and
    without session.mu already held — it has no locking of its own.
  - A JSON response body is read in full; an SSE response body is scanned
    event-by-event and returned as soon as the event whose JSON-RPC id matches
    the request is found, without waiting for the server to close the stream.
  - ID correlation compares raw JSON bytes of the "id" field rather than
    decoding it into a Go type, so it isn't coupled to any assumption about
    whether an id is numeric or string — it only has to match what this
    client itself encoded when sending the request.
  - This client advertises protocolVersion "2025-03-26" in initialize and
    records whatever version the server's InitializeResult actually reports;
    it does not implement multi-version negotiation or retry-on-mismatch.
  - A stale session (404 on a request that carried a session id) triggers one
    re-initialize + resend of the same request, so callers get a real result or
    a real error, never a message that only claims a retry will happen.
  - A 202 response is only valid for notification/response-only POST bodies;
    this client always sends requests (always carries an id), so a 202 here is
    treated as a protocol error rather than "success with no result".
*/
package mcp_gateway

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const mcpUpstreamTimeout = 30 * time.Second

// maxSSELineSize bounds a single SSE "data:" line, which for this transport
// is an entire JSON-RPC message on one line (JSON has no literal newlines
// unless the server pretty-prints, which it shouldn't). bufio.Scanner's
// default 64KiB limit is too small for a tools/list response listing many
// tools with full input schemas.
const maxSSELineSize = 8 << 20 // 8MiB

// MCPClientFactory builds a client for one downstream MCP server. Clients are
// built per request from that request's Access Policy, not cached, so a
// server only ever becomes reachable through the policy that authorizes it.
type MCPClientFactory interface {
	ClientFor(server ServerInfo) MCPServerClient
}

// HTTPMCPClientFactory shares one http.Client (and its connection pool)
// across all clients it produces.
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

// NewHTTPMCPClient builds a standalone client with its own connection pool.
// Production goes through HTTPMCPClientFactory so the pool is shared.
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
		return requestOutcome{}, fmt.Errorf("send request to %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	receivedSessionID := resp.Header.Get("Mcp-Session-Id")

	if resp.StatusCode == http.StatusNotFound && sentSessionID != "" {
		io.Copy(io.Discard, resp.Body)
		return requestOutcome{staleSession: true}, nil
	}

	if resp.StatusCode == http.StatusAccepted {
		io.Copy(io.Discard, resp.Body)
		return requestOutcome{}, fmt.Errorf("MCP server returned 202 Accepted for a JSON-RPC request, which is not a valid response to a request carrying an id")
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return requestOutcome{}, fmt.Errorf("MCP server returned status %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	jsonBody, err := readJSONRPCResponse(resp.Body, resp.Header.Get("Content-Type"), id)
	if err != nil {
		return requestOutcome{}, err
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

// initializeLocked requires c.session.mu to already be held by the caller.
func (c *HTTPMCPClient) initializeLocked(ctx context.Context, token string) error {
	res, err := c.doRequest(ctx, token, "initialize", map[string]any{
		"protocolVersion": "2025-03-26",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "Dipole MCP Identity Gateway",
			"version": "0.1.0",
		},
	}, "")
	if err != nil {
		return err
	}
	if res.staleSession {
		return fmt.Errorf("unexpected: initialize reported a stale session")
	}

	c.session.id = res.receivedSessionID
	c.session.sessionless = c.session.id == ""
	return nil
}

func remarshal(src any, dst any) error {
	raw, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dst)
}

// readJSONRPCResponse reads a response body and returns the bytes of the
// JSON-RPC message that answers requestID.
//
// For application/json the whole body is one JSON-RPC object and is read in
// full. For text/event-stream the body is scanned line by line as events
// arrive; each completed event's "id" field is compared (as raw JSON bytes,
// not decoded into a Go type) against requestID's own JSON encoding, and the
// function returns as soon as a match is found — it does not wait for the
// server to close the stream, so a long-lived SSE connection that keeps
// emitting notifications after our response has already been sent will not
// block this call.
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

// sseEventMatchesID reports whether event's JSON-RPC "id" field, taken as raw
// JSON bytes, equals wantID (already the canonical decimal encoding of the
// request id this client sent). Comparing raw bytes rather than decoding into
// a Go type avoids assuming ids are numeric on the wire; this client only
// needs to recognize the exact id it itself generated.
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
