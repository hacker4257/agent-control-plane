package httpx

import "net/http"

func registerDashboardRoutes(r interface{ Get(string, http.HandlerFunc) }) {
	r.Get("/dashboard/summary", handleDashboardSummary)
}

func handleDashboardSummary(w http.ResponseWriter, r *http.Request) {
	if dashboardSvc == nil {
		writeJSON(w, http.StatusOK, map[string]int{
			"sessions_count":          0,
			"pending_approvals_count": 0,
			"blocked_actions_count":   0,
			"policy_hits_count":       0,
		})
		return
	}

	summary, err := dashboardSvc.Summary(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "STORE_ERROR", "failed to load dashboard summary", nil)
		return
	}

	writeJSON(w, http.StatusOK, summary)
}
