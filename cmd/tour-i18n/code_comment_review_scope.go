package main

import (
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func commentRowNeedsReview(r commentAuditRow) bool {
	return r.ReviewRequired || r.EnglishResidueSuspect || r.Missing || r.Extra || !r.StructureMatch || !r.ProtectedUnchanged
}

type commentReviewUnit struct {
	StableIndex int               `json:"stable_index"`
	Kind        i18n.UnitKind     `json:"parent_kind"`
	ID          string            `json:"parent_unit_id"`
	Route       string            `json:"route"`
	Path        string            `json:"source_path"`
	Slots       []commentAuditRow `json:"slots"`
	Diagnostics []string          `json:"diagnostics"`
}
type commentReviewPackage struct {
	Package string              `json:"package"`
	Units   []commentReviewUnit `json:"units"`
}
type commentReviewLocale struct {
	Locale   string                 `json:"locale"`
	Base     string                 `json:"base_language"`
	Packages []commentReviewPackage `json:"packages"`
}

// Proposal only: no ratings, replacements, lifecycle receipts or invalidation.
// Both locale order and formal Unit order come from the existing audit/catalog.
func commentReviewerScope(audits []commentLocaleAudit, inventory []commentInventoryUnit) []commentReviewLocale {
	out := []commentReviewLocale{}
	for _, a := range audits {
		locale := commentReviewLocale{Locale: a.Locale, Base: a.Base, Packages: []commentReviewPackage{}}
		for _, p := range a.Packages {
			if p.State != "CURRENT" {
				continue
			}
			pkg := commentReviewPackage{Package: p.Package, Units: []commentReviewUnit{}}
			for _, u := range inventory {
				if u.Package != p.Package {
					continue
				}
				unit := commentReviewUnit{StableIndex: u.StableIndex, Kind: u.Kind, ID: u.ID, Route: u.Route, Path: u.Path, Slots: []commentAuditRow{}, Diagnostics: []string{}}
				for _, r := range p.Rows {
					if r.ID == u.ID && commentRowNeedsReview(r) {
						unit.Slots = append(unit.Slots, r)
					}
				}
				for _, d := range p.Diagnostics {
					if strings.HasPrefix(d, u.ID+": ") {
						unit.Diagnostics = append(unit.Diagnostics, d)
					}
				}
				if len(unit.Slots) > 0 || len(unit.Diagnostics) > 0 {
					pkg.Units = append(pkg.Units, unit)
				}
			}
			if len(pkg.Units) > 0 {
				locale.Packages = append(locale.Packages, pkg)
			}
		}
		if len(locale.Packages) > 0 {
			out = append(out, locale)
		}
	}
	return out
}
