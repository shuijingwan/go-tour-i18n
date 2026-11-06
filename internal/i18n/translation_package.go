package i18n

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// TranslationPackageDescriptor registers inventory, not lifecycle policy.
// Attempts, task semantics, predecessor selection and QC stay in i18n.
// Units use their frozen inventory indexes; Snapshot indexes are ordinal in
// that complete ordered scope and immutable within each Snapshot.
type TranslationPackageDescriptor struct {
	ID string
	// Frozen Content Scope package identity; projection never invents authority.
	AuthoritySHA256 string
	Units           []AdaptedTranslationUnit
	// Filled only by the source provider's verified membership-successor reader.
	// Inventory metadata, not new serialized lifecycle/acceptance authority.
	Membership *TranslationPackageMembership `json:"-"`
}

type TranslationPackageMembership struct {
	Predecessors []TranslationPackagePredecessor
	References   []GlossaryArchiveReference
}
type TranslationPackagePredecessor struct{ ID, AuthoritySHA256 string }

// HistoricalTranslationPackageCatalog explicitly projects a frozen descriptor.
// It cannot register a new lifecycle scope or choose candidates by path.
func HistoricalTranslationPackageCatalog(c *Catalog, id, authority string) (*Catalog, error) {
	for _, d := range append(append([]TranslationPackageDescriptor{}, c.TranslationPackages...), c.HistoricalTranslationPackages...) {
		if d.ID != id || d.AuthoritySHA256 != authority {
			continue
		}
		out := *c
		out.TranslationPackages = []TranslationPackageDescriptor{d}
		out.TranslationScope = nil
		out.Adapted = append([]AdaptedTranslationUnit{}, c.Adapted...)
		for _, u := range d.Units {
			found := false
			for i := range out.Adapted {
				if out.Adapted[i].Unit.ID == u.Unit.ID {
					out.Adapted[i] = u
					found = true
				}
			}
			if !found {
				out.Adapted = append(out.Adapted, u)
			}
		}
		if err := out.UseTranslationPackage(id); err != nil {
			return nil, err
		}
		return &out, nil
	}
	return nil, fmt.Errorf("unregistered historical package authority %s", id)
}

func (c *Catalog) RegisterTranslationPackage(d TranslationPackageDescriptor) error {
	if d.ID == "" || strings.ContainsAny(d.ID, "/\\\x00") || len(d.Units) == 0 {
		return fmt.Errorf("invalid translation package descriptor")
	}
	if d.AuthoritySHA256 != "" {
		b, err := hex.DecodeString(d.AuthoritySHA256)
		if err != nil || len(b) != 32 || strings.ToLower(d.AuthoritySHA256) != d.AuthoritySHA256 {
			return fmt.Errorf("invalid package authority SHA")
		}
	}
	for _, p := range c.TranslationPackages {
		if p.ID == d.ID {
			return fmt.Errorf("duplicate translation package %s", d.ID)
		}
	}
	units := append([]AdaptedTranslationUnit(nil), d.Units...)
	sort.Slice(units, func(i, j int) bool { return units[i].StableIndex < units[j].StableIndex })
	seen := map[string]bool{}
	previous := 0
	for i := range units {
		u := &units[i]
		if u.Unit.ID == "" || seen[u.Unit.ID] || u.StableIndex <= previous || u.Adapter == nil ||
			u.Unit.Kind != UnitKindPage || sum(u.Unit.Source) != u.Unit.SourceSHA256 || u.Route == "" {
			return fmt.Errorf("invalid package Unit %s", u.Unit.ID)
		}
		if err := validateTransportBundlePath(u.Unit.SourcePath); err != nil {
			return err
		}
		if _, err := c.Unit(u.Unit.ID); err == nil {
			return fmt.Errorf("package Unit shadows registered identity %s", u.Unit.ID)
		}
		seen[u.Unit.ID], previous = true, u.StableIndex
		u.Unit.Source = bytes.Clone(u.Unit.Source)
	}
	d.Units = units
	c.TranslationPackages = append(c.TranslationPackages, d)
	c.Adapted = append(c.Adapted, units...)
	return nil
}

func (c *Catalog) UseTranslationPackage(id string) error {
	for _, p := range c.TranslationPackages {
		if p.ID != id {
			continue
		}
		ids := []string{}
		for _, u := range p.Units {
			ids = append(ids, u.Unit.ID)
		}
		return c.UseTranslationUnitScope(ids)
	}
	return fmt.Errorf("unregistered translation package %s", id)
}

// FinalizedTranslationPackageCandidates is the shared post-QC authorization
// boundary. Projection/activation may install these bytes in package-specific
// paths, but cannot reinterpret A, pick a latest replacement or accept a subset.
// It writes no state. Callers register and select the complete descriptor scope.
func FinalizedTranslationPackageCandidates(root string, c *Catalog, locale, snapshotID string) (map[string][]byte, error) {
	if c == nil || len(c.TranslationScope) == 0 {
		return nil, fmt.Errorf("complete translation package scope required")
	}
	complete := false
	for _, p := range c.TranslationPackages {
		if len(p.Units) != len(c.TranslationScope) {
			continue
		}
		match := true
		for i, u := range p.Units {
			if u.Unit.ID != c.TranslationScope[i] {
				match = false
			}
		}
		complete = complete || match
	}
	if !complete {
		return nil, fmt.Errorf("package authorization requires an exact registered descriptor scope")
	}
	a, err := RequireCurrentTranslationAuthorization(root, c, locale, snapshotID)
	if err != nil {
		return nil, err
	}
	if err := finalizationMatchesLatestPromotionCandidates(root, c, locale, snapshotID); err != nil {
		return nil, err
	}
	out := map[string][]byte{}
	for _, record := range a.Historical.Finalization.Units {
		u, err := c.Unit(record.UnitID)
		if err != nil {
			return nil, err
		}
		b, err := readSnapshotReferencedFile(root, record.Snapshot.CandidatePath)
		if err != nil || sum(b) != record.Snapshot.CandidateSHA256 {
			return nil, fmt.Errorf("package candidate identity mismatch: %s", record.UnitID)
		}
		if _, ok := out[u.SourcePath]; ok {
			return nil, fmt.Errorf("package output path collision: %s", u.SourcePath)
		}
		out[u.SourcePath] = b
	}
	return out, nil
}
