package contractcmd

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/heurema/goalrail/apps/cli/internal/contract"
	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
	"github.com/heurema/goalrail/apps/cli/internal/spine"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

func Run(ctx context.Context, out *term.Output, workDir string, args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		_, err := fmt.Fprint(out.Stdout, Usage())
		return err
	}
	if args[0] != "validate" {
		return exitcode.UsageError(fmt.Errorf("unknown contract command %q", args[0]))
	}
	return runValidate(ctx, out, workDir, args[1:])
}

func runValidate(ctx context.Context, out *term.Output, workDir string, args []string) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail contract validate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	fileValue := flags.String("file", "", "contract JSON file")
	formatValue := flags.String("format", string(term.FormatText), "output format: text or json")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, writeErr := fmt.Fprint(out.Stdout, ValidateUsage())
			return writeErr
		}
		return exitcode.UsageError(err)
	}

	if flags.NArg() != 0 {
		return exitcode.UsageError(fmt.Errorf("unexpected arguments: %v", flags.Args()))
	}
	if *fileValue == "" {
		return exitcode.UsageError(errors.New("missing required --file <contract.json>"))
	}

	format, err := term.ParseFormat(*formatValue)
	if err != nil {
		return exitcode.UsageError(err)
	}

	filePath := resolvePath(workDir, *fileValue)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return exitcode.RuntimeError(fmt.Errorf("read contract file %q: %w", *fileValue, err))
	}

	var parsed spine.Contract
	if err := json.Unmarshal(data, &parsed); err != nil {
		return exitcode.RuntimeError(fmt.Errorf("parse contract file %q: %w", *fileValue, err))
	}

	report := contract.Validate(parsed)
	if format == term.FormatJSON {
		if err := term.WriteJSON(out.Stdout, report); err != nil {
			return err
		}
	} else if err := writeText(out.Stdout, report); err != nil {
		return err
	}

	if !report.Valid {
		return exitcode.ValidationError(contract.ErrValidationFailed)
	}
	return nil
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

func writeText(w io.Writer, report spine.ContractValidationReport) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Contract validation: %s\n", report.ContractID)
	if report.Valid {
		b.WriteString("Status: valid\n")
	} else {
		b.WriteString("Status: invalid\n\nFindings:\n")
		for _, finding := range report.Findings {
			fmt.Fprintf(&b, "- [%s] %s: %s\n", finding.Severity, finding.Field, finding.Message)
		}
	}

	_, err := fmt.Fprint(w, b.String())
	return err
}

func Usage() string {
	return "Usage: goalrail contract <command> [options]\n\nCommands:\n  validate    validate a contract JSON file\n\nRun goalrail contract validate --help for validate options.\n"
}

func ValidateUsage() string {
	return "Usage: goalrail contract validate --file <contract.json> [--format text|json]\n\nValidates the minimum contract fields needed before approval or execution.\n"
}
