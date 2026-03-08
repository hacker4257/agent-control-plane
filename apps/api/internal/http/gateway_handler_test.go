package httpx

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-control-plane/apps/api/internal/service"
)

func TestPreflightBadJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/preflight", bytes.NewBufferString("{"))
	rr := httptest.NewRecorder()

	handlePreflight(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestPreflightShellDangerousBlocked(t *testing.T) {
	gatewaySvc = service.NewGatewayService(nil)
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
	gatewaySvc = service.NewGatewayService(nil)
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
	gatewaySvc = service.NewGatewayService(nil)
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
	gatewaySvc = service.NewGatewayService(nil)
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
