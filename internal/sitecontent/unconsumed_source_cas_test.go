package sitecontent

import (
	"bytes"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestUnconsumedStructuredSourceCorrectionCAS(t *testing.T) {
	root := siteFixture(t, "current-authorization-fresh-fixture")
	old, _ := readRegular(root, "data/site-shell/translation.html")
	at := bytes.LastIndex(old, []byte("</main>"))
	proposal := func(label string) Reference {
		body := append(bytes.Clone(old[:at]), []byte("<section><p>Synthetic "+label+" source content.</p></section>\n")...)
		body = append(body, old[at:]...)
		name := "data/" + label + "-body.html"
		writeFixture(t, root, name, body)
		p := StructuredSourceProposal{Schema: StructuredSourceSuccessorSchema, Package: "site-v2-shell", Previous: Reference{"data/site-shell/translation.html", digest(old)}, Body: Reference{name, digest(body)}, InsertAt: at}
		b, _ := Encode(p)
		name = "data/" + label + "-proposal.json"
		writeFixture(t, root, name, b)
		ref, _ := ReferenceFile(root, name)
		return ref
	}
	first := proposal("first")
	a, err := BuildStructuredSourceSuccessor(root, first)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AdoptStructuredSourceSuccessor(root, first, a.BaseIdentity); err != nil {
		t.Fatal(err)
	}
	pointer, _ := ReferenceFile(root, structuredSourceCurrent(a.Previous.Path))
	second := proposal("corrected")
	if _, err := CorrectUnconsumedStructuredSource(root, second, a.BaseIdentity, digest([]byte("wrong"))); err == nil {
		t.Fatal("wrong pointer CAS accepted")
	}
	b, err := CorrectUnconsumedStructuredSource(root, second, a.BaseIdentity, pointer.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CorrectUnconsumedStructuredSource(root, second, a.BaseIdentity, pointer.SHA256); err != nil {
		t.Fatal("identical replay", err)
	}
	if _, err := CheckStructuredSourceSuccessor(root, Reference{StructuredSourceSuccessorPath(a), func() string { v, _ := Encode(a); return digest(v) }()}); err != nil {
		t.Fatal("old receipt/body rewritten", err)
	}
	corpus, _ := readRegular(root, i18n.GlossaryCorpusPath)
	if _, err := AdoptCorpusProjection(root, digest(corpus), true); err != nil {
		t.Fatal(err)
	}
	if err := CheckUnifiedCorpus(root); err != nil {
		t.Fatal("installed source corpus canonical identity", err)
	}
	current, _ := ReferenceFile(root, structuredSourceCurrent(a.Previous.Path))
	if _, err := CorrectUnconsumedStructuredSource(root, proposal("forbidden"), b.BaseIdentity, current.SHA256); err == nil {
		t.Fatal("consumed source authority overwritten")
	}
	after, _ := ReferenceFile(root, current.Path)
	if after != current {
		t.Fatal("rejected correction mutated pointer")
	}
}
