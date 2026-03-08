package httpx

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func registerSessionRoutes(r chi.Router) {
	r.Get("/sessions", handleListSessions)
	r.Get("/sessions/{session_id}", handleGetSession)
	r.Get("/sessions/{session_id}/timeline", handleSessionTimeline)
	r.Get("/events/{event_id}", handleGetEvent)
}

func handleListSessions(w http.ResponseWriter, r *http.Request) {
	if sessionSvc == nil {
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

	sessions, err := sessionSvc.ListSessions(r.Context(), limit, offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "STORE_ERROR", "failed to list sessions", nil)
		return
	}

	items := make([]map[string]interface{}, 0, len(sessions))
	for _, s := range sessions {
		items = append(items, map[string]interface{}{
			"session_id":         s.SessionID,
			"objective":          s.Objective,
			"agent_id":           s.AgentID,
			"user_id":            s.UserID,
			"environment":        s.Environment,
			"status":             s.Status,
			"started_at":         s.StartedAt,
			"ended_at":           s.EndedAt,
			"risk_score":         s.RiskScore,
			"approvals_count":    s.ApprovalsCount,
			"blocked_count":      s.BlockedCount,
			"touched_resources":  s.TouchedResources,
			"last_event_at":      s.LastEventAt,
			"updated_at":         s.UpdatedAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":     items,
		"page":      page,
		"page_size": limit,
		"total":     len(items),
	})
}

func handleGetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session_id")
	if sessionSvc == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"session_id": sessionID,
		})
		return
	}

	s, err := sessionSvc.GetSession(r.Context(), sessionID)
	if err != nil {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "session not found", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"session_id":         s.SessionID,
		"objective":          s.Objective,
		"agent_id":           s.AgentID,
		"user_id":            s.UserID,
		"environment":        s.Environment,
		"status":             s.Status,
		"started_at":         s.StartedAt,
		"ended_at":           s.EndedAt,
		"risk_score":         s.RiskScore,
		"approvals_count":    s.ApprovalsCount,
		"blocked_count":      s.BlockedCount,
		"touched_resources":  s.TouchedResources,
		"last_event_at":      s.LastEventAt,
		"updated_at":         s.UpdatedAt,
	})
}

func handleSessionTimeline(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session_id")
	limit := parsePositiveInt(r.URL.Query().Get("limit"), 100)
	offset := parsePositiveInt(r.URL.Query().Get("offset"), 0)

	if sessionSvc == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"items":       []interface{}{},
			"next_cursor": nil,
		})
		return
	}

	events, err := sessionSvc.ListSessionTimeline(r.Context(), sessionID, limit, offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "STORE_ERROR", "failed to query session timeline", nil)
		return
	}

	items := make([]map[string]interface{}, 0, len(events))
	for _, e := range events {
		items = append(items, map[string]interface{}{
			"event_id":            e.EventID,
			"session_id":          e.SessionID,
			"step_id":             e.StepID,
			"correlation_id":      e.CorrelationID,
			"event_type":          e.EventType,
			"decision":            e.Decision,
			"tool":                e.Tool,
			"action":              e.Action,
			"resource":            e.Resource,
			"risk_score":          e.RiskScore,
			"risk_tags":           e.RiskTags,
			"matched_policy_ids":  e.MatchedPolicyIDs,
			"reason_code":         e.ReasonCode,
			"reason_text":         e.ReasonText,
			"input_summary":       e.InputSummary,
			"output_summary":      e.OutputSummary,
			"artifact_refs":       e.ArtifactRefs,
			"actor_type":          e.ActorType,
			"actor_id":            e.ActorID,
			"created_at":          e.CreatedAt,
		})
	}

	nextCursor := interface{}(nil)
	if len(items) == limit {
		nextCursor = offset + len(items)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":       items,
		"next_cursor": nextCursor,
	})
}

func handleGetEvent(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"event_id": chi.URLParam(r, "event_id"),
	})
}

func parsePositiveInt(v string, fallback int) int {
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}
