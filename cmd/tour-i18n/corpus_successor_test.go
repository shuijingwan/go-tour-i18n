package main

import "testing"

func TestCorpusSuccessorCLIRequiresExplicitAuthorityAndCAS(t *testing.T) {
	for _, args := range [][]string{nil, {"--apply"}, {"--source", "ambiguous"}, {"--input", "../escape"}, {"--input", "missing.json", "extra"}, {"--input", "missing.json", "--apply"}} {
		if err := corpusSuccessorCommand(t.TempDir(), args); err == nil {
			t.Fatal("ambiguous/unsafe/unbound corpus successor request accepted", args)
		}
	}
}
