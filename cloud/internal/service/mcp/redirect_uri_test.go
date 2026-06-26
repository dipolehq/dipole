package mcp

import "testing"

func TestMatchRedirectURI_ExactMatch(t *testing.T) {
	if !MatchRedirectURI("https://claude.ai/api/mcp/auth_callback", "https://claude.ai/api/mcp/auth_callback") {
		t.Error("exact match should pass")
	}
}

func TestMatchRedirectURI_NonLocalhostMismatch(t *testing.T) {
	if MatchRedirectURI("https://claude.ai/api/mcp/auth_callback", "https://other.ai/api/mcp/auth_callback") {
		t.Error("different non-localhost hosts should not match")
	}
}

func TestMatchRedirectURI_LocalhostPortIgnored(t *testing.T) {
	if !MatchRedirectURI("http://localhost:8080/oauth/callback", "http://localhost:7777/oauth/callback") {
		t.Error("localhost with different ports should match")
	}
}

func TestMatchRedirectURI_Localhost127PortIgnored(t *testing.T) {
	if !MatchRedirectURI("http://127.0.0.1:3000/oauth/callback", "http://127.0.0.1:7777/oauth/callback") {
		t.Error("127.0.0.1 with different ports should match")
	}
}

func TestMatchRedirectURI_LocalhostDifferentPath_Fails(t *testing.T) {
	if MatchRedirectURI("http://localhost:8080/other/path", "http://localhost:7777/oauth/callback") {
		t.Error("localhost with different paths should not match")
	}
}

func TestMatchRedirectURI_LocalhostDifferentScheme_Fails(t *testing.T) {
	if MatchRedirectURI("https://localhost:8080/oauth/callback", "http://localhost:7777/oauth/callback") {
		t.Error("localhost with different schemes should not match")
	}
}

func TestMatchRedirectURI_NonLocalhostDifferentPort_Fails(t *testing.T) {
	if MatchRedirectURI("https://example.com:8080/callback", "https://example.com:443/callback") {
		t.Error("non-localhost with different ports should not match")
	}
}

func TestMatchRedirectURIInList_Found(t *testing.T) {
	list := []string{"http://localhost:7777/oauth/callback", "https://claude.ai/api/mcp/auth_callback"}
	if !MatchRedirectURIInList("http://localhost:9999/oauth/callback", list) {
		t.Error("should match localhost entry in list regardless of port")
	}
}

func TestMatchRedirectURIInList_NotFound(t *testing.T) {
	list := []string{"http://localhost:7777/oauth/callback"}
	if MatchRedirectURIInList("https://unknown.com/callback", list) {
		t.Error("should not match when URI is not in list")
	}
}
