package localrun

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/heurema/goalrail/internal/domain"
)

const (
	dogfoodAdmissionSchema       = "goalrail.dogfood-admission/v0"
	dogfoodAdmissionChange       = "activate-dogfood-run-v0"
	dogfoodAdmissionRecordName   = "activate-dogfood-run-v0.admission.json"
	dogfoodAdmissionConsumedName = "activate-dogfood-run-v0.admission.consumed.json"
	maxDogfoodAdmissionBytes     = 4 << 10
)

type dogfoodAdmissionRecord struct {
	Schema         string                `json:"schema"`
	Change         string                `json:"change"`
	WorkSpecDigest domain.WorkSpecDigest `json:"work_spec_digest"`
	BaseRevision   string                `json:"base_revision"`
}

type dogfoodAdmission struct {
	recordPath   string
	consumedPath string
}

func newDogfoodAdmission(store *Store) (*dogfoodAdmission, error) {
	if store == nil {
		return nil, errors.New("dogfood admission requires local-run state")
	}
	return &dogfoodAdmission{
		recordPath:   filepath.Join(store.Root(), dogfoodAdmissionRecordName),
		consumedPath: filepath.Join(store.Root(), dogfoodAdmissionConsumedName),
	}, nil
}

func (admission *dogfoodAdmission) consume(workSpec domain.FrozenWorkSpec) error {
	if admission == nil {
		return ErrActivationRequired
	}
	spec := workSpec.Spec()
	resolvedParent, err := filepath.EvalSymlinks(filepath.Dir(admission.recordPath))
	if err != nil {
		return ErrActivationRequired
	}
	resolvedRecord := filepath.Join(resolvedParent, filepath.Base(admission.recordPath))
	if pathWithin(spec.Repository.Root, resolvedRecord) {
		return ErrActivationRequired
	}
	parentInfo, err := os.Stat(resolvedParent)
	if err != nil ||
		!parentInfo.IsDir() ||
		parentInfo.Mode().Perm()&0o077 != 0 {
		return ErrActivationRequired
	}

	sourceInfo, raw, err := readDogfoodAdmissionFile(admission.recordPath)
	if err != nil {
		return ErrActivationRequired
	}
	record, err := decodeDogfoodAdmission(raw)
	if err != nil ||
		record.Schema != dogfoodAdmissionSchema ||
		record.Change != dogfoodAdmissionChange ||
		record.WorkSpecDigest != workSpec.Digest() ||
		record.BaseRevision != spec.Repository.BaseRevision {
		return ErrActivationRequired
	}

	// Publishing the fixed hard-link marker is the atomic consume point. Only
	// one process can create it, and it happens before any run ID or claim.
	if err := os.Link(admission.recordPath, admission.consumedPath); err != nil {
		return ErrActivationRequired
	}
	consumedInfo, consumedRaw, err := readDogfoodAdmissionFile(admission.consumedPath)
	if err != nil ||
		!os.SameFile(sourceInfo, consumedInfo) ||
		!bytes.Equal(raw, consumedRaw) {
		_ = os.Remove(admission.consumedPath)
		return ErrActivationRequired
	}
	return nil
}

func readDogfoodAdmissionFile(path string) (os.FileInfo, []byte, error) {
	info, err := os.Lstat(path)
	if err != nil ||
		!info.Mode().IsRegular() ||
		info.Mode().Perm()&0o077 != 0 {
		return nil, nil, ErrActivationRequired
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, ErrActivationRequired
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil || !os.SameFile(info, openedInfo) {
		return nil, nil, ErrActivationRequired
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxDogfoodAdmissionBytes+1))
	if err != nil || len(raw) > maxDogfoodAdmissionBytes {
		return nil, nil, ErrActivationRequired
	}
	return openedInfo, raw, nil
}

func decodeDogfoodAdmission(raw []byte) (dogfoodAdmissionRecord, error) {
	var record dogfoodAdmissionRecord
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return dogfoodAdmissionRecord{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return dogfoodAdmissionRecord{}, errors.New("multiple admission records")
		}
		return dogfoodAdmissionRecord{}, err
	}
	return record, nil
}
