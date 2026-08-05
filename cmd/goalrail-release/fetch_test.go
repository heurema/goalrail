package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestFetchSourceReadsOnlyWithinTheBound(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("artifact"))
	}))
	defer server.Close()

	raw, err := fetchSourceWithClient(context.Background(), server.Client(), server.URL, 8)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "artifact" {
		t.Fatalf("artifact = %q, want artifact", raw)
	}
	if _, err := fetchSourceWithClient(context.Background(), server.Client(), server.URL, 7); err == nil || !strings.Contains(err.Error(), "exceeds 7 bytes") {
		t.Fatalf("bounded fetch error = %v, want size failure", err)
	}
}

func TestFetchSourceRejectsAnotherRedirectAuthority(t *testing.T) {
	var targetReached atomic.Bool
	target := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		targetReached.Store(true)
	}))
	defer target.Close()
	source := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		http.Redirect(w, request, target.URL, http.StatusFound)
	}))
	defer source.Close()

	_, err := fetchSourceWithClient(context.Background(), source.Client(), source.URL, 1024)
	if err == nil || !strings.Contains(err.Error(), "refuse source redirect outside") {
		t.Fatalf("redirect error = %v, want authority refusal", err)
	}
	if targetReached.Load() {
		t.Fatal("redirect target was reached")
	}
}

func TestFetchSourceRequiresHTTPSAndAPositiveBound(t *testing.T) {
	for _, test := range []struct {
		name     string
		rawURL   string
		maxBytes int64
		want     string
	}{
		{name: "plain HTTP", rawURL: "http://example.test/artifact", maxBytes: 1, want: "is not HTTPS"},
		{name: "zero bound", rawURL: "https://example.test/artifact", maxBytes: 0, want: "must be positive"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := fetchSourceWithClient(context.Background(), http.DefaultClient, test.rawURL, test.maxBytes)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}
