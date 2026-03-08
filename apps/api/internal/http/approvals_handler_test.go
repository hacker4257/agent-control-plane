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

type approvalsStoreFake struct {
	approvals      []repo.ApprovalRecord
	approval       repo.ApprovalRecord
	listErr        error
	getErr         error
	decisionErr    error
	decisionResult repo.ApprovalRecord
	decisionInput  repo.ApprovalDecisionInput
	listStatusArg  string
	listLimitArg   int
	listOffsetArg  int
}

func (s *approvalsStoreFake) InsertEvent(ctx context.Context, e repo.EventRecord) error { return nil }
func (s *approvalsStoreFake) UpsertSessionProjection(ctx context.Context, u repo.SessionProjectionUpdate) error {
	return nil
}
func (s *approvalsStoreFake) ListSessions(ctx context.Context, limit, offset int) ([]repo.SessionRecord, error) {
	return []repo.SessionRecord{}, nil
}
func (s *approvalsStoreFake) GetSessionByID(ctx context.Context, sessionID string) (repo.SessionRecord, error) {
	return repo.SessionRecord{}, nil
}
func (s *approvalsStoreFake) GetDashboardSummary(ctx context.Context) (repo.DashboardSummary, error) {
	return repo.DashboardSummary{}, nil
}
func (s *approvalsStoreFake) CreateApproval(ctx context.Context, a repo.ApprovalRecord) error {
	return nil
}
func (s *approvalsStoreFake) ListSessionEvents(ctx context.Context, sessionID string, limit, offset int) ([]repo.EventRecord, error) {
	return []repo.EventRecord{}, nil
}
func (s *approvalsStoreFake) ListApprovals(ctx context.Context, status string, limit, offset int) ([]repo.ApprovalRecord, error) {
	s.listStatusArg = status
	s.listLimitArg = limit
	s.listOffsetArg = offset
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.approvals, nil
}
func (s *approvalsStoreFake) GetApprovalByID(ctx context.Context, approvalID string) (repo.ApprovalRecord, error) {
	if s.getErr != nil {
		return repo.ApprovalRecord{}, s.getErr
	}
	return s.approval, nil
}
func (s *approvalsStoreFake) ApplyApprovalDecision(ctx context.Context, in repo.ApprovalDecisionInput) (repo.ApprovalRecord, error) {
	s.decisionInput = in
	if s.decisionErr != nil {
		return repo.ApprovalRecord{}, s.decisionErr
	}
	return s.decisionResult, nil
}
func (s *approvalsStoreFake) ListPolicyRules(ctx context.Context, enabledOnly bool, limit, offset int) ([]repo.PolicyRule, error) {
	return []repo.PolicyRule{}, nil
}
func (s *approvalsStoreFake) GetPolicyRuleByID(ctx context.Context, policyID string) (repo.PolicyRule, error) {
	return repo.PolicyRule{}, repo.ErrNotFound
}
func (s *approvalsStoreFake) CreatePolicyRule(ctx context.Context, in repo.PolicyRule) (repo.PolicyRule, error) {
	return in, nil
}
func (s *approvalsStoreFake) UpdatePolicyRule(ctx context.Context, policyID string, in repo.PolicyRule) (repo.PolicyRule, error) {
	return repo.PolicyRule{}, repo.ErrNotFound
}
func (s *approvalsStoreFake) SetPolicyEnabled(ctx context.Context, policyID string, enabled bool) error {
	return nil
}

func TestListApprovalsFromStore(t *testing.T) {
	now := time.Now().UTC()
	store := &approvalsStoreFake{approvals: []repo.ApprovalRecord{{
		ApprovalID:    "appr_1",
		SessionID:     "sess_1",
		Status:        "pending",
		Action:        "push",
		Tool:          "github",
		Resource:      "repo:org/api-service/branch:main",
		TriggerReason: "Main branch write requires approval",
		RequestedAt:   now,
	}}}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/approvals?status=pending&page=1&page_size=20", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if store.listStatusArg != "pending" {
		t.Fatalf("expected status arg pending, got %s", store.listStatusArg)
	}
	if store.listLimitArg != 20 {
		t.Fatalf("expected limit 20, got %d", store.listLimitArg)
	}
	if store.listOffsetArg != 0 {
		t.Fatalf("expected offset 0, got %d", store.listOffsetArg)
	}
}

func TestGetApprovalByIDFromStore(t *testing.T) {
	now := time.Now().UTC()
	store := &approvalsStoreFake{approval: repo.ApprovalRecord{
		ApprovalID:    "appr_2",
		SessionID:     "sess_2",
		Status:        "pending",
		Action:        "push",
		Tool:          "github",
		Resource:      "repo:org/api-service/branch:main",
		TriggerReason: "Main branch write requires approval",
		RequestedAt:   now,
	}}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/approvals/appr_2", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestGetApprovalNotFound(t *testing.T) {
	store := &approvalsStoreFake{getErr: repo.ErrNotFound}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/approvals/missing", nil))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestApprovalDecisionApprove(t *testing.T) {
	now := time.Now().UTC()
	store := &approvalsStoreFake{decisionResult: repo.ApprovalRecord{
		ApprovalID:      "appr_3",
		SessionID:       "sess_3",
		Status:          "approved",
		DecidedAt:       &now,
		ApproverID:      "security.lead",
		DecisionComment: "approved for emergency patch",
	}}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	payload := map[string]string{
		"decision":         "approve",
		"approver_id":      "security.lead",
		"decision_comment": "approved for emergency patch",
	}
	body, _ := json.Marshal(payload)

	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/approvals/appr_3/decision", bytes.NewReader(body)))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if store.decisionInput.ApprovalID != "appr_3" {
		t.Fatalf("expected approval_id appr_3, got %s", store.decisionInput.ApprovalID)
	}
	if store.decisionInput.Decision != "approve" {
		t.Fatalf("expected decision approve, got %s", store.decisionInput.Decision)
	}
}

func TestApprovalDecisionBadJSON(t *testing.T) {
	store := &approvalsStoreFake{}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/approvals/appr_1/decision", bytes.NewBufferString("{")))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestApprovalDecisionInvalidDecision(t *testing.T) {
	store := &approvalsStoreFake{}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	payload := map[string]string{"decision": "unknown"}
	body, _ := json.Marshal(payload)

	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/approvals/appr_1/decision", bytes.NewReader(body)))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestApprovalDecisionNotFound(t *testing.T) {
	store := &approvalsStoreFake{decisionErr: repo.ErrNotFound}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	payload := map[string]string{"decision": "approve"}
	body, _ := json.Marshal(payload)

	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/approvals/appr_missing/decision", bytes.NewReader(body)))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestListApprovalsStoreError(t *testing.T) {
	store := &approvalsStoreFake{listErr: errors.New("db error")}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/approvals", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}
