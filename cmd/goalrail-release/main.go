package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/heurema/goalrail/internal/releasebundle"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: goalrail-release <lock-check|binary-version|build|verify> [flags]")
	}
	switch args[0] {
	case "lock-check":
		set := flag.NewFlagSet("lock-check", flag.ContinueOnError)
		set.SetOutput(stderr)
		repo := set.String("repo", ".", "Goalrail repository root")
		if err := set.Parse(args[1:]); err != nil {
			return err
		}
		if set.NArg() != 0 {
			return fmt.Errorf("lock-check accepts no positional arguments")
		}
		summary, err := releasebundle.CheckSourceLock(*repo)
		if err != nil {
			return err
		}
		return writeJSON(stdout, summary)
	case "binary-version":
		set := flag.NewFlagSet("binary-version", flag.ContinueOnError)
		set.SetOutput(stderr)
		path := set.String("path", "", "Goalrail binary path")
		if err := set.Parse(args[1:]); err != nil {
			return err
		}
		if set.NArg() != 0 || *path == "" {
			return fmt.Errorf("binary-version requires exactly --path")
		}
		version, err := releasebundle.ReadGoBinaryVersion(*path)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, version)
		return err
	case "build":
		options, err := parseReleaseFlags("build", args[1:], stderr)
		if err != nil {
			return err
		}
		metadata, err := releasebundle.Build(ctx, releasebundle.BuildOptions{
			RepoRoot: options.RepoRoot, DistDir: options.DistDir,
			SourceCacheDir: options.SourceCacheDir, ReleaseVersion: options.ReleaseVersion,
			FetchSource: fetchSource,
		})
		if err != nil {
			return err
		}
		return writeJSON(stdout, metadata)
	case "verify":
		options, err := parseReleaseFlags("verify", args[1:], stderr)
		if err != nil {
			return err
		}
		if err := releasebundle.Verify(ctx, releasebundle.VerifyOptions{
			RepoRoot: options.RepoRoot, DistDir: options.DistDir,
			SourceCacheDir: options.SourceCacheDir, ReleaseVersion: options.ReleaseVersion,
		}); err != nil {
			return err
		}
		return writeJSON(stdout, map[string]any{
			"status":          "verified",
			"release_version": options.ReleaseVersion,
		})
	default:
		return fmt.Errorf("unknown goalrail-release command %q", args[0])
	}
}

type releaseFlags struct {
	RepoRoot       string
	DistDir        string
	SourceCacheDir string
	ReleaseVersion string
}

func parseReleaseFlags(name string, args []string, stderr io.Writer) (releaseFlags, error) {
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	set.SetOutput(stderr)
	var options releaseFlags
	set.StringVar(&options.RepoRoot, "repo", ".", "Goalrail repository root")
	set.StringVar(&options.DistDir, "dist", "", "release artifact directory outside the checkout")
	set.StringVar(&options.SourceCacheDir, "source-cache", "", "verified source cache outside the checkout")
	set.StringVar(&options.ReleaseVersion, "version", "", "exact v-prefixed release version")
	if err := set.Parse(args); err != nil {
		return releaseFlags{}, err
	}
	if set.NArg() != 0 {
		return releaseFlags{}, fmt.Errorf("%s accepts no positional arguments", name)
	}
	if options.DistDir == "" || options.SourceCacheDir == "" || options.ReleaseVersion == "" {
		return releaseFlags{}, fmt.Errorf("%s requires --dist, --source-cache, and --version", name)
	}
	return options, nil
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}
