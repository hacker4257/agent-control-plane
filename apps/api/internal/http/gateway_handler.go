package httpx

import (
	"encoding/json"
	"net/http"
	"strings"

	"agent-control-plane/apps/api/internal/service"
)

type preflightRequest struct {
	SessionID     string `json:"session_id"`
	StepID        string `json:"step_id"`
	CorrelationID string `json:"correlation_id"`
	AgentID       string `json:"agent_id"`
	UserID        string `json:"user_id"`
	Environment   string `json:"environment"`
	Objective     string `json:"objective"`
	Tool          string `json:"tool"`
	Action        string `json:"action"`
	Resource      string `json:"resource"`
	InputSummary  string `json:"input_summary"`
	Command       string `json:"command"`
}

type preflightResponse = service.PreflightOutput

type postflightRequest struct {
	SessionID     string   `json:"session_id"`
	StepID        string   `json:"step_id"`
	CorrelationID string   `json:"correlation_id"`
	AgentID       string   `json:"agent_id"`
	Environment   string   `json:"environment"`
	Tool          string   `json:"tool"`
	Action        string   `json:"action"`
	Resource      string   `json:"resource"`
	Result        string   `json:"result"`
	OutputSummary string   `json:"output_summary"`
	ArtifactRefs  []string `json:"artifact_refs"`
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
		SessionID:     strings.TrimSpace(req.SessionID),
		StepID:        strings.TrimSpace(req.StepID),
		CorrelationID: strings.TrimSpace(req.CorrelationID),
		AgentID:       strings.TrimSpace(req.AgentID),
		UserID:        strings.TrimSpace(req.UserID),
		Environment:   strings.TrimSpace(req.Environment),
		Objective:     strings.TrimSpace(req.Objective),
		Tool:          strings.TrimSpace(req.Tool),
		Action:        strings.TrimSpace(req.Action),
		Resource:      strings.TrimSpace(req.Resource),
		InputSummary:  strings.TrimSpace(req.InputSummary),
		Command:       strings.TrimSpace(req.Command),
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "STORE_ERROR", "failed to process preflight", nil)
		return
	}

	writeJSON(w, http.StatusOK, resp)

	BroadcastEvent("preflight", resp)
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
		SessionID:     strings.TrimSpace(req.SessionID),
		StepID:        strings.TrimSpace(req.StepID),
		CorrelationID: strings.TrimSpace(req.CorrelationID),
		AgentID:       strings.TrimSpace(req.AgentID),
		Environment:   strings.TrimSpace(req.Environment),
		Tool:          strings.TrimSpace(req.Tool),
		Action:        strings.TrimSpace(req.Action),
		Resource:      strings.TrimSpace(req.Resource),
		Result:        strings.TrimSpace(req.Result),
		OutputSummary: strings.TrimSpace(req.OutputSummary),
		ArtifactRefs:  req.ArtifactRefs,
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "STORE_ERROR", "failed to process postflight", nil)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]bool{"accepted": true})

	BroadcastEvent("postflight", map[string]interface{}{
		"session_id": strings.TrimSpace(req.SessionID),
		"tool":       strings.TrimSpace(req.Tool),
		"action":     strings.TrimSpace(req.Action),
		"resource":   strings.TrimSpace(req.Resource),
		"result":     strings.TrimSpace(req.Result),
	})
}
