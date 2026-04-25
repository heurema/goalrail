package proof

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/heurema/goalrail/apps/cli/internal/spine"
)

func RenderText(packet spine.Proof) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Proof: %s\n", packet.ID)
	fmt.Fprintf(&b, "Contract: %s\n", packet.ContractID)
	fmt.Fprintf(&b, "Run: %s\n", packet.RunID)
	fmt.Fprintf(&b, "Verdict: %s\n", packet.Verdict)
	if packet.DecisionSummary != "" {
		fmt.Fprintf(&b, "Decision summary: %s\n", packet.DecisionSummary)
	}

	writeList(&b, "Changed scope", packet.ChangedScope)
	writeList(&b, "Unchanged scope", packet.UnchangedScope)

	b.WriteString("Coverage:\n")
	if len(packet.Coverage) == 0 {
		b.WriteString("- none\n")
	} else {
		for _, item := range packet.Coverage {
			fmt.Fprintf(&b, "- [%s] %s", item.Status, item.Criterion)
			if len(item.Evidence) > 0 {
				fmt.Fprintf(&b, " (evidence: %s)", strings.Join(item.Evidence, "; "))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("Checks:\n")
	if len(packet.Checks) == 0 {
		b.WriteString("- none\n")
	} else {
		for _, check := range packet.Checks {
			fmt.Fprintf(&b, "- [%s] %s", check.Status, check.Name)
			if check.Note != "" {
				fmt.Fprintf(&b, ": %s", check.Note)
			}
			b.WriteString("\n")
		}
	}

	writeList(&b, "Residual risks", packet.ResidualRisks)
	return b.String()
}

func RenderJSON(w io.Writer, packet spine.Proof) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(packet)
}

func writeList(b *strings.Builder, title string, values []string) {
	fmt.Fprintf(b, "%s:\n", title)
	if len(values) == 0 {
		b.WriteString("- none\n")
		return
	}
	for _, value := range values {
		fmt.Fprintf(b, "- %s\n", value)
	}
}
