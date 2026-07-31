// Package updatecheck asks whether a newer Goalrail release exists.
//
// It is the only package `gr` links that can open a connection, and that is the
// point: the binary carried no transport at all until this check existed, so the
// guarantee that nothing else reaches the network is kept by nothing else
// importing this package and by the diagnosis being handed a function rather
// than constructing one.
//
// The question is asked of the Go module proxy, which already answers it for
// every `go install` of this module: one request, no credential, no quota, and
// no header of our own. A request that named the tool would be the single bit
// announcing an installation, which is exactly what the toolchain does not send.
package updatecheck

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ModulePath is the module the proxy is asked about.
const ModulePath = "github.com/heurema/goalrail"

// Source is what a report names as the counterparty it asked. The report says
// this where the user is, because the request itself says nothing.
const Source = "proxy.golang.org"

// PublicProxy is the default the Go toolchain ships with. The check proceeds
// only when a user's configuration still puts this first.
const PublicProxy = "https://proxy.golang.org"

// DisableEnvironment turns the check off on its own, for a user who wants their
// toolchain configuration untouched.
const DisableEnvironment = "GR_NO_UPDATE_CHECK"

// Timeout bounds the one request. The measured warm answer is a quarter of a
// second; anything past this is a network worth giving up on rather than a
// diagnosis worth delaying.
const Timeout = 3 * time.Second

// Interval is how long an answer is reused. Every comparable tool settled on a
// day, and nothing here argues for a different number.
const Interval = 24 * time.Hour

// ErrDeclined reports that no request was made because the user had already
// said not to. It is distinct from a failure on purpose: a declined check is
// the tool obeying, and a failed one is the tool trying and not getting through.
var ErrDeclined = errors.New("update check declined")

// DeclinedError carries which refusal applied, so the report can say which one.
type DeclinedError struct{ Reason string }

func (e DeclinedError) Error() string { return "update check declined: " + e.Reason }
func (e DeclinedError) Unwrap() error { return ErrDeclined }

// Environment is what the check needs to read about the machine. Both are
// injectable so a test never depends on the machine it runs on.
type Environment struct {
	Lookup    func(string) string
	ConfigDir func() (string, error)
}

func (e Environment) lookup(name string) string {
	if e.Lookup == nil {
		return os.Getenv(name)
	}
	return e.Lookup(name)
}

func (e Environment) configDir() (string, error) {
	if e.ConfigDir == nil {
		return os.UserConfigDir()
	}
	return e.ConfigDir()
}

// Check is the shape the diagnosis is handed. It returns the newest released
// version and when that answer was obtained, which may be earlier than now when
// it came from the cache.
type Check func(context.Context) (version string, checkedAt time.Time, err error)

// New composes the check the shipped command uses: refusals first, then the
// cache, then one request. It is constructed in exactly one place, because
// nothing that is not handed one can call it.
func New(stateRoot string, environment Environment) Check {
	return newFrom(stateRoot, environment, nil, PublicProxy)
}

// newFrom is New with the transport as parameters, so the cache-hit and
// refresh paths can be exercised against a stand-in source counting requests.
func newFrom(stateRoot string, environment Environment, client *http.Client, base string) Check {
	return func(ctx context.Context) (string, time.Time, error) {
		if reason, declined := Declined(environment); declined {
			return "", time.Time{}, DeclinedError{Reason: reason}
		}
		if version, at, ok := readCache(stateRoot); ok && time.Since(at) < Interval {
			return version, at, nil
		}
		version, err := latestFrom(ctx, client, base)
		if err != nil {
			return "", time.Time{}, err
		}
		at := writeCache(stateRoot, version)
		return version, at, nil
	}
}

// Declined reports whether the user has already refused this traffic, and which
// refusal applied.
//
// The toolchain's own settings come first. A user who ran `go env -w
// GOPROXY=off` has said their module lookups stay home; asking anyway by hand
// would override an instruction they believe they gave.
func Declined(environment Environment) (string, bool) {
	if environment.lookup(DisableEnvironment) != "" {
		return DisableEnvironment + " is set", true
	}
	if continuousIntegration(environment) {
		return "this is continuous integration", true
	}
	if proxy := goSetting("GOPROXY", environment); proxy != "" && !startsWithPublicProxy(proxy) {
		return "GOPROXY does not start with the public proxy", true
	}
	for _, name := range []string{"GOPRIVATE", "GONOPROXY"} {
		if patterns := goSetting(name, environment); matchesModule(patterns) {
			return name + " covers this module", true
		}
	}
	return "", false
}

// continuousIntegration keeps automated runs from asking. The variables are the
// ones build systems set for exactly this purpose.
func continuousIntegration(environment Environment) bool {
	for _, name := range []string{"CI", "CONTINUOUS_INTEGRATION", "BUILD_NUMBER", "GITHUB_ACTIONS"} {
		if value := environment.lookup(name); value != "" && value != "0" && value != "false" {
			return true
		}
	}
	return false
}

// goSetting resolves a Go environment variable the way the `go` command does:
// the process environment first, then the environment file it writes.
//
// This matters more than it looks. `go env -w GOPROXY=off` — the documented way
// to set it persistently — writes only to that file and leaves the process
// environment empty, so a check reading the environment alone would conclude the
// machine was at its default and make the very request the user disabled.
//
// The `go` binary is never executed to find this out: a user who installed a
// release binary need not have a toolchain at all.
func goSetting(name string, environment Environment) string {
	if value := environment.lookup(name); value != "" {
		return value
	}
	return goEnvFile(name, environment)
}

func goEnvFile(name string, environment Environment) string {
	path := environment.lookup("GOENV")
	switch path {
	case "off":
		return ""
	case "":
		configuration, err := environment.configDir()
		if err != nil {
			return ""
		}
		path = filepath.Join(configuration, "go", "env")
	}
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	scanner := bufio.NewScanner(io.LimitReader(file, 1<<20))
	for scanner.Scan() {
		key, value, found := strings.Cut(scanner.Text(), "=")
		if found && strings.TrimSpace(key) == name {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// startsWithPublicProxy decides what "directs lookups elsewhere" means for a
// list-valued setting. Only the first entry decides: a user who put their own
// mirror in front has said where their lookups go, even when the public proxy
// follows as a fallback. `off` and `direct` decline for the same reason.
//
// The first entry is resolved the way the go command's own proxy list is:
// empty tokens are skipped and a bare host containing a dot gains the https
// scheme, so `GOPROXY=proxy.golang.org,direct` reads as the default it is
// rather than declining with a reason that would be false under the
// toolchain's semantics.
func startsWithPublicProxy(value string) bool {
	for _, token := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == '|' }) {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if strings.Contains(token, ".") && !strings.Contains(token, ":/") && !strings.HasPrefix(token, "/") {
			token = "https://" + token
		}
		return strings.TrimRight(token, "/") == PublicProxy
	}
	return false
}

// matchesModule mirrors the matching the go command applies to GOPRIVATE and
// GONOPROXY — module.MatchPrefixPatterns — because these are the toolchain's
// settings and honouring them means agreeing with the toolchain about what they
// cover. A pattern is a glob over a module-path *prefix* with the same number
// of path elements: `github.com/heurema` covers this module, and so does
// `github.com` alone, which is exactly how a corporate exclusion is usually
// written. The first shipped version matched the full path only and missed the
// bare-prefix form entirely — the review caught it against the real toolchain.
//
// path.Match, not filepath.Match: the latter switches separator by platform.
// No token trimming, no case folding: the go command does neither, and
// diverging in either direction means disagreeing with it about coverage.
func matchesModule(patterns string) bool {
	for _, pattern := range strings.Split(patterns, ",") {
		pattern = strings.TrimSuffix(pattern, "/")
		if pattern == "" {
			continue
		}
		elements := strings.Count(pattern, "/") + 1
		prefix := ModulePath
		if fields := strings.SplitN(ModulePath, "/", elements+1); len(fields) > elements {
			prefix = strings.Join(fields[:elements], "/")
		}
		if matched, err := path.Match(pattern, prefix); err == nil && matched {
			return true
		}
	}
	return false
}

// latest asks the proxy for the newest release. The proxy answers with the
// newest stable version, prereleases already filtered, so there is no version
// sorting here to get wrong. It is unexported on purpose: New is the only
// route to the transport from outside, so the confinement tests have one
// symbol to hold.
//
// Nothing about the request identifies Goalrail: no agent string, no query
// parameter, no header of our own. What goes out is what the standard library
// would have sent anyway, which is what the `go` command sends for this same
// module.
func latest(ctx context.Context, client *http.Client) (string, error) {
	return latestFrom(ctx, client, PublicProxy)
}

// latestFrom is Latest with the source as a parameter, so a test can stand one
// up locally and observe the request that goes out rather than trusting a
// description of it.
func latestFrom(ctx context.Context, client *http.Client, base string) (string, error) {
	if client == nil {
		client = &http.Client{Timeout: Timeout}
	}
	// One request means one: a redirect would send it to another host while the
	// report still named the source it was addressed to, so a redirect is a
	// failure rather than a hop.
	redirectless := *client
	redirectless.CheckRedirect = func(*http.Request, []*http.Request) error {
		return errors.New("the source redirected")
	}
	client = &redirectless
	url := fmt.Sprintf("%s/%s/@latest", base, ModulePath)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		// The proxy's error bodies carry its own internal paths and git errors.
		// None of that is a Goalrail diagnostic, so none of it is read.
		return "", fmt.Errorf("the source answered %d", response.StatusCode)
	}
	var answer struct {
		Version string `json:"Version"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<16)).Decode(&answer); err != nil {
		return "", fmt.Errorf("the source answered something unreadable")
	}
	if !IsRelease(answer.Version) {
		// A module with no tags answers 200 with a pseudo-version. That is not
		// an answer to this question.
		return "", fmt.Errorf("the source named no release")
	}
	return answer.Version, nil
}

// IsRelease reports whether a version is a plain release tag: `v` and three
// decimal fields, nothing after.
//
// Everything else the toolchain can stamp — a pseudo-version between tags, a
// `+dirty` marker, a prerelease suffix, `(devel)`, `unknown` — is outside the
// comparison rather than guessed at. The release workflow triggers on `v*`, so a
// prerelease tag is publishable and a binary built from one reports it verbatim;
// the rule is applied to both sides for that reason.
func IsRelease(version string) bool {
	_, ok := parseRelease(version)
	return ok
}

// Newer reports whether latest is a later release than current. Both must be
// plain release tags; anything else answers false.
func Newer(latest, current string) bool {
	left, ok := parseRelease(latest)
	if !ok {
		return false
	}
	right, ok := parseRelease(current)
	if !ok {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return left[index] > right[index]
		}
	}
	return false
}

// parseRelease is the whole ordering, written out because the standard library
// orders nothing of this shape and a dependency is not available here. Comparing
// these as strings would put v0.10.0 below v0.9.0, which is the mistake this
// exists to make impossible.
func parseRelease(version string) ([3]int, bool) {
	var fields [3]int
	rest, found := strings.CutPrefix(version, "v")
	if !found {
		return fields, false
	}
	parts := strings.Split(rest, ".")
	if len(parts) != 3 {
		return fields, false
	}
	for index, part := range parts {
		if part == "" || strings.TrimLeft(part, "0123456789") != "" {
			return fields, false
		}
		// Canonical release tags have no zero-padding, and the padding is not
		// harmless: a field can be padded to tens of kilobytes and still parse,
		// which would carry that many attacker-chosen bytes into the report.
		if len(part) > 1 && part[0] == '0' {
			return fields, false
		}
		number, err := strconv.Atoi(part)
		if err != nil {
			return fields, false
		}
		fields[index] = number
	}
	return fields, true
}

// cacheName is the one file this check leaves behind. It holds a public fact —
// the newest released version — and nothing about the user or the repository.
const cacheName = "update-check.json"

type cached struct {
	Version string `json:"version"`
}

// cacheBound is more than any honest cache needs: the file holds one version
// string in one small JSON object.
const cacheBound = 4096

// readCache returns the remembered answer and when it was written. The file's
// modification time is the freshness test, so no expiry is written into a file
// a user might copy between machines.
//
// It opens with the same discipline the escalation artifact taught this
// repository: O_NOFOLLOW so a planted symlink is not followed, O_NONBLOCK so a
// planted FIFO cannot hang the diagnosis before any timeout applies, a
// regular-file check on the open descriptor, and a size bound. Anything
// unusual is simply a cache miss.
func readCache(stateRoot string) (string, time.Time, bool) {
	if stateRoot == "" {
		return "", time.Time{}, false
	}
	file, err := os.OpenFile(filepath.Join(stateRoot, cacheName), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", time.Time{}, false
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > cacheBound {
		return "", time.Time{}, false
	}
	contents, err := io.ReadAll(io.LimitReader(file, cacheBound))
	if err != nil {
		return "", time.Time{}, false
	}
	var remembered cached
	if err := json.Unmarshal(contents, &remembered); err != nil || !IsRelease(remembered.Version) {
		return "", time.Time{}, false
	}
	return remembered.Version, info.ModTime(), true
}

// writeCache remembers the answer, and reports the time to show for it. A cache
// that cannot be written costs nothing but a request next time, so its failure
// is not the user's problem.
func writeCache(stateRoot, version string) time.Time {
	now := time.Now().UTC()
	if stateRoot == "" {
		return now
	}
	contents, err := json.Marshal(cached{Version: version})
	if err != nil {
		return now
	}
	if err := os.MkdirAll(stateRoot, 0o700); err != nil {
		return now
	}
	// The same refusal to follow a planted link applies on the way out.
	file, err := os.OpenFile(filepath.Join(stateRoot, cacheName), os.O_WRONLY|os.O_CREATE|os.O_TRUNC|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return now
	}
	defer file.Close()
	_, _ = file.Write(contents)
	return now
}
