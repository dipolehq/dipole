package protocols

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildWorkloadEvent_ProducesValidProto(t *testing.T) {
	data := &EventData{
		EventType:        "http.request",
		ConnectionID:     "550e8400-e29b-41d4-a716-446655440000",
		PolicyID:         "661e8400-e29b-41d4-a716-446655440001",
		ClientWorkloadID: "772e8400-e29b-41d4-a716-446655440002",
		ServerWorkloadID: "883e8400-e29b-41d4-a716-446655440003",
		ResourceSetID:    "ffffffff-ffff-ffff-ffff-ffffffffffff",
		Outcome:          "Modified",
		TargetHost:       "api.stripe.com",
		TargetPort:       443,
		SourceIP:         "10.0.0.5",
		SourcePort:       54321,
		HTTPMethod:       "POST",
		HTTPUri:          "/v1/charges",
		HTTPStatus:       200,
		Headers: []map[string]string{
			{"Content-Type": "application/json"},
			{"Host": "api.stripe.com"},
		},
	}

	event := BuildWorkloadEvent(data)

	if event.EventId == "" {
		t.Error("event_id should not be empty")
	}
	if event.ConnectionId != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("connection_id = %q, want UUID", event.ConnectionId)
	}
	if event.EventType != "http.request" {
		t.Errorf("event_type = %q, want http.request", event.EventType)
	}
	if event.PolicyId != "661e8400-e29b-41d4-a716-446655440001" {
		t.Errorf("policy_id = %q", event.PolicyId)
	}
	if event.ClientWorkloadId != "772e8400-e29b-41d4-a716-446655440002" {
		t.Errorf("client_workload_id = %q", event.ClientWorkloadId)
	}
	if event.ServerWorkloadId != "883e8400-e29b-41d4-a716-446655440003" {
		t.Errorf("server_workload_id = %q", event.ServerWorkloadId)
	}
	if event.ResourceSetId != "ffffffff-ffff-ffff-ffff-ffffffffffff" {
		t.Errorf("resource_set_id = %q", event.ResourceSetId)
	}
	if event.Outcome != "Modified" {
		t.Errorf("outcome = %q, want Modified", event.Outcome)
	}
	if event.Severity != "Info" {
		t.Errorf("severity = %q, want Info", event.Severity)
	}
	if event.Timestamp == 0 {
		t.Error("timestamp should not be zero")
	}
	if len(event.Payload) == 0 {
		t.Error("payload should not be empty")
	}

	var payload map[string]any
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("payload is not valid JSON: %v", err)
	}

	network, ok := payload["network"].(map[string]any)
	if !ok {
		t.Fatal("missing network in payload")
	}
	if network["serverWorkloadHost"] != "api.stripe.com" {
		t.Errorf("serverWorkloadHost = %v", network["serverWorkloadHost"])
	}
	if network["clientWorkloadIP"] != "10.0.0.5" {
		t.Errorf("clientWorkloadIP = %v", network["clientWorkloadIP"])
	}
}

func TestBuildWorkloadEvent_ErrorOutcomeSetsSeverityError(t *testing.T) {
	data := &EventData{
		EventType:    "http.request",
		ConnectionID: "550e8400-e29b-41d4-a716-446655440000",
		Outcome:      "Error",
		TargetHost:   "api.example.com",
		TargetPort:   443,
		SourceIP:     "10.0.0.1",
		SourcePort:   12345,
	}

	event := BuildWorkloadEvent(data)

	if event.Severity != "Error" {
		t.Errorf("severity = %q, want Error", event.Severity)
	}
}

func TestBuildWorkloadEvent_RedactsAuthorizationHeader(t *testing.T) {
	data := &EventData{
		EventType:    "http.request",
		ConnectionID: "550e8400-e29b-41d4-a716-446655440000",
		Outcome:      "Modified",
		TargetHost:   "api.stripe.com",
		TargetPort:   443,
		SourceIP:     "10.0.0.5",
		SourcePort:   54321,
		HTTPMethod:   "GET",
		HTTPUri:      "/v1/charges",
		HTTPStatus:   200,
		Headers: []map[string]string{
			{"Content-Type": "application/json"},
			{"Authorization": "Bearer sk_live_secret_key"},
			{"Set-Cookie": "session=abc123"},
			{"Cookie": "session_id=s3cr3t_cookie_value"},
			{"X-API-Key": "sk_live_apikey123"},
			{"X-Request-Id": "req-456"},
		},
	}

	event := BuildWorkloadEvent(data)

	payloadStr := string(event.Payload)

	if strings.Contains(payloadStr, "sk_live_secret_key") {
		t.Error("Authorization header value should be redacted")
	}
	if strings.Contains(payloadStr, "Bearer sk_live") {
		t.Error("Authorization Bearer token should be redacted")
	}
	if strings.Contains(payloadStr, "session=abc123") {
		t.Error("Set-Cookie value should be redacted")
	}
	if strings.Contains(payloadStr, "s3cr3t_cookie_value") {
		t.Error("Cookie header value should be redacted")
	}
	if strings.Contains(payloadStr, "sk_live_apikey123") {
		t.Error("X-API-Key header value should be redacted")
	}
	if !strings.Contains(payloadStr, "[REDACTED]") {
		t.Error("expected [REDACTED] placeholder in payload")
	}
	if !strings.Contains(payloadStr, "application/json") {
		t.Error("Content-Type should be preserved")
	}
	if !strings.Contains(payloadStr, "req-456") {
		t.Error("X-Request-Id should be preserved")
	}
}

func TestBuildWorkloadEvent_NoHTTPData_OmitsApplication(t *testing.T) {
	data := &EventData{
		EventType:    "postgres.request",
		ConnectionID: "550e8400-e29b-41d4-a716-446655440000",
		Outcome:      "Modified",
		TargetHost:   "db.internal",
		TargetPort:   5432,
		SourceIP:     "10.0.0.5",
		SourcePort:   54321,
	}

	event := BuildWorkloadEvent(data)

	var payload map[string]any
	json.Unmarshal(event.Payload, &payload)

	if _, ok := payload["application"]; ok {
		t.Error("application section should be omitted for non-HTTP events")
	}
	if _, ok := payload["network"]; !ok {
		t.Error("network section should always be present")
	}
}
