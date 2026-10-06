package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

// FoundationUpgrade is the one additive V2-A -> V2-D static source contract
// upgrade. It cannot update upstream, Tour source/identity, locale completions,
// snapshots or arbitrary changed global authority.
type FoundationUpgrade struct {
	OldSHA         string `json:"old_sha256"`
	NewSHA         string `json:"new_sha256"`
	TourIdentity   string `json:"unchanged_tour_identity"`
	AlreadyCurrent bool   `json:"already_current"`
}

func UpgradeFoundation(root string, apply bool) (*FoundationUpgrade, error) {
	oldBytes, err := readRegular(root, GlobalPath)
	if err != nil {
		return nil, err
	}
	var old Global
	if err := StrictJSON(oldBytes, &old); err != nil {
		return nil, err
	}
	snapshot, err := readRegular(root, SnapshotPath)
	if err != nil {
		return nil, err
	}
	next, err := BuildGlobal(root, snapshot)
	if err != nil {
		return nil, err
	}
	newBytes, err := Encode(next)
	if err != nil {
		return nil, err
	}
	// Reconstruct precisely the previous V2-A contract from the new static source
	// authority. No source/route/dependency drift may hide inside this upgrade.
	legacyBytes, _ := Encode(next)
	var legacy Global
	if err := StrictJSON(legacyBytes, &legacy); err != nil {
		return nil, err
	}
	sources := []Source{}
	for _, s := range legacy.Sources {
		if s.Package != "site-v2-shell" {
			sources = append(sources, s)
		}
	}
	legacy.Sources = sources
	for i := range legacy.Surfaces {
		s := &legacy.Surfaces[i]
		if s.Package == "site-v2-shell" {
			s.Parser = "site-surface"
		}
		if s.Package == "learn-docs-v1" {
			s.Parser = "surface-specific-pending"
		}
	}
	for i := range legacy.Packages {
		p := &legacy.Packages[i]
		p.ParserContract = ""
		part := []Source{}
		for _, s := range legacy.Sources {
			if s.Package == p.ID {
				part = append(part, s)
			}
		}
		p.SourceIdentity = identity(struct {
			Sources   []Source
			Contracts []Dependency
		}{part, p.Dependencies})
		p.Identity = ""
		p.Identity = identity(*p)
	}
	legacy.Identity = ""
	legacy.Identity = identity(legacy)
	// The uncommitted V2-D v1 parser contract has no locale evidence. The repair
	// accepts only its exact static identity, with the same frozen sources/Tour.
	var priorD Global
	if err := StrictJSON(newBytes, &priorD); err != nil {
		return nil, err
	}
	for i := range priorD.Surfaces {
		if priorD.Surfaces[i].Parser == UnitContract {
			priorD.Surfaces[i].Parser = "go-learning/content-units/v1"
		}
	}
	for i := range priorD.Packages {
		if priorD.Packages[i].ParserContract == UnitContract {
			priorD.Packages[i].ParserContract = "go-learning/content-units/v1"
			priorD.Packages[i].Identity = ""
			priorD.Packages[i].Identity = identity(priorD.Packages[i])
		}
	}
	priorD.Identity = ""
	priorD.Identity = identity(priorD)
	plan := &FoundationUpgrade{OldSHA: digest(oldBytes), NewSHA: digest(newBytes), TourIdentity: next.Packages[0].Identity, AlreadyCurrent: reflect.DeepEqual(old, *next)}
	if !plan.AlreadyCurrent && !reflect.DeepEqual(old, legacy) && !reflect.DeepEqual(old, priorD) {
		return nil, fmt.Errorf("foundation upgrade refuses non-V2-A or changed authority")
	}
	if old.Packages[0].Identity != plan.TourIdentity {
		return nil, fmt.Errorf("foundation upgrade would change Tour identity")
	}
	if !apply || plan.AlreadyCurrent {
		return plan, nil
	}
	current, err := readRegular(root, GlobalPath)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(current, oldBytes) {
		return nil, fmt.Errorf("global authority concurrent change")
	}
	dir := filepath.Dir(filepath.Join(root, GlobalPath))
	f, err := os.CreateTemp(dir, ".site-foundation-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f.Name())
	if err := f.Chmod(0644); err != nil {
		f.Close()
		return nil, err
	}
	if _, err := f.Write(newBytes); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	if err := os.Rename(f.Name(), filepath.Join(root, GlobalPath)); err != nil {
		return nil, err
	}
	return plan, nil
}
