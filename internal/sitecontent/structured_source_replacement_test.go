package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func consumedReplacementFixture(t *testing.T) (string, Reference, Reference, Reference, Reference) {
	t.Helper()
	t.Cleanup(contentidentity.BeginMemoization()) // same pure-computation scope as the canonical CLI
	root, baseline, generation, _ := structuredLifecycleFixture(t, "site-v2-shell")
	name := "data/site-shell/home.html" // shared mechanism, not the real integration route
	before, err := readRegular(root, name)
	if err != nil {
		t.Fatal(err)
	}
	at := bytes.LastIndex(before, []byte("</main>"))
	if at < 0 {
		t.Fatal("main required")
	}
	body := append(bytes.Clone(before[:at]), []byte("<p>Synthetic consumed addition.</p>\n")...)
	body = append(body, before[at:]...)
	writeFixture(t, root, "data/append-body.html", body)
	p := StructuredSourceProposal{Schema: StructuredSourceSuccessorSchema, Package: "site-v2-shell", Previous: Reference{name, digest(before)}, Body: Reference{"data/append-body.html", digest(body)}, InsertAt: at}
	b, _ := Encode(p)
	writeFixture(t, root, "data/append-proposal.json", b)
	ref, _ := ReferenceFile(root, "data/append-proposal.json")
	a, err := BuildStructuredSourceSuccessor(root, ref)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AdoptStructuredSourceSuccessor(root, ref, a.BaseIdentity); err != nil {
		t.Fatal(err)
	}
	parent, _ := ReferenceFile(root, StructuredSourceSuccessorPath(a))
	oldCorpus, _ := readRegular(root, i18n.GlossaryCorpusPath)
	corpus, err := AdoptCorpusProjection(root, digest(oldCorpus), true)
	if err != nil {
		t.Fatal(err)
	}
	consumption, _ := ReferenceFile(root, CorpusSuccessorPath(corpus))
	return root, parent, consumption, baseline, generation
}

func replacementProposalFixture(t *testing.T, root string, parent, consumption Reference) (Reference, StructuredSourceReplacementProposal) {
	t.Helper()
	a, err := CheckStructuredSourceSuccessor(root, parent)
	if err != nil {
		t.Fatal(err)
	}
	old, err := readRegularReference(root, a.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Current.Units) < 3 {
		t.Fatal("fixture requires three slots")
	}
	// Source-owned synthetic replacement: replace first slot, remove another,
	// retain other source slots, add a distinct slot. No localized target input.
	remove := a.Current.Units[2]
	first := a.Current.Units[0]
	body := append(bytes.Clone(old[:remove.Start]), old[remove.End:]...)
	body = append(append(bytes.Clone(body[:first.Start]), []byte("Synthetic replacement heading.")...), body[first.End:]...)
	at := bytes.LastIndex(body, []byte("</main>"))
	body = append(append(bytes.Clone(body[:at]), []byte("<p>Synthetic newly added replacement slot.</p>\n")...), body[at:]...)
	writeFixture(t, root, "data/replacement-body.html", body)
	s := documentSource(a.Current)
	s.SHA256 = digest(body)
	d, err := ParseDocument(s, body)
	if err != nil {
		t.Fatal(err)
	}
	pointer, _ := ReferenceFile(root, structuredSourceCurrent(a.Previous.Path))
	p := StructuredSourceReplacementProposal{Schema: StructuredSourceReplacementSchema, Package: a.Package, Predecessor: parent, PredecessorIdentity: a.Identity, PriorPointer: pointer, Consumption: consumption, PredecessorHeads: []Reference{parent}, Body: Reference{"data/replacement-body.html", digest(body)}, Replaced: []SourceSlotReplacement{{Old: first.ID, New: d.Units[0].ID}}}
	b, _ := Encode(p)
	writeFixture(t, root, "data/replacement-proposal.json", b)
	ref, _ := ReferenceFile(root, "data/replacement-proposal.json")
	return ref, p
}

func TestConsumedStructuredReplacementAuthority(t *testing.T) {
	root, parent, consumption, baseline, generation := consumedReplacementFixture(t)
	proposal, p := replacementProposalFixture(t, root, parent, consumption)
	old, err := CheckStructuredSourceSuccessor(root, parent)
	if err != nil {
		t.Fatal(err)
	}
	protected := map[string][]byte{}
	var completion Locale
	localeBytes, err := readRegular(root, LocalePath("zh-CN"))
	if err != nil {
		t.Fatal(err)
	}
	if err := StrictJSON(localeBytes, &completion); err != nil {
		t.Fatal(err)
	}
	protected[LocalePath("zh-CN")] = localeBytes
	for _, c := range completion.Packages {
		for _, r := range c.Evidence {
			raw, err := readRegularReference(root, r)
			if err != nil {
				t.Fatal(err)
			}
			protected[r.Path] = raw
			if c.Package == "site-v2-shell" && strings.Contains(r.Path, "surface/") {
				var s IntegratedSurfaceReceipt
				if err := StrictJSON(raw, &s); err != nil {
					t.Fatal(err)
				}
				raw, err = readRegularReference(root, s.Evidence)
				if err != nil {
					t.Fatal(err)
				}
				protected[s.Evidence.Path] = raw
			}
		}
	}
	for _, name := range []string{parent.Path, consumption.Path, baseline.Path, generation.Path, old.Body.Path, old.Previous.Path, i18n.GlossaryCorpusPath} {
		b, err := readRegular(root, name)
		if err != nil {
			t.Fatal(err)
		}
		protected[name] = b
	}
	plan, err := InspectConsumedStructuredSourceReplacement(root, proposal)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Mutation != "none" || plan.Applied || plan.AlreadyCurrent || plan.Authority.Replacement == nil {
		t.Fatal("readonly plan", plan)
	}
	a := plan.Authority
	if len(a.CarryIDs) == 0 || len(a.AddedIDs) == 0 || len(a.Replacement.Removed) == 0 || len(a.Replacement.Replaced) != 1 {
		t.Fatal("four-way source partition", a)
	}
	if _, err := os.Stat(pathJoin(root, plan.Receipt.Path)); !os.IsNotExist(err) {
		t.Fatal("preflight wrote receipt")
	}
	if _, err := AdoptStructuredSourceSuccessor(root, proposal, a.BaseIdentity); err == nil {
		t.Fatal("replacement masqueraded as append")
	}
	if _, err := CorrectUnconsumedStructuredSource(root, proposal, a.BaseIdentity, p.PriorPointer.SHA256); err == nil {
		t.Fatal("replacement masqueraded as unconsumed correction")
	}
	if _, err := ApplyConsumedStructuredSourceReplacement(root, proposal, p.PredecessorIdentity, digest([]byte("wrong"))); err == nil {
		t.Fatal("wrong CAS accepted")
	}
	got, err := ApplyConsumedStructuredSourceReplacement(root, proposal, p.PredecessorIdentity, p.PriorPointer.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Applied || got.Mutation != "known" {
		t.Fatal(got)
	}
	for name, b := range protected {
		now, err := readRegular(root, name)
		if err != nil || !bytes.Equal(b, now) {
			t.Fatal("historical source/corpus/language bytes changed", name, err)
		}
	}
	if _, err := CheckStructuredSourceSuccessor(root, parent); err != nil {
		t.Fatal("historical source unreadable", err)
	}
	if _, err := ReadHistoricalAdoptedCorpusSuccessor(root, consumption); err != nil {
		t.Fatal("historical corpus unreadable", err)
	}
	v := newClosureValidation(root)
	if _, _, _, err := v.historicalStructuredGeneration(root, generation); err != nil {
		t.Fatal("historical Generation", err)
	}
	b, err := v.historicalStructuredClosure(root, baseline, map[string]bool{})
	if err != nil {
		t.Fatal("historical closure", err)
	}
	if err := v.verify(); err != nil {
		t.Fatal(err)
	}
	delta, err := sourceDeltaFromBaseline(root, "zh-CN", *a, b)
	if err != nil {
		t.Fatal("accepted ancestry proof", err)
	}
	if len(delta.Pending) == 0 {
		t.Fatal("source replacement invented language PASS")
	}
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	docs, raw, err := CurrentPackageDocuments(root, g, p.Package)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range docs {
		if d.Path == a.Previous.Path {
			found = reflect.DeepEqual(d, a.Current) && digest(raw[d.Path]) == p.Body.SHA256
		}
	}
	if !found {
		t.Fatal("current semantic projection not switched")
	}
	if err := CheckUnifiedCorpus(root); err == nil {
		t.Fatal("old corpus treated as current after replacement")
	}
	next, err := BuildAdoptedCorpusSuccessor(root)
	if err != nil || !next.Changed {
		t.Fatal("shared corpus successor unavailable", err)
	}
	for _, c := range next.Result.Contributors {
		if c.Path == a.Previous.Path {
			for _, context := range c.Contexts {
				if !strings.HasPrefix(context.ID, a.Identity+"/") {
					t.Fatal("replacement semantic projection lacks authority identity")
				}
			}
		}
	}
	// No corpus apply, language output or closure is created by source apply.
	if _, err := PlanStructuredAssets(root, "zh-CN"); err == nil {
		t.Fatal("source authority bypassed current language eligibility")
	}
	// Future shared corpus publication is exercised only in this TempDir.
	// It must become CURRENT without relabeling the old full Review.
	corpusBytes, err := readRegular(root, i18n.GlossaryCorpusPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AdoptCorpusProjection(root, digest(corpusBytes), true); err != nil {
		t.Fatal("shared replacement corpus publication", err)
	}
	if err := CheckUnifiedCorpus(root); err != nil {
		t.Fatal("replacement corpus current-check", err)
	}
	if _, err := i18n.RequireUnifiedGlossaryReview(root, "zh-CN"); err == nil {
		t.Fatal("old Review promoted to new corpus")
	}
	if _, err := ReadHistoricalAdoptedCorpusSuccessor(root, consumption); err != nil {
		t.Fatal("old corpus after successor publication", err)
	}
	replay, err := ApplyConsumedStructuredSourceReplacement(root, proposal, p.PredecessorIdentity, p.PriorPointer.SHA256)
	if err != nil || !replay.AlreadyCurrent || replay.Mutation != "none" {
		t.Fatal("exact replay", err, replay)
	}
}

func TestConsumedStructuredReplacementRejectsDrift(t *testing.T) {
	root, parent, consumption, _, _ := consumedReplacementFixture(t)
	ref, p := replacementProposalFixture(t, root, parent, consumption)
	a, err := CheckStructuredSourceSuccessor(root, parent)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{parent.Path, a.Body.Path, a.Previous.Path, consumption.Path} {
		before, err := readRegular(root, name)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, name, append(bytes.Clone(before), ' '))
		if _, err := InspectConsumedStructuredSourceReplacement(root, ref); err == nil {
			t.Fatal("historical dependency tamper accepted", name)
		}
		writeFixture(t, root, name, before)
	}
	for label, change := range map[string]func(*StructuredSourceReplacementProposal){
		"predecessor-identity": func(p *StructuredSourceReplacementProposal) { p.PredecessorIdentity = digest([]byte("wrong")) },
		"body-SHA":             func(p *StructuredSourceReplacementProposal) { p.Body.SHA256 = digest([]byte("wrong")) },
		"source-authority-SHA": func(p *StructuredSourceReplacementProposal) { p.Predecessor.SHA256 = digest([]byte("wrong")) },
		"pointer-SHA":          func(p *StructuredSourceReplacementProposal) { p.PriorPointer.SHA256 = digest([]byte("wrong")) },
		"wrong-package":        func(p *StructuredSourceReplacementProposal) { p.Package = "learn-docs-v1" },
		"wrong-slot": func(p *StructuredSourceReplacementProposal) {
			p.Replaced = []SourceSlotReplacement{{Old: "missing", New: "missing"}}
		},
		"unconsumed":    func(p *StructuredSourceReplacementProposal) { p.Consumption = Reference{parent.Path, parent.SHA256} },
		"append-schema": func(p *StructuredSourceReplacementProposal) { p.Schema = StructuredSourceSuccessorSchema },
	} {
		t.Run(label, func(t *testing.T) {
			bad := p
			change(&bad)
			b, _ := Encode(bad)
			name := "data/bad-" + label + ".json"
			writeFixture(t, root, name, b)
			r, _ := ReferenceFile(root, name)
			if _, err := InspectConsumedStructuredSourceReplacement(root, r); err == nil {
				t.Fatal("mismatch accepted")
			}
		})
	}
	before, _ := readRegular(root, p.PriorPointer.Path)
	body, _ := readRegular(root, p.Body.Path)
	_, err = applyConsumedStructuredSourceReplacement(root, ref, p.PredecessorIdentity, p.PriorPointer.SHA256, func() { writeFixture(t, root, p.Body.Path, append(bytes.Clone(body), ' ')) })
	if err == nil {
		t.Fatal("concurrent mutation accepted")
	}
	after, _ := readRegular(root, p.PriorPointer.Path)
	if !bytes.Equal(before, after) {
		t.Fatal("failed concurrent apply changed pointer")
	}
	writeFixture(t, root, p.Body.Path, body)
	plan, err := InspectConsumedStructuredSourceReplacement(root, ref)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(pathJoin(root, plan.Receipt.Path)); err != nil {
		t.Fatal("orphan receipt proof absent", err)
	}
	if _, err := ApplyConsumedStructuredSourceReplacement(root, ref, p.PredecessorIdentity, p.PriorPointer.SHA256); err != nil {
		t.Fatal("same exact orphan replay", err)
	}
	receiptBytes, err := readRegular(root, plan.Receipt.Path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(pathJoin(root, plan.Receipt.Path)); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectConsumedStructuredSourceReplacement(root, ref); err == nil || !strings.Contains(err.Error(), "mutation=unknown") {
		t.Fatal("result pointer without receipt accepted", err)
	}
	writeFixture(t, root, plan.Receipt.Path, receiptBytes)
	writeFixture(t, root, StructuredSourceReplacementPending, []byte("interrupted\n"))
	if _, err := InspectConsumedStructuredSourceReplacement(root, ref); err == nil || !strings.Contains(err.Error(), "mutation=unknown") {
		t.Fatal("partial publication accepted", err)
	}
	if _, err := LoadCurrent(root); err == nil {
		t.Fatal("pending publication loaded")
	}
}

func TestTranslationConsumedReplacementReadonly(t *testing.T) {
	if os.Getenv("GO_TOUR_CONSUMED_REPLACEMENT_PROOF") != "1" {
		t.Skip("explicit readonly real source proof")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	dir := "/tmp/consumed-structured-source-replacement-20261023"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	// A copied root carries real immutable bytes; only temporary proposal/body
	// files are added. No real pointer, corpus or language artifact is written.
	copyRoot := t.TempDir()
	for _, name := range []string{"data", "locales", "_content", "internal", "docs", "go.mod", "go.sum", "UPSTREAM_MANIFEST.tsv", "production", "AGENTS.md", "README.md"} {
		base := filepath.Join(root, name)
		if err := filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			writeFixture(t, copyRoot, filepath.ToSlash(rel), b)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	name := "data/site-shell/translation.html"
	pointer, err := ReferenceFile(copyRoot, structuredSourceCurrent(name))
	if err != nil {
		t.Fatal(err)
	}
	var parent Reference
	if err := readReference(copyRoot, pointer, &parent); err != nil {
		t.Fatal(err)
	}
	a, err := CheckStructuredSourceSuccessor(copyRoot, parent)
	if err != nil {
		t.Fatal(err)
	}
	body, err := readRegularReference(copyRoot, a.Body)
	if err != nil {
		t.Fatal(err)
	}
	u := a.Current.Units[0]
	body = append(append(bytes.Clone(body[:u.Start]), []byte("Synthetic replacement preflight fixture.")...), body[u.End:]...)
	writeFixture(t, copyRoot, "data/readonly-replacement-body.html", body)
	s := documentSource(a.Current)
	s.SHA256 = digest(body)
	d, err := ParseDocument(s, body)
	if err != nil {
		t.Fatal(err)
	}
	currentCorpus, err := readRegular(copyRoot, i18n.GlossaryCorpusPath)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(pathJoin(copyRoot, CorpusSuccessorRoot))
	if err != nil {
		t.Fatal(err)
	}
	var consumed Reference
	for _, entry := range entries {
		r, err := ReferenceFile(copyRoot, CorpusSuccessorRoot+entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		var p CorpusSuccessor
		if err := readReference(copyRoot, r, &p); err != nil {
			t.Fatal(err)
		}
		if p.ResultSHA == digest(currentCorpus) {
			if consumed.Path != "" {
				t.Fatal("ambiguous consumption")
			}
			consumed = r
		}
	}
	g, err := loadSourceCurrent(copyRoot)
	if err != nil {
		t.Fatal(err)
	}
	heads := []Reference{}
	for _, r := range g.StructuredSuccessors {
		v, err := CheckStructuredSourceSuccessor(copyRoot, r)
		if err != nil {
			t.Fatal(err)
		}
		if v.Package == a.Package {
			heads = append(heads, r)
		}
	}
	sort.Slice(heads, func(i, j int) bool { return heads[i].Path < heads[j].Path })
	p := StructuredSourceReplacementProposal{Schema: StructuredSourceReplacementSchema, Package: a.Package, Predecessor: parent, PredecessorIdentity: a.Identity, PriorPointer: pointer, Consumption: consumed, PredecessorHeads: heads, Body: Reference{"data/readonly-replacement-body.html", digest(body)}, Replaced: []SourceSlotReplacement{{Old: u.ID, New: d.Units[0].ID}}}
	b, _ := Encode(p)
	writeFixture(t, copyRoot, "data/readonly-replacement-proposal.json", b)
	ref, _ := ReferenceFile(copyRoot, "data/readonly-replacement-proposal.json")
	plan, err := InspectConsumedStructuredSourceReplacement(copyRoot, ref)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Mutation != "none" || plan.Applied {
		t.Fatal("unexpected mutation")
	}
	for file, raw := range map[string][]byte{"replacement-proposal.json": b, "replacement-body.html": body} {
		if err := os.WriteFile(filepath.Join(dir, file), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	b, _ = json.MarshalIndent(plan, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "translation-readonly-preflight.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(pathJoin(copyRoot, plan.Receipt.Path)); !os.IsNotExist(err) {
		t.Fatal("preflight published receipt")
	}
}
