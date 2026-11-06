package sitecontent

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func semanticReviewFixture(t *testing.T) (string, *i18n.Catalog, i18n.RepresentationReviewOptions) {
	t.Helper()
	root, _, current, frozen := replacementPageFixtureSize(t, 4, 2, func(i int, b []byte) []byte {
		if i == 3 {
			return []byte("# Custom Marshal and Unmarshal\n\nRead `API` [the manual](https://example.com).\n")
		}
		return b
	})
	// Freeze a historical parser in which this prose identifier was editable.
	before := frozen.Documents[2]
	before.Slots = append([]Unit(nil), before.Slots...)
	for i, s := range before.Slots {
		keep := []ProtectedToken{}
		for _, p := range s.Protected {
			if p.Raw == "Marshal" {
				s.Source = strings.ReplaceAll(s.Source, p.Token, p.Raw)
			} else {
				keep = append(keep, p)
			}
		}
		s.Protected = keep
		s.SourceSHA = unitSourceIdentity(s.Source, keep)
		before.Slots[i] = s
	}
	before.Units = before.Slots
	unit, _ := current.Unit(current.TranslationScope[2])
	a := unit.Adapter.(PageAdapter)
	var err error
	before, err = FormalPage(before, a.Source, 3)
	if err != nil {
		t.Fatal(err)
	}
	frozen.Documents[2] = before
	old := &i18n.Catalog{}
	pkg := i18n.TranslationPackageDescriptor{ID: frozen.Package, AuthoritySHA256: frozen.PackageSHA256}
	for i, binding := range current.TranslationPackages[0].Units {
		binding.Unit.ID = frozen.Documents[i].Units[0].ID
		binding.Adapter = PageAdapter{frozen.Documents[i], binding.Unit.Source}
		pkg.Units = append(pkg.Units, binding)
	}
	if err = old.RegisterTranslationPackage(pkg); err != nil {
		t.Fatal(err)
	}
	if err = old.UseTranslationPackage(pkg.ID); err != nil {
		t.Fatal(err)
	}
	AttachPageReplacementResolver(old)
	e, err := i18n.ExportRetranslationBatch(root, old, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: old.TranslationScope})
	if err != nil {
		t.Fatal(err)
	}
	b, m, err := i18n.ExportTranslationUnitGenerationBundle(root, old, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	files, err := i18n.ReadTransportBundle(b, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	outputs := map[string][]byte{}
	for _, o := range m.ExpectedOutputs {
		outputs[o.UnitID] = bytes.ReplaceAll(files["formal/inputs/"+filepath.Base(o.BundlePath)], []byte("Custom Marshal and Unmarshal"), []byte("自定义编组与解组"))
	}
	surfaceImport(t, root, old, b, outputs)
	if p, err := i18n.ProcessRetranslationBatch(root, old, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID}); err != nil || p.ValidationPassed != 4 {
		t.Fatal(p, err)
	}
	const snapshot = "semantic-predecessor"
	if _, _, err = i18n.CreateQualityCheckCandidateSnapshot(root, old, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: snapshot}); err != nil {
		t.Fatal(err)
	}
	if _, err = i18n.RecordQualityCheckResults(root, old, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: snapshot, ReviewerSession: "synthetic-original-reviewer", UnitIDs: old.TranslationScope, Rating: "A"}); err != nil {
		t.Fatal(err)
	}
	frozenRef := freezeReplacementFixture(t, root, frozen)
	mapping, err := resolvePageReplacements(root, current, frozenRef)
	if err != nil {
		t.Fatal(err)
	}
	rows := []map[string]any{}
	for _, id := range old.TranslationScope[2:] {
		p := mapping.Units[id]
		rows = append(rows, map[string]any{"current_unit_id": p.CurrentUnitID, "snapshot_representation_unit_id": p.OldUnitID, "source_path": p.SourcePath, "source_sha256": p.SourceSHA256, "labels": map[string]bool{"representation_reconciliation_required": true, "true_language_defect": false, "unknown": p.OldUnitID == old.TranslationScope[2]}})
	}
	impact, _ := json.Marshal(map[string]any{"units": rows})
	writeFixture(t, root, "impact.json", impact)
	snapshotBytes, _ := readRegular(root, "data/quality-check-snapshots/zh-CN/"+snapshot+"/manifest.json")
	return root, current, i18n.RepresentationReviewOptions{UnitID: current.TranslationScope[2], ReviewID: "synthetic-semantic-review", Reconciliation: i18n.RepresentationReconciliationOptions{Locale: "zh-CN", Package: pkg.ID, ReconciliationID: "semantic-machine-successors", PreviousSnapshotID: snapshot, SnapshotSHA256: digest(snapshotBytes), Impact: i18n.GlossaryArchiveReference{Path: "impact.json", SHA256: digest(impact)}, HistoricalDocuments: frozenRef}}
}

func TestRepresentationSemanticReviewAuthority(t *testing.T) {
	root, c, o := semanticReviewFixture(t)
	canonicalUnit, err := c.Unit(o.UnitID)
	if err != nil {
		t.Fatal(err)
	}
	canonicalInput, _, err := canonicalUnit.Adapter.ProtectedInput(nil)
	if err != nil {
		t.Fatal(err)
	}
	qcPath := "data/quality-check-snapshots/zh-CN/" + o.Reconciliation.PreviousSnapshotID + "/quality-check-results.json"
	oldQC, err := readRegular(root, qcPath)
	if err != nil {
		t.Fatal(err)
	}
	oldAttempts, err := os.ReadDir(filepath.Join(root, "data/retranslation-runs/zh-CN"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		after, _ := readRegular(root, qcPath)
		if !bytes.Equal(oldQC, after) {
			t.Error("semantic authority created/modified a language rating")
		}
		attempts, _ := os.ReadDir(filepath.Join(root, "data/retranslation-runs/zh-CN"))
		if len(attempts) != len(oldAttempts) {
			t.Error("semantic authority created Generation provenance")
		}
	}()
	plan, err := i18n.ReconcileTranslationUnitRepresentations(root, c, o.Reconciliation)
	if err != nil || plan.Ready != 1 || plan.Blocked != 1 || plan.Applied {
		t.Fatal(plan, err)
	}
	b, m, err := i18n.ExportRepresentationReviewBundle(root, c, o)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = i18n.CheckRepresentationReviewBundle(root, c, b); err != nil {
		t.Fatal(err)
	}
	if m.Question.SourceText != "Custom Marshal and Unmarshal" || m.Question.ReviewedTarget != "自定义编组与解组" {
		t.Fatal(m.Question)
	}
	files, err := i18n.ReadTransportBundle(b, 512, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"predecessor-source.txt", "current-source.txt", "old-representation.json", "current-representation.json", "changed-representation.json", "all-representation-changes.json", "question.json", "formal-impact-record.json", "machine-evidence.json", "REVIEWER.md", "decision-template.json"} {
		if len(files[name]) == 0 {
			t.Fatal("missing self-contained input", name)
		}
	}
	provenanceFound := false
	for _, f := range m.Files {
		if strings.HasSuffix(f.RepositoryPath, "generation-imports/initial-manifest.json") {
			var provenance i18n.GenerationResultBundleManifest
			if err = json.Unmarshal(files[f.BundlePath], &provenance); err != nil {
				t.Fatal(err)
			}
			if provenance.Provider != "chatgpt" || provenance.Model != i18n.FormalChatGPTGenerationModel {
				t.Fatal("original provider/model not preserved", provenance)
			}
			provenanceFound = true
		}
	}
	if !provenanceFound {
		t.Fatal("existing original Generation provenance missing from transport")
	}
	decision := i18n.RepresentationReviewDecision{ReviewID: o.ReviewID, BundleIdentity: m.Identity, BundleSHA256: digest(b), Locale: o.Reconciliation.Locale, Package: o.Reconciliation.Package, OldUnitID: m.Replacement.OldUnitID, CurrentUnitID: o.UnitID, PreviousSnapshotID: o.Reconciliation.PreviousSnapshotID, FormalImpactSHA256: o.Reconciliation.Impact.SHA256, ReviewerSession: "synthetic-independent-semantic-reviewer", Provider: "chatgpt", Model: i18n.FormalChatGPTGenerationModel, Decision: "APPROVE_RECONCILIATION", Finding: "Synthetic fixture decision only; preserve the entire reviewed concept heading.", GenerationIndependent: true}
	for _, name := range []string{"bundle-identity", "bundle-sha", "old-unit", "current-unit", "snapshot", "impact", "A", "B", "C", "D", "replacement", "session", "independence", "provider", "model"} {
		t.Run(name, func(t *testing.T) {
			d := decision
			switch name {
			case "bundle-identity":
				d.BundleIdentity = strings.Repeat("0", 64)
			case "bundle-sha":
				d.BundleSHA256 = strings.Repeat("0", 64)
			case "old-unit":
				d.OldUnitID = o.UnitID
			case "current-unit":
				d.CurrentUnitID = decision.OldUnitID
			case "snapshot":
				d.PreviousSnapshotID = "wrong"
			case "impact":
				d.FormalImpactSHA256 = strings.Repeat("0", 64)
			case "A", "B", "C", "D":
				d.Decision = name
			case "session":
				d.ReviewerSession = ""
			case "independence":
				d.GenerationIndependent = false
			case "provider":
				d.Provider = "codex"
			case "model":
				d.Model = "unknown"
			}
			e, _ := json.Marshal(d)
			if name == "replacement" {
				var v map[string]any
				json.Unmarshal(e, &v)
				v["replacement"] = "not permitted"
				e, _ = json.Marshal(v)
			}
			if _, _, err := i18n.RecordRepresentationReview(root, c, b, e); err == nil {
				t.Fatal("invalid semantic authority accepted")
			}
			if _, err := os.Lstat(filepath.Join(root, "data/representation-reconciliation-reviews/zh-CN", o.ReviewID)); !os.IsNotExist(err) {
				t.Fatal("rejected evidence mutated formal state", err)
			}
		})
	}
	e, _ := json.Marshal(decision)
	r, _, err := i18n.RecordRepresentationReview(root, c, b, e)
	if err != nil {
		t.Fatal(err)
	}
	if r.Decision.Decision != "APPROVE_RECONCILIATION" {
		t.Fatal(r)
	}
	afterInput, _, err := canonicalUnit.Adapter.ProtectedInput(nil)
	if err != nil || afterInput != canonicalInput {
		t.Fatal("semantic review changed canonical Generation representation", err)
	}
	if _, err := os.Stat(pathJoin(root, ShellRepresentationCurrent)); !os.IsNotExist(err) {
		t.Fatal("semantic approval created canonical representation pointer")
	}
	if _, _, err = i18n.RecordRepresentationReview(root, c, b, e); err == nil {
		t.Fatal("duplicate record accepted")
	}
	options := o.Reconciliation
	options.LanguageReviewIDs = []string{o.ReviewID}
	plan, err = i18n.ReconcileTranslationUnitRepresentations(root, c, options)
	if err != nil || plan.Ready != 2 || plan.Blocked != 0 || plan.Applied {
		t.Fatal(plan, err)
	}
	for _, u := range plan.Receipt.Units {
		if u.Replacement.CurrentUnitID == o.UnitID && (u.SemanticAuthority == nil || u.UnknownResolvedMachine) {
			t.Fatal("semantic proof conflated with machine proof", u)
		}
	}
	// No processed successor, new rating, Generation attempt or apply is created
	// by record/preflight. Only two immutable semantic authority files + ZIP.
	p, err := i18n.BuildTranslationUnitProgress(root, c, "zh-CN")
	if err != nil || p.Missing != 2 {
		t.Fatal(p, err)
	}
	// Separate frozen synthetic copy exercises the approved reader, without
	// consuming or mutating the negative-decision fixture below.
	approvedRoot := t.TempDir()
	if err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		writeFixture(t, approvedRoot, rel, data)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	apply := options
	apply.Apply = true
	if result, err := i18n.ReconcileTranslationUnitRepresentations(approvedRoot, c, apply); err != nil || !result.Applied {
		t.Fatal(result, err)
	}
	if progress, err := i18n.BuildTranslationUnitProgress(approvedRoot, c, "zh-CN"); err != nil || progress.Passed != 4 {
		t.Fatal("approved semantic evidence not consumed by shared reader", progress, err)
	}
	// A negative formal decision is separately immutable, and keeps exact finding.
	negative := o
	negative.ReviewID = "synthetic-requires-language-revision"
	nb, nm, err := i18n.ExportRepresentationReviewBundle(root, c, negative)
	if err != nil {
		t.Fatal(err)
	}
	nd := decision
	nd.ReviewID = negative.ReviewID
	nd.BundleIdentity = nm.Identity
	nd.BundleSHA256 = digest(nb)
	nd.Decision = "REQUIRE_LANGUAGE_REVISION"
	nd.Finding = "Exact synthetic revision finding."
	ne, _ := json.Marshal(nd)
	if _, _, err = i18n.RecordRepresentationReview(root, c, nb, ne); err != nil {
		t.Fatal(err)
	}
	options.LanguageReviewIDs = []string{negative.ReviewID}
	plan, err = i18n.ReconcileTranslationUnitRepresentations(root, c, options)
	if err != nil || plan.Blocked != 1 || !strings.Contains(plan.Rows[0].Reason, nd.Finding) {
		t.Fatal(plan, err)
	}
	// Changed source, target or receipt context invalidates authority; hashes are
	// checked again from archived exact response bytes, not a manually set flag.
	if err = os.Chmod(filepath.Join(root, r.Evidence.Path), 0644); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, r.Evidence.Path, []byte(`{"decision":"APPROVE_RECONCILIATION"}`))
	options.LanguageReviewIDs = []string{o.ReviewID}
	plan, err = i18n.ReconcileTranslationUnitRepresentations(root, c, options)
	if err != nil || plan.Blocked != 1 {
		t.Fatal(plan, err)
	}
}
