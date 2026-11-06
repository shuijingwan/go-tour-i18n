package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCurrentValidationReopenTourNativeSelector(t *testing.T) {
	c := retranslationTestCatalog(3)
	c.Examples = []Example{
		retranslationTestExample("native/example", "_content/tour/example.go", "package p\n// Return the result.\nfunc result() {}\n"),
		retranslationTestExample("native/code-only", "_content/tour/code.go", "package p\nvar value int\n"),
	}
	pkg, authority, scope, units, err := translationAuthorizationScope(c)
	if err != nil || pkg != tourNativeTranslationPackage || len(units) != 4 {
		t.Fatal("native complete Page/eligible Example fixture", err)
	}
	selected, err := SelectCurrentValidationReopenPackage(c, tourNativeTranslationPackage)
	if err != nil {
		t.Fatal(err)
	}
	p, a, s, ordered, err := translationAuthorizationScope(selected)
	if err != nil || p != pkg || a != authority || s != scope || !reflect.DeepEqual(units, ordered) || selected == c || len(selected.TranslationScope) != 0 || len(c.TranslationScope) != 0 || len(c.TranslationPackages) != 0 || len(selected.TranslationPackages) != 0 {
		t.Fatal("native selection changed scope/authority or fabricated registration", err)
	}
	if err := selected.UseTranslationPackage(tourNativeTranslationPackage); err == nil || !strings.Contains(err.Error(), "unregistered translation package") {
		t.Fatal("native Tour became a registry-backed package", err)
	}

	// Additional packages coexist without becoming part of the native default.
	_, registered, _ := futureContentFixtureSize(t, 1)
	d := registered.TranslationPackages[0]
	d.AuthoritySHA256 = sum([]byte("synthetic registered source authority"))
	if err := c.RegisterTranslationPackage(d); err != nil {
		t.Fatal(err)
	}
	selected, err = SelectCurrentValidationReopenPackage(c, tourNativeTranslationPackage)
	if err != nil {
		t.Fatal(err)
	}
	p, a, s, ordered, err = translationAuthorizationScope(selected)
	if err != nil || p != pkg || a != authority || s != scope || !reflect.DeepEqual(units, ordered) {
		t.Fatal("registered inventory polluted native default scope", err)
	}
	t.Run("registered-package", func(t *testing.T) {
		scoped, err := SelectCurrentValidationReopenPackage(c, d.ID)
		if err != nil || len(c.TranslationScope) != 0 {
			t.Fatal(err)
		}
		p, a, _, ordered, err := translationAuthorizationScope(scoped)
		if err != nil || p != d.ID || a != d.AuthoritySHA256 || len(ordered) != len(d.Units) {
			t.Fatal("registered complete descriptor path changed", err)
		}
		if _, err := SelectCurrentValidationReopenPackage(scoped, d.ID); err != nil {
			t.Fatal("exact already-selected registered scope rejected", err)
		}
	})
	ids := []string{}
	for _, u := range units {
		ids = append(ids, u.ID)
	}
	for name, explicit := range map[string][]string{
		"partial": ids[:1], "complete-explicit": ids,
		"different-order": {ids[1], ids[0], ids[2], ids[3]},
		"foreign-package": {d.Units[0].Unit.ID},
		"mixed":           {ids[0], d.Units[0].Unit.ID},
	} {
		t.Run(name, func(t *testing.T) {
			partial := *c
			if err := partial.UseTranslationUnitScope(explicit); err != nil {
				t.Fatal(err)
			}
			before := append([]string{}, partial.TranslationScope...)
			if _, err := SelectCurrentValidationReopenPackage(&partial, tourNativeTranslationPackage); err == nil || !strings.Contains(err.Error(), "explicit scope cannot be widened") {
				t.Fatal("explicit partial/different scope reinterpreted as native default", err)
			}
			if !reflect.DeepEqual(before, partial.TranslationScope) {
				t.Fatal("caller scope mutated on rejection")
			}
		})
	}
	for _, wrong := range []string{"", "unknown-package"} {
		if _, err := SelectCurrentValidationReopenPackage(c, wrong); err == nil {
			t.Fatal("wrong/unregistered package accepted", wrong)
		}
	}
	if _, err := SelectCurrentValidationReopenPackage(nil, tourNativeTranslationPackage); err == nil {
		t.Fatal("nil catalog accepted")
	}
	if _, err := SelectCurrentValidationReopenPackage(&Catalog{}, tourNativeTranslationPackage); err == nil {
		t.Fatal("empty inventory accepted as complete native scope")
	}
	shadow := *c
	foreign := d.Units[0]
	foreign.Unit.ID = ids[0]
	shadow.Adapted = append(append([]AdaptedTranslationUnit{}, c.Adapted...), foreign)
	if _, err := SelectCurrentValidationReopenPackage(&shadow, tourNativeTranslationPackage); err == nil || !strings.Contains(err.Error(), "foreign adapter") {
		t.Fatal("foreign adapter masqueraded as native Unit", err)
	}
}

func TestCurrentValidationReopenTourNativeAuthority(t *testing.T) {
	const locale, snapshot = "zh-CN", "native-before-audit-drift"
	// The initialized shared fixture has explicit locale/glossary/machine
	// authority. Use a fresh native Catalog, not its registered card package.
	root, _, _ := futureContentFixtureSize(t, 1)
	c := retranslationTestCatalog(2)
	copyTestLanguageAuditMachineAuthority(t, root)
	copyBundleAuthority(t, root, currentValidationReopenDependencies)
	copyBundleAuthority(t, root, append(append([]string{}, commentGateAuthorityPaths...), "internal/i18n/validate.go", "internal/i18n/go_example_validate.go", "internal/i18n/go_example_protect.go", "internal/i18n/translate_protect.go", "internal/i18n/preformatted.go"))
	ids := []string{c.Pages[0].ID, c.Pages[1].ID}
	e, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: locale, Generator: RetranslationGeneratorChatGPT, UnitIDs: ids})
	if err != nil {
		t.Fatal(err)
	}
	bundle, manifest, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: locale, BatchID: e.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	_, files, err := readGenerationBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	outputs := map[string][]byte{}
	for _, u := range manifest.ExpectedOutputs {
		raw := string(files["formal/inputs/"+filepath.Base(u.BundlePath)])
		outputs[u.UnitID] = []byte(strings.NewReplacer("* Page", "* 页面", "Use ", "在此页面使用 ", " on this page.", "。").Replace(raw))
	}
	futureImport(t, root, c, bundle, outputs)
	if r, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: locale, BatchID: e.BatchID}); err != nil || r.ValidationPassed != len(ids) {
		t.Fatal("native fixture processing", r, err)
	}
	materializeSnapshotSources(t, root, c)
	futureSnapshot(t, root, c, snapshot)
	recordQualityCheckRatings(t, root, c, snapshot, "", "A", ids)
	if _, _, err := FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: locale, SnapshotID: snapshot}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := EstablishCurrentTranslationAuthorization(root, c, locale, snapshot); err != nil {
		t.Fatal("native baseline proof", err)
	}
	qcPath, finalPath := qualityCheckResultsPath(root, locale, snapshot), qualityCheckFinalizationPath(root, locale, snapshot)
	beforeQC, err := os.ReadFile(qcPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeFinal, err := os.ReadFile(finalPath)
	if err != nil {
		t.Fatal(err)
	}
	// A synthetic policy epoch makes the accepted punctuation fail current
	// validation. Only this TempDir authority changes; history/candidates do not.
	metadata, err := os.ReadFile(filepath.Join(root, "locales", locale, "locale.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m Locale
	if err := json.Unmarshal(metadata, &m); err != nil {
		t.Fatal(err)
	}
	m.LanguageAudit.Version++
	m.LanguageAudit.Policy.SentenceEnd = map[string]string{"。": "."}
	writeSnapshotSource(t, root, "locales/"+locale+"/locale.json", mustJSON(m))
	assessment, err := AssessCurrentTranslation(root, c, locale, snapshot)
	if err != nil || assessment.Package != tourNativeTranslationPackage || assessment.Affected != 2 || assessment.Compatible != 0 || assessment.Stale != 0 || assessment.Unknown != 0 {
		t.Fatal("native current machine findings", err)
	}
	snapshotRef, err := successorReference(root, "data/quality-check-snapshots/"+locale+"/"+snapshot+"/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	o := CurrentValidationReopenOptions{ContentPackage: tourNativeTranslationPackage, Locale: locale, ReopenID: "native-current-failure", PreviousSnapshotID: snapshot, SnapshotSHA256: snapshotRef.SHA256}
	evidence := func(a CurrentTranslationAuthorization, path string) CurrentValidationReopenOptions {
		a.Identity = authorizationIdentity(a)
		raw := mustJSON(a)
		writeSnapshotSource(t, root, path, raw)
		x := o
		x.Evidence = GlossaryArchiveReference{Path: path, SHA256: sum(raw)}
		x.AssessmentIdentity = a.Identity
		return x
	}
	o = evidence(*assessment, "engineering/native-assessment.json")
	r, path, err := InspectCurrentValidationReopen(root, c, o)
	if err != nil || r.CurrentValidation.Package != tourNativeTranslationPackage || r.UnitCount != 2 {
		t.Fatal("native readonly reopen", err)
	}
	for i, u := range r.Units {
		if u.UnitID != ids[i] || u.PreviousRating != "A" || !strings.Contains(r.CurrentValidation.Findings[i].Reason, "sentence-punctuation") {
			t.Fatal("native ordered A/finding binding changed", u)
		}
	}
	if _, err := os.Stat(filepath.Join(root, path)); !os.IsNotExist(err) {
		t.Fatal("inspection published a receipt", err)
	}
	for _, name := range []string{"assessment-package", "assessment-predecessor", "assessment-order", "affected-subset", "affected-order", "foreign-selector"} {
		t.Run(name, func(t *testing.T) {
			x := o
			want := ""
			switch name {
			case "assessment-package", "assessment-predecessor", "assessment-order":
				raw := mustJSON(assessment)
				var a CurrentTranslationAuthorization
				if err := json.Unmarshal(raw, &a); err != nil {
					t.Fatal(err)
				}
				if name == "assessment-package" {
					a.Package, want = "learn-docs-v1", "identity/package mismatch"
				} else if name == "assessment-predecessor" {
					a.Historical.Finalization.SnapshotID, want = "foreign-predecessor", "predecessor/locale mismatch"
				} else {
					a.Units[0], a.Units[1] = a.Units[1], a.Units[0]
					want = "current re-attestation changed"
				}
				x = evidence(a, "engineering/"+name+".json")
			case "affected-subset":
				x.UnitIDs, want = ids[:1], "Unit set/order must exactly match"
			case "affected-order":
				x.UnitIDs, want = []string{ids[1], ids[0]}, "Unit set/order must exactly match"
			case "foreign-selector":
				x.ContentPackage, want = "unregistered-package", "unregistered translation package"
			}
			if _, _, err := InspectCurrentValidationReopen(root, c, x); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatal("wrong native authority failure class", err, "want", want)
			}
		})
	}
	afterQC, _ := os.ReadFile(qcPath)
	afterFinal, _ := os.ReadFile(finalPath)
	if !bytes.Equal(beforeQC, afterQC) || !bytes.Equal(beforeFinal, afterFinal) || len(c.TranslationScope) != 0 || len(c.TranslationPackages) != 0 {
		t.Fatal("native inspection changed history, scope or registration")
	}
}
