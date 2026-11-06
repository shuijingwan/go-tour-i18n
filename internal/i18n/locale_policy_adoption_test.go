//go:build linux

package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func policyAdoptionFixture(t *testing.T) (string, LocalePolicyAdoptionOptions) {
	t.Helper()
	root := t.TempDir()
	locale := "fr-FR"
	writeInitializableLocale(t, root, locale, Locale{Locale: locale, HTMLLang: locale, LanguageName: "Français", EnglishName: "French", Phase: "scaffold", TranslationUnit: "present.Section"})
	authority := testLanguageAuditAuthority(locale, "Latin", map[string]string{"。": "."})
	b, err := json.Marshal(authority)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "policy.json"), b, 0644); err != nil {
		t.Fatal(err)
	}
	options := LocalePolicyAdoptionOptions{Locale: locale, Input: "policy.json"}
	p, err := AdoptLocaleLanguagePolicy(root, options)
	if err != nil {
		t.Fatal(err)
	}
	options.Apply = true
	options.PriorSHA256 = p.PriorSHA256
	options.InputSHA256 = p.InputPolicySHA256
	options.ResultSHA256 = p.ResultSHA256
	return root, options
}
func policyAdoptionRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func policyAdoptionNoStaging(t *testing.T, root string) {
	t.Helper()
	p, err := filepath.Glob(filepath.Join(root, "locales/fr-FR/.language-policy-adopt-*"))
	if err != nil || len(p) != 0 {
		t.Fatal("publication left staging files", p, err)
	}
}

func TestLocalePolicyAdoptionConcurrentDriftFailClosed(t *testing.T) {
	for _, what := range []string{"metadata-bytes", "input-bytes", "metadata-inode", "input-inode", "metadata-symlink", "abort"} {
		t.Run(what, func(t *testing.T) {
			root, o := policyAdoptionFixture(t)
			metadata := filepath.Join(root, "locales/fr-FR/locale.json")
			policy := filepath.Join(root, "policy.json")
			expected := policyAdoptionRead(t, metadata)
			_, err := adoptLocaleLanguagePolicy(root, o, func() error {
				path := metadata
				if strings.HasPrefix(what, "input-") {
					path = policy
				}
				switch what {
				case "abort":
					return fmt.Errorf("test failure before atomic publication")
				case "metadata-symlink":
					if err := os.Rename(metadata, metadata+".old"); err != nil {
						return err
					}
					if err := os.Symlink("locale.json.old", metadata); err != nil {
						return err
					}
				case "metadata-inode", "input-inode":
					b, err := os.ReadFile(path)
					if err != nil {
						return err
					}
					if err := os.Rename(path, path+".old"); err != nil {
						return err
					}
					if err := os.WriteFile(path, b, 0644); err != nil {
						return err
					}
				default:
					b, err := os.ReadFile(path)
					if err != nil {
						return err
					}
					if err := os.WriteFile(path, append(b, '\n'), 0644); err != nil {
						return err
					}
					if path == metadata {
						expected = append(expected, '\n')
					}
				}
				return nil
			})
			if err == nil || !strings.Contains(err.Error(), "mutation=none") {
				t.Fatal("concurrent change/abort did not fail closed", err)
			}
			if !bytes.Equal(policyAdoptionRead(t, metadata), expected) {
				t.Fatal("publisher overwrote concurrent writer or partially published")
			}
			policyAdoptionNoStaging(t, root)
		})
	}
}

func TestLocalePolicyAdoptionAtomicAndSerialized(t *testing.T) {
	root, o := policyAdoptionFixture(t)
	path := filepath.Join(root, "locales/fr-FR/locale.json")
	before := policyAdoptionRead(t, path)
	old, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer old.Close()
	oldInfo, err := old.Stat()
	if err != nil {
		t.Fatal(err)
	}
	staged := make(chan struct{})
	proceed := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := adoptLocaleLanguagePolicy(root, o, func() error { close(staged); <-proceed; return nil })
		done <- err
	}()
	select {
	case <-staged:
	case err := <-done:
		t.Fatalf("publisher failed before staging: %v", err)
	}
	// A second actual publisher cannot pass CAS concurrently with the first.
	_, err = AdoptLocaleLanguagePolicy(root, o)
	if err == nil || !strings.Contains(err.Error(), "busy") {
		close(proceed)
		<-done
		t.Fatal("concurrent publisher was not rejected", err)
	}
	if !bytes.Equal(policyAdoptionRead(t, path), before) {
		close(proceed)
		<-done
		t.Fatal("staging exposed partial target bytes")
	}
	close(proceed)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	after := policyAdoptionRead(t, path)
	if sum(after) != o.ResultSHA256 {
		t.Fatal("atomic result identity mismatch")
	}
	oldBytes, err := io.ReadAll(old)
	if err != nil || !bytes.Equal(oldBytes, before) {
		t.Fatal("reader observed partial/in-place metadata rewrite", err)
	}
	now, err := os.Stat(path)
	if err != nil || os.SameFile(oldInfo, now) {
		t.Fatal("metadata was not atomically replaced", err)
	}
	if _, err := AdoptLocaleLanguagePolicy(root, o); err == nil {
		t.Fatal("old prior silently refreshed after publication")
	}
	policyAdoptionNoStaging(t, root)
}
