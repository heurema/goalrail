package projectcmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
	"github.com/heurema/goalrail/apps/cli/internal/projectconfig"
	"github.com/heurema/goalrail/apps/cli/internal/projectscan"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

const nextSuggestedCommand = "goalrail work start --title <title>"

type Options struct {
	CacheRoot string
	Now       func() time.Time
}

type Output struct {
	RepoBindingID     string                                 `json:"repo_binding_id"`
	CanonicalRepoRoot string                                 `json:"canonical_repo_root"`
	HeadSHA           string                                 `json:"head_sha"`
	LocalCacheDir     string                                 `json:"local_cache_dir"`
	BaselineRebuilt   bool                                   `json:"baseline_rebuilt"`
	Baseline          *projectscan.RepositoryBaselineProfile `json:"baseline"`
	Overlay           projectscan.WorkspaceOverlay           `json:"overlay"`
	Freshness         projectscan.FreshnessResult            `json:"freshness"`
	NextCommand       string                                 `json:"next_suggested_command,omitempty"`
}

func Run(ctx context.Context, out *term.Output, workDir string, args []string) error {
	return RunWithOptions(ctx, out, workDir, args, Options{})
}

func RunWithOptions(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if len(args) == 0 {
		_, err := fmt.Fprint(out.Stdout, Usage())
		return err
	}

	switch args[0] {
	case "--help", "-h":
		_, err := fmt.Fprint(out.Stdout, Usage())
		return err
	case "scan":
		return runScan(ctx, out, workDir, args[1:], options)
	case "status":
		return runStatus(ctx, out, workDir, args[1:], options)
	default:
		return exitcode.UsageError(fmt.Errorf("unknown project command %q", args[0]))
	}
}

func runScan(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail project scan", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	formatValue := flags.String("format", string(term.FormatText), "output format: text or json")
	refresh := flags.Bool("refresh", false, "refresh the local Project Scan baseline cache for the current HEAD")
	rerun := flags.Bool("rerun", false, "rebuild the local RepositoryBaselineProfile for the current HEAD")
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
	format, err := term.ParseFormat(*formatValue)
	if err != nil {
		return exitcode.UsageError(err)
	}

	facts, config, err := loadProjectContext(ctx, workDir, "goalrail project scan requires a Git worktree with .goalrail/project.yml; run goalrail init first")
	if err != nil {
		return err
	}
	cache := projectscan.NewCache(options.CacheRoot)
	cacheDir, err := cache.Directory(config.RepoBindingID, facts.CanonicalRepoRoot)
	if err != nil {
		return exitcode.RuntimeError(fmt.Errorf("resolve project scan cache: %w", err))
	}

	baseline, ok, err := cache.LoadLatestBaseline(config.RepoBindingID, facts.CanonicalRepoRoot)
	if err != nil {
		return exitcode.RuntimeError(fmt.Errorf("read project scan cache: %w", err))
	}
	needsRebuild := *refresh || *rerun || !ok || baseline.SchemaVersion != projectscan.SchemaVersion || baseline.HeadSHA != facts.HeadSHA
	rebuilt := false
	if needsRebuild {
		built, err := projectscan.BuildBaseline(ctx, facts.CanonicalRepoRoot, config.RepoBindingID, projectscan.DefaultBuildOptions())
		if err != nil {
			return exitcode.RuntimeError(fmt.Errorf("build project baseline: %w", err))
		}
		if err := cache.WriteBaseline(built); err != nil {
			return exitcode.RuntimeError(fmt.Errorf("write project baseline cache: %w", err))
		}
		baseline = built
		ok = true
		rebuilt = true
	}

	overlay, rawStatus, err := projectscan.BuildOverlay(ctx, facts.CanonicalRepoRoot, config.RepoBindingID, &baseline, projectscan.OverlayOptions{Now: options.Now})
	if err != nil {
		return exitcode.RuntimeError(fmt.Errorf("refresh project overlay: %w", err))
	}
	if err := cache.WriteOverlay(overlay, rawStatus); err != nil {
		return exitcode.RuntimeError(fmt.Errorf("write project overlay cache: %w", err))
	}

	var baselinePtr *projectscan.RepositoryBaselineProfile
	if ok {
		baselinePtr = &baseline
	}
	output := Output{
		RepoBindingID:     config.RepoBindingID,
		CanonicalRepoRoot: facts.CanonicalRepoRoot,
		HeadSHA:           facts.HeadSHA,
		LocalCacheDir:     cacheDir,
		BaselineRebuilt:   rebuilt,
		Baseline:          baselinePtr,
		Overlay:           overlay,
		Freshness:         projectscan.EvaluateFreshness(facts.HeadSHA, baselinePtr, overlay),
		NextCommand:       nextSuggestedCommand,
	}
	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	return writeScanText(out.Stdout, output)
}

func runStatus(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail project status", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	formatValue := flags.String("format", string(term.FormatText), "output format: text or json")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, writeErr := fmt.Fprint(out.Stdout, StatusUsage())
			return writeErr
		}
		return exitcode.UsageError(err)
	}
	if flags.NArg() != 0 {
		return exitcode.UsageError(fmt.Errorf("unexpected arguments: %v", flags.Args()))
	}
	format, err := term.ParseFormat(*formatValue)
	if err != nil {
		return exitcode.UsageError(err)
	}

	facts, config, err := loadProjectContext(ctx, workDir, "goalrail project status requires a Git worktree with .goalrail/project.yml; run goalrail init first")
	if err != nil {
		return err
	}
	cache := projectscan.NewCache(options.CacheRoot)
	cacheDir, err := cache.Directory(config.RepoBindingID, facts.CanonicalRepoRoot)
	if err != nil {
		return exitcode.RuntimeError(fmt.Errorf("resolve project scan cache: %w", err))
	}
	baseline, ok, err := cache.LoadLatestBaseline(config.RepoBindingID, facts.CanonicalRepoRoot)
	if err != nil {
		return exitcode.RuntimeError(fmt.Errorf("read project scan cache: %w", err))
	}
	var baselinePtr *projectscan.RepositoryBaselineProfile
	if ok {
		baselinePtr = &baseline
	}
	overlay, rawStatus, err := projectscan.BuildOverlay(ctx, facts.CanonicalRepoRoot, config.RepoBindingID, baselinePtr, projectscan.OverlayOptions{Now: options.Now})
	if err != nil {
		return exitcode.RuntimeError(fmt.Errorf("refresh project overlay: %w", err))
	}
	if err := cache.WriteOverlay(overlay, rawStatus); err != nil {
		return exitcode.RuntimeError(fmt.Errorf("write project overlay cache: %w", err))
	}

	output := Output{
		RepoBindingID:     config.RepoBindingID,
		CanonicalRepoRoot: facts.CanonicalRepoRoot,
		HeadSHA:           facts.HeadSHA,
		LocalCacheDir:     cacheDir,
		BaselineRebuilt:   false,
		Baseline:          baselinePtr,
		Overlay:           overlay,
		Freshness:         projectscan.EvaluateFreshness(facts.HeadSHA, baselinePtr, overlay),
		NextCommand:       nextSuggestedCommand,
	}
	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	return writeStatusText(out.Stdout, output)
}

func loadProjectContext(ctx context.Context, workDir string, notGitMessage string) (projectscan.GitFacts, projectconfig.Config, error) {
	facts, err := projectscan.DiscoverGit(ctx, workDir)
	if err != nil {
		if errors.Is(err, projectscan.ErrNotGitRepository) {
			return projectscan.GitFacts{}, projectconfig.Config{}, exitcode.UsageError(errors.New(notGitMessage))
		}
		return projectscan.GitFacts{}, projectconfig.Config{}, exitcode.RuntimeError(err)
	}
	config, ok, err := projectconfig.Read(facts.CanonicalRepoRoot)
	if err != nil {
		return projectscan.GitFacts{}, projectconfig.Config{}, err
	}
	if !ok {
		return projectscan.GitFacts{}, projectconfig.Config{}, exitcode.UsageError(errors.New("missing .goalrail/project.yml; run goalrail init first"))
	}
	if strings.TrimSpace(config.RepoBindingID) == "" {
		return projectscan.GitFacts{}, projectconfig.Config{}, exitcode.ValidationError(errors.New("local .goalrail/project.yml is missing repo_binding_id; run goalrail init again"))
	}
	return facts, config, nil
}

func Usage() string {
	return "Usage: goalrail project <command> [options]\n\nCommands:\n  scan      build or refresh the local Project Scan baseline and overlay\n  status    refresh overlay and report Project Scan freshness\n\nRun goalrail project <command> --help for command usage.\n"
}

func ScanUsage() string {
	return "Usage: goalrail project scan [--format text|json] [--refresh] [--rerun]\n\nBuilds or refreshes the local Project Scan v0 RepositoryBaselineProfile for the current committed HEAD and refreshes the WorkspaceOverlay. Use --refresh to force a local baseline cache refresh; --rerun remains supported for compatibility. Requires a Git worktree with .goalrail/project.yml. It does not call the server, clone repositories, run checks, upload source, create a context pack, gate, or proof.\n"
}

func StatusUsage() string {
	return "Usage: goalrail project status [--format text|json]\n\nRefreshes the cheap local WorkspaceOverlay and reports Project Scan freshness. It does not rebuild the RepositoryBaselineProfile by default and does not call the server.\n"
}

func writeScanText(w io.Writer, output Output) error {
	var b strings.Builder
	b.WriteString("Project scan complete\n\n")
	writeCommonText(&b, output)
	fmt.Fprintf(&b, "\nNext: %s\n", output.NextCommand)
	_, err := fmt.Fprint(w, b.String())
	return err
}

func writeStatusText(w io.Writer, output Output) error {
	var b strings.Builder
	b.WriteString("Project status\n\n")
	writeCommonText(&b, output)
	_, err := fmt.Fprint(w, b.String())
	return err
}

func writeCommonText(b *strings.Builder, output Output) {
	baselineID := "missing"
	baselineStatus := projectscan.FreshnessMissingBaseline
	partiality := []string{}
	if output.Baseline != nil {
		baselineID = output.Baseline.RepositoryBaselineProfileID
		baselineStatus = output.Baseline.Status
		partiality = output.Baseline.Partiality.Reasons
	}
	fmt.Fprintf(b, "Baseline: %s\n", baselineID)
	fmt.Fprintf(b, "Repo binding: %s\n", output.RepoBindingID)
	fmt.Fprintf(b, "HEAD: %s\n", shortSHA(output.HeadSHA))
	fmt.Fprintf(b, "Status: %s\n", baselineStatus)
	fmt.Fprintf(b, "Overlay: %s\n", output.Overlay.State)
	if len(output.Overlay.ScanCriticalChangedPaths) > 0 {
		fmt.Fprintf(b, "Scan-critical changes: %s\n", strings.Join(output.Overlay.ScanCriticalChangedPaths, ", "))
	}
	if len(partiality) == 0 && len(output.Overlay.PartialityReasons) == 0 {
		b.WriteString("Partiality: none\n")
	} else {
		reasons := append([]string{}, partiality...)
		reasons = append(reasons, output.Overlay.PartialityReasons...)
		fmt.Fprintf(b, "Partiality: %s\n", strings.Join(uniqueStrings(reasons), ", "))
	}
	freshness := output.Freshness.Status
	if output.Freshness.StructuralRescanRecommended {
		freshness = "structural_rescan_recommended"
	}
	fmt.Fprintf(b, "Freshness: %s\n", freshness)
}

func shortSHA(value string) string {
	if len(value) > 12 {
		return value[:12]
	}
	return value
}

func uniqueStrings(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
