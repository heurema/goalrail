package boundedio

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestReadRegularFileReadsWithinTheBound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact.md")
	if err := os.WriteFile(path, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	raw, err := ReadRegularFile(path, "artifact", 64)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "payload" {
		t.Fatalf("read %q, want %q", raw, "payload")
	}
}

func TestReadRegularFileRejectsOversizedArtifact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact.md")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 65)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadRegularFile(path, "artifact", 64); err == nil {
		t.Fatal("an oversized artifact was accepted")
	}
}

func TestReadRegularFileRejectsSymlink(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "target.md")
	if err := os.WriteFile(target, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "link.md")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadRegularFile(link, "artifact", 64); err == nil {
		t.Fatal("a substituted symbolic link was followed outside the verified boundary")
	}
}

func TestReadRegularFileDoesNotBlockOnFifo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact.md")
	if err := syscall.Mkfifo(path, 0o644); err != nil {
		t.Skipf("cannot create FIFO on this platform: %v", err)
	}
	// The defect this guards against is a hang, so the test must fail by
	// deadline rather than by waiting forever.
	done := make(chan error, 1)
	go func() {
		_, err := ReadRegularFile(path, "artifact", 64)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("a FIFO was accepted as a regular file")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("reading a FIFO blocked instead of failing with a bounded reason")
	}
}

func TestReadRegularFileRejectsDirectoryAndNonPositiveBound(t *testing.T) {
	directory := t.TempDir()
	if _, err := ReadRegularFile(directory, "artifact", 64); err == nil {
		t.Fatal("a directory was accepted as a regular file")
	}
	path := filepath.Join(directory, "artifact.md")
	if err := os.WriteFile(path, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadRegularFile(path, "artifact", 0); err == nil {
		t.Fatal("a non-positive size bound was accepted")
	}
}
