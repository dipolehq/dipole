/*
Routes:

	GET /api/v1/dashboard
*/
package rest

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/service/reporting"
)

type dashboardService interface {
	GetStats(ctx context.Context, tenantID uuid.UUID, timeRange string) (*reporting.DashboardStats, error)
}

type DashboardHandler struct {
	svc dashboardService
}

func NewDashboardHandler(svc dashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

func (h *DashboardHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/dashboard", h.GetStats)
}

func (h *DashboardHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	timeRange := r.URL.Query().Get("timeRange")
	if timeRange == "" {
		timeRange = "6h"
	}
	stats, err := h.svc.GetStats(r.Context(), tenantID, timeRange)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load dashboard stats"})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
