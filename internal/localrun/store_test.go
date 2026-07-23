package localrun

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreWritesOnceAndAcceptsOnlyIdenticalIdempotence(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WriteBytesOnce("prepared/example/work-spec.json", []byte("one"), true); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteBytesOnce("prepared/example/work-spec.json", []byte("one"), true); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteBytesOnce("prepared/example/work-spec.json", []byte("two"), true); !errors.Is(err, ErrStateAlreadyExists) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if err := store.WriteBytesOnce("prepared/example/claim.json", []byte("one"), false); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteBytesOnce("prepared/example/claim.json", []byte("one"), false); !errors.Is(err, ErrStateAlreadyExists) {
		t.Fatalf("expected duplicate rejection, got %v", err)
	}

	info, err := os.Stat(filepath.Join(store.Root(), "prepared", "example", "work-spec.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("state permissions = %o, want 600", info.Mode().Perm())
	}
}

func TestStoreRejectsEscapingPaths(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WriteBytesOnce("../outside", []byte("x"), false); err == nil {
		t.Fatal("expected path escape rejection")
	}
}
