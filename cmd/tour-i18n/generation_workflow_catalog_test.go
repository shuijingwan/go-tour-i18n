package main

import "testing"

func TestGenerationWorkflowReadonlyCatalogBoundary(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{[]string{"generation-bundle", "current-check"}, true},
		{[]string{"generation-bundle", "import", "--input-dir", "staging"}, true},
		{[]string{"generation-bundle", "import", "--result", "result.zip"}, true},
		{[]string{"retranslation", "process"}, true},
		{[]string{"generation-bundle", "export"}, false},
		{[]string{"quality-check", "snapshot"}, false},
		{[]string{"quality-check", "finalize"}, false},
		{[]string{"retranslation", "promote"}, false},
		{[]string{"site-content", "activation-apply"}, false},
		{nil, false},
	} {
		if got := readonlyGenerationCatalog(tc.args); got != tc.want {
			t.Fatalf("%v: got %v, want %v", tc.args, got, tc.want)
		}
	}
}
