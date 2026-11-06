package sitecontent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
)

type tourAuditRecord struct {
	Owner, UnitID, UnitKind, SourceSHA, TargetSHA string
	Source, Target                                languageaudit.Document
}
type tourAuditMembership struct {
	Source                                               string
	Pages, Examples, ExampleInventory, ProtectedExamples int
	UIAssets, UIRecords, UIVisibleFields                 int
}
type tourAuditHit struct {
	Locale                               string `json:"locale"`
	Surface                              string `json:"surface"`
	Owner                                string `json:"owner"`
	UnitID                               string `json:"unit_id,omitempty"`
	UnitKind                             string `json:"unit_kind"`
	Class                                string `json:"class"`
	Location                             string `json:"location"`
	Start, End, VisibleStart, VisibleEnd int
	Reason                               string                 `json:"reason"`
	Classification                       languageaudit.Decision `json:"classification"`
}

// A full readonly language inventory, not a new currentness/authorization gate.
// Every TU is selected by complete workflow membership + ready canonical status;
// no staging, failed attempt, or raw candidate directory is enumerated.
func TestLanguageExhaustionTourReadonly(t *testing.T) {
	if os.Getenv("GO_TOUR_LANGUAGE_EXHAUSTION_READONLY") != "1" {
		t.Skip("explicit readonly current Tour probe")
	}
	start := time.Now()
	root := repositoryRoot(t)
	locale := "zh-CN"
	if _, _, err := i18n.CommentLocaleBaseLanguage(root, locale); err != nil {
		t.Fatal(err)
	}
	policy := auditFixturePolicy(locale, "Han") // exactly the Surface010 fixture profile
	glossary, err := i18n.LoadGlossary(root, locale)
	if err != nil {
		t.Fatal(err)
	}
	policy.ProtectedIdentities = append([]string(nil), glossary.Keep...)
	first, membership, err := currentTourAuditRecords(root, locale)
	if err != nil {
		t.Fatal(err)
	}
	second, again, err := currentTourAuditRecords(root, locale)
	if err != nil || !reflect.DeepEqual(first, second) || !reflect.DeepEqual(membership, again) {
		t.Fatal("repeated complete current projection differs", err)
	}
	scan := func(records []tourAuditRecord) []tourAuditHit {
		hits := []tourAuditHit{}
		for _, r := range records {
			result, err := languageaudit.Scan(policy, r.Source, r.Target)
			if err != nil {
				t.Fatal(r.Owner, r.UnitID, err)
			}
			for _, d := range result.Diagnostics {
				hits = append(hits, tourAuditHit{locale, "tour", r.Owner, r.UnitID, r.UnitKind, d.Class, d.Locator, d.Start, d.End, d.VisibleStart, d.VisibleEnd, fmt.Sprintf("%s; profile=%s; %s", d.Class, d.Profile, d.Evidence), d.Decision})
			}
		}
		return hits
	}
	hits := scan(first)
	if !reflect.DeepEqual(hits, scan(second)) {
		t.Fatal("repeated full scan differs")
	}
	counts := map[string]map[languageaudit.Decision]int{}
	owners := map[languageaudit.Decision]map[string]bool{languageaudit.Defect: {}, languageaudit.Candidate: {}}
	defects, candidates := 0, 0
	coverage := []map[string]any{}
	for _, r := range first {
		coverage = append(coverage, map[string]any{"owner": r.Owner, "unit_id": r.UnitID, "unit_kind": r.UnitKind, "source_sha256": r.SourceSHA, "target_sha256": r.TargetSHA, "visible_blocks": len(r.Target.Blocks)})
	}
	for _, h := range hits {
		if counts[h.Class] == nil {
			counts[h.Class] = map[languageaudit.Decision]int{}
		}
		counts[h.Class][h.Classification]++
		owners[h.Classification][h.Owner] = true
		if h.Classification == languageaudit.Defect {
			defects++
		} else {
			candidates++
		}
	}
	sortedOwners := map[languageaudit.Decision][]string{}
	for decision, names := range owners {
		for name := range names {
			sortedOwners[decision] = append(sortedOwners[decision], name)
		}
		sort.Strings(sortedOwners[decision])
	}
	report := map[string]any{"locale": locale, "surface": "tour", "membership": membership, "policy": policy, "scanned_assets": membership.Pages + membership.Examples + membership.UIAssets, "scanned_records": len(first), "deterministic_defects": defects, "review_candidates": candidates, "classes": counts, "owners": sortedOwners, "coverage": coverage, "hits": hits, "repeated_current_projection_and_scan_deterministic": true, "seconds": time.Since(start).Seconds(), "scope": "ready canonical Page/eligible Example + complete UI catalog/article headers/course descriptions; advisory profile diagnostics, no QC/current authorization/Surface judgment"}
	if out := os.Getenv("GO_TOUR_LANGUAGE_EXHAUSTION_EVIDENCE"); out != "" {
		if !filepath.IsAbs(out) || filepath.Dir(out) == "/tmp" || len(out) < 5 || out[:5] != "/tmp/" {
			t.Fatal("evidence must be a file in a /tmp subdirectory")
		}
		b, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(out, append(b, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("PAGES=%d EXAMPLES=%d UI_ASSETS=%d RECORDS=%d DETERMINISTIC_DEFECTS=%d REVIEW_CANDIDATES=%d REPEATED_SCAN_DETERMINISTIC=yes DURATION=%.2fs", membership.Pages, membership.Examples, membership.UIAssets, len(first), defects, candidates, time.Since(start).Seconds())
}

func currentTourAuditRecords(root, locale string) ([]tourAuditRecord, tourAuditMembership, error) {
	membership := tourAuditMembership{Source: "checked current source catalog + data/tour-pages.tsv + data/tour-examples.tsv + locales/" + locale + "/status.tsv (native workflow eligibility/ready canonical readers)"}
	c, err := checkedCatalog(root)
	if err != nil {
		return nil, membership, err
	}
	units, err := i18n.ReadTourLanguageAuditUnits(root, locale, c)
	if err != nil {
		return nil, membership, err
	}
	records := []tourAuditRecord{}
	pages := map[string]tourAuditRecord{}
	for _, item := range units {
		sv, tv, err := ProjectTourLanguageAudit(root, c, item.Unit, item.Owner, item.Target)
		if err != nil {
			return nil, membership, fmt.Errorf("%s: %w", item.Unit.ID, err)
		}
		r := tourAuditRecord{item.Owner, item.Unit.ID, string(item.Unit.Kind), item.Unit.SourceSHA256, digest(item.Target), sv, tv}
		records = append(records, r)
		switch item.Unit.Kind {
		case i18n.UnitKindPage:
			membership.Pages++
			pages[item.Unit.ID] = r
		case i18n.UnitKindExample:
			membership.Examples++
		default:
			return nil, membership, fmt.Errorf("unknown native TU kind")
		}
	}
	wantTotal, wantPages, wantExamples, err := i18n.LocaleWorkflowUnitCounts(c)
	if err != nil || wantTotal != len(units) || wantPages != membership.Pages || wantExamples != membership.Examples {
		return nil, membership, fmt.Errorf("workflow exact-set mismatch: %v", err)
	}
	membership.ExampleInventory = len(c.Examples)
	membership.ProtectedExamples = len(c.Examples) - membership.Examples
	// All non-eligible Examples are source-owned code, not omitted language TUs.
	for _, e := range c.Examples {
		if e.EligibleTranslation {
			continue
		}
		u, err := c.Unit(e.ID)
		if err != nil {
			return nil, membership, err
		}
		sv, tv, err := ProjectTourLanguageAudit(root, c, u, e.SourcePath, e.Source)
		if err != nil || len(sv.Blocks) != 0 || len(tv.Blocks) != 0 {
			return nil, membership, fmt.Errorf("non-eligible Example exposed language: %s: %v", e.ID, err)
		}
	}
	fs := os.DirFS(filepath.Join(root, "internal", "tour", "ui"))
	sourceUI, err := ui.LoadFromFS("en", fs)
	if err != nil {
		return nil, membership, err
	}
	targetUI, err := ui.LoadFromFS(locale, fs)
	if err != nil {
		return nil, membership, err
	}
	uiOwner := "internal/tour/ui/" + locale + ".json"
	keys := []string{}
	for key := range sourceUI.Messages {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		s, t := sourceUI.Messages[key], targetUI.Messages[key]
		sv, err := ProjectTourUIMessageAudit(uiOwner, key, s, s)
		if err != nil {
			return nil, membership, err
		}
		tv, err := ProjectTourUIMessageAudit(uiOwner, key, t, s)
		if err != nil {
			return nil, membership, err
		}
		records = append(records, tourAuditRecord{uiOwner, "", "ui-message", digest([]byte(s.Text)), digest([]byte(t.Text)), sv, tv})
	}
	membership.UIRecords = len(keys)
	membership.UIVisibleFields = len(keys)
	membership.UIAssets++
	sourceHeaders, err := i18n.ReadTourArticleAuditHeaders(root, c)
	if err != nil {
		return nil, membership, err
	}
	targetHeaders, err := i18n.LoadArticleMetadata(root, locale, c)
	if err != nil {
		return nil, membership, err
	}
	articles := []string{}
	for name := range sourceHeaders {
		articles = append(articles, name)
	}
	sort.Strings(articles)
	headerOwner := "locales/" + locale + "/article-metadata.json"
	for _, name := range articles {
		s, t := sourceHeaders[name], targetHeaders[name]
		sv, tv := languageaudit.Document{Owner: headerOwner}, languageaudit.Document{Owner: headerOwner}
		for _, v := range []struct{ field, source, target string }{{"title", s.Title, t.Title}, {"subtitle", s.Subtitle, t.Subtitle}} {
			sd, err := ProjectLanguageAuditText(headerOwner, "article/"+name+"/"+v.field, "metadata", v.source, nil)
			if err != nil {
				return nil, membership, err
			}
			td, err := ProjectLanguageAuditText(headerOwner, "article/"+name+"/"+v.field, "metadata", v.target, nil)
			if err != nil {
				return nil, membership, err
			}
			sv.Blocks = append(sv.Blocks, sd.Blocks...)
			tv.Blocks = append(tv.Blocks, td.Blocks...)
		}
		records = append(records, tourAuditRecord{headerOwner, "", "article-metadata", digest([]byte(s.Title + "\n" + s.Subtitle)), digest([]byte(t.Title + "\n" + t.Subtitle)), sv, tv})
	}
	membership.UIRecords += len(articles)
	membership.UIVisibleFields += 2 * len(articles)
	membership.UIAssets++
	metadata, err := i18n.LoadCourseMetadata(root, locale, c)
	if err != nil {
		return nil, membership, err
	}
	canonicalDescriptions := map[string]string{}
	if metadata.SchemaVersion == i18n.CourseMetadataSchemaVersionV2 {
		source, err := i18n.LoadCourseSourceDescriptions(root, c)
		if err != nil {
			return nil, membership, err
		}
		for _, p := range source.Pages {
			canonicalDescriptions[p.PageID] = p.Description
		}
	}
	courseOwner := "locales/" + locale + "/course-metadata.json"
	for _, p := range metadata.Pages {
		page, ok := pages[p.PageID]
		if !ok {
			return nil, membership, fmt.Errorf("SEO without canonical Page %s", p.PageID)
		}
		sv := page.Source
		sv.Owner = courseOwner
		sourceSHA := page.SourceSHA
		if metadata.SchemaVersion == i18n.CourseMetadataSchemaVersionV2 {
			source, ok := canonicalDescriptions[p.PageID]
			if !ok {
				return nil, membership, fmt.Errorf("missing v2 source description")
			}
			sv, err = ProjectLanguageAuditText(courseOwner, "course/"+p.PageID, "metadata", source, nil)
			if err != nil {
				return nil, membership, err
			}
			sourceSHA = digest([]byte(source))
		}
		tv, err := ProjectLanguageAuditText(courseOwner, "course/"+p.PageID, "metadata", p.Description, nil)
		if err != nil {
			return nil, membership, err
		}
		records = append(records, tourAuditRecord{courseOwner, p.PageID, "course-description", sourceSHA, digest([]byte(p.Description)), sv, tv})
	}
	membership.UIRecords += len(metadata.Pages)
	membership.UIVisibleFields += len(metadata.Pages)
	membership.UIAssets++
	return records, membership, nil
}
