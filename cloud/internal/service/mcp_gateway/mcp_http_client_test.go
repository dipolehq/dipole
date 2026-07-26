/*
mcp_http_client_test.go tests the HTTPMCPClient that calls downstream MCP
servers using JSON-RPC over HTTP with Bearer token authorization.
*/
package mcp_gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPMCPClient_ListTools(t *testing.T) {
	var receivedMethod string
	var receivedToken string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.Header.Get("Authorization")

		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)
		receivedMethod = req["method"].(string)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req["id"],
			"result": map[string]any{
				"tools": []map[string]any{
					{"name": "create_issue", "description": "Create a Jira issue"},
					{"name": "search_issues", "description": "Search issues"},
				},
			},
		})
	}))
	defer srv.Close()

	client := newClientFromTestServer(srv)

	tools, err := client.ListTools(context.Background(), "test-token-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedMethod != "tools/list" {
		t.Errorf("expected method 'tools/list', got '%s'", receivedMethod)
	}
	if receivedToken != "Bearer test-token-123" {
		t.Errorf("expected 'Bearer test-token-123', got '%s'", receivedToken)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	if tools[0].Name != "create_issue" {
		t.Errorf("expected first tool 'create_issue', got '%s'", tools[0].Name)
	}
	if tools[1].Description != "Search issues" {
		t.Errorf("expected description 'Search issues', got '%s'", tools[1].Description)
	}
}

func TestHTTPMCPClient_CallTool(t *testing.T) {
	var receivedMethod string
	var receivedParams map[string]any
	var receivedToken string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.Header.Get("Authorization")

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got '%s'", r.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)
		receivedMethod = req["method"].(string)
		if p, ok := req["params"].(map[string]any); ok {
			receivedParams = p
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req["id"],
			"result":  map[string]any{"issueId": "PROJ-123"},
		})
	}))
	defer srv.Close()

	client := newClientFromTestServer(srv)

	args := json.RawMessage(`{"project": "PROJ", "summary": "Test issue"}`)
	result, err := client.CallTool(context.Background(), "my-bearer-token", "create_issue", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedMethod != "tools/call" {
		t.Errorf("expected method 'tools/call', got '%s'", receivedMethod)
	}
	if receivedToken != "Bearer my-bearer-token" {
		t.Errorf("expected 'Bearer my-bearer-token', got '%s'", receivedToken)
	}
	if receivedParams["name"] != "create_issue" {
		t.Errorf("expected tool name 'create_issue', got '%v'", receivedParams["name"])
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected map result")
	}
	if resultMap["issueId"] != "PROJ-123" {
		t.Errorf("expected issueId 'PROJ-123', got '%v'", resultMap["issueId"])
	}
}

func TestHTTPMCPClient_ListResources(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req["id"],
			"result": map[string]any{
				"resources": []map[string]any{
					{"uri": "file:///docs/readme.md", "name": "README", "mimeType": "text/markdown"},
				},
			},
		})
	}))
	defer srv.Close()

	client := newClientFromTestServer(srv)

	resources, err := client.ListResources(context.Background(), "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if resources[0].URI != "file:///docs/readme.md" {
		t.Errorf("expected URI 'file:///docs/readme.md', got '%s'", resources[0].URI)
	}
	if resources[0].MimeType != "text/markdown" {
		t.Errorf("expected mimeType 'text/markdown', got '%s'", resources[0].MimeType)
	}
}

func TestHTTPMCPClient_ReadResource(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req["id"],
			"result": map[string]any{
				"uri":      "file:///docs/readme.md",
				"mimeType": "text/markdown",
				"text":     "# Hello World",
			},
		})
	}))
	defer srv.Close()

	client := newClientFromTestServer(srv)

	content, err := client.ReadResource(context.Background(), "token", "file:///docs/readme.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content == nil {
		t.Fatal("expected non-nil content")
	}
	if content.Text != "# Hello World" {
		t.Errorf("expected text '# Hello World', got '%s'", content.Text)
	}
	if content.URI != "file:///docs/readme.md" {
		t.Errorf("expected URI 'file:///docs/readme.md', got '%s'", content.URI)
	}
}

func TestHTTPMCPClient_ServerError_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer srv.Close()

	client := newClientFromTestServer(srv)

	_, err := client.ListTools(context.Background(), "token")
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
	if !contains(err.Error(), "500") {
		t.Errorf("expected error to mention status 500, got: %v", err)
	}
}

func TestHTTPMCPClient_JSONRPCError_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req["id"],
			"error": map[string]any{
				"code":    -32600,
				"message": "Invalid request",
			},
		})
	}))
	defer srv.Close()

	client := newClientFromTestServer(srv)

	_, err := client.CallTool(context.Background(), "token", "bad_tool", nil)
	if err == nil {
		t.Fatal("expected error for JSON-RPC error response")
	}
	if !contains(err.Error(), "-32600") {
		t.Errorf("expected error code -32600, got: %v", err)
	}
	if !contains(err.Error(), "Invalid request") {
		t.Errorf("expected error message 'Invalid request', got: %v", err)
	}
}

// newClientFromTestServer creates an HTTPMCPClient pointing at the test server.
func newClientFromTestServer(srv *httptest.Server) *HTTPMCPClient {
	return &HTTPMCPClient{
		baseURL:    srv.URL,
		httpClient: srv.Client(),
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
