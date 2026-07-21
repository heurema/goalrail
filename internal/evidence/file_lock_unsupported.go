//go:build !darwin && !linux

package evidence

import "os"

func lockEvidenceFile(_ *os.File, _ bool) error {
	return ErrInterprocessLockUnsupported
}

func unlockEvidenceFile(_ *os.File) error {
	return nil
}
