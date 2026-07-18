package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestHealthEndpoint(t *testing.T) {
	h := &APIHandlers{}
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() == "" {
		t.Error("expected non-empty body")
	}
}

func TestParseListOptions_Defaults(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	opts := parseListOptions(req)

	if opts.Page != 1 {
		t.Errorf("expected page=1, got %d", opts.Page)
	}
	if opts.PerPage != 25 {
		t.Errorf("expected perPage=25, got %d", opts.PerPage)
	}
}

func TestParseListOptions_Cap100(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/test?perPage=500", nil)
	opts := parseListOptions(req)

	if opts.PerPage != 25 {
		t.Errorf("expected perPage=25 (>100 treated as invalid), got %d", opts.PerPage)
	}
}

func TestParseListOptions_ValidValues(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/test?page=3&perPage=50&orderBy=name&search=test", nil)
	opts := parseListOptions(req)

	if opts.Page != 3 {
		t.Errorf("expected page=3, got %d", opts.Page)
	}
	if opts.PerPage != 50 {
		t.Errorf("expected perPage=50, got %d", opts.PerPage)
	}
	if opts.OrderBy != "name" {
		t.Errorf("expected orderBy=name, got %s", opts.OrderBy)
	}
	if opts.Search != "test" {
		t.Errorf("expected search=test, got %s", opts.Search)
	}
}

func TestRateLimiter_AllowsWithinLimit(t *testing.T) {
	rl := NewRateLimiter(10, 5)
	tenantID := uuid.New()

	for i := 0; i < 10; i++ {
		if !rl.allow(tenantID, 10) {
			t.Fatalf("request %d should have been allowed", i)
		}
	}
}

func TestRateLimiter_DeniesOverLimit(t *testing.T) {
	rl := NewRateLimiter(5, 2)
	tenantID := uuid.New()

	for i := 0; i < 5; i++ {
		rl.allow(tenantID, 5)
	}

	if rl.allow(tenantID, 5) {
		t.Error("request over limit should be denied")
	}
}
