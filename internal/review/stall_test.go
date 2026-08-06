package review

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/ambient"
)

// A reviewer that speaks once and then stops is the measured field shape: two
// full passes on one branch spent 25 and then 50 minutes that way, each
// recording nothing at all after a single blocking call. The deadline could not
// tell it apart from work.
func TestAStalledReviewerIsStoppedLongBeforeItsDeadline(t *testing.T) {
	root := branchWithWork(t)
	stateRoot := t.TempDir()
	stubReviewer(t, "codex", `cat >/dev/null; echo "reading the diff"; sleep 600`)

	const deadline = 30 * time.Second
	const bound = 2 * time.Second
	started := time.Now()
	_, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main",
		Author: ambient.ScaffoldClaudeCode, Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
		Deadline: deadline, ProgressBound: bound,
	})
	elapsed := time.Since(started)

	if err == nil {
		t.Fatal("a stalled reviewer reported success")
	}
	if !errors.Is(err, ErrReviewerStalled) {
		t.Fatalf("a stall was not reported as one: %v", err)
	}
	if strings.Contains(err.Error(), "deadline") {
		t.Fatalf("a stall was reported as a deadline overrun: %v", err)
	}
	// The point of the bound is that the caller stops paying for silence.
	if elapsed >= deadline {
		t.Fatalf("the stalled reviewer still cost the whole deadline: %s", elapsed)
	}
	// What it last did travels with the stop, because the alternative is reading
	// provider session files to learn what the command already knew.
	if !strings.Contains(err.Error(), "reading the diff") {
		t.Fatalf("the stop does not name the reviewer's last activity: %v", err)
	}
	if _, found, _ := ReadReceipt(stateRoot, root, "work"); found {
		t.Fatal("a receipt was written for a review that reviewed nothing")
	}
}

// The bound must not punish a reviewer that is working. A slow one keeps
// speaking, and only the deadline may stop it.
func TestASlowButSpeakingReviewerIsNotStalled(t *testing.T) {
	root := branchWithWork(t)
	stateRoot := t.TempDir()
	stubReviewer(t, "codex", `cat >/dev/null; while :; do echo working; sleep 0.2; done`)

	_, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main",
		Author: ambient.ScaffoldClaudeCode, Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
		Deadline: 3 * time.Second, ProgressBound: 2 * time.Second,
	})

	if err == nil {
		t.Fatal("a reviewer past its deadline reported success")
	}
	if errors.Is(err, ErrReviewerStalled) {
		t.Fatalf("a working reviewer was reported as stalled: %v", err)
	}
	if !strings.Contains(err.Error(), "deadline") {
		t.Fatalf("the refusal does not name the deadline: %v", err)
	}
	if _, found, _ := ReadReceipt(stateRoot, root, "work"); found {
		t.Fatal("a receipt was written for a review that ran out of time")
	}
}

// A bound that cannot fire is a caller's mistake, not a bound. A defaulted one
// is fitted to a short deadline instead, because a caller who shortened the
// deadline never named a silence policy at all.
func TestTheProgressBoundIsResolvedAgainstTheDeadline(t *testing.T) {
	for _, fixture := range []struct {
		name      string
		requested time.Duration
		deadline  time.Duration
		want      time.Duration
		refuses   bool
	}{
		{name: "default under a long deadline", deadline: DefaultDeadline, want: DefaultProgressBound},
		{name: "default fitted to a short deadline", deadline: 4 * time.Second, want: 2 * time.Second},
		{name: "explicit and shorter", requested: time.Minute, deadline: DefaultDeadline, want: time.Minute},
		{name: "explicit at the deadline", requested: DefaultDeadline, deadline: DefaultDeadline, refuses: true},
		{name: "explicit past the deadline", requested: time.Hour, deadline: DefaultDeadline, refuses: true},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			got, err := resolveProgressBound(fixture.requested, fixture.deadline)
			if fixture.refuses {
				if err == nil {
					t.Fatalf("a bound that can never fire was accepted: %s", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != fixture.want {
				t.Fatalf("bound = %s, want %s", got, fixture.want)
			}
		})
	}
}

// The caller supplies a name; nothing else crosses into the invocation. These
// assert that what is rendered is exactly one removal, and that every other
// argument is byte-identical to a run without isolation.
func TestIsolationRendersOneRemovalAndTouchesNothingElse(t *testing.T) {
	instructions := []byte("review instructions")
	_, plain, _, err := reviewCommand(ambient.ScaffoldCodex, "base..head", "high", "", instructions, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, isolated, _, err := reviewCommand(ambient.ScaffoldCodex, "base..head", "high", "", instructions, []string{"some-integration"})
	if err != nil {
		t.Fatal(err)
	}

	const rendered = "mcp_servers.some-integration.enabled=false"
	position := -1
	for index, argument := range isolated {
		if argument == rendered {
			position = index
			break
		}
	}
	if position <= 0 || isolated[position-1] != "-c" {
		t.Fatalf("the removal was not rendered as one -c pair: %#v", isolated)
	}
	withoutPair := append(append([]string{}, isolated[:position-1]...), isolated[position+1:]...)
	if !reflect.DeepEqual(withoutPair, plain) {
		t.Fatalf("isolation changed more than it added\nwith:    %#v\nwithout: %#v", withoutPair, plain)
	}

	// The boundary that matters: nothing a caller supplies may reach these.
	for _, fixed := range []string{"sandbox_mode=read-only", "model_reasoning_effort=high"} {
		if !containsArgument(isolated, fixed) {
			t.Fatalf("isolation disturbed a fixed argument: %q missing from %#v", fixed, isolated)
		}
	}
}

func TestARejectedIntegrationNameNeverReachesAnInvocation(t *testing.T) {
	for _, name := range []string{
		"", strings.Repeat("a", 65), "-c model=other", "has space", "with\nnewline",
		"-leading-dash", ".leading-dot", "quote'injected", "semi;colon",
		// A dot is a configuration separator for at least one provider, so a
		// dotted name renders a nested key and removes nothing.
		"vendor.tool",
	} {
		if err := validateIntegrations([]string{name}); err == nil {
			t.Fatalf("the integration name %q was accepted", name)
		}
	}
	for _, name := range []string{"a", "codebase-x", "server_2", "A9"} {
		if err := validateIntegrations([]string{name}); err != nil {
			t.Fatalf("the ordinary integration name %q was refused: %v", name, err)
		}
	}
}

// Both installed providers can remove one integration by name, and the rendered
// invocation says so per provider. The refusal path remains for a provider that
// cannot, because accepting a request and keeping the integration would read as
// isolation that happened.
func TestEachProviderRendersItsOwnRemoval(t *testing.T) {
	for _, fixture := range []struct {
		reviewer ambient.Scaffold
		expect   string
	}{
		{reviewer: ambient.ScaffoldCodex, expect: "mcp_servers.some-integration.enabled=false"},
		{reviewer: ambient.ScaffoldClaudeCode, expect: `{"deniedMcpServers":[{"serverName":"some-integration"}]}`},
	} {
		t.Run(string(fixture.reviewer), func(t *testing.T) {
			if !supportsIntegrationRemoval(fixture.reviewer) {
				t.Fatalf("%s was claimed not to support per-integration removal", fixture.reviewer)
			}
			_, arguments, _, err := reviewCommand(fixture.reviewer, "base..head", "high", "", []byte("x"), []string{"some-integration"})
			if err != nil {
				t.Fatal(err)
			}
			if !containsArgument(arguments, fixture.expect) {
				t.Fatalf("%s did not render its own removal: %#v", fixture.reviewer, arguments)
			}
		})
	}
	if supportsIntegrationRemoval(ambient.Scaffold("some-future-provider")) {
		t.Fatal("an unknown provider was claimed to support removal it has never been checked for")
	}
}

// A refusal that costs nothing: an unusable name is caught before any reviewer
// is invoked, not after one has been paid for.
func TestIsolationRefusesBeforeAnythingIsSpent(t *testing.T) {
	root := branchWithWork(t)
	stateRoot := t.TempDir()
	marker := t.TempDir() + "/started"
	stubReviewer(t, "claude", `touch `+marker+`; cat >/dev/null; echo reviewed`)

	_, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main",
		Author:              ambient.ScaffoldCodex,
		Selection:           Selection{Reviewer: ambient.ScaffoldClaudeCode, Mode: "cross", Reason: "test"},
		WithoutIntegrations: []string{"vendor.tool"},
	})
	if err == nil {
		t.Fatal("an unusable integration name was accepted")
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Fatal("the reviewer ran before the isolation request was refused")
	}
}

// An isolated review is a different review, and its receipt says so.
func TestAnIsolatedReviewIsRecordedAsOne(t *testing.T) {
	root := branchWithWork(t)
	stateRoot := t.TempDir()
	stubReviewer(t, "codex", `cat >/dev/null; echo "a finding"`)

	result, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main",
		Author:              ambient.ScaffoldClaudeCode,
		Selection:           Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
		WithoutIntegrations: []string{"some-integration"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.Receipt.WithoutIntegrations, []string{"some-integration"}) {
		t.Fatalf("the receipt does not record what the reviewer ran without: %#v", result.Receipt.WithoutIntegrations)
	}

	// The same head, report, model and effort without isolation must not address
	// the same stored receipt, or one run would erase the other's evidence.
	plain, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main",
		Author: ambient.ScaffoldClaudeCode, Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
		Full: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plain.ReceiptPath == result.ReceiptPath {
		t.Fatalf("an isolated and an ordinary review shared one receipt path: %s", plain.ReceiptPath)
	}
}

func containsArgument(arguments []string, want string) bool {
	for _, argument := range arguments {
		if argument == want {
			return true
		}
	}
	return false
}
