package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/sourcestorage"
)

func CorpusAuthorityPaths(root string) ([]string, error) {
	paths := []string{i18n.GlossaryCorpusPath}
	entries, err := os.ReadDir(pathJoin(root, "data/glossary-corpus-history"))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".json") {
			return nil, fmt.Errorf("unsafe archived corpus entry")
		}
		paths = append(paths, "data/glossary-corpus-history/"+entry.Name())
	}
	sort.Strings(paths)
	return paths, nil
}

// CreateSourceStorageCompatibility is a deterministic storage-only operation.
// Its sole historical trust root is the still-present SHA-exact old ZIP. It
// creates no language judgment, glossary refresh, coverage or completion.
func CreateSourceStorageCompatibility(root string, apply bool) (*sourcestorage.Compatibility, string, error) {
	m, err := loadSourceManifest(root)
	if err != nil {
		return nil, "", err
	}
	oldRef := contentidentity.Reference{Path: SnapshotPath, SHA256: m.Legacy.SHA256}
	old, err := sourcestorage.ReadOldZIP(root, oldRef)
	if err != nil {
		return nil, "", err
	}
	entries, err := sourcestorage.ZIPEntries(old)
	if err != nil {
		return nil, "", err
	}
	if len(entries) != len(m.Legacy.Paths) {
		return nil, "", fmt.Errorf("old ZIP/registered subset membership differs")
	}
	for _, name := range m.Legacy.Paths {
		if _, ok := entries[name]; !ok {
			return nil, "", fmt.Errorf("old registered entry missing")
		}
	}
	manifest, err := readRegular(root, SourceManifestPath)
	if err != nil {
		return nil, "", err
	}
	p := &sourcestorage.Compatibility{Schema: sourcestorage.CompatibilitySchema, OldStorage: oldRef, Manifest: contentidentity.Reference{Path: SourceManifestPath, SHA256: digest(manifest)}, ManifestIdentity: m.Identity, Corpora: []sourcestorage.CorpusBinding{}, Entries: []sourcestorage.EquivalentEntry{}, Equality: "byte-exact"}
	paths, err := CorpusAuthorityPaths(root)
	if err != nil {
		return nil, "", err
	}
	required := map[string]bool{}
	for _, path := range paths {
		corpus, err := i18n.LoadGlossaryCorpusAt(root, path)
		if err != nil {
			return nil, "", err
		}
		bound := false
		for _, ref := range corpus.Sources {
			if ref.Path == SnapshotPath {
				if ref.SHA256 != oldRef.SHA256 {
					return nil, "", fmt.Errorf("corpus old ZIP reference differs")
				}
				bound = true
			}
		}
		if !bound {
			continue
		}
		names, err := i18n.CorpusStorageRequirements(corpus)
		if err != nil {
			return nil, "", err
		}
		b, err := readRegular(root, path)
		if err != nil {
			return nil, "", err
		}
		p.Corpora = append(p.Corpora, sourcestorage.CorpusBinding{Reference: contentidentity.Reference{Path: path, SHA256: digest(b)}, SemanticIdentity: corpus.Identity, RequiredEntries: names})
		for _, name := range names {
			required[name] = true
		}
	}
	files := map[string]FrozenSourceFile{}
	for _, f := range m.Files {
		files[f.Path] = f
	}
	names := []string{}
	for name := range required {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		prior, ok := entries[name]
		if !ok {
			return nil, "", fmt.Errorf("corpus source entry missing from old ZIP: %s", name)
		}
		f, ok := files[name]
		if !ok || f.Classification != "upstream-exact" || f.UpstreamPath != name {
			return nil, "", fmt.Errorf("ambiguous adopted corpus source: %s", name)
		}
		current, err := readAdoptedSource(root, f)
		if err != nil {
			return nil, "", err
		}
		if !bytes.Equal(prior, current) {
			return nil, "", fmt.Errorf("old/new source bytes differ: %s", name)
		}
		p.Entries = append(p.Entries, sourcestorage.EquivalentEntry{OldPath: name, OldSHA: digest(prior), NewPath: name, NewSHA: digest(current), Role: f.Role})
	}
	sourcestorage.SortBindings(p.Corpora)
	sourcestorage.SealCompatibility(p)
	if err := sourcestorage.VerifyCompatibility(root, *p, old); err != nil {
		return nil, "", err
	}
	path := sourcestorage.CompatibilityPath(*p)
	if apply {
		if err := saveImmutable(root, path, p); err != nil {
			return nil, "", err
		}
	}
	return p, path, nil
}
