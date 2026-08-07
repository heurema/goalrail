package doctor

import (
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/harness"
)

// After an authorized setup the true invocation is the toolchain on this
// machine. A bundle exists so the compiler is fetched once, pinned by digest and
// verified per file; reporting a package name instead sends every later planning
// command back to the registry the bundle was installed to avoid — and on a
// machine with no Node it names a command that cannot run at all.
func TestTheInvocationNamesTheInstalledToolchain(t *testing.T) {
	planning := PlanningState{
		Runtime: ComponentReadiness{
			State: ComponentReady, Path: "/home/owner/.local/share/goalrail/bundles/v0.2.1/linux_arm64/runtime/node/bin/node",
		},
		Compiler: ComponentReadiness{
			State: ComponentReady, Path: "/home/owner/.local/share/goalrail/bundles/v0.2.1/linux_arm64/compiler/node_modules/@fission-ai/openspec/bin/openspec.js",
		},
	}

	invocation := plannedInvocation(planning)

	if strings.Contains(invocation, "npx") {
		t.Fatalf("a verified installation is still told to resolve a package: %q", invocation)
	}
	for _, want := range []string{planning.Runtime.Path, planning.Compiler.Path, "new change <name> --schema " + harness.SchemaName} {
		if !strings.Contains(invocation, want) {
			t.Fatalf("the invocation omits %q: %q", want, invocation)
		}
	}
}

// Where no bundle answers, naming the stock CLI is the honest answer: that is
// what the agent would have to reach for. This repository's own release workflow
// also reads the pinned package out of this field, so the fallback's shape is
// depended on rather than incidental.
func TestTheInvocationFallsBackToTheStockCommand(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		planning PlanningState
	}{
		{"nothing verified", PlanningState{}},
		{"runtime missing", PlanningState{
			Runtime:  ComponentReadiness{State: ComponentMissing},
			Compiler: ComponentReadiness{State: ComponentReady, Path: "/bundle/openspec.js"},
		}},
		{"ready without a path", PlanningState{
			Runtime:  ComponentReadiness{State: ComponentReady},
			Compiler: ComponentReadiness{State: ComponentReady},
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			invocation := plannedInvocation(testCase.planning)
			if invocation != harness.PinnedNewChange {
				t.Fatalf("invocation = %q, want the stock command", invocation)
			}
			if !strings.Contains(invocation, "--yes ") {
				t.Fatalf("the release workflow reads the pinned package after --yes, and it is absent: %q", invocation)
			}
		})
	}
}

// A reported command a reader cannot paste is one they will retype wrongly.
func TestTheInvocationSurvivesAnAwkwardPath(t *testing.T) {
	planning := PlanningState{
		Runtime:  ComponentReadiness{State: ComponentReady, Path: "/home/a b/bundle/node"},
		Compiler: ComponentReadiness{State: ComponentReady, Path: "/home/a b/bundle/openspec.js"},
	}

	invocation := plannedInvocation(planning)

	if !strings.Contains(invocation, `'/home/a b/bundle/node'`) {
		t.Fatalf("a path carrying a space is not quoted: %q", invocation)
	}
}
