package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/heurema/goalrail/internal/boundedio"
	"github.com/heurema/goalrail/internal/domain"
	lineagestore "github.com/heurema/goalrail/internal/lineage"
)

func runLineage(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: gr lineage <begin|attach|inspect> [flags]")
	}
	switch args[0] {
	case "begin":
		return runLineageBegin(ctx, args[1:], stdout, stderr)
	case "attach":
		return runLineageAttach(ctx, args[1:], stdout, stderr)
	case "inspect":
		return runLineageInspect(args[1:], stdout, stderr)
	case "-h", "--help":
		_, err := fmt.Fprintln(stdout, "usage: gr lineage <begin|attach|inspect>\n  begin: --repo <path> --change <current-change-id> --actor-ref <scheme:identity>\n  attach: --repo <path> --event <event.json> [--replica <canonical.json> --replica-digest <sha256:...> --replica-schema <schema>]\n  inspect: --repo <path> --lookup <durable-id>")
		return err
	default:
		return fmt.Errorf("unknown gr lineage command %q", args[0])
	}
}

func runLineageInspect(args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("lineage inspect", flag.ContinueOnError)
	set.SetOutput(stderr)
	repository := set.String("repo", ".", "managed repository path")
	lookup := set.String("lookup", "", "bounded durable identifier to resolve")
	set.Usage = func() {
		fmt.Fprintln(stderr, "usage: gr lineage inspect --repo <path> --lookup <durable-id>")
		set.PrintDefaults()
	}
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if set.NArg() != 0 || strings.TrimSpace(*lookup) == "" {
		return fmt.Errorf("lineage inspect requires --lookup, with no positional arguments")
	}
	store, err := lineagestore.NewStore(*repository)
	if err != nil {
		return err
	}
	resolution, err := store.Resolve(*lookup)
	if err != nil {
		return err
	}
	return writeJSON(stdout, resolution)
}

func runLineageAttach(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("lineage attach", flag.ContinueOnError)
	set.SetOutput(stderr)
	repository := set.String("repo", ".", "managed repository path")
	eventPath := set.String("event", "", "canonical lineage-event JSON file")
	replicaPath := set.String("replica", "", "optional exact canonical evidence payload")
	replicaDigest := set.String("replica-digest", "", "expected SHA-256 digest of the replica")
	replicaSchema := set.String("replica-schema", "", "expected schema of the replica")
	set.Usage = func() {
		fmt.Fprintln(stderr, "usage: gr lineage attach --repo <path> --event <event.json> [--replica <canonical.json> --replica-digest <sha256:...> --replica-schema <schema>]")
		set.PrintDefaults()
	}
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if set.NArg() != 0 || strings.TrimSpace(*eventPath) == "" {
		return fmt.Errorf("lineage attach requires --event, with no positional arguments")
	}
	replicaFlags := 0
	for _, value := range []string{*replicaPath, *replicaDigest, *replicaSchema} {
		if strings.TrimSpace(value) != "" {
			replicaFlags++
		}
	}
	if replicaFlags != 0 && replicaFlags != 3 {
		return fmt.Errorf("--replica, --replica-digest, and --replica-schema must be supplied together")
	}
	eventRaw, err := boundedio.ReadRegularFile(*eventPath, "lineage event", domain.MaxLineageEventBytes)
	if err != nil {
		return fmt.Errorf("read lineage event: %w", err)
	}
	event, err := domain.DecodeLineageEvent(bytes.NewReader(eventRaw))
	if err != nil {
		return err
	}
	artifact, err := domain.FreezeLineageEvent(event)
	if err != nil {
		return err
	}
	if !bytes.Equal(eventRaw, artifact.CanonicalJSON()) {
		return fmt.Errorf("lineage event file must contain exact canonical JSON")
	}
	var replica *lineagestore.Replica
	if replicaFlags == 3 {
		raw, err := boundedio.ReadRegularFile(*replicaPath, "evidence replica", lineagestore.MaxReplicaBytes)
		if err != nil {
			return err
		}
		prepared, err := lineagestore.PrepareReplica(bytes.NewReader(raw), domain.SHA256Digest(*replicaDigest), *replicaSchema)
		if err != nil {
			return err
		}
		replica = &prepared
	}
	receipt, err := lineagestore.Attach(ctx, lineagestore.AttachOptions{
		Repository: *repository, Event: event, Replica: replica,
	})
	if err != nil {
		return err
	}
	return writeJSON(stdout, receipt)
}

func runLineageBegin(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("lineage begin", flag.ContinueOnError)
	set.SetOutput(stderr)
	repository := set.String("repo", ".", "managed repository path")
	changeID := set.String("change", "", "current Goalrail change ID")
	actorRef := set.String("actor-ref", "", "bounded actor reference")
	set.Usage = func() {
		fmt.Fprintln(stderr, "usage: gr lineage begin --repo <path> --change <current-change-id> --actor-ref <scheme:identity>")
		set.PrintDefaults()
	}
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if set.NArg() != 0 || *changeID == "" || *actorRef == "" {
		return fmt.Errorf("lineage begin requires --change and --actor-ref, with no positional arguments")
	}
	receipt, err := lineagestore.Begin(ctx, lineagestore.BeginOptions{
		Repository: *repository,
		ChangeID:   *changeID,
		ActorRef:   *actorRef,
	})
	if err != nil {
		return err
	}
	return writeJSON(stdout, receipt)
}
