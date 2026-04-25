package readinesscmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
	"github.com/heurema/goalrail/apps/cli/internal/readiness"
	"github.com/heurema/goalrail/apps/cli/internal/spine"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

func Run(ctx context.Context, out *term.Output, workDir string, args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		_, err := fmt.Fprint(out.Stdout, Usage())
		return err
	}
	if args[0] != "scan" {
		return exitcode.UsageError(fmt.Errorf("unknown readiness command %q", args[0]))
	}
	return runScan(ctx, out, workDir, args[1:])
}

func runScan(ctx context.Context, out *term.Output, workDir string, args []string) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail readiness scan", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	pathValue := flags.String("path", "", "filesystem path to scan")
	formatValue := flags.String("format", string(term.FormatText), "output format: text or json")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, writeErr := fmt.Fprint(out.Stdout, ScanUsage())
			return writeErr
		}
		return exitcode.UsageError(err)
	}

	if flags.NArg() != 0 {
		return exitcode.UsageError(fmt.Errorf("unexpected arguments: %v", flags.Args()))
	}
	if *pathValue == "" {
		return exitcode.UsageError(errors.New("missing required --path <path>"))
	}

	format, err := term.ParseFormat(*formatValue)
	if err != nil {
		return exitcode.UsageError(err)
	}

	scanPath := resolvePath(workDir, *pathValue)
	info, err := os.Stat(scanPath)
	if err != nil {
		return exitcode.RuntimeError(fmt.Errorf("scan path %q: %w", *pathValue, err))
	}
	if !info.IsDir() {
		return exitcode.RuntimeError(fmt.Errorf("scan path %q is not a directory", *pathValue))
	}

	report, err := readiness.Scan(os.DirFS(scanPath))
	if err != nil {
		return exitcode.RuntimeError(fmt.Errorf("scan path %q: %w", *pathValue, err))
	}

	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, report)
	}
	return writeText(out.Stdout, *pathValue, report)
}

func resolvePath(workDir, value string) string {
	if filepath.IsAbs(value) {
		return value
	}
	if workDir == "" {
		workDir = "."
	}
	return filepath.Join(workDir, value)
}

func writeText(w io.Writer, scannedPath string, report spine.ReadinessReport) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Readiness scan: %s\n", scannedPath)
	fmt.Fprintf(&b, "Score: %d/100\n", report.Score)
	fmt.Fprintf(&b, "Status: %s\n\n", report.Status)

	b.WriteString("Findings:\n")
	for _, finding := range report.Findings {
		fmt.Fprintf(&b, "- [%s] %s: %s\n", finding.Status, finding.Check, finding.Message)
	}

	b.WriteString("\nEvidence:\n")
	for _, item := range report.Evidence {
		detail := item.Detail
		if len(item.Paths) > 0 {
			detail = strings.Join(item.Paths, ", ")
			if item.Detail != "" {
				detail += " (" + item.Detail + ")"
			}
		}
		fmt.Fprintf(&b, "- %s: %s\n", item.Check, detail)
	}

	b.WriteString("\nRecommended next actions:\n")
	for _, action := range report.RecommendedNextActions {
		fmt.Fprintf(&b, "- %s\n", action)
	}

	_, err := fmt.Fprint(w, b.String())
	return err
}

func Usage() string {
	return "Usage: goalrail readiness <command> [options]\n\nCommands:\n  scan    scan local repository readiness evidence\n\nRun goalrail readiness scan --help for scan options.\n"
}

func ScanUsage() string {
	return "Usage: goalrail readiness scan --path <path> [--format text|json]\n\nScans local filesystem evidence only. It does not shell out to git, npm, go, or other external commands.\n"
}
