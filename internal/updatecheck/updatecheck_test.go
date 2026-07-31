package updatecheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// TestOnlyPlainReleasesAreComparable pins the whitelist. Every other shape the
// toolchain can stamp is outside the comparison rather than guessed at, because
// nothing can be concluded from comparing a release against a build that is not
// one.
func TestOnlyPlainReleasesAreComparable(t *testing.T) {
	for _, release := range []string{"v0.1.1", "v1.0.0", "v0.0.1", "v10.20.30"} {
		if !IsRelease(release) {
			t.Errorf("%q is a release and was not recognized", release)
		}
	}
	for _, other := range []string{
		"v0.1.2-0.20260730182101-c49814687b66", // between tags
		"v0.1.1+dirty",                         // a modified tree
		"v0.1.1-rc.1",                          // a prerelease, publishable under a v* trigger
		"v2.0.0+incompatible",
		"(devel)", // no version control behind the build
		"unknown", // no build information at all
		"0.1.1",   // no leading v
		"v0.1",    // two fields
		"v0.1.1.1",
		"v0.1.x",
		"v..",
		"",
		"v01.2.3", // zero-padding is not canonical, and padding is how a
		"v0.00.1", // "release tag" smuggles unbounded bytes into a report
		"v0.1.01",
	} {
		if IsRelease(other) {
			t.Errorf("%q is not a plain release and was treated as one", other)
		}
	}
}

// TestNewerOrdersNumericallyNotAsText pins the mistake every improvisation makes.
// Compared as text, v0.10.0 sorts below v0.9.0 and a user two releases behind is
// told they are current.
func TestNewerOrdersNumericallyNotAsText(t *testing.T) {
	for _, testCase := range []struct {
		latest, current string
		want            bool
	}{
		{"v0.10.0", "v0.9.0", true},
		{"v0.9.0", "v0.10.0", false},
		{"v0.1.11", "v0.1.9", true},
		{"v1.0.0", "v0.999.999", true},
		{"v0.1.2", "v0.1.1", true},
		{"v0.1.1", "v0.1.1", false},
		{"v0.1.0", "v0.1.1", false},
		// Neither side may be anything but a plain release.
		{"v0.2.0", "(devel)", false},
		{"v0.2.0", "v0.1.1+dirty", false},
		{"v0.2.0-rc.1", "v0.1.1", false},
	} {
		if got := Newer(testCase.latest, testCase.current); got != testCase.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", testCase.latest, testCase.current, got, testCase.want)
		}
	}
}

// TestDeclinedHonoursTheToolchainSettingWhereItActuallyLives is the test that
// matters most for the promise. `go env -w GOPROXY=off` writes to the Go
// environment file and leaves the process environment empty, so a check reading
// only the environment would conclude the machine was at its default and make
// exactly the request the user disabled.
func TestDeclinedHonoursTheToolchainSettingWhereItActuallyLives(t *testing.T) {
	configuration := t.TempDir()
	if err := os.MkdirAll(filepath.Join(configuration, "go"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configuration, "go", "env"), []byte("GOPROXY=off\nGOSUMDB=off\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	environment := Environment{
		Lookup:    func(string) string { return "" }, // nothing in the process environment
		ConfigDir: func() (string, error) { return configuration, nil },
	}
	reason, declined := Declined(environment)
	if !declined {
		t.Fatal("a GOPROXY=off written by `go env -w` was not honoured")
	}
	if reason == "" {
		t.Error("a declined check named no reason")
	}
}

// TestDeclinedReadsEachRefusal covers the rest of them, including the rule for a
// list: only the first entry decides, because a user who put their own mirror in
// front has said where their lookups go.
func TestDeclinedReadsEachRefusal(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		environment map[string]string
		want        bool
	}{
		{"nothing set", map[string]string{}, false},
		{"the default proxy list", map[string]string{"GOPROXY": "https://proxy.golang.org,direct"}, false},
		{"the public proxy with a trailing slash", map[string]string{"GOPROXY": "https://proxy.golang.org/"}, false},
		{"proxying turned off", map[string]string{"GOPROXY": "off"}, true},
		{"lookups going direct", map[string]string{"GOPROXY": "direct"}, true},
		{"a private mirror first", map[string]string{"GOPROXY": "https://mirror.internal,https://proxy.golang.org"}, true},
		{"a pipe-separated private mirror first", map[string]string{"GOPROXY": "https://mirror.internal|https://proxy.golang.org"}, true},
		{"the module marked private", map[string]string{"GOPRIVATE": "github.com/heurema/*"}, true},
		{"another module marked private", map[string]string{"GOPRIVATE": "example.com/*"}, false},
		{"the module excluded from proxying", map[string]string{"GONOPROXY": "github.com/heurema/goalrail"}, true},
		// The shapes below are the ones the first version missed. Each was
		// proven against module.MatchPrefixPatterns — the function the go
		// command itself resolves these settings with — before being pinned.
		{"the bare organization prefix, the canonical corporate value", map[string]string{"GOPRIVATE": "github.com/heurema"}, true},
		{"the bare prefix with a trailing slash", map[string]string{"GOPRIVATE": "github.com/heurema/"}, true},
		{"the bare host", map[string]string{"GOPRIVATE": "github.com"}, true},
		{"a glob over the prefix", map[string]string{"GOPRIVATE": "*.com/heurema"}, true},
		{"a list containing the bare prefix", map[string]string{"GOPRIVATE": "example.com,github.com/heurema"}, true},
		{"a different organization", map[string]string{"GOPRIVATE": "github.com/other"}, false},
		{"a prefix that stops mid-element", map[string]string{"GOPRIVATE": "github.com/heurema/goal"}, false},
		{"a sibling that shares a prefix string", map[string]string{"GONOPROXY": "github.com/heurema-x"}, false},
		// And the GOPROXY shapes the go command normalizes before reading.
		{"the public proxy without a scheme", map[string]string{"GOPROXY": "proxy.golang.org,direct"}, false},
		{"a leading empty entry before the public proxy", map[string]string{"GOPROXY": ",https://proxy.golang.org,direct"}, false},
		{"our own switch", map[string]string{DisableEnvironment: "1"}, true},
		{"continuous integration", map[string]string{"CI": "true"}, true},
		{"a CI variable explicitly false", map[string]string{"CI": "false"}, false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			environment := Environment{
				Lookup:    func(name string) string { return testCase.environment[name] },
				ConfigDir: func() (string, error) { return t.TempDir(), nil },
			}
			if _, declined := Declined(environment); declined != testCase.want {
				t.Errorf("declined = %v, want %v", declined, testCase.want)
			}
		})
	}
}

// TestGoEnvFileIsSkippedWhenDisabled pins that GOENV=off means what it says.
func TestGoEnvFileIsSkippedWhenDisabled(t *testing.T) {
	configuration := t.TempDir()
	if err := os.MkdirAll(filepath.Join(configuration, "go"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configuration, "go", "env"), []byte("GOPROXY=off\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	environment := Environment{
		Lookup:    func(name string) string { return map[string]string{"GOENV": "off"}[name] },
		ConfigDir: func() (string, error) { return configuration, nil },
	}
	if _, declined := Declined(environment); declined {
		t.Error("the environment file was read although GOENV=off disables it")
	}
}

// TestLatestAcceptsOnlyAnAnswerToTheQuestionAsked walks the failure shapes the
// source actually produces. A module with no tags answers 200 with a
// pseudo-version, so a status code is not an answer.
func TestLatestAcceptsOnlyAnAnswerToTheQuestionAsked(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		handler http.HandlerFunc
		want    string
	}{
		{"a release", func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]string{"Version": "v9.9.9"})
		}, "v9.9.9"},
		{"a pseudo-version from a tagless module", func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]string{"Version": "v0.0.0-20250915201037-7f05d217867b"})
		}, ""},
		{"an error the source explains at length", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found: /tmp/gopath/pkg/mod/cache: fatal: could not read Username"))
		}, ""},
		{"an empty body", func(w http.ResponseWriter, _ *http.Request) {}, ""},
		{"something that is not JSON", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("<html>maintenance</html>"))
		}, ""},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(testCase.handler)
			defer server.Close()
			version, err := latestFrom(context.Background(), server.Client(), server.URL)
			if testCase.want == "" {
				if err == nil {
					t.Fatalf("accepted %q as an answer", version)
				}
				return
			}
			if err != nil {
				t.Fatalf("rejected a good answer: %v", err)
			}
			if version != testCase.want {
				t.Errorf("read %q, want %q", version, testCase.want)
			}
		})
	}
}

// TestTheRequestSaysNothingAboutTheInstallation pins the disclosure boundary:
// no agent string of ours, no query parameter, no header the standard library
// would not have sent anyway.
func TestTheRequestSaysNothingAboutTheInstallation(t *testing.T) {
	var seen *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Clone(context.Background())
		_ = json.NewEncoder(w).Encode(map[string]string{"Version": "v9.9.9"})
	}))
	defer server.Close()

	if _, err := latestFrom(context.Background(), server.Client(), server.URL); err != nil {
		t.Fatal(err)
	}
	if seen == nil {
		t.Fatal("no request was observed")
	}
	// Membership, not a prefix check: "Go-http-client/1.1 gr/1" would pass a
	// prefix check while announcing the tool. The two values are the standard
	// library's own defaults for the two protocols it can negotiate.
	if agent := seen.Header.Get("User-Agent"); agent != "Go-http-client/1.1" && agent != "Go-http-client/2.0" {
		t.Errorf("the request carried the agent string %q; it must be whatever the standard library sends and nothing of ours", agent)
	}
	if query := seen.URL.RawQuery; query != "" {
		t.Errorf("the request carried a query parameter: %q", query)
	}
	for name := range seen.Header {
		switch name {
		case "User-Agent", "Accept-Encoding":
		default:
			t.Errorf("the request carried the header %q, which the toolchain would not have sent", name)
		}
	}
}

// TestAnAnswerIsRememberedForADay pins the cache: a second check inside the
// interval asks nothing, and an answer older than the interval is refreshed.
func TestAnAnswerIsRememberedForADay(t *testing.T) {
	stateRoot := t.TempDir()
	if _, _, ok := readCache(stateRoot); ok {
		t.Fatal("an empty state root answered from a cache")
	}
	writeCache(stateRoot, "v0.1.1")
	version, at, ok := readCache(stateRoot)
	if !ok || version != "v0.1.1" {
		t.Fatalf("read back %q, ok=%v", version, ok)
	}
	if time.Since(at) > time.Minute {
		t.Errorf("a freshly written answer is dated %s", at)
	}

	// The file's own modification time is the freshness test, so ageing the file
	// ages the answer.
	stale := time.Now().Add(-Interval - time.Hour)
	if err := os.Chtimes(filepath.Join(stateRoot, cacheName), stale, stale); err != nil {
		t.Fatal(err)
	}
	_, at, ok = readCache(stateRoot)
	if !ok {
		t.Fatal("an aged answer became unreadable")
	}
	if time.Since(at) < Interval {
		t.Error("an aged answer still reads as fresh")
	}
}

// TestACorruptCacheIsNotAnAnswer pins that a file someone edited, or a partial
// write, sends the check back to the source rather than into a bad comparison.
func TestACorruptCacheIsNotAnAnswer(t *testing.T) {
	for _, contents := range []string{"", "{", `{"version":"(devel)"}`, `{"version":""}`} {
		stateRoot := t.TempDir()
		if err := os.WriteFile(filepath.Join(stateRoot, cacheName), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
		if version, _, ok := readCache(stateRoot); ok {
			t.Errorf("cache %q was accepted as %q", contents, version)
		}
	}
}

// TestADeclinedCheckNeverReachesTheTransport pins the order: refusals are read
// before anything is dialled, so a refusing user's machine makes no request even
// if the source is reachable.
func TestADeclinedCheckNeverReachesTheTransport(t *testing.T) {
	asked := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { asked = true }))
	defer server.Close()

	check := New(t.TempDir(), Environment{
		Lookup:    func(name string) string { return map[string]string{DisableEnvironment: "1"}[name] },
		ConfigDir: func() (string, error) { return t.TempDir(), nil },
	})
	if _, _, err := check(context.Background()); err == nil {
		t.Fatal("a declined check reported success")
	}
	if asked {
		t.Error("a declined check still made a request")
	}
}

// TestTheRequestSaysNothingOverHTTP2 repeats the disclosure check on the
// protocol the real proxy actually negotiates, where the standard library's
// default agent string differs.
func TestTheRequestSaysNothingOverHTTP2(t *testing.T) {
	var seen *http.Request
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Clone(context.Background())
		_ = json.NewEncoder(w).Encode(map[string]string{"Version": "v9.9.9"})
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	if _, err := latestFrom(context.Background(), server.Client(), server.URL); err != nil {
		t.Fatal(err)
	}
	if seen == nil {
		t.Fatal("no request was observed")
	}
	if seen.ProtoMajor != 2 {
		t.Fatalf("the stand-in negotiated HTTP/%d; this test exists for HTTP/2", seen.ProtoMajor)
	}
	if agent := seen.Header.Get("User-Agent"); agent != "Go-http-client/2.0" {
		t.Errorf("the request carried the agent string %q over HTTP/2", agent)
	}
	if query := seen.URL.RawQuery; query != "" {
		t.Errorf("the request carried a query parameter: %q", query)
	}
	for name := range seen.Header {
		switch name {
		case "User-Agent", "Accept-Encoding":
		default:
			t.Errorf("the request carried the header %q, which the toolchain would not have sent", name)
		}
	}
}

// TestARedirectIsAFailureNotAHop pins that one request means one: the
// redirected-to host receives nothing and the check reports failure, so the
// report can never name a source that did not actually answer.
func TestARedirectIsAFailureNotAHop(t *testing.T) {
	followed := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { followed = true }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer source.Close()

	if version, err := latestFrom(context.Background(), source.Client(), source.URL); err == nil {
		t.Fatalf("a redirect was followed to an answer: %q", version)
	}
	if followed {
		t.Error("the redirected-to host received a request")
	}
}

// TestTheComposedCheckAsksOnlyWhenTheCacheIsDue exercises New's actual wiring
// against a stand-in source that counts requests — the paths the primitive
// tests cannot see.
func TestTheComposedCheckAsksOnlyWhenTheCacheIsDue(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		_ = json.NewEncoder(w).Encode(map[string]string{"Version": "v9.9.9"})
	}))
	defer server.Close()
	stateRoot := t.TempDir()
	environment := Environment{
		Lookup:    func(string) string { return "" },
		ConfigDir: func() (string, error) { return t.TempDir(), nil },
	}
	check := newFrom(stateRoot, environment, server.Client(), server.URL)

	version, _, err := check(context.Background())
	if err != nil || version != "v9.9.9" {
		t.Fatalf("first ask: %q, %v", version, err)
	}
	if requests != 1 {
		t.Fatalf("first ask made %d requests", requests)
	}
	if _, _, err := check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Errorf("a fresh cache still asked: %d requests", requests)
	}

	stale := time.Now().Add(-Interval - time.Hour)
	if err := os.Chtimes(filepath.Join(stateRoot, cacheName), stale, stale); err != nil {
		t.Fatal(err)
	}
	if _, _, err := check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Errorf("an aged cache did not refresh: %d requests", requests)
	}
	if _, at, ok := readCache(stateRoot); !ok || time.Since(at) > time.Minute {
		t.Error("the refresh did not rewrite the cache")
	}
}

// TestAPlantedCacheCannotHangOrRedirectTheDiagnosis pins the file discipline
// this repository already applies to the escalation artifact: a FIFO must not
// block, a symlink must not be followed, and both read as a plain cache miss.
func TestAPlantedCacheCannotHangOrRedirectTheDiagnosis(t *testing.T) {
	fifoRoot := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(fifoRoot, cacheName), 0o600); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, _, ok := readCache(fifoRoot); ok {
			t.Error("a FIFO was read as a cache")
		}
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a planted FIFO hung the cache read")
	}

	linkRoot := t.TempDir()
	secret := filepath.Join(linkRoot, "elsewhere.json")
	if err := os.WriteFile(secret, []byte(`{"version":"v9.9.9"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(linkRoot, cacheName)); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := readCache(linkRoot); ok {
		t.Error("a symlinked cache was followed")
	}
	// And the write side refuses the same plant, so the target survives.
	writeCache(linkRoot, "v0.1.1")
	contents, err := os.ReadFile(secret)
	if err != nil || string(contents) != `{"version":"v9.9.9"}` {
		t.Errorf("a planted symlink redirected the cache write: %s, %v", contents, err)
	}
}
