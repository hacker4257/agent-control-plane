package httpx

import (
	"encoding/json"
	"net/http"
)

type errorEnvelope struct {
	Error struct {
		Code    string      `json:"code"`
		Message string      `json:"message"`
		Details interface{} `json:"details,omitempty"`
	} `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, code, message string, details interface{}) {
	var env errorEnvelope
	env.Error.Code = code
	env.Error.Message = message
	env.Error.Details = details
	writeJSON(w, status, env)
}
