package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestShellParserSemanticDependency(t *testing.T) {
	root := repositoryRoot(t)
	path := "data/site-shell/successor/sources/30d7c2ef242aa2aa13e64aa5a8917baca377710cb0ef77a68dba3cfb27694272.json"
	before, err := readRegular(root, path)
	if err != nil {
		t.Fatal(err)
	}
	var a ShellSuccessorAuthority
	if err = json.Unmarshal(before, &a); err != nil {
		t.Fatal(err)
	}
	if err = checkShellParserDependencies(root, a); err != nil {
		t.Fatal(err)
	}
	t.Logf("frozen_semantic_identity=%s", shellParserSemanticIdentity(a.Sources))
	tmp := t.TempDir()
	for _, s := range a.Sources {
		b, err := readRegular(root, s.Path)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, tmp, s.Path, b)
	}
	for _, ref := range a.ParserReferences {
		b, err := readRegular(root, ref.Path)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, tmp, ref.Path, b)
	}
	file := "internal/sitecontent/formal_page.go"
	b, err := readRegular(tmp, file)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, tmp, file, append(b, []byte("\n// Unrelated implementation edit.\n")...))
	if err = checkShellParserDependencies(tmp, a); err != nil {
		t.Fatal("unrelated file edit caused false stale", err)
	}
	// A self-consistent expected representation with changed semantics cannot
	// be accepted merely because source SHA and an implementation file match.
	bad := a
	bad.Sources = append([]ShellLanguageSource{}, a.Sources...)
	bad.Sources[0].StructureSHA = digest([]byte("other semantic output"))
	bad.ParserSemanticIdentity = shellParserSemanticIdentity(bad.Sources)
	if checkShellParserDependencies(tmp, bad) == nil {
		t.Fatal("semantic output drift accepted")
	}
	writeFixture(t, tmp, "go.mod", []byte("module changed\n"))
	if checkShellParserDependencies(tmp, a) == nil {
		t.Fatal("real module dependency drift accepted")
	}
	after, err := os.ReadFile(filepath.Join(root, path))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("old evidence mutated")
	}
}
