package httpx

import (
	"encoding/json"
	"net/http"

	"agent-control-plane/apps/api/internal/service"
)

type preflightRequest struct {
	SessionID     string                 `json:"session_id"`
	StepID        string                 `json:"step_id"`
	CorrelationID string                 `json:"correlation_id"`
	Agent         map[string]interface{} `json:"agent"`
	ToolCall      map[string]interface{} `json:"tool_call"`
}

type preflightResponse = service.PreflightOutput

type postflightRequest struct {
	SessionID     string   `json:"session_id"`
	StepID        string   `json:"step_id"`
	CorrelationID string   `json:"correlation_id"`
	Tool          string   `json:"tool"`
	Action        string   `json:"action"`
	Resource      string   `json:"resource"`
	Result        string   `json:"result"`
	OutputSummary string   `json:"output_summary"`
	ArtifactRefs  []string `json:"artifact_refs"`
	ActorID       string   `json:"actor_id"`
}

func registerGatewayRoutes(r interface{ Post(string, http.HandlerFunc) }) {
	r.Post("/gateway/preflight", handlePreflight)
	r.Post("/gateway/postflight", handlePostflight)
}

func handlePreflight(w http.ResponseWriter, r *http.Request) {
	var req preflightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body", nil)
		return
	}
	if gatewaySvc == nil {
		writeJSON(w, http.StatusOK, service.PreflightOutput{
			Decision:         "ALLOW",
			DecisionID:       "dec_default_allow",
			MatchedPolicyIDs: []string{},
			ReasonCode:       "DEFAULT_ALLOW",
			ReasonText:       "No policy matched",
			RiskTags:         []string{},
		})
		return
	}

	resp, err := gatewaySvc.ProcessPreflight(r.Context(), service.PreflightInput{
		SessionID:     req.SessionID,
		StepID:        req.StepID,
		CorrelationID: req.CorrelationID,
		AgentID:       getField(req.Agent, "agent_id"),
		UserID:        getField(req.Agent, "user_id"),
		Environment:   getField(req.Agent, "environment"),
		Objective:     getField(req.Agent, "objective"),
		Tool:          getField(req.ToolCall, "tool"),
		Action:        getField(req.ToolCall, "action"),
		Resource:      getField(req.ToolCall, "resource"),
		InputSummary:  getField(req.ToolCall, "input_summary"),
		Command:       getField(req.ToolCall, "command"),
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "STORE_ERROR", "failed to process preflight", nil)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func handlePostflight(w http.ResponseWriter, r *http.Request) {
	var req postflightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body", nil)
		return
	}
	if gatewaySvc == nil {
		writeJSON(w, http.StatusAccepted, map[string]bool{"accepted": true})
		return
	}

	if err := gatewaySvc.ProcessPostflight(r.Context(), service.PostflightInput{
		SessionID:     req.SessionID,
		StepID:        req.StepID,
		CorrelationID: req.CorrelationID,
		Tool:          req.Tool,
		Action:        req.Action,
		Resource:      req.Resource,
		Result:        req.Result,
		OutputSummary: req.OutputSummary,
		ArtifactRefs:  req.ArtifactRefs,
		ActorID:       req.ActorID,
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "STORE_ERROR", "failed to process postflight", nil)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]bool{"accepted": true})
}

func getField(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}
