package sitecontent

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func integratedReviewerFixture(t *testing.T) (string, IntegratedSurfaceReviewerOptions) {
	t.Helper()
	// The performance fixture is shell-only. This Bundle test needs a real
	// current A-only Page checkpoint plus a structured baseline/delta. English
	// source-identical comments use their legitimate English locale authority.
	const locale = "en-GB"
	root, _, initial, page := structuredLifecycleFixture(t, "learn-docs-v1", locale)
	// Match the Learn structured baseline/delta world that this Bundle test
	// exercises; the shell-only performance fixture has a different delta.
	const term = "The Go Programming Language"
	structuredGlossarySuccessorWithLabel(t, root, term, "Synthetic replacement label", "structured-current-review", locale)
	bundle, manifest, err := ExportStructuredAssets(root, locale)
	if err != nil {
		t.Fatal(err)
	}
	var old StructuredGeneration
	if err := readReference(root, initial, &old); err != nil {
		t.Fatal(err)
	}
	output := StructuredOutput{Schema: StructuredSchema, BundleSHA: manifest.Identity, Outputs: []StructuredTarget{}}
	for _, wanted := range manifest.Plan.Expected {
		for _, target := range old.Output.Outputs {
			if target.Path == wanted {
				target.Slots = append([]Target(nil), target.Slots...)
				for i := range target.Slots {
					target.Slots[i].Text = strings.ReplaceAll(target.Slots[i].Text, term, "Synthetic replacement label")
				}
				output.Outputs = append(output.Outputs, target)
			}
		}
	}
	raw, _ := Encode(output)
	_, deltaPath, err := ImportStructuredAssets(root, bundle, raw, old.Provenance)
	if err != nil {
		t.Fatal(err)
	}
	delta, err := ReferenceFile(root, deltaPath)
	if err != nil {
		t.Fatal(err)
	}
	generations := []Reference{initial, delta}
	for _, name := range append(append([]string{}, integratedSurfaceComposition...), integratedSurfaceReviewerAuthorities...) {
		b, err := readRegular(repositoryRoot(t), name)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, name, b)
	}
	// Only Learn/Docs consumes this delta. The shell uses its complete original
	// Generation, independently current under the unchanged shell glossary.
	_, name, err := FinalizePackageClosure(root, locale, "learn-docs-v1", page, generations)
	if err != nil {
		t.Fatal(err)
	}
	learn, err := ReferenceFile(root, name)
	if err != nil {
		t.Fatal(err)
	}
	_, shellPath, err := FinalizePackageClosure(root, locale, "site-v2-shell", nil, []Reference{initial})
	if err != nil {
		t.Fatal(err)
	}
	shell, err := ReferenceFile(root, shellPath)
	if err != nil {
		t.Fatal(err)
	}
	return root, IntegratedSurfaceReviewerOptions{Locale: locale, Closures: []Reference{learn, shell}}
}

func TestIntegratedSurfaceReviewerBundle(t *testing.T) {
	root, o := integratedReviewerFixture(t)
	b, m, err := ExportIntegratedSurfaceReviewerBundle(root, o)
	if err != nil {
		t.Fatal(err)
	}
	parsed, files, err := readIntegratedSurfaceReviewerBundle(b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m, parsed) || len(m.Closures) != 2 || m.ReviewRequiredCount != 3 || m.CarriedCount != 55 || len(m.Fresh) != 3 || len(m.Carried) != 55 {
		t.Fatalf("wrong exact fresh/carried scope: fresh=%d carried=%d payloads=%d/%d", m.ReviewRequiredCount, m.CarriedCount, len(m.Fresh), len(m.Carried))
	}
	if _, ok := files[i18n.LanguageAttentionPath]; !ok {
		t.Fatal("Surface Reviewer missing complete language attention")
	}
	// The Learn book delta is fresh, the other 49 Pages + six data documents
	// carry; the two full prospective shell documents remain fresh.
	for i, id := range []string{"_content/learn/books.yaml", "data/site-shell/home.html", "data/site-shell/translation.html"} {
		if m.Fresh[i].Asset.ID != id {
			t.Fatal("fresh scope escaped the fixture authority", m.Fresh[i].Asset.ID)
		}
	}
	other, _, err := ExportIntegratedSurfaceReviewerBundle(root, IntegratedSurfaceReviewerOptions{Scope: files[m.Scope.BundlePath]})
	if err != nil || !bytes.Equal(b, other) {
		t.Fatal("nondeterministic canonical ZIP", err)
	}
	if _, err := CheckIntegratedSurfaceReviewerBundle(root, b); err != nil {
		t.Fatal("current-check", err)
	}
	pagePayloads, structured := 0, 0
	for _, p := range append(append([]IntegratedSurfaceReviewerPayload{}, m.Fresh...), m.Carried...) {
		if len(files[p.Source.BundlePath]) == 0 || digest(files[p.Source.BundlePath]) != p.Source.SHA256 || digest(files[p.Target.BundlePath]) != p.Asset.TargetSHA || p.Representation == nil {
			t.Fatal("missing complete source/target/representation")
		}
		if len(p.Asset.CandidateArtifacts) > 0 {
			pagePayloads++
			if len(p.Provenance) < 3 {
				t.Fatal("missing Page Snapshot/candidate provenance")
			}
		}
		if len(p.Generations) > 0 {
			structured++
		}
		if p.Role == "fresh_review_required" && p.Asset.State == "carry" {
			t.Fatal("carried promoted to fresh")
		}
	}
	if pagePayloads != 49 || structured != 9 {
		t.Fatal("incomplete Page/structured payload", pagePayloads, structured)
	}
	for _, name := range []string{"evidence/" + m.Glossary.Path, "evidence/" + m.UnifiedReview.Path, "integration/existing-coverage.json", "integration/canonical-routes.json", "evidence/production/identity.json", "integration/internal/site/runtime.go", "integration/internal/site/shell_render.go", "authority/docs/INTEGRATED_SURFACE_REVIEW.md"} {
		if len(files[name]) == 0 {
			t.Fatal("missing integration material", name)
		}
	}
	var scope IntegratedSurfaceScope
	if err := StrictJSON(files[m.Scope.BundlePath], &scope); err != nil {
		t.Fatal(err)
	}
	canonical, _ := Encode(scope)
	if digest(canonical) != m.Scope.SHA256 {
		t.Fatal("scope SHA mismatch")
	}
	// Transport corruption fails before any expensive current graph traversal.
	for _, kind := range []string{"missing", "extra", "tampered-payload", "manifest", "scope"} {
		t.Run(kind, func(t *testing.T) {
			entries := []i18n.TransportBundleEntry{}
			manifest := bytes.Clone(files["manifest.json"])
			for name, data := range files {
				if name == "manifest.json" {
					continue
				}
				if kind == "missing" && name == m.Fresh[0].Target.BundlePath {
					continue
				}
				if kind == "tampered-payload" && name == m.Fresh[0].Target.BundlePath {
					data = append(bytes.Clone(data), 'x')
				}
				if kind == "scope" && name == m.Scope.BundlePath {
					data = append(bytes.Clone(data), 'x')
				}
				entries = append(entries, i18n.TransportBundleEntry{Path: name, Data: data})
			}
			if kind == "extra" {
				entries = append(entries, i18n.TransportBundleEntry{Path: "unexpected.txt", Data: []byte("unexpected\n")})
			}
			if kind == "manifest" {
				manifest = bytes.Replace(manifest, []byte(m.Identity), []byte(strings.Repeat("0", 64)), 1)
			}
			bad, err := i18n.WriteDeterministicTransportBundle(manifest, entries)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := CheckIntegratedSurfaceReviewerBundle(root, bad); err == nil {
				t.Fatal("corrupt transport accepted")
			}
		})
	}
	// Every row remains bound to current formal repository evidence. This
	// legacy synthetic fixture's source authority is SnapshotPath, not the
	// incidental root copies used by some shared Page adapters.
	fresh := m.Fresh[0]
	candidate := m.Carried[0].Asset.CandidateArtifacts[0].Unit.CandidatePath
	for kind, name := range map[string]string{"closure": m.Closures[0].Path, "glossary": m.Glossary.Path, "review": m.UnifiedReview.Path, "candidate": candidate, "structured-target": fresh.Generations[0].Path, "runtime": "internal/tour/project.go", "public": "production/identity.json", "source": SnapshotPath, "semantic-authority": "docs/INTEGRATED_SURFACE_REVIEW.md"} {
		t.Run(kind+"-drift", func(t *testing.T) {
			before, err := readRegular(root, name)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, name, before)
			writeFixture(t, root, name, append(bytes.Clone(before), []byte("tamper\n")...))
			if _, err := CheckIntegratedSurfaceReviewerBundle(root, b); err == nil {
				t.Fatal("stale formal input accepted")
			}
		})
	}
	for _, locale := range []string{"fr-FR", "ja-JP"} {
		t.Run(locale+"-scope-input", func(t *testing.T) {
			s := scope
			s.Locale = locale
			s.Identity = ""
			s.Identity = identity(s)
			raw, _ := Encode(s)
			got, _, _, err := integratedSurfaceBundleInputs(IntegratedSurfaceReviewerOptions{Scope: raw})
			if err != nil || got != locale {
				t.Fatal("locale-specific input handling", err)
			}
			if _, _, _, err := integratedSurfaceBundleInputs(IntegratedSurfaceReviewerOptions{Locale: "zh-CN", Scope: raw}); err == nil {
				t.Fatal("mismatched locale accepted")
			}
		})
	}
	bad := scope
	bad.Routes = append(append([]string{}, scope.Routes...), "/forged/")
	bad.Identity = ""
	bad.Identity = identity(bad)
	raw, _ := Encode(bad)
	if _, _, err := ExportIntegratedSurfaceReviewerBundle(root, IntegratedSurfaceReviewerOptions{Scope: raw}); err == nil {
		t.Fatal("self-consistent but stale scope accepted")
	}
	// Execution instructions are not a new semantic dependency for this transport.
	before, _ := readRegular(root, "docs/CHATGPT_LANGUAGE_GENERATION.md")
	writeFixture(t, root, "docs/CHATGPT_LANGUAGE_GENERATION.md", append(bytes.Clone(before), []byte("\nSynthetic execution-only handoff clarification.\n")...))
	if _, err := CheckIntegratedSurfaceReviewerBundle(root, b); err != nil {
		t.Fatal("execution-only docs staled language review transport", err)
	}
}

// Audit the actual exported transport without regenerating it or doing another
// expensive current-check; the CLI owns that independent read-only check.
func TestIntegratedSurfaceReviewerRealBundle(t *testing.T) {
	name := os.Getenv("GO_TOUR_INTEGRATED_SURFACE_BUNDLE")
	if name == "" {
		t.Skip("explicit read-only exported Integrated Surface ZIP")
	}
	b, err := readRegular(repositoryRoot(t), name)
	if err != nil {
		t.Fatal(err)
	}
	m, files, err := readIntegratedSurfaceReviewerBundle(b)
	if err != nil {
		t.Fatal(err)
	}
	if m.ScopeIdentity != "faa455c829fd1ae7bb3454bd0df60a65ee099262fc056c779444c5f81028f1b3" || m.Scope.SHA256 != "bd033c9743c0e3193ddc2be7d7874951d1d5c28897c6596abab32fa8d5dea974" || m.ReviewRequiredCount != 14 || m.CarriedCount != 518 {
		t.Fatal("real scope/count/identity drift")
	}
	homepage, structured, page := false, 0, 0
	for _, p := range m.Fresh {
		if p.Asset.ID == "_content/index.md" {
			homepage = p.Asset.State == "context-review-required" && len(p.Asset.CandidateArtifacts) == 1 && bytes.Contains(files[p.Source.BundlePath], []byte("package main")) && bytes.Contains(files[p.Target.BundlePath], []byte("package main"))
		}
		if len(p.Generations) > 0 {
			structured++
		}
		if len(p.Asset.CandidateArtifacts) > 0 {
			page++
		}
	}
	if !homepage || structured != 9 || page != 5 {
		t.Fatal("Homepage/shared structured/Page payload missing", homepage, structured, page)
	}
	if len(files["integration/context-only-tour.json"]) == 0 {
		t.Fatal("missing carried Tour integration context")
	}
	var effective Global
	if err := StrictJSON(files["integration/effective-content-scope.json"], &effective); err != nil {
		t.Fatal(err)
	}
	if effective.ShellAdoption == nil || len(files["integration/"+ShellAdoptionCurrent]) == 0 || len(files["evidence/"+effective.ShellAdoption.Path]) == 0 {
		t.Fatal("missing effective source-adoption composition context")
	}
	t.Logf("fresh=%d carried=%d bytes=%d sha=%s", m.ReviewRequiredCount, m.CarriedCount, len(b), digest(b))
}
