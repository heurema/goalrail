package httpserver

import (
	"encoding/json"
	"net/http"
)

type errorResponse struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// RespondJSON writes a JSON response with the given HTTP status code.
func RespondJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// RespondError writes a structured JSON error response.
func RespondError(w http.ResponseWriter, status int, code string, message string) {
	RespondJSON(w, status, errorResponse{
		Error: errorDetail{
			Code:    code,
			Message: message,
		},
	})
}

func respondInvalidJSON(w http.ResponseWriter) {
	RespondError(w, http.StatusBadRequest, "invalid_json", "invalid JSON request body")
}

func respondInternalError(w http.ResponseWriter) {
	RespondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
}
