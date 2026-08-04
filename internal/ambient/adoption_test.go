package ambient

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInitializeRecordsAnOptionalAdoption(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, time.August, 4, 12, 0, 0, 0, time.UTC)
	marker, created, err := InitializeWithAdoption(root, func() time.Time { return now }, &Adoption{
		ReplacedSchema: "intent-driven",
		RulesDigest:    "abc123",
		HadRules:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !created || marker.Adoption == nil {
		t.Fatalf("marker = %#v, created = %v", marker, created)
	}
	if marker.Adoption.AdoptedAt != now || marker.Adoption.ReplacedSchema != "intent-driven" {
		t.Fatalf("adoption = %#v", marker.Adoption)
	}

	read, err := ReadMarker(root)
	if err != nil {
		t.Fatal(err)
	}
	if read.Adoption == nil || read.Adoption.RulesDigest != "abc123" || !read.Adoption.HadRules {
		t.Fatalf("read adoption = %#v", read.Adoption)
	}
}

func TestLegacyMarkerRemainsValidAndCanGainAnAdoption(t *testing.T) {
	root := t.TempDir()
	markerPath := filepath.Join(root, filepath.FromSlash(MarkerPath))
	if err := os.MkdirAll(filepath.Dir(markerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := "{\n  \"schema\": \"goalrail.ambient-marker/v0\",\n  \"initialized_at\": \"2026-08-01T00:00:00Z\"\n}\n"
	if err := os.WriteFile(markerPath, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	read, err := ReadMarker(root)
	if err != nil || read.Adoption != nil {
		t.Fatalf("legacy marker = %#v, %v", read, err)
	}

	now := time.Date(2026, time.August, 4, 13, 0, 0, 0, time.UTC)
	updated, created, err := InitializeWithAdoption(root, func() time.Time { return now }, &Adoption{
		ReplacedSchema: "intent-driven",
		RulesDigest:    "digest",
	})
	if err != nil || created || updated.Adoption == nil {
		t.Fatalf("updated marker = %#v, created = %v, err = %v", updated, created, err)
	}
}
