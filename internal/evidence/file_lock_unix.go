//go:build darwin || linux

package evidence

import (
	"fmt"
	"os"
	"syscall"
)

func lockEvidenceFile(file *os.File, exclusive bool) error {
	mode := syscall.LOCK_SH
	if exclusive {
		mode = syscall.LOCK_EX
	}
	if err := syscall.Flock(int(file.Fd()), mode); err != nil {
		return fmt.Errorf("lock evidence record: %w", err)
	}
	return nil
}

func unlockEvidenceFile(file *os.File) error {
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); err != nil {
		return fmt.Errorf("unlock evidence record: %w", err)
	}
	return nil
}
