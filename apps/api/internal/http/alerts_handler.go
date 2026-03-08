package httpx

import "net/http"

func registerAlertRoutes(r interface{ Get(string, http.HandlerFunc); Post(string, http.HandlerFunc) }) {
	r.Get("/alerts", handleListAlerts)
	r.Get("/alerts/{alert_id}", handleGetAlert)
	r.Post("/alerts/{alert_id}/status", handleAlertStatus)
	r.Get("/alerts/{alert_id}/export", handleAlertExport)
}

func handleListAlerts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": []interface{}{}})
}

func handleGetAlert(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"alert_id": "alert_mock_001"})
}

func handleAlertStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "investigating"})
}

func handleAlertExport(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"format": "markdown", "content": "# incident summary"})
}
