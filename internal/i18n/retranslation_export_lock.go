package i18n

import (
	"fmt"
	"os"
	"syscall"
)

// Lock the existing repository directory: no mutable ordinal pointer or
// permanent lock artifact is added to formal state. All batch exporters use
// the same kernel lock before scanning and publishing numeric ordinals.
func lockRetranslationBatchNamespace(root string) (func(), error) {
	f, err := os.Open(root)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("retranslation batch namespace has an active exporter")
	}
	return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); _ = f.Close() }, nil
}
