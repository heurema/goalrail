package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/domain"
)

func TestSetupPlanEmitsCanonicalReadOnlyPlan(t *testing.T) {
	repository := lineageCommandRepository(t)
	var stdout, stderr bytes.Buffer
	if err := runSetup(context.Background(), []string{
		"plan", "--repo", repository, "--home", t.TempDir(),
	}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	plan, err := domain.DecodeSetupPlan(bytes.NewReader(stdout.Bytes()))
	if err != nil {
		t.Fatalf("decode setup plan: %v\n%s", err, stdout.String())
	}
	if plan.Schema != domain.SetupPlanSchemaV1 || plan.State != domain.SetupPlanIncomplete {
		t.Fatalf("setup plan = %+v", plan)
	}
	frozen, err := domain.FreezeSetupPlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stdout.Bytes(), frozen.CanonicalJSON()) {
		t.Fatalf("setup plan output is not exact canonical JSON")
	}
}

func TestSetupVerificationAndApplyRequireExplicitArtifacts(t *testing.T) {
	for _, command := range []string{"verify-plan", "apply"} {
		t.Run(command, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := run(
				context.Background(),
				[]string{"setup", command},
				strings.NewReader(""),
				&stdout,
				&stderr,
				productionService,
			)
			if err == nil || !strings.Contains(err.Error(), "requires --bundle") {
				t.Fatalf("setup %s error = %v", command, err)
			}
			if stdout.Len() != 0 {
				t.Fatalf("failed setup %s wrote stdout: %q", command, stdout.String())
			}
		})
	}
}

func TestSetupRollbackRequiresTheExactRecoveryArtifact(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run(
		context.Background(),
		[]string{"setup", "rollback"},
		strings.NewReader(""),
		&stdout,
		&stderr,
		productionService,
	)
	if err == nil || !strings.Contains(err.Error(), "requires --recovery") {
		t.Fatalf("setup rollback error = %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("failed setup rollback wrote stdout: %q", stdout.String())
	}
}

func TestSetupHelpNamesVerificationReceiptsAndRollback(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run(
		context.Background(),
		[]string{"help", "setup"},
		strings.NewReader(""),
		&stdout,
		&stderr,
		productionService,
	)
	if err != nil {
		t.Fatal(err)
	}
	text := stdout.String()
	for _, required := range []string{
		"setup <plan|verify-plan|apply|no-write|rollback>", "plan:", "--setup-manifest", "--bundle", "--plan", "--authorization", "--release-metadata",
		"--continuation-ref", "canonical setup receipt", "bounded doctor summary", "no-write:", "refused|missing-authorization",
		"rollback:", "--recovery",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("setup help does not mention %q:\n%s", required, text)
		}
	}
	for _, forbidden := range []string{"provider trust", "feature prompt", "conversation"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("setup receipt help retained prohibited content %q:\n%s", forbidden, text)
		}
	}
}

func TestSetupApplyAndNoWriteRequireBoundedContinuationInputs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run(
		context.Background(),
		[]string{"setup", "no-write"},
		strings.NewReader(""),
		&stdout,
		&stderr,
		productionService,
	)
	if err == nil || !strings.Contains(err.Error(), "requires --plan") {
		t.Fatalf("setup no-write error = %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("failed setup no-write wrote stdout: %q", stdout.String())
	}
}
