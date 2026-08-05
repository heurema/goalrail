package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/heurema/goalrail/internal/admission"
	"github.com/heurema/goalrail/internal/admissiongit"
	"github.com/heurema/goalrail/internal/admissionlocal"
	"github.com/heurema/goalrail/internal/domain"
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
	collection, err := githubadmission.Collect(ctx, githubadmission.CollectRequest{Seed: seed, Event: event, Reader: reader})
	if err != nil {
		return err
	}
	if err := writeExclusivePacket(*outputPath, collection.Artifact.CanonicalJSON()); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "wrote %s\n", filepath.Base(*outputPath))
	return err
}

// runGitHubVerify is the authoritative shared-admission path. Provider
// collection and core verification happen inside one trusted invocation,
// because a packet written to disk cannot carry authenticated authority: only
// this process observed the provider.
func runGitHubVerify(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("github-verify", flag.ContinueOnError)
	set.SetOutput(stderr)
	repository := set.String("repo", "", "repository containing the frozen Git objects")
	base := set.String("base", "", "full immutable base commit ID")
	head := set.String("head", "", "full immutable head commit ID")
	eventName := set.String("event-name", "", "GitHub workflow event name")
	eventPath := set.String("event", "", "GitHub workflow event JSON path")
	apiBase := set.String("api", "https://api.github.com", "GitHub API base URL")
	packetPath := set.String("packet-out", "", "optional path for the canonical audit packet")
	jsonOutput := set.Bool("json", false, "emit canonical admission-result/v1 JSON")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return admissionCommandError{err: err, code: 2}
	}
	if set.NArg() != 0 || *repository == "" || *base == "" || *head == "" || *eventName == "" || *eventPath == "" || !*jsonOutput {
		return admissionCommandError{
			err:  errors.New("github-verify requires --repo, --base, --head, --event-name, --event, and --json"),
			code: 2,
		}
	}
	eventFile, err := os.Open(*eventPath)
	if err != nil {
		return admissionCommandError{err: fmt.Errorf("open GitHub event: %w", err), code: 2}
	}
	event, decodeErr := githubadmission.DecodeWorkflowEvent(*eventName, eventFile)
	closeErr := eventFile.Close()
	if decodeErr != nil {
		return admissionCommandError{err: decodeErr, code: 2}
	}
	if closeErr != nil {
		return admissionCommandError{err: closeErr, code: 2}
	}
	seed, err := admissiongit.CollectPacketSeed(ctx, *repository, *base, *head)
	if err != nil {
		return admissionCommandError{err: err, code: 2}
	}
	reader, err := githubadmission.NewRESTReader(*apiBase, os.Getenv("GITHUB_TOKEN"), nil)
	if err != nil {
		return admissionCommandError{err: err, code: 2}
	}
	collection, err := githubadmission.Collect(ctx, githubadmission.CollectRequest{Seed: seed, Event: event, Reader: reader})
	if err != nil {
		return admissionCommandError{err: err, code: 2}
	}
	packet, err := domain.DecodeAdmissionPacket(bytes.NewReader(collection.Artifact.CanonicalJSON()))
	if err != nil {
		return admissionCommandError{err: err, code: 2}
	}
	frozen, err := admissiongit.CollectFrozenRange(ctx, *repository, *base, *head, packet)
	if err != nil {
		return admissionCommandError{err: err, code: 2}
	}
	result, err := admission.Verify(admission.Input{Range: frozen, Packet: packet, Candidates: collection.Candidates})
	if err != nil {
		return admissionCommandError{err: err, code: 2}
	}
	if *packetPath != "" {
		if err := writeExclusivePacket(*packetPath, collection.Artifact.CanonicalJSON()); err != nil {
			return admissionCommandError{err: err, code: 2}
		}
	}
	canonical, exitCode, err := admission.CanonicalResult(result)
	if err != nil {
		return admissionCommandError{err: err, code: 2}
	}
	if _, err := stdout.Write(canonical); err != nil {
		return admissionCommandError{err: fmt.Errorf("write admission result: %w", err), code: 2}
	}
	if exitCode != admission.ExitAllowed {
		return admissionCommandError{err: errors.New("lineage admission denied"), code: exitCode}
	}
	return nil
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
