package sitecontent

import (
	"bytes"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const sharedFixtureLocale = "en-GB"

func sharedRepresentationFixture(t *testing.T) (string, Reference, Reference, *Reference) {
	return sharedRepresentationFixtureWithHomepage(t, false)
}

func sharedRepresentationFixtureWithHomepage(t *testing.T, homepage bool) (string, Reference, Reference, *Reference) {
	t.Helper()
	root, _, _, _ := shellAdoptionFixture(t)
	for _, name := range append(append(append([]string{}, shellRepresentationParserPaths...), sharedRepresentationParserPaths...), sharedReattestationValidationPaths...) {
		b, err := readRegular(repositoryRoot(t), name)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, name, b)
	}
	// Transport-only source-identical teaching comments require real English
	// locale authority. Homepage successor compatibility is tested separately.
	englishCompatibilityLocale(t, root)
	if homepage {
		plan, err := PlanShellRepresentationSuccessor(root, ShellTeachingRepresentation)
		if err != nil {
			t.Fatal("canonical Homepage fixture", err)
		}
		b, _ := Encode(plan)
		writeFixture(t, root, "data/homepage-representation-plan.json", b)
		ref, _ := ReferenceFile(root, "data/homepage-representation-plan.json")
		if _, err := ApplyShellRepresentationSuccessor(root, ref, plan.PriorIdentity, plan.PriorPointerSHA); err != nil {
			t.Fatal(err)
		}
	}
	c, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackagesReadonly(root, c); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage("site-v2-shell"); err != nil {
		t.Fatal(err)
	}
	batch, _ := importSyntheticSharedPagesForLocale(t, root, c, sharedFixtureLocale, c.TranslationScope)
	if r, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: sharedFixtureLocale, BatchID: batch}); err != nil || r.ValidationPassed != len(c.TranslationScope) {
		t.Fatal(r, err)
	}
	const snapshot = "shared-page-baseline"
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: sharedFixtureLocale, SnapshotID: snapshot, ContentPackage: "site-v2-shell"}); err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{Locale: sharedFixtureLocale, SnapshotID: snapshot, ReviewerSession: "synthetic-independent-reviewer", UnitIDs: c.TranslationScope, Rating: "A"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.FinalizeQualityCheck(root, c, i18n.QualityCheckFinalizeOptions{Locale: sharedFixtureLocale, SnapshotID: snapshot}); err != nil {
		t.Fatal(err)
	}
	page, err := ReferenceFile(root, "data/quality-check-snapshots/"+sharedFixtureLocale+"/"+snapshot+"/finalization.json")
	if err != nil {
		t.Fatal(err)
	}
	_, generation := importStructuredFixture(t, root, sharedFixtureLocale)
	closure, path, err := FinalizePackageClosure(root, sharedFixtureLocale, "site-v2-shell", &page, []Reference{generation})
	if err != nil {
		t.Fatal(err)
	}
	cr, _ := ReferenceFile(root, path)
	scope, err := BuildIntegratedSurfaceScope(root, sharedFixtureLocale, []Reference{cr})
	if err != nil {
		t.Fatal(err)
	}
	md := []byte("# Synthetic independent shared predecessor\n\nPASS\n")
	writeFixture(t, root, "data/shared-predecessor.md", md)
	r := IntegratedSurfaceReceipt{Schema: IntegratedSurfaceSchema, Scope: *scope, ReviewID: "shared-predecessor", Reviewer: "synthetic-independent-reviewer", Decision: "passed", Findings: []string{}, Evidence: Reference{"data/shared-predecessor.md", digest(md)}}
	b, _ := Encode(r)
	_, rp, err := RecordIntegratedSurface(root, b)
	if err != nil {
		t.Fatal(err)
	}
	sr, _ := ReferenceFile(root, rp)
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := packageByID(g, "site-v2-shell")
	evidence := []Reference{cr, sr}
	l := Locale{Schema: LocaleSchema, Locale: sharedFixtureLocale, Packages: []Completion{{Package: p.ID, State: "complete", PackageIdentity: p.Identity, SourceIdentity: p.SourceIdentity, Surfaces: p.Surfaces, Routes: p.Routes, RouteFamilies: routeFamilies(p.ID), EvidenceKind: PackageCompletionKind, Evidence: evidence, EvidenceIdentity: identity(evidence), ContextIdentity: closure.Identity}}}
	b, _ = Encode(l)
	writeFixture(t, root, LocalePath(sharedFixtureLocale), b)
	return root, cr, generation, &page
}

func TestSharedStructuredRepresentationSuccessor(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, baseline, consumed, page := sharedRepresentationFixture(t)
	const source = "_content/site.tmpl"
	const slot = "0719e14f5f889a9c0dccb5eb291c9fe1de9604b6f5423b3c1497204c4276a85d"
	raw, err := readRegular(root, source)
	if err != nil {
		t.Fatal(err)
	}
	old, err := ExtractShellLanguageSource(source, raw)
	if err != nil {
		t.Fatal(err)
	}
	q := SharedRepresentationRequest{Schema: SharedRepresentationSchema + "/request", Package: "site-v2-shell", AssetID: "source:" + source, Path: source, SourceSHA: digest(raw), PreviousIdentity: identity(*old), Generation: consumed, Rule: SharedRepresentationRule, Slots: []string{slot}}
	land := func(name string, value any) Reference {
		t.Helper()
		b, err := Encode(value)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, name, b)
		ref, err := ReferenceFile(root, name)
		if err != nil {
			t.Fatal(err)
		}
		return ref
	}
	request := land("data/shared-request.json", q)
	before, err := PlanStructuredAssets(root, sharedFixtureLocale)
	if err != nil {
		t.Fatal(err)
	}
	oldGen, oldFiles, _, err := historicalStructuredGeneration(root, consumed)
	if err != nil {
		t.Fatal(err)
	}
	oldReceipt, _ := readRegular(root, consumed.Path)
	plan, err := PlanSharedStructuredRepresentation(root, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Authority.Released) != 1 || plan.Authority.Released[0].SlotID != slot || plan.Authority.Released[0].Token.Raw != "activate/deactivate" || plan.Authority.Released[0].Token.Role != "atom" || plan.Authority.New.SourceSHA != old.SourceSHA || plan.Authority.New.Page != nil || plan.Authority.LocaleScope != "all-locales" {
		t.Fatal("wrong exact topology/source scope")
	}
	for _, s := range plan.Authority.New.Slots {
		for _, p := range s.Unit.Protected {
			if p.Raw == "activate/deactivate" {
				t.Fatal("natural slash alternatives still frozen")
			}
		}
	}
	input := land("data/shared-plan.json", plan)
	currentBefore, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	docsBefore, _, err := CurrentPackageDocuments(root, currentBefore, "site-v2-shell")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range docsBefore {
		if hasSharedDocumentAuthority(d) {
			t.Fatal("prospective proof adopted without publication")
		}
	}
	if _, err := InspectShellRepresentationSuccessor(root, input); err == nil {
		t.Fatal("Homepage family accepted shared structured schema")
	}
	if _, err := InspectShellRepresentationReattestation(root, input); err == nil {
		t.Fatal("re-attestation accepted topology mutation")
	}
	inspection, err := InspectSharedStructuredRepresentation(root, input)
	if err != nil || inspection.Applied || inspection.AlreadyCurrent || inspection.Mutation != "none" {
		t.Fatal("readonly plan inspection", inspection, err)
	}
	for _, name := range []string{"package", "path", "raw", "predecessor", "unknown-slot", "duplicate-slot", "machine-slot", "unknown-rule", "wrong-generation"} {
		t.Run(name, func(t *testing.T) {
			b, _ := Encode(q)
			var bad SharedRepresentationRequest
			_ = StrictJSON(b, &bad)
			switch name {
			case "package":
				bad.Package = "learn-docs-v1"
			case "path":
				bad.Path = "internal/site/runtime.go"
				bad.AssetID = "source:" + bad.Path
			case "raw":
				bad.SourceSHA = strings.Repeat("0", 64)
			case "predecessor":
				bad.PreviousIdentity = strings.Repeat("1", 64)
			case "unknown-slot":
				bad.Slots = []string{strings.Repeat("2", 64)}
			case "duplicate-slot":
				bad.Slots = []string{slot, slot}
			case "machine-slot":
				bad.Slots = []string{old.Slots[0].Unit.ID}
			case "unknown-rule":
				bad.Rule = "unknown"
			case "wrong-generation":
				bad.Generation = baseline
			}
			if _, err := PlanSharedStructuredRepresentation(root, land("data/bad-"+name+".json", bad)); err == nil {
				t.Fatal("invalid shared authority accepted")
			}
		})
	}
	t.Run("ambiguous-head", func(t *testing.T) {
		writeFixture(t, root, plan.CurrentPath, []byte(`[{"path":"a","sha256":"b"},{"path":"c","sha256":"d"}]`))
		defer os.Remove(pathJoin(root, plan.CurrentPath))
		if _, err := PlanSharedStructuredRepresentation(root, request); err == nil {
			t.Fatal("multiple heads accepted")
		}
	})
	t.Run("CAS", func(t *testing.T) {
		if _, err := ApplySharedStructuredRepresentation(root, input, "wrong", plan.PriorPointerSHA); err == nil {
			t.Fatal("wrong prior identity accepted")
		}
		load := func(v *structuredSuccessorValidation) (*Global, error) { return loadSourceCurrent(root, v) }
		if _, err := applySharedStructuredRepresentation(root, input, plan.PriorIdentity, plan.PriorPointerSHA, load, func() { writeFixture(t, root, plan.CurrentPath, []byte("tampered")) }); err == nil {
			t.Fatal("concurrent head mutation accepted")
		}
		if err := os.Remove(pathJoin(root, plan.CurrentPath)); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(pathJoin(root, SharedRepresentationPending)); !os.IsNotExist(err) {
			t.Fatal("precommit failure created pending state", err)
		}
	})
	result, err := ApplySharedStructuredRepresentation(root, input, plan.PriorIdentity, plan.PriorPointerSHA)
	if err != nil || !result.Applied || result.Mutation != "known" {
		t.Fatal("isolated publication", result, err)
	}
	replay, err := ApplySharedStructuredRepresentation(root, input, plan.PriorIdentity, plan.PriorPointerSHA)
	if err != nil || !replay.AlreadyCurrent || replay.Applied || replay.Mutation != "none" {
		t.Fatal("exact installed replay", replay, err)
	}
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	docsAfter, _, err := CurrentPackageDocuments(root, g, "site-v2-shell")
	if err != nil {
		t.Fatal(err)
	}
	changed := 0
	for i, d := range docsAfter {
		if !reflect.DeepEqual(d, docsBefore[i]) {
			changed++
			if d.Path != source || !hasSharedDocumentAuthority(d) {
				t.Fatal("unaffected asset changed", d.Path)
			}
		}
	}
	if changed != 1 {
		t.Fatal("non-exact affected asset set", changed)
	}
	_, again, _, err := historicalStructuredGeneration(root, consumed)
	if err != nil || !reflect.DeepEqual(again, oldFiles) {
		t.Fatal("old Generation history changed", err)
	}
	after, err := PlanStructuredAssets(root, sharedFixtureLocale)
	if err != nil {
		t.Fatal(err)
	}
	var selected *StructuredAsset
	for i, a := range after.Assets {
		if a.ID == q.AssetID {
			selected = &after.Assets[i]
			if a.State != "generation-required" || a.Assessment == nil || a.Assessment.RepresentationSuccessor == nil || len(a.CarriedSlots) != 0 {
				t.Fatal("old language PASS carried to new representation")
			}
		} else {
			for _, oldAsset := range before.Assets {
				// Existing reopening semantics suppress an unrelated incomplete
				// package's initial work into context. It must not acquire a new
				// representation, target, finding or language carry authority.
				if a.ID == oldAsset.ID && a.Package != q.Package && oldAsset.Assessment == nil && oldAsset.State == "generation-required" && a.State == "context-only" {
					a.State = oldAsset.State
				}
				if oldAsset.ID == a.ID && !reflect.DeepEqual(oldAsset, a) {
					beforeBytes, _ := Encode(oldAsset)
					afterBytes, _ := Encode(a)
					t.Fatalf("unaffected plan asset changed %s\nbefore=%s\nafter=%s", a.ID, beforeBytes, afterBytes)
				}
			}
		}
	}
	if selected == nil || !reflect.DeepEqual(after.Expected, []string{selected.TargetPath}) {
		t.Fatal("non-exact generation scope", after.Expected)
	}
	bundle, m, err := ExportStructuredAssets(root, sharedFixtureLocale)
	if err != nil {
		t.Fatal("synthetic successor Bundle", err)
	}
	output := StructuredOutput{Schema: StructuredSchema, BundleSHA: m.Identity, Outputs: []StructuredTarget{{Path: selected.TargetPath, Slots: []Target{}}}}
	for _, u := range selected.Document.Units {
		value := u.Source
		if value == "Press Enter to activate/deactivate dropdown" {
			value = "按 Enter 键可激活或停用下拉菜单"
		}
		output.Outputs[0].Slots = append(output.Outputs[0].Slots, Target{u.ID, value})
	}
	b, _ := Encode(output)
	_, newPath, err := ImportStructuredAssets(root, bundle, b, oldGen.Provenance)
	if err != nil {
		t.Fatal("synthetic natural-language reconstruction", err)
	}
	newRef, _ := ReferenceFile(root, newPath)
	_, newFiles, _, err := historicalStructuredGeneration(root, newRef)
	if err != nil {
		t.Fatal("new frozen Generation reader", err)
	}
	if !bytes.Contains(newFiles[selected.TargetPath], []byte("按 Enter 键可激活或停用下拉菜单")) {
		t.Fatal("new representation did not express replacement")
	}
	_, closurePath, err := FinalizePackageClosure(root, sharedFixtureLocale, "site-v2-shell", page, []Reference{consumed, newRef})
	if err != nil {
		t.Fatal("successor closure", err)
	}
	closure, _ := ReferenceFile(root, closurePath)
	if _, err := BuildIntegratedSurfaceScope(root, sharedFixtureLocale, []Reference{closure}); err != nil {
		t.Fatal("successor Surface scope", err)
	}
	if got, _ := readRegular(root, source); !bytes.Equal(got, raw) {
		t.Fatal("raw source changed")
	}
	if got, _ := readRegular(root, consumed.Path); !bytes.Equal(got, oldReceipt) {
		t.Fatal("historical Generation bytes changed")
	}
	// Current semantic authority remains fail closed after a parser dependency
	// mutation; frozen new/old Generation decoding must remain independent.
	name := sharedRepresentationParserPaths[0]
	prior, _ := readRegular(root, name)
	writeFixture(t, root, name, append(bytes.Clone(prior), []byte("\n// synthetic dependency drift\n")...))
	if _, _, err := CurrentPackageDocuments(root, g, "site-v2-shell"); err == nil {
		t.Fatal("current dependency drift accepted")
	}
	if _, _, _, err := historicalStructuredGeneration(root, newRef); err != nil {
		t.Fatal("historical representation depended on current parser/head", err)
	}
	writeFixture(t, root, name, prior)
	t.Log("exact template owner; one slash atom delta; same raw; old/new history; affected-only generation/closure/Surface; Homepage unchanged")
}

func TestSharedRepresentationSelectedDeltaBound(t *testing.T) {
	raw := []byte(`<p>Read and/or write.</p><p>Use encoding/json and HTTP/2 with <code>json.Marshal</code>.</p><p>Choose time/space now.</p><p>Use package custom/widgets here.</p>`)
	old, err := ExtractShellLanguageSource("_content/site.tmpl", raw)
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{old.Slots[0].Unit.ID}
	sort.Strings(ids)
	next, delta, err := sharedRepresentationDelta(*old, raw, ids)
	if err != nil || len(delta) != 1 || delta[0].Token.Raw != "and/or" {
		t.Fatal("source-role delta", delta, err)
	}
	for i, s := range old.Slots {
		if i != 0 && !reflect.DeepEqual(s, next.Slots[i]) {
			t.Fatal("unrequested parser delta adopted")
		}
	}
	machines := map[string]bool{"encoding/json": false, "HTTP/2": false}
	for _, s := range next.Slots {
		for _, p := range s.Unit.Protected {
			if _, ok := machines[p.Raw]; ok {
				machines[p.Raw] = true
				if p.Role != "atom" {
					t.Fatal("machine role changed")
				}
			}
		}
	}
	for raw, found := range machines {
		if !found {
			t.Fatal("real machine identity released", raw)
		}
	}
	// The shared extractor owns explicit <code> as a frozen structural gap,
	// rather than an editable slot. Its API bytes must remain exact as well.
	if validateShellStructure(*next, raw, bytes.Replace(raw, []byte("json.Marshal"), []byte("json.Unmarshal"), 1)) == nil {
		t.Fatal("explicit code/API gap opened")
	}
	for _, s := range old.Slots {
		if strings.Contains(s.Unit.Source, "Use package") {
			if _, _, err := sharedRepresentationDelta(*old, raw, []string{s.Unit.ID}); err == nil {
				t.Fatal("unknown declared machine namespace treated as prose")
			}
		}
	}
	ref := Reference{SharedRepresentationRoot + "receipts/" + strings.Repeat("a", 64) + ".json", strings.Repeat("b", 64)}
	d := withSharedDocumentAuthority(shellDocument(*next), ref)
	values := map[string]string{}
	for _, s := range next.Slots {
		values[s.Unit.ID] = s.Unit.Source
	}
	if got, err := reconstructAuthorityStructuredRepresentation(d, raw, values); err != nil || !bytes.Equal(got, raw) {
		t.Fatal("frozen prospective round-trip", err)
	}
	for _, s := range next.Slots {
		for _, p := range s.Unit.Protected {
			values[s.Unit.ID] = strings.Replace(s.Unit.Source, p.Token, "changed", 1)
			if _, err := reconstructAuthorityStructuredRepresentation(d, raw, values); err == nil {
				t.Fatal("protected membership bypassed")
			}
			values[s.Unit.ID] = strings.Replace(s.Unit.Source, p.Token, p.Token+p.Token, 1)
			if _, err := reconstructAuthorityStructuredRepresentation(d, raw, values); err == nil {
				t.Fatal("duplicate protected binding accepted")
			}
			values[s.Unit.ID] = s.Unit.Source
		}
	}
}

func TestSharedRepresentationPendingRecoveryStop(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, SharedRepresentationPending, []byte("synthetic interrupted publication"))
	if _, err := ApplySharedStructuredRepresentation(root, Reference{Path: "missing-plan.json", SHA256: strings.Repeat("a", 64)}, "prior", "sha"); err == nil || !strings.Contains(err.Error(), "mutation=unknown") {
		t.Fatal("blind replay after unknown publication", err)
	}
	if _, _, err := selectSharedStructuredDocuments(root, &Global{}, "site-v2-shell", nil, nil); err == nil || !strings.Contains(err.Error(), "mutation=unknown") {
		t.Fatal("current reader bypassed unknown publication", err)
	}
	if _, err := PlanShellRepresentationSuccessor(root, ShellTeachingRepresentation); err == nil || !strings.Contains(err.Error(), "mutation=unknown") {
		t.Fatal("Homepage publisher ignored shared unknown state", err)
	}
	if _, err := os.Stat(pathJoin(root, SharedRepresentationPending)); err != nil {
		t.Fatal("recovery marker removed automatically", err)
	}
}
