package lineage

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/heurema/goalrail/internal/boundedio"
	"github.com/heurema/goalrail/internal/domain"
)

var ErrConflict = errors.New("lineage artifact conflict")

type Store struct {
	repositoryRoot string
}

type BeginReceipt struct {
	WorkUnitID   domain.WorkUnitID   `json:"work_unit_id"`
	AnchorRef    string              `json:"anchor_ref"`
	AnchorDigest domain.SHA256Digest `json:"anchor_digest"`
	EventRefs    []string            `json:"event_refs"`
	Created      bool                `json:"created"`
}

type AttachReceipt struct {
	WorkUnitID  domain.WorkUnitID   `json:"work_unit_id"`
	EventRef    string              `json:"event_ref"`
	EventDigest domain.SHA256Digest `json:"event_digest"`
	ReplicaRef  string              `json:"replica_ref,omitempty"`
	Created     bool                `json:"created"`
}

func NewStore(repositoryRoot string) (*Store, error) {
	root, err := filepath.Abs(repositoryRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve lineage repository root: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve lineage repository root: %w", err)
	}
	return &Store{repositoryRoot: filepath.Clean(root)}, nil
}

func (store *Store) Begin(unit domain.WorkUnit, events []domain.LineageEvent) (BeginReceipt, error) {
	if store == nil || store.repositoryRoot == "" {
		return BeginReceipt{}, fmt.Errorf("lineage store is unavailable")
	}
	unitArtifact, err := domain.FreezeWorkUnit(unit)
	if err != nil {
		return BeginReceipt{}, err
	}
	anchorRef := filepath.ToSlash(filepath.Join(".goalrail", "work-units", string(unit.ID), "unit.json"))

	type frozenEvent struct {
		reference string
		bytes     []byte
	}
	frozenEvents := make([]frozenEvent, 0, len(events))
	for _, event := range events {
		if event.WorkUnitID != unit.ID {
			return BeginReceipt{}, fmt.Errorf("lineage event names work unit %s, want %s", event.WorkUnitID, unit.ID)
		}
		artifact, freezeErr := domain.FreezeLineageEvent(event)
		if freezeErr != nil {
			return BeginReceipt{}, freezeErr
		}
		reference := filepath.ToSlash(filepath.Join(
			".goalrail", "work-units", string(unit.ID), "events", digestComponent(event.SemanticDigest)+".json",
		))
		frozenEvents = append(frozenEvents, frozenEvent{reference: reference, bytes: artifact.CanonicalJSON()})
	}

	paths := make([]string, 0, len(frozenEvents)+1)
	contents := make([][]byte, 0, len(frozenEvents)+1)
	paths = append(paths, anchorRef)
	contents = append(contents, unitArtifact.CanonicalJSON())
	for _, event := range frozenEvents {
		paths = append(paths, event.reference)
		contents = append(contents, event.bytes)
	}
	for index := range paths {
		if err := store.preflight(paths[index], contents[index]); err != nil {
			return BeginReceipt{}, err
		}
	}

	created := false
	for index := range paths {
		wrote, err := store.writeOnce(paths[index], contents[index])
		if err != nil {
			return BeginReceipt{}, err
		}
		created = created || wrote
	}
	eventRefs := make([]string, 0, len(frozenEvents))
	for _, event := range frozenEvents {
		eventRefs = append(eventRefs, event.reference)
	}
	return BeginReceipt{
		WorkUnitID:   unit.ID,
		AnchorRef:    anchorRef,
		AnchorDigest: unitArtifact.Digest(),
		EventRefs:    eventRefs,
		Created:      created,
	}, nil
}

func (store *Store) LoadWorkUnit(id domain.WorkUnitID) (domain.WorkUnit, []byte, error) {
	if !domain.IsCanonicalID(string(id)) || !strings.HasPrefix(string(id), "wu_") {
		return domain.WorkUnit{}, nil, fmt.Errorf("work-unit ID must be canonical")
	}
	reference := filepath.ToSlash(filepath.Join(".goalrail", "work-units", string(id), "unit.json"))
	raw, err := boundedio.ReadRegularFile(
		filepath.Join(store.repositoryRoot, filepath.FromSlash(reference)), reference, domain.MaxWorkUnitBytes,
	)
	if err != nil {
		return domain.WorkUnit{}, nil, err
	}
	unit, err := domain.DecodeWorkUnit(bytes.NewReader(raw))
	if err != nil {
		return domain.WorkUnit{}, nil, err
	}
	artifact, err := domain.FreezeWorkUnit(unit)
	if err != nil {
		return domain.WorkUnit{}, nil, err
	}
	if !bytes.Equal(raw, artifact.CanonicalJSON()) {
		return domain.WorkUnit{}, nil, fmt.Errorf("work-unit anchor is not canonical JSON")
	}
	return unit, raw, nil
}

func (store *Store) ListEvents(id domain.WorkUnitID) ([]domain.LineageEvent, error) {
	if _, _, err := store.LoadWorkUnit(id); err != nil {
		return nil, err
	}
	directory := filepath.Join(store.repositoryRoot, ".goalrail", "work-units", string(id), "events")
	entries, err := os.ReadDir(directory)
	if errors.Is(err, fs.ErrNotExist) {
		return []domain.LineageEvent{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read lineage events: %w", err)
	}
	if len(entries) > domain.MaxLineageRelations*domain.MaxLineageReferences {
		return nil, fmt.Errorf("lineage event count exceeds the bounded store limit")
	}
	events := make([]domain.LineageEvent, 0, len(entries))
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			return nil, fmt.Errorf("lineage event directory contains unsupported entry %s", entry.Name())
		}
		reference := filepath.ToSlash(filepath.Join(".goalrail", "work-units", string(id), "events", entry.Name()))
		raw, err := boundedio.ReadRegularFile(
			filepath.Join(store.repositoryRoot, filepath.FromSlash(reference)), reference, domain.MaxLineageEventBytes,
		)
		if err != nil {
			return nil, err
		}
		event, err := domain.DecodeLineageEvent(bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		artifact, err := domain.FreezeLineageEvent(event)
		if err != nil || !bytes.Equal(raw, artifact.CanonicalJSON()) {
			return nil, fmt.Errorf("lineage event %s is not canonical", entry.Name())
		}
		if digestComponent(event.SemanticDigest)+".json" != entry.Name() {
			return nil, fmt.Errorf("lineage event filename does not match semantic digest: %s", entry.Name())
		}
		events = append(events, event)
	}
	sort.Slice(events, func(first, second int) bool {
		return events[first].SemanticDigest < events[second].SemanticDigest
	})
	return events, nil
}

func (store *Store) ListWorkUnitIDs() ([]domain.WorkUnitID, error) {
	directory := filepath.Join(store.repositoryRoot, ".goalrail", "work-units")
	entries, err := os.ReadDir(directory)
	if errors.Is(err, fs.ErrNotExist) {
		return []domain.WorkUnitID{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read work-unit store: %w", err)
	}
	if len(entries) > 4096 {
		return nil, fmt.Errorf("work-unit count exceeds the bounded store limit")
	}
	ids := make([]domain.WorkUnitID, 0, len(entries))
	for _, entry := range entries {
		id := domain.WorkUnitID(entry.Name())
		if entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() ||
			!domain.IsCanonicalID(string(id)) || !strings.HasPrefix(string(id), "wu_") {
			return nil, fmt.Errorf("work-unit store contains unsupported entry %s", entry.Name())
		}
		ids = append(ids, id)
	}
	sort.Slice(ids, func(first, second int) bool { return ids[first] < ids[second] })
	return ids, nil
}

func (store *Store) Attach(event domain.LineageEvent, replica *Replica) (AttachReceipt, error) {
	unit, _, err := store.LoadWorkUnit(event.WorkUnitID)
	if err != nil {
		return AttachReceipt{}, err
	}
	if err := validateEventCardinality(unit, event); err != nil {
		return AttachReceipt{}, err
	}
	artifact, err := domain.FreezeLineageEvent(event)
	if err != nil {
		return AttachReceipt{}, err
	}
	event = mustDecodeEvent(artifact.CanonicalJSON())
	eventRef := filepath.ToSlash(filepath.Join(
		".goalrail", "work-units", string(event.WorkUnitID), "events", digestComponent(event.SemanticDigest)+".json",
	))
	if replica != nil {
		validated, err := PrepareReplica(bytes.NewReader(replica.Canonical), replica.Digest, replica.Schema)
		if err != nil {
			return AttachReceipt{}, err
		}
		if replica.Reference != "" && replica.Reference != validated.Reference {
			return AttachReceipt{}, fmt.Errorf("replica reference does not match its content digest")
		}
		replica = &validated
		if !eventReferencesDigest(event, replica.Digest) {
			return AttachReceipt{}, fmt.Errorf("replica digest is not named by the lineage event")
		}
		if err := store.preflight(replica.Reference, replica.Canonical); err != nil {
			return AttachReceipt{}, err
		}
	}
	if err := store.preflight(eventRef, artifact.CanonicalJSON()); err != nil {
		return AttachReceipt{}, err
	}
	created := false
	replicaRef := ""
	if replica != nil {
		wrote, err := store.writeOnce(replica.Reference, replica.Canonical)
		if err != nil {
			return AttachReceipt{}, err
		}
		created = wrote
		replicaRef = replica.Reference
	}
	wrote, err := store.writeOnce(eventRef, artifact.CanonicalJSON())
	if err != nil {
		return AttachReceipt{}, err
	}
	return AttachReceipt{
		WorkUnitID: event.WorkUnitID, EventRef: eventRef, EventDigest: event.SemanticDigest,
		ReplicaRef: replicaRef, Created: created || wrote,
	}, nil
}

func validateEventCardinality(unit domain.WorkUnit, event domain.LineageEvent) error {
	for _, requirement := range unit.RequiredRelations {
		if requirement.Relation == event.Relation && requirement.Cardinality != event.Cardinality {
			return fmt.Errorf("lineage relation %s requires %s cardinality", event.Relation, requirement.Cardinality)
		}
	}
	return nil
}

func mustDecodeEvent(raw []byte) domain.LineageEvent {
	event, err := domain.DecodeLineageEvent(bytes.NewReader(raw))
	if err != nil {
		panic(err)
	}
	return event
}

func eventReferencesDigest(event domain.LineageEvent, digest domain.SHA256Digest) bool {
	for _, reference := range append(append([]domain.ContentAddressedEvidenceReference(nil), event.Sources...), event.Targets...) {
		if reference.Digest == digest {
			return true
		}
	}
	return false
}

func (store *Store) preflight(relative string, desired []byte) error {
	absolute := filepath.Join(store.repositoryRoot, filepath.FromSlash(relative))
	raw, err := boundedio.ReadRegularFile(absolute, relative, max(len(desired), domain.MaxLineageEventBytes))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !bytes.Equal(raw, desired) {
		return fmt.Errorf("%w: %s already contains different canonical bytes", ErrConflict, relative)
	}
	return nil
}

func (store *Store) writeOnce(relative string, desired []byte) (bool, error) {
	absolute := filepath.Join(store.repositoryRoot, filepath.FromSlash(relative))
	if err := ensureDirectoryChain(store.repositoryRoot, filepath.Dir(absolute)); err != nil {
		return false, err
	}
	file, err := os.OpenFile(absolute, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		raw, readErr := boundedio.ReadRegularFile(absolute, relative, max(len(desired), domain.MaxLineageEventBytes))
		if readErr != nil {
			return false, readErr
		}
		if !bytes.Equal(raw, desired) {
			return false, fmt.Errorf("%w: %s was concurrently written with different bytes", ErrConflict, relative)
		}
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("create %s: %w", relative, err)
	}
	writeErr := func() error {
		if _, err := file.Write(desired); err != nil {
			return err
		}
		return file.Sync()
	}()
	closeErr := file.Close()
	if writeErr != nil {
		_ = os.Remove(absolute)
		return false, fmt.Errorf("write %s: %w", relative, writeErr)
	}
	if closeErr != nil {
		_ = os.Remove(absolute)
		return false, fmt.Errorf("close %s: %w", relative, closeErr)
	}
	return true, nil
}

func ensureDirectoryChain(root, target string) error {
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("lineage path escapes repository root")
	}
	current := root
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, fs.ErrNotExist) {
			if err := os.Mkdir(current, 0o755); err != nil && !errors.Is(err, fs.ErrExist) {
				return fmt.Errorf("create lineage directory: %w", err)
			}
			info, err = os.Lstat(current)
		}
		if err != nil {
			return fmt.Errorf("inspect lineage directory: %w", err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("lineage directory %s is not a real directory", current)
		}
	}
	return nil
}

func digestComponent(digest domain.SHA256Digest) string {
	return strings.TrimPrefix(string(digest), "sha256:")
}
