package sitecontent

import (
	"fmt"
	"sort"
	"strings"
)

type UnitChange struct {
	ID          string `json:"id"`
	Path        string `json:"path"`
	State       string `json:"state"`
	Reason      string `json:"reason"`
	Replacement string `json:"replacement_id,omitempty"`
}
type Reconciliation struct {
	Contract     string       `json:"contract"`
	OldSHA       string       `json:"old_sha256"`
	NewSHA       string       `json:"new_sha256"`
	Units        []UnitChange `json:"units"`
	NeedsMapping []string     `json:"needs_mapping"`
}

func validateDocuments(docs []Document) error {
	if len(docs) == 0 {
		return fmt.Errorf("empty document set")
	}
	paths := map[string]bool{}
	ids := map[string]bool{}
	for _, d := range docs {
		if d.Contract != UnitContract || !validPath(d.Path) || paths[d.Path] || (d.Kind != "page" && d.Kind != "data") || !validSHA(d.SourceSHA) || !validSHA(d.StructureSHA) {
			return fmt.Errorf("unknown/invalid document identity")
		}
		if d.UnitScope != "" && d.UnitScope != PageContract {
			return fmt.Errorf("unknown formal TU contract")
		}
		if d.UnitScope == PageContract {
			if len(d.Units) != 1 || d.Kind != "page" || d.StableIndex < 1 || d.StableIndex > 60 {
				return fmt.Errorf("invalid formal Page TU")
			}
			raw, err := restoreBlock(d.Units[0], d.Units[0].Source)
			if err != nil {
				return err
			}
			parsed, err := ParseWorkflowDocument(documentSource(d), []byte(raw), d.StableIndex)
			if err != nil || identity(parsed) != identity(d) {
				return fmt.Errorf("formal Page source/protection identity mismatch")
			}
		}
		paths[d.Path] = true
		for _, dep := range d.Dependencies {
			if !validPath(dep.Path) || !validSHA(dep.SHA256) || (dep.Kind != "asset" && dep.Kind != "data" && dep.Kind != "contract") {
				return fmt.Errorf("invalid dependency context")
			}
		}
		last := 0
		for _, u := range d.Units {
			restored, err := restoreBlock(u, u.Source)
			if err != nil || u.ID != digest([]byte(UnitContract+"\n"+u.Locator)) || u.SourceSHA != unitSourceIdentity(u.Source, u.Protected) || u.Start < last || u.End <= u.Start || u.End-u.Start != len(restored) || ids[u.ID] {
				return fmt.Errorf("invalid content-unit identity")
			}
			last = u.End
			ids[u.ID] = true
		}
	}
	return nil
}

// ReconcileDocuments is deliberately read-only. No commit-wide invalidation,
// network access, source application, guessed moves or historical rewriting.
// Route/file moves require explicit authority mapping before activation.
func ReconcileDocuments(old, next []Document) (*Reconciliation, error) {
	if err := validateDocuments(old); err != nil {
		return nil, err
	}
	if err := validateDocuments(next); err != nil {
		return nil, err
	}
	result := &Reconciliation{Contract: UnitContract, OldSHA: documentIdentity(old), NewSHA: documentIdentity(next), Units: []UnitChange{}, NeedsMapping: []string{}}
	prior := map[string]Document{}
	future := map[string]Document{}
	for _, d := range old {
		prior[d.Path] = d
	}
	for _, d := range next {
		future[d.Path] = d
	}
	a, b := unitsByID(old), unitsByID(next)
	removed := map[string][]Unit{}
	added := map[string][]Unit{}
	key := func(u Unit) string { return strings.Split(u.Locator, ":")[0] }
	for _, u := range a {
		if b[u.ID].ID == "" {
			removed[key(u)] = append(removed[key(u)], u)
		}
	}
	for _, u := range b {
		if a[u.ID].ID == "" {
			added[key(u)] = append(added[key(u)], u)
		}
	}
	replacements := map[string]string{}
	ambiguous := map[string]bool{}
	for k, units := range removed {
		newUnits := added[k]
		if len(units) == 1 && len(newUnits) == 1 {
			replacements[units[0].ID] = newUnits[0].ID
		} else if len(newUnits) > 0 {
			for _, u := range units {
				ambiguous[u.ID] = true
			}
			result.NeedsMapping = append(result.NeedsMapping, k)
		}
	}
	for _, d := range next {
		p, ok := prior[d.Path]
		if ok && (p.Route != d.Route || p.Kind != d.Kind) {
			result.NeedsMapping = append(result.NeedsMapping, d.Path)
		}
	}
	for _, d := range old {
		for _, u := range d.Units {
			n, ok := b[u.ID]
			state, reason := "retired", "removed_source_unit"
			replacement := replacements[u.ID]
			if replacement != "" {
				state, reason = "stale", "changed_source_unit"
			}
			if ambiguous[u.ID] {
				state, reason = "ambiguous", "source_unit_mapping_required"
			}
			if ok {
				p := future[d.Path]
				state, reason = "carry", "exact_source_and_protected_context"
				if n.SourceSHA != u.SourceSHA || p.StructureSHA != d.StructureSHA || p.Route != d.Route || p.Kind != d.Kind {
					state, reason = "stale", "changed_source_route_or_protected_context"
				}
			}
			result.Units = append(result.Units, UnitChange{ID: u.ID, Path: d.Path, State: state, Reason: reason, Replacement: replacement})
		}
	}
	for _, d := range next {
		for _, u := range d.Units {
			if _, ok := a[u.ID]; !ok {
				result.Units = append(result.Units, UnitChange{ID: u.ID, Path: d.Path, State: "generation", Reason: "new_or_changed_source_unit"})
			}
		}
	}
	sort.Slice(result.Units, func(i, j int) bool { return result.Units[i].ID < result.Units[j].ID })
	sort.Strings(result.NeedsMapping)
	return result, nil
}
