package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"agent-control-plane/apps/api/internal/repo"
	"github.com/go-chi/chi/v5"
)

type policyUpsertRequest struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	ScopeAgent       string                 `json:"scope_agent"`
	ScopeTool        string                 `json:"scope_tool"`
	ScopeEnvironment string                 `json:"scope_environment"`
	ScopeResourcePat string                 `json:"scope_resource_pat"`
	ConditionExpr    map[string]interface{} `json:"condition_expr"`
	Decision         string                 `json:"decision"`
	Priority         int                    `json:"priority"`
	Enabled          *bool                  `json:"enabled,omitempty"`
	CreatedBy        string                 `json:"created_by"`
}

type policyEvaluatePreviewRequest struct {
	Tool        string `json:"tool"`
	Action      string `json:"action"`
	Resource    string `json:"resource"`
	Environment string `json:"environment"`
}

func registerPolicyRoutes(r chi.Router) {
	r.Get("/policies", handleListPolicies)
	r.Post("/policies", handleCreatePolicy)
	r.Patch("/policies/{policy_id}", handleUpdatePolicy)
	r.Post("/policies/{policy_id}/enable", handleEnablePolicy)
	r.Post("/policies/{policy_id}/disable", handleDisablePolicy)
	r.Post("/policies/evaluate-preview", handlePolicyEvaluatePreview)
}

func handleListPolicies(w http.ResponseWriter, r *http.Request) {
	if policySvc == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"items": []interface{}{}})
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

	enabledOnly := false
	if v := strings.TrimSpace(r.URL.Query().Get("enabled")); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "BAD_REQUEST", "enabled must be boolean", nil)
			return
		}
		enabledOnly = parsed
	}

	items, err := policySvc.List(r.Context(), enabledOnly, limit, offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "STORE_ERROR", "failed to list policies", nil)
		return
	}

	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		result = append(result, policyToJSON(item))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":     result,
		"page":      page,
		"page_size": limit,
		"total":     len(result),
	})
}

func handleCreatePolicy(w http.ResponseWriter, r *http.Request) {
	if policySvc == nil {
		writeJSON(w, http.StatusCreated, map[string]string{"policy_id": "pol_mock_001"})
		return
	}

	var req policyUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body", nil)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeErr(w, http.StatusBadRequest, "BAD_REQUEST", "name is required", nil)
		return
	}
	if strings.TrimSpace(req.Decision) == "" {
		writeErr(w, http.StatusBadRequest, "BAD_REQUEST", "decision is required", nil)
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	created, err := policySvc.Create(r.Context(), repo.PolicyRule{
		Name:             strings.TrimSpace(req.Name),
		Description:      strings.TrimSpace(req.Description),
		ScopeAgent:       strings.TrimSpace(req.ScopeAgent),
		ScopeTool:        strings.TrimSpace(req.ScopeTool),
		ScopeEnvironment: strings.TrimSpace(req.ScopeEnvironment),
		ScopeResourcePat: strings.TrimSpace(req.ScopeResourcePat),
		ConditionExpr:    req.ConditionExpr,
		Decision:         strings.TrimSpace(req.Decision),
		Priority:         req.Priority,
		Enabled:          enabled,
		CreatedBy:        strings.TrimSpace(req.CreatedBy),
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "STORE_ERROR", "failed to create policy", nil)
		return
	}
	writeJSON(w, http.StatusCreated, policyToJSON(created))
}

func handleUpdatePolicy(w http.ResponseWriter, r *http.Request) {
	if policySvc == nil {
		writeJSON(w, http.StatusOK, map[string]string{"policy_id": chi.URLParam(r, "policy_id")})
		return
	}

	policyID := chi.URLParam(r, "policy_id")
	var req policyUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body", nil)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeErr(w, http.StatusBadRequest, "BAD_REQUEST", "name is required", nil)
		return
	}
	if strings.TrimSpace(req.Decision) == "" {
		writeErr(w, http.StatusBadRequest, "BAD_REQUEST", "decision is required", nil)
		return
	}

	updated, err := policySvc.Update(r.Context(), policyID, repo.PolicyRule{
		Name:             strings.TrimSpace(req.Name),
		Description:      strings.TrimSpace(req.Description),
		ScopeAgent:       strings.TrimSpace(req.ScopeAgent),
		ScopeTool:        strings.TrimSpace(req.ScopeTool),
		ScopeEnvironment: strings.TrimSpace(req.ScopeEnvironment),
		ScopeResourcePat: strings.TrimSpace(req.ScopeResourcePat),
		ConditionExpr:    req.ConditionExpr,
		Decision:         strings.TrimSpace(req.Decision),
		Priority:         req.Priority,
		CreatedBy:        strings.TrimSpace(req.CreatedBy),
	})
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "NOT_FOUND", "policy not found", nil)
			return
		}
		writeErr(w, http.StatusInternalServerError, "STORE_ERROR", "failed to update policy", nil)
		return
	}
	writeJSON(w, http.StatusOK, policyToJSON(updated))
}

func handleEnablePolicy(w http.ResponseWriter, r *http.Request) {
	handleSetPolicyEnabled(w, r, true)
}

func handleDisablePolicy(w http.ResponseWriter, r *http.Request) {
	handleSetPolicyEnabled(w, r, false)
}

func handleSetPolicyEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	if policySvc == nil {
		status := "disabled"
		if enabled {
			status = "enabled"
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": status})
		return
	}

	policyID := chi.URLParam(r, "policy_id")
	if err := policySvc.SetEnabled(r.Context(), policyID, enabled); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "NOT_FOUND", "policy not found", nil)
			return
		}
		writeErr(w, http.StatusInternalServerError, "STORE_ERROR", "failed to update policy status", nil)
		return
	}

	status := "disabled"
	if enabled {
		status = "enabled"
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"policy_id": policyID,
		"status":    status,
	})
}

func handlePolicyEvaluatePreview(w http.ResponseWriter, r *http.Request) {
	if policySvc == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"decision": "ALLOW", "matched_policy_ids": []string{}})
		return
	}

	var req policyEvaluatePreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body", nil)
		return
	}

	decision, matched, err := policySvc.EvaluatePreview(
		r.Context(),
		strings.TrimSpace(req.Tool),
		strings.TrimSpace(req.Action),
		strings.TrimSpace(req.Resource),
		strings.TrimSpace(req.Environment),
	)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "STORE_ERROR", "failed to evaluate policy preview", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"decision":           decision,
		"matched_policy_ids": matched,
	})
}

func policyToJSON(p repo.PolicyRule) map[string]interface{} {
	return map[string]interface{}{
		"policy_id":          p.PolicyID,
		"name":               p.Name,
		"description":        p.Description,
		"scope_agent":        p.ScopeAgent,
		"scope_tool":         p.ScopeTool,
		"scope_environment":  p.ScopeEnvironment,
		"scope_resource_pat": p.ScopeResourcePat,
		"condition_expr":     p.ConditionExpr,
		"decision":           p.Decision,
		"priority":           p.Priority,
		"enabled":            p.Enabled,
		"created_by":         p.CreatedBy,
		"created_at":         p.CreatedAt,
		"updated_at":         p.UpdatedAt,
	}
}
