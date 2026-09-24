//go:build !linux || !amd64

package i18n

import "errors"

func renameGenerationDirNoReplace(oldPath, newPath string) error {
	return errors.New("atomic no-replace generation directory install requires Linux amd64 renameat2 support")
}
