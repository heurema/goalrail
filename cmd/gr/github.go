package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/heurema/goalrail/internal/admissiongit"
	"github.com/heurema/goalrail/internal/admissionlocal"
	"github.com/heurema/goalrail/internal/githubadmission"
)

func runCommitMessageCheck(args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("commit-msg", flag.ContinueOnError)
	set.SetOutput(stderr)
	messagePath := set.String("file", "", "commit message file supplied by Git")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if set.NArg() != 0 || strings.TrimSpace(*messagePath) == "" {
		return errors.New("commit-msg requires --file")
	}
	file, err := os.Open(*messagePath)
	if err != nil {
		return fmt.Errorf("open commit message: %w", err)
	}
	workUnitID, validationErr := admissionlocal.ValidateCommitMessage(file)
	closeErr := file.Close()
	if validationErr != nil {
		return validationErr
	}
	if closeErr != nil {
		return closeErr
	}
	_, err = fmt.Fprintln(stdout, workUnitID)
	return err
}

func runGitHubCollect(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("github-collect", flag.ContinueOnError)
	set.SetOutput(stderr)
	repository := set.String("repo", "", "repository containing the frozen Git objects")
	base := set.String("base", "", "full immutable base commit ID")
	head := set.String("head", "", "full immutable head commit ID")
	eventName := set.String("event-name", "", "GitHub workflow event name")
	eventPath := set.String("event", "", "GitHub workflow event JSON path")
	outputPath := set.String("output", "", "exclusive temporary packet path")
	apiBase := set.String("api", "https://api.github.com", "GitHub API base URL")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if set.NArg() != 0 || *repository == "" || *base == "" || *head == "" || *eventName == "" || *eventPath == "" || *outputPath == "" {
		return errors.New("github-collect requires --repo, --base, --head, --event-name, --event, and --output")
	}
	eventFile, err := os.Open(*eventPath)
	if err != nil {
		return fmt.Errorf("open GitHub event: %w", err)
	}
	event, decodeErr := githubadmission.DecodeWorkflowEvent(*eventName, eventFile)
	closeErr := eventFile.Close()
	if decodeErr != nil {
		return decodeErr
	}
	if closeErr != nil {
		return closeErr
	}
	seed, err := admissiongit.CollectPacketSeed(ctx, *repository, *base, *head)
	if err != nil {
		return err
	}
	reader, err := githubadmission.NewRESTReader(*apiBase, os.Getenv("GITHUB_TOKEN"), nil)
	if err != nil {
		return err
	}
	artifact, err := githubadmission.Collect(ctx, githubadmission.CollectRequest{Seed: seed, Event: event, Reader: reader})
	if err != nil {
		return err
	}
	if err := writeExclusivePacket(*outputPath, artifact.CanonicalJSON()); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "wrote %s\n", filepath.Base(*outputPath))
	return err
}

func writeExclusivePacket(path string, raw []byte) (err error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return errors.New("temporary packet output must be a normalized absolute path")
	}
	info, statErr := os.Lstat(path)
	if statErr == nil {
		if info.Mode()&fs.ModeSymlink != 0 {
			return errors.New("temporary packet output is a symlink")
		}
		return errors.New("temporary packet output already exists")
	}
	if !errors.Is(statErr, fs.ErrNotExist) {
		return statErr
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	remove := true
	defer func() {
		if remove {
			_ = os.Remove(path)
		}
	}()
	if _, err := file.Write(raw); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	remove = false
	return nil
}
