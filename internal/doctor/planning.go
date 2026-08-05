package doctor

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/harness"
)

type ComponentState string

const (
	ComponentReady               ComponentState = "ready"
	ComponentMissing             ComponentState = "missing"
	ComponentIncompatible        ComponentState = "incompatible"
	ComponentUnverifiedIntegrity ComponentState = "unverified_integrity"
	ComponentInvalid             ComponentState = "invalid"
	ComponentNotRequired         ComponentState = "not_required"
)

type ComponentReadiness struct {
	Kind              string              `json:"kind"`
	ID                string              `json:"id"`
	RequiredVersion   string              `json:"required_version,omitempty"`
	RequiredIntegrity domain.SHA256Digest `json:"required_integrity,omitempty"`
	ObservedVersion   string              `json:"observed_version,omitempty"`
	Path              string              `json:"path,omitempty"`
	State             ComponentState      `json:"state"`
	Detail            string              `json:"detail,omitempty"`
}

type PlanningObservation struct {
	Bundle   ComponentReadiness `json:"bundle"`
	Runtime  ComponentReadiness `json:"runtime"`
	Compiler ComponentReadiness `json:"compiler"`
}

type PlanningObserver interface {
	ObservePlanning(context.Context, domain.SetupProfile) PlanningObservation
}

type PlanningObserverFunc func(context.Context, domain.SetupProfile) PlanningObservation

func (function PlanningObserverFunc) ObservePlanning(ctx context.Context, profile domain.SetupProfile) PlanningObservation {
	return function(ctx, profile)
}

type defaultPlanningObserver struct{}

func (defaultPlanningObserver) ObservePlanning(ctx context.Context, profile domain.SetupProfile) PlanningObservation {
	observation := PlanningObservation{
		Bundle: ComponentReadiness{
			Kind: "goalrail_bundle", ID: "goalrail", RequiredVersion: profile.CompatibleGoalrailBundle,
			ObservedVersion: harness.Version,
		},
	}
	if compatibleGoalrailBundle(harness.Version, profile.CompatibleGoalrailBundle) {
		observation.Bundle.State = ComponentReady
	} else if strings.TrimSpace(harness.Version) == "" || harness.Version == "unknown" || harness.Version == "(devel)" {
		observation.Bundle.State = ComponentUnverifiedIntegrity
		observation.Bundle.Detail = "this development binary has no release identity compatible with the declared setup bundle"
	} else {
		observation.Bundle.State = ComponentIncompatible
		observation.Bundle.Detail = "the running Goalrail release is outside the declared compatible bundle range"
	}
	observation.Runtime = observeExecutable(ctx, "runtime", profile.Planning.Runtime, profile.Planning.RuntimeVersion)
	compilerExecutable := profile.Planning.Adapter.ID
	if compilerExecutable == "" {
		compilerExecutable = executableName(profile.Planning.Compiler)
	}
	observation.Compiler = observeExecutable(ctx, "compiler", compilerExecutable, profile.Planning.CompilerVersion)
	observation.Compiler.ID = profile.Planning.Compiler
	return observation
}

func observeExecutable(ctx context.Context, kind, executable, requiredVersion string) ComponentReadiness {
	component := ComponentReadiness{Kind: kind, ID: executable, RequiredVersion: requiredVersion}
	if strings.TrimSpace(executable) == "" {
		component.State = ComponentNotRequired
		return component
	}
	path, err := exec.LookPath(executable)
	if err != nil {
		component.State = ComponentMissing
		component.Detail = executable + " is not available on PATH"
		return component
	}
	component.Path = path
	commandContext, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	command := exec.CommandContext(commandContext, path, "--version")
	output := &boundedVersionOutput{limit: 8 << 10}
	command.Stdout, command.Stderr = output, output
	if err := command.Run(); err != nil {
		component.State = ComponentInvalid
		component.Detail = "version inspection failed"
		return component
	}
	component.ObservedVersion = extractVersion(output.String())
	if component.ObservedVersion == "" {
		component.State = ComponentInvalid
		component.Detail = "version output did not contain a bounded semantic version"
		return component
	}
	if component.ObservedVersion != strings.TrimPrefix(requiredVersion, "v") {
		component.State = ComponentIncompatible
		component.Detail = "observed version differs from the exact declared version"
		return component
	}
	component.State = ComponentReady
	return component
}

type boundedVersionOutput struct {
	value strings.Builder
	limit int
}

func (output *boundedVersionOutput) Write(payload []byte) (int, error) {
	original := len(payload)
	remaining := output.limit - output.value.Len()
	if remaining > 0 {
		if len(payload) > remaining {
			payload = payload[:remaining]
		}
		_, _ = output.value.Write(payload)
	}
	return original, nil
}

func (output *boundedVersionOutput) String() string { return output.value.String() }

func extractVersion(raw string) string {
	for _, field := range strings.Fields(raw) {
		candidate := strings.Trim(strings.TrimPrefix(field, "v"), " ,;()[]")
		parts := strings.Split(candidate, ".")
		if len(parts) < 3 {
			continue
		}
		valid := true
		for index, part := range parts[:3] {
			if index == 2 {
				part, _, _ = strings.Cut(part, "-")
				part, _, _ = strings.Cut(part, "+")
			}
			if _, err := strconv.Atoi(part); err != nil {
				valid = false
			}
		}
		if valid {
			return strings.Join(parts[:3], ".")
		}
	}
	return ""
}

func compatibleGoalrailBundle(version, constraint string) bool {
	if constraint != ">=0.2.0 <0.3.0" {
		return false
	}
	version = strings.TrimPrefix(version, "v")
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}
	major, majorErr := strconv.Atoi(parts[0])
	minor, minorErr := strconv.Atoi(parts[1])
	return majorErr == nil && minorErr == nil && major == 0 && minor == 2
}

func executableName(packageName string) string {
	packageName = strings.TrimSpace(packageName)
	if slash := strings.LastIndex(packageName, "/"); slash >= 0 {
		packageName = packageName[slash+1:]
	}
	return strings.TrimPrefix(packageName, "@")
}
