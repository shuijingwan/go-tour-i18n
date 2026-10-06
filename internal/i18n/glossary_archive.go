package i18n

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GlossaryArchiveReference identifies exact immutable bytes, never a Git revision.
type GlossaryArchiveReference struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

func glossaryArchivePath(locale, sha string) (string, error) {
	if err := ValidateLocaleName(locale); err != nil {
		return "", err
	}
	if !validSHA256(sha) {
		return "", fmt.Errorf("invalid glossary archive SHA-256")
	}
	return "data/glossary-history/" + locale + "/" + sha + ".yaml", nil
}

// compatibilityPath checks every path component, including parent symlinks.
func compatibilityPath(root, relative string, createParents bool) (string, error) {
	if err := validateGenerationInstallPath(relative); err != nil {
		return "", err
	}
	parts := strings.Split(relative, "/")
	current := root
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			if i == len(parts)-1 {
				return current, nil
			}
			if !createParents {
				return "", err
			}
			if err := os.Mkdir(current, 0755); err != nil && !os.IsExist(err) {
				return "", err
			}
			info, err = os.Lstat(current)
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 || (i < len(parts)-1 && !info.IsDir()) || (i == len(parts)-1 && !info.Mode().IsRegular()) {
			return "", fmt.Errorf("unsafe compatibility path %s", relative)
		}
	}
	return current, nil
}

func readCompatibilityFile(root, relative string) ([]byte, error) {
	path, err := compatibilityPath(root, relative, false)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

func writeCompatibilityImmutable(root, relative string, data []byte) error {
	path, err := compatibilityPath(root, relative, true)
	if err != nil {
		return err
	}
	old, err := os.ReadFile(path)
	if err == nil {
		if bytes.Equal(old, data) {
			return nil
		}
		return fmt.Errorf("immutable compatibility artifact differs: %s", relative)
	}
	if !os.IsNotExist(err) {
		return err
	}
	// The shared primitive links a complete temporary file without replacement.
	return writeNewGenerationFile(path, data)
}

func archiveGlossaryBytes(root, locale string, data []byte) (GlossaryArchiveReference, error) {
	sha := sum(data)
	path, err := glossaryArchivePath(locale, sha)
	if err != nil {
		return GlossaryArchiveReference{}, err
	}
	if err := writeCompatibilityImmutable(root, path, data); err != nil {
		return GlossaryArchiveReference{}, err
	}
	return GlossaryArchiveReference{Path: path, SHA256: sha}, nil
}

// ArchiveCurrentGlossary is the required prepare step before overwriting a
// reviewed glossary. It preserves the full-byte Review gate unchanged.
func ArchiveCurrentGlossary(root, locale string) (GlossaryArchiveReference, error) {
	if err := RequireCurrentGlossaryReview(root, locale); err != nil {
		return GlossaryArchiveReference{}, err
	}
	data, err := readCompatibilityFile(root, glossaryReviewGlossaryPath(locale))
	if err != nil {
		return GlossaryArchiveReference{}, err
	}
	return archiveGlossaryBytes(root, locale, data)
}

func readArchivedGlossary(root, locale, sha string) ([]byte, error) {
	path, err := glossaryArchivePath(locale, sha)
	if err != nil {
		return nil, err
	}
	data, err := readCompatibilityFile(root, path)
	if err != nil {
		return nil, err
	}
	if sum(data) != sha {
		return nil, fmt.Errorf("glossary archive hash mismatch: %s", path)
	}
	return data, nil
}
