package i18n

import (
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
	"github.com/shuijingwan/go-tour-i18n/internal/languageview"
	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
	"sort"
)

type LanguageReviewExample struct {
	UnitID string `json:"unit_id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

func addTourSurfaceLanguageAttention(root, locale string, c *Catalog, pkg *LocaleSurfaceReviewPackage, statuses map[string]*Status) error {
	attention, err := NewLanguageAttention(root, locale)
	if err != nil {
		return err
	}
	glossary, err := LoadGlossary(root, locale)
	if err != nil {
		return err
	}
	policy, _, err := LoadLanguageAuditPolicy(root, locale, glossary)
	if err != nil {
		return err
	}
	pair := func(owner, locator, source, target string, rich bool) error {
		s, t := languageaudit.Document{Owner: owner}, languageaudit.Document{Owner: owner}
		if rich {
			s.Blocks, err = languageview.HTML([]byte(source), languageview.Options{})
			if err != nil {
				return err
			}
			t.Blocks, err = languageview.HTML([]byte(target), languageview.Options{})
			if err != nil {
				return err
			}
		} else {
			s.Blocks = []languageaudit.Block{{Locator: locator, Context: "metadata", Start: -1, End: -1, Pieces: languageview.Plain(source)}}
			t.Blocks = []languageaudit.Block{{Locator: locator, Context: "metadata", Start: -1, End: -1, Pieces: languageview.Plain(target)}}
		}
		for i := range s.Blocks {
			s.Blocks[i].Locator = locator + "/" + s.Blocks[i].Locator
		}
		for i := range t.Blocks {
			t.Blocks[i].Locator = locator + "/" + t.Blocks[i].Locator
		}
		r, err := languageaudit.Scan(policy, s, t)
		if err != nil {
			return err
		}
		return attention.Add("tour-context", owner, "", "", r)
	}
	workflow, _, _, err := localeWorkflowUnits(c)
	if err != nil {
		return err
	}
	ids := []string{}
	for id := range workflow {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		u := workflow[id]
		target, err := loadReadyTranslationUnitCandidate(root, c, u, locale, statuses[id])
		if err != nil {
			return err
		}
		r, err := TranslationUnitLanguageAudit(root, c, u, locale, target, glossary)
		if err != nil {
			return err
		}
		if err := attention.Add("tour", statuses[id].CandidatePath, id, u.Kind, r); err != nil {
			return err
		}
		if u.Kind == UnitKindExample {
			pkg.Examples = append(pkg.Examples, LanguageReviewExample{id, string(u.Source), string(target)})
		}
	}
	for _, m := range pkg.UI {
		owner := "internal/tour/ui/" + locale + ".json"
		source := ui.Message{Kind: m.Kind, Text: m.Source}
		target := ui.Message{Kind: m.Kind, Text: m.Target}
		sv, e := ProjectTourUIMessageAudit(owner, m.Key, source, source)
		if e != nil {
			return e
		}
		tv, e := ProjectTourUIMessageAudit(owner, m.Key, target, source)
		if e != nil {
			return e
		}
		r, e := languageaudit.Scan(policy, sv, tv)
		if e != nil {
			return e
		}
		if e := attention.Add("tour-ui", owner, "", "", r); e != nil {
			return e
		}
	}
	for _, a := range pkg.Articles {
		if err := pair("locales/"+locale+"/article-metadata.json", a.Article+"/title", a.SourceTitle, a.TargetTitle, false); err != nil {
			return err
		}
		if err := pair("locales/"+locale+"/article-metadata.json", a.Article+"/subtitle", a.SourceSubtitle, a.TargetSubtitle, false); err != nil {
			return err
		}
	}
	for _, p := range pkg.CoursePages {
		source := p.CanonicalDescription
		if source == "" {
			source = p.Source
		}
		if err := pair("locales/"+locale+"/course-metadata.json", p.PageID, source, p.Description, false); err != nil {
			return err
		}
	}
	if err := attention.Verify(root); err != nil {
		return err
	}
	pkg.LanguageAttention = &attention
	if len(pkg.Examples)+len(pkg.CoursePages) != pkg.Coverage.TranslationUnits {
		return fmt.Errorf("Tour review full source/target coverage mismatch")
	}
	return nil
}
