package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	openspecadapter "github.com/heurema/goalrail/internal/adapters/openspec"
	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/localrun"
)

type runService interface {
	Prepare(context.Context, io.Reader) (localrun.PreparedRun, error)
	InspectPrepared(domain.WorkSpecDigest) (localrun.PreparedRun, error)
	Start(context.Context, localrun.StartInput) (localrun.StartResult, error)
	Finish(context.Context, localrun.FinishInput) (localrun.TerminalReceipt, error)
	InspectRun(domain.RunID) (localrun.RunInspection, error)
}

type serviceFactory func(string) (runService, error)

type preparedOutput struct {
	Digest      domain.WorkSpecDigest `json:"digest"`
	WorkSpec    json.RawMessage       `json:"work_spec"`
	Preparation localrun.Preparation  `json:"preparation"`
	Claim       *localrun.LaunchClaim `json:"claim,omitempty"`
	State       localrun.RunState     `json:"state"`
}

func main() {
	if err := run(
		context.Background(),
		os.Args[1:],
		os.Stdin,
		os.Stdout,
		os.Stderr,
		productionService,
	); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(
	ctx context.Context,
	args []string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	factory serviceFactory,
) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: gr <init|connect|health|disconnect|prepare|inspect|start|finish> [flags]; run gr help")
	}
	switch args[0] {
	case "prepare":
		return runPrepare(ctx, args[1:], stdout, stderr, factory)
	case "inspect":
		return runInspect(args[1:], stdout, stderr, factory)
	case "start":
		return runStart(ctx, args[1:], stdin, stdout, stderr, factory)
	case "finish":
		return runFinish(ctx, args[1:], stdout, stderr, factory)
	case "init":
		return runInit(args[1:], stdout, stderr)
	case "connect":
		return runConnect(args[1:], stdout, stderr)
	case "disconnect":
		return runDisconnect(args[1:], stdout, stderr)
	case "health":
		return runHealth(args[1:], stdout, stderr)
	case "hook":
		// Ambient entry point: fail-quiet by contract, so a Goalrail problem
		// never breaks a session the user started for their own reasons.
		return runHook(args[1:], stdin, stdout)
	case "help":
		if len(args) == 1 {
			return writeHelp(stdout)
		}
		if len(args) == 2 && isCommand(args[1]) {
			return run(ctx, []string{args[1], "--help"}, stdin, stdout, stderr, factory)
		}
		return fmt.Errorf("usage: gr help [init|connect|health|disconnect|prepare|inspect|start|finish]")
	case "-h", "--help":
		return writeHelp(stdout)
	default:
		return fmt.Errorf("unknown gr command %q", args[0])
	}
}

func isCommand(value string) bool {
	switch value {
	case "prepare", "inspect", "start", "finish", "init", "connect", "disconnect", "health":
		return true
	default:
		return false
	}
}

func writeHelp(output io.Writer) error {
	_, err := fmt.Fprint(output, `usage: gr <init|connect|health|disconnect|prepare|inspect|start|finish> [flags]

Background attachment:
  connect → init → work normally; ordinary work needs no Goalrail command

  connect     attach Goalrail to a scaffold's sessions; needs --yes
  init        mark this repository as one Goalrail attaches to
  health      report whether the attachment is active, and what is missing
  disconnect  detach and remove everything connect added

  After connecting, your scaffold asks you to review and trust the registered
  commands; nothing runs until you do. "health" tells you if that step is
  pending, which otherwise looks exactly like a broken install.

Wrapper lifecycle (benchmark and owner-driven runs):
  prepare → inspect → start → finish

  prepare  freeze and display one WorkSpec; does not launch a provider
  inspect  display prepared state or a later run receipt
  start    explicitly request one prepared provider run; owner-assisted
  finish   record explicit check results and stop for review

Run "gr help <command>" or "gr <command> --help" for command flags.
`)
	return err
}

func productionService(stateDirectory string) (runService, error) {
	store, err := localrun.NewStore(stateDirectory)
	if err != nil {
		return nil, err
	}
	return localrun.NewService(
		store,
		localrun.GitObserver{},
		openspecadapter.IntentResolver{},
	), nil
}

func runPrepare(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
	factory serviceFactory,
) error {
	set := flag.NewFlagSet("prepare", flag.ContinueOnError)
	set.SetOutput(stderr)
	filePath := set.String("file", "", "path to one WorkSpec JSON file")
	stateDirectory := set.String("state-dir", "", "Goalrail local state directory")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if set.NArg() != 0 || strings.TrimSpace(*filePath) == "" {
		return fmt.Errorf("prepare requires exactly --file <work-spec.json>")
	}
	file, err := os.Open(*filePath)
	if err != nil {
		return fmt.Errorf("open WorkSpec: %w", err)
	}
	service, err := factory(*stateDirectory)
	if err != nil {
		file.Close()
		return err
	}
	prepared, prepareErr := service.Prepare(ctx, file)
	closeErr := file.Close()
	if prepareErr != nil {
		return prepareErr
	}
	if closeErr != nil {
		return fmt.Errorf("close WorkSpec: %w", closeErr)
	}
	return writeJSON(stdout, preparedOutput{
		Digest:      prepared.WorkSpec.Digest(),
		WorkSpec:    json.RawMessage(prepared.WorkSpec.CanonicalJSON()),
		Preparation: prepared.Preparation,
		Claim:       prepared.Claim,
		State:       prepared.State,
	})
}

func runInspect(
	args []string,
	stdout io.Writer,
	stderr io.Writer,
	factory serviceFactory,
) error {
	set := flag.NewFlagSet("inspect", flag.ContinueOnError)
	set.SetOutput(stderr)
	digest := set.String("digest", "", "prepared WorkSpec digest")
	runID := set.String("run", "", "generated run ID")
	stateDirectory := set.String("state-dir", "", "Goalrail local state directory")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if set.NArg() != 0 || (*digest == "") == (*runID == "") {
		return fmt.Errorf("inspect requires exactly one of --digest or --run")
	}
	service, err := factory(*stateDirectory)
	if err != nil {
		return err
	}
	if *digest != "" {
		if !validDigest(*digest) {
			return fmt.Errorf("invalid WorkSpec digest")
		}
		prepared, err := service.InspectPrepared(domain.WorkSpecDigest(*digest))
		if err != nil {
			return err
		}
		return writeJSON(stdout, preparedOutput{
			Digest:      prepared.WorkSpec.Digest(),
			WorkSpec:    json.RawMessage(prepared.WorkSpec.CanonicalJSON()),
			Preparation: prepared.Preparation,
			Claim:       prepared.Claim,
			State:       prepared.State,
		})
	}
	if !domain.IsCanonicalID(*runID) {
		return fmt.Errorf("invalid run ID")
	}
	inspection, err := service.InspectRun(domain.RunID(*runID))
	if err != nil {
		return err
	}
	return writeJSON(stdout, inspection)
}

func runStart(
	ctx context.Context,
	args []string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	factory serviceFactory,
) error {
	flags, adapterArguments := splitAdapterArguments(args)
	set := flag.NewFlagSet("start", flag.ContinueOnError)
	set.SetOutput(stderr)
	digest := set.String("digest", "", "prepared WorkSpec digest")
	adapter := set.String("adapter", "", "provider adapter; v0 recognizes codex but is not activated")
	stateDirectory := set.String("state-dir", "", "Goalrail local state directory")
	if err := set.Parse(flags); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if set.NArg() != 0 || !validDigest(*digest) {
		return fmt.Errorf("start requires --digest <sha256:...>")
	}
	if *adapter != "codex" {
		return fmt.Errorf("v0 recognizes only the codex adapter and does not activate it")
	}
	service, err := factory(*stateDirectory)
	if err != nil {
		return err
	}
	result, err := service.Start(ctx, localrun.StartInput{
		WorkSpecDigest: domain.WorkSpecDigest(*digest),
		Adapter:        *adapter,
		Arguments:      adapterArguments,
		Stdin:          stdin,
		Stdout:         stdout,
		Stderr:         stderr,
	})
	if err != nil {
		return err
	}
	return writeJSON(stdout, result)
}

func runFinish(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
	factory serviceFactory,
) error {
	set := flag.NewFlagSet("finish", flag.ContinueOnError)
	set.SetOutput(stderr)
	runID := set.String("run", "", "generated run ID")
	stateDirectory := set.String("state-dir", "", "Goalrail local state directory")
	var results resultFlags
	set.Var(&results, "result", "check result: <id>=<pass|fail|unavailable>[,<evidence-ref>[,<sha256>]]")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if set.NArg() != 0 || !domain.IsCanonicalID(*runID) {
		return fmt.Errorf("finish requires --run <canonical-run-id>")
	}
	service, err := factory(*stateDirectory)
	if err != nil {
		return err
	}
	receipt, err := service.Finish(ctx, localrun.FinishInput{
		RunID:   domain.RunID(*runID),
		Results: []localrun.CheckResult(results),
	})
	if err != nil {
		return err
	}
	return writeJSON(stdout, receipt)
}

type resultFlags []localrun.CheckResult

func (results *resultFlags) String() string {
	return fmt.Sprintf("%d result(s)", len(*results))
}

func (results *resultFlags) Set(value string) error {
	identity, remainder, found := strings.Cut(value, "=")
	if !found || !domain.IsCanonicalID(identity) {
		return fmt.Errorf("result must begin with a canonical check ID and '='")
	}
	fields := strings.Split(remainder, ",")
	if len(fields) == 0 || len(fields) > 3 {
		return fmt.Errorf("result must contain state and at most one evidence reference and digest")
	}
	result := localrun.CheckResult{
		ID:    domain.WorkSpecCheckID(identity),
		State: domain.CheckResultState(fields[0]),
	}
	if len(fields) >= 2 {
		result.EvidenceRef = fields[1]
	}
	if len(fields) == 3 {
		result.EvidenceDigest = fields[2]
	}
	switch result.State {
	case domain.CheckResultPass, domain.CheckResultFail, domain.CheckResultUnavailable:
	default:
		return fmt.Errorf("unsupported result state %q", result.State)
	}
	*results = append(*results, result)
	return nil
}

func splitAdapterArguments(args []string) ([]string, []string) {
	for index, argument := range args {
		if argument == "--" {
			return append([]string(nil), args[:index]...), append([]string(nil), args[index+1:]...)
		}
	}
	return append([]string(nil), args...), nil
}

func validDigest(value string) bool {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, character := range strings.TrimPrefix(value, "sha256:") {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return false
		}
	}
	return true
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("write JSON: %w", err)
	}
	return nil
}
