package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Optional on legacy manifests. New explicit package Snapshots bind the whole
// registered descriptor, not a name inferred from SnapshotID or a Unit subset.
type QualityCheckSnapshotPackage struct {
	ID                  string                     `json:"id"`
	AuthoritySHA256     string                     `json:"authority_sha256"`
	ScopeIdentitySHA256 string                     `json:"scope_identity_sha256"`
	UnitIDs             []string                   `json:"unit_ids"`
	AuthorityReferences []GlossaryArchiveReference `json:"authority_references"`
}

func snapshotPackageCatalog(c *Catalog, pkg string) (*Catalog, *QualityCheckSnapshotPackage, error) {
	if c == nil {
		return nil, nil, fmt.Errorf("Snapshot catalog required")
	}
	if pkg == "" {
		// Existing internal callers may explicitly select a legacy catalog scope.
		// The formal CLI forbids such hidden state without --content-package.
		return c, nil, nil
	}
	scoped := *c
	if err := scoped.UseTranslationPackage(pkg); err != nil {
		return nil, nil, err
	}
	for _, d := range scoped.TranslationPackages {
		if d.ID == pkg && !validSHA256(d.AuthoritySHA256) {
			return nil, nil, fmt.Errorf("Snapshot package requires registered source authority SHA256")
		}
	}
	if len(c.TranslationScope) > 0 && !reflect.DeepEqual(c.TranslationScope, scoped.TranslationScope) {
		return nil, nil, fmt.Errorf("Snapshot requires the complete selected registered package; partial/mixed scope rejected")
	}
	id, authority, scope, units, err := translationAuthorizationScope(&scoped)
	if err != nil || id != pkg {
		return nil, nil, fmt.Errorf("Snapshot registered package authority mismatch: %v", err)
	}
	binding := &QualityCheckSnapshotPackage{ID: id, AuthoritySHA256: authority, ScopeIdentitySHA256: scope, UnitIDs: []string{}, AuthorityReferences: []GlossaryArchiveReference{}}
	for _, u := range units {
		binding.UnitIDs = append(binding.UnitIDs, u.ID)
	}
	return &scoped, binding, nil
}

func snapshotPackageReferences(root, locale string, c *Catalog) ([]GlossaryArchiveReference, error) {
	units, _, _, err := localeWorkflowUnitList(c)
	if err != nil {
		return nil, err
	}
	byPath := map[string]string{}
	for _, d := range c.TranslationPackages {
		if d.Membership != nil {
			_, _, scope, _, e := translationAuthorizationScope(c)
			selected, _, e2 := snapshotPackageCatalog(c, d.ID)
			if e != nil || e2 != nil {
				continue
			}
			_, _, other, _, e := translationAuthorizationScope(selected)
			if e != nil || scope != other {
				continue
			}
			for _, ref := range d.Membership.References {
				b, e := readCompatibilityFile(root, ref.Path)
				if e != nil || sum(b) != ref.SHA256 {
					return nil, fmt.Errorf("Snapshot membership authority stale: %s", ref.Path)
				}
				byPath[ref.Path] = ref.SHA256
			}
		}
	}
	for _, u := range units {
		refs, err := translationGenerationContexts(root, locale, u.Adapter)
		if err != nil {
			return nil, err
		}
		refs = append(refs, GlossaryArchiveReference{Path: u.SourcePath, SHA256: u.SourceSHA256})
		for _, ref := range refs {
			b, err := readCompatibilityFile(root, ref.Path)
			if err != nil || !validSHA256(ref.SHA256) || sum(b) != ref.SHA256 {
				return nil, fmt.Errorf("Snapshot package/source authority stale: %s: %v", ref.Path, err)
			}
			if old, found := byPath[ref.Path]; found && old != ref.SHA256 {
				return nil, fmt.Errorf("conflicting Snapshot package authority: %s", ref.Path)
			}
			byPath[ref.Path] = ref.SHA256
		}
	}
	paths := make([]string, 0, len(byPath))
	for path := range byPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	refs := make([]GlossaryArchiveReference, 0, len(paths))
	for _, path := range paths {
		refs = append(refs, GlossaryArchiveReference{Path: path, SHA256: byPath[path]})
	}
	return refs, nil
}

func snapshotPackageCommitCheck(root, locale string, c *Catalog, b *QualityCheckSnapshotPackage) (func() error, error) {
	refs, err := snapshotPackageReferences(root, locale, c)
	if err != nil {
		return nil, err
	}
	b.AuthorityReferences = refs
	modes := map[string]os.FileMode{}
	for _, ref := range refs {
		st, err := os.Stat(filepath.Join(root, filepath.FromSlash(ref.Path)))
		if err != nil {
			return nil, err
		}
		modes[ref.Path] = st.Mode()
	}
	return func() error {
		_, got, err := snapshotPackageCatalog(c, b.ID)
		if err != nil {
			return err
		}
		got.AuthorityReferences, err = snapshotPackageReferences(root, locale, c)
		if err != nil || !reflect.DeepEqual(*got, *b) {
			return fmt.Errorf("Snapshot package/source authority changed before commit: %v", err)
		}
		for path, mode := range modes {
			st, err := os.Stat(filepath.Join(root, filepath.FromSlash(path)))
			if err != nil || st.Mode() != mode {
				return fmt.Errorf("Snapshot package dependency mode changed: %s", path)
			}
		}
		return nil
	}, nil
}

func checkSnapshotPackageBinding(root string, c *Catalog, s *QualityCheckSnapshotManifest, current bool) error {
	b := s.ContentPackage
	if b == nil {
		return nil // Immutable legacy Snapshots keep their original reader.
	}
	if b.ID == "" || !validSHA256(b.AuthoritySHA256) || !validSHA256(b.ScopeIdentitySHA256) || len(b.UnitIDs) != len(s.Units) {
		return fmt.Errorf("invalid Snapshot package binding")
	}
	for i, u := range s.Units {
		if u.UnitKind != UnitKindPage || b.UnitIDs[i] != u.UnitID {
			return fmt.Errorf("Snapshot package ordered membership mismatch")
		}
	}
	paths := []string{b.ID}
	for _, u := range s.Units {
		paths = append(paths, string(u.UnitKind)+":"+u.SourcePath)
	}
	if sum(mustJSON(paths)) != b.ScopeIdentitySHA256 {
		return fmt.Errorf("Snapshot package source scope mismatch")
	}
	for i, ref := range b.AuthorityReferences {
		if !validSHA256(ref.SHA256) || validateTransportBundlePath(ref.Path) != nil || (i > 0 && b.AuthorityReferences[i-1].Path >= ref.Path) {
			return fmt.Errorf("invalid Snapshot package authority reference set")
		}
	}
	for _, u := range s.Units {
		found := false
		for _, ref := range b.AuthorityReferences {
			found = found || (ref.Path == u.SourcePath && ref.SHA256 == u.SourceSHA256)
		}
		if !found {
			return fmt.Errorf("Snapshot package lacks exact source authority")
		}
	}
	if c == nil {
		return nil // Historical manifest/hash readers do not invent a registry.
	}
	if len(c.TranslationScope) == 0 {
		return fmt.Errorf("package Snapshot requires an explicit complete --content-package catalog")
	}
	_, want, err := snapshotPackageCatalog(c, b.ID)
	if err != nil {
		return err
	}
	if !current {
		want.AuthorityReferences = b.AuthorityReferences
	} else {
		want.AuthorityReferences, err = snapshotPackageReferences(root, s.Locale, c)
		if err != nil {
			return err
		}
	}
	if !reflect.DeepEqual(*want, *b) {
		return fmt.Errorf("Snapshot package authority/source binding mismatch")
	}
	for _, record := range s.Units {
		u, err := c.Unit(record.UnitID)
		if err != nil || u.SourcePath != record.SourcePath || u.SourceSHA256 != record.SourceSHA256 || u.Kind != record.UnitKind {
			return fmt.Errorf("Snapshot package Unit/source identity mismatch: %s", record.UnitID)
		}
	}
	return nil
}

func strictSnapshotPackageJSON(data []byte, s *QualityCheckSnapshotManifest) error {
	if s.ContentPackage == nil {
		return nil
	}
	var checked QualityCheckSnapshotManifest
	return contentidentity.StrictJSON(data, &checked)
}
