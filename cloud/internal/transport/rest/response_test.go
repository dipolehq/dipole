package rest

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dipolehq/dipole/cloud/internal/service"
)

func TestRespondDomainError_ForbiddenError_Returns403WithDetail(t *testing.T) {
	w := httptest.NewRecorder()
	err := &service.ForbiddenError{Resource: "trust_providers", RequiredPermission: "write"}

	respondDomainError(w, err)

	if w.Code != 403 {
		t.Fatalf("status = %d, want 403", w.Code)
	}

	var resp ErrorResponse
	if decodeErr := json.NewDecoder(w.Body).Decode(&resp); decodeErr != nil {
		t.Fatalf("decode response: %v", decodeErr)
	}
	if resp.Error.Code != "forbidden" {
		t.Errorf("Error.Code = %q, want %q", resp.Error.Code, "forbidden")
	}
	if resp.Error.Message == "" {
		t.Error("expected a non-empty message identifying the resource/permission that was denied")
	}
}

func TestRespondDomainError_PlainErrForbidden_Returns403(t *testing.T) {
	// A bare service.ErrForbidden (not wrapped in *ForbiddenError) must
	// still map to 403 with a generic message, not fall through to 500.
	w := httptest.NewRecorder()

	respondDomainError(w, service.ErrForbidden)

	if w.Code != 403 {
		t.Fatalf("status = %d, want 403", w.Code)
	}

	var resp ErrorResponse
	if decodeErr := json.NewDecoder(w.Body).Decode(&resp); decodeErr != nil {
		t.Fatalf("decode response: %v", decodeErr)
	}
	if resp.Error.Code != "forbidden" {
		t.Errorf("Error.Code = %q, want %q", resp.Error.Code, "forbidden")
	}
}
