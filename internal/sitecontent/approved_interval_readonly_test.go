package sitecontent

import (
	"bytes"
	"encoding/json"
	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApprovedIntervalJSONReadonly(t *testing.T) {
	if os.Getenv("GO_TOUR_APPROVED_INTERVAL_READONLY") != "1" {
		t.Skip("explicit readonly engineering probe")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	c, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err = AttachRegisteredTranslationPackagesReadonly(root, c); err != nil {
		t.Fatal(err)
	}
	if err = c.UseTranslationPackage("learn-docs-v1"); err != nil {
		t.Fatal(err)
	}
	b, err := readRegular(root, "data/retranslation-runs/zh-CN/chatgpt-zh-CN-064/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var m i18n.RetranslationBatchManifest
	if err = json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	const id = "b49d8e873a0f66f6803077eb534583a155d2bd209a032d7c00fba8fea856da5c"
	request, _ := json.Marshal(i18n.ApprovedIntervalSelection{Schema: i18n.ApprovedIntervalSelectionSchema, Locale: m.Locale, BatchID: m.BatchID, ContentPackage: "learn-docs-v1", Units: []i18n.ApprovedIntervalSelectionUnit{{CurrentUnitID: id, Mode: i18n.ApprovedIntervalMode, LegacyProvenance: &i18n.GlossaryArchiveReference{Path: "data/generation-provenance-authorities/zh-CN/6b6304a944e3abdda45deed5dfc0c00576529a5a6e37533f92ce4d3cbf5e6053.json", SHA256: "17292b864444bb98746dd7b65d0eb3babaa7f1785c1b77fe31d325ced2916709"}}}})
	rows, proofs, err := i18n.InspectApprovedIntervalReadiness(root, c, &m, request)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, r := range rows {
		counts[r.RecoveryClass]++
		if r.UnitID == id {
			t.Logf("json class=%s blocker=%s", r.RecoveryClass, r.Blocker)
		}
	}
	t.Logf("readiness=%v", counts)
	if counts["APPROVED_INTERVAL_PASS"] != 1 {
		t.Fatal("approved interval not ready")
	}
	if counts["UNAFFECTED_CURRENT_RESTORE_PASS"] != 28 || counts["DETERMINISTIC_REBASE_PASS"] != 3 || len(rows) != 32 || len(proofs) != 1 {
		t.Fatal("exact 064 readiness changed", counts)
	}
	p := proofs[0]
	if p.Record.UnitID != id || p.Projection.CandidateSHA256 != "9d978bca55c326a2c24d43684da62383523fe01901c4e22563c3512581dc904d" || len(p.Projection.Findings) == 0 {
		t.Fatal("real proof identity mismatch")
	}
	var codec approvedPageCodec
	if err = StrictJSON(p.Projection.Codec, &codec); err != nil {
		t.Fatal(err)
	}
	reviewed, err := readRegular(root, p.Snapshot.CandidatePath)
	if err != nil {
		t.Fatal(err)
	}
	historical, err := codec.Old.Restore(codec.Baseline)
	if err != nil || !bytes.Equal(bridgeEOF(historical), reviewed) {
		t.Fatal("real historical restore not exact", err)
	}
	restored, err := codec.Restore(p.Projection.Text)
	if err != nil || !bytes.Equal(bridgeEOF(restored), reviewed) {
		t.Fatal("real projection restore not exact", err)
	}
	for _, f := range p.Projection.Findings {
		if f.Candidate.Space != p.Projection.ParentCandidate.Space || f.Candidate.Unit != "byte" || f.Candidate.Start < p.Projection.ParentCandidate.End && p.Projection.ParentCandidate.Start < f.Candidate.End {
			t.Fatal("typed nonoverlap failed")
		}
	}
	if out := os.Getenv("GO_TOUR_APPROVED_INTERVAL_EVIDENCE"); out != "" {
		if !filepath.IsAbs(out) || !strings.HasPrefix(filepath.Clean(out), "/tmp/") {
			t.Fatal("engineering evidence must be in /tmp")
		}
		evidence, _ := json.MarshalIndent(struct {
			Counts         map[string]int
			Proof          i18n.ApprovedIntervalTransport
			FormalMutation string
		}{counts, p, "none"}, "", "  ")
		if err = os.WriteFile(out, append(evidence, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
}
