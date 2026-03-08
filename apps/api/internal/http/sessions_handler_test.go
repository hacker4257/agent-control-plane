package httpx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-control-plane/apps/api/internal/repo"
)

type sessionStoreFake struct {
	sessions []repo.SessionRecord
	session  repo.SessionRecord
	getErr   error
}

func (s *sessionStoreFake) InsertEvent(ctx context.Context, e repo.EventRecord) error { return nil }
func (s *sessionStoreFake) UpsertSessionProjection(ctx context.Context, u repo.SessionProjectionUpdate) error {
	return nil
}
func (s *sessionStoreFake) ListSessions(ctx context.Context, limit, offset int) ([]repo.SessionRecord, error) {
	return s.sessions, nil
}
func (s *sessionStoreFake) GetSessionByID(ctx context.Context, sessionID string) (repo.SessionRecord, error) {
	if s.getErr != nil {
		return repo.SessionRecord{}, s.getErr
	}
	return s.session, nil
}
func (s *sessionStoreFake) GetDashboardSummary(ctx context.Context) (repo.DashboardSummary, error) {
	return repo.DashboardSummary{}, nil
}
func (s *sessionStoreFake) CreateApproval(ctx context.Context, a repo.ApprovalRecord) error {
	return nil
}
func (s *sessionStoreFake) GetApprovalByID(ctx context.Context, approvalID string) (repo.ApprovalRecord, error) {
	return repo.ApprovalRecord{}, repo.ErrNotFound
}
func (s *sessionStoreFake) ApplyApprovalDecision(ctx context.Context, in repo.ApprovalDecisionInput) (repo.ApprovalRecord, error) {
	return repo.ApprovalRecord{}, nil
}
func (s *sessionStoreFake) ListApprovals(ctx context.Context, status string, limit, offset int) ([]repo.ApprovalRecord, error) {
	return []repo.ApprovalRecord{}, nil
}
func (s *sessionStoreFake) ListPolicyRules(ctx context.Context, enabledOnly bool, limit, offset int) ([]repo.PolicyRule, error) {
	return []repo.PolicyRule{}, nil
}
func (s *sessionStoreFake) GetPolicyRuleByID(ctx context.Context, policyID string) (repo.PolicyRule, error) {
	return repo.PolicyRule{}, repo.ErrNotFound
}
func (s *sessionStoreFake) CreatePolicyRule(ctx context.Context, in repo.PolicyRule) (repo.PolicyRule, error) {
	return in, nil
}
func (s *sessionStoreFake) UpdatePolicyRule(ctx context.Context, policyID string, in repo.PolicyRule) (repo.PolicyRule, error) {
	return repo.PolicyRule{}, repo.ErrNotFound
}
func (s *sessionStoreFake) SetPolicyEnabled(ctx context.Context, policyID string, enabled bool) error {
	return nil
}
func (s *sessionStoreFake) ListSessionEvents(ctx context.Context, sessionID string, limit, offset int) ([]repo.EventRecord, error) {
	return []repo.EventRecord{}, nil
}

func TestListSessionsFromStore(t *testing.T) {
	now := time.Now().UTC()
	store := &sessionStoreFake{sessions: []repo.SessionRecord{{
		SessionID:        "sess_1",
		Objective:        "Fix prod auth",
		AgentID:          "coding-agent-prod",
		Environment:      "prod",
		Status:           "approval_pending",
		RiskScore:        87,
		StartedAt:        now,
		UpdatedAt:        now,
		TouchedResources: []string{"repo:org/api-service/branch:main"},
	}}}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/sessions?page=1&page_size=20", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestGetSessionByIDFromStore(t *testing.T) {
	now := time.Now().UTC()
	store := &sessionStoreFake{session: repo.SessionRecord{
		SessionID:   "sess_2",
		Objective:   "Deploy patch",
		AgentID:     "ops-agent",
		Environment: "prod",
		Status:      "completed",
		RiskScore:   40,
		StartedAt:   now,
		UpdatedAt:   now,
	}}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/sessions/sess_2", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestGetSessionNotFound(t *testing.T) {
	store := &sessionStoreFake{getErr: errors.New("not found")}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/sessions/missing", nil))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}
