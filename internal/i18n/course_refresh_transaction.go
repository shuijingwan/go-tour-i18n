package i18n

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// ApplyCourseMetadataRefresh stages old/new bytes and receipt before mutation.
// Publication is the commit point. A normal publication failure restores the
// old file; an interrupted/failed rollback retains staging and fails closed.
func ApplyCourseMetadataRefresh(root string, cat *Catalog, options CourseMetadataRefreshOptions) (*CourseSEOTransition, []string, error) {
	return applyCourseMetadataRefresh(root, cat, options, os.Link)
}

func applyCourseMetadataRefresh(root string, cat *Catalog, options CourseMetadataRefreshOptions, publish func(string, string) error) (receipt *CourseSEOTransition, stale []string, resultErr error) {
	if err := CheckCourseRefreshTransactionState(root, options.Locale); err != nil {
		return nil, nil, err
	}
	path := courseMetadataRelativePath(options.Locale)
	old, err := readCompatibilityFile(root, path)
	if err != nil {
		return nil, nil, err
	}
	current, stale, err := RefreshCourseMetadata(root, cat, options)
	if err != nil {
		return nil, nil, err
	}
	if len(stale) == 0 {
		return nil, stale, nil
	}
	e, err := buildCourseTransition(root, options.Locale, cat, old, current, false)
	if err != nil {
		return nil, nil, err
	}
	// Archives are immutable and unreferenced until the receipt commits. Save
	// them before any overwrite, even if a later operation fails.
	if err := saveCourseHistory(root, e.Old, old); err != nil {
		return nil, nil, err
	}
	if err := saveCourseHistory(root, e.New, current); err != nil {
		return nil, nil, err
	}
	parent, err := compatibilityPath(root, "locales/"+options.Locale+"/.course-refresh-staging-transaction", true)
	if err != nil {
		return nil, nil, err
	}
	if err := os.Mkdir(parent, 0700); err != nil {
		return nil, nil, err
	}
	cleanup := true
	defer func() {
		if recoverValue := recover(); recoverValue != nil {
			cleanup = false
			panic(recoverValue)
		}
		if cleanup {
			if err := os.RemoveAll(parent); err != nil {
				resultErr = fmt.Errorf("refresh staging cleanup failed: %w", err)
			}
		}
	}()
	backup := filepath.Join(parent, "old.json")
	staged := filepath.Join(parent, "new.json")
	stagedReceipt := filepath.Join(parent, "receipt.json")
	for _, item := range []struct {
		path  string
		bytes []byte
	}{{backup, old}, {staged, current}, {stagedReceipt, append(mustJSON(e), '\n')}} {
		if err := writeNewGenerationFile(item.path, item.bytes); err != nil {
			return nil, nil, err
		}
	}
	if err := checkCourseTransitionInputs(root, e); err != nil {
		return nil, nil, err
	}
	pre, err := readCompatibilityFile(root, path)
	if err != nil || !bytes.Equal(pre, old) {
		return nil, nil, fmt.Errorf("Course SEO refresh precommit base changed")
	}
	destination, err := compatibilityPath(root, path, false)
	if err != nil {
		return nil, nil, err
	}
	receiptPath, err := compatibilityPath(root, courseTransitionPath(e), true)
	if err != nil {
		return nil, nil, err
	}
	if err := os.Rename(staged, destination); err != nil {
		return nil, nil, err
	}
	installed, err := readCompatibilityFile(root, path)
	if err == nil && !bytes.Equal(installed, current) {
		err = fmt.Errorf("Course SEO installed output mismatch")
	}
	if err == nil {
		err = checkCourseTransitionInputs(root, e)
	}
	if err == nil {
		sealed, readErr := os.ReadFile(stagedReceipt)
		if readErr != nil || !bytes.Equal(sealed, append(mustJSON(e), '\n')) {
			err = fmt.Errorf("staged Course SEO receipt changed")
		}
	}
	if err == nil {
		err = publish(stagedReceipt, receiptPath)
	}
	if err != nil {
		if rollbackErr := os.Rename(backup, destination); rollbackErr != nil {
			cleanup = false
			return nil, nil, fmt.Errorf("mutation=unknown; retained %s; publication=%v rollback=%v", parent, err, rollbackErr)
		}
		restored, readErr := readCompatibilityFile(root, path)
		if readErr != nil || !bytes.Equal(restored, old) {
			cleanup = false
			return nil, nil, fmt.Errorf("mutation=unknown; rollback verification failed; retained %s", parent)
		}
		return nil, nil, fmt.Errorf("mutation=none; Course SEO refresh rolled back: %w", err)
	}
	return e, stale, nil
}
