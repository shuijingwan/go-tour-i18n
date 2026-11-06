package sitecontent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestSurfaceMixedStructuredReadonly(t *testing.T) {
	out := os.Getenv("GO_TOUR_MIXED_FINDING_READONLY")
	if out == "" {
		t.Skip("explicit real receipt readonly transport proof")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	root, start := repositoryRoot(t), time.Now()
	ref, err := ReferenceFile(root, "data/locale-language/zh-CN/surface/20261026-zh-CN-integrated-surface-005.json")
	if err != nil {
		t.Fatal(err)
	}
	b, m, err := ExportStructuredAssetsRevision(root, "zh-CN", ref)
	if err != nil {
		t.Fatal(err)
	}
	sources := []string{}
	for _, a := range m.Plan.Assets {
		if a.State == "generation-required" {
			if a.Document == nil || a.Package != "site-v2-shell" {
				t.Fatal("unexpected structured owner", a.ID, a.Package)
			}
			sources = append(sources, a.Document.Path)
		}
	}
	if len(m.Plan.Expected) != 1 || len(sources) != 1 || sources[0] != "data/site-shell/translation.html" {
		t.Fatal("wrong exact structured selection", m.Plan.Expected, sources)
	}
	transport := filepath.Join(out, "structured-revision-disposable.zip")
	if err := os.WriteFile(transport, b, 0600); err != nil {
		t.Fatal(err)
	}
	report, _ := json.MarshalIndent(map[string]any{"result": "PASS", "expected": m.Plan.Expected, "sources": sources, "selected_count": len(m.Plan.Expected), "identity": m.Identity, "transport_bytes": len(b), "seconds": time.Since(start).Seconds(), "formal_repository_mutation": "none"}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "structured-selection.json"), append(report, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("structured selected=1 exact translation.html; transport only in /tmp; seconds=%.2f", time.Since(start).Seconds())
}
