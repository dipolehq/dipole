package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

func withTestTenant(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxTenantID, tenantID)
}

func decodeJSONResponse(t *testing.T, w *httptest.ResponseRecorder, dest any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), dest); err != nil {
		t.Fatalf("failed to decode response body %q: %v", w.Body.String(), err)
	}
}

func TestComplianceAttributeStatus(t *testing.T) {
	cases := []struct {
		name    string
		level   string
		present bool
		want    string
	}{
		{"optional, missing", "optional", false, "optional"},
		{"optional, present", "optional", true, "optional"},
		{"required, present -> compliant", "required", true, "compliant"},
		{"required, missing -> required (violation)", "required", false, "required"},
		{"recommended, present -> compliant", "recommended", true, "compliant"},
		{"recommended, missing -> recommended (violation)", "recommended", false, "recommended"},
		{"empty level treated as optional", "", false, "optional"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := complianceAttributeStatus(tc.level, tc.present)
			if got != tc.want {
				t.Errorf("complianceAttributeStatus(%q, %v) = %q, want %q", tc.level, tc.present, got, tc.want)
			}
		})
	}
}

// --- fakes for GetComplianceReport handler-level tests ---

type fakeGlobalPolicyComplianceStore struct {
	p   *domain.GlobalPolicyCompliance
	err error
}

func (f *fakeGlobalPolicyComplianceStore) Get(ctx context.Context, tenantID uuid.UUID) (*domain.GlobalPolicyCompliance, error) {
	return f.p, f.err
}
func (f *fakeGlobalPolicyComplianceStore) Upsert(ctx context.Context, tenantID uuid.UUID, p *domain.GlobalPolicyCompliance) error {
	return nil
}

type fakeComplianceChecker struct {
	policies []domain.AccessPolicyWithRelations
	err      error
}

func (f *fakeComplianceChecker) ListActiveWithRelations(ctx context.Context, tenantID uuid.UUID) ([]domain.AccessPolicyWithRelations, error) {
	return f.policies, f.err
}

type fakeResourceSetRepoForCompliance struct {
	sets []domain.ResourceSet
}

func (f *fakeResourceSetRepoForCompliance) Create(ctx context.Context, tenantID uuid.UUID, rs *domain.ResourceSet) error {
	return nil
}
func (f *fakeResourceSetRepoForCompliance) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.ResourceSet, error) {
	return nil, nil
}
func (f *fakeResourceSetRepoForCompliance) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.ResourceSet, int, error) {
	return f.sets, len(f.sets), nil
}
func (f *fakeResourceSetRepoForCompliance) Update(ctx context.Context, tenantID uuid.UUID, rs *domain.ResourceSet) error {
	return nil
}
func (f *fakeResourceSetRepoForCompliance) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return nil
}
func (f *fakeResourceSetRepoForCompliance) GetRoleIDs(ctx context.Context, tenantID, resourceSetID uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}
func (f *fakeResourceSetRepoForCompliance) GetResourceCounts(ctx context.Context, tenantID uuid.UUID, resourceSetIDs []uuid.UUID) (map[uuid.UUID]domain.ResourceSetCounts, error) {
	return nil, nil
}

func TestGetComplianceReport_PerPolicyStatusReflectsActualRelations(t *testing.T) {
	tenantID := uuid.New()
	compliantPolicyID := uuid.New()
	violatingPolicyID := uuid.New()
	defaultRSID := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")

	h := &APIHandlers{
		globalPolicyCompliance: &fakeGlobalPolicyComplianceStore{
			p: &domain.GlobalPolicyCompliance{
				TenantID:          tenantID,
				APTrustProvider:   "required",
				APAccessCondition: "recommended",
			},
		},
		complianceChecker: &fakeComplianceChecker{
			policies: []domain.AccessPolicyWithRelations{
				{
					Policy:         domain.AccessPolicy{ID: compliantPolicyID, Name: "compliant-policy", ResourceSetID: &defaultRSID},
					ClientWorkload: domain.ClientWorkload{Name: "client-a"},
					ServerWorkload: domain.ServerWorkload{Name: "server-a"},
					TrustProviders: []domain.TrustProvider{{Name: "tp-a"}},
					// No access conditions — but the setting is only "recommended", not "required".
				},
				{
					Policy:         domain.AccessPolicy{ID: violatingPolicyID, Name: "violating-policy", ResourceSetID: &defaultRSID},
					ClientWorkload: domain.ClientWorkload{Name: "client-b"},
					ServerWorkload: domain.ServerWorkload{Name: "server-b"},
					// No trust provider at all — setting is "required", so this must be flagged.
				},
			},
		},
		resourceSets: &fakeResourceSetRepoForCompliance{
			sets: []domain.ResourceSet{{ID: defaultRSID, Name: "Default"}},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/global-policy-compliance/report", nil)
	req = req.WithContext(withTestTenant(req.Context(), tenantID))
	w := httptest.NewRecorder()

	h.GetComplianceReport(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Items []struct {
			ID              uuid.UUID `json:"id"`
			TrustProvider   string    `json:"trustProvider"`
			AccessCondition string    `json:"accessCondition"`
			ResourceSet     string    `json:"resourceSet"`
		} `json:"items"`
		Total int `json:"total"`
	}
	decodeJSONResponse(t, w, &resp)

	if resp.Total != 2 {
		t.Fatalf("expected 2 items, got %d", resp.Total)
	}

	byID := map[uuid.UUID]struct {
		TrustProvider   string
		AccessCondition string
		ResourceSet     string
	}{}
	for _, item := range resp.Items {
		byID[item.ID] = struct {
			TrustProvider   string
			AccessCondition string
			ResourceSet     string
		}{item.TrustProvider, item.AccessCondition, item.ResourceSet}
	}

	compliant, ok := byID[compliantPolicyID]
	if !ok {
		t.Fatal("compliant policy missing from response")
	}
	if compliant.TrustProvider != "compliant" {
		t.Errorf("expected compliant policy's trustProvider to be 'compliant' (it has one, required is satisfied), got %q", compliant.TrustProvider)
	}
	if compliant.AccessCondition != "recommended" {
		t.Errorf("expected compliant policy's accessCondition to be 'recommended' (missing, but only recommended not required), got %q", compliant.AccessCondition)
	}
	if compliant.ResourceSet != "Default" {
		t.Errorf("expected resource set name 'Default', got %q (raw UUID leak regression)", compliant.ResourceSet)
	}

	violating, ok := byID[violatingPolicyID]
	if !ok {
		t.Fatal("violating policy missing from response")
	}
	if violating.TrustProvider != "required" {
		t.Errorf("expected violating policy's trustProvider to be 'required' (missing a required Trust Provider), got %q", violating.TrustProvider)
	}
}

func TestGetComplianceReport_NoGlobalPolicySet_ReturnsEmptyItems(t *testing.T) {
	h := &APIHandlers{
		globalPolicyCompliance: &fakeGlobalPolicyComplianceStore{p: nil},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/global-policy-compliance/report", nil)
	req = req.WithContext(withTestTenant(req.Context(), uuid.New()))
	w := httptest.NewRecorder()

	h.GetComplianceReport(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Items []any `json:"items"`
		Total int   `json:"total"`
	}
	decodeJSONResponse(t, w, &resp)
	if resp.Total != 0 || len(resp.Items) != 0 {
		t.Errorf("expected empty items when no global policy is configured, got total=%d items=%d", resp.Total, len(resp.Items))
	}
}

func TestGetComplianceReport_ResourceSetLookupFails_StillReturnsReport(t *testing.T) {
	tenantID := uuid.New()
	policyID := uuid.New()
	defaultRSID := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")

	h := &APIHandlers{
		globalPolicyCompliance: &fakeGlobalPolicyComplianceStore{
			p: &domain.GlobalPolicyCompliance{TenantID: tenantID, APTrustProvider: "optional", APAccessCondition: "optional"},
		},
		complianceChecker: &fakeComplianceChecker{
			policies: []domain.AccessPolicyWithRelations{
				{
					Policy:         domain.AccessPolicy{ID: policyID, Name: "p1", ResourceSetID: &defaultRSID},
					ClientWorkload: domain.ClientWorkload{Name: "c1"},
					ServerWorkload: domain.ServerWorkload{Name: "s1"},
				},
			},
		},
		resourceSets: &erroringResourceSetRepo{},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/global-policy-compliance/report", nil)
	req = req.WithContext(withTestTenant(req.Context(), tenantID))
	w := httptest.NewRecorder()

	h.GetComplianceReport(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 even when resource set lookup fails, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Items []struct {
			ResourceSet string `json:"resourceSet"`
		} `json:"items"`
		Total int `json:"total"`
	}
	decodeJSONResponse(t, w, &resp)
	if resp.Total != 1 {
		t.Fatalf("expected 1 item even with resource set lookup failure, got %d", resp.Total)
	}
	if resp.Items[0].ResourceSet != "Default" {
		t.Errorf("expected fallback resource set name 'Default', got %q", resp.Items[0].ResourceSet)
	}
}

// erroringResourceSetRepo simulates a resource set lookup failure (e.g. a
// transient DB error) to verify GetComplianceReport degrades gracefully
// (falls back to "Default" per row) instead of failing the whole report.
type erroringResourceSetRepo struct {
	fakeResourceSetRepoForCompliance
}

func (f *erroringResourceSetRepo) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.ResourceSet, int, error) {
	return nil, 0, errResourceSetLookupFailed
}

var errResourceSetLookupFailed = errString("resource set lookup failed")

type errString string

func (e errString) Error() string { return string(e) }
