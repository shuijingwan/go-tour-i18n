package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func unboundPageFixture(t *testing.T, line, target string) (PageAdapter, PageAdapter, []byte) {
	t.Helper()
	source := []byte("# Example\n\n" + line + "\n")
	d, err := ParseWorkflowDocument(Source{Path: "_content/doc/unbound.md", Kind: "page", Route: "/doc/unbound", SHA256: digest(source)}, source, 1)
	if err != nil {
		t.Fatal(err)
	}
	old := *d
	old.Slots = append([]Unit(nil), d.Slots...)
	for i, s := range old.Slots {
		if string(source[s.Start:s.End]) == line {
			s.Source = line
			s.Protected = nil
			s.SourceSHA = unitSourceIdentity(line, nil)
			old.Slots[i] = s
		}
	}
	old.Units = old.Slots
	old, err = FormalPage(old, source, 1)
	if err != nil {
		t.Fatal(err)
	}
	return PageAdapter{old, source}, PageAdapter{*d, source}, []byte(strings.Replace(old.Units[0].Source, line, target, 1))
}

func TestDuplicateBindingsRestoreEqualityIsNotRevisionEquivalence(t *testing.T) {
	const line = "Package A depends on package B."
	const target = "包 A 依赖包 B；A 与 B 有不同记录。"
	old, current, raw := unboundPageFixture(t, line, target)
	if _, err := current.ReconcileRepresentation(old, raw); err == nil {
		t.Fatal("ambiguous binding accepted")
	}
	if _, err := current.ReconciliationQuestion(old, raw); err == nil {
		t.Fatal("semantic authority could select repeated occurrences")
	}
	before, err := old.Restore(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	tokens := map[string]string{}
	for _, p := range current.Document.Units[0].Protected {
		if p.Raw == "A" || p.Raw == "B" {
			tokens[p.Raw] = p.Token
		}
	}
	if len(tokens) != 2 {
		t.Fatal(tokens)
	}
	sourcePart := strings.NewReplacer("A", tokens["A"], "B", tokens["B"]).Replace(line)
	allowFirstAEdit := map[int]bool{}
	variants := 0
	for ai, ap := range exactLiteralSites([]byte(target), 0, len(target), "A") {
		for _, bp := range exactLiteralSites([]byte(target), 0, len(target), "B") {
			type substitution struct {
				at   int
				text string
			}
			patches := []substitution{{ap, tokens["A"]}, {bp, tokens["B"]}}
			sort.Slice(patches, func(i, j int) bool { return patches[i].at > patches[j].at })
			part := target
			for _, p := range patches {
				part = part[:p.at] + p.text + part[p.at+1:]
			}
			variant := strings.Replace(current.Document.Units[0].Source, sourcePart, part, 1)
			after, err := current.Restore(variant)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("fixture must restore identical bytes", err)
			}
			if err = current.Validate(after, &i18n.Glossary{}); err != nil {
				t.Fatal(err)
			}
			// The SAME rendered first-A edit is legal only when the first A is
			// unprotected. Therefore equal restore/validator results do not define
			// an equivalence relation on future exact revision permissions.
			var edited string
			if ai == 0 {
				edited = strings.Replace(variant, tokens["A"], "C", 1)
			} else {
				edited = strings.Replace(variant, "A", "C", 1)
			}
			changed, err := current.Restore(edited)
			allowed := err == nil
			if allowed {
				if err = current.Validate(changed, &i18n.Glossary{}); err != nil {
					t.Fatal(err)
				}
			}
			allowFirstAEdit[ai] = allowed
			variants++
		}
	}
	if variants != 4 || allowFirstAEdit[0] || !allowFirstAEdit[1] {
		t.Fatal("revision permission counterexample missing", variants, allowFirstAEdit)
	}
	proof, scaffold, err := current.UnbindableRevision(old, raw)
	if err != nil || len(proof.Intervals) != 1 {
		t.Fatal(proof, err)
	}
	restored, err := current.Restore(string(scaffold))
	if err != nil || bytes.Equal(before, restored) || !bytes.Contains(restored, []byte(line)) {
		t.Fatal("scaffold mislabeled accepted candidate", err)
	}
	if !bytes.Equal(before, bytes.Replace(restored, []byte(line), []byte(target), 1)) {
		t.Fatal("outside-scope translation changed")
	}
	for _, tc := range []struct{ line, target string }{{"Custom Marshal and Unmarshal", "自定义编组与解组"}, {"Return 404 or 410.", "返回 404 或 410。"}, {"Package A depends on package A.", "A、A、A"}} {
		old, cur, raw := unboundPageFixture(t, tc.line, tc.target)
		if _, _, err := cur.UnbindableRevision(old, raw); err == nil {
			t.Fatal("non-applicable/semantically distinct machine authority accepted", tc.line)
		}
	}
}

func machineRevisionFixture(t *testing.T) (string, *i18n.Catalog, i18n.RepresentationRevisionOptions) {
	t.Helper()
	const line = "Package A depends on package B."
	const target = "包 A 依赖包 B；A 与 B 有不同记录。"
	root, _, current, frozen := replacementPageFixtureSize(t, 3, 2, func(i int, b []byte) []byte {
		if i == 3 {
			return []byte("# Example\n\n" + line + "\n")
		}
		return b
	})
	oldDoc := frozen.Documents[2]
	oldDoc.Slots = append([]Unit(nil), oldDoc.Slots...)
	unit, _ := current.Unit(current.TranslationScope[2])
	a := unit.Adapter.(PageAdapter)
	for i, s := range oldDoc.Slots {
		if string(a.Source[s.Start:s.End]) == line {
			s.Source = line
			s.Protected = nil
			s.SourceSHA = unitSourceIdentity(line, nil)
			oldDoc.Slots[i] = s
		}
	}
	oldDoc.Units = oldDoc.Slots
	var err error
	oldDoc, err = FormalPage(oldDoc, a.Source, 3)
	if err != nil {
		t.Fatal(err)
	}
	frozen.Documents[2] = oldDoc
	old := &i18n.Catalog{}
	pkg := i18n.TranslationPackageDescriptor{ID: frozen.Package, AuthoritySHA256: frozen.PackageSHA256}
	for i, b := range current.TranslationPackages[0].Units {
		b.Unit.ID = frozen.Documents[i].Units[0].ID
		b.Adapter = PageAdapter{frozen.Documents[i], b.Unit.Source}
		pkg.Units = append(pkg.Units, b)
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
	for _, out := range m.ExpectedOutputs {
		outputs[out.UnitID] = bytes.Replace(files["formal/inputs/"+filepath.Base(out.BundlePath)], []byte(line), []byte(target), 1)
	}
	surfaceImport(t, root, old, b, outputs)
	if p, err := i18n.ProcessRetranslationBatch(root, old, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID}); err != nil || p.ValidationPassed != 3 {
		t.Fatal(p, err)
	}
	const snapshot = "machine-revision-old"
	if _, _, err = i18n.CreateQualityCheckCandidateSnapshot(root, old, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: snapshot}); err != nil {
		t.Fatal(err)
	}
	if _, err = i18n.RecordQualityCheckResults(root, old, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: snapshot, ReviewerSession: "synthetic-independent-reviewer", UnitIDs: old.TranslationScope, Rating: "A"}); err != nil {
		t.Fatal(err)
	}
	historical := freezeReplacementFixture(t, root, frozen)
	mapping, err := resolvePageReplacements(root, current, historical)
	if err != nil {
		t.Fatal(err)
	}
	rows := []map[string]any{}
	for _, id := range old.TranslationScope[2:] {
		p := mapping.Units[id]
		rows = append(rows, map[string]any{"current_unit_id": p.CurrentUnitID, "snapshot_representation_unit_id": p.OldUnitID, "source_path": p.SourcePath, "source_sha256": p.SourceSHA256, "labels": map[string]bool{"representation_reconciliation_required": true, "true_language_defect": false, "unknown": id == old.TranslationScope[2]}})
	}
	impact, _ := json.Marshal(map[string]any{"units": rows})
	writeFixture(t, root, "impact.json", impact)
	snapshotBytes, _ := readRegular(root, "data/quality-check-snapshots/zh-CN/"+snapshot+"/manifest.json")
	return root, current, i18n.RepresentationRevisionOptions{UnitID: current.TranslationScope[2], ReopenID: "machine-unbindable-revision", Reconciliation: i18n.RepresentationReconciliationOptions{Locale: "zh-CN", Package: pkg.ID, ReconciliationID: "machine-successors", PreviousSnapshotID: snapshot, SnapshotSHA256: digest(snapshotBytes), Impact: i18n.GlossaryArchiveReference{Path: "impact.json", SHA256: digest(impact)}, HistoricalDocuments: historical}}
}

func TestRepresentationMachineRevisionExportContract(t *testing.T) {
	root, c, o := machineRevisionFixture(t)
	qaPath := "data/quality-check-snapshots/zh-CN/" + o.Reconciliation.PreviousSnapshotID + "/quality-check-results.json"
	qaBefore, _ := readRegular(root, qaPath)
	for _, name := range []string{"old-id", "other-unit", "snapshot", "impact", "frozen", "semantic-receipt"} {
		t.Run(name, func(t *testing.T) {
			bad := o
			switch name {
			case "old-id":
				b, _ := readRegular(root, o.Reconciliation.HistoricalDocuments.Path)
				var f HistoricalPageRepresentations
				json.Unmarshal(b, &f)
				bad.UnitID = f.Documents[2].Units[0].ID
			case "other-unit":
				bad.UnitID = c.TranslationScope[0]
			case "snapshot":
				bad.Reconciliation.SnapshotSHA256 = strings.Repeat("0", 64)
			case "impact":
				bad.Reconciliation.Impact.SHA256 = strings.Repeat("0", 64)
			case "frozen":
				bad.Reconciliation.HistoricalDocuments.SHA256 = strings.Repeat("0", 64)
			case "semantic-receipt":
				bad.Reconciliation.LanguageReviewIDs = []string{"not-machine-authority"}
			}
			if _, _, err := i18n.PrepareRepresentationRevision(root, c, bad); err == nil {
				t.Fatal("invalid machine revision authority accepted")
			}
		})
	}
	r, path, err := i18n.PrepareRepresentationRevision(root, c, o)
	if err != nil {
		t.Fatal(err)
	}
	if r.AuthoritySource != i18n.RepresentationMachineAuthority || r.Units[0].PreviousRating != "A" || !strings.Contains(r.Units[0].Finding, "representation-unbindable") {
		t.Fatal(r)
	}
	if _, err = os.Lstat(filepath.Join(root, path)); !os.IsNotExist(err) {
		t.Fatal("readonly wrote receipt")
	}
	o.Record = true
	if _, _, err = i18n.PrepareRepresentationRevision(root, c, o); err != nil {
		t.Fatal(err)
	}
	if _, _, err = i18n.PrepareRepresentationRevision(root, c, o); err == nil {
		t.Fatal("record overwrite accepted")
	}
	if _, err = i18n.CheckQualityCheckSurfaceReopen(root, c, "zh-CN", o.ReopenID); err != nil {
		t.Fatal(err)
	}
	receiptBytes, _ := readRegular(root, path)
	for _, name := range []string{"finding", "scaffold", "missing-proof", "unit"} {
		t.Run("tamper-"+name, func(t *testing.T) {
			var bad i18n.QualityCheckSurfaceReopen
			json.Unmarshal(receiptBytes, &bad)
			switch name {
			case "finding":
				bad.Units[0].Finding = "arbitrary translation request"
			case "scaffold":
				bad.ReplacementContext.Machine.ScaffoldSHA256 = strings.Repeat("0", 64)
			case "missing-proof":
				bad.ReplacementContext.Machine = nil
			case "unit":
				bad.Units[0].UnitID = c.TranslationScope[0]
			}
			data, _ := json.Marshal(bad)
			writeFixture(t, root, path, data)
			defer writeFixture(t, root, path, receiptBytes)
			if _, err := i18n.CheckQualityCheckSurfaceReopen(root, c, "zh-CN", o.ReopenID); err == nil {
				t.Fatal("tampered machine authority accepted")
			}
		})
	}
	revision, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: []string{o.UnitID}, UnitKind: i18n.UnitKindPage, BatchID: "chatgpt-zh-CN-002", AllowReexport: true, PreviousSnapshotID: o.Reconciliation.PreviousSnapshotID, SurfaceReopenID: o.ReopenID})
	if err != nil {
		t.Fatal(err)
	}
	zip, m, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: revision.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	files, err := i18n.ReadTransportBundle(zip, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	if len(files["formal/representation-revision-scope.json"]) == 0 || !bytes.Contains(files["generation-context.json"], []byte("INCOMPLETE revision scaffold")) {
		t.Fatal("unbound scaffold not explicitly identified")
	}
	baseline := files["formal/protected-candidates/"+filepath.Base(m.ExpectedOutputs[0].BundlePath)]
	unit, _ := c.Unit(o.UnitID)
	candidate, err := unit.Adapter.Restore(string(baseline))
	if err != nil || !bytes.Contains(candidate, []byte("Package A depends on package B.")) {
		t.Fatal("invalid source-only scaffold", err)
	}
	qaAfter, _ := readRegular(root, qaPath)
	if !bytes.Equal(qaBefore, qaAfter) {
		t.Fatal("machine finding rewrote language judgment")
	}
	if _, err = os.Stat(filepath.Join(root, revision.BatchPath, "result.json")); !os.IsNotExist(err) {
		t.Fatal("fake Generation result")
	}
	// Explicit synthetic provider output in TempDir, separate from the machine
	// authority operation above. No real Generation or Reviewer is invoked.
	output := files["formal/inputs/"+filepath.Base(m.ExpectedOutputs[0].BundlePath)]
	surfaceImport(t, root, c, zip, map[string][]byte{o.UnitID: output})
	if p, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: revision.BatchID}); err != nil || p.ValidationPassed != 1 {
		t.Fatal(p, err)
	}
	successor, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "machine-revision-successor"})
	if err != nil {
		t.Fatal(err)
	}
	scope, err := i18n.BuildQualityCheckScope(root, c, i18n.QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: successor.SnapshotID, PreviousSnapshotID: o.Reconciliation.PreviousSnapshotID})
	if err != nil || scope.ACount != 2 || scope.PendingCount != 1 || scope.Pending[0].UnitID != o.UnitID {
		t.Fatal("new model output inherited predecessor A", scope, err)
	}
	qaAfter, _ = readRegular(root, qaPath)
	if !bytes.Equal(qaBefore, qaAfter) {
		t.Fatal("old A changed after synthetic revision")
	}
}
