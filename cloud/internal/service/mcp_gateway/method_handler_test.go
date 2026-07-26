package mcp_gateway

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

type mockMCPClient struct {
	tools      []MCPTool
	resources  []MCPResource
	callResult any
	lastToken  string
	lastTool   string
}

func (c *mockMCPClient) ListTools(_ context.Context, token string) ([]MCPTool, error) {
	c.lastToken = token
	return c.tools, nil
}

func (c *mockMCPClient) CallTool(_ context.Context, token string, toolName string, _ json.RawMessage) (any, error) {
	c.lastToken = token
	c.lastTool = toolName
	return c.callResult, nil
}

func (c *mockMCPClient) ListResources(_ context.Context, token string) ([]MCPResource, error) {
	c.lastToken = token
	return c.resources, nil
}

func (c *mockMCPClient) ReadResource(_ context.Context, token string, uri string) (*MCPResourceContent, error) {
	c.lastToken = token
	for _, r := range c.resources {
		if r.URI == uri {
			return &MCPResourceContent{URI: uri, Text: "content"}, nil
		}
	}
	return nil, nil
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

	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), "https://gw.dipolehq.com/")
	clients := map[string]MCPServerClient{"jira": jiraClient, "slack": slackClient}
	handler := NewMethodHandler(rightAuth, clients, tenantID, nil)

	result, err := handler.HandleMethod(&RequestContext{UserID: "alice@corp.com"}, "tools/list", nil)
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
	if !names["jira__create_issue"] {
		t.Error("missing jira__create_issue")
	}
	if !names["jira__search_issues"] {
		t.Error("missing jira__search_issues")
	}
	if !names["slack__send_message"] {
		t.Error("missing slack__send_message")
	}
	if !names["slack__list_channels"] {
		t.Error("missing slack__list_channels")
	}

	// Verify annotations preserved
	for _, tool := range tools {
		if tool.Name == "jira__search_issues" {
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

	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), "https://gw.dipolehq.com/")
	clients := map[string]MCPServerClient{"jira": jiraClient, "github": githubClient}
	handler := NewMethodHandler(rightAuth, clients, tenantID, nil)

	// Call jira tool
	params := json.RawMessage(`{"name": "jira__create_issue", "arguments": {"project": "PROJ"}}`)
	_, err := handler.HandleMethod(&RequestContext{UserID: "alice@corp.com"}, "tools/call", params)
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
	params = json.RawMessage(`{"name": "github__create_pr", "arguments": {"title": "fix"}}`)
	_, err = handler.HandleMethod(&RequestContext{UserID: "alice@corp.com"}, "tools/call", params)
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
	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), "https://gw.dipolehq.com/")
	handler := NewMethodHandler(rightAuth, map[string]MCPServerClient{}, tenantID, nil)

	params := json.RawMessage(`{"name": "unknown__tool", "arguments": {}}`)
	_, err := handler.HandleMethod(&RequestContext{UserID: "alice@corp.com"}, "tools/call", params)
	if err == nil {
		t.Fatal("expected error for unknown server prefix")
	}
}

func TestToolsCall_NoPrefixSeparator_Error(t *testing.T) {
	tenantID := uuid.New()
	policyFinder := &mockRightSidePolicyFinder{}
	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), "https://gw.dipolehq.com/")
	handler := NewMethodHandler(rightAuth, map[string]MCPServerClient{}, tenantID, nil)

	params := json.RawMessage(`{"name": "no-prefix-tool", "arguments": {}}`)
	_, err := handler.HandleMethod(&RequestContext{UserID: "alice@corp.com"}, "tools/call", params)
	if err == nil {
		t.Fatal("expected error for tool name without prefix")
	}
}

func TestPing_RespondsDirect(t *testing.T) {
	tenantID := uuid.New()
	policyFinder := &mockRightSidePolicyFinder{}
	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), "https://gw.dipolehq.com/")
	handler := NewMethodHandler(rightAuth, map[string]MCPServerClient{}, tenantID, nil)

	result, err := handler.HandleMethod(&RequestContext{UserID: "alice@corp.com"}, "ping", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for ping")
	}
}
