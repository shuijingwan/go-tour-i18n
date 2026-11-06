package main

import "testing"

func TestRetryHandoffCLIRejectsInvalidFlagSets(t *testing.T) {
	for _, test := range []struct {
		task string
		args []string
	}{
		{"retry-handoff", nil},
		{"retry-handoff", []string{"--bundle", "missing.zip"}},
		{"retry-handoff", []string{"--bundle", "missing.zip", "--output", "out.zip", "--output-dir", "out"}},
		{"retry-handoff-check", []string{"--bundle", "a.zip", "--bundle", "b.zip"}},
		{"retry-handoff-unpack", []string{"--bundle", "a.zip", "--output", "out.zip"}},
	} {
		if err := retryHandoffCommand(t.TempDir(), nil, test.task, test.args); err == nil {
			t.Fatal("invalid CLI arguments accepted", test)
		}
	}
}
