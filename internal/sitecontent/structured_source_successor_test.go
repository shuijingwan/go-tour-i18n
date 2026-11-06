package sitecontent

import (
	"bytes"
	"os"
	"reflect"
	"testing"
)

func TestStructuredSourceSuccessorAppendAuthority(t *testing.T) {
	root := siteFixture(t, "current-authorization-fresh-fixture")
	before, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	name := "data/site-shell/translation.html"
	old, err := readRegular(root, name)
	if err != nil {
		t.Fatal(err)
	}
	at := bytes.LastIndex(old, []byte("</main>"))
	if at < 0 {
		t.Fatal("fixture main missing")
	}
	body := append(bytes.Clone(old[:at]), []byte("<section><h2>Synthetic new source section</h2><p>Synthetic new source prose.</p></section>\n")...)
	body = append(body, old[at:]...)
	writeFixture(t, root, "data/synthetic-source-body.html", body)
	proposal := StructuredSourceProposal{Schema: StructuredSourceSuccessorSchema, Package: "site-v2-shell", Previous: Reference{name, digest(old)}, Body: Reference{"data/synthetic-source-body.html", digest(body)}, InsertAt: at}
	b, _ := Encode(proposal)
	writeFixture(t, root, "data/source-proposal.json", b)
	ref, _ := ReferenceFile(root, "data/source-proposal.json")
	a, err := BuildStructuredSourceSuccessor(root, ref)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.AddedIDs) != 2 || len(a.CarryIDs) != len(a.Old.Units) {
		t.Fatal("exact source slot partition", a)
	}
	if _, err := os.Stat(pathJoin(root, StructuredSourceSuccessorRoot)); !os.IsNotExist(err) {
		t.Fatal("preflight mutation")
	}
	if _, err := AdoptStructuredSourceSuccessor(root, ref, before.Identity); err != nil {
		t.Fatal(err)
	}
	if _, err := AdoptStructuredSourceSuccessor(root, ref, before.Identity); err != nil {
		t.Fatal("identical replay", err)
	}
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	docs, raw, err := CurrentPackageDocuments(root, g, "site-v2-shell")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range docs {
		if d.Path == name && (!bytes.Equal(raw[name], body) || !reflect.DeepEqual(d, a.Current)) {
			t.Fatal("current source projection")
		}
	}
	actual, _ := readRegular(root, name)
	if !bytes.Equal(actual, old) {
		t.Fatal("historical source mutated")
	}
	writeFixture(t, root, proposal.Body.Path, append(body, ' '))
	if _, err := LoadCurrent(root); err == nil {
		t.Fatal("successor body drift accepted")
	}
	writeFixture(t, root, proposal.Body.Path, body)
	for _, change := range []func(*StructuredSourceProposal){
		func(p *StructuredSourceProposal) { p.Package = "learn-docs-v1" },
		func(p *StructuredSourceProposal) { p.InsertAt = 0 },
		func(p *StructuredSourceProposal) { p.Body = p.Previous },
		func(p *StructuredSourceProposal) { p.Previous.SHA256 = digest([]byte("wrong")) },
	} {
		p := proposal
		change(&p)
		b, _ := Encode(p)
		writeFixture(t, root, "data/bad-source-proposal.json", b)
		r, _ := ReferenceFile(root, "data/bad-source-proposal.json")
		if _, err := BuildStructuredSourceSuccessor(root, r); err == nil {
			t.Fatal("invalid source successor accepted")
		}
	}
}
