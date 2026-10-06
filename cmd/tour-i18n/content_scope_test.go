package main

import "testing"

func TestContentScopeCommandRejectsInvalidArguments(t *testing.T) {
	for _, args := range [][]string{nil, {"unknown"}, {"check"}, {"check", "--all", "--locale", "he"}, {"bootstrap-tour", "--locale", "he"}, {"repair-uncommitted-v2-a"}, {"repair-uncommitted-v2-a", "--all", "--locale", "he"}, {"init", "--all"}, {"inventory-check"}, {"status", "--all", "unexpected"}} {
		if contentScopeCommand(t.TempDir(), args) == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
