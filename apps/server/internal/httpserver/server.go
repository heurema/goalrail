package httpserver

import (
	"net/http"
	"time"
)

// NewServer creates the HTTP server used by goalrail-server.
func NewServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
