package sitecontent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

type naturalRoleFixture struct {
	Path   string `json:"source_path"`
	Raw    string `json:"raw"`
	Source string `json:"source"`
	Old    Unit   `json:"old_slot"`
}

func naturalRoleFixtures(t *testing.T) []naturalRoleFixture {
	t.Helper()
	b, e := os.ReadFile("testdata/semantic-role-natural-language.json")
	if e != nil {
		t.Fatal(e)
	}
	var fixtures []naturalRoleFixture
	if e := json.Unmarshal(b, &fixtures); e != nil {
		t.Fatal(e)
	}
	if len(fixtures) != 16 {
		t.Fatal("exact incident set changed", len(fixtures))
	}
	return fixtures
}

func TestSemanticRoleNaturalLanguage(t *testing.T) {
	for _, word := range []string{"e.g.", "i.e.", "NOTE", "IMPORTANT", "and/or", "U.S.", "FAQ", "OTHER", "SomeHeading", "word/meaning", "a.b", "x_y", "time/space", "file/package", "key/value", "/", "//", "GET", "PATH", "-ing", "123"} {
		text, tokens, e := protectBlock([]byte("Read "+word+" here."), "markdown/Paragraph")
		if e != nil || len(tokens) != 0 || !strings.Contains(text, word) {
			t.Fatalf("prose spelling became identity: %s %v %+v", word, e, tokens)
		}
	}
	for n, f := range naturalRoleFixtures(t) {
		t.Run(f.Path+"/"+f.Raw+"/"+string(rune('a'+n)), func(t *testing.T) {
			d, raw := formalFixture(t, "x.md", f.Source+"\n")
			for _, s := range d.Slots {
				for _, p := range s.Protected {
					if p.Role == "atom" && p.Raw == f.Raw {
						t.Fatalf("incident remains frozen: %s", f.Raw)
					}
				}
			}
			if !strings.Contains(d.Units[0].Source, f.Raw) {
				t.Fatal("visible prose missing", f.Raw)
			}
			// This is transport evidence, never a translation selected by code.
			target := strings.Replace(d.Units[0].Source, f.Raw, "可编辑文本", 1)
			got := checkFormal(t, d, raw, target, true)
			if !bytes.Contains(got, []byte("可编辑文本")) {
				t.Fatal("mutation not represented")
			}
			old, e := restoreBlock(f.Old, f.Old.Source)
			if e != nil || old != f.Source {
				t.Fatal("old protected raw reinterpreted", e)
			}
			for _, p := range f.Old.Protected {
				if p.Raw == f.Raw {
					if _, e := restoreBlock(f.Old, strings.Replace(f.Old.Source, p.Token, "可编辑文本", 1)); e == nil {
						t.Fatal("old token exactness weakened")
					}
				}
			}
		})
	}
}

func TestSemanticRoleMachineControls(t *testing.T) {
	for _, tc := range []struct{ name, source, atom string }{
		{"identifier", "Use `CreateOrder` here.", "`CreateOrder`"},
		{"package", "Use encoding/json here.", "encoding/json"},
		{"api", "Use json.Marshal here.", "json.Marshal"},
		{"method", "When calling Reader.Open here.", "Reader.Open"},
		{"named-api", "Use function CreateOrder here.", "CreateOrder"},
		{"bare-api-authority", "Use ReadVarint here.", "ReadVarint"},
		{"receiver-api-authority", "Use Tx.Stmt here.", "Tx.Stmt"},
		{"registered-id", "Read GO-2022-0191 here.", "GO-2022-0191"},
		{"certificate-id", "Read Certificate A1234 here.", "A1234"},
		{"flag", "Use --race here.", "--race"},
		{"short-flag", "Use flag -r here.", "-r"},
		{"file", "Read main.go here.", "main.go"},
		{"path", "Read ./testdata/fuzz here.", "./testdata/fuzz"},
		{"url", "Read https://go.dev/doc here.", "https://go.dev/doc"},
		{"host", "Read pkg.go.dev here.", "pkg.go.dev"},
		{"literal", "Use `\"NOTE\"` here.", "`\"NOTE\"`"},
		{"keyword", "Use `select` here.", "`select`"},
		{"enum", "Use state RUNNING here.", "RUNNING"},
		{"track", "Use the PUBLIC track here.", "PUBLIC"},
		{"env", "Set environment variable CUSTOM_ROOT here.", "CUSTOM_ROOT"},
		{"known-env", "Set GOPATH here.", "GOPATH"},
		{"command", "Run `go test ./...` here.", "`go test ./...`"},
		{"version", "Use v1.2.3 here.", "v1.2.3"},
		{"go-release", "Use Go 1.25 here.", "1.25"},
		{"protocol", "Use HTTP here.", "HTTP"},
		{"protocol-method", "Use HTTP GET here.", "GET"},
		{"status", "Use HTTP 404 here.", "404"},
		{"acronym", "Use CLI here.", "CLI"},
		{"standard", "Use FIPS here.", "FIPS"},
		{"vulnerability-namespace", "Use CVE here.", "CVE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, raw := formalFixture(t, "x.md", tc.source+"\n")
			var locked *ProtectedToken
			for i := range d.Units[0].Protected {
				p := &d.Units[0].Protected[i]
				if p.Raw == tc.atom {
					locked = p
					break
				}
			}
			if locked == nil {
				t.Fatal("identity was released", tc.atom, d.Units[0])
			}
			checkFormal(t, d, raw, d.Units[0].Source, true)
			checkFormal(t, d, raw, strings.Replace(d.Units[0].Source, locked.Token, "modified", 1), false)
		})
	}
	if classifyProseToken("strange()=", "function ", "") != ambiguousFailClosed {
		t.Fatal("explicit ambiguous identity did not fail closed")
	}
	// A Go AST, not source spelling, establishes teaching identifiers.
	d, raw := formalFixture(t, "x.md", "```go\nfunc main() {\n // Use message with NOTE and/or FAQ.\n message := 1\n _ = message\n}\n```\n")
	if len(d.Slots) != 1 || !strings.Contains(d.Slots[0].Source, "NOTE and/or FAQ") {
		t.Fatal(d.Slots)
	}
	if len(d.Slots[0].Protected) != 1 || d.Slots[0].Protected[0].Raw != "message" {
		t.Fatal("AST identity lost", d.Slots)
	}
	checkFormal(t, d, raw, d.Units[0].Source, true)
}

func TestSemanticRoleShellPage(t *testing.T) {
	raw := []byte("---\ntitle: Example\nsummary: Example summary.\ntemplate: true\n---\n<p>NOTE</p><p>IMPORTANT and/or FAQ, e.g. U.S.</p><p>Use <code>NOTE</code> with CLI in state P0.</p>\n")
	page, e := ExtractShellPageRepresentation("_content/index.md", raw)
	if e != nil {
		t.Fatal(e)
	}
	for _, word := range []string{"NOTE", "IMPORTANT", "and/or", "FAQ", "e.g", "U.S"} {
		if !strings.Contains(page.Page.Units[0].Source, word) {
			t.Fatal("shell prose frozen", word)
		}
	}
	lockedCode := false
	for _, p := range page.Page.Units[0].Protected {
		if p.Raw == "<code>NOTE</code>" {
			lockedCode = true
		}
	}
	if !lockedCode {
		t.Fatal("explicit code identity released")
	}
	adapter := ShellPageAdapter{Source: *page, Raw: raw}
	target := strings.Replace(page.Page.Units[0].Source, "Use", "使用", 1)
	candidate, e := adapter.Restore(target)
	if e != nil {
		t.Fatal(e)
	}
	if e = adapter.Validate(candidate, &i18n.Glossary{}); e != nil {
		t.Fatal("mixed inline semantic identity binding", e)
	}
	if e = adapter.Validate(bytes.Replace(candidate, []byte("CLI"), []byte("修改"), 1), &i18n.Glossary{}); e == nil {
		t.Fatal("mixed inline machine identity mutation accepted")
	}
	if !standardGoPackages["encoding/json"] || standardGoPackages["time/space"] {
		t.Fatal("standard package authority is a spelling heuristic")
	}
}

// Read only the eleven affected historical selections. A frozen Snapshot owns
// its Document/candidate/validation; current export classification cannot alter
// their interpretation. No broad historical scan or new formal evidence.
func TestSemanticRoleHistoricalSelections(t *testing.T) {
	paths := map[string]bool{}
	for _, f := range naturalRoleFixtures(t) {
		paths[f.Path] = true
	}
	file, e := os.Open("testdata/protected-page-before.json.gz")
	if e != nil {
		t.Fatal(e)
	}
	defer file.Close()
	z, e := gzip.NewReader(file)
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	var pages []struct{ Document *Document }
	if e = json.NewDecoder(z).Decode(&pages); e != nil {
		t.Fatal(e)
	}
	root := repositoryRoot(t)
	manifest, e := os.ReadFile(filepath.Join(root, "data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-014/manifest.json"))
	if e != nil {
		t.Fatal(e)
	}
	var snapshot i18n.QualityCheckSnapshotManifest
	if e = json.Unmarshal(manifest, &snapshot); e != nil {
		t.Fatal(e)
	}
	count := 0
	probe := []map[string]any{}
	for _, p := range pages {
		if p.Document == nil || !paths[p.Document.Path] {
			continue
		}
		d := *p.Document
		source, e := restoreBlock(d.Units[0], d.Units[0].Source)
		if e != nil {
			t.Fatal(e)
		}
		a := PageAdapter{d, []byte(source)}
		matched := false
		for _, s := range snapshot.Units {
			if s.UnitID != d.Units[0].ID {
				continue
			}
			matched = true
			validation, e := os.ReadFile(filepath.Join(root, s.ValidationPath))
			if e != nil || digest(validation) != s.ValidationSHA256 {
				t.Fatal("frozen validation drift", e)
			}
			var v struct {
				RawResponsePath string `json:"raw_response_path"`
				Attempt         int    `json:"attempt"`
			}
			if e = json.Unmarshal(validation, &v); e != nil || v.Attempt != s.Attempt {
				t.Fatal(e)
			}
			raw, e := os.ReadFile(filepath.Join(root, "data/retranslation-runs/zh-CN", s.SelectedBatchID, v.RawResponsePath))
			if e != nil {
				t.Fatal(e)
			}
			candidate, e := os.ReadFile(filepath.Join(root, s.CandidatePath))
			if e != nil || digest(candidate) != s.CandidateSHA256 {
				t.Fatal(e)
			}
			got, e := a.Restore(string(raw))
			if e != nil || !bytes.Equal(append(bytes.TrimRight(got, "\n"), '\n'), candidate) {
				t.Fatal("historical interpretation changed", d.Path, e)
			}
			next, e := ParseWorkflowDocument(documentSource(d), a.Source, d.StableIndex)
			if e != nil {
				t.Fatal(e)
			}
			current := PageAdapter{*next, a.Source}
			baseline, e := current.RebaseRevisionPredecessor(a, raw)
			row := map[string]any{"source_path": d.Path, "snapshot_unit_id": s.UnitID, "current_unit_id": next.Units[0].ID, "historical_raw_sha256": digest(raw), "candidate_sha256": s.CandidateSHA256, "historical_interpretation": "PASS", "byte_exact_rebase": e == nil}
			if e != nil {
				// Newly protected HTTP status identities were prose in old raw.
				// That raw retains its meaning; automatic rebase must fail closed.
				if !strings.Contains(e.Error(), "outside historical source atoms") {
					t.Fatal(d.Path, e)
				}
				t.Logf("explicit reconciliation required (no inferred target mapping): %s: %v", d.Path, e)
				row["rebase_blocker"] = e.Error()
				op, _ := protectionPieces(d.Units[0])
				np, _ := protectionPieces(next.Units[0])
				introduced := []string{}
				for _, q := range np {
					if q.token == nil {
						continue
					}
					inside := false
					for _, p := range op {
						if p.token != nil && q.start >= p.start && q.end <= p.end {
							inside = true
						}
					}
					if !inside {
						introduced = append(introduced, q.token.Raw)
					}
				}
				row["introduced_machine_identity"] = introduced
			} else {
				after, e := current.Restore(string(baseline))
				if e != nil || !bytes.Equal(got, after) {
					t.Fatal("byte-exact rebase failed", d.Path, e)
				}
			}
			probe = append(probe, row)
			count++
		}
		if !matched {
			t.Fatal("exact historical selection missing", d.Path)
		}
	}
	if count != 11 {
		t.Fatal(count)
	}
	if output := os.Getenv("GO_TOUR_SYS08_HISTORICAL_PROBE"); output != "" {
		b, e := json.MarshalIndent(probe, "", "  ")
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(output, append(b, '\n'), 0600); e != nil {
			t.Fatal(e)
		}
	}
	t.Logf("OLD_REPRESENTATION_COMPATIBILITY=PASS affected historical raw/candidate/Snapshot=11")
}
