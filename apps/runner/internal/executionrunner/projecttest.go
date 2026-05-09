package executionrunner

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

type projectTestResult struct {
	ProcessStatus     string
	ExitCode          *int
	EnforcementReport enforcementReport
}

func (r *Runner) validateProjectTestPlan(plan executionCommandPlan, lease executionLease, run runStarted) error {
	if err := r.validateCommandPlanTrace(plan, lease, run); err != nil {
		return err
	}
	if plan.CommandKind != "project_test" {
		return fmt.Errorf("unsupported execution command kind %q", plan.CommandKind)
	}
	if plan.Action != "run_declared_test_target" {
		return fmt.Errorf("unsupported execution command action %q", plan.Action)
	}
	if strings.TrimSpace(plan.SourceProjectProbeReceiptID) == "" {
		return errors.New("execution command plan source project probe receipt is required")
	}
	if strings.TrimSpace(plan.SelectedTargetID) == "" {
		return errors.New("execution command plan selected target id is required")
	}
	if plan.DeclaredTestTarget == nil {
		return errors.New("execution command plan declared test target is required")
	}
	target := normalizeProjectTestTarget(*plan.DeclaredTestTarget)
	if !isSupportedProjectTestTarget(target) {
		return fmt.Errorf("unsupported project test target family %q", target.SourceKind)
	}
	if projectTestTargetID(target) != plan.SelectedTargetID {
		return fmt.Errorf("execution command plan selected target %q does not match declared target", plan.SelectedTargetID)
	}
	if target.SourcePath != "package.json" {
		return fmt.Errorf("execution command plan target source path %q is outside root package manifest scope", target.SourcePath)
	}
	if plan.ShellAllowed {
		return errors.New("execution command plan unexpectedly allows shell")
	}
	if len(plan.Argv) != 0 {
		return fmt.Errorf("execution command plan argv must be empty for project test, got %d arguments", len(plan.Argv))
	}
	if plan.WorkingDirectory != "." {
		return fmt.Errorf("execution command plan working directory %q is outside project test scope", plan.WorkingDirectory)
	}
	if _, err := projectTestWorkingDirectory(r.config.WorkspaceRoot, plan.WorkingDirectory); err != nil {
		return err
	}
	if len(plan.PathScope) != 1 || plan.PathScope[0] != "." {
		return fmt.Errorf("execution command plan path scope %#v is outside project test scope", plan.PathScope)
	}
	if plan.TimeoutSeconds <= 0 || plan.TimeoutSeconds > 120 || plan.MaxStdoutBytes != 0 || plan.MaxStderrBytes != 0 {
		return fmt.Errorf("execution command plan output/timeout policy = %d/%d/%d, want 1..120/0/0", plan.TimeoutSeconds, plan.MaxStdoutBytes, plan.MaxStderrBytes)
	}
	if plan.NetworkAllowed {
		return errors.New("execution command plan unexpectedly allows network")
	}
	if plan.WorkspaceWriteAllowed {
		return errors.New("execution command plan unexpectedly allows workspace writes")
	}
	if plan.ScratchWriteAllowed {
		return errors.New("execution command plan unexpectedly allows scratch writes")
	}
	if len(plan.AllowedArtifactKinds) != 0 {
		return fmt.Errorf("execution command plan allowed artifacts = %#v, want none", plan.AllowedArtifactKinds)
	}
	if plan.ChangedPathsAllowed {
		return errors.New("execution command plan unexpectedly allows changed paths")
	}
	if plan.RawSourceUploadAllowed {
		return errors.New("execution command plan unexpectedly allows raw source upload")
	}
	if plan.State != "planned" {
		return fmt.Errorf("execution command plan state %q is not planned", plan.State)
	}
	return nil
}

func rejectProjectTestExecution() projectTestResult {
	return projectTestResult{
		ProcessStatus: "policy_rejected",
		EnforcementReport: enforcementReport{
			NetworkPolicy:             "disabled_required",
			NetworkEnforcement:        "unavailable",
			WorkspaceWritePolicy:      "disabled_required",
			WorkspaceWriteEnforcement: "unavailable",
			ProcessTreeEnforcement:    "unavailable",
			ScratchWritePolicy:        "allowed_runner_local",
			Decision:                  "policy_rejected",
			Reason:                    "enforcement_unavailable",
		},
	}
}

func projectTestWorkingDirectory(workspaceRoot string, workingDirectory string) (string, error) {
	root := filepath.Clean(strings.TrimSpace(workspaceRoot))
	if root == "" || root == "." {
		return "", errors.New("workspace root is required")
	}
	if strings.TrimSpace(workingDirectory) != "." {
		return "", errors.New("unsupported working directory")
	}
	return root, nil
}

func normalizeProjectTestTarget(candidate projectProbeTestTargetCandidate) projectProbeTestTargetCandidate {
	candidate.Name = strings.TrimSpace(candidate.Name)
	candidate.SourcePath = strings.TrimSpace(candidate.SourcePath)
	candidate.SourceKind = strings.TrimSpace(candidate.SourceKind)
	return candidate
}

func isSupportedProjectTestTarget(candidate projectProbeTestTargetCandidate) bool {
	candidate = normalizeProjectTestTarget(candidate)
	if candidate.SourceKind != "package_json_script" {
		return false
	}
	return candidate.Name == "test" || strings.HasPrefix(candidate.Name, "test:")
}

func projectTestTargetID(candidate projectProbeTestTargetCandidate) string {
	candidate = normalizeProjectTestTarget(candidate)
	return candidate.SourcePath + "#" + candidate.SourceKind + ":" + candidate.Name
}
