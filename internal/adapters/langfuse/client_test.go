package langfuse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/domain"
)

var clientTestTime = time.Date(2026, time.July, 22, 10, 0, 0, 0, time.UTC)

func TestClientReadsBoundedCursorPagesWithExactSessionFilter(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		username, password, ok := request.BasicAuth()
		if !ok || username != "public-test" || password != "secret-test" {
			t.Errorf("unexpected basic auth")
		}
		if request.URL.Path != "/api/public/v2/observations" || request.URL.Query().Get("fields") != "core,basic" ||
			request.URL.Query().Get("limit") != "2" || request.URL.Query().Get("fromStartTime") == "" ||
			request.URL.Query().Get("toStartTime") == "" {
			t.Errorf("unexpected request URL: %s", request.URL.String())
		}
		var filters []sessionFilter
		if err := json.Unmarshal([]byte(request.URL.Query().Get("filter")), &filters); err != nil ||
			len(filters) != 1 || filters[0].Column != "sessionId" || filters[0].Value != "session-1" {
			t.Errorf("unexpected filter: %#v err=%v", filters, err)
		}
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Query().Get("cursor") == "" {
			fmt.Fprint(writer, observationPageJSON("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "obs-1", "cursor-2"))
			return
		}
		if request.URL.Query().Get("cursor") != "cursor-2" {
			t.Errorf("unexpected cursor: %q", request.URL.Query().Get("cursor"))
		}
		fmt.Fprint(writer, observationPageJSON("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "obs-2", ""))
	}))
	defer server.Close()

	client := mustTestClient(t, server.URL, 2, 10, defaultResponseBytes)
	observations, err := client.ListSessionObservations(context.Background(), validTraceQuery())
	if err != nil {
		t.Fatalf("list observations: %v", err)
	}
	if requests != 2 || len(observations) != 2 ||
		observations[0].TraceReference != "langfuse-trace:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" ||
		observations[1].TraceReference != "langfuse-trace:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("unexpected observations: requests=%d values=%#v", requests, observations)
	}
}

func TestClientRejectsMalformedOversizedAndRepeatedCursorResponses(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		maxBytes int64
		want     error
	}{
		{name: "unknown field", body: `{"data":[],"meta":{"cursor":null},"rawPrompt":"forbidden"}`, maxBytes: 1024, want: ErrMalformedResponse},
		{name: "oversized", body: strings.Repeat("x", 1025), maxBytes: 1024, want: ErrMalformedResponse},
		{name: "wrong session", body: strings.Replace(observationPageJSON("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "obs-1", ""), "session-1", "other-session", 1), maxBytes: 4096, want: ErrMalformedResponse},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				fmt.Fprint(writer, test.body)
			}))
			defer server.Close()
			client := mustTestClient(t, server.URL, 2, 2, test.maxBytes)
			_, err := client.ListSessionObservations(context.Background(), validTraceQuery())
			if !errors.Is(err, test.want) {
				t.Fatalf("list error = %v, want %v", err, test.want)
			}
		})
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprint(writer, observationPageJSON("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "obs-1", "same-cursor"))
	}))
	defer server.Close()
	client := mustTestClient(t, server.URL, 2, 3, 4096)
	if _, err := client.ListSessionObservations(context.Background(), validTraceQuery()); !errors.Is(err, ErrMalformedResponse) {
		t.Fatalf("repeated cursor error = %v", err)
	}
}

func TestClientRedactsCredentialsAndResponseBodiesFromErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(writer, `secret-test authorization: raw-provider-body`)
	}))
	defer server.Close()
	client := mustTestClient(t, server.URL, 2, 2, 4096)
	_, err := client.ListSessionObservations(context.Background(), validTraceQuery())
	if !errors.Is(err, ErrObservationRead) {
		t.Fatalf("read error = %v", err)
	}
	for _, forbidden := range []string{"public-test", "secret-test", "authorization", "raw-provider-body"} {
		if strings.Contains(strings.ToLower(err.Error()), strings.ToLower(forbidden)) {
			t.Fatalf("error leaked %q: %v", forbidden, err)
		}
	}
}

func TestClientRequiresInjectedTimeoutAndBoundedUTCQuery(t *testing.T) {
	if _, err := NewClient(ClientConfig{BaseURL: "https://cloud.langfuse.com", PublicKey: "public", SecretKey: "secret", HTTPClient: &http.Client{}}); !errors.Is(err, ErrInvalidClientConfig) {
		t.Fatalf("missing timeout error = %v", err)
	}
	if _, err := NewClient(ClientConfig{BaseURL: "http://langfuse.example", PublicKey: "public", SecretKey: "secret", HTTPClient: &http.Client{Timeout: time.Second}}); !errors.Is(err, ErrInvalidClientConfig) {
		t.Fatalf("cleartext remote base URL error = %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprint(writer, `{"data":[],"meta":{"cursor":null}}`)
	}))
	defer server.Close()
	client := mustTestClient(t, server.URL, 2, 2, 4096)
	query := validTraceQuery()
	query.ToStartTime = query.FromStartTime
	if _, err := client.ListSessionObservations(context.Background(), query); !errors.Is(err, ErrInvalidTraceQuery) {
		t.Fatalf("invalid query error = %v", err)
	}
}

func mustTestClient(t *testing.T, baseURL string, pageSize, maxPages int, maxBytes int64) *Client {
	t.Helper()
	client, err := NewClient(ClientConfig{
		BaseURL: baseURL, PublicKey: "public-test", SecretKey: "secret-test",
		HTTPClient: &http.Client{Timeout: time.Second}, PageSize: pageSize,
		MaxPages: maxPages, MaxResponseBytes: maxBytes,
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	return client
}

func validTraceQuery() domain.TraceObservationQuery {
	return domain.TraceObservationQuery{
		SessionID: "session-1", FromStartTime: clientTestTime,
		ToStartTime: clientTestTime.Add(time.Hour),
	}
}

func observationPageJSON(traceID, observationID, cursor string) string {
	metaCursor := "null"
	if cursor != "" {
		metaCursor = fmt.Sprintf("%q", cursor)
	}
	return fmt.Sprintf(
		`{"data":[{"id":%q,"traceId":%q,"startTime":"2026-07-22T10:01:00Z","endTime":"2026-07-22T10:02:00Z","projectId":"project-1","parentObservationId":null,"type":"SPAN","name":"Codex Turn","level":"DEFAULT","statusMessage":null,"version":null,"environment":"development","bookmarked":false,"public":false,"userId":null,"sessionId":"session-1"}],"meta":{"cursor":%s}}`,
		observationID, traceID, metaCursor,
	)
}
