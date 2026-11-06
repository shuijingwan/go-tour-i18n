//go:build linux

package i18n

import (
	"os"
	"syscall"
)

// Directory inode locking serializes adoption publishers without lock files.
func lockLocalePolicyAdoption(dir *os.File) (func(), error) {
	if err := syscall.Flock(int(dir.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return nil, err
	}
	return func() { _ = syscall.Flock(int(dir.Fd()), syscall.LOCK_UN) }, nil
}
