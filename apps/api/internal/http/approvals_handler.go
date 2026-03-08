package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"agent-control-plane/apps/api/internal/repo"
	"github.com/go-chi/chi/v5"
)

func registerApprovalRoutes(r chi.Router) {
	r.Get("/approvals", handleListApprovals)
	r.Get("/approvals/{approval_id}", handleGetApproval)
	r.Post("/approvals/{approval_id}/decision", handleApprovalDecision)
}

type approvalDecisionRequest struct {
	Decision        string `json:"decision"`
	ApproverID      string `json:"approver_id"`
	DecisionComment string `json:"decision_comment"`
}

func handleListApprovals(w http.ResponseWriter, r *http.Request) {
	if approvalSvc == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"items":     []interface{}{},
			"page":      1,
			"page_size": 20,
			"total":     0,
		})
		return
	}

	limit := parsePositiveInt(r.URL.Query().Get("page_size"), 20)
	if limit <= 0 {
		limit = 20
	}
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))

	approvals, err := approvalSvc.List(r.Context(), status, limit, offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "STORE_ERROR", "failed to list approvals", nil)
		return
	}

	items := make([]map[string]interface{}, 0, len(approvals))
	for _, a := range approvals {
		items = append(items, approvalToJSON(a))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":     items,
		"page":      page,
		"page_size": limit,
		"total":     len(items),
	})
}

func handleGetApproval(w http.ResponseWriter, r *http.Request) {
	approvalID := chi.URLParam(r, "approval_id")
	if approvalSvc == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"approval_id": approvalID})
		return
	}

	a, err := approvalSvc.Get(r.Context(), approvalID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "NOT_FOUND", "approval not found", nil)
			return
		}
		writeErr(w, http.StatusInternalServerError, "STORE_ERROR", "failed to get approval", nil)
		return
	}

	writeJSON(w, http.StatusOK, approvalToJSON(a))
}

func handleApprovalDecision(w http.ResponseWriter, r *http.Request) {
	if approvalSvc == nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
		return
	}

	approvalID := chi.URLParam(r, "approval_id")
	var req approvalDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body", nil)
		return
	}

	decision := strings.ToLower(strings.TrimSpace(req.Decision))
	if decision != "approve" && decision != "reject" {
		writeErr(w, http.StatusBadRequest, "BAD_REQUEST", "decision must be approve or reject", nil)
		return
	}

	updated, err := approvalSvc.Decide(r.Context(), repo.ApprovalDecisionInput{
		ApprovalID:      approvalID,
		Decision:        decision,
		ApproverID:      strings.TrimSpace(req.ApproverID),
		DecisionComment: strings.TrimSpace(req.DecisionComment),
	})
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "NOT_FOUND", "approval not found", nil)
			return
		}
		writeErr(w, http.StatusInternalServerError, "STORE_ERROR", "failed to apply approval decision", nil)
		return
	}

	writeJSON(w, http.StatusOK, approvalToJSON(updated))
}

func approvalToJSON(a repo.ApprovalRecord) map[string]interface{} {
	return map[string]interface{}{
		"approval_id":         a.ApprovalID,
		"session_id":          a.SessionID,
		"step_id":             a.StepID,
		"event_id":            a.EventID,
		"status":              a.Status,
		"action":              a.Action,
		"tool":                a.Tool,
		"resource":            a.Resource,
		"objective":           a.Objective,
		"trigger_reason":      a.TriggerReason,
		"risk_tags":           a.RiskTags,
		"potential_impact":    a.PotentialImpact,
		"suggested_safe_alt":  a.SuggestedSafeAlt,
		"requested_at":        a.RequestedAt,
		"decided_at":          a.DecidedAt,
		"approver_id":         a.ApproverID,
		"decision_comment":    a.DecisionComment,
	}
}
