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

type fakeStore struct {
	inserted          []repo.EventRecord
	events            []repo.EventRecord
	projectionUpdates []repo.SessionProjectionUpdate
	createdApprovals  []repo.ApprovalRecord
	policyRules       []repo.PolicyRule
	insertErr         error
	projectionErr     error
	approvalCreateErr error
	listErr           error
}

func (f *fakeStore) InsertEvent(ctx context.Context, e repo.EventRecord) error {
	if f.insertErr != nil {
		return f.insertErr
	}
	f.inserted = append(f.inserted, e)
	return nil
}

func (f *fakeStore) UpsertSessionProjection(ctx context.Context, u repo.SessionProjectionUpdate) error {
	if f.projectionErr != nil {
		return f.projectionErr
	}
	f.projectionUpdates = append(f.projectionUpdates, u)
	return nil
}

func (f *fakeStore) CreateApproval(ctx context.Context, a repo.ApprovalRecord) error {
	if f.approvalCreateErr != nil {
		return f.approvalCreateErr
	}
	f.createdApprovals = append(f.createdApprovals, a)
	return nil
}

func (f *fakeStore) ListSessions(ctx context.Context, limit, offset int) ([]repo.SessionRecord, error) {
	return []repo.SessionRecord{}, nil
}

func (f *fakeStore) GetSessionByID(ctx context.Context, sessionID string) (repo.SessionRecord, error) {
	return repo.SessionRecord{}, errors.New("not found")
}

func (f *fakeStore) GetDashboardSummary(ctx context.Context) (repo.DashboardSummary, error) {
	return repo.DashboardSummary{}, nil
}

func (f *fakeStore) GetApprovalByID(ctx context.Context, approvalID string) (repo.ApprovalRecord, error) {
	return repo.ApprovalRecord{}, repo.ErrNotFound
}

func (f *fakeStore) ApplyApprovalDecision(ctx context.Context, in repo.ApprovalDecisionInput) (repo.ApprovalRecord, error) {
	return repo.ApprovalRecord{}, nil
}

func (f *fakeStore) ListApprovals(ctx context.Context, status string, limit, offset int) ([]repo.ApprovalRecord, error) {
	return []repo.ApprovalRecord{}, nil
}

func (f *fakeStore) ListPolicyRules(ctx context.Context, enabledOnly bool, limit, offset int) ([]repo.PolicyRule, error) {
	return f.policyRules, nil
}

func (f *fakeStore) GetPolicyRuleByID(ctx context.Context, policyID string) (repo.PolicyRule, error) {
	return repo.PolicyRule{}, repo.ErrNotFound
}

func (f *fakeStore) CreatePolicyRule(ctx context.Context, in repo.PolicyRule) (repo.PolicyRule, error) {
	return in, nil
}

func (f *fakeStore) UpdatePolicyRule(ctx context.Context, policyID string, in repo.PolicyRule) (repo.PolicyRule, error) {
	return repo.PolicyRule{}, repo.ErrNotFound
}

func (f *fakeStore) SetPolicyEnabled(ctx context.Context, policyID string, enabled bool) error {
	return nil
}

func (f *fakeStore) ListSessionEvents(ctx context.Context, sessionID string, limit, offset int) ([]repo.EventRecord, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.events, nil
}

func TestPreflightPersistsEvent(t *testing.T) {
	store := &fakeStore{
		policyRules: []repo.PolicyRule{
			{
				PolicyID: "pol_test_github_main", ScopeTool: "github", ScopeEnvironment: "prod",
				ConditionExpr: map[string]interface{}{"action": "push", "resource_contains": "branch:main"},
				Decision: "REQUIRE_APPROVAL", Priority: 10, Enabled: true,
			},
		},
	}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	payload := preflightRequest{
		SessionID:     "sess_1",
		StepID:        "step_1",
		CorrelationID: "corr_1",
		AgentID:       "coding-agent-prod",
		Environment:   "prod",
		Tool:          "github",
		Action:        "push",
		Resource:      "repo:org/api-service/branch:main",
		InputSummary:  "push commit fix(auth)",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/preflight", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	handlePreflight(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if len(store.inserted) != 1 {
		t.Fatalf("expected 1 inserted event, got %d", len(store.inserted))
	}
	if store.inserted[0].EventType != "approval_requested" {
		t.Fatalf("expected approval_requested event type, got %s", store.inserted[0].EventType)
	}
	if len(store.projectionUpdates) != 1 {
		t.Fatalf("expected 1 projection update, got %d", len(store.projectionUpdates))
	}
	if store.projectionUpdates[0].Status != "approval_pending" {
		t.Fatalf("expected status approval_pending, got %s", store.projectionUpdates[0].Status)
	}
	if len(store.createdApprovals) != 1 {
		t.Fatalf("expected 1 created approval, got %d", len(store.createdApprovals))
	}
	if store.createdApprovals[0].Status != "pending" {
		t.Fatalf("expected pending approval status, got %s", store.createdApprovals[0].Status)
	}
	if store.createdApprovals[0].EventID == "" {
		t.Fatal("expected approval to be linked to event_id")
	}
}

func TestPostflightPersistsEvent(t *testing.T) {
	store := &fakeStore{}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	payload := postflightRequest{
		SessionID:     "sess_1",
		StepID:        "step_1",
		CorrelationID: "corr_1",
		AgentID:       "coding-agent-prod",
		Environment:   "prod",
		Tool:          "github",
		Action:        "create_pull_request",
		Resource:      "repo:org/api-service/branch:feature-x",
		Result:        "completed",
		OutputSummary: "created PR #4182",
		ArtifactRefs:  []string{"art_1"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/postflight", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	handlePostflight(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", rr.Code)
	}
	if len(store.inserted) != 1 {
		t.Fatalf("expected 1 inserted event, got %d", len(store.inserted))
	}
	if store.inserted[0].EventType != "tool_completed" {
		t.Fatalf("expected tool_completed event type, got %s", store.inserted[0].EventType)
	}
	if len(store.projectionUpdates) != 1 {
		t.Fatalf("expected 1 projection update, got %d", len(store.projectionUpdates))
	}
	if store.projectionUpdates[0].Status != "completed" {
		t.Fatalf("expected status completed, got %s", store.projectionUpdates[0].Status)
	}
}

func TestTimelineReadsFromStore(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{
		events: []repo.EventRecord{
			{
				EventID:       "evt_1",
				SessionID:     "sess_1",
				StepID:        "step_1",
				CorrelationID: "corr_1",
				EventType:     "policy_blocked",
				Decision:      "BLOCK",
				Tool:          "shell",
				Action:        "exec",
				Resource:      "host:prod",
				RiskTags:      []string{"destructive_action"},
				CreatedAt:     now,
			},
		},
	}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	rr := httptest.NewRecorder()
	r := NewRouter()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/sessions/sess_1/timeline?limit=10&offset=0", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp struct {
		Items []map[string]interface{} `json:"items"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 timeline item, got %d", len(resp.Items))
	}
}

func TestPostflightFailedSetsBlockedProjection(t *testing.T) {
	store := &fakeStore{}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	payload := postflightRequest{
		SessionID:     "sess_2",
		StepID:        "step_2",
		CorrelationID: "corr_2",
		AgentID:       "ops-agent",
		Environment:   "prod",
		Tool:          "shell",
		Action:        "exec",
		Resource:      "host:prod",
		Result:        "failed",
		OutputSummary: "command failed",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/postflight", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	handlePostflight(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", rr.Code)
	}
	if len(store.projectionUpdates) != 1 {
		t.Fatalf("expected 1 projection update, got %d", len(store.projectionUpdates))
	}
	if store.projectionUpdates[0].Status != "blocked" {
		t.Fatalf("expected status blocked, got %s", store.projectionUpdates[0].Status)
	}
}

func TestPreflightProjectionFailure(t *testing.T) {
	store := &fakeStore{
		projectionErr: errors.New("projection failed"),
		policyRules: []repo.PolicyRule{
			{
				PolicyID: "pol_test_github_main", ScopeTool: "github", ScopeEnvironment: "prod",
				ConditionExpr: map[string]interface{}{"action": "push", "resource_contains": "branch:main"},
				Decision: "REQUIRE_APPROVAL", Priority: 10, Enabled: true,
			},
		},
	}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	payload := preflightRequest{
		SessionID:     "sess_1",
		StepID:        "step_1",
		CorrelationID: "corr_1",
		AgentID:       "coding-agent-prod",
		Environment:   "prod",
		Tool:          "github",
		Action:        "push",
		Resource:      "repo:org/api-service/branch:main",
		InputSummary:  "push commit fix(auth)",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/preflight", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	handlePreflight(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}
