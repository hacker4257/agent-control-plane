package httpx

import "net/http"

func registerInventoryRoutes(r interface{ Get(string, http.HandlerFunc) }) {
	r.Get("/inventory/agents", handleListAgents)
	r.Get("/inventory/tools", handleListTools)
	r.Get("/sessions/{session_id}/artifacts", handleListSessionArtifacts)
	r.Get("/artifacts/{artifact_id}", handleGetArtifact)
}

func handleListAgents(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": []interface{}{}})
}

func handleListTools(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": []interface{}{}})
}

func handleListSessionArtifacts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": []interface{}{}})
}

func handleGetArtifact(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"artifact_id": "art_mock_001"})
}
