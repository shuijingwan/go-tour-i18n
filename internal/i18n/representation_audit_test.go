package i18n

import (
	"encoding/json"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTourPageRepresentationAudit(t *testing.T) {
	output := os.Getenv("GO_TOUR_TOUR_REPRESENTATION_AUDIT")
	if output == "" {
		t.Skip("explicit diagnostic output required")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	c, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	type page struct {
		ID, Path, Route string
		Counts          map[string]int
		Failures        []string
	}
	pages := []page{}
	codeCounts := func(chunks []string) map[string]int {
		counts := map[string]int{}
		for _, chunk := range chunks {
			a := analyzePreformattedGo(chunk)
			prefix := "code/go-safe/"
			if a.Static {
				counts["code/static-opaque"]++
				prefix = "code/static-lexer/"
			} else {
				counts["code/go-safe-block"]++
			}
			for _, c := range a.Comments {
				if !a.Static && c.Translatable {
					counts["code/teaching-body"]++
				}
			}
			fset := token.NewFileSet()
			file := fset.AddFile("audit.go", -1, len(chunk))
			var scan scanner.Scanner
			scan.Init(file, []byte(chunk), func(token.Position, string) {}, scanner.ScanComments)
			for {
				_, tok, literal := scan.Scan()
				if tok == token.EOF {
					break
				}
				if tok == token.COMMENT {
					if strings.HasPrefix(literal, "//") {
						counts[prefix+"line-comment"]++
					} else {
						counts[prefix+"block-comment"]++
					}
				}
				if tok == token.STRING {
					if strings.HasPrefix(literal, "`") {
						counts[prefix+"raw-string"]++
					} else {
						counts[prefix+"string"]++
					}
				}
			}
		}
		return counts
	}
	g := &Glossary{Mandatory: map[string]string{}}
	for _, p := range c.Pages {
		s, err := structuralSignature(p.Source)
		if err != nil {
			t.Fatal(err)
		}
		v := protectTranslation(p.Source, p.SourceSHA256, g)
		restored, failures := v.restore(v.Text)
		if err := validateCandidateWithGlossary(root, c, p.ID, "zh-CN", []byte(restored), g); err != nil {
			failures = append(failures, err.Error())
		}
		counts := map[string]int{"present/inline-code": len(s.InlineCode), "present/link": len(s.Links), "present/directive": len(s.Directives), "present/preformatted": len(s.Preformatted)}
		for _, b := range preformattedBlocks(string(p.Source)) {
			for k, v := range codeCounts([]string{b.Text}) {
				counts[k] += v
			}
			a := analyzePreformattedGo(b.Text)
			if a.Static {
				counts["present/static-code"]++
			} else {
				counts["present/executable-go"]++
			}
			for _, c := range a.Comments {
				if !a.Static && c.Translatable {
					counts["present/teaching-comment"]++
				}
			}
		}
		for _, line := range strings.Split(string(p.Source), "\n") {
			if strings.HasPrefix(line, "- ") {
				counts["present/list-item"]++
			}
		}
		pages = append(pages, page{p.ID, "_content/tour/" + p.Article, "/tour" + p.Route, counts, failures})
	}
	if input := os.Getenv("GO_TOUR_PAGE_CODE_AUDIT_INPUT"); input != "" {
		b, e := os.ReadFile(input)
		if e != nil {
			t.Fatal(e)
		}
		var sourcePages []struct {
			ID, Path, Route string
			Code            []string
		}
		if e = json.Unmarshal(b, &sourcePages); e != nil {
			t.Fatal(e)
		}
		for _, p := range sourcePages {
			if len(p.Code) > 0 {
				pages = append(pages, page{p.ID, p.Path, p.Route, codeCounts(p.Code), nil})
			}
		}
	}
	b, err := json.MarshalIndent(pages, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(output, append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	for _, p := range pages {
		if len(p.Failures) > 0 {
			t.Errorf("%s: %v", p.ID, p.Failures)
		}
	}
	t.Logf("Tour Pages=%d diagnostic source records=%d", len(c.Pages), len(pages))
}
