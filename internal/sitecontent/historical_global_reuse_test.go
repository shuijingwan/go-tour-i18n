package sitecontent

import (
	"bytes"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func historicalGlobalSuccessorFixture(t *testing.T, extraFrozenPaths ...string) (string, *Global, *i18n.GlossarySourceCorpus, Reference, Reference) {
	t.Helper()
	t.Cleanup(contentidentity.BeginMemoization())
	root, _, _, _ := shellAdoptionFixture(t, extraFrozenPaths...)
	name := "data/site-shell/translation.html"
	b, err := readRegular(root, name)
	if err != nil {
		t.Fatal(err)
	}
	at := bytes.LastIndex(b, []byte("</main>"))
	if at < 0 {
		t.Fatal("fixture main missing")
	}
	body := append(append(bytes.Clone(b[:at]), []byte("<p>Synthetic consumed source addition.</p>\n")...), b[at:]...)
	writeFixture(t, root, "data/historical-count-body.html", body)
	proposal := StructuredSourceProposal{Schema: StructuredSourceSuccessorSchema, Package: "site-v2-shell", Previous: Reference{name, digest(b)}, Body: Reference{"data/historical-count-body.html", digest(body)}, InsertAt: at}
	raw, _ := Encode(proposal)
	writeFixture(t, root, "data/historical-count-proposal.json", raw)
	ref, _ := ReferenceFile(root, "data/historical-count-proposal.json")
	a, err := BuildStructuredSourceSuccessor(root, ref)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AdoptStructuredSourceSuccessor(root, ref, a.BaseIdentity); err != nil {
		t.Fatal(err)
	}
	parent, _ := ReferenceFile(root, StructuredSourceSuccessorPath(a))
	oldBytes, _ := readRegular(root, i18n.GlossaryCorpusPath)
	corpus, err := AdoptCorpusProjection(root, digest(oldBytes), true)
	if err != nil {
		t.Fatal(err)
	}
	consumed, _ := ReferenceFile(root, CorpusSuccessorPath(corpus))
	replacement, input := replacementProposalFixture(t, root, parent, consumed)
	plan, err := ApplyConsumedStructuredSourceReplacement(root, replacement, input.PredecessorIdentity, input.PriorPointer.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	g, err := loadSourceCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	old, err := readProjectionCorpus(root, i18n.GlossaryCorpusPath)
	if err != nil {
		t.Fatal(err)
	}
	return root, g, old, parent, plan.Receipt
}

func TestHistoricalGlobalRepetitionBaseline(t *testing.T) {
	root, g, old, _, _ := historicalGlobalSuccessorFixture(t)
	v := successorValidation(root, nil)
	v.disableHistoricalReuse = true
	v.historicalMetrics, v.historicalIdentities = map[string]int{}, map[string]int{}
	if _, err := projectAdoptedCorpusAtSources(root, g, old, false, v); err != nil {
		t.Fatal(err)
	}
	// Builder entries still repeat in this deliberately unowned control;
	// exact-byte L2 reuse may eliminate their snapshot calculations entirely.
	if v.historicalMetrics["BuildGlobal"] <= 1 || len(v.historicalIdentities) != 1 || v.historicalMetrics["snapshotFS"] > 1 || v.historicalMetrics["snapshotFS"] != v.historicalMetrics["global-environment-build"] {
		t.Fatalf("not repeated exact historical construction: %v identities=%v", v.historicalMetrics, v.historicalIdentities)
	}
	t.Logf("HISTORICAL_BASELINE metrics=%v exact_identities=%v", v.historicalMetrics, v.historicalIdentities)
}

func TestHistoricalGlobalInvocationReuse(t *testing.T) {
	root, g, old, parent, receipt := historicalGlobalSuccessorFixture(t)
	baseline := successorValidation(root, nil)
	baseline.disableHistoricalReuse = true
	baseline.historicalMetrics, baseline.historicalIdentities = map[string]int{}, map[string]int{}
	want, err := projectAdoptedCorpusAtSources(root, g, old, false, baseline)
	if err != nil {
		t.Fatal(err)
	}
	v := successorValidation(root, nil)
	v.historicalMetrics, v.historicalIdentities, v.builds = map[string]int{}, map[string]int{}, map[Reference]int{}
	got, err := projectAdoptedCorpusAtSources(root, g, old, false, v)
	if err != nil {
		t.Fatal(err)
	}
	x, _ := Encode(want)
	y, _ := Encode(got)
	if !bytes.Equal(x, y) || want.Identity != got.Identity {
		t.Fatal("corpus bytes/identity changed")
	}
	if baseline.historicalMetrics["BuildGlobal"] <= 1 || v.historicalMetrics["BuildGlobal"] != 1 || v.historicalMetrics["snapshotFS"] > 1 || v.historicalMetrics["snapshotFS"] != v.historicalMetrics["global-environment-build"] || len(baseline.historicalIdentities) != 1 || len(v.historicalIdentities) != 1 || v.builds[parent] != 1 || v.builds[receipt] != 1 {
		t.Fatalf("incorrect reuse: before=%v after=%v successors=%v", baseline.historicalMetrics, v.historicalMetrics, v.builds)
	}
	t.Logf("HISTORICAL_REUSE before_global=%d after_global=%d before_snapshot=%d after_snapshot=%d before_loads=%d after_loads=%d", baseline.historicalMetrics["BuildGlobal"], v.historicalMetrics["BuildGlobal"], baseline.historicalMetrics["snapshotFS"], v.historicalMetrics["snapshotFS"], baseline.historicalMetrics["loadHistoricalGlobal"], v.historicalMetrics["loadHistoricalGlobal"])
	for _, ref := range []Reference{parent, receipt} {
		a, err := baseline.check(ref, 0)
		if err != nil {
			t.Fatal(err)
		}
		b, err := v.check(ref, 0)
		if err != nil {
			t.Fatal(err)
		}
		x, _ := Encode(a)
		y, _ := Encode(b)
		if !bytes.Equal(x, y) || a.Identity != b.Identity {
			t.Fatal("successor changed")
		}
	}
	a, err := CheckShellAdoption(root, *g.ShellAdoption, baseline)
	if err != nil {
		t.Fatal(err)
	}
	b, err := CheckShellAdoption(root, *g.ShellAdoption, v)
	if err != nil {
		t.Fatal(err)
	}
	x, _ = Encode(a)
	y, _ = Encode(b)
	if !bytes.Equal(x, y) {
		t.Fatal("shell adoption changed")
	}
	historical, err := v.historicalGlobal(root)
	if err != nil {
		t.Fatal(err)
	}
	current, err := v.adoptionBase()
	if err != nil || current.Identity == historical.Identity {
		t.Fatal("historical/current confused", err)
	}
	if _, err := v.historicalGlobal(root, Reference{GlobalPath, digest([]byte("different authority"))}); err == nil {
		t.Fatal("wrong authority reused")
	}
	for _, label := range []string{"snapshot", "current-mode"} {
		t.Run(label, func(t *testing.T) {
			key := v.historical.key
			defer func() { v.historical.key = key }()
			if label == "snapshot" {
				v.historical.key.Snapshot.SHA256 = digest([]byte("different snapshot"))
			} else {
				v.historical.key.Mode = "current-source-projection"
			}
			if _, err := v.historicalGlobal(root); err == nil {
				t.Fatal("different identity/mode hit")
			}
		})
	}
	m, err := loadSourceManifest(root)
	if err != nil || len(m.Legacy.Paths) == 0 {
		t.Fatal("frozen fixture manifest", err)
	}
	for _, name := range []string{adoptedSourcePath(m.Legacy.Paths[0]), SourceManifestPath, GlobalPath} {
		t.Run("dependency:"+name, func(t *testing.T) {
			raw, err := readRegular(root, name)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, name, raw)
			writeFixture(t, root, name, append(bytes.Clone(raw), ' '))
			if _, err := v.historicalGlobal(root); err == nil {
				t.Fatal("changed frozen dependency reused")
			}
		})
	}
	if v.historicalMetrics["BuildGlobal"] != 1 {
		t.Fatal("changed world rebuilt/accepted")
	}
}
