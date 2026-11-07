package i18n

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func verifyRecoveryManagedDirectory(root, dir string, expected map[string]bool, pending string) error {
	return filepath.WalkDir(filepath.Join(root, filepath.FromSlash(dir)), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink in recovery managed inventory")
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular recovery inventory member")
		}
		rel, err := repositoryRelativePath(root, path)
		if err != nil {
			return err
		}
		if expected[rel] || (pending != "" && rel == dir+"/"+pending) {
			return nil
		}
		return fmt.Errorf("unexpected recovery inventory member: %s", rel)
	})
}

func recoverySelectionIdentity(latest *latestRetranslationUnits) string {
	type selection struct {
		UnitID, BatchID string
		Manifest        RetranslationBatchUnit
		Result          RetranslationUnitResult
		Machine         *GlossaryArchiveReference
	}
	var rows []selection
	for _, u := range latest.ordered {
		c, ok := latest.selectedByID[u.ID]
		if !ok {
			rows = append(rows, selection{UnitID: u.ID})
			continue
		}
		row := selection{UnitID: u.ID, BatchID: c.batchID, Manifest: c.manifest, Result: c.result}
		if c.machine != nil {
			ref := c.machine.ref
			row.Machine = &ref
		}
		rows = append(rows, row)
	}
	return sum(mustJSON(rows))
}

func normalizeRecoveryContextSelection(root string, m *RetranslationBatchManifest, latest *latestRetranslationUnits, child string, contextOnly bool) error {
	if child == "" {
		return nil
	}
	dir := filepath.Join(root, "data/retranslation-runs", m.Locale, m.BatchID)
	prior, err := readPromotionResult(dir, m.Locale, m.BatchID, len(m.Units))
	if err != nil {
		return err
	}
	results := map[string]RetranslationUnitResult{}
	for _, r := range prior.Units {
		results[r.UnitID] = r
	}
	for _, r := range m.Units {
		choice, ok := latest.selectedByID[r.UnitID]
		if !ok || (!contextOnly && choice.batchID != m.BatchID && choice.batchID != child) {
			return fmt.Errorf("recovery root replaced by unrelated latest selection")
		}
		latest.selectedByID[r.UnitID] = selectedRetranslationUnit{batchID: m.BatchID, batchDir: dir, artifactEOF: m.ArtifactEOF, manifest: r, result: results[r.UnitID]}
	}
	return nil
}

func validateRecoveryAuthorityInventory(a *InstalledDerivedTransportRecovery) error {
	p := a.Predecessor
	base := "data/retranslation-runs/" + a.Locale + "/" + p.BatchID + "/"
	if p.Manifest.Path != base+"manifest.json" || p.Result.Path != base+"result.json" || p.ImportReceipt.Path != base+"generation-imports/initial-manifest.json" || p.Archive.Path != importedGenerationArchive(a.Locale, p.Archive.SHA256) || !validSHA256(p.InputIdentitySHA256) || (p.TaskKind != "translation-unit-revision" && p.TaskKind != TransportRecoveryTaskKind) {
		return fmt.Errorf("recovery predecessor invocation paths/task incomplete")
	}
	for _, r := range []RecoveryFileReference{p.Manifest, p.Result, p.ImportReceipt, p.Archive, a.LanguageAuthority.Reopen, a.LanguageAuthority.Snapshot, a.Construction} {
		if !validRecoveryFileReference(r) {
			return fmt.Errorf("recovery mandatory byte reference missing")
		}
	}
	if !validSHA256(a.LanguageAuthority.ReopenIdentity) || a.CurrentAssessment.OriginalBaseline == nil || !a.CurrentAssessment.HistoricalIntegrity || a.CurrentAssessment.LanguageAuthorityIdentity != a.LanguageAuthority.ReopenIdentity || len(a.CurrentAssessment.CurrentCandidateValidation) != len(a.RecoveryUnits.UnitIDs) || len(a.ContentPackage.Units) == 0 || len(a.CurrentEpoch) == 0 {
		return fmt.Errorf("recovery language/assessment/epoch inventory incomplete")
	}
	for i, u := range p.FinalUnits {
		if u.Attempt < 1 {
			return fmt.Errorf("recovery final attempt missing")
		}
		for _, r := range []RecoveryFileReference{u.Raw, u.Candidate, u.Validation} {
			if !validRecoveryFileReference(r) || !strings.HasPrefix(r.Path, base) {
				return fmt.Errorf("recovery final response outside predecessor invocation")
			}
		}
		if i >= len(a.CurrentAssessment.CurrentCandidateValidation) {
			return fmt.Errorf("recovery current validation set incomplete")
		}
		v := a.CurrentAssessment.CurrentCandidateValidation[i]
		if v.UnitID != u.UnitID || v.CandidateSHA256 != u.Candidate.SHA256 || !validSHA256(v.ValidatorIdentity) || v.Status != "passed" {
			return fmt.Errorf("recovery current validation not exact PASS")
		}
	}
	members := map[string]RecoveryFileReference{}
	last := ""
	for _, r := range a.Members {
		if !validRecoveryFileReference(r) || r.Path <= last || r.Path == "authority.json" {
			return fmt.Errorf("recovery members must be unique and ordered")
		}
		last = r.Path
		members[r.Path] = r
	}
	if members[a.Construction.Path] != a.Construction {
		return fmt.Errorf("construction member is not inventoried")
	}
	for _, d := range a.DirectDrift {
		if members[d.OldProof.Path] != d.OldProof || members[d.CurrentProof.Path] != d.CurrentProof || !validSHA256(d.OldProofIdentity) || !validSHA256(d.CurrentProofIdentity) || !validSHA256(d.OldConstructionIdentity) || !validSHA256(d.CurrentConstructionIdentity) {
			return fmt.Errorf("direct drift proof/identity inventory incomplete")
		}
		for _, r := range []*RecoveryFileReference{d.FrozenConstruction, d.CurrentConstruction} {
			if r != nil && members[r.Path] != *r {
				return fmt.Errorf("direct drift construction reference not inventoried")
			}
		}
		for _, delta := range d.Dependencies {
			if !validRecoveryFileReference(delta.Old) || !validRecoveryFileReference(delta.Current) || delta.Old.Path != delta.Current.Path || delta.Old.SHA256 == delta.Current.SHA256 || len(delta.Declarations) == 0 {
				return fmt.Errorf("invalid classified construction dependency delta")
			}
		}
	}
	if a.RecoveryUnits.ContextRoot != p.Archive.SHA256 {
		return fmt.Errorf("recovery context root differs from initial invocation")
	}
	direct := map[string]bool{}
	for _, d := range a.DirectDrift {
		direct[d.UnitID] = true
	}
	var nonDirect []string
	if len(a.RecoveryUnits.Edges) != len(a.RecoveryUnits.UnitIDs) {
		return fmt.Errorf("incomplete recovery decoder context edges")
	}
	for i, id := range a.RecoveryUnits.UnitIDs {
		if a.RecoveryUnits.Edges[i] != (RecoveryContextEdge{id, p.Archive.SHA256, "complete-installed-decoder-context"}) {
			return fmt.Errorf("invalid recovery decoder context edge")
		}
		if !direct[id] {
			nonDirect = append(nonDirect, id)
		}
	}
	if !bytes.Equal(mustJSON(nonDirect), mustJSON(a.RecoveryUnits.NonDirectUnitIDs)) {
		return fmt.Errorf("recovery non-direct scope differs from R minus D")
	}
	for _, set := range [][]RecoveryDependency{a.CurrentEpoch, a.ContentPackage.References, a.LanguageAuthority.References} {
		sorted, err := sortedRecoveryDependencies(set)
		if err != nil {
			return err
		}
		if !bytes.Equal(mustJSON(set), mustJSON(sorted)) {
			return fmt.Errorf("recovery dependency set is not canonical")
		}
	}
	return nil
}

func verifyRecoveryHistoricalBytes(root string, a *InstalledDerivedTransportRecovery) error {
	refs := []RecoveryFileReference{a.Predecessor.Manifest, a.Predecessor.Result, a.Predecessor.ImportReceipt, a.Predecessor.Archive, a.LanguageAuthority.Reopen, a.LanguageAuthority.Snapshot}
	for _, u := range a.Predecessor.FinalUnits {
		refs = append(refs, u.Raw, u.Candidate, u.Validation)
		for _, r := range u.History {
			refs = append(refs, r.Ref)
		}
	}
	for _, r := range a.LanguageAuthority.References {
		refs = append(refs, r.Ref)
	}
	return verifyRecoveryByteReferences(root, refs)
}

func verifyRecoveryByteReferences(root string, refs []RecoveryFileReference) error {
	seen := map[string]RecoveryFileReference{}
	for _, r := range refs {
		if previous, ok := seen[r.Path]; ok {
			if previous != r {
				return fmt.Errorf("conflicting recovery byte observations")
			}
			continue
		}
		seen[r.Path] = r
		b, err := readCompatibilityFile(root, r.Path)
		if err != nil || sum(b) != r.SHA256 || int64(len(b)) != r.Size {
			return fmt.Errorf("recovery byte observation changed: %s", r.Path)
		}
	}
	return nil
}

func verifyRecoveryFinalHistory(root string, m *RetranslationBatchManifest, u *RecoveryFinalUnit, r RetranslationBatchUnit) error {
	base := "data/retranslation-runs/" + m.Locale + "/" + m.BatchID + "/"
	dir := filepath.Join(root, base, "retries", stringsRecoveryInputBase(r.InputPath))
	if u.Attempt > 1 {
		if err := validateRetryAttemptSequenceForExtension(dir, u.Attempt, u.Attempt, filepath.Ext(r.InputPath)); err != nil {
			return err
		}
		for n := 1; n < u.Attempt; n++ {
			name := base + "retries/" + stringsRecoveryInputBase(r.InputPath) + fmt.Sprintf("/attempt-%03d-validation.json", n)
			ref, b, err := recoveryDiskReference(root, name)
			if err != nil {
				return err
			}
			unit := &TranslationUnit{ID: r.UnitID, Kind: r.UnitKind}
			v, err := decodeRetranslationValidation(b, unit)
			if err != nil {
				return err
			}
			if v.Attempt != n || v.UnitID != r.UnitID || v.BatchID != m.BatchID || v.Locale != m.Locale || (v.Status != "restore_failed" && v.Status != "validation_failed") {
				return fmt.Errorf("recovery predecessor validation history inconsistent")
			}
			u.History = append(u.History, RecoveryDependency{"attempt-validation-history", ref})
		}
	} else if entries, err := os.ReadDir(dir); err != nil && !os.IsNotExist(err) {
		return err
	} else if len(entries) > 0 {
		return fmt.Errorf("passed initial invocation has unconsumed retry files")
	}
	return nil
}

func verifyRecoveryHistoricalBundleScope(m *RetranslationBatchManifest, gm GenerationBundleManifest, id string, attempt int) error {
	if gm.UnitKind != m.UnitKind {
		return fmt.Errorf("historical Generation kind mismatch")
	}
	var expected []GenerationBundleExpectedOutput
	for _, r := range m.Units {
		if attempt > 1 && r.UnitID != id {
			continue
		}
		name := filepath.Base(r.InputPath)
		install := "raw-responses/" + name
		if attempt > 1 {
			install = "retries/" + stringsRecoveryInputBase(r.InputPath) + fmt.Sprintf("/attempt-%03d%s", attempt, filepath.Ext(r.InputPath))
		}
		expected = append(expected, GenerationBundleExpectedOutput{BundlePath: "outputs/" + name, InstallPath: install, UnitID: r.UnitID})
	}
	if !bytes.Equal(mustJSON(expected), mustJSON(gm.ExpectedOutputs)) {
		return fmt.Errorf("historical Generation expected output set/ordered scope mismatch")
	}
	var input strings.Builder
	last := ""
	for _, f := range gm.Files {
		if f.BundlePath <= last {
			return fmt.Errorf("historical Generation inventory order mismatch")
		}
		last = f.BundlePath
		input.WriteString(f.BundlePath + "\x00" + f.SHA256 + "\x00")
	}
	if sum([]byte(input.String())) != gm.InputIdentitySHA256 {
		return fmt.Errorf("historical Generation input identity does not bind inventory")
	}
	return nil
}

func canonicalRecoveryDependencySet(in []RecoveryDependency) ([]RecoveryDependency, error) {
	byKey := map[string]RecoveryDependency{}
	for _, r := range in {
		key := r.Role + "\x00" + r.Ref.Path
		if old, ok := byKey[key]; ok && old != r {
			return nil, fmt.Errorf("conflicting dependency observations")
		}
		byKey[key] = r
	}
	var out []RecoveryDependency
	for _, r := range byKey {
		out = append(out, r)
	}
	return sortedRecoveryDependencies(out)
}
