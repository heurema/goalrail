package harness

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestPinnedCLIAcceptsTheEmbeddedCanon is the development-side guard for the
// no-Node rule.
//
// `gr` never executes the stock OpenSpec CLI, so nothing in a user's repository
// checks that the overlay it materializes is one that CLI accepts. That check has
// to happen where the canon is authored, which is here: the embedded canon is
// materialized into a temporary repository and validated with the pinned,
// telemetry-disabled invocation. A canon the CLI rejects then fails our build
// rather than a user's first change.
//
// It skips loudly where the runtime or the network is unavailable, so a
// contributor without Node can still build — and the skip says what was not
// checked, because a silent skip would read as a pass.
func TestPinnedCLIAcceptsTheEmbeddedCanon(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping the pinned-CLI check: -short requested, so the canon is unverified here")
	}
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("skipping the pinned-CLI check: no npx on PATH, so the canon is unverified here; " +
			"run it on a machine with Node 20.19+ before releasing a canon")
	}
	// The pinned package must already be in the npm cache: an ordinary `go test`
	// run must never reach for the network, and an offline machine must skip
	// loudly rather than fail. `--offline` makes npm error instead of fetching.
	probe := exec.Command("npx", "--yes", "--offline", pinnedPackage, "--version")
	probe.Env = append(os.Environ(), "OPENSPEC_TELEMETRY=0")
	if output, err := probe.CombinedOutput(); err != nil {
		t.Skipf("skipping the pinned-CLI check: %s is not in the npm cache, and this test never "+
			"fetches the network; prime it with `npx --yes %s --version` before releasing a canon\n%s",
			pinnedPackage, pinnedPackage, output)
	}

	root := t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if _, err := EnsureConfig(root, false); err != nil {
		t.Fatalf("ensure config: %v", err)
	}

	// What this proves, exactly: the pinned CLI accepts the schema's structure
	// (`schema validate`), creates a change against it, and resolves every
	// template. It does NOT run `openspec validate` on the created change, and
	// running it would prove little: the validator was measured accepting a
	// gutted design.md outright — it checks spec deltas, not sections. An
	// earlier version of this comment promised "strict validation" that was
	// never executed; a test's description must not claim more than it runs.
	run := func(arguments ...string) (string, error) {
		command := exec.Command("npx", append([]string{"--yes", "--offline", pinnedPackage}, arguments...)...)
		command.Dir = root
		command.Env = append(os.Environ(), "OPENSPEC_TELEMETRY=0")
		output, err := command.CombinedOutput()
		return string(output), err
	}

	if output, err := run("schema", "validate", SchemaName); err != nil {
		t.Fatalf("the pinned CLI rejects the embedded schema: %v\n%s", err, output)
	}
	if output, err := run("new", "change", "canon-probe", "--schema", SchemaName); err != nil {
		t.Fatalf("the pinned CLI cannot create a change against the embedded schema: %v\n%s", err, output)
	}
	// Every template the canon carries must be reachable through the schema.
	if output, err := run("templates", "--schema", SchemaName); err != nil {
		t.Fatalf("the pinned CLI cannot resolve the embedded templates: %v\n%s", err, output)
	} else {
		for _, template := range []string{"context", "intent", "proposal", "spec", "design", "tasks"} {
			if !strings.Contains(output, template) {
				t.Errorf("the pinned CLI does not resolve the %s template:\n%s", template, output)
			}
		}
	}
	if _, err := os.Stat(filepath.Join(root, "openspec", "changes", "canon-probe")); err != nil {
		t.Errorf("the change directory was not created: %v", err)
	}
}
