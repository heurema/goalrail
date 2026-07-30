package ambient

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Scope is where a scaffold's registration lives.
//
// It is a property of the scaffold, not a preference. Registered inside a
// repository the hook is never invoked in unrelated sessions at all, so the
// boundary becomes a property of where it is installed rather than of Goalrail's
// own first act — which is why that is preferred wherever the scaffold permits
// it.
type Scope string

const (
	// ScopeUser is the scaffold's own configuration for all of the user's work.
	// The hook fires for every session the user starts anywhere, and the marker
	// check is what confines action to initialized repositories.
	ScopeUser Scope = "user"

	// ScopeRepository is a settings file inside one repository that the
	// repository cannot hand to anyone else.
	ScopeRepository Scope = "repository"
)

// TrustEvidence records what is actually established about a scaffold's review
// step, so a disclosure can never claim more than was found out.
type TrustEvidence string

const (
	// TrustGateObserved means a live session showed the scaffold refusing to run
	// a registered hook until the user trusts it.
	TrustGateObserved TrustEvidence = "gate-observed"

	// TrustGateObservedAbsent means a live session showed a registered hook
	// running with no approval step.
	TrustGateObservedAbsent TrustEvidence = "gate-observed-absent"

	// TrustGateDocumentedAbsent means the provider's documentation records no
	// approval step, and no live session has confirmed it.
	TrustGateDocumentedAbsent TrustEvidence = "gate-documented-absent"

	// TrustGateUnknown means neither observation nor documentation settles it.
	TrustGateUnknown TrustEvidence = "gate-unknown"
)

// scaffoldProfile is everything scaffold-specific in one place: where the
// registration goes, which events carry the meaning, which events an earlier
// arrangement wrote, and what is known about the trust gate. Keeping the three
// answers together stops a disclosure or an event set from drifting away from
// the scope it belongs to.
type scaffoldProfile struct {
	scope Scope

	// events are the events registered now.
	events []string

	// superseded are events an earlier arrangement registered and this one does
	// not. They are still read and still removed: a registration nobody looks at
	// keeps firing.
	superseded []string

	trust TrustEvidence

	// configHome is the directory whose presence means this scaffold is in use.
	configHome string
}

const (
	sessionStartEvent = "SessionStart"
	stopEvent         = "Stop"
	sessionEndEvent   = "SessionEnd"
)

// profiles is the fixed per-scaffold arrangement this version supports.
//
// The first scaffold registers at user scope because registering inside a
// repository is externally blocked there, and its stop-like event means the end
// of a session. The second registers inside the repository, and its stop-like
// event fires once per turn — so retention is bound to the event that fires when
// the session ends, and the per-turn event it used to name is carried as
// superseded rather than forgotten.
var profiles = map[Scaffold]scaffoldProfile{
	ScaffoldCodex: {
		scope:      ScopeUser,
		events:     []string{sessionStartEvent, stopEvent},
		trust:      TrustGateObserved,
		configHome: ".codex",
	},
	ScaffoldClaudeCode: {
		scope:      ScopeRepository,
		events:     []string{sessionStartEvent, sessionEndEvent},
		superseded: []string{stopEvent},
		trust:      TrustGateObservedAbsent,
		configHome: ".claude",
	},
}

func profileFor(scaffold Scaffold) (scaffoldProfile, error) {
	profile, known := profiles[scaffold]
	if !known {
		return scaffoldProfile{}, fmt.Errorf("unsupported scaffold %q", scaffold)
	}
	return profile, nil
}

// ScopeOf reports where a scaffold's registration belongs.
func ScopeOf(scaffold Scaffold) (Scope, error) {
	profile, err := profileFor(scaffold)
	if err != nil {
		return "", err
	}
	return profile.scope, nil
}

// TrustEvidenceOf reports what is established about a scaffold's review step.
func TrustEvidenceOf(scaffold Scaffold) TrustEvidence {
	profile, err := profileFor(scaffold)
	if err != nil {
		return TrustGateUnknown
	}
	return profile.trust
}

// managedEvents names the events a registration writes for one scaffold. Every
// event listed here must be present for the attachment to work, so presence,
// repair, and removal all walk the same list.
func managedEvents(scaffold Scaffold) []string {
	profile, err := profileFor(scaffold)
	if err != nil {
		return nil
	}
	return profile.events
}

// knownEvents names every event a registration of ours may occupy, current and
// superseded. Reading and removal use this: an event we no longer write can still
// hold a handler that fires.
func knownEvents(scaffold Scaffold) []string {
	profile, err := profileFor(scaffold)
	if err != nil {
		return nil
	}
	return append(append([]string(nil), profile.events...), profile.superseded...)
}

// SupersededEvents names events an earlier arrangement registered for a scaffold.
func SupersededEvents(scaffold Scaffold) []string {
	profile, err := profileFor(scaffold)
	if err != nil {
		return nil
	}
	return append([]string(nil), profile.superseded...)
}

// RepositorySettingsPath is the per-user, per-repository settings file a
// repository-scope registration is written to.
//
// It is deliberately the file the provider describes as the user's own rather
// than the shareable one beside it: a registration a commit could carry would
// install a command in every teammate's sessions on one user's consent.
const RepositorySettingsPath = ".claude/settings.local.json"

// Target is one concrete place a registration lives.
type Target struct {
	Scaffold Scaffold `json:"scaffold"`
	Scope    Scope    `json:"scope"`
	Path     string   `json:"path"`

	// Repository is set for a repository-scope target, so a report can say which
	// repository the registration belongs to.
	Repository string `json:"repository,omitempty"`
}

// RegistrationTarget returns where this scaffold's registration belongs.
//
// A repository-scope scaffold needs the repository root; a user-scope one needs
// the home directory. Passing the irrelevant one is harmless, which keeps callers
// that have both from having to know which applies.
func RegistrationTarget(scaffold Scaffold, home, repositoryRoot string) (Target, error) {
	profile, err := profileFor(scaffold)
	if err != nil {
		return Target{}, err
	}
	switch profile.scope {
	case ScopeRepository:
		if strings.TrimSpace(repositoryRoot) == "" {
			return Target{}, fmt.Errorf("scaffold %q registers per repository and needs a repository root", scaffold)
		}
		return Target{
			Scaffold:   scaffold,
			Scope:      ScopeRepository,
			Path:       filepath.Join(repositoryRoot, filepath.FromSlash(RepositorySettingsPath)),
			Repository: repositoryRoot,
		}, nil
	default:
		configPath, pathErr := ConfigPath(scaffold, home)
		if pathErr != nil {
			return Target{}, pathErr
		}
		return Target{Scaffold: scaffold, Scope: ScopeUser, Path: configPath}, nil
	}
}

// SupersededTarget returns the place a scaffold's registration used to live, and
// whether there is one.
//
// For a scaffold that moved into the repository, that is its user-scope
// configuration: a registration left there duplicates the repository-scope one,
// and every hook then fires twice. It is reported and removed by the consented
// command that owns user configuration — never rewritten by initialization.
func SupersededTarget(scaffold Scaffold, home string) (Target, bool) {
	profile, err := profileFor(scaffold)
	if err != nil || profile.scope != ScopeRepository {
		return Target{}, false
	}
	configPath, pathErr := ConfigPath(scaffold, home)
	if pathErr != nil {
		return Target{}, false
	}
	return Target{Scaffold: scaffold, Scope: ScopeUser, Path: configPath}, true
}

// DetectScaffolds returns the supported scaffolds whose own configuration
// directory exists under the given home, in the fixed supported order.
//
// Detection reads configuration presence and never the session environment. A
// scaffold's session variables say which agent launched `gr`, not which agent the
// user works in, and the live-run evidence in this repository shows them being
// deliberately unset — a detector consulting them would answer differently
// depending on who ran the command.
func DetectScaffolds(home string) []Scaffold {
	var found []Scaffold
	for _, scaffold := range SupportedScaffolds() {
		profile, err := profileFor(scaffold)
		if err != nil {
			continue
		}
		info, statErr := os.Stat(filepath.Join(home, profile.configHome))
		if statErr == nil && info.IsDir() {
			found = append(found, scaffold)
		}
	}
	return found
}

// repositoryScopePathContained verifies that writing the registration path stays
// inside the repository. It resolves the deepest existing ancestor of the path
// and refuses when that resolution leaves the resolved repository root, or when
// the settings file itself is a link.
//
// The root is resolved too, so a repository that legitimately lives behind a
// symlinked prefix (as /tmp does on macOS) is not refused for it.
func repositoryScopePathContained(repositoryRoot, settingsPath string) error {
	resolvedRoot, err := filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	if info, lstatErr := os.Lstat(settingsPath); lstatErr == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("the registration path %s is a symbolic link; refusing to write through it", settingsPath)
	}
	// Walk up to the deepest ancestor that exists and resolve that.
	ancestor := filepath.Dir(settingsPath)
	for {
		if _, statErr := os.Lstat(ancestor); statErr == nil {
			break
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			break
		}
		ancestor = parent
	}
	resolvedAncestor, err := filepath.EvalSymlinks(ancestor)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", ancestor, err)
	}
	separator := string(filepath.Separator)
	if resolvedAncestor != resolvedRoot &&
		!strings.HasPrefix(resolvedAncestor+separator, resolvedRoot+separator) {
		return fmt.Errorf("the registration path resolves outside the repository (%s); refusing to write there",
			resolvedAncestor)
	}
	return nil
}

// HasAnyRegistration reports whether one target carries any handler of ours,
// current or superseded.
//
// It deliberately differs from the presence test a registration uses: that one
// demands every current event, because a partial registration must read as absent
// so re-running can repair it. Here the question is "is there something of ours
// here at all", which is what a report about a scope this arrangement no longer
// writes has to answer.
func HasAnyRegistration(target Target) (bool, error) {
	registered, err := registeredExecutables(target.Scaffold, target.Path)
	if err != nil {
		return false, err
	}
	return len(registered) > 0, nil
}

// SupersededRegistrations reports events at one target that carry a handler of
// ours which this arrangement no longer writes.
func SupersededRegistrations(target Target) ([]string, error) {
	return supersededPresent(target.Scaffold, target.Path)
}

// ErrPathNotIgnored reports a repository-scope settings path that version control
// would carry. Registering there would install hooks in a teammate's sessions on
// the strength of one user's consent.
var ErrPathNotIgnored = errors.New("the registration path is not ignored by version control")

// IgnoreEntries are the entries a repository needs so that neither the
// registration nor the participation marker can be committed.
func IgnoreEntries() []string {
	return []string{".goalrail/", RepositorySettingsPath}
}

// IgnoreState reports whether a repository's version control ignores one path.
//
// Where the directory is not a repository at all, nothing can commit the file, so
// the question does not arise and the answer is "ignored". Where the check cannot
// be run at all — no version control tool available — the answer is "not
// ignored", and the reason is named: the two cases look identical from here, and
// treating an unverifiable path as safe is how a hook reaches a teammate who never
// agreed to it. Refusing costs the user a flag; guessing costs someone else their
// consent.
func IgnoreState(repositoryRoot, relativePath string) (bool, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return false, fmt.Errorf("%w: git is not available, so whether %s would be committed cannot be established",
			ErrPathNotIgnored, relativePath)
	}
	if !isGitRepository(repositoryRoot) {
		return true, nil
	}
	command := exec.Command("git", "-C", repositoryRoot, "check-ignore", "-q", "--", relativePath)
	if err := command.Run(); err == nil {
		return true, nil
	}
	// A tracked file is the dangerous case: it is already in the index, so an
	// ignore entry would not stop it from being committed.
	if tracked(repositoryRoot, relativePath) {
		return false, fmt.Errorf("%w: %s is already tracked", ErrPathNotIgnored, relativePath)
	}
	return false, nil
}

func isGitRepository(repositoryRoot string) bool {
	command := exec.Command("git", "-C", repositoryRoot, "rev-parse", "--git-dir")
	return command.Run() == nil
}

func tracked(repositoryRoot, relativePath string) bool {
	command := exec.Command("git", "-C", repositoryRoot, "ls-files", "--error-unmatch", "--", relativePath)
	return command.Run() == nil
}

// AddIgnoreEntries appends the missing entries to the repository's ignore file
// and reports which it added. It is only ever called for a user who asked.
func AddIgnoreEntries(repositoryRoot string, entries []string) ([]string, error) {
	ignorePath := filepath.Join(repositoryRoot, ".gitignore")
	existing, err := os.ReadFile(ignorePath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read .gitignore: %w", err)
	}
	present := make(map[string]bool)
	for _, line := range strings.Split(string(existing), "\n") {
		present[strings.TrimSpace(line)] = true
	}
	var added []string
	var addition strings.Builder
	for _, entry := range entries {
		if present[entry] {
			continue
		}
		ignored, ignoreErr := IgnoreState(repositoryRoot, entry)
		if ignoreErr != nil {
			// A tracked path, or a check that cannot run: an ignore entry would
			// change nothing, and writing one anyway would dress the refusal that
			// follows as half-fixed.
			continue
		}
		if ignored {
			// Already covered by a rule that does not name it literally.
			continue
		}
		addition.WriteString(entry)
		addition.WriteString("\n")
		added = append(added, entry)
	}
	if len(added) == 0 {
		return nil, nil
	}
	content := string(existing)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if err := os.WriteFile(ignorePath, []byte(content+addition.String()), 0o644); err != nil {
		return nil, fmt.Errorf("write .gitignore: %w", err)
	}
	return added, nil
}
