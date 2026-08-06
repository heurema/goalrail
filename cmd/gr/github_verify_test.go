package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The authoritative shared path is a composition — event decoding, packet
// seeding, provider collection, range freezing, verification, output and exit
// code — and none of it was exercised through the command itself. The proof for
// the change that introduced it claimed otherwise; this is that claim's check.
func TestGitHubVerifyEntrypointRefusesIncompleteInvocations(t *testing.T) {
	for _, fixture := range []struct {
		name      string
		arguments []string
		wants     string
	}{
		{name: "no arguments", arguments: []string{"github-verify"}, wants: "requires"},
		{name: "no json", arguments: []string{
			"github-verify", "--repo", ".", "--base", strings.Repeat("a", 40),
			"--head", strings.Repeat("b", 40), "--event-name", "pull_request", "--event", "event.json",
		}, wants: "requires"},
		{name: "positional argument", arguments: []string{"github-verify", "extra", "--json"}, wants: "requires"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := run(context.Background(), fixture.arguments, strings.NewReader(""), &stdout, &stderr, nil)
			if err == nil {
				t.Fatal("an incomplete invocation was accepted")
			}
			if !strings.Contains(err.Error(), fixture.wants) {
				t.Fatalf("refusal = %v, want it to name what is required", err)
			}
			if code := exitCodeOf(err); code != 2 {
				t.Fatalf("exit code = %d, want the usage category", code)
			}
			if stdout.Len() != 0 {
				t.Fatalf("a refused invocation wrote a result: %q", stdout.String())
			}
		})
	}
}

// The composition is wired in the order the design fixes, and the order itself
// is the property worth asserting: the repository's own authority is
// established before the provider is contacted at all, so a range Goalrail
// cannot vouch for never becomes a provider request.
func TestGitHubVerifyEstablishesRepositoryAuthorityBeforeTouchingTheProvider(t *testing.T) {
	repository := scratchRepository(t)
	if _, _, err := runCommand(t, "init", "--repo", repository, "--scaffold", "codex"); err != nil {
		t.Fatal(err)
	}
	gitCommand(t, repository, "add", "-A")
	gitCommand(t, repository, "commit", "-m", "initialize")
	base := strings.TrimSpace(gitOutput(t, repository, "rev-parse", "HEAD"))
	writeFile(t, filepath.Join(repository, "internal", "app.go"), "package internal\n")
	gitCommand(t, repository, "add", "-A")
	// No work-unit trailer: the repository cannot say which work unit this range
	// belongs to, which is a question no provider can answer.
	gitCommand(t, repository, "commit", "-m", "add material code")
	head := strings.TrimSpace(gitOutput(t, repository, "rev-parse", "HEAD"))

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		writer.Write([]byte(`{}`))
	}))
	defer server.Close()

	eventPath := filepath.Join(t.TempDir(), "event.json")
	event := `{"action":"opened","number":1,"repository":{"full_name":"acme/widget"},` +
		`"pull_request":{"number":1,"base":{"sha":"` + base + `"},"head":{"sha":"` + head + `"}}}`
	if err := os.WriteFile(eventPath, []byte(event), 0o600); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	err := run(context.Background(), []string{
		"github-verify", "--repo", repository, "--base", base, "--head", head,
		"--event-name", "pull_request", "--event", eventPath, "--api", server.URL, "--json",
	}, strings.NewReader(""), &out, &errOut, nil)
	if err == nil {
		t.Fatal("a range with no work-unit authority produced a verdict")
	}
	if requests != 0 {
		t.Fatalf("the provider was contacted %d times before repository authority existed", requests)
	}
	if out.Len() != 0 {
		t.Fatalf("a refused composition wrote a result: %q", out.String())
	}
	if code := exitCodeOf(err); code != 2 {
		t.Fatalf("exit code = %d, want the usage category for an input failure", code)
	}
}

// The event decoder is reached through the command, so an unsupported workflow
// event is refused there rather than becoming an admission question.
func TestGitHubVerifyRefusesAnUnsupportedWorkflowEvent(t *testing.T) {
	eventPath := filepath.Join(t.TempDir(), "event.json")
	event := `{"action":"opened","number":1,"repository":{"full_name":"acme/widget"},` +
		`"pull_request":{"number":1,"base":{"sha":"` + strings.Repeat("a", 40) + `"},` +
		`"head":{"sha":"` + strings.Repeat("b", 40) + `"}}}`
	if err := os.WriteFile(eventPath, []byte(event), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	err := run(context.Background(), []string{
		"github-verify", "--repo", ".", "--base", strings.Repeat("a", 40), "--head", strings.Repeat("b", 40),
		"--event-name", "check_run", "--event", eventPath, "--json",
	}, strings.NewReader(""), &out, &errOut, nil)
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("an unsupported event was not refused by the decoder: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("a refused event wrote a result: %q", out.String())
	}
}
