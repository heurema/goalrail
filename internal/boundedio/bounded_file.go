// Package boundedio reads already resolved local artifacts under an explicit
// size bound without following a substituted symbolic link or blocking on a
// substituted pipe.
package boundedio

import (
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
	if limit <= 0 {
		return nil, fmt.Errorf("read %s: size bound must be positive", label)
	}
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", label, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", label, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", label)
	}

	raw, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", label, err)
	}
	if len(raw) > limit {
		return nil, fmt.Errorf("%s exceeds %d bytes", label, limit)
	}
	return raw, nil
}
