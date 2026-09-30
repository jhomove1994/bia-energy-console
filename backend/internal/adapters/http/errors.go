package httpapi

import (
	"encoding/json"
	"net/http"
)

// Error codes returned in the JSON error body.
const (
	codeInvalidBody        = "invalid_body"
	codeInvalidCredentials = "invalid_credentials"
	codeUnauthorized       = "unauthorized"
	codeNotFound           = "not_found"
	codeInvalidQuery       = "invalid_query"
	codeInternal           = "internal"
)

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// writeJSON writes v as JSON with the given status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes the standard error shape {"error":{"code","message"}}.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorBody{Error: errorDetail{Code: code, Message: message}})
}
