package mcp_gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/events"
	"github.com/google/uuid"
)

func testScope(tenantID uuid.UUID) TenantScope {
	return TenantScope{TenantID: tenantID, TenantSubdomain: "test", GatewayURL: testGatewayURL}
}

type mapClientFactory map[string]MCPServerClient

func (f mapClientFactory) ClientFor(server ServerInfo) MCPServerClient {
	if client, ok := f[server.Name]; ok {
		return client
	}
	return unreachableMCPClient{}
}

type unreachableMCPClient struct{}

func (unreachableMCPClient) ListTools(context.Context, string) ([]MCPTool, http.Header, error) {
	return nil, nil, errors.New("unreachable MCP server")
}

func (unreachableMCPClient) CallTool(context.Context, string, string, json.RawMessage) (any, http.Header, error) {
	return nil, nil, errors.New("unreachable MCP server")
}

func (unreachableMCPClient) ListResources(context.Context, string) ([]MCPResource, http.Header, error) {
	return nil, nil, errors.New("unreachable MCP server")
}

func (unreachableMCPClient) ReadResource(context.Context, string, string) (*MCPResourceContent, http.Header, error) {
	return nil, nil, errors.New("unreachable MCP server")
}

type mockDecrypter struct {
	token string
}

func (d *mockDecrypter) DecryptCredential(_ []byte, _ string) (string, error) {
	return d.token, nil
}

type mockMCPClient struct {
	tools      []MCPTool
	resources  []MCPResource
	callResult any
	callErr    error
	lastToken  string
	lastTool   string
}

func (c *mockMCPClient) ListTools(_ context.Context, token string) ([]MCPTool, http.Header, error) {
	c.lastToken = token
	return c.tools, nil, nil
}

func (c *mockMCPClient) CallTool(_ context.Context, token string, toolName string, _ json.RawMessage) (any, http.Header, error) {
	c.lastToken = token
	c.lastTool = toolName
	if c.callErr != nil {
		return nil, nil, c.callErr
	}
	return c.callResult, nil, nil
}

func (c *mockMCPClient) ListResources(_ context.Context, token string) ([]MCPResource, http.Header, error) {
	c.lastToken = token
	return c.resources, nil, nil
}

func (c *mockMCPClient) ReadResource(_ context.Context, token string, uri string) (*MCPResourceContent, http.Header, error) {
	c.lastToken = token
	for _, r := range c.resources {
		if r.URI == uri {
			return &MCPResourceContent{URI: uri, Text: "content"}, nil, nil
		}
	}
	return nil, nil, nil
}

type mockRightSidePolicyFinder struct {
	policies []RightSidePolicy
}

func (f *mockRightSidePolicyFinder) FindGatewayPolicies(_ context.Context, _ uuid.UUID, _ string) ([]RightSidePolicy, error) {
	return f.policies, nil
}

func (f *mockRightSidePolicyFinder) FindPolicyForServer(_ context.Context, _ uuid.UUID, _ string, host string, port int) (*RightSidePolicy, error) {
	for _, p := range f.policies {
		if p.ServerWorkload.Host == host && p.ServerWorkload.Port == port {
			return &p, nil
		}
	}
	return nil, nil
}

type mockTokenStore struct {
	tokens map[string]*UserTokenEntry
}

func newMockTokenStore() *mockTokenStore {
	return &mockTokenStore{tokens: make(map[string]*UserTokenEntry)}
}

func (s *mockTokenStore) Get(_ context.Context, _ uuid.UUID, userID string, serverID uuid.UUID) (*UserTokenEntry, error) {
	key := userID + ":" + serverID.String()
	entry, ok := s.tokens[key]
	if !ok {
		return nil, nil
	}
	return entry, nil
}

func (s *mockTokenStore) Store(_ context.Context, _ uuid.UUID, userID string, serverID uuid.UUID, token *UserTokenEntry) error {
	key := userID + ":" + serverID.String()
	s.tokens[key] = token
	return nil
}

func TestToolsList_AggregatesWithPrefixes(t *testing.T) {
	jiraID := uuid.New()
	slackID := uuid.New()
	tenantID := uuid.New()

	jiraClient := &mockMCPClient{
		tools: []MCPTool{
			{Name: "create_issue", Description: "Create a Jira issue"},
			{Name: "search_issues", Description: "Search issues", Annotations: map[string]any{"readOnlyHint": true}},
		},
	}
	slackClient := &mockMCPClient{
		tools: []MCPTool{
			{Name: "send_message", Description: "Send a Slack message"},
			{Name: "list_channels", Description: "List channels"},
		},
	}

	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{ServerWorkload: ServerInfo{ID: jiraID, Name: "jira"}, CredentialType: "oauth_authorization_code", SharedAccessToken: "jira-token"},
			{ServerWorkload: ServerInfo{ID: slackID, Name: "slack"}, CredentialType: "oauth_authorization_code", SharedAccessToken: "slack-token"},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, nil)
	clients := mapClientFactory{"jira": jiraClient, "slack": slackClient}
	handler := NewMethodHandler(rightAuth, clients, nil, nil)

	result, err := handler.HandleMethod(context.Background(), &RequestContext{UserID: "alice@corp.com", Scope: testScope(tenantID)}, "tools/list", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected map result")
	}
	tools, ok := resp["tools"].([]MCPTool)
	if !ok {
		t.Fatal("expected tools array")
	}
	if len(tools) != 4 {
		t.Fatalf("expected 4 tools, got %d", len(tools))
	}

	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name] = true
	}

	jiraPrefix := toolPrefix(ServerInfo{ID: jiraID, Name: "jira"})
	slackPrefix := toolPrefix(ServerInfo{ID: slackID, Name: "slack"})

	if !names[jiraPrefix+"__create_issue"] {
		t.Errorf("missing %s__create_issue", jiraPrefix)
	}
	if !names[jiraPrefix+"__search_issues"] {
		t.Errorf("missing %s__search_issues", jiraPrefix)
	}
	if !names[slackPrefix+"__send_message"] {
		t.Errorf("missing %s__send_message", slackPrefix)
	}
	if !names[slackPrefix+"__list_channels"] {
		t.Errorf("missing %s__list_channels", slackPrefix)
	}

	// Verify annotations preserved
	for _, tool := range tools {
		if tool.Name == jiraPrefix+"__search_issues" {
			if tool.Annotations == nil || tool.Annotations["readOnlyHint"] != true {
				t.Error("annotations not preserved on jira__search_issues")
			}
		}
	}
}

func TestToolsCall_RoutesToCorrectServer(t *testing.T) {
	jiraID := uuid.New()
	githubID := uuid.New()
	tenantID := uuid.New()

	jiraClient := &mockMCPClient{callResult: map[string]any{"issueId": "PROJ-123"}}
	githubClient := &mockMCPClient{callResult: map[string]any{"prId": 42}}

	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{ServerWorkload: ServerInfo{ID: jiraID, Name: "jira"}, CredentialType: "oauth_authorization_code", SharedAccessToken: "jira-token"},
			{ServerWorkload: ServerInfo{ID: githubID, Name: "github"}, CredentialType: "oauth_authorization_code", SharedAccessToken: "github-token"},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, nil)
	clients := mapClientFactory{"jira": jiraClient, "github": githubClient}
	handler := NewMethodHandler(rightAuth, clients, nil, nil)

	// Call jira tool
	jiraPrefix := toolPrefix(ServerInfo{ID: jiraID, Name: "jira"})
	params := json.RawMessage(fmt.Sprintf(`{"name": "%s__create_issue", "arguments": {"project": "PROJ"}}`, jiraPrefix))
	_, err := handler.HandleMethod(context.Background(), &RequestContext{UserID: "alice@corp.com", Scope: testScope(tenantID)}, "tools/call", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jiraClient.lastTool != "create_issue" {
		t.Errorf("expected tool name 'create_issue', got '%s'", jiraClient.lastTool)
	}
	if jiraClient.lastToken != "jira-token" {
		t.Errorf("expected jira-token, got '%s'", jiraClient.lastToken)
	}

	// Call github tool
	githubPrefix := toolPrefix(ServerInfo{ID: githubID, Name: "github"})
	params = json.RawMessage(fmt.Sprintf(`{"name": "%s__create_pr", "arguments": {"title": "fix"}}`, githubPrefix))
	_, err = handler.HandleMethod(context.Background(), &RequestContext{UserID: "alice@corp.com", Scope: testScope(tenantID)}, "tools/call", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if githubClient.lastTool != "create_pr" {
		t.Errorf("expected tool name 'create_pr', got '%s'", githubClient.lastTool)
	}
	if githubClient.lastToken != "github-token" {
		t.Errorf("expected github-token, got '%s'", githubClient.lastToken)
	}
}

func TestToolsCall_UnknownPrefix_Error(t *testing.T) {
	tenantID := uuid.New()
	policyFinder := &mockRightSidePolicyFinder{}
	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, nil)
	handler := NewMethodHandler(rightAuth, mapClientFactory{}, nil, nil)

	params := json.RawMessage(`{"name": "unknown__tool", "arguments": {}}`)
	_, err := handler.HandleMethod(context.Background(), &RequestContext{UserID: "alice@corp.com", Scope: testScope(tenantID)}, "tools/call", params)
	if err == nil {
		t.Fatal("expected error for unknown server prefix")
	}
}

func TestToolsCall_NoPrefixSeparator_Error(t *testing.T) {
	tenantID := uuid.New()
	policyFinder := &mockRightSidePolicyFinder{}
	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, nil)
	handler := NewMethodHandler(rightAuth, mapClientFactory{}, nil, nil)

	params := json.RawMessage(`{"name": "no-prefix-tool", "arguments": {}}`)
	_, err := handler.HandleMethod(context.Background(), &RequestContext{UserID: "alice@corp.com", Scope: testScope(tenantID)}, "tools/call", params)
	if err == nil {
		t.Fatal("expected error for tool name without prefix")
	}
}

func TestSanitizeToolName_NoConsecutiveUnderscores(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Customer Database", "customer_database"},
		{"foo  bar", "foo_bar"},
		{"foo__bar", "foo_bar"},
		{"foo---bar", "foo---bar"},
		{"foo@#$bar", "foobar"},
		{"  leading spaces", "leading_spaces"},
		{"trailing spaces  ", "trailing_spaces"},
		{"MCP Server (v2)", "mcp_server_v2"},
		{"already_clean", "already_clean"},
		{"UPPER-case-MIX", "upper-case-mix"},
	}
	for _, tc := range cases {
		got := sanitizeToolName(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeToolName(%q) = %q, want %q", tc.input, got, tc.want)
		}
		if strings.Contains(got, "__") {
			t.Errorf("sanitizeToolName(%q) = %q contains '__'", tc.input, got)
		}
	}
}

func TestToolPrefix_UniqueForDifferentIDs(t *testing.T) {
	id1 := uuid.MustParse("a36a9b13-2d72-442f-bb0f-1bb567f63680")
	id2 := uuid.MustParse("72c91e44-1a3b-4f8c-9d2e-0a1b2c3d4e5f")

	p1 := toolPrefix(ServerInfo{ID: id1, Name: "Customer Database"})
	p2 := toolPrefix(ServerInfo{ID: id2, Name: "Customer Database"})

	if p1 == p2 {
		t.Errorf("same sanitized name with different IDs must produce different prefixes: %q vs %q", p1, p2)
	}
	if !strings.HasPrefix(p1, "customer_database_") {
		t.Errorf("prefix should start with sanitized name: %q", p1)
	}
}

func TestToolsCall_SuccessfulCallToServer(t *testing.T) {
	serverID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	tenantID := uuid.New()

	client := &mockMCPClient{callResult: map[string]any{"ok": true}}
	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{ServerWorkload: ServerInfo{ID: serverID, Name: "Customer Database MCP Server"}, CredentialType: "api_key"},
		},
	}
	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, &mockDecrypter{token: "secret"})
	clients := mapClientFactory{"Customer Database MCP Server": client}

	handler := NewMethodHandler(rightAuth, clients, nil, nil)

	prefix := toolPrefix(ServerInfo{ID: serverID, Name: "Customer Database MCP Server"})
	params := json.RawMessage(fmt.Sprintf(`{"name": %q, "arguments": {}}`, prefix+"__list_customers"))
	result, err := handler.HandleMethod(context.Background(), &RequestContext{UserID: "alice@corp.com", Scope: testScope(tenantID)}, "tools/call", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestToolsCall_FailurePath_ReturnsError(t *testing.T) {
	serverID := uuid.MustParse("66666666-7777-8888-9999-000000000000")
	tenantID := uuid.New()

	client := &mockMCPClient{callErr: errors.New("downstream failed")}
	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{ServerWorkload: ServerInfo{ID: serverID, Name: "Payments API"}, CredentialType: "api_key"},
		},
	}
	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, &mockDecrypter{token: "secret"})
	clients := mapClientFactory{"Payments API": client}

	handler := NewMethodHandler(rightAuth, clients, nil, nil)

	prefix := toolPrefix(ServerInfo{ID: serverID, Name: "Payments API"})
	params := json.RawMessage(fmt.Sprintf(`{"name": %q, "arguments": {}}`, prefix+"__charge"))
	_, err := handler.HandleMethod(context.Background(), &RequestContext{UserID: "alice@corp.com", Scope: testScope(tenantID)}, "tools/call", params)
	if err == nil {
		t.Fatal("expected error from downstream failure")
	}
}

func TestToolsListAndCall_RoundTrip(t *testing.T) {
	serverID := uuid.MustParse("deadbeef-1234-5678-abcd-ef0123456789")
	tenantID := uuid.New()

	client := &mockMCPClient{
		tools:      []MCPTool{{Name: "query_data", Description: "Query"}},
		callResult: map[string]any{"rows": 42},
	}

	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{ServerWorkload: ServerInfo{ID: serverID, Name: "My  Server!"}, CredentialType: "api_key"},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, &mockDecrypter{token: "secret"})
	clients := mapClientFactory{"My  Server!": client}
	handler := NewMethodHandler(rightAuth, clients, nil, nil)

	// Step 1: tools/list
	result, err := handler.HandleMethod(context.Background(), &RequestContext{UserID: "u", Scope: testScope(tenantID)}, "tools/list", nil)
	if err != nil {
		t.Fatalf("tools/list error: %v", err)
	}
	resp := result.(map[string]any)
	tools := resp["tools"].([]MCPTool)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	// The exposed name must not contain spaces or special chars
	exposedName := tools[0].Name
	for _, c := range exposedName {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			t.Fatalf("exposed tool name %q contains invalid char %q", exposedName, string(c))
		}
	}

	// Step 2: tools/call with the exact name from tools/list
	callParams := json.RawMessage(fmt.Sprintf(`{"name": %q, "arguments": {}}`, exposedName))
	_, err = handler.HandleMethod(context.Background(), &RequestContext{UserID: "u", Scope: testScope(tenantID)}, "tools/call", callParams)
	if err != nil {
		t.Fatalf("tools/call with name from tools/list failed: %v", err)
	}
	if client.lastTool != "query_data" {
		t.Errorf("expected downstream tool 'query_data', got %q", client.lastTool)
	}
	if client.lastToken != "secret" {
		t.Errorf("expected token 'secret', got %q", client.lastToken)
	}
}

func TestPing_RespondsDirect(t *testing.T) {
	tenantID := uuid.New()
	policyFinder := &mockRightSidePolicyFinder{}
	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, nil)
	handler := NewMethodHandler(rightAuth, mapClientFactory{}, nil, nil)

	result, err := handler.HandleMethod(context.Background(), &RequestContext{UserID: "alice@corp.com", Scope: testScope(tenantID)}, "ping", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for ping")
	}
}

func TestInitialize_ReturnsCapabilities(t *testing.T) {
	tenantID := uuid.New()
	policyFinder := &mockRightSidePolicyFinder{}
	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, nil)
	handler := NewMethodHandler(rightAuth, mapClientFactory{}, nil, nil)

	result, err := handler.HandleMethod(context.Background(), &RequestContext{UserID: "alice@corp.com", Scope: testScope(tenantID)}, "initialize", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected map result")
	}

	if resp["protocolVersion"] != "2024-11-05" {
		t.Errorf("expected protocolVersion '2024-11-05', got '%v'", resp["protocolVersion"])
	}

	serverInfo, ok := resp["serverInfo"].(map[string]any)
	if !ok {
		t.Fatal("expected serverInfo map")
	}
	if serverInfo["name"] != "Dipole MCP Identity Gateway" {
		t.Errorf("unexpected server name: %v", serverInfo["name"])
	}
	if serverInfo["version"] != "0.1.0" {
		t.Errorf("unexpected server version: %v", serverInfo["version"])
	}

	capabilities, ok := resp["capabilities"].(map[string]any)
	if !ok {
		t.Fatal("expected capabilities map")
	}
	if _, ok := capabilities["tools"]; !ok {
		t.Error("missing tools capability")
	}
	if _, ok := capabilities["resources"]; !ok {
		t.Error("missing resources capability")
	}
}

func TestResourcesList_AggregatesWithoutPrefixes(t *testing.T) {
	jiraID := uuid.New()
	slackID := uuid.New()
	tenantID := uuid.New()

	jiraClient := &mockMCPClient{
		resources: []MCPResource{
			{URI: "file:///jira/boards.json", Name: "Jira Boards", MimeType: "application/json"},
		},
	}
	slackClient := &mockMCPClient{
		resources: []MCPResource{
			{URI: "file:///slack/channels.json", Name: "Channels"},
		},
	}

	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{ServerWorkload: ServerInfo{ID: jiraID, Name: "jira"}, CredentialType: "oauth_authorization_code", SharedAccessToken: "jira-token"},
			{ServerWorkload: ServerInfo{ID: slackID, Name: "slack"}, CredentialType: "oauth_authorization_code", SharedAccessToken: "slack-token"},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, nil)
	clients := mapClientFactory{"jira": jiraClient, "slack": slackClient}
	handler := NewMethodHandler(rightAuth, clients, nil, nil)

	result, err := handler.HandleMethod(context.Background(), &RequestContext{UserID: "alice@corp.com", Scope: testScope(tenantID)}, "resources/list", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected map result")
	}
	resources, ok := resp["resources"].([]MCPResource)
	if !ok {
		t.Fatal("expected resources array")
	}
	if len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(resources))
	}

	// Resources should NOT have server-name prefixes (unlike tools)
	uris := map[string]bool{}
	for _, r := range resources {
		uris[r.URI] = true
	}
	if !uris["file:///jira/boards.json"] {
		t.Error("missing jira resource")
	}
	if !uris["file:///slack/channels.json"] {
		t.Error("missing slack resource")
	}
}

func TestResourcesRead_RoutesToCorrectServer(t *testing.T) {
	jiraID := uuid.New()
	tenantID := uuid.New()

	jiraClient := &mockMCPClient{
		resources: []MCPResource{
			{URI: "file:///jira/board.json"},
		},
	}

	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{ServerWorkload: ServerInfo{ID: jiraID, Name: "jira"}, CredentialType: "oauth_authorization_code", SharedAccessToken: "jira-token"},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, nil)
	clients := mapClientFactory{"jira": jiraClient}
	handler := NewMethodHandler(rightAuth, clients, nil, nil)

	params := json.RawMessage(`{"uri": "file:///jira/board.json"}`)
	result, err := handler.HandleMethod(context.Background(), &RequestContext{UserID: "alice@corp.com", Scope: testScope(tenantID)}, "resources/read", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, ok := result.(*MCPResourceContent)
	if !ok {
		t.Fatal("expected *MCPResourceContent result")
	}
	if content.URI != "file:///jira/board.json" {
		t.Errorf("expected URI 'file:///jira/board.json', got '%s'", content.URI)
	}
	if jiraClient.lastToken != "jira-token" {
		t.Errorf("expected jira-token, got '%s'", jiraClient.lastToken)
	}
}

func TestResourcesRead_NotFound_Error(t *testing.T) {
	jiraID := uuid.New()
	tenantID := uuid.New()

	jiraClient := &mockMCPClient{
		resources: []MCPResource{}, // No resources — ReadResource returns nil
	}

	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{ServerWorkload: ServerInfo{ID: jiraID, Name: "jira"}, CredentialType: "oauth_authorization_code", SharedAccessToken: "jira-token"},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, nil)
	clients := mapClientFactory{"jira": jiraClient}
	handler := NewMethodHandler(rightAuth, clients, nil, nil)

	params := json.RawMessage(`{"uri": "file:///nonexistent.json"}`)
	_, err := handler.HandleMethod(context.Background(), &RequestContext{UserID: "alice@corp.com", Scope: testScope(tenantID)}, "resources/read", params)
	if err == nil {
		t.Fatal("expected error for resource not found")
	}
}

type recordingRepo struct {
	mu             sync.Mutex
	securityEvents []*events.SecurityEvent
}

func (r *recordingRepo) InsertSecurityEvent(_ context.Context, event *events.SecurityEvent) error {
	r.mu.Lock()
	r.securityEvents = append(r.securityEvents, event)
	r.mu.Unlock()
	return nil
}

func (r *recordingRepo) InsertWorkloadEventBatch(_ context.Context, _ []events.WorkloadEvent) error {
	return nil
}

func (r *recordingRepo) InsertAuditEvent(_ context.Context, _ *events.AuditEvent) error {
	return nil
}

func (r *recordingRepo) QuerySecurityEvents(_ context.Context, _ uuid.UUID, _ events.QueryOptions) ([]events.SecurityEvent, int, error) {
	return nil, 0, nil
}

func (r *recordingRepo) QueryWorkloadEvents(_ context.Context, _ uuid.UUID, _ events.QueryOptions) ([]events.WorkloadEvent, int, error) {
	return nil, 0, nil
}

func (r *recordingRepo) QueryAuditEvents(_ context.Context, _ uuid.UUID, _ events.QueryOptions) ([]events.AuditEvent, int, error) {
	return nil, 0, nil
}

func TestEmitRightSideAuthEvent_CancelledRequestContext_StillEmits(t *testing.T) {
	serverID := uuid.New()
	tenantID := uuid.New()

	client := &mockMCPClient{callResult: map[string]any{"ok": true}}
	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{ServerWorkload: ServerInfo{ID: serverID, Name: "Test Server"}, CredentialType: "api_key"},
		},
	}
	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, &mockDecrypter{token: "secret"})
	clients := mapClientFactory{"Test Server": client}

	repo := &recordingRepo{}
	bus := events.NewEventBus(repo, nil)
	handler := NewMethodHandler(rightAuth, clients, bus, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	prefix := toolPrefix(ServerInfo{ID: serverID, Name: "Test Server"})
	params := json.RawMessage(fmt.Sprintf(`{"name": %q, "arguments": {}}`, prefix+"__do_thing"))
	_, _ = handler.HandleMethod(ctx, &RequestContext{UserID: "user@test.com", Scope: testScope(tenantID)}, "tools/call", params)

	time.Sleep(200 * time.Millisecond)

	repo.mu.Lock()
	count := len(repo.securityEvents)
	repo.mu.Unlock()

	if count < 1 {
		t.Fatalf("expected at least 1 security event emitted despite cancelled context, got %d", count)
	}

	repo.mu.Lock()
	ev := repo.securityEvents[0]
	repo.mu.Unlock()

	if ev.Source != "mcp_gateway" {
		t.Errorf("expected source 'mcp_gateway', got %q", ev.Source)
	}
	if !ev.Allowed {
		t.Errorf("expected allowed=true, got false")
	}
	if ev.ServerWorkload != "Test Server" {
		t.Errorf("expected server workload 'Test Server', got %q", ev.ServerWorkload)
	}
}
