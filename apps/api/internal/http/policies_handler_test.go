package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-control-plane/apps/api/internal/repo"
)

type policiesStoreFake struct {
	items             []repo.PolicyRule
	getItem           repo.PolicyRule
	createResult      repo.PolicyRule
	updateResult      repo.PolicyRule
	listErr           error
	getErr            error
	createErr         error
	updateErr         error
	setEnabledErr     error
	evalErr           error
	lastEnabledOnly   bool
	lastLimit         int
	lastOffset        int
	lastCreateInput   repo.PolicyRule
	lastUpdateID      string
	lastUpdateInput   repo.PolicyRule
	lastSetEnabledID  string
	lastSetEnabledVal bool
}

func (s *policiesStoreFake) InsertEvent(ctx context.Context, e repo.EventRecord) error { return nil }
func (s *policiesStoreFake) UpsertSessionProjection(ctx context.Context, u repo.SessionProjectionUpdate) error {
	return nil
}
func (s *policiesStoreFake) ListSessions(ctx context.Context, limit, offset int) ([]repo.SessionRecord, error) {
	return []repo.SessionRecord{}, nil
}
func (s *policiesStoreFake) GetSessionByID(ctx context.Context, sessionID string) (repo.SessionRecord, error) {
	return repo.SessionRecord{}, nil
}
func (s *policiesStoreFake) GetDashboardSummary(ctx context.Context) (repo.DashboardSummary, error) {
	return repo.DashboardSummary{}, nil
}
func (s *policiesStoreFake) CreateApproval(ctx context.Context, a repo.ApprovalRecord) error {
	return nil
}
func (s *policiesStoreFake) ListSessionEvents(ctx context.Context, sessionID string, limit, offset int) ([]repo.EventRecord, error) {
	return []repo.EventRecord{}, nil
}
func (s *policiesStoreFake) ListApprovals(ctx context.Context, status string, limit, offset int) ([]repo.ApprovalRecord, error) {
	return []repo.ApprovalRecord{}, nil
}
func (s *policiesStoreFake) GetApprovalByID(ctx context.Context, approvalID string) (repo.ApprovalRecord, error) {
	return repo.ApprovalRecord{}, repo.ErrNotFound
}
func (s *policiesStoreFake) ApplyApprovalDecision(ctx context.Context, in repo.ApprovalDecisionInput) (repo.ApprovalRecord, error) {
	return repo.ApprovalRecord{}, nil
}

func (s *policiesStoreFake) ListPolicyRules(ctx context.Context, enabledOnly bool, limit, offset int) ([]repo.PolicyRule, error) {
	s.lastEnabledOnly = enabledOnly
	s.lastLimit = limit
	s.lastOffset = offset
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.items, nil
}

func (s *policiesStoreFake) GetPolicyRuleByID(ctx context.Context, policyID string) (repo.PolicyRule, error) {
	if s.getErr != nil {
		return repo.PolicyRule{}, s.getErr
	}
	return s.getItem, nil
}

func (s *policiesStoreFake) CreatePolicyRule(ctx context.Context, in repo.PolicyRule) (repo.PolicyRule, error) {
	s.lastCreateInput = in
	if s.createErr != nil {
		return repo.PolicyRule{}, s.createErr
	}
	if s.createResult.PolicyID == "" {
		in.PolicyID = "pol_created"
		in.CreatedAt = time.Now().UTC()
		in.UpdatedAt = in.CreatedAt
		return in, nil
	}
	return s.createResult, nil
}

func (s *policiesStoreFake) UpdatePolicyRule(ctx context.Context, policyID string, in repo.PolicyRule) (repo.PolicyRule, error) {
	s.lastUpdateID = policyID
	s.lastUpdateInput = in
	if s.updateErr != nil {
		return repo.PolicyRule{}, s.updateErr
	}
	if s.updateResult.PolicyID == "" {
		in.PolicyID = policyID
		in.UpdatedAt = time.Now().UTC()
		return in, nil
	}
	return s.updateResult, nil
}

func (s *policiesStoreFake) SetPolicyEnabled(ctx context.Context, policyID string, enabled bool) error {
	s.lastSetEnabledID = policyID
	s.lastSetEnabledVal = enabled
	return s.setEnabledErr
}

func TestListPoliciesFromStore(t *testing.T) {
	now := time.Now().UTC()
	store := &policiesStoreFake{items: []repo.PolicyRule{{
		PolicyID:      "pol_1",
		Name:          "Protect main",
		Decision:      "REQUIRE_APPROVAL",
		Priority:      10,
		Enabled:       true,
		ConditionExpr: map[string]interface{}{"action": "push"},
		CreatedAt:     now,
		UpdatedAt:     now,
	}}}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/policies?enabled=true&page=1&page_size=20", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !store.lastEnabledOnly {
		t.Fatal("expected enabledOnly=true")
	}
	if store.lastLimit != 20 || store.lastOffset != 0 {
		t.Fatalf("expected limit=20 offset=0, got limit=%d offset=%d", store.lastLimit, store.lastOffset)
	}
}

func TestCreatePolicyFromStore(t *testing.T) {
	now := time.Now().UTC()
	store := &policiesStoreFake{createResult: repo.PolicyRule{
		PolicyID:      "pol_2",
		Name:          "Block dangerous shell",
		Decision:      "BLOCK",
		Priority:      5,
		Enabled:       true,
		ConditionExpr: map[string]interface{}{"action": "exec"},
		CreatedAt:     now,
		UpdatedAt:     now,
	}}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	payload := map[string]interface{}{
		"name":           "Block dangerous shell",
		"decision":       "BLOCK",
		"priority":       5,
		"enabled":        true,
		"condition_expr": map[string]interface{}{"action": "exec"},
	}
	body, _ := json.Marshal(payload)

	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/policies", bytes.NewReader(body)))

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
	if store.lastCreateInput.Name != "Block dangerous shell" {
		t.Fatalf("expected create input captured")
	}
}

func TestCreatePolicyBadJSON(t *testing.T) {
	store := &policiesStoreFake{}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/policies", bytes.NewBufferString("{")))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreatePolicyMissingName(t *testing.T) {
	store := &policiesStoreFake{}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	payload := map[string]interface{}{"decision": "ALLOW"}
	body, _ := json.Marshal(payload)

	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/policies", bytes.NewReader(body)))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestUpdatePolicyNotFound(t *testing.T) {
	store := &policiesStoreFake{updateErr: repo.ErrNotFound}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	payload := map[string]interface{}{
		"name":           "Updated policy",
		"decision":       "ALLOW",
		"condition_expr": map[string]interface{}{},
	}
	body, _ := json.Marshal(payload)

	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodPatch, "/api/v1/policies/pol_missing", bytes.NewReader(body)))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestEnablePolicyNotFound(t *testing.T) {
	store := &policiesStoreFake{setEnabledErr: repo.ErrNotFound}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/policies/pol_missing/enable", nil))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestDisablePolicyFromStore(t *testing.T) {
	store := &policiesStoreFake{}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/policies/pol_1/disable", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if store.lastSetEnabledID != "pol_1" || store.lastSetEnabledVal {
		t.Fatalf("expected disable call for pol_1")
	}
}

func TestListPoliciesStoreError(t *testing.T) {
	store := &policiesStoreFake{listErr: errors.New("db error")}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/policies", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestPolicyEvaluatePreviewEndpoint(t *testing.T) {
	store := &policiesStoreFake{items: []repo.PolicyRule{{
		PolicyID:         "pol_1",
		Decision:         "REQUIRE_APPROVAL",
		ScopeTool:        "github",
		ScopeEnvironment: "prod",
		ScopeResourcePat: "branch:main",
		ConditionExpr:    map[string]interface{}{"action": "push"},
		Priority:         10,
		Enabled:          true,
	}}}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	payload := map[string]string{
		"tool":        "github",
		"action":      "push",
		"resource":    "repo:org/api-service/branch:main",
		"environment": "prod",
	}
	body, _ := json.Marshal(payload)

	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/policies/evaluate-preview", bytes.NewReader(body)))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["decision"] != "REQUIRE_APPROVAL" {
		t.Fatalf("expected REQUIRE_APPROVAL, got %v", resp["decision"])
	}
}
