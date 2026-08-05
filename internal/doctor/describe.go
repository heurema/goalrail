package doctor

import (
	"fmt"
	"strings"
)

func Describe(diagnosis Diagnosis) string {
	var report strings.Builder
	fmt.Fprintf(&report, "goalrail %s — %s\n", diagnosis.Version, diagnosis.Repository)
	fmt.Fprintf(&report, "state: %s\n", diagnosis.Category)
	fmt.Fprintf(&report, "managed: %t", diagnosis.Managed)
	if diagnosis.ProjectID != "" {
		fmt.Fprintf(&report, " (%s)", diagnosis.ProjectID)
	}
	report.WriteByte('\n')
	fmt.Fprintf(&report, "local readiness: %t\n", diagnosis.LocallyReady)
	fmt.Fprintf(&report, "lineage readiness: %t\n", diagnosis.LineageReady)
	fmt.Fprintf(&report, "shared admission active: %t (%s)\n", diagnosis.SharedAdmissionActive, diagnosis.SharedAdmission.State)
	fmt.Fprintf(&report, "fully enforced: %t\n", diagnosis.Working)
	if diagnosis.Managed {
		fmt.Fprintf(&report, "project canon: current=%t (%s)\n", diagnosis.ProjectCanon.Current, diagnosis.ProjectCanon.Canon)
		fmt.Fprintf(&report, "planning: ready=%t, bundle=%s, runtime=%s, compiler=%s\n",
			diagnosis.Planning.Ready,
			diagnosis.Planning.Bundle.State,
			diagnosis.Planning.Runtime.State,
			diagnosis.Planning.Compiler.State,
		)
		fmt.Fprintf(&report, "shared integration: prepared=%t; repository files alone do not prove activation\n", diagnosis.PreparedAdmission.Prepared)
		fmt.Fprintf(&report, "invocation: %s\n", diagnosis.Invocation)
	}
	for _, reason := range diagnosis.Reasons {
		fmt.Fprintf(&report, "\n%s [%s]: %s\n", reason.Code, reason.State, reason.Detail)
		if reason.Path != "" {
			fmt.Fprintf(&report, "path: %s\n", reason.Path)
		}
		if reason.NextAction != "" {
			fmt.Fprintf(&report, "next: %s\n", reason.NextAction)
		}
	}
	return report.String()
}
