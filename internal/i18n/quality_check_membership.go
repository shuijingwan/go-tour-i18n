package i18n

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
)

// A membership transition consumes complete finalized predecessors in their
// own ordered scopes. It never makes the primary Snapshot claim sibling Units
// or gives a revised candidate its predecessor's A. The projection and current
// eligibility readers are the same ones used to create the Candidate Snapshot.
type qualityCheckMembershipBaseline struct {
	primary           *QualityCheckSnapshotManifest
	results           map[string]effectiveQualityCheckResult
	checks            []func() error
	references        []GlossaryArchiveReference
	contextReferences []GlossaryArchiveReference
}

func (e *qualityCheckEvaluation) previousQualityCheckResults(s *QualityCheckSnapshotManifest, previous string, seen map[string]bool) (*QualityCheckSnapshotManifest, map[string]effectiveQualityCheckResult, error) {
	if !e.originalOnly {
		baseline, err := e.membershipQualityCheckBaseline(s, previous)
		if err != nil {
			return nil, nil, err
		}
		if baseline != nil {
			return baseline.primary, cloneEffectiveQualityCheckResults(baseline.results), nil
		}
	}
	predecessor, err := e.machinePredecessorEvaluation(s, previous)
	if err != nil {
		return nil, nil, err
	}
	return predecessor.loadEffectiveQualityCheckResults(previous, seen)
}

func (e *qualityCheckEvaluation) membershipQualityCheckBaseline(s *QualityCheckSnapshotManifest, previous string) (*qualityCheckMembershipBaseline, error) {
	if b := e.membership[s.SnapshotID]; b != nil {
		if b.primary.SnapshotID != previous {
			return nil, fmt.Errorf("membership QC predecessor changed")
		}
		return b, nil
	}
	members, membershipCheck, err := snapshotMembershipCatalogs(e.root, e.catalog)
	if err != nil || len(members) == 0 {
		return nil, err
	}
	inventory := *e.catalog
	inventory.TranslationScope = nil
	primary, registered, err := selectSnapshotTranslationPackage(e.root, &inventory, e.locale, previous)
	if err != nil {
		return nil, err
	}
	if !registered {
		return nil, fmt.Errorf("membership QC requires a registered predecessor")
	}
	_, _, primaryScope, _, err := translationAuthorizationScope(primary)
	if err != nil {
		return nil, err
	}
	_, _, currentScope, _, err := translationAuthorizationScope(e.catalog)
	if err != nil {
		return nil, err
	}
	// A later complete current-package predecessor uses the ordinary QC lineage.
	if primaryScope == currentScope {
		return nil, nil
	}
	var primaryMember *Catalog
	for _, c := range members {
		_, _, scope, _, err := translationAuthorizationScope(c)
		if err != nil {
			return nil, err
		}
		if scope == primaryScope {
			primaryMember = c
			break
		}
	}
	if s.ContentPackage == nil {
		return nil, fmt.Errorf("membership QC successor requires current package binding")
	}
	if primaryMember == nil {
		return nil, fmt.Errorf("QC predecessor outside formal membership projection")
	}

	snapshotPath := filepath.ToSlash(filepath.Join("data/quality-check-snapshots", e.locale, s.SnapshotID, "manifest.json"))
	snapshotData, err := readSnapshotReferencedFile(e.root, snapshotPath)
	if err != nil {
		return nil, err
	}
	var frozenSnapshot QualityCheckSnapshotManifest
	if err := json.Unmarshal(snapshotData, &frozenSnapshot); err != nil || !reflect.DeepEqual(*s, frozenSnapshot) {
		return nil, fmt.Errorf("membership QC Snapshot changed during evaluation")
	}

	// Select from the frozen Snapshot, never from a latest batch or a path union.
	// Ordinary repository evidence binds the exact processed final attempt;
	// machine evidence is decoded in the declared historical member world.
	frozen := &latestRetranslationUnits{selectedByID: map[string]selectedRetranslationUnit{}}
	frozen.ordered, frozen.pageCount, frozen.exampleCount, err = localeWorkflowUnitList(e.catalog)
	if err != nil {
		return nil, err
	}
	evidence := map[string]*snapshotUnitRepositoryEvidence{}
	// Preparation belongs to this one evaluation and exact historical Catalog,
	// not to an individual Unit. The existing reader shares only the receipt's
	// prepared QC world: every Unit still rebuilds its machine proof, and the
	// complete historical/current references are freshly checked by b.check.
	historicalMachines := map[*Catalog]map[string]*machineSelection{}
	for _, u := range s.Units {
		c := members[u.UnitID]
		if c == nil {
			return nil, fmt.Errorf("Snapshot Unit outside membership projection")
		}
		var ev *snapshotUnitRepositoryEvidence
		var choice selectedRetranslationUnit
		if u.RepresentationReconciliation != nil {
			var machine *machineSelection
			if historicalMachines[c] == nil {
				historicalMachines[c] = map[string]*machineSelection{}
			}
			ev, machine, err = machineSnapshotEvidenceInWorld(e.root, c, e.locale, u, historicalMachines[c], true)
			if err != nil {
				return nil, err
			}
			choice = selectedRetranslationUnit{batchID: u.SelectedBatchID, machine: machine, result: RetranslationUnitResult{UnitID: u.UnitID, UnitKind: u.UnitKind, Status: "passed", CandidatePath: u.CandidatePath, ValidationPath: u.ValidationPath}}
		} else {
			ev, err = readSnapshotUnitRepositoryEvidence(e.root, c, e.locale, u)
			if err != nil {
				return nil, err
			}
			m, err := readRetranslationProcessManifest(ev.batchDir, e.locale, u.SelectedBatchID)
			if err != nil {
				return nil, err
			}
			choice = selectedRetranslationUnit{batchID: u.SelectedBatchID, batchDir: ev.batchDir, artifactEOF: m.ArtifactEOF, manifest: ev.record, result: ev.result}
		}
		frozen.selectedByID[u.UnitID], evidence[u.UnitID] = choice, ev
	}
	proofs, assessmentCheck, err := snapshotMembershipAssessments(e.root, e.locale, members, frozen, map[*Catalog]string{primaryMember: previous})
	if err != nil {
		return nil, err
	}
	b := &qualityCheckMembershipBaseline{results: map[string]effectiveQualityCheckResult{}, checks: []func() error{membershipCheck, assessmentCheck}}
	b.checks = append(b.checks, func() error {
		return checkSnapshotPackageBinding(e.root, e.catalog, s, true)
	})
	refs := map[string]GlossaryArchiveReference{}
	bind := func(ref GlossaryArchiveReference) error {
		data, err := readSnapshotReferencedFile(e.root, ref.Path)
		if err != nil || !validSHA256(ref.SHA256) || sum(data) != ref.SHA256 {
			return fmt.Errorf("membership QC evidence drift: %s", ref.Path)
		}
		if old, ok := refs[ref.Path]; ok && old != ref {
			return fmt.Errorf("conflicting membership QC evidence: %s", ref.Path)
		}
		refs[ref.Path] = ref
		return nil
	}
	for c, a := range proofs {
		if a == nil || !a.Ready() {
			return nil, fmt.Errorf("membership QC requires complete current-compatible finalized A predecessors")
		}
		original := newQualityCheckEvaluation(e.root, e.locale, c)
		original.originalOnly = true
		if err := original.materializeOriginalFinalization(&a.Historical); err != nil {
			return nil, err
		}
		m := original.effective[a.Historical.Finalization.SnapshotID]
		for id, r := range m.results {
			if members[id] != c {
				return nil, fmt.Errorf("QC predecessor outside ordered membership")
			}
			if _, duplicate := b.results[id]; duplicate {
				return nil, fmt.Errorf("overlapping membership QC predecessors")
			}
			r.currentReattested = true
			b.results[id] = r
		}
		if c == primaryMember {
			if a.Historical.Finalization.SnapshotID != previous {
				return nil, fmt.Errorf("membership primary QC identity mismatch")
			}
			b.primary = m.snapshot
		}
		for _, ref := range a.Historical.References {
			if ref.Path == a.Historical.Finalization.SnapshotManifestPath || ref.Path == filepath.ToSlash(filepath.Join("data/quality-check-snapshots", e.locale, a.Historical.Finalization.SnapshotID, "finalization.json")) {
				b.contextReferences = append(b.contextReferences, ref)
			}
		}
		b.contextReferences = append(b.contextReferences, a.Glossary, a.UnifiedReview)
		for _, ref := range append(append([]GlossaryArchiveReference{}, a.Historical.References...), a.Glossary, a.UnifiedReview) {
			if err := bind(ref); err != nil {
				return nil, err
			}
		}
	}
	if b.primary == nil || len(b.results) != len(s.Units) {
		return nil, fmt.Errorf("incomplete membership historical QC baseline")
	}
	g, err := LoadGlossary(e.root, e.locale)
	if err != nil {
		return nil, err
	}
	outputs, checks, err := prepareSnapshotRevisionOutputs(e.root, e.catalog, e.locale, g, frozen, members)
	if err != nil {
		return nil, err
	}
	b.checks = append(b.checks, checks...)
	for _, u := range s.Units {
		choice := frozen.selectedByID[u.UnitID]

		for _, ref := range []GlossaryArchiveReference{{Path: u.CandidatePath, SHA256: u.CandidateSHA256}, {Path: u.ValidationPath, SHA256: u.ValidationSHA256}, {Path: u.SourcePath, SHA256: u.SourceSHA256}} {
			if err := bind(ref); err != nil {
				return nil, err
			}
		}
		if choice.machine != nil {
			for _, ref := range machineAuthorityReferences(choice.machine) {
				if err := bind(ref); err != nil {
					return nil, err
				}
			}
		} else {
			base, err := repositoryRelativePath(e.root, choice.batchDir)
			if err != nil {
				return nil, err
			}
			for _, path := range []string{base + "/manifest.json", base + "/result.json", base + "/" + choice.manifest.InputPath, base + "/" + evidence[u.UnitID].validation.RawResponsePath} {
				data, err := readSnapshotReferencedFile(e.root, path)
				if err != nil {
					return nil, err
				}
				if err := bind(GlossaryArchiveReference{Path: path, SHA256: sum(data)}); err != nil {
					return nil, err
				}
			}
		}
		v, err := validateSnapshotMembershipCandidate(e.root, e.catalog, e.locale, g, evidence[u.UnitID].unit, choice, snapshotPreparedOutput([]map[string]protectedTranslation{outputs}, u.UnitID), members, proofs, e.compatibility)
		if err != nil {
			return nil, err
		}
		if sum(v.candidate) != u.CandidateSHA256 || sum(v.validationData) != u.ValidationSHA256 || v.attempt != u.Attempt {
			return nil, fmt.Errorf("membership Snapshot final attempt drift")
		}
	}
	if err := bind(GlossaryArchiveReference{Path: snapshotPath, SHA256: sum(snapshotData)}); err != nil {
		return nil, err
	}

	b.contextReferences = append(b.contextReferences, s.ContentPackage.AuthorityReferences...)
	for _, ref := range s.ContentPackage.AuthorityReferences {
		if err := bind(ref); err != nil {
			return nil, err
		}
	}
	for _, ref := range refs {
		b.references = append(b.references, ref)
	}
	sort.Slice(b.contextReferences, func(i, j int) bool { return b.contextReferences[i].Path < b.contextReferences[j].Path })
	sort.Slice(b.references, func(i, j int) bool { return b.references[i].Path < b.references[j].Path })
	b.checks = append(b.checks, func() error {
		for _, ref := range b.references {
			if err := bind(ref); err != nil {
				return err
			}
		}
		return nil
	})
	if err := b.check(); err != nil {
		return nil, err
	}
	if e.membership == nil {
		e.membership = map[string]*qualityCheckMembershipBaseline{}
	}
	e.membership[s.SnapshotID] = b
	for id, ev := range evidence {
		e.repositoryEvidence[s.SnapshotID+"\x00"+id] = ev
	}
	return b, nil
}

func (b *qualityCheckMembershipBaseline) check() error {
	for _, check := range b.checks {
		if check != nil {
			if err := check(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *qualityCheckEvaluation) membershipChecks() error {
	for _, b := range e.membership {
		if err := b.check(); err != nil {
			return err
		}
	}
	return nil
}
