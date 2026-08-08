/*
mcp_tracing_handler.go serves the MCP Authorization Tracing endpoint.

Returns the current in-memory ring buffer of recent MCP Gateway authorization
decisions. The frontend polls this endpoint every 2-3 seconds for near-real-time
diagnostic data. Entries are ephemeral and not persisted.
*/
package rest

import (
	"net/http"

	gw "github.com/dipolehq/dipole/cloud/internal/service/mcp_gateway"
	"github.com/google/uuid"
)

type MCPTracingHandler struct {
	bus *gw.TracingBus
}

func NewMCPTracingHandler(bus *gw.TracingBus) *MCPTracingHandler {
	return &MCPTracingHandler{bus: bus}
}

func (h *MCPTracingHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/events/mcp-tracing", h.HandleGet)
}

func (h *MCPTracingHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	// Scoped to the authenticated caller's tenant. The buffer is process-wide and
	// shared by every tenant, and entries carry end-user identities, so an
	// unscoped read would disclose one tenant's users to another's administrators.
	tenantID := TenantIDFromContext(r.Context())
	if tenantID == uuid.Nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	entries := h.bus.Recent(tenantID)
	if entries == nil {
		entries = []gw.TracingEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"entries": entries,
	})
}
