package httpx

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
	payload := preflightRequest{
		SessionID:     "sess_1",
		StepID:        "step_1",
		CorrelationID: "corr_1",
		Agent:         map[string]interface{}{"environment": "prod"},
		ToolCall: map[string]interface{}{
			"tool":          "shell",
			"action":        "exec",
			"resource":      "host:prod",
			"input_summary": "rm -rf /tmp/cache",
		},
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
	payload := preflightRequest{
		SessionID:     "sess_2",
		StepID:        "step_2",
		CorrelationID: "corr_2",
		Agent:         map[string]interface{}{"environment": "prod"},
		ToolCall: map[string]interface{}{
			"tool":          "github",
			"action":        "push",
			"resource":      "repo:org/api-service/branch:main",
			"input_summary": "push commit fix(auth)",
		},
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
	payload := preflightRequest{
		SessionID:     "sess_3",
		StepID:        "step_3",
		CorrelationID: "corr_3",
		Agent:         map[string]interface{}{"environment": "prod"},
		ToolCall: map[string]interface{}{
			"tool":          "browser",
			"action":        "submit",
			"resource":      "https://payment.example.com/checkout",
			"input_summary": "submit payment form",
		},
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
	payload := preflightRequest{
		SessionID:     "sess_4",
		StepID:        "step_4",
		CorrelationID: "corr_4",
		Agent:         map[string]interface{}{"environment": "staging"},
		ToolCall: map[string]interface{}{
			"tool":          "github",
			"action":        "create_pull_request",
			"resource":      "repo:org/api-service/branch:feature-x",
			"input_summary": "create PR",
		},
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
