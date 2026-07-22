// Command goalrail-canary is the small repository-local operator surface for
// the intent-canary-v0 lifecycle. It deliberately has no dashboard, daemon, or
// workflow-engine behavior.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/heurema/goalrail/internal/adapters/langfuse"
	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/evidence"
	"github.com/heurema/goalrail/internal/operator"
)

const (
	runContextEnvironment         = "GOALRAIL_RUN_CONTEXT"
	evidencePathEnvironment       = "GOALRAIL_EVIDENCE_PATH"
	repoRootEnvironment           = "GOALRAIL_REPO_ROOT"
	defaultEvidenceRelativePath   = "canary/intent-canary-v0/events.jsonl"
	defaultEvidenceV2RelativePath = "canary/intent-canary-v0/events-v2.jsonl"
	maxProviderInputBytes         = 64 * 1024
	langfuseBaseURLEnvironment    = "LANGFUSE_BASE_URL"
	langfuseHostEnvironment       = "LANGFUSE_HOST"
	langfusePublicKeyEnvironment  = "LANGFUSE_PUBLIC_KEY"
	langfuseSecretKeyEnvironment  = "LANGFUSE_SECRET_KEY"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "goalrail-canary:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve working directory: %w", err)
	}
	repoDefault := firstNonempty(os.Getenv(repoRootEnvironment), workingDirectory)
	global := flag.NewFlagSet("goalrail-canary", flag.ContinueOnError)
	global.SetOutput(stderr)
	repoRoot := global.String("repo", repoDefault, "repository root")
	storePath := global.String("store", os.Getenv(evidencePathEnvironment), "append-only evidence JSONL path")
	manifestVersion := global.Uint("manifest-version", uint(domain.IntentCanaryV0ManifestVersion), "immutable canary manifest version: 1 or 2")
	if err := global.Parse(args); err != nil {
		return err
	}
	remaining := global.Args()
	if len(remaining) == 0 {
		return errors.New("command is required: start, exclude, bind-context, record-basis, record-phase, freeze-checks, reconcile, inspect, deliver, abandon, assess, correct, report, or disable")
	}

	absRepo, err := filepath.Abs(filepath.Clean(*repoRoot))
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	resolvedStore := strings.TrimSpace(*storePath)
	if resolvedStore == "" {
		defaultPath, pathErr := defaultEvidencePath(uint32(*manifestVersion))
		if pathErr != nil {
			return pathErr
		}
		resolvedStore = filepath.Join(absRepo, filepath.FromSlash(defaultPath))
	} else if !filepath.IsAbs(resolvedStore) {
		resolvedStore = filepath.Join(absRepo, resolvedStore)
	}
	store, err := evidence.NewStoreForManifest(resolvedStore, uint32(*manifestVersion))
	if err != nil {
		return err
	}
	service, err := operator.NewServiceForManifest(store, absRepo, uint32(*manifestVersion))
	if err != nil {
		return err
	}

	command, commandArgs := remaining[0], remaining[1:]
	switch command {
	case "start":
		return runStart(service, resolvedStore, absRepo, commandArgs, stdin, stdout, stderr)
	case "exclude":
		return runExclude(service, commandArgs, stdout, stderr)
	case "bind-context":
		return runBindContext(service, commandArgs, stderr)
	case "record-basis":
		return runRecordBasis(service, commandArgs, stderr)
	case "record-phase":
		return runRecordPhase(service, commandArgs, stderr)
	case "freeze-checks":
		return runFreezeChecks(service, commandArgs, stdout, stderr)
	case "reconcile":
		return runReconcile(service, commandArgs, stdout, stderr)
	case "inspect":
		return runInspect(service, commandArgs, stdout, stderr)
	case "deliver":
		return runDeliver(service, commandArgs, stderr)
	case "abandon":
		return runAbandon(service, commandArgs, stderr)
	case "assess":
		return runAssess(service, commandArgs, stderr)
	case "correct":
		return runCorrect(service, commandArgs, stderr)
	case "report":
		return runReport(service, commandArgs, stdout)
	case "disable":
		return runDisable(service, commandArgs, stdout, stderr)
	case "bind-hook":
		return runBindHook(service, commandArgs, stdin, stderr)
	case "bind-launch":
		return runBindLaunch(service, commandArgs, stdin, stderr)
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func runBindContext(service *operator.Service, args []string, stderr io.Writer) error {
	set := flag.NewFlagSet("bind-context", flag.ContinueOnError)
	set.SetOutput(stderr)
	changeID := set.String("change", "", "stable change ID")
	actor := set.String("actor", "", "operator actor ID")
	source := set.String("source", "", "bounded Context Pack source reference")
	contextPackID := set.String("context-pack", "", "stable Context Pack ID")
	contextVersion := set.Uint("context-version", 0, "Context Pack version")
	if err := set.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(set); err != nil {
		return err
	}
	eventID, err := generatedEventID()
	if err != nil {
		return err
	}
	return service.RecordContextBinding(operator.ContextBindingInput{
		EventID: eventID, ChangeID: domain.ChangeID(*changeID), OccurredAt: time.Now().UTC(),
		Actor: domain.ActorID(*actor), SourceRef: domain.EvidenceReference(*source),
		ContextPackID: domain.ContextPackID(*contextPackID), ContextPackVersion: uint32(*contextVersion),
	})
}

func runRecordBasis(service *operator.Service, args []string, stderr io.Writer) error {
	set := flag.NewFlagSet("record-basis", flag.ContinueOnError)
	set.SetOutput(stderr)
	changeID := set.String("change", "", "stable change ID")
	actor := set.String("actor", "", "owner or operator actor ID")
	source := set.String("source", "", "bounded assessment-basis source reference")
	intentRef := set.String("intent-ref", "", "bounded confirmed-intent reference")
	intentID := set.String("intent-id", "", "stable intent ID")
	intentVersion := set.Uint("intent-version", 0, "confirmed intent version")
	timing := set.String("timing", "", "pre_execution for flow or post_delivery for baseline")
	var desiredOutcomes, nonGoals, successSignals stringList
	set.Var(&desiredOutcomes, "desired-outcome-id", "stable desired-outcome ID; repeat for every item")
	set.Var(&nonGoals, "non-goal-id", "stable non-goal ID; repeat for every item")
	set.Var(&successSignals, "success-signal-id", "stable success-signal ID; repeat for every item")
	if err := set.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(set); err != nil {
		return err
	}
	eventID, err := generatedEventID()
	if err != nil {
		return err
	}
	return service.RecordAssessmentBasis(operator.AssessmentBasisInput{
		EventID: eventID, ChangeID: domain.ChangeID(*changeID), OccurredAt: time.Now().UTC(),
		Actor: domain.ActorID(*actor), SourceRef: domain.EvidenceReference(*source),
		Basis: domain.CanaryAssessmentBasis{
			IntentRef: domain.EvidenceReference(*intentRef), IntentID: domain.IntentID(*intentID),
			IntentVersion: uint32(*intentVersion), Timing: domain.AssessmentBasisTiming(*timing),
			DesiredOutcomeIDs: desiredOutcomes.intentItemIDs(), NonGoalIDs: nonGoals.intentItemIDs(),
			SuccessSignalIDs: successSignals.intentItemIDs(),
		},
	})
}

func runRecordPhase(service *operator.Service, args []string, stderr io.Writer) error {
	set := flag.NewFlagSet("record-phase", flag.ContinueOnError)
	set.SetOutput(stderr)
	changeID := set.String("change", "", "stable change ID")
	actor := set.String("actor", "", "operator actor ID")
	source := set.String("source", "", "bounded flow-phase source reference")
	started := set.String("started-at", "", "flow phase start, RFC3339")
	completed := set.String("completed-at", "", "flow phase completion, RFC3339")
	if err := set.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(set); err != nil {
		return err
	}
	startedAt, err := time.Parse(time.RFC3339Nano, *started)
	if err != nil {
		return fmt.Errorf("parse started-at: %w", err)
	}
	completedAt, err := time.Parse(time.RFC3339Nano, *completed)
	if err != nil {
		return fmt.Errorf("parse completed-at: %w", err)
	}
	eventID, err := generatedEventID()
	if err != nil {
		return err
	}
	return service.RecordFlowPhase(operator.FlowPhaseInput{
		EventID: eventID, ChangeID: domain.ChangeID(*changeID), OccurredAt: time.Now().UTC(),
		Actor: domain.ActorID(*actor), SourceRef: domain.EvidenceReference(*source),
		StartedAt: startedAt.UTC(), CompletedAt: completedAt.UTC(),
	})
}

func runReconcile(service *operator.Service, args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("reconcile", flag.ContinueOnError)
	set.SetOutput(stderr)
	changeID := set.String("change", "", "stable change ID")
	actor := set.String("actor", "", "operator actor ID")
	sourceRef := set.String("source", "", "bounded reconciliation source reference")
	ownerReviewRef := set.String("owner-review-ref", "", "flow-only owner-review evidence reference")
	ownerReviewStart := set.String("owner-review-start", "", "flow-only owner-review start, RFC3339")
	ownerReviewEnd := set.String("owner-review-end", "", "flow-only owner-review end, RFC3339")
	correctionReason := set.String("correction-reason", "", "reason for explicitly superseding prior telemetry")
	if err := set.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(set); err != nil {
		return err
	}
	if err := requireCanonical("change", *changeID); err != nil {
		return err
	}

	var ownerReview *domain.CanaryTimingInterval
	ownerFields := 0
	for _, value := range []string{*ownerReviewRef, *ownerReviewStart, *ownerReviewEnd} {
		if strings.TrimSpace(value) != "" {
			ownerFields++
		}
	}
	if ownerFields != 0 && ownerFields != 3 {
		return errors.New("owner-review-ref, owner-review-start, and owner-review-end must be supplied together")
	}
	if ownerFields == 3 {
		startedAt, err := time.Parse(time.RFC3339Nano, *ownerReviewStart)
		if err != nil {
			return fmt.Errorf("parse owner-review-start: %w", err)
		}
		endedAt, err := time.Parse(time.RFC3339Nano, *ownerReviewEnd)
		if err != nil {
			return fmt.Errorf("parse owner-review-end: %w", err)
		}
		ownerReview = &domain.CanaryTimingInterval{
			Reference: domain.EvidenceReference(*ownerReviewRef),
			StartedAt: startedAt.UTC(), EndedAt: endedAt.UTC(),
		}
	}

	baseURL := firstNonempty(os.Getenv(langfuseBaseURLEnvironment), os.Getenv(langfuseHostEnvironment))
	publicKey, secretKey := os.Getenv(langfusePublicKeyEnvironment), os.Getenv(langfuseSecretKeyEnvironment)
	if baseURL == "" || publicKey == "" || secretKey == "" {
		return errors.New("reconcile requires LANGFUSE_BASE_URL or LANGFUSE_HOST plus LANGFUSE_PUBLIC_KEY and LANGFUSE_SECRET_KEY")
	}
	client, err := langfuse.NewClient(langfuse.ClientConfig{
		BaseURL: baseURL, PublicKey: publicKey, SecretKey: secretKey,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	})
	if err != nil {
		return err
	}
	eventID, err := generatedEventID()
	if err != nil {
		return err
	}
	receipt, err := service.ReconcileTelemetry(context.Background(), client, operator.ReconcileTelemetryInput{
		EventID: eventID, ChangeID: domain.ChangeID(*changeID), OccurredAt: time.Now().UTC(),
		Actor: domain.ActorID(*actor), SourceRef: domain.EvidenceReference(*sourceRef),
		OwnerReview: ownerReview, CorrectionReason: domain.EvidenceReasonCode(*correctionReason),
	})
	if err != nil {
		return err
	}
	return writeJSON(stdout, receipt)
}

func defaultEvidencePath(manifestVersion uint32) (string, error) {
	switch manifestVersion {
	case domain.IntentCanaryV0ManifestVersion:
		return defaultEvidenceRelativePath, nil
	case domain.IntentCanaryV0ManifestVersion2:
		return defaultEvidenceV2RelativePath, nil
	default:
		return "", fmt.Errorf("unsupported manifest version %d", manifestVersion)
	}
}

func runDisable(service *operator.Service, args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("disable", flag.ContinueOnError)
	set.SetOutput(stderr)
	actor := set.String("actor", "", "operator actor ID")
	source := set.String("source", "", "bounded source reference")
	reason := set.String("reason", "", "bounded stop reason code")
	if err := set.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(set); err != nil {
		return err
	}
	eventID, err := newID("event")
	if err != nil {
		return err
	}
	receipt, err := service.Disable(operator.DisableInput{
		EventID:    domain.EvidenceEventID(eventID),
		OccurredAt: time.Now().UTC(),
		Actor:      domain.ActorID(*actor),
		SourceRef:  domain.EvidenceReference(*source),
		Reason:     domain.EvidenceReasonCode(*reason),
	})
	if err != nil {
		return err
	}
	return writeJSON(stdout, receipt)
}

func runReport(service *operator.Service, args []string, stdout io.Writer) error {
	if len(args) != 0 {
		return errors.New("report accepts no command arguments")
	}
	report, err := service.Report()
	if err != nil {
		return err
	}
	return writeJSON(stdout, report)
}

func runStart(
	service *operator.Service,
	storePath string,
	repoRoot string,
	args []string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
) error {
	flagArgs, launchArgs := splitLaunchArgs(args)
	set := flag.NewFlagSet("start", flag.ContinueOnError)
	set.SetOutput(stderr)
	changeID := set.String("change", "", "stable change ID")
	intentVersion := set.Uint("intent-version", 0, "confirmed intent version")
	actor := set.String("actor", "", "operator actor ID")
	source := set.String("source", "", "bounded source reference")
	reason := set.String("reason", "", "categorical eligible-admission reason")
	synthetic := set.Bool("synthetic", false, "allow only a synthetic pre-activation assignment")
	if err := set.Parse(flagArgs); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return fmt.Errorf("unexpected start arguments: %s", strings.Join(set.Args(), " "))
	}
	if err := requireCanonical("change", *changeID); err != nil {
		return err
	}
	eventID, err := newID("event")
	if err != nil {
		return err
	}
	runID, err := newID("run")
	if err != nil {
		return err
	}
	receipt, err := service.Start(operator.StartInput{
		EventID:       domain.EvidenceEventID(eventID),
		ChangeID:      domain.ChangeID(*changeID),
		RunID:         domain.RunID(runID),
		IntentVersion: uint32(*intentVersion),
		OccurredAt:    time.Now().UTC(),
		Actor:         domain.ActorID(*actor),
		SourceRef:     domain.EvidenceReference(*source),
		Reason:        domain.EvidenceReasonCode(*reason),
		Synthetic:     *synthetic,
	})
	if err != nil {
		return err
	}
	if len(launchArgs) != 0 {
		fmt.Fprintf(stderr, "started %s as ordinal %d (%s)\n", receipt.ChangeID, receipt.Ordinal, receipt.Variant)
		child := exec.Command(launchArgs[0], launchArgs[1:]...)
		child.Dir = repoRoot
		child.Env = append(os.Environ(),
			runContextEnvironment+"="+receipt.RunContextEnv,
			evidencePathEnvironment+"="+storePath,
			repoRootEnvironment+"="+repoRoot,
		)
		child.Stdin = stdin
		child.Stdout = stdout
		child.Stderr = stderr
		if err := child.Run(); err != nil {
			return fmt.Errorf("launched command failed after assignment: %w", err)
		}
		return nil
	}
	return writeJSON(stdout, receipt)
}

func runExclude(service *operator.Service, args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("exclude", flag.ContinueOnError)
	set.SetOutput(stderr)
	changeID := set.String("change", "", "stable change ID")
	actor := set.String("actor", "", "operator actor ID")
	source := set.String("source", "", "bounded source reference")
	reason := set.String("reason", "", "categorical exclusion reason")
	synthetic := set.Bool("synthetic", false, "allow only synthetic pre-activation admission")
	if err := set.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(set); err != nil {
		return err
	}
	if err := requireCanonical("change", *changeID); err != nil {
		return err
	}
	eventID, err := generatedEventID()
	if err != nil {
		return err
	}
	receipt, err := service.Exclude(operator.ExcludeInput{
		EventID:    eventID,
		ChangeID:   domain.ChangeID(*changeID),
		OccurredAt: time.Now().UTC(),
		Actor:      domain.ActorID(*actor),
		SourceRef:  domain.EvidenceReference(*source),
		Reason:     domain.EvidenceReasonCode(*reason),
		Synthetic:  *synthetic,
	})
	if err != nil {
		return err
	}
	return writeJSON(stdout, receipt)
}

func runFreezeChecks(service *operator.Service, args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("freeze-checks", flag.ContinueOnError)
	set.SetOutput(stderr)
	changeID := set.String("change", "", "stable change ID")
	actor := set.String("actor", "", "operator actor ID")
	source := set.String("source", "", "bounded check-selection source reference")
	var checks stringList
	set.Var(&checks, "check-ref", "bounded check reference; repeat for every selected check")
	if err := set.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(set); err != nil {
		return err
	}
	eventID, err := generatedEventID()
	if err != nil {
		return err
	}
	receipt, err := service.FreezeChecks(operator.FreezeChecksInput{
		EventID:    eventID,
		ChangeID:   domain.ChangeID(*changeID),
		OccurredAt: time.Now().UTC(),
		Actor:      domain.ActorID(*actor),
		SourceRef:  domain.EvidenceReference(*source),
		CheckRefs:  checks.references(),
	})
	if err != nil {
		return err
	}
	return writeJSON(stdout, receipt)
}

func runInspect(service *operator.Service, args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("inspect", flag.ContinueOnError)
	set.SetOutput(stderr)
	changeID := set.String("change", "", "stable change ID")
	if err := set.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(set); err != nil {
		return err
	}
	if err := requireCanonical("change", *changeID); err != nil {
		return err
	}
	view, err := service.Inspect(domain.ChangeID(*changeID))
	if err != nil {
		return err
	}
	if view.Admission == nil && view.Assignment == nil {
		return operator.ErrChangeNotStarted
	}
	return writeJSON(stdout, view)
}

func runDeliver(service *operator.Service, args []string, stderr io.Writer) error {
	set := flag.NewFlagSet("deliver", flag.ContinueOnError)
	set.SetOutput(stderr)
	changeID := set.String("change", "", "stable change ID")
	actor := set.String("actor", "", "delivery actor ID")
	source := set.String("source", "", "bounded delivery source reference")
	green := set.Bool("green", false, "all selected checks passed")
	var checks stringList
	var overhead optionalFloat
	set.Var(&checks, "check-ref", "bounded check reference; repeat for every selected check")
	set.Var(&overhead, "overhead-minutes", "measured flow-only overhead")
	if err := set.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(set); err != nil {
		return err
	}
	eventID, err := generatedEventID()
	if err != nil {
		return err
	}
	return service.Deliver(operator.DeliverInput{
		EventID:             eventID,
		ChangeID:            domain.ChangeID(*changeID),
		OccurredAt:          time.Now().UTC(),
		Actor:               domain.ActorID(*actor),
		SourceRef:           domain.EvidenceReference(*source),
		CheckRefs:           checks.references(),
		ChecksGreen:         *green,
		FlowOverheadMinutes: overhead.pointer(),
	})
}

func runAbandon(service *operator.Service, args []string, stderr io.Writer) error {
	set := flag.NewFlagSet("abandon", flag.ContinueOnError)
	set.SetOutput(stderr)
	changeID := set.String("change", "", "stable change ID")
	actor := set.String("actor", "", "abandonment actor ID")
	source := set.String("source", "", "bounded abandonment source reference")
	reason := set.String("reason", "", "categorical abandonment reason")
	processCaused := set.Bool("process-caused", false, "the flow caused this abandonment")
	var overhead optionalFloat
	set.Var(&overhead, "overhead-minutes", "measured flow-only overhead")
	if err := set.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(set); err != nil {
		return err
	}
	eventID, err := generatedEventID()
	if err != nil {
		return err
	}
	return service.Abandon(operator.AbandonInput{
		EventID:             eventID,
		ChangeID:            domain.ChangeID(*changeID),
		OccurredAt:          time.Now().UTC(),
		Actor:               domain.ActorID(*actor),
		SourceRef:           domain.EvidenceReference(*source),
		Reason:              domain.EvidenceReasonCode(*reason),
		ProcessCaused:       *processCaused,
		FlowOverheadMinutes: overhead.pointer(),
	})
}

func runAssess(service *operator.Service, args []string, stderr io.Writer) error {
	set := flag.NewFlagSet("assess", flag.ContinueOnError)
	set.SetOutput(stderr)
	changeID := set.String("change", "", "stable change ID")
	owner := set.String("owner", "", "owner actor ID")
	source := set.String("source", "", "bounded assessment source reference")
	outcome := set.String("outcome", "", "owner outcome: match, partial, or miss")
	var repeat optionalBool
	var desiredOutcomes, nonGoals, successSignals stringList
	set.Var(&repeat, "repeat-opt-in", "flow owner choice: yes or no")
	set.Var(&desiredOutcomes, "desired-outcome", "item judgment as ITEM_ID=achieved|partial|missed; repeat for every outcome")
	set.Var(&nonGoals, "non-goal", "item judgment as ITEM_ID=preserved|violated; repeat for every non-goal")
	set.Var(&successSignals, "success-signal", "item judgment as ITEM_ID=observed|missing; repeat for every signal")
	if err := set.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(set); err != nil {
		return err
	}
	eventID, err := generatedEventID()
	if err != nil {
		return err
	}
	judgments, err := parseItemJudgments(desiredOutcomes, nonGoals, successSignals)
	if err != nil {
		return err
	}
	return service.Assess(operator.AssessInput{
		EventID:       eventID,
		ChangeID:      domain.ChangeID(*changeID),
		OccurredAt:    time.Now().UTC(),
		Owner:         domain.ActorID(*owner),
		SourceRef:     domain.EvidenceReference(*source),
		Outcome:       domain.IntentOutcome(*outcome),
		RepeatOptIn:   repeat.pointer(),
		ItemJudgments: judgments,
	})
}

func runCorrect(service *operator.Service, args []string, stderr io.Writer) error {
	set := flag.NewFlagSet("correct", flag.ContinueOnError)
	set.SetOutput(stderr)
	kind := set.String("kind", "", "correction kind: material, checks, or assessment")
	changeID := set.String("change", "", "stable change ID")
	owner := set.String("owner", "", "owner actor ID")
	actor := set.String("actor", "", "operator actor ID for check correction")
	source := set.String("source", "", "bounded correction source reference")
	reason := set.String("reason", "", "categorical correction reason")
	outcome := set.String("outcome", "", "corrected outcome for assessment correction")
	var repeat optionalBool
	var checks stringList
	var desiredOutcomes, nonGoals, successSignals stringList
	set.Var(&repeat, "repeat-opt-in", "corrected flow owner choice: yes or no")
	set.Var(&checks, "check-ref", "corrected bounded check reference; repeat for the effective set")
	set.Var(&desiredOutcomes, "desired-outcome", "corrected ITEM_ID=achieved|partial|missed judgment")
	set.Var(&nonGoals, "non-goal", "corrected ITEM_ID=preserved|violated judgment")
	set.Var(&successSignals, "success-signal", "corrected ITEM_ID=observed|missing judgment")
	if err := set.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(set); err != nil {
		return err
	}
	eventID, err := generatedEventID()
	if err != nil {
		return err
	}
	commonTime := time.Now().UTC()
	judgments, err := parseItemJudgments(desiredOutcomes, nonGoals, successSignals)
	if err != nil {
		return err
	}
	switch *kind {
	case "material":
		if *actor != "" || *outcome != "" || repeat.set || len(checks) != 0 || len(judgments) != 0 {
			return errors.New("material correction does not accept assessment fields")
		}
		return service.RecordMaterialCorrection(operator.MaterialCorrectionInput{
			EventID:    eventID,
			ChangeID:   domain.ChangeID(*changeID),
			OccurredAt: commonTime,
			Owner:      domain.ActorID(*owner),
			SourceRef:  domain.EvidenceReference(*source),
			Reason:     domain.EvidenceReasonCode(*reason),
		})
	case "assessment":
		if *actor != "" || len(checks) != 0 {
			return errors.New("assessment correction does not accept check fields")
		}
		return service.CorrectAssessment(operator.CorrectAssessmentInput{
			EventID:       eventID,
			ChangeID:      domain.ChangeID(*changeID),
			OccurredAt:    commonTime,
			Owner:         domain.ActorID(*owner),
			SourceRef:     domain.EvidenceReference(*source),
			Reason:        domain.EvidenceReasonCode(*reason),
			Outcome:       domain.IntentOutcome(*outcome),
			RepeatOptIn:   repeat.pointer(),
			ItemJudgments: judgments,
		})
	case "checks":
		if *owner != "" || *outcome != "" || repeat.set || len(judgments) != 0 {
			return errors.New("check correction does not accept owner-assessment fields")
		}
		_, err := service.CorrectChecks(operator.CorrectChecksInput{
			EventID:    eventID,
			ChangeID:   domain.ChangeID(*changeID),
			OccurredAt: commonTime,
			Actor:      domain.ActorID(*actor),
			SourceRef:  domain.EvidenceReference(*source),
			Reason:     domain.EvidenceReasonCode(*reason),
			CheckRefs:  checks.references(),
		})
		return err
	default:
		return errors.New("correct --kind must be material, checks, or assessment")
	}
}

func parseItemJudgments(
	desiredOutcomes stringList,
	nonGoals stringList,
	successSignals stringList,
) ([]domain.IntentItemJudgment, error) {
	groups := []struct {
		category domain.IntentItemCategory
		values   stringList
	}{
		{category: domain.IntentCategoryDesiredOutcome, values: desiredOutcomes},
		{category: domain.IntentCategoryNonGoal, values: nonGoals},
		{category: domain.IntentCategorySuccessSignal, values: successSignals},
	}
	judgments := make([]domain.IntentItemJudgment, 0, len(desiredOutcomes)+len(nonGoals)+len(successSignals))
	for _, group := range groups {
		for _, value := range group.values {
			itemID, judgment, found := strings.Cut(value, "=")
			if !found || strings.TrimSpace(itemID) == "" || strings.TrimSpace(judgment) == "" {
				return nil, fmt.Errorf("%s judgment must use ITEM_ID=value", group.category)
			}
			judgments = append(judgments, domain.IntentItemJudgment{
				ItemID: domain.IntentItemID(itemID), Category: group.category,
				Judgment: domain.IntentItemJudgmentValue(judgment),
			})
		}
	}
	return judgments, nil
}

func runBindHook(service *operator.Service, args []string, stdin io.Reader, stderr io.Writer) error {
	if len(args) != 0 {
		return errors.New("bind-hook accepts provider hook JSON only on stdin")
	}
	raw, err := readBounded(stdin)
	if err != nil {
		return err
	}
	eventID, err := generatedEventID()
	if err != nil {
		return err
	}
	_, err = service.BindLifecycleHook(operator.HookInput{
		EventID:        eventID,
		OccurredAt:     time.Now().UTC(),
		Actor:          "goalrail-hook",
		SourceRef:      "codex-hook:lifecycle",
		EncodedContext: os.Getenv(runContextEnvironment),
		RawHook:        raw,
	})
	return err
}

func runBindLaunch(service *operator.Service, args []string, stdin io.Reader, stderr io.Writer) error {
	if len(args) != 0 {
		return errors.New("bind-launch accepts provider create_thread receipt JSON only on stdin")
	}
	raw, err := readBounded(stdin)
	if err != nil {
		return err
	}
	eventID, err := generatedEventID()
	if err != nil {
		return err
	}
	_, err = service.BindLaunchReceipt(operator.LaunchReceiptInput{
		EventID:        eventID,
		OccurredAt:     time.Now().UTC(),
		Actor:          "goalrail-launcher",
		SourceRef:      "codex-app:create-thread",
		EncodedContext: os.Getenv(runContextEnvironment),
		RawReceipt:     raw,
	})
	return err
}

func splitLaunchArgs(args []string) ([]string, []string) {
	for index, arg := range args {
		if arg == "--" {
			return args[:index], args[index+1:]
		}
	}
	return args, nil
}

func readBounded(reader io.Reader) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, maxProviderInputBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read provider input: %w", err)
	}
	if len(raw) > maxProviderInputBytes {
		return nil, errors.New("provider input exceeds 64 KiB")
	}
	return raw, nil
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func requireNoArgs(set *flag.FlagSet) error {
	if set.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(set.Args(), " "))
	}
	return nil
}

func requireCanonical(field, value string) error {
	if !domain.IsCanonicalID(value) {
		return fmt.Errorf("%s must be a canonical non-empty ID", field)
	}
	return nil
}

func newID(prefix string) (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate %s ID: %w", prefix, err)
	}
	return prefix + "-" + hex.EncodeToString(bytes[:]), nil
}

func generatedEventID() (domain.EvidenceEventID, error) {
	value, err := newID("event")
	return domain.EvidenceEventID(value), err
}

func firstNonempty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

type stringList []string

func (list *stringList) String() string { return strings.Join(*list, ",") }

func (list *stringList) Set(value string) error {
	*list = append(*list, value)
	return nil
}

func (list stringList) references() []domain.EvidenceReference {
	references := make([]domain.EvidenceReference, len(list))
	for index, value := range list {
		references[index] = domain.EvidenceReference(value)
	}
	return references
}

func (list stringList) intentItemIDs() []domain.IntentItemID {
	ids := make([]domain.IntentItemID, len(list))
	for index, value := range list {
		ids[index] = domain.IntentItemID(value)
	}
	return ids
}

type optionalFloat struct {
	set   bool
	value float64
}

func (value *optionalFloat) String() string {
	if !value.set {
		return ""
	}
	return strconv.FormatFloat(value.value, 'g', -1, 64)
}

func (value *optionalFloat) Set(raw string) error {
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return err
	}
	value.set = true
	value.value = parsed
	return nil
}

func (value optionalFloat) pointer() *float64 {
	if !value.set {
		return nil
	}
	copy := value.value
	return &copy
}

type optionalBool struct {
	set   bool
	value bool
}

func (value *optionalBool) String() string {
	if !value.set {
		return ""
	}
	if value.value {
		return "yes"
	}
	return "no"
}

func (value *optionalBool) Set(raw string) error {
	switch raw {
	case "yes":
		value.set = true
		value.value = true
	case "no":
		value.set = true
		value.value = false
	default:
		return errors.New("must be yes or no")
	}
	return nil
}

func (value optionalBool) pointer() *bool {
	if !value.set {
		return nil
	}
	copy := value.value
	return &copy
}
