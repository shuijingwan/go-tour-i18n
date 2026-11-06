package sitecontent

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestStructuredSourceSuccessorDeltaLifecycle(t *testing.T) {
	root, _, _, _ := structuredLifecycleFixture(t, "site-v2-shell")
	name := "data/site-shell/translation.html"
	old, _ := readRegular(root, name)
	at := bytes.LastIndex(old, []byte("</main>"))
	body := append(bytes.Clone(old[:at]), []byte("<section><h2>Synthetic addition</h2><p>Synthetic new source text.</p></section>\n")...)
	body = append(body, old[at:]...)
	writeFixture(t, root, "data/synthetic-expanded-source.html", body)
	proposal := StructuredSourceProposal{Schema: StructuredSourceSuccessorSchema, Package: "site-v2-shell", Previous: Reference{name, digest(old)}, Body: Reference{"data/synthetic-expanded-source.html", digest(body)}, InsertAt: at}
	b, _ := Encode(proposal)
	writeFixture(t, root, "data/source-expansion.json", b)
	ref, _ := ReferenceFile(root, "data/source-expansion.json")
	a, err := BuildStructuredSourceSuccessor(root, ref)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AdoptStructuredSourceSuccessor(root, ref, a.BaseIdentity); err != nil {
		t.Fatal(err)
	}
	authority, _ := ReferenceFile(root, StructuredSourceSuccessorPath(a))
	delta, err := PlanStructuredSourceSuccessor(root, "zh-CN", authority)
	if err != nil {
		t.Fatal(err)
	}
	if len(delta.Pending) != 2 || len(delta.Carry) != len(a.Old.Units) {
		t.Fatal("wrong pending/carry set")
	}
	if _, err := PlanStructuredAssets(root, "zh-CN"); err == nil {
		t.Fatal("old corpus authorized added source")
	}
	corpus, _ := readRegular(root, i18n.GlossaryCorpusPath)
	if _, err := AdoptCorpusProjection(root, digest(corpus), true); err != nil {
		t.Fatal(err)
	}
	if err := CheckUnifiedCorpus(root); err != nil {
		t.Fatal("current source corpus", err)
	}
	if _, err := PlanStructuredAssets(root, "zh-CN"); err == nil {
		t.Fatal("old full Review authorized expanded corpus")
	}
	review, _, err := i18n.ExportUnifiedGlossaryBundle(root, "zh-CN", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.RecordUnifiedGlossaryReview(root, review, "synthetic-expanded-review", "synthetic-independent-reviewer", "synthetic-generation", "passed", nil); err != nil {
		t.Fatal(err)
	}
	zip, m, err := ExportStructuredAssets(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CheckStructuredAssetsBundle(root, zip); err != nil {
		t.Fatal("Bundle currentness", err)
	}
	var asset StructuredAsset
	for _, v := range m.Plan.Assets {
		if v.Document != nil && v.Document.Path == name {
			asset = v
		}
	}
	if asset.State != "generation-required" || !reflect.DeepEqual(asset.CarriedSlots, delta.Carry) {
		t.Fatal("exact carry not frozen")
	}
	if !reflect.DeepEqual(m.Plan.Expected, []string{asset.TargetPath}) {
		t.Fatal("unrelated structured output scope", m.Plan.Expected)
	}
	carry := map[string]bool{}
	for _, v := range asset.CarriedSlots {
		carry[v.ID] = true
	}
	target := StructuredTarget{Path: asset.TargetPath, Slots: []Target{}}
	for _, u := range asset.Document.Units {
		if !carry[u.ID] {
			target.Slots = append(target.Slots, Target{ID: u.ID, Text: u.Source})
		}
	}
	out := StructuredOutput{Schema: StructuredSchema, BundleSHA: m.Identity, Outputs: []StructuredTarget{target}}
	b, _ = Encode(out)
	if _, _, err := ImportStructuredAssets(root, zip, b, Provenance{Provider: "codex", Model: "gpt-5.6-sol-high", Session: "synthetic-generation", GeneratedAt: "2026-10-22T00:00:00Z"}); err != nil {
		t.Fatal("shared delta restore/import", err)
	}
	bad := target
	bad.Slots = append(append([]Target{}, target.Slots...), asset.CarriedSlots[0])
	if _, err := structuredTargetValues(asset, bad, &i18n.Glossary{Mandatory: map[string]string{}}); err == nil {
		t.Fatal("Generation allowed to rewrite carried slot")
	}
	bad = target
	bad.Slots = append([]Target{}, target.Slots...)
	bad.Slots[0].ID = "unknown"
	if _, err := structuredTargetValues(asset, bad, &i18n.Glossary{Mandatory: map[string]string{}}); err == nil {
		t.Fatal("wrong output slot identity accepted")
	}
	actual, _ := readRegular(root, name)
	if !bytes.Equal(actual, old) {
		t.Fatal("historical source changed")
	}
}
