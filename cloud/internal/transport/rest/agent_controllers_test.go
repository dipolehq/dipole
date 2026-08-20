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
)

// fakeAgentControllerRepo is an in-memory AgentControllerRepository for
// testing the REST handlers without a database.
type fakeAgentControllerRepo struct {
	controllers map[uuid.UUID]*domain.AgentController
	updated     []domain.AgentController
}

func newFakeAgentControllerRepo(cs ...*domain.AgentController) *fakeAgentControllerRepo {
	f := &fakeAgentControllerRepo{controllers: make(map[uuid.UUID]*domain.AgentController)}
	for _, c := range cs {
		f.controllers[c.ID] = c
	}
	return f
}

func (f *fakeAgentControllerRepo) Create(_ context.Context, _ uuid.UUID, ac *domain.AgentController) error {
	if ac.ID == uuid.Nil {
		ac.ID = uuid.New()
	}
	f.controllers[ac.ID] = ac
	return nil
}
func (f *fakeAgentControllerRepo) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.AgentController, error) {
	return f.controllers[id], nil
}
func (f *fakeAgentControllerRepo) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.AgentController, int, error) {
	return nil, 0, nil
}
func (f *fakeAgentControllerRepo) Update(_ context.Context, _ uuid.UUID, ac *domain.AgentController) error {
	f.updated = append(f.updated, *ac)
	f.controllers[ac.ID] = ac
	return nil
}
func (f *fakeAgentControllerRepo) Delete(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	delete(f.controllers, id)
	return nil
}
func (f *fakeAgentControllerRepo) ConsumeDeviceCode(_ context.Context, _ uuid.UUID, _ string, _ []byte) (*domain.AgentController, error) {
	return nil, nil
}

// fakeTrustProviderRepoForAgentController is a minimal tenant-scoped
// TrustProviderRepository fake: GetByID only returns a provider when the
// tenantID matches, mirroring the real repo's "WHERE tenant_id = $1 AND id
// = $2" scoping.
type fakeTrustProviderRepoForAgentController struct {
	providers map[uuid.UUID]*domain.TrustProvider // keyed by provider ID
}

func newFakeTrustProviderRepoForAgentController(ps ...*domain.TrustProvider) *fakeTrustProviderRepoForAgentController {
	f := &fakeTrustProviderRepoForAgentController{providers: make(map[uuid.UUID]*domain.TrustProvider)}
	for _, p := range ps {
		f.providers[p.ID] = p
	}
	return f
}

func (f *fakeTrustProviderRepoForAgentController) Create(_ context.Context, _ uuid.UUID, _ *domain.TrustProvider) error {
	return nil
}
func (f *fakeTrustProviderRepoForAgentController) GetByID(_ context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.TrustProvider, error) {
	tp, ok := f.providers[id]
	if !ok || tp.TenantID != tenantID {
		return nil, nil
	}
	return tp, nil
}
func (f *fakeTrustProviderRepoForAgentController) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.TrustProvider, int, error) {
	return nil, 0, nil
}
func (f *fakeTrustProviderRepoForAgentController) Update(_ context.Context, _ uuid.UUID, _ *domain.TrustProvider) error {
	return nil
}
func (f *fakeTrustProviderRepoForAgentController) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}
func (f *fakeTrustProviderRepoForAgentController) CountInactive(_ context.Context, _ uuid.UUID) int {
	return 0
}

func updateAgentControllerRequest(tenantID, controllerID uuid.UUID, body any) *http.Request {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/agent-controllers/"+controllerID.String(), bytes.NewReader(b))
	req.SetPathValue("id", controllerID.String())
	return req.WithContext(withTestTenant(req.Context(), tenantID))
}

func TestUpdateAgentController_CrossTenantTrustProvider_Rejected(t *testing.T) {
	tenantID := uuid.New()
	otherTenantID := uuid.New()
	controllerID := uuid.New()
	foreignTrustProviderID := uuid.New()

	acRepo := newFakeAgentControllerRepo(&domain.AgentController{ID: controllerID, TenantID: tenantID, Name: "controller-1"})
	tpRepo := newFakeTrustProviderRepoForAgentController(&domain.TrustProvider{ID: foreignTrustProviderID, TenantID: otherTenantID})
	h := &APIHandlers{agentControllers: acRepo, trustProviders: tpRepo}

	req := updateAgentControllerRequest(tenantID, controllerID, map[string]any{
		"name":            "controller-1",
		"trustProviderId": foreignTrustProviderID,
	})
	w := httptest.NewRecorder()

	h.UpdateAgentController(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 when referencing another tenant's trust provider, got %d: %s", w.Code, w.Body.String())
	}
	if len(acRepo.updated) != 0 {
		t.Error("expected no update to be persisted when the trust provider reference is rejected")
	}
}

func TestUpdateAgentController_SameTenantTrustProvider_Allowed(t *testing.T) {
	tenantID := uuid.New()
	controllerID := uuid.New()
	trustProviderID := uuid.New()

	acRepo := newFakeAgentControllerRepo(&domain.AgentController{ID: controllerID, TenantID: tenantID, Name: "controller-1"})
	tpRepo := newFakeTrustProviderRepoForAgentController(&domain.TrustProvider{ID: trustProviderID, TenantID: tenantID})
	h := &APIHandlers{agentControllers: acRepo, trustProviders: tpRepo}

	req := updateAgentControllerRequest(tenantID, controllerID, map[string]any{
		"name":            "controller-1",
		"trustProviderId": trustProviderID,
	})
	w := httptest.NewRecorder()

	h.UpdateAgentController(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when referencing own tenant's trust provider, got %d: %s", w.Code, w.Body.String())
	}
	if len(acRepo.updated) != 1 || acRepo.updated[0].TrustProviderID == nil || *acRepo.updated[0].TrustProviderID != trustProviderID {
		t.Errorf("expected the controller to be updated with trust provider %s, got %v", trustProviderID, acRepo.updated)
	}
}

func TestUpdateAgentController_NonexistentTrustProvider_Rejected(t *testing.T) {
	tenantID := uuid.New()
	controllerID := uuid.New()
	missingTrustProviderID := uuid.New()

	acRepo := newFakeAgentControllerRepo(&domain.AgentController{ID: controllerID, TenantID: tenantID, Name: "controller-1"})
	tpRepo := newFakeTrustProviderRepoForAgentController()
	h := &APIHandlers{agentControllers: acRepo, trustProviders: tpRepo}

	req := updateAgentControllerRequest(tenantID, controllerID, map[string]any{
		"name":            "controller-1",
		"trustProviderId": missingTrustProviderID,
	})
	w := httptest.NewRecorder()

	h.UpdateAgentController(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for a nonexistent trust provider, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAgentController_NoTrustProvider_Allowed(t *testing.T) {
	tenantID := uuid.New()
	controllerID := uuid.New()

	acRepo := newFakeAgentControllerRepo(&domain.AgentController{ID: controllerID, TenantID: tenantID, Name: "controller-1"})
	tpRepo := newFakeTrustProviderRepoForAgentController()
	h := &APIHandlers{agentControllers: acRepo, trustProviders: tpRepo}

	req := updateAgentControllerRequest(tenantID, controllerID, map[string]any{
		"name": "controller-1",
	})
	w := httptest.NewRecorder()

	h.UpdateAgentController(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when no trust provider is referenced, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateAgentController_CrossTenantTrustProvider_Rejected(t *testing.T) {
	tenantID := uuid.New()
	otherTenantID := uuid.New()
	foreignTrustProviderID := uuid.New()

	acRepo := newFakeAgentControllerRepo()
	tpRepo := newFakeTrustProviderRepoForAgentController(&domain.TrustProvider{ID: foreignTrustProviderID, TenantID: otherTenantID})
	h := &APIHandlers{
		agentControllers: acRepo,
		trustProviders:   tpRepo,
		registrationSvc:  fakeControllerCreatorForAgentController{},
	}

	b, _ := json.Marshal(map[string]any{
		"name":            "controller-1",
		"trustProviderId": foreignTrustProviderID,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent-controllers", bytes.NewReader(b))
	req = req.WithContext(withTestTenant(req.Context(), tenantID))
	w := httptest.NewRecorder()

	h.CreateAgentController(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 when creating with another tenant's trust provider, got %d: %s", w.Code, w.Body.String())
	}
	if len(acRepo.updated) != 0 {
		t.Error("expected no update to be persisted when the trust provider reference is rejected")
	}
}

// fakeControllerCreatorForAgentController is a minimal ControllerCreator
// fake that mints a bare AgentController without touching a database.
type fakeControllerCreatorForAgentController struct{}

func (fakeControllerCreatorForAgentController) CreateController(_ context.Context, tenantID uuid.UUID, name string) (*domain.AgentController, error) {
	return &domain.AgentController{ID: uuid.New(), TenantID: tenantID, Name: name}, nil
}
func (fakeControllerCreatorForAgentController) RegenerateDeviceCode(_ context.Context, _, _ uuid.UUID) (string, error) {
	return "", nil
}
