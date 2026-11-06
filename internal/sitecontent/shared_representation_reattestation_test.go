package sitecontent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func landSharedReattestationFixture(t *testing.T, root, path string, value any) Reference {
	t.Helper()
	b, err := Encode(value)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, path, b)
	ref, err := ReferenceFile(root, path)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func TestSharedRepresentationReattestationLifecycle(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, _, generation, _ := sharedRepresentationFixtureWithHomepage(t, true)
	raw, err := readRegular(root, "_content/site.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	old, err := ExtractShellLanguageSource("_content/site.tmpl", raw)
	if err != nil {
		t.Fatal(err)
	}
	const slot = "0719e14f5f889a9c0dccb5eb291c9fe1de9604b6f5423b3c1497204c4276a85d"
	q := SharedRepresentationRequest{Schema: SharedRepresentationSchema + "/request", Package: "site-v2-shell", AssetID: "source:_content/site.tmpl", Path: "_content/site.tmpl", SourceSHA: digest(raw), PreviousIdentity: identity(*old), Generation: generation, Rule: SharedRepresentationRule, Slots: []string{slot}}
	request := landSharedReattestationFixture(t, root, "data/shared-request.json", q)
	p, err := PlanSharedStructuredRepresentation(root, request)
	if err != nil {
		t.Fatal(err)
	}
	input := landSharedReattestationFixture(t, root, "data/shared-plan.json", p)
	if _, err := InspectSharedStructuredRepresentation(root, input); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplySharedStructuredRepresentation(root, input, p.PriorIdentity, p.PriorPointerSHA); err != nil {
		t.Fatal(err)
	}
	baseBytes, _ := readRegular(root, p.Receipt.Path)
	baseHead, _ := readRegular(root, p.CurrentPath)
	historyBytes, _ := readRegular(root, generation.Path)
	_, historical, _, err := historicalStructuredGeneration(root, generation)
	if err != nil {
		t.Fatal(err)
	}
	before, err := PlanStructuredAssets(root, sharedFixtureLocale)
	if err != nil {
		t.Fatal("post-apply source-only package comparison", err)
	}
	noop, err := PlanSharedRepresentationReattestation(root, p.Receipt)
	if err != nil || noop.Eligible || !noop.BaseCurrent {
		t.Fatal("unchanged dependencies unexpectedly eligible", err)
	}
	if _, err := ApplySharedRepresentationReattestation(root, landSharedReattestationFixture(t, root, "data/noop.json", noop), noop.PriorPointerSHA); err == nil {
		t.Fatal("no-op authority published")
	}
	t.Run("fresh-base-binds-additive-checker", func(t *testing.T) {
		path := sharedReattestationValidationPaths[0]
		b := mustSharedFixtureBytes(t, root, path)
		defer writeFixture(t, root, path, b)
		writeFixture(t, root, path, append(bytes.Clone(b), []byte("\n// later additive checker epoch\n")...))
		if _, err := PlanStructuredAssets(root, sharedFixtureLocale); err == nil {
			t.Fatal("fresh base omitted additive checker dependency")
		}
		if proposed, err := PlanSharedRepresentationReattestation(root, p.Receipt); err != nil || !proposed.Eligible {
			t.Fatal("additive checker drift cannot bootstrap", err)
		}
	})
	parser := sharedRepresentationParserPaths[0]
	code, _ := readRegular(root, parser)
	writeFixture(t, root, parser, append(bytes.Clone(code), []byte("\n// synthetic dependency epoch\n")...))
	if _, err := PlanStructuredAssets(root, sharedFixtureLocale); err == nil {
		t.Fatal("drift silently accepted without authority")
	}
	rea, err := PlanSharedRepresentationReattestation(root, p.Receipt)
	if err != nil {
		t.Fatal("bootstrap required stale current checker", err)
	}
	if !rea.Eligible || rea.BaseCurrent || !reflect.DeepEqual(rea.Authority.Base, p.Authority) || rea.Authority.RepresentationIdentity != p.Authority.NewIdentity || len(rea.Authority.Changes) != 1 || len(rea.Authority.Added) != 0 || len(rea.Authority.Removed) != 0 {
		t.Fatal("dependency-only invariant lost")
	}
	reviewed := landSharedReattestationFixture(t, root, "data/shared-dependency-plan.json", rea)
	if _, err := InspectSharedRepresentationReattestation(root, reviewed); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"base", "representation-identity", "topology", "released", "dependency-addition", "mode", "old-input", "source-package", "registry", "source-ref"} {
		t.Run("plan-"+name, func(t *testing.T) {
			b, _ := Encode(rea)
			var bad SharedRepresentationReattestationPlan
			_ = StrictJSON(b, &bad)
			switch name {
			case "base":
				bad.Authority.BaseReceipt.SHA256 = strings.Repeat("a", 64)
			case "representation-identity":
				bad.Authority.RepresentationIdentity = strings.Repeat("a", 64)
			case "topology":
				bad.Authority.Base.New.Slots[0].Unit.Source += "changed"
			case "released":
				bad.Authority.Base.Released[0].Token.Raw = "other"
			case "dependency-addition":
				bad.Authority.CurrentInputs = append(bad.Authority.CurrentInputs, ShellRepresentationInput{Reference: Reference{"internal/unknown.go", strings.Repeat("a", 64)}, Mode: 0644})
			case "mode":
				bad.Authority.CurrentInputs[0].Mode ^= 0111
			case "old-input":
				bad.Authority.OldInputs[0].Reference.SHA256 = strings.Repeat("a", 64)
			case "source-package":
				bad.Authority.Base.SourcePackage.SourceIdentity = strings.Repeat("a", 64)
			case "registry":
				bad.Authority.Base.RegistrySHA = strings.Repeat("a", 64)
			case "source-ref":
				bad.Authority.Base.SourceAuthority[0].SHA256 = strings.Repeat("a", 64)
			}
			bad.Authority.Identity = ""
			bad.Authority.Identity = identity(bad.Authority)
			bad.Receipt = sharedReattestationReceipt(&bad.Authority)
			bad.Identity = ""
			bad.Identity = identity(bad)
			if _, err := InspectSharedRepresentationReattestation(root, landSharedReattestationFixture(t, root, "data/bad-"+name+".json", bad)); err == nil {
				t.Fatal("mutated invariant accepted")
			}
		})
	}
	for name, path := range map[string]string{"raw-source": q.Path, "source-authority": p.Authority.SourceAuthority[0].Path, "base-head": p.CurrentPath, "base-receipt": p.Receipt.Path} {
		t.Run(name, func(t *testing.T) {
			b, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, path, b)
			writeFixture(t, root, path, append(bytes.Clone(b), ' '))
			// Whitespace remains valid JSON but changes the exact hash/head.
			if _, err := PlanSharedRepresentationReattestation(root, p.Receipt); err == nil {
				t.Fatal("source/base bytes drift accepted")
			}
		})
	}
	t.Run("registry-drift", func(t *testing.T) {
		b, _ := readRegular(root, i18n.GlossaryCorpusPath)
		defer writeFixture(t, root, i18n.GlossaryCorpusPath, b)
		var corpus i18n.GlossarySourceCorpus
		_ = StrictJSON(b, &corpus)
		for i := range corpus.Contributors {
			c := &corpus.Contributors[i]
			if c.ID == q.AssetID {
				c.Contexts[0].Text += " synthetic drift"
				c.SourceSHA = i18n.ContributorIdentity(*c)
			}
		}
		corpus.Identity = i18n.CorpusIdentity(corpus)
		if err := i18n.ValidateGlossaryCorpus(&corpus); err != nil {
			t.Fatal(err)
		}
		b, _ = Encode(corpus)
		writeFixture(t, root, i18n.GlossaryCorpusPath, b)
		if _, err := PlanSharedRepresentationReattestation(root, p.Receipt); err == nil {
			t.Fatal("registry drift accepted")
		}
	})
	t.Run("mode-drift", func(t *testing.T) {
		st, _ := os.Stat(pathJoin(root, parser))
		defer os.Chmod(pathJoin(root, parser), st.Mode())
		_ = os.Chmod(pathJoin(root, parser), st.Mode()^0100)
		if _, err := PlanSharedRepresentationReattestation(root, p.Receipt); err == nil {
			t.Fatal("mode drift accepted")
		}
	})
	t.Run("stale-reviewed-plan", func(t *testing.T) {
		b, _ := readRegular(root, parser)
		defer writeFixture(t, root, parser, b)
		writeFixture(t, root, parser, append(bytes.Clone(b), []byte("\n// later epoch\n")...))
		if _, err := InspectSharedRepresentationReattestation(root, reviewed); err == nil {
			t.Fatal("stale reviewed plan accepted")
		}
	})
	t.Run("ambiguous-head", func(t *testing.T) {
		writeFixture(t, root, rea.CurrentPath, []byte(`[{"path":"one"},{"path":"two"}]`))
		defer os.Remove(pathJoin(root, rea.CurrentPath))
		if _, err := PlanSharedRepresentationReattestation(root, p.Receipt); err == nil {
			t.Fatal("ambiguous head accepted")
		}
	})
	if _, err := InspectShellRepresentationReattestation(root, reviewed); err == nil {
		t.Fatal("Homepage accepted shared re-attestation")
	}
	homeOnly := landSharedReattestationFixture(t, root, "data/homepage-dependency-kind.json", ShellRepresentationReattestationPlan{Schema: ShellRepresentationReattestationSchema + "/plan"})
	if _, err := InspectSharedRepresentationReattestation(root, homeOnly); err == nil {
		t.Fatal("shared asset accepted Homepage re-attestation")
	}
	if _, err := InspectSharedStructuredRepresentation(root, reviewed); err == nil {
		t.Fatal("dependency plan accepted as topology successor")
	}
	if _, err := ApplySharedRepresentationReattestation(root, reviewed, strings.Repeat("0", 64)); err == nil {
		t.Fatal("wrong prior accepted")
	}
	if _, err := applySharedReattestation(root, reviewed, rea.PriorPointerSHA, func() { writeFixture(t, root, rea.CurrentPath, []byte("concurrent")) }); err == nil {
		t.Fatal("concurrent pointer mutation accepted")
	}
	if err := os.Remove(pathJoin(root, rea.CurrentPath)); err != nil {
		t.Fatal(err)
	}
	result, err := ApplySharedRepresentationReattestation(root, reviewed, rea.PriorPointerSHA)
	if err != nil || !result.Applied || result.Mutation != "known" {
		t.Fatal("isolated apply", result, err)
	}
	replay, err := ApplySharedRepresentationReattestation(root, reviewed, rea.PriorPointerSHA)
	if err != nil || replay.Applied || !replay.AlreadyCurrent || replay.Mutation != "none" {
		t.Fatal("exact no-op", err)
	}
	if _, err := PlanSharedRepresentationReattestation(root, p.Receipt); err == nil {
		t.Fatal("already current preflight generated second authority")
	}
	after, err := PlanStructuredAssets(root, sharedFixtureLocale)
	if err != nil {
		t.Fatal("isolated current structured plan", err)
	}
	if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(after.Expected, []string{"locales/" + sharedFixtureLocale + "/structured/" + q.Path}) {
		t.Fatal("re-attestation changed language work scope")
	}
	for _, a := range after.Assets {
		if a.ID == q.AssetID {
			if a.State != "generation-required" {
				t.Fatal(a.State)
			}
			for _, u := range a.Document.Units {
				for _, p := range u.Protected {
					if p.Raw == "activate/deactivate" {
						t.Fatal("released prose refrozen")
					}
				}
			}
		}
	}
	bundle, _, err := ExportStructuredAssets(root, sharedFixtureLocale)
	if err != nil {
		t.Fatal("frozen additive context transport", err)
	}
	_, archived, err := readStructuredBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range []Reference{rea.Receipt, {Path: rea.CurrentPath, SHA256: digest(mustSharedFixtureBytes(t, root, rea.CurrentPath))}} {
		if digest(archived["inputs/"+ref.Path]) != ref.SHA256 {
			t.Fatal("actual additive context missing from Bundle", ref.Path)
		}
	}
	_, files, _, err := historicalStructuredGeneration(root, generation)
	if err != nil || !reflect.DeepEqual(files, historical) {
		t.Fatal("frozen predecessor history changed", err)
	}
	for path, b := range map[string][]byte{p.Receipt.Path: baseBytes, p.CurrentPath: baseHead, generation.Path: historyBytes, q.Path: raw} {
		if got, _ := readRegular(root, path); !bytes.Equal(got, b) {
			t.Fatal("immutable bytes changed", path)
		}
	}
	for _, path := range []string{parser, sharedReattestationValidationPaths[0]} {
		t.Run("installed-current-input-drift-"+filepath.Base(path), func(t *testing.T) {
			b := mustSharedFixtureBytes(t, root, path)
			defer writeFixture(t, root, path, b)
			writeFixture(t, root, path, append(bytes.Clone(b), []byte("\n// unadopted later epoch\n")...))
			if _, err := PlanStructuredAssets(root, sharedFixtureLocale); err == nil {
				t.Fatal("installed dependency currentness silently carried")
			}
		})
	}
	t.Log("ISOLATED_REATTESTATION_LIFECYCLE=PASS CURRENT_STRUCTURED_PLAN=PASS EXPECTED_COUNT=1; all other structured assets unchanged; native Homepage authority preserved")
}

func mustSharedFixtureBytes(t *testing.T, root, path string) []byte {
	t.Helper()
	b, err := readRegular(root, path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestSharedReattestationPendingStops(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, SharedReattestationPending, []byte("synthetic unknown publication"))
	for _, run := range []func() error{
		func() error { _, e := PlanSharedRepresentationReattestation(root, Reference{}); return e },
		func() error { _, e := InspectSharedRepresentationReattestation(root, Reference{}); return e },
		func() error { _, e := ApplySharedRepresentationReattestation(root, Reference{}, "prior"); return e },
		func() error { _, e := PlanShellRepresentationSuccessor(root, ShellTeachingRepresentation); return e },
		func() error { _, e := PlanSharedStructuredRepresentation(root, Reference{}); return e },
	} {
		if err := run(); err == nil || !strings.Contains(err.Error(), "mutation=unknown") {
			t.Fatal("pending recovery bypassed", err)
		}
	}
	if _, err := os.Stat(pathJoin(root, SharedReattestationPending)); err != nil {
		t.Fatal("pending erased", err)
	}
}

func TestSharedReattestationSurface008Isolation(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	real, request, _ := shared008Request(t)
	base := Reference{installedShared008Receipt, installedShared008ReceiptSHA}
	if _, err := os.Stat(pathJoin(real, base.Path)); os.IsNotExist(err) {
		t.Skip("installed evidence unavailable")
	}
	root, extend := mirrorShared008Authority(t, real, request)
	var r SharedRepresentationSuccessor
	if err := readReference(real, base, &r); err != nil {
		t.Fatal(err)
	}
	extend(sharedRepresentationHead(r.Request.Package, r.Request.AssetID))
	extend("data/glossary-corpus-history")
	// This existing family selects its installed producer by exact corpus SHA
	// over a guarded directory inventory. A subset would spuriously bootstrap
	// through today's whole-corpus parser instead of its frozen projection.
	extend(strings.TrimSuffix(CorpusSuccessorRoot, "/"))
	// Derived consumers (including SEO) resolve their own scopes through the
	// complete compatibility graph, beyond a TU finalization's referenced edges.
	extend("data/glossary-compatibility/zh-CN")
	// Unified Review eligibility also selects from a complete guarded directory;
	// the latest current review need not be referenced by an older Generation.
	reviews := "data/unified-glossary-reviews/zh-CN"
	extend(reviews)
	// Retained Tour uses its independent native checkpoint selector rather than
	// a shared representation receipt. Preserve that exact discovery inventory.
	extend("data/locale-surface-reviews/zh-CN")
	extend("locales/zh-CN/candidates")
	for _, path := range []string{"internal/tour/languages.go", "internal/tour/project.go", "internal/tour/seo.go"} {
		extend(path)
	}
	entries, err := os.ReadDir(pathJoin(real, reviews))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		var review i18n.UnifiedGlossaryReview
		b := mustSharedFixtureBytes(t, real, reviews+"/"+entry.Name())
		if err := StrictJSON(b, &review); err != nil {
			t.Fatal(err)
		}
		extend("data/unified-glossary-reviews/zh-CN-bundles/" + review.BundleSHA + ".zip")
	}
	for _, path := range sharedReattestationValidationPaths {
		extend(path)
	}
	// Runtime machine authority reads implementation bytes, while current TU
	// authorization discovers all sealed epochs in its own selector. Neither is
	// necessarily reachable from an older Generation's References. Transporting
	// these module files does not add them to any receipt's dependency envelope.
	for _, dir := range []string{"internal/i18n", "internal/sitecontent", "data/translation-authorizations/zh-CN"} {
		if err := filepath.WalkDir(pathJoin(real, dir), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			if strings.HasPrefix(dir, "internal/") && (!strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go")) {
				return nil
			}
			rel, err := filepath.Rel(real, path)
			if err != nil {
				return err
			}
			extend(filepath.ToSlash(rel))
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	// Canonical TU currentness discovers finalized Snapshots by directory, not
	// just via an older closure's References. Preserve the complete metadata
	// selector inventory; candidate/batch bytes still follow declared edges.
	qcDir := "data/quality-check-snapshots/zh-CN"
	snapshots, err := os.ReadDir(pathJoin(real, qcDir))
	if err != nil {
		t.Fatal(err)
	}
	for _, snapshot := range snapshots {
		if !snapshot.IsDir() {
			continue
		}
		for _, name := range []string{"manifest.json", "finalization.json", "quality-check-results.json"} {
			path := qcDir + "/" + snapshot.Name() + "/" + name
			if _, err := os.Stat(pathJoin(real, path)); err == nil {
				extend(path)
			} else if !os.IsNotExist(err) {
				t.Fatal(err)
			}
		}
	}
	// Finalization and Snapshot encode some repository evidence by symbolic
	// Snapshot/batch IDs. Follow only those declared edges, including nodes that
	// the readonly reader discovers later; missing batch evidence becomes pending
	// A-only coverage instead of returning an os.PathError.
	loadedBatches := map[string]bool{}
	completeQCEvidence := func() {
		t.Helper()
		for pass := 0; pass < 40; pass++ {
			manifests, batches := map[string]bool{}, map[string]bool{}
			if err := filepath.WalkDir(pathJoin(root, "data/quality-check-snapshots/zh-CN"), func(path string, entry os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() {
					return nil
				}
				switch entry.Name() {
				case "finalization.json":
					b, err := os.ReadFile(path)
					if err != nil {
						return err
					}
					var f i18n.QualityCheckFinalization
					if err := json.Unmarshal(b, &f); err != nil {
						return err
					}
					for _, input := range f.QCResults {
						manifests[input.SnapshotID] = true
					}
				case "manifest.json":
					b, err := os.ReadFile(path)
					if err != nil {
						return err
					}
					var snapshot i18n.QualityCheckSnapshotManifest
					if err := json.Unmarshal(b, &snapshot); err != nil {
						return err
					}
					for _, unit := range snapshot.Units {
						if unit.RepresentationReconciliation == nil && unit.SelectedBatchID != "" {
							batches[unit.SelectedBatchID] = true
						}
					}
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			added := false
			for snapshot := range manifests {
				path := "data/quality-check-snapshots/zh-CN/" + snapshot + "/manifest.json"
				if _, err := os.Stat(pathJoin(root, path)); os.IsNotExist(err) {
					extend(path)
					added = true
				} else if err != nil {
					t.Fatal(err)
				}
			}
			for batch := range batches {
				if loadedBatches[batch] {
					continue
				}
				for _, name := range []string{"manifest.json", "result.json"} {
					extend("data/retranslation-runs/zh-CN/" + batch + "/" + name)
				}
				// Retry validation discovers the complete per-unit attempt
				// directory; earlier audits are not explicit manifest References.
				batchDir := "data/retranslation-runs/zh-CN/" + batch
				// Read only file inventory, including the supported legacy Page
				// transport shape. The production decoder still validates and
				// normalizes the original bytes; no substitute manifest is written.
				type inputFile struct {
					InputPath string `json:"input_path"`
				}
				var batchManifest struct {
					Units []inputFile `json:"units"`
					Pages []inputFile `json:"pages"`
				}
				if err := json.Unmarshal(mustSharedFixtureBytes(t, real, batchDir+"/manifest.json"), &batchManifest); err != nil {
					t.Fatal(err)
				}
				for _, unit := range append(batchManifest.Units, batchManifest.Pages...) {
					name := filepath.Base(unit.InputPath)
					retries := batchDir + "/retries/" + strings.TrimSuffix(name, filepath.Ext(name))
					if _, err := os.Stat(pathJoin(real, retries)); err == nil {
						extend(retries)
					} else if !os.IsNotExist(err) {
						t.Fatal(err)
					}
				}
				loadedBatches[batch] = true
				added = true
			}
			if !added {
				return
			}
		}
		t.Fatal("bounded QC repository evidence closure exceeded")
	}
	// Copy only an exact missing dependency reported by the existing reader.
	// No request, source, target, Generation or authority is synthesized here.
	complete := func(run func() error) {
		t.Helper()
		seen := map[string]bool{}
		for attempt := 0; attempt < 40; attempt++ {
			err := run()
			if err == nil {
				return
			}
			// The checkpoint deliberately hides failed Surface discovery from its
			// caller. Expose the same readonly reader's missing fixture path here.
			if strings.Contains(err.Error(), "exact-current Tour Surface checkpoint required") {
				catalog, e := checkedCatalog(root)
				if e != nil {
					t.Fatal(e)
				}
				if _, e := i18n.CurrentExactLocaleSurfaceReviewAGate(root, "zh-CN", catalog); e != nil {
					err = e
				}
			}
			var missing *os.PathError
			if !errors.As(err, &missing) || !os.IsNotExist(missing.Err) {
				t.Fatal(err)
			}
			rel, e := filepath.Rel(root, missing.Path)
			if e != nil || !validPath(filepath.ToSlash(rel)) {
				t.Fatal(err)
			}
			if seen[rel] {
				t.Fatal("missing authority unavailable in exact real graph", rel, err)
			}
			seen[rel] = true
			t.Log("isolated reader requires", rel)
			extend(filepath.ToSlash(rel))
			completeQCEvidence()
		}
		t.Fatal("bounded isolated dependency closure exceeded")
	}
	// Freeze the reader's full source world before re-attestation, so the
	// independently authorized pending source work can be proved beforehand.
	manifest, err := loadSourceManifest(real)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range manifest.Files {
		if _, err := os.Stat(pathJoin(root, f.Path)); os.IsNotExist(err) {
			extend(f.Path)
		} else if err != nil {
			t.Fatal(err)
		}
	}
	completeQCEvidence()
	var sourceRef Reference
	var source *StructuredSourceSuccessor
	for _, ref := range r.SourceAuthority {
		if !strings.HasPrefix(ref.Path, StructuredSourceSuccessorRoot+"receipts/") {
			continue
		}
		s, err := CheckStructuredSourceSuccessor(root, ref)
		if err != nil {
			t.Fatal(err)
		}
		if s.Current.Path == "data/site-shell/translation.html" {
			sourceRef, source = ref, s
		}
	}
	if source == nil {
		t.Fatal("pre-existing translation source authority missing")
	}
	var locale Locale
	if err := StrictJSON(mustSharedFixtureBytes(t, root, LocalePath("zh-CN")), &locale); err != nil {
		t.Fatal(err)
	}
	validation := newClosureValidation(root)
	var baseline *structuredBaseline
	complete(func() error {
		for _, c := range locale.Packages {
			if c.Package == source.Package {
				var err error
				baseline, err = validation.historicalStructuredCompletion(root, "zh-CN", c)
				return err
			}
		}
		return fmt.Errorf("pre-existing source completion missing")
	})
	preexistingDelta, err := sourceDeltaFromBaseline(root, "zh-CN", *source, baseline)
	if err != nil || len(preexistingDelta.Pending) != 52 {
		t.Fatal("pre-existing source delta", preexistingDelta, err)
	}
	prior := &StructuredAssessment{}
	priorAsset := StructuredAsset{ID: "source:" + source.Current.Path, Package: source.Package, Document: &source.Current}
	matched, err := validation.sourceExpansionAssessment(root, StructuredPlan{Locale: "zh-CN"}, priorAsset, baseline, prior)
	if err != nil || !matched || prior.Classification != "affected" || prior.SourceSuccessor == nil || *prior.SourceSuccessor != sourceRef {
		t.Fatal("source obligation not independently established before re-attestation", prior, err)
	}
	if err := validation.verify(); err != nil {
		t.Fatal(err)
	}

	var p *SharedRepresentationReattestationPlan
	complete(func() error { var err error; p, err = PlanSharedRepresentationReattestation(root, base); return err })
	if !p.Eligible || p.Authority.BaseIdentity != r.Identity || p.Authority.RepresentationIdentity != r.NewIdentity || !reflect.DeepEqual(p.Authority.Base, r) {
		t.Fatal("real sealed base invariant changed")
	}
	input := landSharedReattestationFixture(t, root, "data/synthetic-shared-reattestation-plan.json", p)
	complete(func() error { _, err := InspectSharedRepresentationReattestation(root, input); return err })
	if _, err := ApplySharedRepresentationReattestation(root, input, p.PriorPointerSHA); err != nil {
		t.Fatal(err)
	}
	var plan *StructuredPlan
	complete(func() error { var err error; plan, err = PlanStructuredAssets(root, "zh-CN"); return err })
	want := []string{"locales/zh-CN/structured/_content/site.tmpl", "locales/zh-CN/structured/data/site-shell/translation.html"}
	if !reflect.DeepEqual(plan.Expected, want) {
		t.Fatal("non-exact global pending obligations", plan.Expected)
	}
	others, representationAffected := 0, 0
	var translation *StructuredAsset
	for _, a := range plan.Assets {
		if a.Assessment != nil && a.Assessment.RepresentationSuccessor != nil {
			representationAffected++
			if a.ID != r.Request.AssetID || *a.Assessment.RepresentationSuccessor != base {
				t.Fatal("representation reopened another asset", a.ID)
			}
		}
		if a.ID == r.Request.AssetID {
			if a.State != "generation-required" || !reflect.DeepEqual(a.Document, ptrSharedDocument(r)) {
				t.Fatal("affected representation/state", a.State)
			}
			for _, u := range a.Document.Units {
				for _, p := range u.Protected {
					if p.Raw == "activate/deactivate" {
						t.Fatal("released prose refrozen")
					}
				}
			}
		} else if a.ID == priorAsset.ID {
			copy := a
			translation = &copy
			if a.State != "generation-required" || a.Assessment == nil || a.Assessment.SourceSuccessor == nil || *a.Assessment.SourceSuccessor != sourceRef || a.Assessment.RepresentationSuccessor != nil || a.Assessment.Classification != prior.Classification || a.Assessment.Reason != prior.Reason {
				t.Fatal("pre-existing obligation lost or attributed to representation", a)
			}
		} else {
			others++
			if a.State != "carry" && a.State != "context-only" {
				t.Fatal("unaffected asset reopened", a.ID, a.State)
			}
		}
	}
	if others != 12 || representationAffected != 1 || translation == nil {
		t.Fatal("unaffected structured inventory", others)
	}
	t.Run("revision-finding-narrowing", func(t *testing.T) {
		findingRef := Reference{"data/locale-language/zh-CN/surface/20261031-zh-CN-integrated-surface-008.json", "b525bc49d7e50625c245c647c39dd666c2ca3cdd64974190047c28b0c700b18f"}
		extend(findingRef.Path)
		var finding IntegratedSurfaceReceipt
		if err := readReference(root, findingRef, &finding); err != nil {
			t.Fatal(err)
		}
		if len(finding.ExactFindings) != 1 || finding.ExactFindings[0].ID != r.Request.Path || finding.ExactFindings[0].Class != "structured-surface-language" {
			t.Fatal("exact finding semantics changed")
		}
		// Selection alone skipped the real export's failing currentness gate.
		// Consume the original sealed finding through that complete reader,
		// including ordinary scope rejection and exact successor predecessor
		// proof. All publication above belongs to this disposable mirror.
		var validated *IntegratedSurfaceReceipt
		complete(func() error {
			var err error
			validated, err = validatedSurfaceFinding(root, "zh-CN", findingRef)
			return err
		})
		if !reflect.DeepEqual(validated, &finding) {
			t.Fatal("historical judgment rewritten")
		}
		globalBytes, _ := Encode(plan)
		var revision StructuredPlan
		if err := StrictJSON(globalBytes, &revision); err != nil {
			t.Fatal(err)
		}
		// This is the production structured-export --task revision selector,
		// including its bounded shared owner/currentness resolver.
		if err := selectStructuredSurfaceRevision(root, &revision, validated); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(revision.Expected, []string{want[0]}) {
			t.Fatal("revision finding expanded output scope", revision.Expected)
		}
		found := false
		for _, a := range revision.Assets {
			if a.ID == translation.ID {
				found = true
				if a.State != "context-only" || !reflect.DeepEqual(a.Assessment, translation.Assessment) || !reflect.DeepEqual(a.Document, translation.Document) || a.TargetPath != translation.TargetPath {
					t.Fatal("revision context falsely completed independent source work")
				}
			}
		}
		if !found {
			t.Fatal("pending source context disappeared")
		}
		if after, _ := Encode(plan); !bytes.Equal(globalBytes, after) || translation.State != "generation-required" {
			t.Fatal("revision selection mutated global pending state")
		}
		t.Log("FULL_VALIDATED_SURFACE_FINDING=PASS REVISION_EXPORT_SELECTION=PASS REVISION_EXPECTED_COUNT=1 REVISION_EXPECTED_OUTPUTS=locales/zh-CN/structured/_content/site.tmpl TRANSLATION_HTML_GLOBAL_PENDING_PRESERVED=yes")
	})
	if _, _, _, err := sharedRepresentationConsumedGeneration(root, r.Request.Generation, newClosureValidation(root)); err != nil {
		t.Fatal("historical frozen source consumption", err)
	}
	for _, ref := range []Reference{base, r.Request.Generation} {
		before, err := readRegular(real, ref.Path)
		if err != nil {
			t.Fatal(err)
		}
		after, err := readRegular(root, ref.Path)
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal("historical bytes rewritten", ref.Path, err)
		}
	}
	physicalHead := sharedRepresentationHead(r.Request.Package, r.Request.AssetID)
	b, _ := readRegular(real, physicalHead)
	a, _ := readRegular(root, physicalHead)
	if !bytes.Equal(b, a) {
		t.Fatal("base head changed")
	}
	t.Log("GLOBAL_CURRENT_STRUCTURED_PLAN=PASS GLOBAL_EXPECTED_COUNT=2 REPRESENTATION_SUCCESSOR_AFFECTED_COUNT=1 REPRESENTATION_SUCCESSOR_AFFECTED_ASSET=source:_content/site.tmpl PREEXISTING_SOURCE_DELTA_ASSET=source:data/site-shell/translation.html PREEXISTING_SOURCE_DELTA_PENDING_SLOTS=52")
}

func ptrSharedDocument(r SharedRepresentationSuccessor) *Document {
	d := sharedRepresentationDocument(r)
	return &d
}

// Diagnose a pre-existing source-language delta independently of the shared
// representation selector. This reads sealed source/completion/history only;
// it neither publishes authority nor fabricates a completed successor world.
func TestSharedReattestationExistingSourceDeltaDiagnosis(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	if _, err := os.Stat(pathJoin(root, installedShared008Receipt)); os.IsNotExist(err) {
		t.Skip("installed evidence unavailable")
	}
	var base SharedRepresentationSuccessor
	if err := readReference(root, Reference{installedShared008Receipt, installedShared008ReceiptSHA}, &base); err != nil {
		t.Fatal(err)
	}
	var locale Locale
	if err := StrictJSON(mustSharedFixtureBytes(t, root, LocalePath("zh-CN")), &locale); err != nil {
		t.Fatal(err)
	}
	v := newClosureValidation(root)
	for _, completion := range locale.Packages {
		if completion.Package != base.Request.Package {
			continue
		}
		baseline, err := v.historicalStructuredCompletion(root, "zh-CN", completion)
		if err != nil {
			t.Fatal(err)
		}
		for _, ref := range base.SourceAuthority {
			if !strings.HasPrefix(ref.Path, StructuredSourceSuccessorRoot+"receipts/") {
				continue
			}
			a, err := CheckStructuredSourceSuccessor(root, ref)
			if err != nil {
				t.Fatal(err)
			}
			delta, err := sourceDeltaFromBaseline(root, "zh-CN", *a, baseline)
			if err != nil {
				t.Fatal(err)
			}
			if len(delta.Pending) == 0 {
				t.Fatal("expected nonempty independently authorized source delta")
			}
			t.Logf("PREEXISTING_SOURCE_ASSET=source:%s BASELINE_CLOSURE=%s SOURCE_AUTHORITY=%s PENDING_SOURCE_LANGUAGE_SLOTS=%d", a.Current.Path, baseline.ClosureRef.Path, ref.Path, len(delta.Pending))
			if err := v.verify(); err != nil {
				t.Fatal(err)
			}
			return
		}
	}
	t.Fatal("existing source/completion authority not found")
}
