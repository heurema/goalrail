package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/heurema/goalrail/internal/admission"
	"github.com/heurema/goalrail/internal/admissiongit"
	"github.com/heurema/goalrail/internal/domain"
)

type admissionCommandError struct {
	err  error
	code int
}

func (err admissionCommandError) Error() string { return err.err.Error() }
func (err admissionCommandError) Unwrap() error { return err.err }
func (err admissionCommandError) ExitCode() int { return err.code }

func runVerifyLineage(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("verify-lineage", flag.ContinueOnError)
	set.SetOutput(stderr)
	repository := set.String("repo", "", "repository containing the frozen Git objects")
	base := set.String("base", "", "full immutable base commit ID")
	head := set.String("head", "", "full immutable head commit ID")
	packetPath := set.String("packet", "", "path to one admission-packet/v1 JSON file")
	jsonOutput := set.Bool("json", false, "emit canonical admission-result/v1 JSON")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return admissionCommandError{err: err, code: 2}
	}
	if set.NArg() != 0 || strings.TrimSpace(*repository) == "" || strings.TrimSpace(*base) == "" || strings.TrimSpace(*head) == "" || strings.TrimSpace(*packetPath) == "" || !*jsonOutput {
		return admissionCommandError{err: fmt.Errorf("verify-lineage requires --repo, --base, --head, --packet, and --json"), code: 2}
	}
	packetFile, err := os.Open(*packetPath)
	if err != nil {
		return admissionCommandError{err: fmt.Errorf("open admission packet: %w", err), code: 2}
	}
	packet, decodeErr := domain.DecodeAdmissionPacket(packetFile)
	closeErr := packetFile.Close()
	if decodeErr != nil {
		return admissionCommandError{err: decodeErr, code: 2}
	}
	if closeErr != nil {
		return admissionCommandError{err: fmt.Errorf("close admission packet: %w", closeErr), code: 2}
	}
	frozen, err := admissiongit.CollectFrozenRange(ctx, *repository, *base, *head, packet)
	if err != nil {
		return admissionCommandError{err: err, code: 2}
	}
	result, err := admission.Verify(admission.Input{Range: frozen, Packet: packet})
	if err != nil {
		return admissionCommandError{err: err, code: 2}
	}
	canonical, exitCode, err := admission.CanonicalResult(result)
	if err != nil {
		return admissionCommandError{err: err, code: 2}
	}
	if _, err := stdout.Write(canonical); err != nil {
		return admissionCommandError{err: fmt.Errorf("write admission result: %w", err), code: 2}
	}
	if exitCode != admission.ExitAllowed {
		return admissionCommandError{err: fmt.Errorf("lineage admission denied"), code: exitCode}
	}
	return nil
}
