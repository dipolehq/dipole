package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/scope"
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
	h := &APIHandlers{resourceSets: repo}

	req := deleteResourceSetRequest(scope.DefaultResourceSetID)
	w := httptest.NewRecorder()

	h.DeleteResourceSet(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when deleting the Default Resource Set, got %d: %s", w.Code, w.Body.String())
	}
	if len(repo.deleted) != 0 {
		t.Error("expected the Default Resource Set to NOT be deleted")
	}
}

func TestDeleteResourceSet_NonDefaultResourceSet_Allowed(t *testing.T) {
	customID := uuid.New()
	customRS := &domain.ResourceSet{ID: customID, Name: "Staging"}
	repo := newFakeResourceSetRepoForDelete(customRS)
	h := &APIHandlers{resourceSets: repo}

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
