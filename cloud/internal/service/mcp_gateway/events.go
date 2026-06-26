/*
events.go emits workload events for MCP Gateway traffic.

GatewayEventEmitter records mcp.request and mcp.response events for every
proxied MCP interaction. Each event includes userId from the left-side token
to enable per-user audit trails and user-level filtering in the admin UI.
*/
package mcp_gateway

import (
	"time"

	"github.com/google/uuid"
)

type MCPWorkloadEvent struct {
	Meta        EventMeta        `json:"meta"`
	Workload    EventWorkload    `json:"workload"`
	Outcome     EventOutcome     `json:"outcome"`
	Application EventApplication `json:"application"`
}

type EventMeta struct {
	EventType    string `json:"eventType"`
	ConnectionID string `json:"connectionId"`
	Severity     string `json:"severity"`
	Timestamp    string `json:"timestamp"`
}

type EventWorkload struct {
	Client EventActor `json:"client"`
	Server EventActor `json:"server"`
}

type EventActor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type EventOutcome struct {
	Result string `json:"result"`
}

type EventApplication struct {
	MCP EventMCP `json:"mcp"`
}

type EventMCP struct {
	HTTPMethod       string `json:"httpMethod"`
	URI              string `json:"uri"`
	UserID           string `json:"userId"`
	HTTPResponseCode int    `json:"httpResponseCode,omitempty"`
}

type EventSink interface {
	Emit(event *MCPWorkloadEvent)
}

type GatewayEventEmitter struct {
	sink EventSink
}

func NewGatewayEventEmitter(sink EventSink) *GatewayEventEmitter {
	return &GatewayEventEmitter{sink: sink}
}

func (e *GatewayEventEmitter) EmitRequest(userID string, clientName string, serverName string, serverID uuid.UUID) {
	if e.sink == nil {
		return
	}
	e.sink.Emit(&MCPWorkloadEvent{
		Meta: EventMeta{
			EventType:    "mcp.request",
			ConnectionID: uuid.New().String(),
			Severity:     "Info",
			Timestamp:    time.Now().UTC().Format(time.RFC3339Nano),
		},
		Workload: EventWorkload{
			Client: EventActor{Name: clientName},
			Server: EventActor{ID: serverID.String(), Name: serverName},
		},
		Outcome: EventOutcome{Result: "Modified"},
		Application: EventApplication{
			MCP: EventMCP{
				HTTPMethod: "POST",
				URI:        "/mcp",
				UserID:     userID,
			},
		},
	})
}

func (e *GatewayEventEmitter) EmitResponse(userID string, clientName string, serverName string, serverID uuid.UUID, httpResponseCode int) {
	if e.sink == nil {
		return
	}
	e.sink.Emit(&MCPWorkloadEvent{
		Meta: EventMeta{
			EventType:    "mcp.response",
			ConnectionID: uuid.New().String(),
			Severity:     "Info",
			Timestamp:    time.Now().UTC().Format(time.RFC3339Nano),
		},
		Workload: EventWorkload{
			Client: EventActor{Name: clientName},
			Server: EventActor{ID: serverID.String(), Name: serverName},
		},
		Outcome: EventOutcome{Result: "Modified"},
		Application: EventApplication{
			MCP: EventMCP{
				HTTPMethod:       "POST",
				URI:              "/mcp",
				UserID:           userID,
				HTTPResponseCode: httpResponseCode,
			},
		},
	})
}
