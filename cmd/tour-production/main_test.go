package main

import (
	"strings"
	"testing"
)

func TestRunRequiresBuildInjectedLocale(t *testing.T) {
	original := productionLocale
	productionLocale = ""
	t.Cleanup(func() { productionLocale = original })
	if err := run(nil); err == nil || !strings.Contains(err.Error(), "was not injected") {
		t.Fatalf("run without injected locale error = %v", err)
	}
}
