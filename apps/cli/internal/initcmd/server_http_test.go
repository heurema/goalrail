package initcmd

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
	"github.com/heurema/goalrail/apps/cli/internal/spine"
)

func TestInitHTTPClientNilUsesDefaultTimeout(t *testing.T) {
	t.Parallel()

	client, ok := initHTTPClient(nil).(*http.Client)
	if !ok {
		t.Fatalf("initHTTPClient(nil) type = %T, want *http.Client", initHTTPClient(nil))
	}
	if client.Timeout != defaultInitHTTPTimeout {
		t.Fatalf("default init HTTP timeout = %s, want %s", client.Timeout, defaultInitHTTPTimeout)
	}
}

func TestInitServerCallsRetry5xxAndCloseRetryBody(t *testing.T) {
	t.Parallel()

	session := validSession("https://goalrail.example")
	tests := []struct {
		name     string
		wantPath string
		success  string
		call     func(context.Context, HTTPClient) error
	}{
		{
			name:     "repository_context_init",
			wantPath: "/v1/init/repository-context",
			success:  repositoryContextInitResponseJSON(true, true, "main"),
			call: func(ctx context.Context, client HTTPClient) error {
				_, err := postRepositoryContextInit(ctx, client, session, spine.RepositoryContextInitRequest{
					Provider:                    "github",
					RepositoryFullName:          "heurema/goalrail",
					RepositoryURL:               "git@github.com:heurema/goalrail.git",
					WorkflowBaseBranch:          "main",
					SuggestedProjectSlug:        "github-heurema-goalrail",
					SuggestedProjectDisplayName: "heurema/goalrail",
				})
				return err
			},
		},
		{
			name:     "project_repo_binding_init",
			wantPath: "/v1/projects/018f0000-0000-7000-8000-000000000003/repo-bindings/init",
			success:  `{"repo_binding_id":"018f0000-0000-7000-8000-000000000004","project_id":"018f0000-0000-7000-8000-000000000003","organization_id":"018f0000-0000-7000-8000-000000000002","provider":"github","repository_full_name":"heurema/goalrail","repository_url":"git@github.com:heurema/goalrail.git","provider_default_branch":"main","workflow_base_branch":"main","state":"active","created":true,"message":"Repository binding initialized."}`,
			call: func(ctx context.Context, client HTTPClient) error {
				_, err := postRepoBindingInit(ctx, client, session, "018f0000-0000-7000-8000-000000000003", spine.RepoBindingInitRequest{
					Provider:           "github",
					RepositoryFullName: "heurema/goalrail",
					RepositoryURL:      "git@github.com:heurema/goalrail.git",
					WorkflowBaseBranch: "main",
				})
				return err
			},
		},
		{
			name:     "repository_context_snapshot",
			wantPath: "/v1/repo-bindings/018f0000-0000-7000-8000-000000000004/context-snapshots",
			success:  repositoryContextSnapshotResponseJSON(true),
			call: func(ctx context.Context, client HTTPClient) error {
				_, err := postRepositoryContextSnapshot(ctx, client, session, "018f0000-0000-7000-8000-000000000004", spine.RepositoryContextSnapshotRequest{
					Source:        repositoryContextSnapshotSource,
					SchemaVersion: 1,
				})
				return err
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var retryBodyClosed atomic.Bool
			client := &queuedInitHTTPClient{results: []queuedInitHTTPResult{
				{response: initHTTPResponse(http.StatusServiceUnavailable, `{"error":{"code":"temporarily_unavailable","message":"try later"}}`, &retryBodyClosed)},
				{response: initHTTPResponse(http.StatusCreated, tt.success, nil)},
			}}

			if err := tt.call(context.Background(), client); err != nil {
				t.Fatalf("server call error = %v", err)
			}
			if got := len(client.requests); got != 2 {
				t.Fatalf("requests = %d, want 2 after 5xx retry", got)
			}
			for _, request := range client.requests {
				if request.URL.Path != tt.wantPath {
					t.Fatalf("request path = %s, want %s", request.URL.Path, tt.wantPath)
				}
			}
			if !retryBodyClosed.Load() {
				t.Fatal("retry response body was not closed")
			}
		})
	}
}

func TestInitServerCallRetriesTransientTransportError(t *testing.T) {
	t.Parallel()

	client := &queuedInitHTTPClient{results: []queuedInitHTTPResult{
		{err: transientInitNetError("temporary network timeout")},
		{response: initHTTPResponse(http.StatusCreated, repositoryContextInitResponseJSON(true, true, "main"), nil)},
	}}

	_, err := postRepositoryContextInit(context.Background(), client, validSession("https://goalrail.example"), spine.RepositoryContextInitRequest{
		Provider:           "github",
		RepositoryFullName: "heurema/goalrail",
		RepositoryURL:      "git@github.com:heurema/goalrail.git",
		WorkflowBaseBranch: "main",
	})
	if err != nil {
		t.Fatalf("postRepositoryContextInit() error = %v", err)
	}
	if got := len(client.requests); got != 2 {
		t.Fatalf("requests = %d, want 2 after transient transport retry", got)
	}
}

func TestInitServerCallDoesNotRetry4xx(t *testing.T) {
	t.Parallel()

	client := &queuedInitHTTPClient{results: []queuedInitHTTPResult{
		{response: initHTTPResponse(http.StatusConflict, `{"error":{"code":"repository_context_conflict","message":"existing binding differs"}}`, nil)},
	}}

	_, err := postRepositoryContextInit(context.Background(), client, validSession("https://goalrail.example"), spine.RepositoryContextInitRequest{
		Provider:           "github",
		RepositoryFullName: "heurema/goalrail",
		RepositoryURL:      "git@github.com:heurema/goalrail.git",
		WorkflowBaseBranch: "main",
	})
	if err == nil {
		t.Fatal("postRepositoryContextInit() error = nil, want conflict")
	}
	if got := len(client.requests); got != 1 {
		t.Fatalf("requests = %d, want 1 for non-retryable 4xx", got)
	}
	if got := exitcode.ForError(err); got != exitcode.Validation {
		t.Fatalf("exit code = %d, want validation", got)
	}
	if !strings.Contains(err.Error(), "repository_context_conflict") {
		t.Fatalf("error = %q, want server conflict code", err.Error())
	}
}

func TestInitServerCallRespectsContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := &queuedInitHTTPClient{results: []queuedInitHTTPResult{
		{response: initHTTPResponse(http.StatusCreated, repositoryContextInitResponseJSON(true, true, "main"), nil)},
	}}

	_, err := postRepositoryContextInit(ctx, client, validSession("https://goalrail.example"), spine.RepositoryContextInitRequest{
		Provider:           "github",
		RepositoryFullName: "heurema/goalrail",
		RepositoryURL:      "git@github.com:heurema/goalrail.git",
		WorkflowBaseBranch: "main",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("postRepositoryContextInit() error = %v, want context canceled", err)
	}
	if got := len(client.requests); got != 0 {
		t.Fatalf("requests = %d, want 0 after canceled context", got)
	}
}

func TestInitServerTimeoutErrorIsUserFacing(t *testing.T) {
	t.Parallel()

	client := &queuedInitHTTPClient{results: []queuedInitHTTPResult{
		{err: transientInitNetError("temporary network timeout")},
		{err: transientInitNetError("temporary network timeout")},
		{err: transientInitNetError("temporary network timeout")},
	}}

	_, err := postRepositoryContextInit(context.Background(), client, validSession("https://goalrail.example"), spine.RepositoryContextInitRequest{
		Provider:           "github",
		RepositoryFullName: "heurema/goalrail",
		RepositoryURL:      "git@github.com:heurema/goalrail.git",
		WorkflowBaseBranch: "main",
	})
	if err == nil {
		t.Fatal("postRepositoryContextInit() error = nil, want timeout")
	}
	if got := len(client.requests); got != defaultInitRetryAttempts {
		t.Fatalf("requests = %d, want %d", got, defaultInitRetryAttempts)
	}
	if !strings.Contains(err.Error(), "server request timed out or could not complete") {
		t.Fatalf("error = %q, want timeout wording", err.Error())
	}
}

type queuedInitHTTPClient struct {
	results  []queuedInitHTTPResult
	requests []*http.Request
}

type queuedInitHTTPResult struct {
	response *http.Response
	err      error
}

func (c *queuedInitHTTPClient) Do(request *http.Request) (*http.Response, error) {
	if request.Body != nil {
		_, _ = io.ReadAll(request.Body)
	}
	c.requests = append(c.requests, request)
	if len(c.requests) > len(c.results) {
		return nil, errors.New("unexpected init HTTP request")
	}
	result := c.results[len(c.requests)-1]
	return result.response, result.err
}

func initHTTPResponse(statusCode int, body string, closed *atomic.Bool) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       &trackingInitReadCloser{Reader: strings.NewReader(body), closed: closed},
		Header:     make(http.Header),
	}
}

type trackingInitReadCloser struct {
	*strings.Reader
	closed *atomic.Bool
}

func (r *trackingInitReadCloser) Close() error {
	if r.closed != nil {
		r.closed.Store(true)
	}
	return nil
}

type transientInitNetError string

func (e transientInitNetError) Error() string {
	return string(e)
}

func (e transientInitNetError) Timeout() bool {
	return true
}

func (e transientInitNetError) Temporary() bool {
	return true
}
