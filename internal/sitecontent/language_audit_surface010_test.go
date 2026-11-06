package sitecontent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
)

// This probe consumes immutable recorded fixtures, not a new Surface judgment.
// No currentness workflow, exporter, Bundle, or lifecycle mutation is invoked.
func TestLanguageExhaustionSurface010Readonly(t *testing.T) {
	if os.Getenv("GO_TOUR_LANGUAGE_EXHAUSTION_READONLY") != "1" {
		t.Skip("explicit readonly recorded-fixture probe")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	ref := Reference{"data/locale-language/zh-CN/surface/20261103-zh-CN-integrated-surface-010.json", "4d2168b72e193245cf993f5c03cfbb7aebf2f022b675847b549924b3abab9042"}
	var receipt IntegratedSurfaceReceipt
	if err := readReference(root, ref, &receipt); err != nil {
		t.Fatal(err)
	}
	copy := receipt
	copy.Identity = ""
	sc := receipt.Scope
	sc.Identity = ""
	if receipt.Identity != identity(copy) || receipt.Scope.Identity != identity(sc) || receipt.Decision != "failed" || len(receipt.ExactFindings) != 8 {
		t.Fatal("recorded fixture identity/exact set")
	}
	if _, err := readRegularReference(root, receipt.Evidence); err != nil {
		t.Fatal(err)
	}
	locale := receipt.Scope.Locale
	base, _, err := i18n.CommentLocaleBaseLanguage(root, locale)
	if err != nil || base != "zh" {
		t.Fatal("fixture locale metadata", err)
	}
	// The selected Han/Latin audit profile is explicit test data. Locale metadata
	// alone does not establish punctuation or spacing policy for any language.
	policy := auditFixturePolicy(locale, "Han")
	glossary, err := i18n.LoadGlossary(root, locale)
	if err != nil {
		t.Fatal(err)
	}
	policy.ProtectedIdentities = append([]string(nil), glossary.Keep...)
	type assetReport struct {
		Owner                string
		FindingClass         string
		SourceSHA, TargetSHA string
		Diagnostics          []languageaudit.Diagnostic
	}
	currentSources := map[string][]byte{}
	err = withReadonlyStructuredValidation(root, func(ctx *structuredSuccessorValidation) error {
		g, err := LoadCurrent(root, ctx)
		if err != nil {
			return err
		}
		seen := map[string]bool{}
		for _, ref := range receipt.Scope.Closures {
			var c PackageClosure
			if err := readReference(root, ref, &c); err != nil {
				return err
			}
			if seen[c.Package] {
				continue
			}
			seen[c.Package] = true
			_, raw, err := CurrentPackageDocuments(root, g, c.Package, ctx)
			if err != nil {
				return err
			}
			for name, value := range raw {
				if old, ok := currentSources[name]; ok && !reflect.DeepEqual(old, value) {
					return fmt.Errorf("ambiguous current source %s", name)
				}
				currentSources[name] = value
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal("current source graph", err)
	}
	frozen := newClosureValidation(root)
	reports := []assetReport{}
	classes := map[string]map[string]int{}
	det, candidates := 0, 0
	viewByOwner := map[string]languageaudit.Result{}
	for _, finding := range receipt.ExactFindings {
		source, target, err := languageAuditRecordedAsset(root, &receipt, finding, currentSources, frozen)
		if err != nil {
			t.Fatal(finding.ID, err)
		}
		syntax := AuditHTML
		switch filepath.Ext(finding.ID) {
		case ".md":
			syntax = AuditMarkdown
		case ".tmpl":
			syntax = AuditTemplate
		case ".yaml":
			syntax = AuditShellYAML
		case ".html":
		default:
			t.Fatal("unsupported recorded fixture syntax", finding.ID)
		}
		sv, err := ProjectLanguageAudit(finding.ID, source, syntax)
		if err != nil {
			t.Fatal("source projection", finding.ID, err)
		}
		tv, err := ProjectLanguageAudit(finding.ID, target, syntax)
		if err != nil {
			t.Fatal("target projection", finding.ID, err)
		}
		if out := os.Getenv("GO_TOUR_LANGUAGE_EXHAUSTION_EVIDENCE"); out != "" && strings.HasPrefix(filepath.Clean(out), "/tmp/") {
			b, _ := json.MarshalIndent(tv, "", "  ")
			if err := os.WriteFile(out+"."+digest([]byte(finding.ID))+".view.json", b, 0600); err != nil {
				t.Fatal(err)
			}
		}
		result, err := languageaudit.Scan(policy, sv, tv)
		if err != nil {
			t.Fatal(finding.ID, err)
		}
		again, err := languageaudit.Scan(policy, sv, tv)
		if err != nil || !reflect.DeepEqual(result, again) {
			t.Fatal("scan determinism", finding.ID, err)
		}
		viewByOwner[finding.ID] = result
		for _, d := range result.Diagnostics {
			if classes[d.Class] == nil {
				classes[d.Class] = map[string]int{}
			}
			classes[d.Class][string(d.Decision)]++
			if d.Decision == languageaudit.Defect {
				det++
			} else {
				candidates++
			}
		}
		reports = append(reports, assetReport{finding.ID, finding.Class, digest(source), digest(target), result.Diagnostics})
	}
	// Only this real regression table names recorded owners. The scanner/profile
	// have no paths, findings, Chinese target strings or location exemptions.
	expected := map[string]map[string]int{
		"_content/doc/database/cancel-operations.md": {"script-adjacency": 3},
		"_content/doc/index.html":                    {"script-adjacency": 2},
		"_content/doc/security/vuln/database.md":     {"script-adjacency": 2},
		"_content/doc/tutorial/add-a-test.html":      {"clause-punctuation": 1},
		"_content/site.tmpl":                         {"sentence-punctuation": 2},
		"data/site-shell/translation.html":           {"script-adjacency": 5},
		"_content/testimonials.yaml":                 {"sentence-punctuation": 1},
	}
	failures := []string{}
	for owner, cs := range expected {
		for class, want := range cs {
			got := auditClassCount(viewByOwner[owner], class, languageaudit.Defect)
			if got != want {
				failures = append(failures, fmt.Sprintf("%s %s got=%d want=%d", owner, class, got, want))
			}
		}
	}
	if auditClassCount(viewByOwner["_content/doc/database/cancel-operations.md"], "paragraph-link-split", languageaudit.Candidate) != 1 {
		failures = append(failures, "recorded paragraph split candidate missing")
	}
	containsCandidate := func(owner, needle string) bool {
		for _, d := range viewByOwner[owner].Diagnostics {
			if d.Decision == languageaudit.Candidate && strings.Contains(d.Evidence, needle) {
				return true
			}
		}
		return false
	}
	if !containsCandidate("_content/doc/install.html", "Command Prompt") {
		failures = append(failures, "Command Prompt review candidate missing")
	}
	if !containsCandidate("data/site-shell/translation.html", "issue") {
		failures = append(failures, "issue review candidate missing")
	}
	if !containsCandidate("_content/site.tmpl", "Programming Language") {
		failures = append(failures, "social metadata review candidate missing")
	}
	if auditClassCount(viewByOwner["data/site-shell/translation.html"], "protected-prose-symbol", languageaudit.Candidate) != 2 {
		failures = append(failures, "protected heading symbols missing")
	}
	sort.Strings(failures)
	// Recorded finding coverage is overlapping: three owners have both
	// machine-provable profile violations and reviewer-only semantic shapes.
	report := map[string]any{"scanned_assets": len(reports), "fixture_count": 8, "fixture_machine_covered": len(expected), "fixture_review_candidate_covered": 4, "fixture_unsupported": 0, "deterministic_defects": det, "review_candidates": candidates, "classes": classes, "assets": reports, "fixture_failures": failures, "profile": policy, "scope": "readonly recorded target bytes; parser-derived visible prose and social metadata; literal code/attributes/template actions excluded; not a formal currentness or language judgment"}
	out := os.Getenv("GO_TOUR_LANGUAGE_EXHAUSTION_EVIDENCE")
	if out != "" {
		if !strings.HasPrefix(filepath.Clean(out), "/tmp/") {
			t.Fatal("evidence must remain under /tmp")
		}
		b, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(out, append(b, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if len(failures) > 0 {
		t.Fatal(failures)
	}
	t.Logf("SCANNED_ASSETS=%d DETERMINISTIC_DEFECTS=%d REVIEW_CANDIDATES=%d REPEATED_SCAN_DETERMINISTIC=yes", len(reports), det, candidates)
}

func languageAuditRecordedAsset(root string, r *IntegratedSurfaceReceipt, f SurfaceFinding, sources map[string][]byte, frozen *closureValidation) ([]byte, []byte, error) {
	source, ok := sources[f.ID]
	if !ok {
		return nil, nil, fmt.Errorf("current source absent from canonical package parser")
	}
	if f.Class == i18n.SurfaceTranslationUnitLanguage {
		if len(f.Artifacts) != 1 || f.Artifacts[0].Unit.SourcePath != f.ID || digest(source) != f.Artifacts[0].Unit.SourceSHA256 {
			return nil, nil, fmt.Errorf("exact TU fixture/source mismatch")
		}
		u := f.Artifacts[0].Unit
		target, err := readRegularReference(root, Reference{u.CandidatePath, u.CandidateSHA256})
		return source, target, err
	}
	if f.Class != "structured-surface-language" {
		return nil, nil, fmt.Errorf("unsupported fixture finding class")
	}
	var asset *SurfaceAsset
	for _, list := range [][]SurfaceAsset{r.Scope.Reviewed, r.Scope.Carried} {
		for _, a := range list {
			if a.ID == f.ID {
				if asset != nil {
					return nil, nil, fmt.Errorf("ambiguous recorded owner")
				}
				x := a
				asset = &x
			}
		}
	}
	if asset == nil {
		return nil, nil, fmt.Errorf("missing recorded owner")
	}
	for _, ref := range r.Scope.Closures {
		var c PackageClosure
		if err := readReference(root, ref, &c); err != nil {
			return nil, nil, err
		}
		if c.Package != asset.Package {
			continue
		}
		baseline, err := frozen.historicalStructuredClosure(root, ref, map[string]bool{})
		if err != nil {
			return nil, nil, err
		}
		owner, ok := baseline.Owners[f.ID]
		if !ok {
			return nil, nil, fmt.Errorf("structured fixture has no frozen owner")
		}
		generation := baseline.Generations[owner.Path]
		if generation == nil {
			return nil, nil, fmt.Errorf("frozen Generation missing")
		}
		matched := false
		for _, a := range generation.Plan.Assets {
			if a.Package == asset.Package && a.Document != nil && a.Document.Path == f.ID && a.Document.SourceSHA == digest(source) {
				matched = true
			}
		}
		if !matched || !reflect.DeepEqual(source, baseline.Inputs[owner.Path]["source/"+f.ID]) {
			return nil, nil, fmt.Errorf("recorded/current raw source mismatch")
		}
		target, ok := baseline.Files[f.ID]
		if !ok || digest(target) != asset.TargetSHA {
			return nil, nil, fmt.Errorf("recorded structured target SHA mismatch")
		}
		return source, target, nil

	}
	return nil, nil, fmt.Errorf("exact structured output not found")
}
