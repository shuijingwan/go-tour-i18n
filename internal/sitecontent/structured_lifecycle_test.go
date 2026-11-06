package sitecontent

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func structuredLifecycleFixture(t *testing.T, pkg string, locales ...string) (string, Reference, Reference, *Reference) {
	t.Helper()
	locale := fixtureLocale(locales)
	root := siteFixture(t, "docs/RETRANSLATION_RUNBOOK.md", "docs/TRANSLATION_QUALITY_REVIEW.md", "current-authorization-fresh-fixture")
	if locale == "en-GB" {
		englishCompatibilityLocale(t, root)
	}
	seedSiteLanguagePolicyFixture(t, root, locale, "Latin")
	var page *Reference
	if pkg == "learn-docs-v1" {
		c, err := checkedCatalog(root)
		if err != nil {
			t.Fatal(err)
		}
		ids, err := AttachPageTranslationUnits(root, c)
		if err != nil {
			t.Fatal(err)
		}
		if err := c.UseTranslationPackage("learn-docs-v1"); err != nil {
			t.Fatal(err)
		}
		for _, set := range [][]string{ids[:30], ids[30:]} {
			batch, _ := importSyntheticSharedPagesForLocale(t, root, c, locale, set)
			if r, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: locale, BatchID: batch}); err != nil || r.ValidationPassed != len(set) {
				t.Fatal(r, err)
			}
		}
		const snapshot = "structured-baseline"
		if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: locale, SnapshotID: snapshot}); err != nil {
			t.Fatal(err)
		}
		for _, set := range [][]string{ids[:30], ids[30:]} {
			if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{Locale: locale, SnapshotID: snapshot, ReviewerSession: "synthetic-independent-reviewer", UnitIDs: set, Rating: "A"}); err != nil {
				t.Fatal(err)
			}
		}
		_, path, err := i18n.FinalizeQualityCheck(root, c, i18n.QualityCheckFinalizeOptions{Locale: locale, SnapshotID: snapshot})
		if err != nil {
			t.Fatal(err)
		}
		ref, err := ReferenceFile(root, path)
		if err != nil {
			t.Fatal(err)
		}
		page = &ref
		if _, _, err := i18n.EstablishCurrentTranslationAuthorization(root, c, locale, snapshot); err != nil {
			t.Fatal(err)
		}
	}
	_, gen := importStructuredFixture(t, root, locale)
	closures := []Reference{}
	var closure Reference
	packages := []string{"site-v2-shell"}
	if page != nil {
		packages = []string{"learn-docs-v1", "site-v2-shell"}
	}
	for _, id := range packages {
		p := page
		if id == "site-v2-shell" {
			p = nil
		}
		_, path, err := FinalizePackageClosure(root, locale, id, p, []Reference{gen})
		if err != nil {
			t.Fatal(err)
		}
		ref, _ := ReferenceFile(root, path)
		closures = append(closures, ref)
		if id == pkg {
			closure = ref
		}
	}
	scope, err := BuildIntegratedSurfaceScope(root, locale, closures)
	if err != nil {
		t.Fatal(err)
	}
	md := []byte("# Synthetic independent Surface\n\nPASS\n")
	writeFixture(t, root, "data/structured-surface.md", md)
	r := IntegratedSurfaceReceipt{Schema: IntegratedSurfaceSchema, Scope: *scope, ReviewID: "structured-baseline", Reviewer: "synthetic-independent-reviewer", Decision: "passed", Findings: []string{}, Evidence: Reference{"data/structured-surface.md", digest(md)}}
	raw, _ := Encode(r)
	_, receiptPath, err := RecordIntegratedSurface(root, raw)
	if err != nil {
		t.Fatal(err)
	}
	surface, _ := ReferenceFile(root, receiptPath)
	g := mustGlobal(t, root)
	l := Locale{Schema: LocaleSchema, Locale: locale, Packages: []Completion{}}
	for _, ref := range closures {
		var c PackageClosure
		if err := readReference(root, ref, &c); err != nil {
			t.Fatal(err)
		}
		p, err := packageByID(g, c.Package)
		if err != nil {
			t.Fatal(err)
		}
		evidence := []Reference{ref, surface}
		l.Packages = append(l.Packages, Completion{Package: p.ID, State: "complete", PackageIdentity: p.Identity, SourceIdentity: p.SourceIdentity, Surfaces: p.Surfaces, Routes: p.Routes, RouteFamilies: routeFamilies(p.ID), EvidenceKind: PackageCompletionKind, Evidence: evidence, EvidenceIdentity: identity(evidence), ContextIdentity: c.Identity})
	}
	raw, _ = Encode(l)
	writeFixture(t, root, LocalePath(locale), raw)
	if _, err := CheckLocale(root, g, locale); err != nil {
		t.Fatal("G1 content-scope complete", err)
	}
	if pkg == "site-v2-shell" {
		page = nil
	}
	return root, closure, gen, page
}

func structuredGlossarySuccessor(t *testing.T, root, term string) {
	structuredGlossarySuccessorWithLabel(t, root, term, "Synthetic replacement label", "structured-current-review")
}

func structuredGlossarySuccessorWithLabel(t *testing.T, root, term, label, reviewID string, locales ...string) {
	t.Helper()
	locale := fixtureLocale(locales)
	writeFixture(t, root, "locales/"+locale+"/glossary.yaml", []byte("locale: "+locale+"\nmandatory:\n  synthetic-test-term: 测试术语\n  "+term+": "+label+"\n"))
	b, _, err := i18n.ExportUnifiedGlossaryBundle(root, locale, "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.RecordUnifiedGlossaryReview(root, b, reviewID, "synthetic-independent-reviewer", "synthetic-generation", "passed", nil); err != nil {
		t.Fatal(err)
	}
	// The TU world remains exact and compatible. Freeze its existing machine
	// checkpoint once, as the real qc-012 campaign has already done.
	if _, err := readRegular(root, "data/quality-check-snapshots/"+locale+"/structured-baseline/finalization.json"); err != nil {
		return
	}
	c, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AttachPageTranslationUnits(root, c); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage("learn-docs-v1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.EstablishCurrentTranslationAuthorization(root, c, locale, "structured-baseline"); err != nil {
		t.Fatal("unchanged TU current authorization", err)
	}
}

func TestStructuredBaselineDeltaLifecycle(t *testing.T) {
	for _, tc := range []struct{ pkg, term, changed string }{
		{"learn-docs-v1", "The Go Programming Language", "_content/learn/books.yaml"},
		{"site-v2-shell", "Learning resources", "data/site-shell/home.html"},
	} {
		t.Run(tc.pkg, func(t *testing.T) {
			root, baseline, gen, page := structuredLifecycleFixture(t, tc.pkg)
			var pageBefore []byte
			if page != nil {
				pageBefore, _ = readRegular(root, page.Path)
			}
			before, err := PlanStructuredAssets(root, "zh-CN")
			if err != nil {
				t.Fatal(err)
			}
			for _, a := range before.Assets {
				if a.Package == tc.pkg && a.State != "carry" {
					t.Fatal("G1 not carried", a)
				}
			}
			structuredGlossarySuccessor(t, root, tc.term)
			plan, err := PlanStructuredAssets(root, "zh-CN")
			if err != nil {
				t.Fatal(err)
			}
			expected := "locales/zh-CN/structured/" + tc.changed
			if !reflect.DeepEqual(plan.Expected, []string{expected}) {
				t.Fatal("Expected must be B only", plan.Expected)
			}
			for _, a := range plan.Assets {
				if a.Package != tc.pkg {
					continue
				}
				want := "carry"
				if a.TargetPath == expected {
					want = "generation-required"
				}
				if a.State != want {
					t.Fatalf("G2 state: %s=%s want=%s", a.ID, a.State, want)
				}
			}
			globalPlan, err := PlanLocaleLanguage(root, "zh-CN")
			if err != nil {
				t.Fatal("global planner blocked by affected completion", err)
			}
			if len(globalPlan.Work) == 0 {
				t.Fatal("empty global plan")
			}
			if _, _, err := BuildPackageClosure(root, "zh-CN", tc.pkg, page, []Reference{gen}); err == nil {
				t.Fatal("affected false carry")
			}
			bundle, m, err := ExportStructuredAssets(root, "zh-CN")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := CheckStructuredAssetsBundle(root, bundle); err != nil {
				t.Fatal(err)
			}
			o := StructuredOutput{Schema: StructuredSchema, BundleSHA: m.Identity, Outputs: []StructuredTarget{}}
			// The initial helper's output provides unchanged synthetic values for
			// newly registered fixture assets. This is test data, never language.
			var original StructuredGeneration
			if err := readReference(root, gen, &original); err != nil {
				t.Fatal(err)
			}
			for _, path := range m.Plan.Expected {
				for _, target := range original.Output.Outputs {
					if target.Path != path {
						continue
					}
					copy := target
					copy.Slots = append([]Target(nil), target.Slots...)
					for i := range copy.Slots {
						copy.Slots[i].Text = strings.ReplaceAll(copy.Slots[i].Text, tc.term, "Synthetic replacement label")
					}
					o.Outputs = append(o.Outputs, copy)
				}
			}
			raw, _ := Encode(o)
			provenance := Provenance{Provider: "codex", Model: "gpt-5.6-sol-high", Session: "synthetic-generation", GeneratedAt: "2026-10-06T01:00:00Z"}
			if tc.pkg == "site-v2-shell" {
				structuredDeltaNegatives(t, root, bundle, m, o, provenance)
			}
			_, deltaPath, err := ImportStructuredAssets(root, bundle, raw, Provenance{Provider: "codex", Model: "gpt-5.6-sol-high", Session: "synthetic-generation", GeneratedAt: "2026-10-06T01:00:00Z"})
			if err != nil {
				t.Fatal(err)
			}
			delta, _ := ReferenceFile(root, deltaPath)
			if _, _, err := BuildPackageClosure(root, "zh-CN", tc.pkg, page, []Reference{delta}); err != nil {
				t.Fatal("verified baseline must not require flattened ancestry", err)
			}
			if _, _, err := BuildPackageClosure(root, "zh-CN", tc.pkg, page, []Reference{gen, delta, delta}); err == nil {
				t.Fatal("duplicate conflicting delta accepted")
			}
			closure, path, err := FinalizePackageClosure(root, "zh-CN", tc.pkg, page, []Reference{gen, delta})
			if err != nil {
				t.Fatal(err)
			}
			if closure.StructuredBaseline == nil || *closure.StructuredBaseline != baseline {
				t.Fatal("baseline not bound")
			}
			ref, _ := ReferenceFile(root, path)
			if _, _, err := CheckPackageClosure(root, ref); err != nil {
				t.Fatal(err)
			}
			if tc.pkg == "site-v2-shell" {
				secondProvenance := provenance
				secondProvenance.GeneratedAt = "2026-10-06T02:00:00Z"
				_, secondPath, err := ImportStructuredAssets(root, bundle, raw, secondProvenance)
				if err != nil {
					t.Fatal(err)
				}
				second, _ := ReferenceFile(root, secondPath)
				if _, _, err := BuildPackageClosure(root, "zh-CN", tc.pkg, page, []Reference{gen, delta, second}); err == nil {
					t.Fatal("duplicate conflicting generations accepted")
				}
				for _, extra := range []bool{false, true} {
					bad := *closure
					bad.Files = append([]Reference(nil), closure.Files...)
					if extra {
						bad.Files = append(bad.Files, Reference{"extra-file.yaml", strings.Repeat("0", 64)})
					} else {
						bad.Files = bad.Files[1:]
					}
					bad.Identity = ""
					bad.Identity = identity(bad)
					path := workflowPath(bad.Locale, bad.Package, "closure", bad.Identity)
					if err := saveImmutable(root, path, bad); err != nil {
						t.Fatal(err)
					}
					badRef, _ := ReferenceFile(root, path)
					if _, _, err := CheckPackageClosure(root, badRef); err == nil {
						t.Fatal("incomplete/non-exact merged inventory accepted")
					}
				}
			}
			_, reverse, err := BuildPackageClosure(root, "zh-CN", tc.pkg, page, []Reference{delta, gen})
			if err != nil {
				t.Fatal(err)
			}
			_, merged, err := CheckPackageClosure(root, ref)
			if err != nil || !reflect.DeepEqual(reverse, merged) {
				t.Fatal("order-dependent merge", err)
			}
			scope, err := BuildIntegratedSurfaceScope(root, "zh-CN", []Reference{ref})
			if err != nil {
				t.Fatal(err)
			}
			carried := 0
			for _, a := range scope.Carried {
				if a.Package == tc.pkg {
					carried++
				}
			}
			if len(scope.Reviewed) != 1 || scope.Reviewed[0].ID != tc.changed || carried != len(closure.Files)-1 {
				t.Fatalf("delta Surface: reviewed=%d carried=%d", len(scope.Reviewed), carried)
			}
			// Historical integrity survives the glossary change.
			if _, err := historicalStructuredClosure(root, baseline, map[string]bool{}); err != nil {
				t.Fatal(err)
			}
			var base PackageClosure
			_ = readReference(root, baseline, &base)
			var generation StructuredGeneration
			_ = readReference(root, gen, &generation)
			var surface IntegratedSurfaceReceipt
			l, _ := readLocaleAuthority(root, mustGlobal(t, root), "zh-CN")
			_ = readReference(root, l.Packages[0].Evidence[1], &surface)
			for name, file := range map[string]string{"historical-generation-corrupt": gen.Path, "bundle-hash-drift": generation.Bundle.Path, "baseline-target-drift": baseline.Path, "evidence-hash-drift": surface.Evidence.Path, "downstream-tamper": ref.Path, "missing-current-Unified-Review": plan.Review.Path} {
				t.Run(name, func(t *testing.T) {
					b, _ := readRegular(root, file)
					writeFixture(t, root, file, append(bytes.Clone(b), []byte("tamper")...))
					defer writeFixture(t, root, file, b)
					if _, _, err := CheckPackageClosure(root, ref); err == nil {
						t.Fatal("tampered evidence accepted")
					}
				})
			}
			if page != nil {
				after, _ := readRegular(root, page.Path)
				if !bytes.Equal(pageBefore, after) {
					t.Fatal("TU reopened")
				}
			}
			t.Run("second-successor", func(t *testing.T) {
				structuredSecondSuccessor(t, root, ref, gen, delta, page, tc.pkg, tc.term, tc.changed)
			})
			t.Log("historical baseline/current exact delta/closure/downstream/Surface PASS", tc.pkg)
		})
	}
}

func structuredDeltaNegatives(t *testing.T, root string, bundle []byte, m *StructuredBundle, output StructuredOutput, provenance Provenance) {
	t.Helper()
	// Test the shared authority primitive with a small package view; full exact
	// bundle/current checker and persisted downstream checks run above.
	base := StructuredGeneration{Schema: StructuredSchema, Plan: m.Plan, Output: output, Provenance: provenance}
	base.Plan.Assets = []StructuredAsset{}
	for _, a := range m.Plan.Assets {
		if a.Package == "site-v2-shell" {
			base.Plan.Assets = append(base.Plan.Assets, a)
		}
	}
	clone := func() StructuredGeneration {
		b, _ := Encode(base)
		var r StructuredGeneration
		_ = StrictJSON(b, &r)
		return r
	}
	for _, tc := range []struct {
		name   string
		mutate func(*StructuredGeneration)
	}{
		{"affected-false-carry", func(r *StructuredGeneration) {
			for i := range r.Plan.Assets {
				if r.Plan.Assets[i].State == "generation-required" {
					r.Plan.Assets[i].State = "carry"
				}
			}
		}},
		{"compatible-arbitrary-replacement", func(r *StructuredGeneration) {
			for i := range r.Plan.Assets {
				if r.Plan.Assets[i].State == "carry" {
					r.Plan.Assets[i].State = "generation-required"
				}
			}
		}},
		{"unauthorized-extra-delta", func(r *StructuredGeneration) {
			r.Plan.Expected = append(r.Plan.Expected, "locales/zh-CN/structured/extra.yaml")
		}},
		{"missing-affected-delta", func(r *StructuredGeneration) { r.Plan.Expected = nil }},
		{"wrong-glossary-binding", func(r *StructuredGeneration) { r.Plan.Assets[0].Assessment.GlossarySHA = strings.Repeat("0", 64) }},
		{"current-source-drift", func(r *StructuredGeneration) { r.Plan.Assets[0].SourceSHA = strings.Repeat("0", 64) }},
		{"unknown-parser", func(r *StructuredGeneration) { r.Plan.Assets[0].Document.Contract = "unknown-parser/v1" }},
		{"wrong-baseline-closure", func(r *StructuredGeneration) { r.Plan.Assets[0].Assessment.Closure.SHA256 = strings.Repeat("0", 64) }},
		{"wrong-predecessor-generation", func(r *StructuredGeneration) { r.Plan.Assets[0].Assessment.Generation.SHA256 = strings.Repeat("0", 64) }},
		{"wrong-predecessor-target", func(r *StructuredGeneration) { r.Plan.Assets[0].Assessment.TargetSHA = strings.Repeat("0", 64) }},
		{"cross-package-baseline-misuse", func(r *StructuredGeneration) { r.Plan.Assets[0].Package = "unrelated-package" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := clone()
			tc.mutate(&r)
			if err := validateStructuredDelta(root, &r, m); err == nil {
				t.Fatal("invalid delta authority accepted")
			}
		})
	}
	for _, tc := range []struct {
		name   string
		mutate func(*StructuredOutput)
	}{
		{"validator-failure", func(o *StructuredOutput) {
			for i := range o.Outputs[0].Slots {
				if o.Outputs[0].Slots[i].Text == "Synthetic replacement label" {
					o.Outputs[0].Slots[i].Text = "Wrong mandatory label"
				}
			}
		}},
		{"reconstruction-failure", func(o *StructuredOutput) { o.Outputs[0].Slots[0].Text = "⟦P999⟧" }},
		{"incomplete-non-exact-output", func(o *StructuredOutput) { o.Outputs = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := Encode(output)
			var o StructuredOutput
			_ = StrictJSON(b, &o)
			tc.mutate(&o)
			if _, err := validateStructuredOutput(root, m.Plan, o); err == nil {
				t.Fatal("invalid output accepted")
			}
		})
	}
	t.Run("provenance-input-mismatch", func(t *testing.T) {
		bad := provenance
		bad.Model = "invented-model"
		if validateProvenance(bad) == nil {
			t.Fatal("bad provenance accepted")
		}
		badOutput := output
		badOutput.BundleSHA = strings.Repeat("0", 64)
		b, _ := Encode(badOutput)
		if _, _, err := ImportStructuredAssets(root, bundle, b, provenance); err == nil {
			t.Fatal("input identity mismatch accepted")
		}
	})
}

func mustGlobal(t *testing.T, root string) *Global {
	t.Helper()
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	return g
}
