package pilotlead

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

var ErrLeadLogUnavailable = errors.New("lead log unavailable")

type PurgeResult struct {
	Purged          int
	Kept            int
	KeptUnknownDate int
}

type Store struct {
	Path string
	Now  func() time.Time
}

func NewStore(path string, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{Path: path, Now: now}
}

func (s *Store) PrepareAttempt(record LeadRecord, email string, attemptedAt time.Time) (bool, error) {
	return s.withLockedEntries(true, func(entries []leadEntry, file *os.File) (bool, error) {
		state := leadAttemptState(entries, normalizedEmail(email))
		if state.kind == leadStateDuplicate {
			return false, nil
		}

		attemptRecord := prepareAttemptRecord(record, attemptedAt)
		if state.kind == leadStateRetry {
			updated := mergeLeadRecord(attemptRecord, entries[state.index].record)
			updated = prepareAttemptRecord(updated, attemptedAt)
			line, err := encodeLeadLine(updated)
			if err != nil {
				return false, err
			}
			entries[state.index] = leadEntry{raw: line, record: updated}
			if err := rewriteEntries(file, entries); err != nil {
				return false, err
			}
			return true, nil
		}

		line, err := encodeLeadLine(attemptRecord)
		if err != nil {
			return false, err
		}
		if _, err := file.Seek(0, io.SeekEnd); err != nil {
			return false, err
		}
		if _, err := file.WriteString(line); err != nil {
			return false, err
		}
		if err := file.Sync(); err != nil {
			return false, err
		}
		return true, nil
	})
}

func (s *Store) MarkNotificationResult(email, status, transport, notificationError string, updatedAt time.Time) error {
	_, err := s.withLockedEntries(false, func(entries []leadEntry, file *os.File) (bool, error) {
		targetIndex := -1
		normalized := normalizedEmail(email)
		for index, entry := range entries {
			if entry.record == nil || leadRecordEmail(entry.record) != normalized {
				continue
			}
			if isMarkableAttemptStatus(leadRecordStatus(entry.record)) {
				targetIndex = index
			}
		}
		if targetIndex == -1 {
			return false, ErrLeadLogUnavailable
		}

		record := cloneLeadRecord(entries[targetIndex].record)
		record["notification_status"] = status
		record["notification_updated_at"] = formatUTC(updatedAt)
		if transport != "" {
			record["notification_transport"] = transport
		} else {
			delete(record, "notification_transport")
		}
		if notificationError != "" {
			record["notification_error"] = notificationError
		} else {
			delete(record, "notification_error")
		}

		line, err := encodeLeadLine(record)
		if err != nil {
			return false, err
		}
		entries[targetIndex] = leadEntry{raw: line, record: record}
		if err := rewriteEntries(file, entries); err != nil {
			return false, err
		}
		return true, nil
	})
	return err
}

func (s *Store) DigestRecords(date string, loc *time.Location) ([]LeadRecord, error) {
	if loc == nil {
		loc = time.UTC
	}

	entries, err := s.readEntriesReadOnly()
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	records := make([]LeadRecord, 0)
	seen := make(map[string]int)
	for _, entry := range entries {
		if entry.record == nil || recordLocalDate(entry.record, loc) != date {
			continue
		}
		email := leadRecordEmail(entry.record)
		if email == "" {
			records = append(records, entry.record)
			continue
		}
		if existing, ok := seen[email]; ok {
			records[existing] = entry.record
			continue
		}
		seen[email] = len(records)
		records = append(records, entry.record)
	}
	return records, nil
}

func (s *Store) PurgeBeforeLocalDate(cutoff string, loc *time.Location, dryRun bool) (PurgeResult, error) {
	if _, ok := parseLocalDate(cutoff); !ok {
		return PurgeResult{}, fmt.Errorf("%w: invalid_cutoff", ErrLeadLogUnavailable)
	}
	if loc == nil {
		loc = time.UTC
	}

	file, err := os.OpenFile(s.Path, os.O_RDWR, 0)
	if errors.Is(err, os.ErrNotExist) {
		return PurgeResult{}, nil
	}
	if err != nil {
		return PurgeResult{}, fmt.Errorf("%w: %v", ErrLeadLogUnavailable, err)
	}
	defer file.Close()

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return PurgeResult{}, fmt.Errorf("%w: %v", ErrLeadLogUnavailable, err)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

	entries, err := readEntries(file)
	if err != nil {
		return PurgeResult{}, err
	}

	result := PurgeResult{}
	keptEntries := make([]leadEntry, 0, len(entries))
	for _, entry := range entries {
		date, ok := retentionLocalDate(entry.record, loc)
		if !ok {
			result.Kept++
			result.KeptUnknownDate++
			keptEntries = append(keptEntries, entry)
			continue
		}
		if date < cutoff {
			result.Purged++
			continue
		}
		result.Kept++
		keptEntries = append(keptEntries, entry)
	}

	if dryRun {
		return result, nil
	}
	if err := rewriteEntries(file, keptEntries); err != nil {
		return PurgeResult{}, err
	}
	return result, nil
}

type leadStateKind string

const (
	leadStateNew       leadStateKind = "new"
	leadStateRetry     leadStateKind = "retry"
	leadStateDuplicate leadStateKind = "duplicate"
)

type leadState struct {
	kind  leadStateKind
	index int
}

func leadAttemptState(entries []leadEntry, normalized string) leadState {
	retryIndex := -1
	for index, entry := range entries {
		if entry.record == nil || leadRecordEmail(entry.record) != normalized {
			continue
		}

		status := leadRecordStatus(entry.record)
		if status == StatusNotified || status == "" {
			return leadState{kind: leadStateDuplicate}
		}
		if isRetryableAttemptStatus(status) {
			retryIndex = index
			continue
		}
		return leadState{kind: leadStateDuplicate}
	}
	if retryIndex >= 0 {
		return leadState{kind: leadStateRetry, index: retryIndex}
	}
	return leadState{kind: leadStateNew}
}

func isRetryableAttemptStatus(status string) bool {
	return status == StatusNotificationFailed
}

func isMarkableAttemptStatus(status string) bool {
	return status == StatusReceived || status == StatusPending || status == StatusNotificationFailed
}

func leadRecordEmail(record LeadRecord) string {
	return normalizedEmail(stringValue(record, "email"))
}

func leadRecordStatus(record LeadRecord) string {
	_, ok := record["notification_status"]
	if !ok {
		return ""
	}
	return stringValue(record, "notification_status")
}

func prepareAttemptRecord(record LeadRecord, attemptedAt time.Time) LeadRecord {
	prepared := cloneLeadRecord(record)
	attempted := formatUTC(attemptedAt)
	prepared["notification_status"] = StatusReceived
	prepared["notification_attempted_at"] = attempted
	prepared["notification_updated_at"] = attempted
	delete(prepared, "notification_transport")
	delete(prepared, "notification_error")
	return prepared
}

func mergeLeadRecord(base, existing LeadRecord) LeadRecord {
	merged := cloneLeadRecord(base)
	for key, value := range existing {
		merged[key] = value
	}
	return merged
}

func cloneLeadRecord(record LeadRecord) LeadRecord {
	clone := make(LeadRecord, len(record))
	for key, value := range record {
		clone[key] = value
	}
	return clone
}

func (s *Store) withLockedEntries(createDir bool, fn func([]leadEntry, *os.File) (bool, error)) (bool, error) {
	if createDir {
		if err := os.MkdirAll(filepath.Dir(s.Path), 0o770); err != nil {
			return false, fmt.Errorf("%w: %v", ErrLeadLogUnavailable, err)
		}
	}
	file, err := os.OpenFile(s.Path, os.O_RDWR|os.O_CREATE, 0o660)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrLeadLogUnavailable, err)
	}
	defer file.Close()

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return false, fmt.Errorf("%w: %v", ErrLeadLogUnavailable, err)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

	entries, err := readEntries(file)
	if err != nil {
		return false, err
	}
	return fn(entries, file)
}

func (s *Store) readEntriesReadOnly() ([]leadEntry, error) {
	file, err := os.Open(s.Path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_SH); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLeadLogUnavailable, err)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

	return readEntries(file)
}

func readEntries(file *os.File) ([]leadEntry, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLeadLogUnavailable, err)
	}

	entries := make([]leadEntry, 0)
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("%w: %v", ErrLeadLogUnavailable, err)
		}
		if line != "" {
			entry := leadEntry{raw: line}
			var record LeadRecord
			if json.Unmarshal([]byte(line), &record) == nil {
				entry.record = record
			}
			entries = append(entries, entry)
		}
		if errors.Is(err, io.EOF) {
			break
		}
	}
	return entries, nil
}

func rewriteEntries(file *os.File, entries []leadEntry) error {
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("%w: %v", ErrLeadLogUnavailable, err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("%w: %v", ErrLeadLogUnavailable, err)
	}
	writer := bufio.NewWriter(file)
	for _, entry := range entries {
		if _, err := writer.WriteString(entry.raw); err != nil {
			return fmt.Errorf("%w: %v", ErrLeadLogUnavailable, err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("%w: %v", ErrLeadLogUnavailable, err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("%w: %v", ErrLeadLogUnavailable, err)
	}
	return nil
}

func encodeLeadLine(record LeadRecord) (string, error) {
	encoded, err := json.Marshal(record)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrLeadLogUnavailable, err)
	}
	return string(encoded) + "\n", nil
}

func formatUTC(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
