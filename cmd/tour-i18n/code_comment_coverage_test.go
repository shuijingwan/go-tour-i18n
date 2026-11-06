package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func TestCommentBlockDispositionCanonicalClosure(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	c, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sitecontent.AttachPageTranslationUnits(root, c); err != nil {
		t.Fatal(err)
	}
	baseline, err := commentSourceInventory(c)
	if err != nil {
		t.Fatal(err)
	}
	a, records, err := closeCommentInventory(c, baseline)
	if err != nil {
		t.Fatal(err)
	}
	b, more, err := closeCommentInventory(c, baseline)
	if err != nil {
		t.Fatal(err)
	}
	x, _ := json.Marshal([]any{a, records})
	y, _ := json.Marshal([]any{b, more})
	if string(x) != string(y) {
		t.Fatal("closure is not deterministic")
	}
	input, resolved, remaining, total := 0, 0, 0, 0
	for _, r := range records {
		total++
		if r.BaselineUnsupported {
			input++
			if r.Disposition == "ambiguous-unsupported" {
				remaining++
			} else {
				resolved++
			}
		}
		if r.Disposition == "ambiguous-unsupported" && r.ResidualClass == "" {
			t.Fatal("unknown grammar lost its exact residual record")
		}
	}
	// Frozen current source regression; future adoption changes this fixture
	// explicitly rather than quietly dropping blocks from the denominator.
	if total != 432 || input != 235 || resolved != 235 || remaining != 0 {
		t.Fatal(total, input, resolved, remaining)
	}
	counts := commentInventoryCounts(a)
	learn := counts["learn-docs-v1/page"]
	if learn.Slots != 192 || learn.Teaching != 182 || learn.Protected != 10 {
		t.Fatal(learn)
	}
	if err := requireCompleteCommentInventory(a); err != nil {
		t.Fatal("complete source coverage failed", err)
	}
	unknown := []commentInventoryUnit{{Blocks: []i18n.CommentCodeBlock{i18n.SourceCommentBlockDisposition(i18n.AnalyzeCommentCode("unknown source notation ???", "", "1", false), "", nil)}}}
	if err := requireCompleteCommentInventory(unknown); err == nil || !strings.Contains(err.Error(), "cross-locale audit not run") {
		t.Fatal("unknown grammar passed", err)
	}
	proved := []commentInventoryUnit{{Blocks: []i18n.CommentCodeBlock{i18n.SourceCommentBlockDisposition(i18n.AnalyzeCommentCode("var x int\n", "go", "1", false), "", nil)}}}
	if err := requireCompleteCommentInventory(proved); err != nil {
		t.Fatal(err)
	}
}

func TestCommentBlockDispositionFrontierIsSourceOnly(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	writes := 0
	err = writeCommentFrontierPreflight(root, func(name string, value any) error {
		writes++
		if name != "code-comment-frontier-source-preflight" {
			t.Fatal(name)
		}
		b, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		var evidence struct {
			Pages []struct {
				Scope     string        `json:"scope"`
				Route     string        `json:"route"`
				SHA       string        `json:"source_sha256"`
				Formal    bool          `json:"formal_unit_created"`
				Coverage  bool          `json:"current_local_coverage"`
				Counts    commentCounts `json:"counts"`
				Ambiguous int           `json:"ambiguous_unsupported_blocks"`
			} `json:"pages"`
		}
		if err := json.Unmarshal(b, &evidence); err != nil {
			t.Fatal(err)
		}
		if len(evidence.Pages) != 2 || evidence.Pages[0].Route != "/doc/" || evidence.Pages[1].Route != "/doc/install" {
			t.Fatal(evidence)
		}
		for i, p := range evidence.Pages {
			if p.Formal || p.Coverage || p.Scope != "future-source-only-not-current-package" || len(p.SHA) != 64 {
				t.Fatal(p)
			}
			if p.Ambiguous != 0 || p.Counts.Blocks != []int{1, 7}[i] {
				t.Fatal("frontier coverage incomplete or blocks dropped", p)
			}
		}
		return nil
	})
	if err != nil || writes != 1 {
		t.Fatal(writes, err)
	}
}
