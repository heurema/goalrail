package httpserver

import (
	"net/http"
	"strings"
)

const (
	corsAllowedMethods = "GET, POST, PATCH, OPTIONS"
	corsAllowedHeaders = "Authorization, Content-Type"
)

// WithCORS wraps the router with exact-origin CORS handling.
func WithCORS(next http.Handler, allowedOrigins []string) http.Handler {
	if len(allowedOrigins) == 0 {
		return next
	}

	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			addVary(w.Header(), "Origin")
		}
		if _, ok := allowed[origin]; ok {
			headers := w.Header()
			headers.Set("Access-Control-Allow-Origin", origin)
			if r.Method == http.MethodOptions {
				headers.Set("Access-Control-Allow-Methods", corsAllowedMethods)
				headers.Set("Access-Control-Allow-Headers", corsAllowedHeaders)
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func addVary(headers http.Header, value string) {
	for _, part := range strings.Split(headers.Get("Vary"), ",") {
		if strings.EqualFold(strings.TrimSpace(part), value) {
			return
		}
	}
	headers.Add("Vary", value)
}
