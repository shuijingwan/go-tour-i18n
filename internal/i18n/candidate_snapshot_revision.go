package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
)

// The original current assessment remains sealed. This additive projection
// binds a current predecessor and exact authorized machine-valid replacements;
// none of the replacements acquire a Reviewer rating here.
type QualityCheckSnapshotRevision struct {
	Baseline GlossaryArchiveReference   `json:"baseline"`
	Delta    []QualityCheckSnapshotUnit `json:"delta"`
	Evidence []GlossaryArchiveReference `json:"immutable_evidence"`
	Identity string                     `json:"identity_sha256"`
}

func snapshotRevisionIdentity(p QualityCheckSnapshotRevision) string {
	p.Identity = ""
	return sum(mustJSON(p))
}

func choiceMatchesSnapshot(locale string, choice selectedRetranslationUnit, old QualityCheckSnapshotUnit) bool {
	base := "data/retranslation-runs/" + locale + "/" + choice.batchID + "/"
	return choice.batchID == old.SelectedBatchID && base+choice.result.CandidatePath == old.CandidatePath && base+choice.result.ValidationPath == old.ValidationPath
}

func selectedMixedSnapshot(root string, c *Catalog, locale string, latest *latestRetranslationUnits, outputs ...map[string]protectedTranslation) (*CurrentTranslationAuthorization, *QualityCheckSnapshotRevision, []QualityCheckSnapshotUnit, error) {
	if len(c.TranslationScope) == 0 {
		return nil, nil, nil, nil
	}
	entries, err := os.ReadDir(filepath.Join(root, "data/quality-check-snapshots", locale))
	if os.IsNotExist(err) {
		return nil, nil, nil, nil
	}
	if err != nil {
		return nil, nil, nil, err
	}
	var baseline *QualityCheckSnapshotManifest
	var rejected bool
	for _, e := range entries {
		if !e.IsDir() || e.Name()[0] == '.' {
			continue
		}
		s, err := readQualityCheckSnapshot(root, locale, e.Name(), false)
		if err != nil {
			return nil, nil, nil, err
		}
		if s.CurrentReattestation == nil || len(s.Units) != len(latest.ordered) {
			continue
		}
		inScope := true
		for i, u := range s.Units {
			if u.UnitID != latest.ordered[i].ID {
				inScope = false
			}
		}
		if !inScope {
			continue
		}
		changed, authorized := 0, true
		for _, u := range s.Units {
			choice, ok := latest.selectedByID[u.UnitID]
			if !ok || !choiceMatchesSnapshot(locale, choice, u) {
				changed++
				if !ok || choice.manifest.PreviousSnapshotID != s.SnapshotID {
					authorized = false
				}
			}
		}
		if changed == 0 {
			continue
		}
		rejected = true
		if !authorized {
			continue
		}
		if baseline != nil {
			return nil, nil, nil, fmt.Errorf("ambiguous current revision baseline")
		}
		baseline = s
	}
	if baseline == nil {
		if rejected {
			return nil, nil, nil, fmt.Errorf("latest candidate replacement lacks exact current baseline revision authorization")
		}
		return nil, nil, nil, nil
	}
	baseline, err = readQualityCheckSnapshotForReview(root, locale, baseline.SnapshotID, c)
	if err != nil {
		return nil, nil, nil, err
	}
	p, units, err := buildSnapshotRevision(root, c, locale, baseline, latest, outputs...)
	return baseline.CurrentReattestation, p, units, err
}

func buildSnapshotRevision(root string, c *Catalog, locale string, base *QualityCheckSnapshotManifest, latest *latestRetranslationUnits, outputs ...map[string]protectedTranslation) (*QualityCheckSnapshotRevision, []QualityCheckSnapshotUnit, error) {
	baselinePath := "data/quality-check-snapshots/" + locale + "/" + base.SnapshotID + "/manifest.json"
	ref, err := successorReference(root, baselinePath)
	if err != nil {
		return nil, nil, err
	}
	p := &QualityCheckSnapshotRevision{Baseline: ref, Delta: []QualityCheckSnapshotUnit{}, Evidence: []GlossaryArchiveReference{}}
	refs := map[string]GlossaryArchiveReference{ref.Path: ref}
	if base.CurrentRevision != nil {
		for _, r := range base.CurrentRevision.Evidence {
			refs[r.Path] = r
		}
	}
	add := func(path string) error {
		r, err := successorReference(root, path)
		if err == nil {
			refs[path] = r
		}
		return err
	}
	g, err := LoadGlossary(root, locale)
	if err != nil {
		return nil, nil, err
	}
	results, err := readQualityCheckResults(root, locale, base)
	if err != nil {
		return nil, nil, err
	}
	units := append([]QualityCheckSnapshotUnit(nil), base.Units...)
	invocations := map[string][]GlossaryArchiveReference{}
	for i, old := range base.Units {
		u := latest.ordered[i]
		choice, ok := latest.selectedByID[old.UnitID]
		if !ok || !selectedRetranslationIdentityMatches(u, choice) || old.SourceSHA256 != u.SourceSHA256 || old.SourcePath != u.SourcePath || old.UnitKind != u.Kind {
			return nil, nil, fmt.Errorf("revision baseline source/selection drift: %s", old.UnitID)
		}
		if choiceMatchesSnapshot(locale, choice, old) {
			continue
		}
		r := choice.manifest
		if r.PreviousSnapshotID != base.SnapshotID {
			return nil, nil, fmt.Errorf("revision previous Snapshot mismatch")
		}
		if r.RevisionFeedbackSource == "quality_check" {
			found := false
			if results != nil {
				for _, j := range results.Results {
					if j.UnitID == u.ID && (j.Rating == "B" || j.Rating == "C" || j.Rating == "D") && j.Rating == r.PreviousRating && j.Finding != "" && j.Finding == r.PreviousFinding {
						found = true
					}
				}
			}
			if !found {
				return nil, nil, fmt.Errorf("revision requires exact predecessor B/C/D finding: %s", u.ID)
			}
			if err := add("data/quality-check-snapshots/" + locale + "/" + base.SnapshotID + "/quality-check-results.json"); err != nil {
				return nil, nil, err
			}
		} else if !isReopenFeedback(r.RevisionFeedbackSource) {
			return nil, nil, fmt.Errorf("unauthorized latest replacement: %s", u.ID)
		}
		if authority, err := currentTranslationAuthority(root, u, locale); err != nil || !validSHA256(authority) {
			return nil, nil, fmt.Errorf("unknown revision adapter/parser: %s", u.ID)
		}
		v, boundRefs, err := snapshotBoundCurrentValidationCandidate(root, c, locale, u, choice)
		bound := v != nil
		if err == nil && !bound {
			v, err = validateSelectedRetranslationCandidateWithOutput(root, c, locale, g, u, choice, snapshotPreparedOutput(outputs, u.ID))
		}
		if err != nil {
			return nil, nil, err
		}
		key := choice.batchID
		if v.attempt > 1 {
			key += fmt.Sprintf("/%s/%d", u.ID, v.attempt)
		}
		generationRefs, verified := invocations[key]
		if !verified {
			if bound {
				generationRefs = boundRefs
			} else {
				generationRefs, err = verifySnapshotRevisionInvocation(root, c, locale, choice, v, latest.selectedByID)
			}
			if err != nil {
				return nil, nil, err
			}
			invocations[key] = generationRefs
		}
		for _, ref := range generationRefs {
			refs[ref.Path] = ref
		}
		batch := "data/retranslation-runs/" + locale + "/" + choice.batchID + "/"
		for _, path := range []string{batch + "manifest.json", batch + "result.json", batch + r.InputPath, batch + v.candidatePath, batch + v.validationPath} {
			if err := add(path); err != nil {
				return nil, nil, err
			}
		}
		n := old
		n.SelectedBatchID = choice.batchID
		n.CandidatePath = batch + v.candidatePath
		n.CandidateSHA256 = sum(v.candidate)
		n.ValidationPath = batch + v.validationPath
		n.ValidationSHA256 = sum(v.validationData)
		n.Attempt = v.attempt
		units[i] = n
		p.Delta = append(p.Delta, n)
	}
	if len(p.Delta) == 0 {
		return nil, nil, fmt.Errorf("mixed Snapshot requires a nonempty authorized delta")
	}
	for _, r := range refs {
		p.Evidence = append(p.Evidence, r)
	}
	sort.Slice(p.Evidence, func(i, j int) bool { return p.Evidence[i].Path < p.Evidence[j].Path })
	p.Identity = snapshotRevisionIdentity(*p)
	return p, units, nil
}

func checkMixedSnapshot(root string, c *Catalog, s *QualityCheckSnapshotManifest, current bool, seen map[string]bool) error {
	p := s.CurrentRevision
	if p.Identity != snapshotRevisionIdentity(*p) || len(p.Delta) == 0 || !validSHA256(p.Baseline.SHA256) || p.Baseline.Path == "data/quality-check-snapshots/"+s.Locale+"/"+s.SnapshotID+"/manifest.json" {
		return fmt.Errorf("mixed Snapshot proof identity mismatch")
	}
	last := 0
	for _, u := range p.Delta {
		if u.Index <= last || u.Index > len(s.Units) || !reflect.DeepEqual(s.Units[u.Index-1], u) {
			return fmt.Errorf("mixed Snapshot delta exact-set mismatch")
		}
		last = u.Index
	}
	if !current || c == nil {
		return nil
	}
	if seen[s.SnapshotID] {
		return fmt.Errorf("mixed Snapshot baseline cycle")
	}
	seen[s.SnapshotID] = true
	defer delete(seen, s.SnapshotID)
	for _, r := range p.Evidence {
		b, err := readCompatibilityFile(root, r.Path)
		if err != nil || sum(b) != r.SHA256 {
			return fmt.Errorf("mixed Snapshot evidence changed: %s", r.Path)
		}
	}
	b, err := readCompatibilityFile(root, p.Baseline.Path)
	if err != nil || sum(b) != p.Baseline.SHA256 {
		return fmt.Errorf("mixed Snapshot baseline missing/changed")
	}
	id := filepath.Base(filepath.Dir(p.Baseline.Path))
	if p.Baseline.Path != "data/quality-check-snapshots/"+s.Locale+"/"+id+"/manifest.json" {
		return fmt.Errorf("mixed Snapshot baseline path mismatch")
	}
	base, err := readQualityCheckSnapshot(root, s.Locale, id, false)
	if err != nil {
		return err
	}
	if base.CurrentReattestation == nil || !reflect.DeepEqual(base.CurrentReattestation, s.CurrentReattestation) {
		return fmt.Errorf("mixed Snapshot sealed assessment mismatch")
	}
	if base.CurrentRevision != nil {
		err = checkMixedSnapshot(root, c, base, true, seen)
	} else {
		err = checkSnapshotReattestation(root, c, base, true)
	}
	if err != nil {
		return err
	}
	// Reconstruct the exact selected set from this Snapshot, not an arbitrary
	// future latest batch; new latest selection cannot revoke frozen judgment.
	latest := &latestRetranslationUnits{selectedByID: map[string]selectedRetranslationUnit{}}
	ordered, _, _, err := localeWorkflowUnitList(c)
	if err != nil {
		return err
	}
	latest.ordered = ordered
	if len(ordered) != len(s.Units) {
		return fmt.Errorf("mixed Snapshot scope drift")
	}
	for _, u := range s.Units {
		e, err := readSnapshotUnitRepositoryEvidence(root, c, s.Locale, u)
		if err != nil {
			return err
		}
		latest.selectedByID[u.UnitID] = selectedRetranslationUnit{batchID: u.SelectedBatchID, batchDir: e.batchDir, manifest: e.record, result: e.result}
	}
	g, err := LoadGlossary(root, s.Locale)
	if err != nil {
		return err
	}
	outputs, checks, err := prepareSnapshotRevisionOutputs(root, c, s.Locale, g, latest)
	if err != nil {
		return err
	}
	want, units, err := buildSnapshotRevision(root, c, s.Locale, base, latest, outputs)
	if err != nil {
		return err
	}
	for _, check := range checks {
		if err := check(); err != nil {
			return err
		}
	}
	if !reflect.DeepEqual(want, p) || !reflect.DeepEqual(units, s.Units) {
		return fmt.Errorf("mixed Snapshot baseline + delta projection mismatch")
	}
	return nil
}
