package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-control-plane/apps/api/internal/repo"
	"agent-control-plane/apps/api/internal/service"
)

type fakeGatewayStore struct {
	rules []repo.PolicyRule
}

func (s *fakeGatewayStore) InsertEvent(ctx context.Context, e repo.EventRecord) error { return nil }
func (s *fakeGatewayStore) UpsertSessionProjection(ctx context.Context, u repo.SessionProjectionUpdate) error {
	return nil
}
func (s *fakeGatewayStore) CreateApproval(ctx context.Context, a repo.ApprovalRecord) error {
	return nil
}
func (s *fakeGatewayStore) ListPolicyRules(ctx context.Context, enabledOnly bool, limit, offset int) ([]repo.PolicyRule, error) {
	return s.rules, nil
}

func testPolicyRules() []repo.PolicyRule {
	return []repo.PolicyRule{
		{
			PolicyID: "pol_test_shell", ScopeTool: "shell",
			ConditionExpr: map[string]interface{}{"command_patterns": []interface{}{"rm -rf", "sudo", "curl|sh"}},
			Decision: "BLOCK", Priority: 5, Enabled: true,
		},
		{
			PolicyID: "pol_test_github_main", ScopeTool: "github", ScopeEnvironment: "prod",
			ConditionExpr: map[string]interface{}{"action": "push", "resource_contains": "branch:main"},
			Decision: "REQUIRE_APPROVAL", Priority: 10, Enabled: true,
		},
		{
			PolicyID: "pol_test_browser_pay", ScopeTool: "browser",
			ConditionExpr: map[string]interface{}{"action": "submit", "resource_contains": "payment"},
			Decision: "BLOCK", Priority: 15, Enabled: true,
		},
	}
}

func TestPreflightBadJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/preflight", bytes.NewBufferString("{"))
	rr := httptest.NewRecorder()

	handlePreflight(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestPreflightShellDangerousBlocked(t *testing.T) {
	gatewaySvc = service.NewGatewayService(&fakeGatewayStore{rules: testPolicyRules()})
	t.Cleanup(func() { gatewaySvc = nil })

	payload := preflightRequest{
		SessionID:     "sess_1",
		StepID:        "step_1",
		CorrelationID: "corr_1",
		AgentID:       "agent.shell",
		Environment:   "prod",
		Tool:          "shell",
		Action:        "exec",
		Resource:      "host:prod",
		InputSummary:  "rm -rf /tmp/cache",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/preflight", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handlePreflight(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp preflightResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Decision != "BLOCK" {
		t.Fatalf("expected BLOCK, got %s", resp.Decision)
	}
}

func TestPreflightGithubMainRequiresApproval(t *testing.T) {
	gatewaySvc = service.NewGatewayService(&fakeGatewayStore{rules: testPolicyRules()})
	t.Cleanup(func() { gatewaySvc = nil })

	payload := preflightRequest{
		SessionID:     "sess_2",
		StepID:        "step_2",
		CorrelationID: "corr_2",
		AgentID:       "agent.git",
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

	var resp preflightResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)

	if resp.Decision != "REQUIRE_APPROVAL" {
		t.Fatalf("expected REQUIRE_APPROVAL, got %s", resp.Decision)
	}
	if resp.ApprovalID == "" {
		t.Fatal("expected approval_id to be set")
	}
}

func TestPreflightBrowserPaymentBlocked(t *testing.T) {
	gatewaySvc = service.NewGatewayService(&fakeGatewayStore{rules: testPolicyRules()})
	t.Cleanup(func() { gatewaySvc = nil })

	payload := preflightRequest{
		SessionID:     "sess_3",
		StepID:        "step_3",
		CorrelationID: "corr_3",
		AgentID:       "agent.browser",
		Environment:   "prod",
		Tool:          "browser",
		Action:        "submit",
		Resource:      "https://payment.example.com/checkout",
		InputSummary:  "submit payment form",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/preflight", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handlePreflight(rr, req)

	var resp preflightResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)

	if resp.Decision != "BLOCK" {
		t.Fatalf("expected BLOCK, got %s", resp.Decision)
	}
}

func TestPreflightDefaultAllow(t *testing.T) {
	gatewaySvc = service.NewGatewayService(&fakeGatewayStore{rules: testPolicyRules()})
	t.Cleanup(func() { gatewaySvc = nil })

	payload := preflightRequest{
		SessionID:     "sess_4",
		StepID:        "step_4",
		CorrelationID: "corr_4",
		AgentID:       "agent.git",
		Environment:   "staging",
		Tool:          "github",
		Action:        "create_pull_request",
		Resource:      "repo:org/api-service/branch:feature-x",
		InputSummary:  "create PR",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/preflight", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handlePreflight(rr, req)

	var resp preflightResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)

	if resp.Decision != "ALLOW" {
		t.Fatalf("expected ALLOW, got %s", resp.Decision)
	}
}

func TestPostflightBadJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/postflight", bytes.NewBufferString("{"))
	rr := httptest.NewRecorder()

	handlePostflight(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestPostflightAccepted(t *testing.T) {
	payload := postflightRequest{
		SessionID:     "sess_5",
		StepID:        "step_5",
		CorrelationID: "corr_5",
		AgentID:       "agent.git",
		Environment:   "prod",
		Tool:          "github",
		Action:        "push",
		Resource:      "repo:org/service/branch:feature-x",
		Result:        "completed",
		OutputSummary: "push succeeded",
		ArtifactRefs:  []string{"artifact:commit:abc123"},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/postflight", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handlePostflight(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", rr.Code)
	}
}
