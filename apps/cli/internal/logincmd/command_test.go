package logincmd

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/cli/internal/authstore"
	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

func TestNormalizeServerURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "https", raw: " https://goalrail.example.com/ ", want: "https://goalrail.example.com"},
		{name: "http localhost", raw: "http://localhost:8080/", want: "http://localhost:8080"},
		{name: "keeps base path", raw: "https://example.com/goalrail/", want: "https://example.com/goalrail"},
		{name: "rejects missing scheme", raw: "goalrail.example.com", wantErr: true},
		{name: "rejects file scheme", raw: "file:///tmp/goalrail", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeServerURL(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NormalizeServerURL() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeServerURL() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeServerURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateCallbackState(t *testing.T) {
	code, err := ValidateCallbackState("state-1", url.Values{"code": {"one-time-code"}, "state": {"state-1"}})
	if err != nil {
		t.Fatalf("ValidateCallbackState() error = %v", err)
	}
	if code != "one-time-code" {
		t.Fatalf("code = %q, want one-time-code", code)
	}

	if _, err := ValidateCallbackState("state-1", url.Values{"code": {"one-time-code"}, "state": {"other"}}); err == nil {
		t.Fatal("ValidateCallbackState() state mismatch error = nil, want error")
	}
}

func TestRunRequiresServerURL(t *testing.T) {
	var stdout, stderr strings.Builder
	err := Run(context.Background(), term.New(&stdout, &stderr), nil, Options{})
	if exitcode.ForError(err) != exitcode.Usage {
		t.Fatalf("Run() error = %v, exit = %d, want usage", err, exitcode.ForError(err))
	}
}

func TestNoBrowserPrintsURLInsteadOfOpeningBrowser(t *testing.T) {
	var stdout, stderr strings.Builder
	opened := false
	err := Run(context.Background(), term.New(&stdout, &stderr), []string{"--no-browser", "http://localhost:8080"}, Options{
		Store:           fakeStore{},
		HTTPClient:      fakeHTTPClient{},
		OpenBrowser:     func(string) error { opened = true; return nil },
		skipWaitForTest: true,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if opened {
		t.Fatal("OpenBrowser was called with --no-browser")
	}
	if !strings.Contains(stdout.String(), "http://localhost:8080/cli/login") {
		t.Fatalf("stdout = %q, want printed browser URL", stdout.String())
	}
	if !strings.Contains(stdout.String(), "code_challenge=") {
		t.Fatalf("stdout = %q, want printed browser URL with code_challenge", stdout.String())
	}
	if strings.Contains(stdout.String(), "code_verifier") {
		t.Fatalf("stdout = %q, must not print code_verifier", stdout.String())
	}
}

func TestNoBrowserAfterServerURLMatchesUsage(t *testing.T) {
	var stdout, stderr strings.Builder
	opened := false
	err := Run(context.Background(), term.New(&stdout, &stderr), []string{"http://localhost:8080", "--no-browser"}, Options{
		Store:           fakeStore{},
		HTTPClient:      fakeHTTPClient{},
		OpenBrowser:     func(string) error { opened = true; return nil },
		skipWaitForTest: true,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if opened {
		t.Fatal("OpenBrowser was called with --no-browser")
	}
	if !strings.Contains(stdout.String(), "http://localhost:8080/cli/login") {
		t.Fatalf("stdout = %q, want printed browser URL", stdout.String())
	}
}

func TestNoBrowserLegacyFlagFormsMatchFlagSet(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantOpened bool
	}{
		{name: "single dash before server", args: []string{"-no-browser", "http://localhost:8080"}},
		{name: "single dash after server", args: []string{"http://localhost:8080", "-no-browser"}},
		{name: "single dash true value", args: []string{"-no-browser=true", "http://localhost:8080"}},
		{name: "double dash true value", args: []string{"http://localhost:8080", "--no-browser=true"}},
		{name: "double dash false value", args: []string{"--no-browser=false", "http://localhost:8080"}, wantOpened: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr strings.Builder
			opened := false
			err := Run(context.Background(), term.New(&stdout, &stderr), tt.args, Options{
				Store:           fakeStore{},
				HTTPClient:      fakeHTTPClient{},
				OpenBrowser:     func(string) error { opened = true; return nil },
				skipWaitForTest: true,
			})
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if opened != tt.wantOpened {
				t.Fatalf("OpenBrowser called = %v, want %v", opened, tt.wantOpened)
			}
		})
	}
}

func TestCodeChallengeS256(t *testing.T) {
	got := CodeChallengeS256("cli-code-verifier")
	if got == "" || strings.Contains(got, "=") {
		t.Fatalf("CodeChallengeS256() = %q, want unpadded base64url challenge", got)
	}
	if got != "T0LmpvR0zcDWcwvSHgKGAmsSP6iKl8nQN6EF23dJPiA" {
		t.Fatalf("CodeChallengeS256() = %q, want S256 challenge", got)
	}
}

func TestExchangeCodeSendsVerifier(t *testing.T) {
	var requestBody strings.Builder
	_, err := exchangeCode(context.Background(), fakeHTTPClient{requestBody: &requestBody}, "http://localhost:8080", "one-time-code", "state-1", "cli-code-verifier")
	if err != nil {
		t.Fatalf("exchangeCode() error = %v", err)
	}
	body := requestBody.String()
	if !strings.Contains(body, `"code_verifier":"cli-code-verifier"`) {
		t.Fatalf("request body = %q, want code_verifier", body)
	}
}

type fakeStore struct{}

func (fakeStore) Save(authstore.Session) error {
	return nil
}

type fakeHTTPClient struct {
	requestBody *strings.Builder
}

func (c fakeHTTPClient) Do(request *http.Request) (*http.Response, error) {
	if c.requestBody != nil && request.Body != nil {
		_, _ = io.Copy(c.requestBody, request.Body)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"user_id":"u","access_token":"a","refresh_token":"r","access_token_expires_at":"` + time.Date(2026, 5, 3, 12, 15, 0, 0, time.UTC).Format(time.RFC3339Nano) + `","token_type":"Bearer"}`)),
	}, nil
}
