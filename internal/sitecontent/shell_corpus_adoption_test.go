package sitecontent

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestShellCorpusParserAdoptionCompatibility(t *testing.T) {
	root, _, generation, adoptionRef := shellAdoptionFixture(t)
	var adoption ShellAdoption
	if err := readReference(root, adoptionRef, &adoption); err != nil {
		t.Fatal(err)
	}
	var original ShellSuccessorAuthority
	if err := readReference(root, adoption.Foundation, &original); err != nil {
		t.Fatal(err)
	}
	completion, err := ReferenceFile(root, LocalePath("zh-CN"))
	if err != nil {
		t.Fatal(err)
	}
	scope := ShellAffectedScope{Schema: ShellAffectedSchema, State: "pending-language-work", Package: "site-v2-shell", Locale: "zh-CN", SourceAuthority: adoption.Foundation, SourceIdentity: original.Identity, CurrentCompletion: completion}
	scope.Terminology.RetiredSources = []string{original.HistoricalHome.Path}
	for _, source := range original.Sources {
		contributor, err := ProjectShellSuccessorContributor(source)
		if err != nil {
			t.Fatal(err)
		}
		scope.Terminology.Contributors = append(scope.Terminology.Contributors, contributor)
	}
	sort.Slice(scope.Terminology.Contributors, func(i, j int) bool {
		return scope.Terminology.Contributors[i].ID < scope.Terminology.Contributors[j].ID
	})
	scope.Identity = identity(scope)
	affectedPath := ShellSuccessorRoot + "affected/zh-CN/" + scope.Identity + ".json"
	b, err := Encode(scope)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, affectedPath, b)
	encode := func(v any) []byte {
		t.Helper()
		b, err := Encode(v)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	for _, kind := range []string{
		"exact without adoption", "compatible", "missing adoption", "wrong foundation dependency",
		"adoption tamper", "adoption identity", "adoption path", "adoption canonical bytes", "pointer canonical bytes",
		"frozen tamper", "module evolution", "unapproved runtime dependency", "source bytes", "source membership", "source SHA", "corpus membership", "projection drift",
	} {
		t.Run(kind, func(t *testing.T) {
			restored := map[string][]byte{}
			write := func(path string, b []byte) {
				t.Helper()
				if _, seen := restored[path]; !seen {
					before, err := readRegular(root, path)
					if err != nil {
						t.Fatal(err)
					}
					restored[path] = before
				}
				writeFixture(t, root, path, b)
			}
			defer func() {
				for path, b := range restored {
					writeFixture(t, root, path, b)
				}
			}()
			var a ShellSuccessorAuthority
			// Round-trip private fixture values before changing nested slices.
			if err := StrictJSON(encode(original), &a); err != nil {
				t.Fatal(err)
			}
			foundation := adoption.Foundation
			if kind != "exact without adoption" {
				p := "internal/sitecontent/formal_page.go"
				b, err := readRegular(root, p)
				if err != nil {
					t.Fatal(err)
				}
				write(p, append(bytes.Clone(b), []byte("\n// Equivalent extractor implementation evolution.\n")...))
			}
			publishAdoption := func(next ShellAdoption) {
				next.Identity = ""
				next.Identity = identity(next)
				b := encode(next)
				ref := Reference{Path: shellAdoptionPath(&next), SHA256: digest(b)}
				writeFixture(t, root, ref.Path, b)
				write(ShellAdoptionCurrent, encode(ref))
			}
			publishFoundation := func() {
				a.ParserSemanticIdentity = shellParserSemanticIdentity(a.Sources)
				a.Identity = ""
				a.Identity = identity(a)
				var err error
				foundation, err = shellAuthorityReference(&a)
				if err != nil {
					t.Fatal(err)
				}
				writeFixture(t, root, foundation.Path, encode(a))
				next := adoption
				next.Foundation = foundation
				publishAdoption(next)
			}
			switch kind {
			case "exact without adoption", "missing adoption":
				b, err := readRegular(root, ShellAdoptionCurrent)
				if err != nil {
					t.Fatal(err)
				}
				restored[ShellAdoptionCurrent] = b
				if err := os.Remove(filepath.Join(root, ShellAdoptionCurrent)); err != nil {
					t.Fatal(err)
				}
			case "wrong foundation dependency":
				// The current adoption may be valid for a different dependency
				// inventory; it must never authorize the requested old foundation.
				kept := []Reference{}
				for _, ref := range a.ParserReferences {
					if ref.Path != "internal/sitecontent/formal_page.go" {
						kept = append(kept, ref)
					}
				}
				a.ParserReferences = kept
				publishFoundation()
				a, foundation = original, adoption.Foundation
			case "adoption tamper":
				write(adoptionRef.Path, append(encode(adoption), '\n'))
			case "adoption identity":
				bad := adoption
				bad.Identity = strings.Repeat("0", 64)
				b := encode(bad)
				write(adoptionRef.Path, b)
				write(ShellAdoptionCurrent, encode(Reference{Path: adoptionRef.Path, SHA256: digest(b)}))
			case "adoption path":
				b := encode(adoption)
				ref := Reference{Path: ShellSuccessorRoot + "adoptions/alias.json", SHA256: digest(b)}
				writeFixture(t, root, ref.Path, b)
				write(ShellAdoptionCurrent, encode(ref))
			case "adoption canonical bytes":
				b := append(encode(adoption), '\n')
				write(adoptionRef.Path, b)
				write(ShellAdoptionCurrent, encode(Reference{Path: adoptionRef.Path, SHA256: digest(b)}))
			case "pointer canonical bytes":
				write(ShellAdoptionCurrent, append(encode(adoptionRef), '\n'))
			case "frozen tamper":
				write(foundation.Path, append(encode(a), '\n'))
			case "module evolution", "unapproved runtime dependency":
				p := "go.mod"
				if kind == "unapproved runtime dependency" {
					p = "internal/sitecontent/shell_generation_context.go"
				}
				b, err := readRegular(root, p)
				if err != nil {
					t.Fatal(err)
				}
				write(p, append(bytes.Clone(b), []byte("\n// Unapproved dependency evolution.\n")...))
			case "source bytes":
				p := a.Sources[0].Path
				b, err := readRegular(root, p)
				if err != nil {
					t.Fatal(err)
				}
				write(p, append(bytes.Clone(b), '\n'))
			case "source membership":
				a.Sources[0].Path = a.Sources[1].Path
				publishFoundation()
			case "source SHA":
				a.Sources[0].SourceSHA = strings.Repeat("0", 64)
				publishFoundation()
			case "corpus membership":
				corpus, err := i18n.LoadUnifiedGlossaryCorpus(root)
				if err != nil {
					t.Fatal(err)
				}
				kept := []i18n.LanguageContributor{}
				for _, c := range corpus.Contributors {
					if c.Path != a.Sources[0].Path {
						kept = append(kept, c)
					}
				}
				corpus.Contributors = kept
				corpus.Identity = i18n.CorpusIdentity(*corpus)
				write(i18n.GlossaryCorpusPath, encode(corpus))
			case "projection drift":
				a.Sources[0].Slots[0].Context += "/different-projection"
				publishFoundation()
			}
			err := verifyShellParserDependenciesThroughAdoption(root, foundation, a)
			pass := kind == "compatible" || kind == "exact without adoption"
			if (err == nil) != pass {
				t.Fatalf("kind=%s err=%v", kind, err)
			}
			if _, _, _, err := readPendingShellCorpusInputs(root, affectedPath); (err == nil) != pass {
				t.Fatalf("corpus consumer kind=%s err=%v", kind, err)
			}
			if kind == "projection drift" && !strings.Contains(err.Error(), "source/slot/protection drift") {
				t.Fatal("projection was not reverified", err)
			}
		})
	}
	if _, _, _, err := historicalStructuredGeneration(root, generation); err != nil {
		t.Fatal("historical Generation integrity", err)
	}
	b, err = readRegular(root, generation.Path)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, generation.Path, append(bytes.Clone(b), '\n'))
	defer writeFixture(t, root, generation.Path, b)
	if _, _, _, err := historicalStructuredGeneration(root, generation); err == nil {
		t.Fatal("historical Generation mutation accepted")
	}
}
