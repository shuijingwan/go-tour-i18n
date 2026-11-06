package i18n

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test-only exports let external Page integration tests inspect the existing
// latest selector without requiring today's affected scope to be authorized.
func CommentGateLatestSelectionForTest(root string, c *Catalog, locale, snapshot string) error {
	s, err := readQualityCheckSnapshot(root, locale, snapshot, false)
	if err != nil {
		return err
	}
	latest, err := selectLatestRetranslationUnits(root, c, locale)
	if err != nil {
		return err
	}
	if len(latest.ordered) != len(s.Units) {
		return fmt.Errorf("accepted scope count changed")
	}
	for i, u := range latest.ordered {
		old := s.Units[i]
		choice, ok := latest.selectedByID[u.ID]
		if !ok || choice.result.Status != "passed" || u.ID != old.UnitID || u.SourceSHA256 != old.SourceSHA256 {
			return fmt.Errorf("latest accepted selection changed: %s", u.ID)
		}
		if choice.machine != nil {
			if !qualityCheckSnapshotIdentityMatches(old, choice.machine.snapshot(i+1)) {
				return fmt.Errorf("machine candidate selection changed: %s", u.ID)
			}
		} else if !selectedRetranslationIdentityMatches(u, choice) || !choiceMatchesSnapshot(locale, choice, old) {
			return fmt.Errorf("candidate selection changed: %s", u.ID)
		}
	}
	return nil
}

func CommentGateDependencyIdentityForTest(root string) (string, error) {
	return TranslationMachineAuthority(root, commentGateAuthorityPaths)
}

func TestCommentGateCanonicalValidation(t *testing.T) {
	root := t.TempDir()
	for _, locale := range []string{"zh-CN", "en-GB"} {
		metadata, err := os.ReadFile(filepath.Join(repoRoot(t), "locales", locale, "locale.json"))
		if err != nil {
			t.Fatal(err)
		}
		writeSnapshotSource(t, root, "locales/"+locale+"/locale.json", metadata)
		writeSnapshotSource(t, root, "locales/"+locale+"/glossary.yaml", []byte("mandatory:\n  Go: Go\nkeep:\n  - Go\n"))
	}
	code := "package main\n// Return the vulnerability ID.\nfunc main() {}\n//go:generate tool\n"
	e := retranslationTestExample("example:gate.go", "fixtures/gate.go", code)
	c := &Catalog{Examples: []Example{e}}
	u, err := c.Unit(e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateTranslationUnitCandidate(root, c, e.ID, "en-GB", []byte(code)); err != nil {
		t.Fatal("English exact carry rejected", err)
	}
	err = ValidateTranslationUnitCandidate(root, c, e.ID, "zh-CN", []byte(code))
	var failure *CommentGateError
	if !errors.As(err, &failure) || failure.Reason != "non-English teaching-comment exact source carry" || len(failure.SlotIDs) != 1 {
		t.Fatal("canonical dispatch missed exact carry", err)
	}
	translated := strings.Replace(code, "Return the vulnerability ID.", "返回 vulnerability ID。", 1)
	if err := ValidateTranslationUnitCandidate(root, c, e.ID, "zh-CN", []byte(translated)); err != nil {
		t.Fatal("translated teaching rejected", err)
	}
	for name, target := range map[string]string{
		"missing":   strings.Replace(translated, "// 返回 vulnerability ID。\n", "", 1),
		"extra":     strings.Replace(translated, "func main", "// Extra teaching comment.\nfunc main", 1),
		"structure": strings.Replace(translated, "func main()", "func changed()", 1),
		"protected": strings.Replace(translated, "//go:generate tool", "//go:generate changed", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if ValidateTranslationUnitCandidate(root, c, e.ID, "zh-CN", []byte(target)) == nil {
				t.Fatal("invalid candidate accepted")
			}
			// Also prove the shared comment gate, independently of the earlier
			// Go structure validator, rejects each structural failure.
			if _, err := ValidateTranslationUnitCommentCoverage(root, c, u, "zh-CN", []byte(target)); err == nil {
				t.Fatal("comment gate accepted structural failure")
			}
		})
	}
	// Current metadata, rather than a locale-name shortcut, is the authority.
	metadataPath := filepath.Join(root, "locales/zh-CN/locale.json")
	metadata, _ := os.ReadFile(metadataPath)
	if err := os.WriteFile(metadataPath, bytes.Replace(metadata, []byte(`"html_lang": "zh-CN"`), []byte(`"html_lang": "en"`), 1), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := CommentLocaleBaseLanguage(root, "zh-CN"); err == nil {
		t.Fatal("mismatching language authority accepted")
	}
}
