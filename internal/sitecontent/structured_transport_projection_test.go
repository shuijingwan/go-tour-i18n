package sitecontent

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func structuredTransportReplacementFixture(t *testing.T) (string, Reference) {
	t.Helper()
	root, _, _, _ := structuredLifecycleFixture(t, "site-v2-shell")
	name := "data/site-shell/translation.html"
	old, err := readRegular(root, name)
	if err != nil {
		t.Fatal(err)
	}
	at := bytes.LastIndex(old, []byte("</main>"))
	if at < 0 {
		t.Fatal("synthetic insertion boundary missing")
	}
	expanded := append(bytes.Clone(old[:at]), []byte("<section><h2>Synthetic addition</h2><p>Additional structured source.</p></section>\n")...)
	expanded = append(expanded, old[at:]...)
	writeFixture(t, root, "data/expanded.html", expanded)
	body, _ := ReferenceFile(root, "data/expanded.html")
	proposal := StructuredSourceProposal{Schema: StructuredSourceSuccessorSchema, Package: "site-v2-shell", Previous: Reference{name, digest(old)}, Body: body, InsertAt: at}
	b, _ := Encode(proposal)
	writeFixture(t, root, "data/proposal.json", b)
	ref, _ := ReferenceFile(root, "data/proposal.json")
	a, err := BuildStructuredSourceSuccessor(root, ref)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = AdoptStructuredSourceSuccessor(root, ref, a.BaseIdentity); err != nil {
		t.Fatal(err)
	}
	authority, _ := ReferenceFile(root, StructuredSourceSuccessorPath(a))
	corpus, _ := readRegular(root, i18n.GlossaryCorpusPath)
	if _, err = AdoptCorpusProjection(root, digest(corpus), true); err != nil {
		t.Fatal(err)
	}
	review, _, err := i18n.ExportUnifiedGlossaryBundle(root, "zh-CN", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = i18n.RecordUnifiedGlossaryReview(root, review, "transport-source-review", "synthetic-independent-reviewer", "synthetic-generation", "passed", nil); err != nil {
		t.Fatal(err)
	}
	return root, authority
}

func TestStructuredTransportProjectionCurrentness(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, successor := structuredTransportReplacementFixture(t)
	bundle, m, err := ExportStructuredAssets(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	if m.TransportProfile != StructuredTransportProfile || len(m.ValidationInputs) == 0 || !reflect.DeepEqual(m.Plan.Expected, []string{"locales/zh-CN/structured/data/site-shell/translation.html"}) {
		t.Fatal("nonexact Generation scope", m.Plan.Expected, m.ValidationInputs)
	}
	for _, a := range m.Plan.Assets {
		if a.State == "carry" {
			for _, p := range m.Plan.Expected {
				if p == a.TargetPath {
					t.Fatal("carry became Generation output")
				}
			}
		}
	}
	if _, err = CheckStructuredAssetsBundle(root, bundle); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"source authority", "baseline closure", "source bytes", "current Review", "archive bytes", "archive missing"} {
		t.Run(kind, func(t *testing.T) {
			path := successor.Path
			switch kind {
			case "baseline closure":
				for _, a := range m.Plan.Assets {
					if a.Assessment != nil {
						path = a.Assessment.Closure.Path
						break
					}
				}
			case "source bytes":
				var a StructuredSourceSuccessor
				if err := readReference(root, successor, &a); err != nil {
					t.Fatal(err)
				}
				path = a.Body.Path
			case "current Review":
				path = m.Plan.Review.Path
			case "archive bytes", "archive missing":
				path = m.ValidationInputs[0].Path
			}
			old, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "archive missing" {
				if err = os.Remove(pathJoin(root, path)); err != nil {
					t.Fatal(err)
				}
			} else {
				writeFixture(t, root, path, append(bytes.Clone(old), ' '))
			}
			defer writeFixture(t, root, path, old)
			if _, err = CheckStructuredAssetsBundle(root, bundle); err == nil {
				t.Fatal("currentness accepted", kind)
			}
		})
	}
	for _, kind := range []string{"semantic missing", "provenance omitted", "provenance SHA", "provenance extra", "inventory missing", "inventory extra", "inventory tamper", "source binding", "profile downgrade", "legacy omission"} {
		t.Run(kind, func(t *testing.T) {
			bad := rewriteStructuredTransportRaw(t, bundle, func(m *StructuredBundle, files map[string][]byte) {
				switch kind {
				case "semantic missing":
					p := "inputs/locales/zh-CN/glossary.yaml"
					delete(files, p)
					kept := m.Files[:0]
					for _, f := range m.Files {
						if f.BundlePath != p {
							kept = append(kept, f)
						}
					}
					m.Files = kept
				case "provenance omitted":
					m.ValidationInputs = m.ValidationInputs[1:]
				case "provenance SHA":
					m.ValidationInputs[0].SHA256 = strings.Repeat("0", 64)
				case "provenance extra":
					m.ValidationInputs = append(m.ValidationInputs, Reference{"data/locale-language/zh-CN/structured-bundles/" + strings.Repeat("0", 64) + ".zip", strings.Repeat("0", 64)})
					sort.Slice(m.ValidationInputs, func(i, j int) bool { return m.ValidationInputs[i].Path < m.ValidationInputs[j].Path })
				case "inventory missing":
					delete(files, m.Files[0].BundlePath)
				case "inventory extra":
					files["extra.txt"] = []byte("extra")
				case "inventory tamper":
					files[m.Files[0].BundlePath] = []byte("tamper")
				case "profile downgrade":
					m.TransportProfile = ""
				case "legacy omission":
					m.TransportProfile, m.ValidationInputs = "", nil
				case "source binding":
					for i := range m.Files {
						if strings.HasPrefix(m.Files[i].BundlePath, "source/") {
							m.Files[i].RepositoryPath = "data/unrelated.html"
						}
					}
				}
			})
			if _, err := CheckStructuredAssetsBundle(root, bad); err == nil {
				t.Fatal("forgery accepted", kind)
			}
		})
	}
	if _, err := readStructuredBundleWithDuplicate(t, bundle); err == nil {
		t.Fatal("duplicate transport accepted")
	}
	// The original bound gate is unchanged and precedes repository traversal.
	enormous, err := i18n.WriteDeterministicTransportBundle([]byte("{}"), []i18n.TransportBundleEntry{{Path: "oversized", Data: make([]byte, 32<<20)}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CheckStructuredAssetsBundle(t.TempDir(), enormous); err == nil || !strings.Contains(err.Error(), "exceeds 33554432") {
		t.Fatal("size gate weakened", err)
	}
}

func rewriteStructuredTransportRaw(t *testing.T, b []byte, change func(*StructuredBundle, map[string][]byte)) []byte {
	t.Helper()
	m, files, err := readStructuredBundle(b)
	if err != nil {
		t.Fatal(err)
	}
	change(m, files)
	m.Identity = ""
	m.Identity = identity(*m)
	manifest, _ := Encode(m)
	entries := []i18n.TransportBundleEntry{}
	for name, data := range files {
		if name != "manifest.json" {
			entries = append(entries, i18n.TransportBundleEntry{Path: name, Data: data})
		}
	}
	out, err := i18n.WriteDeterministicTransportBundle(manifest, entries)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// Duplicate archive construction is deliberately test-only, bypassing the
// canonical writer so the reader's own duplicate guard is exercised.
func readStructuredBundleWithDuplicate(t *testing.T, b []byte) (*StructuredBundle, error) {
	t.Helper()
	fs, err := i18n.ReadTransportBundle(b, 256, 32<<20)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, n := range []string{"manifest.json", "manifest.json"} {
		h := &zip.FileHeader{Name: n, Method: zip.Store}
		h.SetMode(0644)
		f, err := w.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = f.Write(fs[n])
	}
	_ = w.Close()
	m, _, err := readStructuredBundle(buf.Bytes())
	return m, err
}

func TestStructuredTransportProjectionRealPayload(t *testing.T) {
	out := os.Getenv("GO_TOUR_STRUCTURED_PROJECTION_PROOF")
	if out == "" {
		t.Skip("explicit old ZIP payload projection proof")
	}
	root := repositoryRoot(t)
	old, err := readRegular(root, "zh-CN-site-v2-shell-translation-structured-generation-20261025.zip")
	if err != nil {
		t.Fatal(err)
	}
	if len(old) != 65143557 || digest(old) != "16d2ca67babb34205769876a908c2a485df17c970d3ae6ae92293cf143844f3b" {
		t.Fatal("unexpected forensic source ZIP")
	}
	// Bounded forensic decode of a known failed artifact, NOT a product reader
	// exception. The public checker must continue rejecting these oversized bytes.
	fs, err := i18n.ReadTransportBundle(old, 256, 70<<20)
	if err != nil {
		t.Fatal(err)
	}
	var before StructuredBundle
	if err = StrictJSON(fs["manifest.json"], &before); err != nil {
		t.Fatal(err)
	}
	encoded, _ := Encode(before)
	var m StructuredBundle
	_ = StrictJSON(encoded, &m)
	entries := []i18n.TransportBundleEntry{}
	allowed := map[Reference]bool{}
	for _, f := range m.Files {
		entries = append(entries, i18n.TransportBundleEntry{Path: f.BundlePath, Data: fs[f.BundlePath]})
		r := Reference{f.RepositoryPath, f.SHA256}
		if structuredValidationArchive(m.Plan.Locale, r) {
			allowed[r] = true
		}
	}
	doc, err := readRegular(root, structuredTransportContract)
	if err != nil {
		t.Fatal(err)
	}
	p := "inputs/" + structuredTransportContract
	m.Files = append(m.Files, i18n.NewTransportBundleFile(p, structuredTransportContract, doc))
	entries = append(entries, i18n.TransportBundleEntry{Path: p, Data: doc})
	start := time.Now()
	entries, err = projectStructuredArchiveTransport(&m, entries, allowed)
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(m.Files, func(i, j int) bool { return m.Files[i].BundlePath < m.Files[j].BundlePath })
	m.Identity = ""
	m.Identity = identity(m)
	manifest, _ := Encode(m)
	b, err := i18n.WriteDeterministicTransportBundle(manifest, entries)
	if err != nil {
		t.Fatal(err)
	}
	_, after, err := readStructuredBundle(b)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range before.Files {
		r := Reference{f.RepositoryPath, f.SHA256}
		if !allowed[r] && !bytes.Equal(fs[f.BundlePath], after[f.BundlePath]) {
			t.Fatal("semantic/context payload changed", f.BundlePath)
		}
	}
	if !reflect.DeepEqual(before.Plan, m.Plan) || len(m.Plan.Expected) != 1 || len(b) >= 32<<20 {
		t.Fatal("projection changed Generation scope")
	}
	if err := compatibleStructuredBundles(before, fs, m, after); err != nil {
		t.Fatal("audited transition", err)
	}
	data, _ := Encode(map[string]any{"before_bytes": len(old), "after_bytes": len(b), "archive_count": len(m.ValidationInputs), "seconds": time.Since(start).Seconds(), "plan_identical": true, "semantic_inputs_identical": true, "identity_changed": before.Identity != m.Identity})
	if err = os.WriteFile(filepath.Join(out, "projection-size-proof.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("payload projection %d -> %d bytes; archives=%d", len(old), len(b), len(m.ValidationInputs))
}
