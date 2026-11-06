package contentidentity

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// ArtifactChanges accounts for both the path set and retained file bytes.
// It makes no claim about the completeness or scope of a caller's inventory.
type ArtifactChanges struct {
	PathsAdded   int `json:"paths_added"`
	PathsRemoved int `json:"paths_removed"`
	BytesChanged int `json:"bytes_changed"`
}

func (c ArtifactChanges) Unchanged() bool {
	return c.PathsAdded == 0 && c.PathsRemoved == 0 && c.BytesChanged == 0
}

// CompareArtifactDigests compares complete before/after inventories produced by
// the caller's existing safe file reader. Missing inventories or invalid hashes
// fail closed; a known empty scope must be an explicit non-nil empty map.
// This helper performs no I/O, scope discovery, mutation or caching.
func CompareArtifactDigests(before, after map[string]string) (ArtifactChanges, error) {
	if before == nil || after == nil {
		return ArtifactChanges{}, fmt.Errorf("complete before/after artifact inventories required")
	}
	for _, inventory := range []map[string]string{before, after} {
		for path, sha := range inventory {
			b, err := hex.DecodeString(sha)
			if path == "" || err != nil || len(b) != 32 || sha != strings.ToLower(sha) {
				return ArtifactChanges{}, fmt.Errorf("invalid artifact path/digest: %q", path)
			}
		}
	}
	var c ArtifactChanges
	for path, sha := range before {
		if current, exists := after[path]; !exists {
			c.PathsRemoved++
		} else if current != sha {
			c.BytesChanged++
		}
	}
	for path := range after {
		if _, exists := before[path]; !exists {
			c.PathsAdded++
		}
	}
	return c, nil
}
