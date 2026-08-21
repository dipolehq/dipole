package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/scope"
	"github.com/dipolehq/dipole/cloud/internal/service"
	"github.com/dipolehq/dipole/cloud/internal/service/resources"
)

type fakeResourceSetRepoForDelete struct {
	sets    map[uuid.UUID]*domain.ResourceSet
	deleted []uuid.UUID
}

func newFakeResourceSetRepoForDelete(sets ...*domain.ResourceSet) *fakeResourceSetRepoForDelete {
	f := &fakeResourceSetRepoForDelete{sets: make(map[uuid.UUID]*domain.ResourceSet)}
	for _, s := range sets {
		f.sets[s.ID] = s
	}
	return f
}

func (f *fakeResourceSetRepoForDelete) Create(_ context.Context, _ uuid.UUID, _ *domain.ResourceSet) error {
	return nil
}
func (f *fakeResourceSetRepoForDelete) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.ResourceSet, error) {
	return f.sets[id], nil
}
func (f *fakeResourceSetRepoForDelete) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.ResourceSet, int, error) {
	return nil, 0, nil
}
func (f *fakeResourceSetRepoForDelete) Update(_ context.Context, _ uuid.UUID, _ *domain.ResourceSet) error {
	return nil
}
func (f *fakeResourceSetRepoForDelete) Delete(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	f.deleted = append(f.deleted, id)
	return nil
}
func (f *fakeResourceSetRepoForDelete) GetRoleIDs(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}
func (f *fakeResourceSetRepoForDelete) GetResourceCounts(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (map[uuid.UUID]domain.ResourceSetCounts, error) {
	return nil, nil
}

func deleteResourceSetRequest(id uuid.UUID) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/resource-sets/"+id.String(), nil)
	req.SetPathValue("id", id.String())
	return req.WithContext(withTestTenant(req.Context(), uuid.New()))
}

func TestDeleteResourceSet_DefaultResourceSet_Rejected(t *testing.T) {
	defaultRS := &domain.ResourceSet{ID: scope.DefaultResourceSetID, Name: "Default"}
	repo := newFakeResourceSetRepoForDelete(defaultRS)
	h := &APIHandlers{resourceSets: resources.NewResourceSetService(repo)}

	req := deleteResourceSetRequest(scope.DefaultResourceSetID)
	w := httptest.NewRecorder()

	h.DeleteResourceSet(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 when deleting the Default Resource Set, got %d: %s", w.Code, w.Body.String())
	}
	if len(repo.deleted) != 0 {
		t.Error("expected the Default Resource Set to NOT be deleted")
	}
}

func TestDeleteResourceSet_NonDefaultResourceSet_Allowed(t *testing.T) {
	customID := uuid.New()
	customRS := &domain.ResourceSet{ID: customID, Name: "Staging"}
	repo := newFakeResourceSetRepoForDelete(customRS)
	h := &APIHandlers{resourceSets: resources.NewResourceSetService(repo)}

	req := deleteResourceSetRequest(customID)
	w := httptest.NewRecorder()

	h.DeleteResourceSet(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 when deleting a non-default Resource Set, got %d: %s", w.Code, w.Body.String())
	}
	if len(repo.deleted) != 1 || repo.deleted[0] != customID {
		t.Errorf("expected %s to be deleted, got %v", customID, repo.deleted)
	}
}

func TestToResourceSetResponse_MarksDefaultResourceSet(t *testing.T) {
	defaultRS := &domain.ResourceSet{ID: scope.DefaultResourceSetID, Name: "Default"}
	resp := toResourceSetResponse(defaultRS)

	if !resp.IsDefault {
		t.Error("expected isDefault=true for the well-known Default Resource Set ID")
	}
}

func TestToResourceSetResponse_NonDefaultResourceSet(t *testing.T) {
	customRS := &domain.ResourceSet{ID: uuid.New(), Name: "Staging"}
	resp := toResourceSetResponse(customRS)

	if resp.IsDefault {
		t.Error("expected isDefault=false for a non-default Resource Set")
	}
}

// --- Plan limit enforcement on resource set creation ---
//
// Regression coverage for a real gap found while auditing plan_limits.go:
// ResourceType defines ResourceResourceSets with real per-plan limits (and
// DBResourceCounter knows how to count them), but no handler ever called
// planLimiter.Check for it — CreateResourceSet had no limit check at all,
// so a starter-plan tenant could create unlimited resource sets despite
// the documented limit of 1. Fixed in CreateResourceSet; these two tests
// prove both sides: at-limit is rejected, under-limit is allowed.

type fakeTenantRepoForResourceSetLimit struct {
	plan string
}

func (f *fakeTenantRepoForResourceSetLimit) Create(context.Context, *domain.Tenant) error { return nil }
func (f *fakeTenantRepoForResourceSetLimit) GetByID(_ context.Context, id uuid.UUID) (*domain.Tenant, error) {
	return &domain.Tenant{ID: id, Plan: f.plan}, nil
}
func (f *fakeTenantRepoForResourceSetLimit) GetBySubdomain(context.Context, string) (*domain.Tenant, error) {
	return nil, nil
}
func (f *fakeTenantRepoForResourceSetLimit) GetByAPITokenHash(context.Context, string) (*domain.Tenant, error) {
	return nil, nil
}
func (f *fakeTenantRepoForResourceSetLimit) List(context.Context, domain.ListOptions) ([]domain.Tenant, int, error) {
	return nil, 0, nil
}
func (f *fakeTenantRepoForResourceSetLimit) Update(context.Context, *domain.Tenant) error { return nil }
func (f *fakeTenantRepoForResourceSetLimit) Delete(context.Context, uuid.UUID) error      { return nil }

type fakeResourceCounterForResourceSetLimit struct {
	count int
}

func (f *fakeResourceCounterForResourceSetLimit) CountResource(context.Context, uuid.UUID, service.ResourceType) (int, error) {
	return f.count, nil
}

func createResourceSetRequest(tenantID uuid.UUID, name string) *http.Request {
	body, _ := json.Marshal(map[string]string{"name": name})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/resource-sets", bytes.NewReader(body))
	return req.WithContext(withTestTenant(req.Context(), tenantID))
}

func TestCreateResourceSet_AtStarterPlanLimit_Rejected(t *testing.T) {
	tenantID := uuid.New()
	// Starter plan allows 1 resource set; a count of 1 means already at
	// capacity, so creating a second one must be rejected.
	limiter := service.NewPlanLimiter(
		&fakeTenantRepoForResourceSetLimit{plan: "starter"},
		&fakeResourceCounterForResourceSetLimit{count: 1},
	)
	svc := resources.NewResourceSetService(newFakeResourceSetRepoForDelete())
	svc.SetLimiter(limiter)
	h := &APIHandlers{resourceSets: svc}

	req := createResourceSetRequest(tenantID, "Staging")
	w := httptest.NewRecorder()

	h.CreateResourceSet(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 when at the starter plan's resource set limit, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateResourceSet_UnderPlanLimit_Allowed(t *testing.T) {
	tenantID := uuid.New()
	// Starter plan allows 1 resource set; a count of 0 means there's room
	// for the tenant's very first one.
	limiter := service.NewPlanLimiter(
		&fakeTenantRepoForResourceSetLimit{plan: "starter"},
		&fakeResourceCounterForResourceSetLimit{count: 0},
	)
	svc := resources.NewResourceSetService(newFakeResourceSetRepoForDelete())
	svc.SetLimiter(limiter)
	h := &APIHandlers{resourceSets: svc}

	req := createResourceSetRequest(tenantID, "Staging")
	w := httptest.NewRecorder()

	h.CreateResourceSet(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201 when under the plan's resource set limit, got %d: %s", w.Code, w.Body.String())
	}
}
