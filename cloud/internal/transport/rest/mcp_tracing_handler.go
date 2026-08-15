/*
mcp_tracing_handler.go serves the MCP Authorization Tracing endpoint.

Returns the current in-memory ring buffer of recent MCP Gateway authorization
decisions from the centralized EventBus. The frontend polls this endpoint every
2-3 seconds for near-real-time diagnostic data. Entries are ephemeral and not
persisted.
*/
package rest

import (
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/events"
	"github.com/google/uuid"
)

type MCPTracingHandler struct {
	bus *events.EventBus
}

func NewMCPTracingHandler(bus *events.EventBus) *MCPTracingHandler {
	return &MCPTracingHandler{bus: bus}
}

func (h *MCPTracingHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/events/mcp-tracing", h.HandleGet)
}

func (h *MCPTracingHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	if tenantID == uuid.Nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	entries := h.bus.RecentTraces(tenantID)
	if entries == nil {
		entries = []events.TraceEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"entries": entries,
	})
}
