package proofcmd

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
	"github.com/heurema/goalrail/apps/cli/internal/proof"
	"github.com/heurema/goalrail/apps/cli/internal/spine"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

func Run(ctx context.Context, out *term.Output, workDir string, args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		_, err := fmt.Fprint(out.Stdout, Usage())
		return err
	}
	if args[0] != "show" {
		return exitcode.UsageError(fmt.Errorf("unknown proof command %q", args[0]))
	}
	return runShow(ctx, out, workDir, args[1:])
}

func runShow(ctx context.Context, out *term.Output, workDir string, args []string) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail proof show", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	fileValue := flags.String("file", "", "proof JSON file")
	formatValue := flags.String("format", string(term.FormatText), "output format: text or json")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, writeErr := fmt.Fprint(out.Stdout, ShowUsage())
			return writeErr
		}
		return exitcode.UsageError(err)
	}

	if flags.NArg() != 0 {
		return exitcode.UsageError(fmt.Errorf("unexpected arguments: %v", flags.Args()))
	}
	if *fileValue == "" {
		return exitcode.UsageError(errors.New("missing required --file <proof.json>"))
	}

	format, err := term.ParseFormat(*formatValue)
	if err != nil {
		return exitcode.UsageError(err)
	}

	filePath := resolvePath(workDir, *fileValue)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return exitcode.RuntimeError(fmt.Errorf("read proof file %q: %w", *fileValue, err))
	}

	var packet spine.Proof
	if err := json.Unmarshal(data, &packet); err != nil {
		return exitcode.RuntimeError(fmt.Errorf("parse proof file %q: %w", *fileValue, err))
	}

	if format == term.FormatJSON {
		return proof.RenderJSON(out.Stdout, packet)
	}

	_, err = fmt.Fprint(out.Stdout, proof.RenderText(packet))
	return err
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

func Usage() string {
	return "Usage: goalrail proof <command> [options]\n\nCommands:\n  show    render a proof JSON file\n\nThis command renders a provided proof packet only. It does not generate proof or run gate logic.\n\nRun goalrail proof show --help for show options.\n"
}

func ShowUsage() string {
	return "Usage: goalrail proof show --file <proof.json> [--format text|json]\n\nRenders a proof packet from local JSON. This does not generate a Goalrail server proof.\n"
}
