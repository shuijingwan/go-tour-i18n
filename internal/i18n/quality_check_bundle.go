package i18n

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const QualityCheckReviewerBundleSchemaVersion = 1

var qualityCheckReviewerAuthorityPaths = []string{
	"docs/LANGUAGE_DEFECT_EXHAUSTION.md",
	"AGENTS.md",
	"docs/SHARED_TRANSLATION_UNIT_LIFECYCLE.md",
	"docs/WORKFLOW_HANDOFF.md",
	"docs/CHATGPT_LANGUAGE_GENERATION.md",
	"docs/GLOSSARY_REVIEW.md",
	"docs/TRANSLATION_QUALITY_REVIEW.md",
	"docs/TRANSLATION_TASK_SPEC.md",
}

func VerifyCurrentQualityCheckReviewerBundle(root string, catalog *Catalog, bundle []byte) (QualityCheckReviewerBundleManifest, error) {
	files, err := ReadTransportBundle(bundle, 512, 128<<20)
	if err != nil {
		return QualityCheckReviewerBundleManifest{}, err
	}
	var manifest QualityCheckReviewerBundleManifest
	if err := decodeStrictBundleJSON(files["manifest.json"], &manifest); err != nil {
		return manifest, fmt.Errorf("parse Quality Check reviewer bundle manifest: %w", err)
	}
	if manifest.SchemaVersion != QualityCheckReviewerBundleSchemaVersion || manifest.Kind != "go-tour-i18n/translation-unit-quality-check-reviewer-bundle" || manifest.Locale == "" || manifest.SnapshotID == "" {
		return manifest, fmt.Errorf("Quality Check reviewer bundle has incompatible contract")
	}
	if err := ValidateTransportBundleInventory(files, manifest.Files, true); err != nil {
		return manifest, err
	}
	current, currentManifest, err := ExportQualityCheckReviewerBundle(root, catalog, QualityCheckReviewerBundleOptions{
		Locale: manifest.Locale, SnapshotID: manifest.SnapshotID, PreviousSnapshotID: manifest.PreviousSnapshotID,
		StartIndex: manifest.StartIndex, Limit: manifest.UnitCount,
	})
	if err != nil {
		return manifest, err
	}
	if !bytes.Equal(current, bundle) {
		return manifest, fmt.Errorf("Quality Check reviewer bundle is stale or non-canonical")
	}
	return currentManifest, nil
}

type QualityCheckReviewerBundleOptions struct {
	Locale             string
	SnapshotID         string
	PreviousSnapshotID string
	StartIndex         int
	Limit              int
}

type QualityCheckReviewerUnit struct {
	Index                int      `json:"index"`
	UnitID               string   `json:"unit_id"`
	UnitKind             UnitKind `json:"unit_kind"`
	SelectedBatchID      string   `json:"selected_batch_id"`
	Attempt              int      `json:"attempt"`
	Source               string   `json:"source"`
	CurrentTarget        string   `json:"current_target"`
	SourceSHA256         string   `json:"source_sha256"`
	CandidateSHA256      string   `json:"candidate_sha256"`
	ValidationSHA256     string   `json:"validation_sha256"`
	RelevantManifestPath string   `json:"relevant_manifest_path"`
	RelevantInputPath    string   `json:"relevant_input_path"`
	RelevantInputSHA256  string   `json:"relevant_input_sha256"`
}

type QualityCheckReviewerContext struct {
	SchemaVersion      int                        `json:"schema_version"`
	Kind               string                     `json:"kind"`
	Locale             string                     `json:"locale"`
	SnapshotID         string                     `json:"snapshot_id"`
	PreviousSnapshotID string                     `json:"previous_snapshot_id,omitempty"`
	Rubric             string                     `json:"rubric"`
	Instruction        string                     `json:"instruction"`
	WorkingSetKind     UnitKind                   `json:"working_set_kind"`
	StartIndex         int                        `json:"start_index"`
	EndIndex           int                        `json:"end_index"`
	UnitCount          int                        `json:"unit_count"`
	Scope              QualityCheckScope          `json:"scope"`
	Units              []QualityCheckReviewerUnit `json:"units"`
}

type QualityCheckReviewerBundleManifest struct {
	SchemaVersion       int                   `json:"schema_version"`
	Kind                string                `json:"kind"`
	Locale              string                `json:"locale"`
	SnapshotID          string                `json:"snapshot_id"`
	PreviousSnapshotID  string                `json:"previous_snapshot_id,omitempty"`
	WorkingSetKind      UnitKind              `json:"working_set_kind"`
	StartIndex          int                   `json:"start_index"`
	EndIndex            int                   `json:"end_index"`
	UnitCount           int                   `json:"unit_count"`
	InputIdentitySHA256 string                `json:"input_identity_sha256"`
	Files               []TransportBundleFile `json:"files"`
}

func ExportQualityCheckReviewerBundle(root string, catalog *Catalog, options QualityCheckReviewerBundleOptions) ([]byte, QualityCheckReviewerBundleManifest, error) {
	if catalog == nil {
		return nil, QualityCheckReviewerBundleManifest{}, errors.New("quality-check reviewer bundle catalog is required")
	}
	if options.Limit == 0 {
		options.Limit = DefaultQualityCheckBatchLimit
	}
	if options.Limit < 1 || options.Limit > DefaultQualityCheckBatchLimit {
		return nil, QualityCheckReviewerBundleManifest{}, fmt.Errorf("quality-check reviewer bundle limit must be 1..%d", DefaultQualityCheckBatchLimit)
	}
	// Export and canonical current-check resolve the same complete package from
	// the exact successor Snapshot. This is selection only, never a saved PASS;
	// the current scope, frozen predecessor and canonical bytes are rebuilt below.
	scoped, _, err := selectSnapshotTranslationPackage(root, catalog, options.Locale, options.SnapshotID)
	if err != nil {
		return nil, QualityCheckReviewerBundleManifest{}, err
	}
	catalog = scoped
	evaluation := newQualityCheckEvaluation(root, options.Locale, catalog)
	scope, err := buildQualityCheckScopeWithEvaluation(evaluation, QualityCheckScopeOptions{
		Locale: options.Locale, SnapshotID: options.SnapshotID, PreviousSnapshotID: options.PreviousSnapshotID,
	})
	if err != nil {
		return nil, QualityCheckReviewerBundleManifest{}, err
	}
	selectedScope, err := selectQualityCheckReviewerWorkingSet(scope, options.StartIndex, options.Limit)
	if err != nil {
		return nil, QualityCheckReviewerBundleManifest{}, err
	}
	// The resolved (including persisted) lineage distinguishes initial QC from
	// successor re-QC; currentness still rebuilds this exact selection.
	if scope.PreviousSnapshotID == "" && len(catalog.Adapted) > 0 {
		ids := []string{}
		for i, u := range selectedScope {
			ids = append(ids, u.UnitID)
			if catalog.RequireInitialTranslationWorkingSet(ids) != nil {
				selectedScope = selectedScope[:i]
				break
			}
		}
	}
	kind := selectedScope[0].UnitKind

	snapshot, err := readQualityCheckSnapshotWithCompatibility(root, options.Locale, options.SnapshotID, true, catalog, evaluation.compatibility)
	if err != nil {
		return nil, QualityCheckReviewerBundleManifest{}, err
	}
	snapshotByID := make(map[string]QualityCheckSnapshotUnit, len(snapshot.Units))
	for _, unit := range snapshot.Units {
		snapshotByID[unit.UnitID] = unit
	}
	context := QualityCheckReviewerContext{
		SchemaVersion: QualityCheckReviewerBundleSchemaVersion,
		Kind:          "go-tour-i18n/translation-unit-quality-check-context",
		Locale:        options.Locale, SnapshotID: options.SnapshotID, PreviousSnapshotID: scope.PreviousSnapshotID,
		Rubric:         TranslationQualityRubric,
		Instruction:    "Return only one A/B/C/D rating and findings for each TranslationUnit. Do not generate replacement text.",
		WorkingSetKind: kind, StartIndex: selectedScope[0].Index,
		EndIndex: selectedScope[len(selectedScope)-1].Index, UnitCount: len(selectedScope), Scope: *scope,
		Units: make([]QualityCheckReviewerUnit, 0, len(selectedScope)),
	}

	type entry struct {
		path, repositoryPath string
		data                 []byte
	}
	entries := []entry{}
	seenFormal := map[string]bool{}
	addFormal := func(bundlePath, repositoryPath string, data []byte) {
		if !seenFormal[bundlePath] {
			seenFormal[bundlePath] = true
			entries = append(entries, entry{bundlePath, repositoryPath, data})
		}
	}
	// Membership is context for carry, never part of the fresh judgment set.
	// Canonical current-check rebuilds the same predecessor proofs independently.
	if baseline := evaluation.membership[options.SnapshotID]; baseline != nil {
		for _, ref := range baseline.contextReferences {
			data, err := readSnapshotReferencedFile(root, ref.Path)
			if err != nil || sum(data) != ref.SHA256 {
				return nil, QualityCheckReviewerBundleManifest{}, fmt.Errorf("membership QC context drift: %s", ref.Path)
			}
			addFormal("formal/membership-context/"+ref.SHA256+filepath.Ext(ref.Path), ref.Path, data)
		}
	}
	compatibilityRefs := map[string]GlossaryArchiveReference{}
	for _, carried := range scope.CarryForward {
		for _, ref := range carried.CompatibilityChain {
			if prior, exists := compatibilityRefs[ref.Path]; exists && prior != ref {
				return nil, QualityCheckReviewerBundleManifest{}, fmt.Errorf("conflicting carry-forward compatibility reference")
			}
			compatibilityRefs[ref.Path] = ref
		}
	}
	compatibilityPaths := make([]string, 0, len(compatibilityRefs))
	for path := range compatibilityRefs {
		compatibilityPaths = append(compatibilityPaths, path)
	}
	sort.Strings(compatibilityPaths)
	compatibilityInputs := map[string]GlossaryArchiveReference{}
	for _, path := range compatibilityPaths {
		ref := compatibilityRefs[path]
		evidence, data, ok := evaluation.compatibility.evidence(ref)
		if !ok {
			return nil, QualityCheckReviewerBundleManifest{}, fmt.Errorf("carry-forward compatibility evidence changed")
		}
		addFormal("formal/compatibility/"+ref.SHA256+".json", ref.Path, data)
		for _, input := range []GlossaryArchiveReference{evidence.OldGlossary, evidence.NewGlossary, evidence.NewFullReview} {
			if prior, exists := compatibilityInputs[input.Path]; exists && prior != input {
				return nil, QualityCheckReviewerBundleManifest{}, fmt.Errorf("conflicting compatibility input reference")
			}
			compatibilityInputs[input.Path] = input
		}
	}
	inputPaths := make([]string, 0, len(compatibilityInputs))
	for path := range compatibilityInputs {
		inputPaths = append(inputPaths, path)
	}
	sort.Strings(inputPaths)
	for _, path := range inputPaths {
		input := compatibilityInputs[path]
		data, err := readCompatibilityFile(root, input.Path)
		if err != nil || sum(data) != input.SHA256 {
			return nil, QualityCheckReviewerBundleManifest{}, fmt.Errorf("compatibility input changed")
		}
		addFormal("formal/compatibility/"+input.SHA256+filepath.Ext(input.Path), input.Path, data)
	}
	attention, err := NewLanguageAttention(root, options.Locale)
	if err != nil {
		return nil, QualityCheckReviewerBundleManifest{}, err
	}
	auditGlossary, err := LoadGlossary(root, options.Locale)
	if err != nil {
		return nil, QualityCheckReviewerBundleManifest{}, err
	}
	for _, scopeUnit := range selectedScope {
		snapshotUnit := snapshotByID[scopeUnit.UnitID]
		evidence, err := evaluation.readSnapshotUnitRepositoryEvidence(options.SnapshotID, snapshotUnit)
		if err != nil {
			return nil, QualityCheckReviewerBundleManifest{}, fmt.Errorf("snapshot index %d (%s): %w", scopeUnit.Index, scopeUnit.UnitID, err)
		}
		if evidence.machine != nil {
			for _, ref := range machineAuthorityReferences(evidence.machine) {
				data, err := readSnapshotReferencedFile(root, ref.Path)
				if err != nil || sum(data) != ref.SHA256 {
					return nil, QualityCheckReviewerBundleManifest{}, fmt.Errorf("machine reconciliation reviewer authority drift: %s", ref.Path)
				}
				addFormal("formal/representation-authority/"+ref.SHA256+filepath.Ext(ref.Path), ref.Path, data)
			}
		}
		manifestRepo := filepath.ToSlash(filepath.Join("data", "retranslation-runs", options.Locale, snapshotUnit.SelectedBatchID, "manifest.json"))
		manifestData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(manifestRepo)))
		if err != nil {
			return nil, QualityCheckReviewerBundleManifest{}, err
		}
		manifestBundle := filepath.ToSlash(filepath.Join("formal", "batches", snapshotUnit.SelectedBatchID, "manifest.json"))
		addFormal(manifestBundle, manifestRepo, manifestData)
		inputRepo := filepath.ToSlash(filepath.Join("data", "retranslation-runs", options.Locale, snapshotUnit.SelectedBatchID, evidence.record.InputPath))
		inputData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(inputRepo)))
		if err != nil {
			return nil, QualityCheckReviewerBundleManifest{}, err
		}
		if sum(inputData) != evidence.record.InputSHA256 {
			return nil, QualityCheckReviewerBundleManifest{}, fmt.Errorf("%s: relevant input hash mismatch", scopeUnit.UnitID)
		}
		inputBundle := filepath.ToSlash(filepath.Join("formal", "batches", snapshotUnit.SelectedBatchID, evidence.record.InputPath))
		addFormal(inputBundle, inputRepo, inputData)
		validationData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(snapshotUnit.ValidationPath)))
		if err != nil {
			return nil, QualityCheckReviewerBundleManifest{}, err
		}
		addFormal(filepath.ToSlash(filepath.Join("formal", "validation", fmt.Sprintf("%03d-%s.json", snapshotUnit.Index, strings.ReplaceAll(snapshotUnit.UnitID, "/", "-")))), snapshotUnit.ValidationPath, validationData)
		result, err := TranslationUnitLanguageAudit(root, catalog, evidence.unit, options.Locale, evidence.candidate, auditGlossary)
		if err != nil {
			return nil, QualityCheckReviewerBundleManifest{}, err
		}
		if err := attention.Add("translation-unit", snapshotUnit.CandidatePath, evidence.unit.ID, evidence.unit.Kind, result); err != nil {
			return nil, QualityCheckReviewerBundleManifest{}, err
		}
		context.Units = append(context.Units, QualityCheckReviewerUnit{
			Index: snapshotUnit.Index, UnitID: snapshotUnit.UnitID, UnitKind: snapshotUnit.UnitKind,
			SelectedBatchID: snapshotUnit.SelectedBatchID, Attempt: snapshotUnit.Attempt,
			Source: string(evidence.unit.Source), CurrentTarget: string(evidence.candidate),
			SourceSHA256: snapshotUnit.SourceSHA256, CandidateSHA256: snapshotUnit.CandidateSHA256,
			ValidationSHA256:     snapshotUnit.ValidationSHA256,
			RelevantManifestPath: manifestBundle, RelevantInputPath: inputBundle, RelevantInputSHA256: evidence.record.InputSHA256,
		})
	}
	if err := attention.Verify(root); err != nil {
		return nil, QualityCheckReviewerBundleManifest{}, err
	}
	attentionData, err := json.MarshalIndent(attention, "", "  ")
	if err != nil {
		return nil, QualityCheckReviewerBundleManifest{}, err
	}
	entries = append(entries, entry{LanguageAttentionPath, "", append(attentionData, '\n')})
	policyBytes, err := readCompatibilityFile(root, attention.Policy.Path)
	if err != nil || sum(policyBytes) != attention.Policy.SHA256 {
		return nil, QualityCheckReviewerBundleManifest{}, fmt.Errorf("language policy changed")
	}
	entries = append(entries, entry{"formal/language-policy.json", attention.Policy.Path, policyBytes})
	contextData, err := json.MarshalIndent(context, "", "  ")
	if err != nil {
		return nil, QualityCheckReviewerBundleManifest{}, err
	}
	contextData = append(contextData, '\n')
	entries = append(entries, entry{"quality-check.json", "", contextData})
	snapshotRepo := filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", options.Locale, options.SnapshotID, "manifest.json"))
	snapshotData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(snapshotRepo)))
	if err != nil {
		return nil, QualityCheckReviewerBundleManifest{}, err
	}
	entries = append(entries, entry{"formal/snapshot-manifest.json", snapshotRepo, snapshotData})
	glossaryData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(snapshot.GlossaryPath)))
	if err != nil || sum(glossaryData) != snapshot.GlossarySHA256 {
		return nil, QualityCheckReviewerBundleManifest{}, fmt.Errorf("snapshot glossary identity mismatch")
	}
	entries = append(entries, entry{"formal/glossary.yaml", snapshot.GlossaryPath, glossaryData})
	for _, repoPath := range qualityCheckReviewerAuthorityPaths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(repoPath)))
		if err != nil {
			return nil, QualityCheckReviewerBundleManifest{}, fmt.Errorf("read Quality Check authority %s: %w", repoPath, err)
		}
		entries = append(entries, entry{filepath.ToSlash(filepath.Join("authority", repoPath)), repoPath, data})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })
	files := make([]TransportBundleFile, 0, len(entries))
	zipEntries := make([]TransportBundleEntry, 0, len(entries))
	var identity strings.Builder
	for _, item := range entries {
		file := NewTransportBundleFile(item.path, item.repositoryPath, item.data)
		files = append(files, file)
		zipEntries = append(zipEntries, TransportBundleEntry{Path: item.path, Data: item.data})
		identity.WriteString(file.BundlePath + "\x00" + file.SHA256 + "\x00")
	}
	manifest := QualityCheckReviewerBundleManifest{
		SchemaVersion: QualityCheckReviewerBundleSchemaVersion,
		Kind:          "go-tour-i18n/translation-unit-quality-check-reviewer-bundle",
		Locale:        options.Locale, SnapshotID: options.SnapshotID, PreviousSnapshotID: scope.PreviousSnapshotID,
		WorkingSetKind: kind, StartIndex: context.StartIndex, EndIndex: context.EndIndex,
		UnitCount: len(context.Units), InputIdentitySHA256: sum([]byte(identity.String())), Files: files,
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, QualityCheckReviewerBundleManifest{}, err
	}
	manifestData = append(manifestData, '\n')
	bundle, err := WriteDeterministicTransportBundle(manifestData, zipEntries)
	if err != nil {
		return nil, QualityCheckReviewerBundleManifest{}, err
	}
	if err := evaluation.membershipChecks(); err != nil {
		return nil, QualityCheckReviewerBundleManifest{}, err
	}
	return bundle, manifest, nil
}

func selectQualityCheckReviewerWorkingSet(scope *QualityCheckScope, requestedStart, limit int) ([]QualityCheckScopeUnit, error) {
	start := requestedStart
	if start == 0 {
		for _, unit := range scope.Pending {
			if unit.RequiredAction == QualityCheckActionRequired {
				start = unit.Index
				break
			}
		}
	}
	if start == 0 {
		return nil, errors.New("quality-check reviewer bundle has no reviewable pending TranslationUnits")
	}
	selected := make([]QualityCheckScopeUnit, 0, limit)
	var kind UnitKind
	started := false
	for _, unit := range scope.Pending {
		if !started {
			if unit.Index != start {
				continue
			}
			started = true
			if unit.RequiredAction != QualityCheckActionRequired {
				return nil, fmt.Errorf("snapshot index %d requires revision before Quality Check", start)
			}
			kind = unit.UnitKind
		}
		if unit.UnitKind != kind || len(selected) == limit {
			break
		}
		if unit.RequiredAction == QualityCheckActionRequired {
			selected = append(selected, unit)
		}
	}
	if !started || len(selected) == 0 {
		return nil, fmt.Errorf("start_index %d is not a pending Quality Check unit", start)
	}
	return selected, nil
}
