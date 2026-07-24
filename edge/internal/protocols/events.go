/*
events.go provides workload event construction for the Agent Proxy.

BuildWorkloadEvent constructs a protobuf WorkloadEvent from connection
metadata. Sensitive HTTP headers (Authorization, Set-Cookie) are redacted
from the JSON payload before the event leaves the proxy, providing
defense-in-depth alongside Cloud-side redaction.
*/
package protocols

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	pb "github.com/dipolehq/dipole/gen/dipole/v1"
)

type EventData struct {
	EventType        string
	ConnectionID     string
	PolicyID         string
	ClientWorkloadID string
	ServerWorkloadID string
	ResourceSetID    string
	Outcome          string
	TargetHost       string
	TargetPort       int
	SourceIP         string
	SourcePort       int
	HTTPMethod       string
	HTTPUri          string
	HTTPStatus       int
	Headers          []map[string]string
}

type eventPayload struct {
	Network     *eventNetwork     `json:"network"`
	Application *eventApplication `json:"application,omitempty"`
}

type eventNetwork struct {
	ClientWorkloadIP   string `json:"clientWorkloadIP"`
	ClientWorkloadPort int    `json:"clientWorkloadPort"`
	TransportProtocol  string `json:"transportProtocol"`
	ServerWorkloadHost string `json:"serverWorkloadHost"`
	ServerWorkloadPort int    `json:"serverWorkloadPort"`
	ProxyPort          int    `json:"proxyPort"`
}

type eventApplication struct {
	HTTP *eventHTTP `json:"http,omitempty"`
}

type eventHTTP struct {
	Method  string           `json:"httpMethod,omitempty"`
	URI     string           `json:"uri,omitempty"`
	Status  int              `json:"statusCode,omitempty"`
	Headers []map[string]any `json:"headers,omitempty"`
}

func BuildWorkloadEvent(data *EventData) *pb.WorkloadEvent {
	severity := "Info"
	if data.Outcome == "Error" {
		severity = "Error"
	}

	payload := buildPayload(data)
	redactedPayload := redactPayloadHeaders(payload)

	return &pb.WorkloadEvent{
		EventId:          uuid.New().String(),
		ConnectionId:     data.ConnectionID,
		EventType:        data.EventType,
		PolicyId:         data.PolicyID,
		ClientWorkloadId: data.ClientWorkloadID,
		ServerWorkloadId: data.ServerWorkloadID,
		ResourceSetId:    data.ResourceSetID,
		Outcome:          data.Outcome,
		Severity:         severity,
		Timestamp:        time.Now().Unix(),
		Payload:          redactedPayload,
	}
}

func buildPayload(data *EventData) []byte {
	p := &eventPayload{
		Network: &eventNetwork{
			ClientWorkloadIP:   data.SourceIP,
			ClientWorkloadPort: data.SourcePort,
			TransportProtocol:  "TCP",
			ServerWorkloadHost: data.TargetHost,
			ServerWorkloadPort: data.TargetPort,
			ProxyPort:          15001,
		},
	}

	if data.HTTPMethod != "" || data.HTTPUri != "" || len(data.Headers) > 0 {
		headers := make([]map[string]any, 0, len(data.Headers))
		for _, h := range data.Headers {
			for k, v := range h {
				headers = append(headers, map[string]any{k: v})
			}
		}

		p.Application = &eventApplication{
			HTTP: &eventHTTP{
				Method:  data.HTTPMethod,
				URI:     data.HTTPUri,
				Status:  data.HTTPStatus,
				Headers: headers,
			},
		}
	}

	result, err := json.Marshal(p)
	if err != nil {
		return []byte("{}")
	}
	return result
}

func redactPayloadHeaders(payload []byte) []byte {
	if len(payload) == 0 {
		return payload
	}

	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return payload
	}

	app, ok := data["application"].(map[string]any)
	if !ok {
		return payload
	}

	modified := false

	if httpData, ok := app["http"].(map[string]any); ok {
		if headers, ok := httpData["headers"].([]any); ok {
			httpData["headers"] = redactSensitiveHeaders(headers)
			modified = true
		}
	}

	if !modified {
		return payload
	}

	result, err := json.Marshal(data)
	if err != nil {
		return payload
	}
	return result
}

func redactSensitiveHeaders(headers []any) []any {
	redacted := make([]any, 0, len(headers))
	for _, h := range headers {
		header, ok := h.(map[string]any)
		if !ok {
			redacted = append(redacted, h)
			continue
		}

		shouldRedact := false
		for key := range header {
			lower := strings.ToLower(key)
			if lower == "authorization" || lower == "set-cookie" {
				shouldRedact = true
				break
			}
		}

		if shouldRedact {
			redacted = append(redacted, map[string]any{"[REDACTED]": "[REDACTED]"})
		} else {
			redacted = append(redacted, header)
		}
	}
	return redacted
}
