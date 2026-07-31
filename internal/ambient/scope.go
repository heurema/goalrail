package ambient

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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

// EnsureWriteWithinRepository verifies that writing one path stays inside the
// repository. It resolves the deepest existing ancestor of the path and refuses
// when that resolution leaves the resolved repository root, or when the file
// itself is a link. Every repository-scope write goes through it — the
// registration, the overlay, the configuration — because a checkout can ship a
// symlink at any of those places and redirect the write into the user's own
// files.
//
// The root is resolved too, so a repository that legitimately lives behind a
// symlinked prefix (as /tmp does on macOS) is not refused for it.
func EnsureWriteWithinRepository(repositoryRoot, targetPath string) error {
	resolvedRoot, err := filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	if info, lstatErr := os.Lstat(targetPath); lstatErr == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("the path %s is a symbolic link; refusing to write through it", targetPath)
	}
	// Walk up to the deepest ancestor that exists and resolve that.
	ancestor := filepath.Dir(targetPath)
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
		return fmt.Errorf("the path resolves outside the repository (%s); refusing to write there",
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
	if !IsRepository(repositoryRoot) {
		return true, nil
	}
	if err := git(repositoryRoot, "check-ignore", "-q", "--", relativePath).Run(); err == nil {
		return true, nil
	}
	// A tracked file is the dangerous case: it is already in the index, so an
	// ignore entry would not stop it from being committed.
	if tracked(repositoryRoot, relativePath) {
		return false, fmt.Errorf("%w: %s is already tracked", ErrPathNotIgnored, relativePath)
	}
	return false, nil
}

// IgnoreSource names the file whose rule decided that a path is not ignored, as
// version control itself reports it, and is empty where no rule decided.
//
// It exists so a refusal can name the cause it actually has rather than the one
// that is usually true. Asserting "a rule your repository shares overrides this"
// about a repository that has no such rule is the same class of mistake the
// diagnosis forbids everywhere else: claiming more than was verified.
func IgnoreSource(repositoryRoot, relativePath string) string {
	output, err := git(repositoryRoot, "check-ignore", "-v", "--no-index", "--", relativePath).Output()
	if err != nil {
		return ""
	}
	// `<source>:<line>:<pattern>\t<path>`, and the source may itself contain a
	// colon on no platform this runs on, so the first field is the file.
	line := strings.TrimSpace(string(output))
	if line == "" {
		return ""
	}
	return strings.SplitN(line, ":", 2)[0]
}

// git runs one version control command against a directory with the caller's own
// repository selection removed from the environment.
//
// Those variables name a repository directly, and they override `-C`. A shell
// that exported them — the bare "dotfiles repository" arrangement is the common
// one — would otherwise have every question here answered about that repository
// instead of the one the user named: the ignore rule would be written into it,
// the check that reads the rule back would consult it, and the answer would be
// "ignored" about a path this repository would carry in a commit. The question
// asked here is always "what governs this directory", never "what did the
// caller's shell select".
func git(repositoryRoot string, arguments ...string) *exec.Cmd {
	command := exec.Command("git", append([]string{"-C", repositoryRoot}, arguments...)...)
	command.Env = environmentWithoutRepositorySelection()
	return command
}

func environmentWithoutRepositorySelection() []string {
	removed := map[string]bool{
		"GIT_DIR":              true,
		"GIT_WORK_TREE":        true,
		"GIT_COMMON_DIR":       true,
		"GIT_INDEX_FILE":       true,
		"GIT_OBJECT_DIRECTORY": true,
	}
	environment := os.Environ()
	kept := make([]string, 0, len(environment))
	for _, entry := range environment {
		name, _, found := strings.Cut(entry, "=")
		if found && removed[name] {
			continue
		}
		kept = append(kept, entry)
	}
	return kept
}

// IsRepository reports whether the directory is under version control at all.
// Where it is not, no commit can carry anything out of it — which is why the
// ignore question does not arise there, and why the caller that installs the
// harness has to distinguish the two rather than reading "ignored" as safety.
func IsRepository(repositoryRoot string) bool {
	return git(repositoryRoot, "rev-parse", "--git-dir").Run() == nil
}

// WorkTreeRoot reports the top of the work tree the directory belongs to, and
// false where the directory has none.
//
// Bareness is the wrong question to ask on its own: a bare repository has no
// work tree, and so does the repository's own directory, and in both cases
// repository content would land beside `HEAD`, `objects` and `refs`. This asks
// the question that covers them together, and it also reports where the clone's
// ignore rule is anchored, which is not the same directory the user named.
func WorkTreeRoot(repositoryRoot string) (string, bool) {
	if !IsRepository(repositoryRoot) {
		return "", false
	}
	output, err := git(repositoryRoot, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", false
	}
	top := strings.TrimSpace(string(output))
	if top == "" {
		return "", false
	}
	return top, true
}

func tracked(repositoryRoot, relativePath string) bool {
	return git(repositoryRoot, "ls-files", "--error-unmatch", "--", relativePath).Run() == nil
}

// IgnoreTargetState reports whether this clone can be given an ignore rule of
// its own and, where it cannot, which reason applies. The three reasons are not
// interchangeable: one is a condition the user will create later and should be
// told about, one has nowhere for repository content to live at all, and one is
// simply a machine this question cannot be asked on.
type IgnoreTargetState int

const (
	// IgnoreTargetWritable reports a clone whose own rule can be written.
	IgnoreTargetWritable IgnoreTargetState = iota
	// IgnoreTargetNotARepository reports a directory not under version control.
	IgnoreTargetNotARepository
	// IgnoreTargetNoWorkTree reports a repository with no work tree.
	IgnoreTargetNoWorkTree
	// IgnoreTargetUnknown reports that version control could not be asked.
	IgnoreTargetUnknown
)

// CloneIgnoreTarget reports where this clone's own ignore rule belongs, and what
// the entries written there are anchored to.
//
// The location is asked of version control rather than assembled from a path.
// In a linked worktree and in a submodule the repository's directory is a
// regular file, so a joined path names nothing there; and the rule that is
// actually consulted lives in the common directory rather than in the
// per-worktree one, which has no `info` at all.
func CloneIgnoreTarget(repositoryRoot string) (string, IgnoreTargetState) {
	path, _, state := cloneIgnoreTarget(repositoryRoot)
	return path, state
}

// cloneIgnoreTarget also reports the top of the work tree, because the rule is
// anchored there rather than at the directory the caller named.
func cloneIgnoreTarget(repositoryRoot string) (string, string, IgnoreTargetState) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", "", IgnoreTargetUnknown
	}
	if !IsRepository(repositoryRoot) {
		return "", "", IgnoreTargetNotARepository
	}
	workTree, hasWorkTree := WorkTreeRoot(repositoryRoot)
	if !hasWorkTree {
		return "", "", IgnoreTargetNoWorkTree
	}
	output, err := git(repositoryRoot, "rev-parse", "--git-common-dir").Output()
	if err != nil {
		return "", "", IgnoreTargetUnknown
	}
	// The answer is relative to the directory the command ran in, which is the
	// repository root because that is what `-C` named.
	common := strings.TrimSpace(string(output))
	if common == "" {
		return "", "", IgnoreTargetUnknown
	}
	if !filepath.IsAbs(common) {
		common = filepath.Join(repositoryRoot, common)
	}
	// A version that does not know the question echoes it back and exits zero, so
	// the answer is believed only where it names a directory that exists. Creating
	// one called `--git-common-dir` inside the user's work tree is not a failure
	// mode worth having.
	if info, statErr := os.Stat(common); statErr != nil || !info.IsDir() {
		return "", "", IgnoreTargetUnknown
	}
	return filepath.Join(common, "info", "exclude"), workTree, IgnoreTargetWritable
}

// AddCloneIgnoreEntries writes the missing entries to this clone's own ignore
// rule — the one no commit can carry — and reports which of them took effect.
//
// Where there is nowhere to write it reports nothing and no error, and where the
// write fails it reports the failure without acting on it: making a path
// unshareable is a service initialization performs where it can, never a
// precondition it imposes. What follows from not having done it is the caller's
// existing refusal, which is reached either way — so the failure belongs in the
// report, not in the exit status.
//
// Whether the clone's rule can cover a path is not knowable until it has been
// written, because a rule the repository shares can outrank it and deciding that
// in advance would mean reimplementing the precedence between ignore rules,
// which this package deliberately leaves to version control. So the entries are
// written, the state is read again, and any entry that turned out to achieve
// nothing is taken back out — an entry with no effect is noise the user would
// later have to diagnose.
func AddCloneIgnoreEntries(repositoryRoot string, entries []string) ([]string, error) {
	ignorePath, workTree, state := cloneIgnoreTarget(repositoryRoot)
	if state != IgnoreTargetWritable {
		return nil, nil
	}
	// The rule is anchored at the top of the work tree, and the caller may have
	// named a directory below it. An entry containing a slash is matched from the
	// anchor, so it has to carry the way back down or it names a path that does
	// not exist.
	anchored := make([]string, 0, len(entries))
	for _, entry := range entries {
		anchored = append(anchored, anchorEntry(workTree, repositoryRoot, entry))
	}
	// Version control does not always create the directory this rule lives in.
	// Some installations do, from their own template; this one does not.
	if err := os.MkdirAll(filepath.Dir(ignorePath), 0o755); err != nil {
		return nil, fmt.Errorf("make room for this clone's own ignore rule: %w", err)
	}
	// Every other write this package makes refuses a symlink, because following
	// one leaves the repository the user named — and here it would also mean the
	// take-back path removing a file the user owns.
	if info, statErr := os.Lstat(ignorePath); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("this clone's own ignore rule at %s is a symbolic link, so writing it would leave this repository", ignorePath)
	}
	added, previous, existed, err := addIgnoreEntries(repositoryRoot, ignorePath, entries, anchored)
	if err != nil {
		return nil, err
	}
	if len(added) == 0 {
		return nil, nil
	}
	var effective []string
	for index, entry := range entries {
		if !slices.Contains(added, anchored[index]) {
			continue
		}
		if ignored, ignoreErr := IgnoreState(repositoryRoot, entry); ignored && ignoreErr == nil {
			effective = append(effective, anchored[index])
		}
	}
	if len(effective) == len(added) {
		return effective, nil
	}
	if err := rewriteIgnoreFile(ignorePath, previous, existed, effective); err != nil {
		// The write stands rather than being half taken back: a visible entry the
		// report names is better than a silent partial state.
		return added, nil
	}
	return effective, nil
}

// anchorEntry rewrites an entry so it means the same path when it is matched
// from the top of the work tree rather than from the directory it describes.
// An entry with no interior slash matches at any depth already, so it is left
// alone; the anchoring only applies to one that carries a path.
func anchorEntry(workTree, repositoryRoot, entry string) string {
	if !strings.Contains(strings.TrimSuffix(entry, "/"), "/") {
		return entry
	}
	// Version control answers with symbolic links resolved and the caller's path
	// generally is not, so the two are made comparable before the distance
	// between them is measured. On a platform whose temporary and system
	// directories are links — which is every macOS — skipping this produces a
	// relative path full of parent steps and an entry that matches nothing.
	relative, err := filepath.Rel(resolved(workTree), resolved(repositoryRoot))
	if err != nil || relative == "." || strings.HasPrefix(relative, "..") {
		return entry
	}
	return filepath.ToSlash(relative) + "/" + entry
}

func resolved(path string) string {
	if evaluated, err := filepath.EvalSymlinks(path); err == nil {
		return evaluated
	}
	return path
}

// rewriteIgnoreFile puts the file back to what it was, plus only the entries
// that were worth keeping.
func rewriteIgnoreFile(ignorePath string, previous []byte, existed bool, keep []string) error {
	if len(keep) == 0 {
		if !existed {
			return os.Remove(ignorePath)
		}
		// Nothing was kept, so the file goes back byte for byte — including a last
		// line the user never terminated, which this must not silently terminate.
		return os.WriteFile(ignorePath, previous, 0o644)
	}
	content := string(previous)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	for _, entry := range keep {
		content += entry + "\n"
	}
	return os.WriteFile(ignorePath, []byte(content), 0o644)
}

// AddIgnoreEntries appends the missing entries to the ignore rules the
// repository shares with everyone who clones it, and reports which it added. It
// is only ever called for a user who asked, because those rules are repository
// content exactly as the overlay is.
func AddIgnoreEntries(repositoryRoot string, entries []string) ([]string, error) {
	if _, hasWorkTree := WorkTreeRoot(repositoryRoot); IsRepository(repositoryRoot) && !hasWorkTree {
		// Nowhere for the rule to apply, and writing it would leave repository
		// content beside `HEAD` and `objects`.
		return nil, nil
	}
	ignorePath := filepath.Join(repositoryRoot, ".gitignore")
	if info, statErr := os.Lstat(ignorePath); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("the ignore rules at %s are a symbolic link, so writing them would leave this repository", ignorePath)
	}
	added, _, _, err := addIgnoreEntries(repositoryRoot, ignorePath, entries, entries)
	return added, err
}

// addIgnoreEntries appends what is missing to one ignore file. paths are the
// paths whose state decides, and lines[i] is the text that would cover paths[i]
// when matched from wherever this particular file is anchored.
func addIgnoreEntries(repositoryRoot, ignorePath string, paths, lines []string) (
	added []string, previous []byte, existed bool, err error,
) {
	existing, err := os.ReadFile(ignorePath)
	if err != nil && !os.IsNotExist(err) {
		return nil, nil, false, fmt.Errorf("read the ignore rules: %w", err)
	}
	existed = err == nil
	present := make(map[string]bool)
	for _, line := range strings.Split(string(existing), "\n") {
		present[strings.TrimSpace(line)] = true
	}
	var addition strings.Builder
	for index, path := range paths {
		// What decides is the state, never a matching line: a rule later in the
		// same file can undo an earlier one, so a literal match proves nothing
		// about whether the path is covered.
		ignored, ignoreErr := IgnoreState(repositoryRoot, path)
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
		if present[lines[index]] {
			// The line is already there and is demonstrably not what decides, so a
			// second copy of it would change nothing but the file.
			continue
		}
		addition.WriteString(lines[index])
		addition.WriteString("\n")
		added = append(added, lines[index])
	}
	if len(added) == 0 {
		return nil, existing, existed, nil
	}
	// A file whose last line was never terminated would otherwise have that line
	// concatenated with the first entry, leaving neither path ignored.
	content := string(existing)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if err := os.WriteFile(ignorePath, []byte(content+addition.String()), 0o644); err != nil {
		return nil, nil, existed, fmt.Errorf("write the ignore rules: %w", err)
	}
	return added, existing, existed, nil
}
