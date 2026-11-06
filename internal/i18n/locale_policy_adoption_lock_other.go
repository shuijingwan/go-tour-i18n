//go:build !linux

package i18n

import (
	"fmt"
	"os"
)

func lockLocalePolicyAdoption(*os.File) (func(), error) {
	return nil, fmt.Errorf("atomic locale policy adoption publisher requires Linux directory locking")
}
