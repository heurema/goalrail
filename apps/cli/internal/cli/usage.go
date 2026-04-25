package cli

import (
	"fmt"
	"strings"
)

func RootUsage(binary string, commands []Command) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Usage: %s <command> [options]\n\n", binary)
	b.WriteString("Goalrail local/demo CLI foundation. This CLI does not implement the production server, hosted execution, gate, or proof generation.\n\n")
	b.WriteString("Commands:\n")
	for _, cmd := range commands {
		fmt.Fprintf(&b, "  %-10s %s\n", cmd.Name, cmd.Summary)
	}
	fmt.Fprintf(&b, "\nRun %s <command> --help for command usage.\n", binary)
	return b.String()
}
