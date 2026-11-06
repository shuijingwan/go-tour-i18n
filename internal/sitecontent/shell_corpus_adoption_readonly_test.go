package sitecontent

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Explicit repository acceptance, never a formal closure/authorization writer.
// All forbidden artifact trees are fingerprinted before and after the reader.
func TestShellCorpusAdoptionReadonlyRepository(t *testing.T) {
	if os.Getenv("GO_TOUR_SHELL_CORPUS_ADOPTION_READONLY") != "1" {
		t.Skip("explicit real repository read-only acceptance")
	}
	root := repositoryRoot(t)
	fingerprint := func() map[string]string {
		t.Helper()
		out := map[string]string{}
		for _, name := range []string{"data/site-shell/successor", "data/glossary-source-corpus.json", "data/quality-check-snapshots", "data/translation-authorizations", "data/locale-language", "data/site-workflow"} {
			err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(name)), func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() {
					return nil
				}
				b, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				out[path] = digest(b)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		return out
	}
	before := fingerprint()
	defer func() {
		if !reflect.DeepEqual(before, fingerprint()) {
			t.Error("FORMAL_ARTIFACT_MUTATION detected")
		}
	}()
	done := contentidentity.BeginMemoization()
	defer done()
	adoptionRef := Reference{Path: "data/site-shell/successor/adoptions/2ec50f613ab4b48abf1dd58078b47c89626bae8d6050d24944aa389bbac258c6.json", SHA256: "229dd61298b389fe9f8fed9814c3762571b1956bbd55e92131a487532d73fb04"}
	adoption, err := CheckShellAdoption(root, adoptionRef)
	if err != nil {
		t.Fatal("ADOPTION_VERIFIED=FAILED", err)
	}
	var frozen ShellSuccessorAuthority
	if err := readReference(root, adoption.Foundation, &frozen); err != nil {
		t.Fatal("FROZEN_AUTHORITY_VERIFIED=FAILED", err)
	}
	oldSHA := ""
	for _, ref := range frozen.ParserReferences {
		if ref.Path == "internal/sitecontent/formal_page.go" {
			oldSHA = ref.SHA256
		}
	}
	current, err := ReferenceFile(root, "internal/sitecontent/formal_page.go")
	if err != nil || oldSHA != "1847269422e692248ef2556db07aeab05b4d46dcae6bfe41c3ceebf3a303067c" || current.SHA256 != "5109232b1ee3765fa4f9ea6686d72ced263e2a321ff60dd0387d33ab0838113a" {
		t.Fatal("parser fixture drift", oldSHA, current, err)
	}
	if err := verifyShellParserDependenciesThroughAdoption(root, adoption.Foundation, frozen); err != nil {
		t.Fatal("CURRENT_PARSER_TRANSITION_VERIFIED=FAILED", err)
	}
	t.Log("FROZEN_AUTHORITY_VERIFIED=PASS ADOPTION_VERIFIED=PASS CURRENT_PARSER_TRANSITION_VERIFIED=PASS CURRENT_SHELL_PROJECTION_VERIFIED=PASS")
	if applied, err := CheckAppliedCorpusSuccessor(root); err != nil || !applied {
		t.Fatal("CORPUS_SUCCESSOR_READONLY=FAILED", applied, err)
	}
	if err := CheckUnifiedCorpus(root); err != nil {
		t.Fatal("CORPUS_SUCCESSOR_READONLY=FAILED", err)
	}
	t.Log("CORPUS_SUCCESSOR_READONLY=PASS")
	page := Reference{Path: "data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-015/finalization.json", SHA256: "530374a7aa563c103f975940b480f998fbd315541c56438e21d2557e22275b1d"}
	generations := []Reference{
		{Path: "data/locale-language/zh-CN/structured-generation/3aa00f3907d55604a78a411e88fe0fb478e232dd5f80cbfea66ca9e80c53bc78.json", SHA256: "e4c7a8b4e9fa94cf06422d9c1a376777faf3a9e43a03f0694a90c358cebd0413"},
		{Path: "data/locale-language/zh-CN/structured-generation/7caebba33580782e5837719efa23a8d1cf4d60f51de84fa3e5c56c8816d25cb5.json", SHA256: "3bdea5f48d19a48c69964d515d87117a095015d9aa6a33e2ad4837b59a10c8c3"},
	}
	closure, files, err := BuildPackageClosure(root, "zh-CN", "learn-docs-v1", &page, generations)
	if err != nil {
		t.Fatal("PACKAGE_CLOSURE_READONLY_EQUIVALENT=FAILED NEXT_BLOCKER=", err)
	}
	if len(files) != 56 || len(closure.Files) != 56 {
		t.Fatal("non-exact learn-docs closure", len(files), len(closure.Files))
	}
	t.Log("PACKAGE_CLOSURE_READONLY_EQUIVALENT=PASS localized_files=56; no artifact saved")
}
