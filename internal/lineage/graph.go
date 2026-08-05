package lineage

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/heurema/goalrail/internal/boundedio"
	"github.com/heurema/goalrail/internal/domain"
)

type ResolutionStatus string

const (
	ResolutionFound     ResolutionStatus = "found"
	ResolutionMissing   ResolutionStatus = "missing"
	ResolutionAmbiguous ResolutionStatus = "ambiguous"
)

type RelationConflict struct {
	Relation domain.LineageRelation `json:"relation"`
	Digests  []domain.SHA256Digest  `json:"event_digests"`
}

type WorkUnitGraph struct {
	Unit        domain.WorkUnit                `json:"unit"`
	Events      []domain.LineageEvent          `json:"events"`
	Conflicts   []RelationConflict             `json:"conflicts"`
	Unavailable []domain.LineageRelation       `json:"unavailable"`
	Replicas    map[domain.SHA256Digest][]byte `json:"-"`
}

type Resolution struct {
	Query       string              `json:"query"`
	Status      ResolutionStatus    `json:"status"`
	WorkUnitIDs []domain.WorkUnitID `json:"work_unit_ids"`
	Graphs      []WorkUnitGraph     `json:"graphs"`
}

func (store *Store) Resolve(query string) (Resolution, error) {
	if strings.TrimSpace(query) == "" || len(query) > 256 {
		return Resolution{}, fmt.Errorf("lineage lookup must be one bounded durable identifier")
	}
	ids, err := store.ListWorkUnitIDs()
	if err != nil {
		return Resolution{}, err
	}
	result := Resolution{Query: query, Status: ResolutionMissing, WorkUnitIDs: []domain.WorkUnitID{}, Graphs: []WorkUnitGraph{}}
	for _, id := range ids {
		graph, err := store.Graph(id)
		if err != nil {
			return Resolution{}, err
		}
		if graphMatches(graph, query) {
			result.WorkUnitIDs = append(result.WorkUnitIDs, id)
			result.Graphs = append(result.Graphs, graph)
		}
	}
	switch len(result.Graphs) {
	case 0:
		result.Status = ResolutionMissing
	case 1:
		result.Status = ResolutionFound
	default:
		result.Status = ResolutionAmbiguous
	}
	return result, nil
}

func (store *Store) Graph(id domain.WorkUnitID) (WorkUnitGraph, error) {
	unit, _, err := store.LoadWorkUnit(id)
	if err != nil {
		return WorkUnitGraph{}, err
	}
	events, err := store.ListEvents(id)
	if err != nil {
		return WorkUnitGraph{}, err
	}
	graph := WorkUnitGraph{Unit: unit, Events: events, Replicas: make(map[domain.SHA256Digest][]byte)}
	claims := make(map[domain.LineageRelation]map[string][]domain.SHA256Digest)
	unavailable := make(map[domain.LineageRelation]struct{})
	available := make(map[domain.LineageRelation]struct{})
	for _, event := range events {
		if event.Cardinality == domain.RelationSingle {
			key := referenceSetKey(event.Targets)
			if claims[event.Relation] == nil {
				claims[event.Relation] = make(map[string][]domain.SHA256Digest)
			}
			claims[event.Relation][key] = append(claims[event.Relation][key], event.SemanticDigest)
		}
		for _, reference := range event.Targets {
			if reference.ArtifactKind == "provider_unavailable" {
				unavailable[event.Relation] = struct{}{}
			}
			if raw, found, err := store.readReplica(reference); err != nil {
				return WorkUnitGraph{}, err
			} else if found {
				graph.Replicas[reference.Digest] = raw
				available[event.Relation] = struct{}{}
			} else if isReplicaReference(reference) {
				unavailable[event.Relation] = struct{}{}
			} else if reference.ArtifactKind != "provider_unavailable" {
				available[event.Relation] = struct{}{}
			}
		}
	}
	for relation, relationClaims := range claims {
		if len(relationClaims) < 2 {
			continue
		}
		var digests []domain.SHA256Digest
		for _, eventDigests := range relationClaims {
			digests = append(digests, eventDigests...)
		}
		sort.Slice(digests, func(first, second int) bool { return digests[first] < digests[second] })
		graph.Conflicts = append(graph.Conflicts, RelationConflict{Relation: relation, Digests: digests})
	}
	sort.Slice(graph.Conflicts, func(first, second int) bool {
		return graph.Conflicts[first].Relation < graph.Conflicts[second].Relation
	})
	for relation := range unavailable {
		if _, resolved := available[relation]; !resolved {
			graph.Unavailable = append(graph.Unavailable, relation)
		}
	}
	sort.Slice(graph.Unavailable, func(first, second int) bool { return graph.Unavailable[first] < graph.Unavailable[second] })
	return graph, nil
}

func (store *Store) readReplica(reference domain.ContentAddressedEvidenceReference) ([]byte, bool, error) {
	if !isReplicaReference(reference) {
		return nil, false, nil
	}
	relative := filepath.ToSlash(filepath.Join(".goalrail", "evidence", "sha256", digestComponent(reference.Digest)))
	raw, err := boundedio.ReadRegularFile(filepath.Join(store.repositoryRoot, filepath.FromSlash(relative)), relative, MaxReplicaBytes)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if domain.DigestCanonicalJSON(raw) != reference.Digest {
		return nil, false, fmt.Errorf("evidence replica digest mismatch at %s", relative)
	}
	return raw, true, nil
}

func isReplicaReference(reference domain.ContentAddressedEvidenceReference) bool {
	want := repositorySourceRef(filepath.ToSlash(filepath.Join(".goalrail", "evidence", "sha256", digestComponent(reference.Digest))))
	return reference.SourceRef == want
}

func graphMatches(graph WorkUnitGraph, query string) bool {
	if query == string(graph.Unit.ID) || query == string(graph.Unit.ProjectID) ||
		query == string(graph.Unit.DeclarationDigest) || query == string(graph.Unit.PolicyDigest) {
		return true
	}
	if referenceMatches(graph.Unit.IntentRef, query) || referenceMatches(graph.Unit.ChangeRef, query) {
		return true
	}
	for _, event := range graph.Events {
		if query == string(event.SemanticDigest) {
			return true
		}
		for _, reference := range append(append([]domain.ContentAddressedEvidenceReference(nil), event.Sources...), event.Targets...) {
			if referenceMatches(reference, query) {
				return true
			}
		}
	}
	return false
}

func referenceMatches(reference domain.ContentAddressedEvidenceReference, query string) bool {
	return query == reference.Identity || query == string(reference.Digest) || query == reference.SourceRef
}

func referenceSetKey(references []domain.ContentAddressedEvidenceReference) string {
	var buffer bytes.Buffer
	for _, reference := range references {
		fmt.Fprintf(&buffer, "%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00", reference.ArtifactKind, reference.Identity,
			reference.Version, reference.Digest, reference.SourceRef, reference.AdapterID)
	}
	return buffer.String()
}
