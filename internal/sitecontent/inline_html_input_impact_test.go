package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"golang.org/x/net/html"
)

func TestFrozenPageInlineHTMLAltCoordinates(t *testing.T) {
	root := repositoryRoot(t)
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	docs, raw, err := WorkflowDocuments(root, g, "learn-docs-v1")
	if err != nil {
		t.Fatal(err)
	}
	wanted := map[string]struct{ index, images int }{
		"_content/doc/modules/major-version.md":   {32, 1},
		"_content/doc/modules/managing-source.md": {34, 3},
		"_content/doc/modules/version-numbers.md": {37, 1},
		"_content/doc/security/fuzz/index.md":     {41, 2},
	}
	for _, d := range docs {
		want, ok := wanted[d.Path]
		if !ok {
			continue
		}
		delete(wanted, d.Path)
		t.Run(d.Path, func(t *testing.T) {
			if d.StableIndex != want.index {
				t.Fatal("frozen registry coordinate changed", d.StableIndex)
			}
			z := html.NewTokenizer(bytes.NewReader(raw[d.Path]))
			count := 0
			candidate := d.Units[0].Source
			for {
				typ := z.Next()
				if typ == html.ErrorToken {
					break
				}
				if typ != html.StartTagToken && typ != html.SelfClosingTagToken {
					continue
				}
				// TagName can mutate tokenizer Raw; retain the original bytes.
				tagRaw := bytes.Clone(z.Raw())
				name, _ := z.TagName()
				if string(name) != "img" {
					continue
				}
				if err := htmlAttributeSpans(tagRaw, 0, "html", func(a, b int, context string) {
					if context != "html/attribute/alt" {
						return
					}
					count++
					visible, _, err := protectBlock(tagRaw[a:b], context)
					if err != nil {
						t.Fatal(err)
					}
					// Derive a natural-language coordinate from the frozen source,
					// without a locale replacement or candidate transformation.
					prefix, _, _ := strings.Cut(visible, "⟦")
					prefix = strings.TrimSpace(prefix)
					if prefix == "" || !strings.Contains(candidate, prefix) {
						t.Fatalf("alt natural language remains locked: %q", visible)
					}
					candidate = strings.Replace(candidate, prefix, "Editable attribute fixture", 1)
				}); err != nil {
					t.Fatal(err)
				}
			}
			if count != want.images {
				t.Fatalf("alt count=%d want=%d", count, want.images)
			}
			adapter := PageAdapter{d, raw[d.Path]}
			got, err := adapter.Restore(candidate)
			if err != nil {
				t.Fatal("editable alt did not restore", err)
			}
			if err := validatePageStructure(d, raw[d.Path], got); err != nil {
				t.Fatal(err)
			}
			if bytes.Count(got, []byte("Editable attribute fixture")) != want.images {
				t.Fatal("alt replacements escaped their coordinates")
			}
			t.Logf("stable=%d editable image alts=%d input_sha=%s UnitID=%s", d.StableIndex, count, digest([]byte(d.Units[0].Source+"\n")), d.Units[0].ID)
		})
	}
	if len(wanted) != 0 {
		t.Fatal("missing frozen exact pages", wanted)
	}
}

// Explicit read-only comparison against the preserved formal generation ZIP.
// No export, recovery staging, candidate, validation or receipt is written.
func TestInlineHTMLRevisionBundleInputImpact(t *testing.T) {
	bundlePath := os.Getenv("GO_TOUR_INLINE_HTML_BEFORE_BUNDLE")
	if bundlePath == "" {
		t.Skip("explicit preserved revision Bundle comparison")
	}
	bundle, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	if digest(bundle) != "fab3d6a3a4587201ed0642a1800d73a4ebd58f4463d7acead3fc05b04522fc83" {
		t.Fatal("preserved bundle SHA mismatch")
	}
	files, err := i18n.ReadTransportBundle(bundle, 512, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	var manifest i18n.GenerationBundleManifest
	if err := json.Unmarshal(files["manifest.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.InputIdentitySHA256 != "5f5dfceb6c1940076dc9310a51f91b0c411c178798c2757ec656c2ff93654ef3" || manifest.BatchID != "chatgpt-zh-CN-040" {
		t.Fatal("preserved bundle identity mismatch")
	}
	if err := i18n.ValidateTransportBundleInventory(files, manifest.Files, true); err != nil {
		t.Fatal(err)
	}
	var batch i18n.RetranslationBatchManifest
	if err := json.Unmarshal(files["formal/manifest.json"], &batch); err != nil {
		t.Fatal(err)
	}
	if len(batch.Units) != 6 {
		t.Fatal("six exact formal units required")
	}
	root := repositoryRoot(t)
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	docs, raw, err := WorkflowDocuments(root, g, "learn-docs-v1")
	if err != nil {
		t.Fatal(err)
	}
	unchanged, changed := 0, 0
	for _, record := range batch.Units {
		found := false
		for _, d := range docs {
			if d.Path != record.SourcePath {
				continue
			}
			found = true
			if d.SourceSHA != record.SourceSHA256 {
				t.Fatal("physical source drift", d.Path)
			}
			input := files["formal/"+record.InputPath]
			if digest(input) != record.InputSHA256 {
				t.Fatal("old formal input SHA mismatch")
			}
			newInput := []byte(stringsTrimLF(d.Units[0].Source) + "\n")
			baseline := files["formal/protected-candidates/"+filepath.Base(record.InputPath)]
			candidate := files["formal/candidates/"+filepath.Base(record.InputPath)]
			if len(baseline) == 0 || len(candidate) == 0 {
				t.Fatal("missing preserved baseline/candidate")
			}
			adapter := PageAdapter{d, raw[d.Path]}
			got, restoreErr := adapter.Restore(string(baseline))
			if d.StableIndex == 10 || d.StableIndex == 29 {
				unchanged++
				if d.Units[0].ID != record.UnitID || !bytes.Equal(input, newInput) || restoreErr != nil || !bytes.Equal(append(bytes.TrimRight(got, "\n"), '\n'), candidate) {
					t.Fatalf("trusted Unit input/baseline changed: stable=%d: %v", d.StableIndex, restoreErr)
				}
				t.Logf("stable=%d UnitID/input unchanged input_sha=%s; preserved protected baseline sha=%s restores byte-exact predecessor", d.StableIndex, digest(newInput), digest(baseline))
			} else {
				changed++
				if bytes.Equal(input, newInput) || d.Units[0].ID == record.UnitID || restoreErr == nil {
					t.Fatal("expected changed protected contract and stale baseline", d.Path)
				}
				t.Logf("stable=%d input_sha old=%s new=%s; UnitID old=%s new=%s; old baseline rejected: %v", d.StableIndex, digest(input), digest(newInput), record.UnitID, d.Units[0].ID, restoreErr)
			}
		}
		if !found {
			t.Fatal("missing exact formal source", record.SourcePath)
		}
	}
	if unchanged != 2 || changed != 4 {
		t.Fatal("unexpected exact input impact", unchanged, changed)
	}
	// Exercise the actual recovery preflight with current frozen descriptors
	// and temporary empty transport/staging locations. The stale gate must
	// reject the old batch before any staging or formal write is possible.
	catalog := &i18n.Catalog{}
	if _, err := AttachPageTranslationUnits(root, catalog); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "must-not-exist")
	_, err = i18n.RecoverGenerationOutputs(root, catalog, bundle, []string{t.TempDir()}, target)
	if err == nil || !strings.Contains(err.Error(), "generation bundle is stale: stale formal unit:") || !strings.Contains(err.Error(), batch.Units[2].UnitID) {
		t.Fatalf("expected exact stale UnitID blocker: %v", err)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatal("failed preflight wrote recovery staging", err)
	}
	t.Logf("same-batch recovery blocker: %v", err)
}
