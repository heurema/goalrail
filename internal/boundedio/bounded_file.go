// Package boundedio reads already resolved local artifacts under an explicit
// size bound without following a substituted symbolic link or blocking on a
// substituted pipe.
package boundedio

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"syscall"
)

// ReadRegularFile opens an already resolved artifact and reads it under the
// supplied size bound.
//
// The regular-file check is made on the open descriptor rather than on the
// pathname, because a check-then-open sequence can be raced: replacing the
// checked path with a FIFO would still block the open, and replacing it with a
// symlink would let the read leave the verified boundary. Opening with
// O_NOFOLLOW rejects a substituted symlink, O_NONBLOCK keeps a substituted FIFO
// from blocking, and the descriptor that is then inspected is the one that is
// read.
func ReadRegularFile(path string, label string, limit int) ([]byte, error) {
	raw, _, err := ReadRegularFileWithInfo(path, label, limit)
	return raw, err
}

// ReadRegularFileWithInfo returns the bytes together with descriptor-backed
// file identity. Callers that protect a later mutation can compare this info
// with pathname snapshots and prove that the bytes came from the checked inode.
func ReadRegularFileWithInfo(path string, label string, limit int) ([]byte, os.FileInfo, error) {
	if limit <= 0 {
		return nil, nil, fmt.Errorf("read %s: size bound must be positive", label)
	}
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("open %s: %w", label, err)
	}
	defer file.Close()
	return ReadOpenFile(file, label, limit)
}

// ReadOpenFile applies the same discipline to a descriptor the caller already
// holds.
//
// Opening with O_NOFOLLOW refuses a symbolic link at the final pathname
// component and nothing else: a parent directory replaced by a link is followed
// before this package is reached. A caller that must stay inside a directory
// therefore opens through a confined root and hands the descriptor here, rather
// than composing a pathname this package would open on its behalf.
func ReadOpenFile(file *os.File, label string, limit int) ([]byte, os.FileInfo, error) {
	if limit <= 0 {
		return nil, nil, fmt.Errorf("read %s: size bound must be positive", label)
	}
	before, err := file.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("stat %s: %w", label, err)
	}
	if !before.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("%s is not a regular file", label)
	}

	raw, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", label, err)
	}
	if len(raw) > limit {
		return nil, nil, fmt.Errorf("%s exceeds %d bytes", label, limit)
	}
	after, err := file.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("stat %s after read: %w", label, err)
	}
	if !os.SameFile(before, after) || before.Mode() != after.Mode() ||
		before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return nil, nil, fmt.Errorf("%s changed while it was read", label)
	}
	return raw, after, nil
}

// DigestRegularFile reports an artifact's SHA-256 and verified identity under the same
// descriptor discipline, without holding its bytes in memory.
//
// An installed runtime is large enough that reading it whole to hash it would
// trade a bounded question for an unbounded allocation, so the content streams
// and only the digest is retained. Everything else is deliberately identical to
// ReadRegularFile: the regular-file check, the symlink and FIFO refusals, and
// the identity comparison all apply to the descriptor that produced the digest.
func DigestRegularFile(path string, label string, limit int64) (string, os.FileInfo, error) {
	if limit <= 0 {
		return "", nil, fmt.Errorf("digest %s: size bound must be positive", label)
	}
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", nil, fmt.Errorf("open %s: %w", label, err)
	}
	defer file.Close()
	return DigestOpenFile(file, label, limit)
}

// DigestOpenFile is the descriptor-taking form, for a caller that opened through
// a confined root so no path segment could leave it.
//
// It returns the identity it verified rather than only the size. A caller that
// also checks metadata — a mode, say — must check the snapshot this comparison
// covered: one taken before the read is stale by the time the digest exists, so
// a change inside that window would be accepted against the old metadata and
// hashed against the new bytes.
func DigestOpenFile(file *os.File, label string, limit int64) (string, os.FileInfo, error) {
	if limit <= 0 {
		return "", nil, fmt.Errorf("digest %s: size bound must be positive", label)
	}
	before, err := file.Stat()
	if err != nil {
		return "", nil, fmt.Errorf("stat %s: %w", label, err)
	}
	if !before.Mode().IsRegular() {
		return "", nil, fmt.Errorf("%s is not a regular file", label)
	}
	if before.Size() > limit {
		return "", nil, fmt.Errorf("%s exceeds %d bytes", label, limit)
	}

	sum := sha256.New()
	written, err := io.Copy(sum, io.LimitReader(file, limit+1))
	if err != nil {
		return "", nil, fmt.Errorf("read %s: %w", label, err)
	}
	if written > limit {
		return "", nil, fmt.Errorf("%s exceeds %d bytes", label, limit)
	}
	after, err := file.Stat()
	if err != nil {
		return "", nil, fmt.Errorf("stat %s after read: %w", label, err)
	}
	if !os.SameFile(before, after) || before.Mode() != after.Mode() ||
		before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return "", nil, fmt.Errorf("%s changed while it was read", label)
	}
	return hex.EncodeToString(sum.Sum(nil)), after, nil
}
