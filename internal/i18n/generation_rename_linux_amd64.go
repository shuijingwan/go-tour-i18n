//go:build linux && amd64

package i18n

import (
	"runtime"
	"syscall"
	"unsafe"
)

const (
	generationRenameAt2Syscall = 316
	generationRenameNoReplace  = 1
	generationAtFDCWD          = ^uintptr(99)
)

func renameGenerationDirNoReplace(oldPath, newPath string) error {
	oldName, err := syscall.BytePtrFromString(oldPath)
	if err != nil {
		return err
	}
	newName, err := syscall.BytePtrFromString(newPath)
	if err != nil {
		return err
	}
	_, _, errno := syscall.Syscall6(
		generationRenameAt2Syscall,
		generationAtFDCWD, uintptr(unsafe.Pointer(oldName)),
		generationAtFDCWD, uintptr(unsafe.Pointer(newName)),
		generationRenameNoReplace, 0,
	)
	runtime.KeepAlive(oldName)
	runtime.KeepAlive(newName)
	if errno != 0 {
		return errno
	}
	return nil
}
