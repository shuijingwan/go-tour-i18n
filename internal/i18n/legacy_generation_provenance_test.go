package i18n

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Minimal publisher/archive unit fixture. Full graph construction and fresh
// public reader are separately exercised by the sitecontent synthetic tests.
func legacyPublisherFixture(t *testing.T) (string, *LegacyGenerationProvenancePlan, []byte) {
	t.Helper()
	root := t.TempDir()
	base := "data/retranslation-runs/zh-CN/chatgpt-zh-CN-001/"
	unit := strings.Repeat("a", 64)
	input := []byte("source\n")
	manifest := []byte("formal historical manifest\n")
	raw := []byte("historical response\n")
	entries := []TransportBundleEntry{{Path: "formal/manifest.json", Data: manifest}, {Path: "formal/inputs/" + unit + ".article", Data: input}}
	files := []TransportBundleFile{NewTransportBundleFile(entries[0].Path, base+"manifest.json", manifest), NewTransportBundleFile(entries[1].Path, base+"inputs/"+unit+".article", input)}
	gm := GenerationBundleManifest{SchemaVersion: 1, Kind: "go-tour-i18n/generation-bundle", Locale: "zh-CN", BatchID: "chatgpt-zh-CN-001", TaskKind: "translation-unit-initial", UnitKind: UnitKindPage, Attempt: 1, InputIdentitySHA256: sum(input), Files: files, ExpectedOutputs: []GenerationBundleExpectedOutput{{BundlePath: "outputs/" + unit + ".article", InstallPath: "raw-responses/" + unit + ".article", UnitID: unit}}}
	mb := mustJSON(gm)
	archive, err := WriteDeterministicTransportBundle(mb, entries)
	if err != nil {
		t.Fatal(err)
	}
	out := NewTransportBundleFile("outputs/"+unit+".article", "", raw)
	imported := GenerationResultBundleManifest{SchemaVersion: 1, Kind: "go-tour-i18n/generation-result-bundle", TaskKind: gm.TaskKind, Locale: gm.Locale, BatchID: gm.BatchID, Attempt: 1, GenerationBundleSHA256: sum(archive), InputIdentitySHA256: gm.InputIdentitySHA256, Provider: "chatgpt", Model: FormalChatGPTGenerationModel, GenerationManifest: NewTransportBundleFile("generation-manifest.json", "", mb), Outputs: []TransportBundleFile{out}}
	ib := mustJSON(imported)
	ip := base + "generation-imports/initial-manifest.json"
	write := func(p string, b []byte) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, p)), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, p), b, 0644); err != nil {
			t.Fatal(err)
		}
	}
	write(ip, ib)
	write(base+"manifest.json", manifest)
	write(base+"inputs/"+unit+".article", input)
	o := LegacyGenerationProvenanceOptions{Locale: gm.Locale, ContentPackage: "synthetic-package", BatchID: gm.BatchID, UnitID: unit, CurrentUnitID: strings.Repeat("b", 64), Attempt: 1, TaskKind: gm.TaskKind, ImportReceipt: GlossaryArchiveReference{ip, sum(ib)}, OriginalBundleSHA256: sum(archive), PreviousSnapshotID: "synthetic-snapshot", SnapshotSHA256: sum([]byte("snapshot"))}
	r := &LegacyGenerationProvenance{Schema: LegacyGenerationProvenanceSchema, Capability: LegacyGenerationProvenanceCapability, OriginalArchiveUnavailable: true, Options: o, InputIdentitySHA256: imported.InputIdentitySHA256, GenerationManifest: imported.GenerationManifest, Output: out, InstallPath: gm.ExpectedOutputs[0].InstallPath, HistoricalManifest: GlossaryArchiveReference{base + "manifest.json", sum(manifest)}, HistoricalInput: GlossaryArchiveReference{base + "inputs/" + unit + ".article", sum(input)}, Dependencies: []LegacyProvenanceDependency{{base + "manifest.json", sum(manifest), int64(len(manifest))}, {base + "inputs/" + unit + ".article", sum(input), int64(len(input))}}}
	r.Identity = legacyProvenanceIdentity(*r)
	r.AuthorityID = r.Identity
	return root, &LegacyGenerationProvenancePlan{PlannedReceipt: r, PlannedPath: legacyProvenancePath(o.Locale, r.Identity), Identity: r.Identity, Mutation: "none"}, archive
}

func TestLegacyGenerationProvenanceArchiveRaceAndDrift(t *testing.T) {
	for _, mode := range []string{"archive-appears", "wrong-archive-SHA", "dependency-drift", "conflicting-record", "publication-unknown"} {
		t.Run(mode, func(t *testing.T) {
			root, p, archive := legacyPublisherFixture(t)
			r := p.PlannedReceipt
			before := func() {
				switch mode {
				case "archive-appears", "wrong-archive-SHA":
					data := archive
					if mode == "wrong-archive-SHA" {
						data = []byte("corrupt archive")
					}
					path, err := compatibilityPath(root, importedGenerationArchive(r.Options.Locale, r.Options.OriginalBundleSHA256), true)
					if err != nil {
						t.Fatal(err)
					}
					if err = os.WriteFile(path, data, 0644); err != nil {
						t.Fatal(err)
					}
				case "dependency-drift":
					if err := os.WriteFile(filepath.Join(root, r.HistoricalInput.Path), []byte("changed"), 0644); err != nil {
						t.Fatal(err)
					}
				case "conflicting-record":
					other := *r
					other.Capability = "whole-batch"
					other.Identity = legacyProvenanceIdentity(other)
					other.AuthorityID = other.Identity
					path, err := compatibilityPath(root, legacyProvenancePath(other.Options.Locale, other.Identity), true)
					if err != nil {
						t.Fatal(err)
					}
					if err = os.WriteFile(path, append(mustJSON(other), '\n'), 0644); err != nil {
						t.Fatal(err)
					}
				case "publication-unknown":
					// The final link loses its source after preflight: never guess no commit.
					if err := os.Remove(filepath.Join(root, legacyProvenanceGuard(r.Options), "receipt.json")); err != nil {
						t.Fatal(err)
					}
				}
			}
			got, err := publishLegacyGenerationProvenance(root, p, func() (*LegacyGenerationProvenance, error) { return r, nil }, before)
			if err == nil {
				t.Fatal("race/drift allowed publication")
			}
			if mode == "publication-unknown" {
				if got.Mutation != "unknown" {
					t.Fatal(got, err)
				}
				if err = legacyNoPending(root, r.Options); err == nil || !strings.Contains(err.Error(), "mutation=unknown") {
					t.Fatal("pending state not guarded", err)
				}
			} else if got.Mutation != "none" {
				t.Fatal(got, err)
			}
			if _, err = os.Lstat(filepath.Join(root, p.PlannedPath)); !os.IsNotExist(err) {
				t.Fatal("rejected record published receipt", err)
			}
		})
	}
}

func TestLegacyGenerationProvenanceArchiveConflict(t *testing.T) {
	root, p, archive := legacyPublisherFixture(t)
	r := p.PlannedReceipt
	path, err := compatibilityPath(root, importedGenerationArchive(r.Options.Locale, r.Options.OriginalBundleSHA256), true)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, archive, 0644); err != nil {
		t.Fatal(err)
	}
	if ref, err := compareLegacyArchive(root, r); err != nil || ref == nil {
		t.Fatal("exact archive not verified", ref, err)
	}
	for _, mode := range []string{"install-path", "output-identity", "formal-input", "formal-manifest", "covered-source"} {
		t.Run(mode, func(t *testing.T) {
			copy := *r
			copy.Dependencies = append([]LegacyProvenanceDependency(nil), r.Dependencies...)
			switch mode {
			case "install-path":
				copy.InstallPath = "raw-responses/other.article"
			case "output-identity":
				copy.Output.BundlePath = "outputs/other.article"
			case "formal-input":
				old, _ := os.ReadFile(filepath.Join(root, r.HistoricalInput.Path))
				defer os.WriteFile(filepath.Join(root, r.HistoricalInput.Path), old, 0644)
				os.WriteFile(filepath.Join(root, r.HistoricalInput.Path), []byte("conflicting"), 0644)
			case "formal-manifest":
				old, _ := os.ReadFile(filepath.Join(root, r.HistoricalManifest.Path))
				defer os.WriteFile(filepath.Join(root, r.HistoricalManifest.Path), old, 0644)
				os.WriteFile(filepath.Join(root, r.HistoricalManifest.Path), []byte("conflicting"), 0644)
			case "covered-source":
				copy.Dependencies[0].SHA256 = strings.Repeat("f", 64)
			}
			if _, err := compareLegacyArchive(root, &copy); err == nil {
				t.Fatal("archive conflict accepted")
			}
		})
	}
}

func TestLegacyGenerationProvenanceCommittedCleanupFailure(t *testing.T) {
	root, p, _ := legacyPublisherFixture(t)
	guard := filepath.Join(root, legacyProvenanceGuard(p.PlannedReceipt.Options))
	defer os.Chmod(guard, 0700)
	got, err := publishLegacyGenerationProvenance(root, p, func() (*LegacyGenerationProvenance, error) { return p.PlannedReceipt, nil }, func() {
		if e := os.Chmod(guard, 0500); e != nil {
			t.Fatal(e)
		}
	})
	if err == nil || !got.Recorded || got.Mutation != "known" || got.ReceiptPath == "" || !strings.Contains(err.Error(), "committed") {
		t.Fatal("committed receipt mislabeled", got, err)
	}
	b, e := os.ReadFile(filepath.Join(root, got.ReceiptPath))
	if e != nil || sum(b) != got.ReceiptSHA256 {
		t.Fatal("known publication not exact", e)
	}
	if e = legacyNoPending(root, p.PlannedReceipt.Options); e == nil {
		t.Fatal("cleanup guard did not block replay")
	}
}
