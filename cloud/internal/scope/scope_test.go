package scope

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestWithScope_ValidScope_AttachesToContext(t *testing.T) {
	tenantID := uuid.New()
	rsID := uuid.New()

	ctx := WithScope(context.Background(), &RequestScope{
		TenantID:      tenantID,
		ResourceSetID: rsID,
	})

	s := FromContext(ctx)
	if s == nil {
		t.Fatal("expected scope to be set, got nil")
	}
	if s.TenantID != tenantID {
		t.Errorf("TenantID = %v, want %v", s.TenantID, tenantID)
	}
	if s.ResourceSetID != rsID {
		t.Errorf("ResourceSetID = %v, want %v", s.ResourceSetID, rsID)
	}
}

func TestWithScope_NilScope_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when WithScope called with nil scope")
		}
	}()
	WithScope(context.Background(), nil)
}

func TestWithScope_NilTenantID_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when WithScope called with nil TenantID")
		}
	}()
	WithScope(context.Background(), &RequestScope{
		TenantID:      uuid.Nil,
		ResourceSetID: uuid.New(),
	})
}

func TestFromContext_NoScope_ReturnsNil(t *testing.T) {
	s := FromContext(context.Background())
	if s != nil {
		t.Errorf("expected nil when no scope set, got %+v", s)
	}
}

func TestTenantID_WithScope_ReturnsTenantID(t *testing.T) {
	tenantID := uuid.New()
	ctx := WithScope(context.Background(), &RequestScope{
		TenantID:      tenantID,
		ResourceSetID: uuid.New(),
	})

	got := TenantID(ctx)
	if got != tenantID {
		t.Errorf("TenantID(ctx) = %v, want %v", got, tenantID)
	}
}

func TestTenantID_NoScope_ReturnsNil(t *testing.T) {
	got := TenantID(context.Background())
	if got != uuid.Nil {
		t.Errorf("TenantID(ctx) = %v, want uuid.Nil", got)
	}
}

func TestResourceSetID_WithScope_ReturnsResourceSetID(t *testing.T) {
	rsID := uuid.New()
	ctx := WithScope(context.Background(), &RequestScope{
		TenantID:      uuid.New(),
		ResourceSetID: rsID,
	})

	got := ResourceSetID(ctx)
	if got != rsID {
		t.Errorf("ResourceSetID(ctx) = %v, want %v", got, rsID)
	}
}

func TestResourceSetID_NoScope_ReturnsNil(t *testing.T) {
	got := ResourceSetID(context.Background())
	if got != uuid.Nil {
		t.Errorf("ResourceSetID(ctx) = %v, want uuid.Nil", got)
	}
}

func TestResolveResourceSetID_NonNil_ReturnsProvided(t *testing.T) {
	id := uuid.New()
	got := ResolveResourceSetID(&id)
	if got != id {
		t.Errorf("ResolveResourceSetID(&id) = %v, want %v", got, id)
	}
}

func TestResolveResourceSetID_Nil_ReturnsDefault(t *testing.T) {
	got := ResolveResourceSetID(nil)
	if got != DefaultResourceSetID {
		t.Errorf("ResolveResourceSetID(nil) = %v, want DefaultResourceSetID %v", got, DefaultResourceSetID)
	}
}

func TestDefaultResourceSetID_IsWellKnownUUID(t *testing.T) {
	expected := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")
	if DefaultResourceSetID != expected {
		t.Errorf("DefaultResourceSetID = %v, want %v", DefaultResourceSetID, expected)
	}
}
