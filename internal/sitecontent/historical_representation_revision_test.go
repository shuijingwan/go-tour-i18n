package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestHistoricalRepresentationRevisionSurvivesCurrentGlossary(t *testing.T) {
	root, c, o := machineRevisionFixture(t)
	o.Record = true
	reopen, reopenPath, err := i18n.PrepareRepresentationRevision(root, c, o)
	if err != nil {
		t.Fatal(err)
	}
	revision, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: []string{o.UnitID}, UnitKind: i18n.UnitKindPage, BatchID: "chatgpt-zh-CN-002", AllowReexport: true, PreviousSnapshotID: o.Reconciliation.PreviousSnapshotID, SurfaceReopenID: o.ReopenID})
	if err != nil {
		t.Fatal(err)
	}
	b, manifest, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: revision.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	files, err := i18n.ReadTransportBundle(b, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	// Explicit synthetic output/QA only in TempDir; no real language invocation.
	output := files["formal/inputs/"+filepath.Base(manifest.ExpectedOutputs[0].BundlePath)]
	surfaceImport(t, root, c, b, map[string][]byte{o.UnitID: output})
	if p, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: revision.BatchID}); err != nil || p.ValidationPassed != 1 {
		t.Fatal(p, err)
	}
	s, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "historical-revision-a"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: s.SnapshotID, UnitIDs: c.TranslationScope, Rating: "A", ReviewerSession: "synthetic-independent-reviewer"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.FinalizeQualityCheck(root, c, i18n.QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: s.SnapshotID}); err != nil {
		t.Fatal(err)
	}
	before, err := i18n.ReadHistoricalTranslationIntegrity(root, c, "zh-CN", s.SnapshotID)
	if err != nil {
		t.Fatal(err)
	}
	if reopen.Units[0].Snapshot.UnitID == o.UnitID || before.Finalization.Units[2].UnitID != o.UnitID {
		t.Fatal("fixture must contain an independently reviewed remapped Unit")
	}
	glossary, _ := readRegular(root, "locales/zh-CN/glossary.yaml")
	writeFixture(t, root, "locales/zh-CN/glossary.yaml", append(bytes.Clone(glossary), []byte("\n# current world changed\n")...))
	after, err := i18n.ReadHistoricalTranslationIntegrity(root, c, "zh-CN", s.SnapshotID)
	left, _ := json.Marshal(before)
	right, _ := json.Marshal(after)
	if err != nil || !bytes.Equal(left, right) {
		t.Fatal("historical revision changed result or used current world", err)
	}
	if _, err := i18n.CheckQualityCheckSurfaceReopen(root, c, "zh-CN", o.ReopenID); err == nil {
		t.Fatal("ordinary current revision checker inherited historical mode")
	}
	for name, path := range map[string]string{"reopen": reopenPath, "independent-QC": "data/quality-check-snapshots/zh-CN/" + o.Reconciliation.PreviousSnapshotID + "/quality-check-results.json", "glossary-archive": "data/glossary-history/zh-CN/" + s.GlossarySHA256 + ".yaml", "machine-raw": reopen.ReplacementContext.Machine.OriginalRaw.Path, "historical-documents": o.Reconciliation.HistoricalDocuments.Path} {
		t.Run(name, func(t *testing.T) {
			data, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			abs := filepath.Join(root, path)
			info, _ := os.Stat(abs)
			if err := os.Chmod(abs, 0644); err != nil {
				t.Fatal(err)
			}
			defer func() { writeFixture(t, root, path, data); os.Chmod(abs, info.Mode()) }()
			changed := append(bytes.Clone(data), ' ')
			if name == "reopen" {
				var tampered i18n.QualityCheckSurfaceReopen
				if err := json.Unmarshal(data, &tampered); err != nil {
					t.Fatal(err)
				}
				tampered.ReplacementContext.Machine.ScaffoldSHA256 = strings.Repeat("0", 64)
				changed, _ = json.Marshal(tampered)
			}
			writeFixture(t, root, path, changed)
			if _, err := i18n.ReadHistoricalTranslationIntegrity(root, c, "zh-CN", s.SnapshotID); err == nil {
				t.Fatal("historical reconstruction accepted changed bound evidence")
			}
		})
	}
	t.Run("current-source-authority", func(t *testing.T) {
		u, _ := c.Unit(o.UnitID)
		data := bytes.Clone(u.Source)
		u.Source[0] ^= 1
		defer copy(u.Source, data)
		if _, err := i18n.ReadHistoricalTranslationIntegrity(root, c, "zh-CN", s.SnapshotID); err == nil {
			t.Fatal("source authority drift accepted")
		}
	})
}
