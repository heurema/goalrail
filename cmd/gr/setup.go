package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/boundedio"
	setupflow "github.com/heurema/goalrail/internal/setup"
)

const maxSetupEvidenceBytes = 8 << 20

func runSetup(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: gr setup <plan|verify-plan|apply|no-write|rollback> [flags]")
	}
	switch args[0] {
	case "plan":
		return runSetupPlan(ctx, args[1:], stdout, stderr)
	case "verify-plan":
		return runSetupVerifyPlan(ctx, args[1:], stdout, stderr)
	case "apply":
		return runSetupApply(ctx, args[1:], stdout, stderr)
	case "no-write":
		return runSetupNoWrite(args[1:], stdout, stderr)
	case "rollback":
		return runSetupRollback(ctx, args[1:], stdout, stderr)
	case "-h", "--help":
		_, err := fmt.Fprintln(stdout, "usage: gr setup <plan|verify-plan|apply|no-write|rollback>\n  plan: --repo <path> [--home <path>] [--os <goos>] [--arch <goarch>] [--scaffold codex|claude-code] [--release-channel <https-url>] [--release-metadata <canonical-release.json>] [--setup-manifest <canonical-manifest.json>]\n  verify-plan: --repo <path> --bundle <temporary-directory> --plan <canonical-plan.json> --authorization <canonical-authorization.json> --release-metadata <canonical-release.json> [--home <path>] [--scaffold codex|claude-code] [--release-channel <https-url>]\n  apply: same verified artifacts plus --continuation-ref <task:bounded-id>; emits one canonical setup receipt, a bounded doctor summary, and the private recovery locator\n  no-write: --plan <canonical-plan.json> --continuation-ref <task:bounded-id> --reason refused|missing-authorization\n  rollback: --repo <path> --recovery <private-recovery.json> [--home <path>]")
		return err
	default:
		return fmt.Errorf("unknown gr setup command %q", args[0])
	}
}

func runSetupPlan(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("setup plan", flag.ContinueOnError)
	set.SetOutput(stderr)
	repository := set.String("repo", ".", "managed repository to inspect")
	home := set.String("home", "", "user home containing proposed durable destinations")
	operatingSystem := set.String("os", "", "target Go operating system; defaults to the current machine")
	architecture := set.String("arch", "", "target Go architecture; defaults to the current machine")
	scaffoldName := set.String("scaffold", string(ambient.ScaffoldCodex), "codex or claude-code")
	releaseChannel := set.String("release-channel", setupflow.DefaultReleaseChannel, "credential-free HTTPS release channel")
	metadataPath := set.String("release-metadata", "", "optional canonical current-release metadata")
	manifestPath := set.String("setup-manifest", "", "optional canonical platform setup manifest")
	set.Usage = func() {
		fmt.Fprintln(stderr, "usage: gr setup plan --repo <path> [--home <path>] [--os <goos>] [--arch <goarch>] [--scaffold codex|claude-code] [--release-channel <https-url>] [--release-metadata <canonical-release.json>] [--setup-manifest <canonical-manifest.json>]")
		fmt.Fprintln(stderr, "Emits one canonical read-only setup plan. A complete plan still requires exact owner authorization before apply.")
		set.PrintDefaults()
	}
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if set.NArg() != 0 {
		return fmt.Errorf("setup plan accepts flags only")
	}
	scaffold := ambient.Scaffold(*scaffoldName)
	if scaffold != ambient.ScaffoldCodex && scaffold != ambient.ScaffoldClaudeCode {
		return fmt.Errorf("unsupported scaffold %q", *scaffoldName)
	}
	evidence := &setupflow.ReleaseEvidence{}
	if strings.TrimSpace(*metadataPath) != "" {
		raw, err := boundedio.ReadRegularFile(*metadataPath, "release metadata", maxSetupEvidenceBytes)
		if err != nil {
			return err
		}
		evidence.MetadataRaw = raw
	}
	if strings.TrimSpace(*manifestPath) != "" {
		raw, err := boundedio.ReadRegularFile(*manifestPath, "setup manifest", maxSetupEvidenceBytes)
		if err != nil {
			return err
		}
		evidence.ManifestRaw = raw
	}
	result, err := setupflow.Generate(ctx, setupflow.PlanOptions{
		RepositoryRoot: *repository,
		Home:           *home,
		OS:             *operatingSystem,
		Arch:           *architecture,
		Scaffold:       scaffold,
		ReleaseChannel: *releaseChannel,
		Evidence:       evidence,
	})
	if err != nil {
		return err
	}
	_, err = stdout.Write(result.Artifact.CanonicalJSON())
	return err
}

func runSetupVerifyPlan(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	options, _, err := parseSetupVerificationOptions("setup verify-plan", args, stderr, false)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	verified, err := setupflow.VerifyPlan(ctx, options)
	if err != nil {
		return err
	}
	return writeJSON(stdout, verified.Report())
}

func runSetupApply(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	options, continuationRef, err := parseSetupVerificationOptions("setup apply", args, stderr, true)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	result, err := setupflow.ExecuteApply(ctx, options, continuationRef)
	if result.Schema != "" {
		if outputErr := writeJSON(stdout, result); outputErr != nil {
			return outputErr
		}
	}
	return err
}

func runSetupNoWrite(args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("setup no-write", flag.ContinueOnError)
	set.SetOutput(stderr)
	plan := set.String("plan", "", "canonical complete setup plan that was not authorized")
	continuationRef := set.String("continuation-ref", "", "bounded reference to the initiating task")
	reason := set.String("reason", "", "refused or missing-authorization")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if set.NArg() != 0 {
		return fmt.Errorf("setup no-write accepts flags only")
	}
	for _, required := range []struct {
		name  string
		value string
	}{
		{name: "--plan", value: *plan},
		{name: "--continuation-ref", value: *continuationRef},
		{name: "--reason", value: *reason},
	} {
		if strings.TrimSpace(required.value) == "" {
			return fmt.Errorf("setup no-write requires %s", required.name)
		}
	}
	result, err := setupflow.EmitNoWriteReceipt(setupflow.NoWriteReceiptOptions{
		PlanPath: *plan, ContinuationRef: *continuationRef, Reason: setupflow.NoWriteReason(*reason),
	})
	if err != nil {
		return err
	}
	return writeJSON(stdout, result)
}

func runSetupRollback(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("setup rollback", flag.ContinueOnError)
	set.SetOutput(stderr)
	repository := set.String("repo", ".", "repository root from the setup attempt")
	home := set.String("home", "", "user home from the setup attempt")
	recovery := set.String("recovery", "", "private exact recovery set emitted by setup apply")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if set.NArg() != 0 {
		return fmt.Errorf("setup rollback accepts flags only")
	}
	if strings.TrimSpace(*recovery) == "" {
		return fmt.Errorf("setup rollback requires --recovery")
	}
	if strings.TrimSpace(*home) == "" {
		resolved, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		*home = resolved
	}
	result, err := setupflow.Rollback(ctx, setupflow.RollbackOptions{
		Home: *home, RepositoryRoot: *repository, RecoveryPath: *recovery,
	})
	if result.Schema != "" {
		if outputErr := writeJSON(stdout, result); outputErr != nil {
			return outputErr
		}
	}
	return err
}

func parseSetupVerificationOptions(command string, args []string, stderr io.Writer, requireContinuation bool) (setupflow.VerifyPlanOptions, string, error) {
	set := flag.NewFlagSet(command, flag.ContinueOnError)
	set.SetOutput(stderr)
	repository := set.String("repo", ".", "managed repository to re-plan")
	home := set.String("home", "", "user home containing the planned durable destinations")
	bundle := set.String("bundle", "", "verified extracted temporary setup bundle")
	plan := set.String("plan", "", "canonical pre-authorized setup plan")
	authorization := set.String("authorization", "", "canonical exact-plan authorization")
	metadata := set.String("release-metadata", "", "canonical retained release metadata")
	releaseChannel := set.String("release-channel", "", "credential-free HTTPS release channel used by the plan")
	scaffoldName := set.String("scaffold", string(ambient.ScaffoldCodex), "codex or claude-code")
	continuationRef := set.String("continuation-ref", "", "bounded reference to the initiating task")
	if err := set.Parse(args); err != nil {
		return setupflow.VerifyPlanOptions{}, "", err
	}
	if set.NArg() != 0 {
		return setupflow.VerifyPlanOptions{}, "", fmt.Errorf("%s accepts flags only", command)
	}
	for _, required := range []struct {
		name  string
		value string
	}{
		{name: "--bundle", value: *bundle},
		{name: "--plan", value: *plan},
		{name: "--authorization", value: *authorization},
		{name: "--release-metadata", value: *metadata},
	} {
		if strings.TrimSpace(required.value) == "" {
			return setupflow.VerifyPlanOptions{}, "", fmt.Errorf("%s requires %s", command, required.name)
		}
	}
	if requireContinuation && strings.TrimSpace(*continuationRef) == "" {
		return setupflow.VerifyPlanOptions{}, "", fmt.Errorf("%s requires --continuation-ref", command)
	}
	scaffold := ambient.Scaffold(*scaffoldName)
	if scaffold != ambient.ScaffoldCodex && scaffold != ambient.ScaffoldClaudeCode {
		return setupflow.VerifyPlanOptions{}, "", fmt.Errorf("unsupported scaffold %q", *scaffoldName)
	}

	return setupflow.VerifyPlanOptions{
		RepositoryRoot:      *repository,
		Home:                *home,
		BundleRoot:          *bundle,
		PlanPath:            *plan,
		AuthorizationPath:   *authorization,
		ReleaseMetadataPath: *metadata,
		ReleaseChannel:      *releaseChannel,
		Scaffold:            scaffold,
	}, strings.TrimSpace(*continuationRef), nil
}
