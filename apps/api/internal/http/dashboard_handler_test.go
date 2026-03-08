package httpx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-control-plane/apps/api/internal/repo"
)

type dashboardStoreFake struct {
	summary repo.DashboardSummary
	err     error
}

func (d *dashboardStoreFake) InsertEvent(ctx context.Context, e repo.EventRecord) error { return nil }
func (d *dashboardStoreFake) UpsertSessionProjection(ctx context.Context, u repo.SessionProjectionUpdate) error {
	return nil
}
func (d *dashboardStoreFake) ListSessions(ctx context.Context, limit, offset int) ([]repo.SessionRecord, error) {
	return []repo.SessionRecord{}, nil
}
func (d *dashboardStoreFake) GetSessionByID(ctx context.Context, sessionID string) (repo.SessionRecord, error) {
	return repo.SessionRecord{}, nil
}
func (d *dashboardStoreFake) ListApprovals(ctx context.Context, status string, limit, offset int) ([]repo.ApprovalRecord, error) {
	return []repo.ApprovalRecord{}, nil
}
func (d *dashboardStoreFake) GetApprovalByID(ctx context.Context, approvalID string) (repo.ApprovalRecord, error) {
	return repo.ApprovalRecord{}, nil
}
func (d *dashboardStoreFake) CreateApproval(ctx context.Context, a repo.ApprovalRecord) error {
	return nil
}
func (d *dashboardStoreFake) ApplyApprovalDecision(ctx context.Context, in repo.ApprovalDecisionInput) (repo.ApprovalRecord, error) {
	return repo.ApprovalRecord{}, nil
}
func (d *dashboardStoreFake) ListPolicyRules(ctx context.Context, enabledOnly bool, limit, offset int) ([]repo.PolicyRule, error) {
	return []repo.PolicyRule{}, nil
}
func (d *dashboardStoreFake) GetPolicyRuleByID(ctx context.Context, policyID string) (repo.PolicyRule, error) {
	return repo.PolicyRule{}, repo.ErrNotFound
}
func (d *dashboardStoreFake) CreatePolicyRule(ctx context.Context, in repo.PolicyRule) (repo.PolicyRule, error) {
	return in, nil
}
func (d *dashboardStoreFake) UpdatePolicyRule(ctx context.Context, policyID string, in repo.PolicyRule) (repo.PolicyRule, error) {
	return repo.PolicyRule{}, repo.ErrNotFound
}
func (d *dashboardStoreFake) SetPolicyEnabled(ctx context.Context, policyID string, enabled bool) error {
	return nil
}
func (d *dashboardStoreFake) GetDashboardSummary(ctx context.Context) (repo.DashboardSummary, error) {
	if d.err != nil {
		return repo.DashboardSummary{}, d.err
	}
	return d.summary, nil
}
func (d *dashboardStoreFake) ListSessionEvents(ctx context.Context, sessionID string, limit, offset int) ([]repo.EventRecord, error) {
	return []repo.EventRecord{}, nil
}

func TestDashboardSummaryFromStore(t *testing.T) {
	store := &dashboardStoreFake{summary: repo.DashboardSummary{
		SessionsCount:         128,
		PendingApprovalsCount: 17,
		BlockedActionsCount:   9,
		PolicyHitsCount:       34,
	}}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestDashboardSummaryStoreError(t *testing.T) {
	store := &dashboardStoreFake{err: errors.New("db error")}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}
