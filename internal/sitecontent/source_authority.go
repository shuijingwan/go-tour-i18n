package sitecontent

import "github.com/shuijingwan/go-tour-i18n/internal/sourcestorage"

const SourceManifestPath = sourcestorage.ManifestPath
const sourceManifestSchema = sourcestorage.ManifestSchema
const sourceContractRoot = sourcestorage.ContractRoot
const sourceMigrationLock = sourcestorage.MigrationLock

type FrozenSourceFile = sourcestorage.FrozenSourceFile
type LegacySourceSubset = sourcestorage.LegacySourceSubset
type FrozenSourceManifest = sourcestorage.FrozenSourceManifest

func ReadSourceSnapshot(root string) ([]byte, error) { return sourcestorage.ReadSnapshot(root) }
func loadSourceManifest(root string) (*FrozenSourceManifest, error) {
	return sourcestorage.LoadManifest(root)
}
func validateSourceManifest(m FrozenSourceManifest) error { return sourcestorage.ValidateManifest(m) }
func readAdoptedSource(root string, f FrozenSourceFile) ([]byte, error) {
	return sourcestorage.ReadAdoptedSource(root, f)
}
func reconstructSourceSubset(root string, m *FrozenSourceManifest) ([]byte, error) {
	return sourcestorage.ReconstructSubset(root, m)
}
func CheckFrozenSources(root string) (*FrozenSourceManifest, error) {
	return sourcestorage.CheckSources(root)
}
func sealSourceManifest(m *FrozenSourceManifest) { sourcestorage.SealManifest(m) }
func gitObjectID(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func adoptedSourcePath(p string) string {
	if len(p) >= 9 && p[:9] == "_content/" {
		return p
	}
	return sourceContractRoot + p
}

// An adopted _content source occupies its original physical path. Only collapse
// that exact pair within one manifest entry; distinct paths remain observed.
func observeManifestSourcePaths(source string, observe func(string) error) error {
	if err := observe(source); err != nil {
		return err
	}
	if adopted := adoptedSourcePath(source); adopted != source {
		return observe(adopted)
	}
	return nil
}
