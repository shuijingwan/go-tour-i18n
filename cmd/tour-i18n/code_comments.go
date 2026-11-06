package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/site"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
	"github.com/shuijingwan/go-tour-i18n/internal/tour"
)

type commentInventoryUnit struct {
	StableIndex int                     `json:"stable_index"`
	Package     string                  `json:"package"`
	Kind        i18n.UnitKind           `json:"parent_kind"`
	ID          string                  `json:"parent_unit_id"`
	Route       string                  `json:"route"`
	Path        string                  `json:"source_path"`
	SourceSHA   string                  `json:"source_sha256"`
	Blocks      []i18n.CommentCodeBlock `json:"blocks"`
}
type commentCounts struct {
	Blocks            int `json:"code_blocks"`
	Slots             int `json:"comment_slots"`
	Teaching          int `json:"teaching_comments"`
	Protected         int `json:"protected_comments"`
	Unclassified      int `json:"unclassified_comments"`
	Exact             int `json:"exact_source_carry"`
	Missing           int `json:"missing"`
	Extra             int `json:"extra"`
	ProtectedMismatch int `json:"protected_mismatch"`
	StructureMismatch int `json:"structure_mismatch"`
}
type commentAuditRow struct {
	Package string        `json:"package"`
	Locale  string        `json:"locale"`
	Kind    i18n.UnitKind `json:"parent_kind"`
	ID      string        `json:"parent_unit_id"`
	Route   string        `json:"route"`
	Path    string        `json:"source_path"`
	i18n.CommentSlotComparison
}
type commentPackageAudit struct {
	Package     string                          `json:"package"`
	State       string                          `json:"state"`
	Error       string                          `json:"error,omitempty"`
	Authority   []i18n.GlossaryArchiveReference `json:"current_authority"`
	Targets     []i18n.AcceptedCommentTarget    `json:"accepted_targets"`
	Page        commentCounts                   `json:"page"`
	Example     commentCounts                   `json:"example"`
	Rows        []commentAuditRow               `json:"slots"`
	Diagnostics []string                        `json:"diagnostics"`
	ReviewUnits []string                        `json:"review_units"`
}
type commentLocaleAudit struct {
	Locale   string                `json:"locale"`
	Base     string                `json:"base_language"`
	Packages []commentPackageAudit `json:"packages"`
}

func commentBlocks(u commentInventoryUnit, b []byte) ([]i18n.CommentCodeBlock, error) {
	if u.Package != "tour-v1" {
		return sitecontent.PageCommentBlocks(u.Path, b)
	}
	if u.Kind == i18n.UnitKindPage {
		return i18n.TourPageCommentBlocks(b), nil
	}
	return []i18n.CommentCodeBlock{i18n.AnalyzeCommentCode(string(b), "go", "example/1", true)}, nil
}

func commentSourceInventory(c *i18n.Catalog) ([]commentInventoryUnit, error) {
	out := []commentInventoryUnit{}
	add := func(pkg string, u *i18n.TranslationUnit, route string) error {
		v := commentInventoryUnit{Package: pkg, Kind: u.Kind, ID: u.ID, Route: route, Path: u.SourcePath, SourceSHA: u.SourceSHA256}
		if pkg == "tour-v1" {
			v.StableIndex = len(out) + 1
		} else {
			for _, a := range c.Adapted {
				if a.Unit.ID == u.ID {
					v.StableIndex = a.StableIndex
					break
				}
			}
		}
		b, err := commentBlocks(v, u.Source)
		if err != nil {
			return err
		}
		v.Blocks = b
		for i := range v.Blocks {
			for j := range v.Blocks[i].Comments {
				v.Blocks[i].Comments[j].SlotID = i18n.CommentSlotIdentity(pkg+"/"+v.Path+"/"+string(v.Kind)+"/"+v.ID, v.Blocks[i], v.Blocks[i].Comments[j])
			}
		}
		out = append(out, v)
		return nil
	}
	for _, p := range c.Pages {
		u, err := c.Unit(p.ID)
		if err != nil {
			return nil, err
		}
		if err = add("tour-v1", u, "/tour"+p.Route); err != nil {
			return nil, err
		}
	}
	for _, e := range c.Examples {
		if !e.EligibleTranslation {
			continue
		}
		u, err := c.Unit(e.ID)
		if err != nil {
			return nil, err
		}
		route := ""
		if len(e.ReferencedBy) > 0 {
			p, err := c.Page(e.ReferencedBy[0])
			if err != nil {
				return nil, err
			}
			route = "/tour" + p.Route
		}
		if err = add("tour-v1", u, route); err != nil {
			return nil, err
		}
	}
	for _, d := range c.TranslationPackages {
		if d.ID == "learn-docs-v1" {
			for _, a := range d.Units {
				u := a.Unit
				if err := add(d.ID, &u, a.Route); err != nil {
					return nil, err
				}
			}
		}
	}
	return out, nil
}

func commentRoadmapLocales(root string) ([]string, error) {
	b, err := os.ReadFile(filepath.Join(root, "docs/LOCALE_ROADMAP.md"))
	if err != nil {
		return nil, err
	}
	registry := map[string]bool{}
	for _, l := range tour.LanguageRegistry() {
		if !l.Official {
			registry[l.Locale] = true
		}
	}
	re := regexp.MustCompile(`(?m)^\| [0-9]+ \| ([A-Za-z0-9-]+) \|`)
	out := []string{}
	seen := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(string(b), -1) {
		if m[1] == "en-US" {
			continue
		}
		if !registry[m[1]] || seen[m[1]] {
			return nil, fmt.Errorf("roadmap/registry mismatch: %s", m[1])
		}
		seen[m[1]] = true
		out = append(out, m[1])
	}
	if len(out) != len(registry) {
		return nil, fmt.Errorf("roadmap exact set incomplete")
	}
	return out, nil
}

func auditCommentPackage(root string, c *i18n.Catalog, locale, base, pkg string, inventory []commentInventoryUnit) commentPackageAudit {
	p := commentPackageAudit{Package: pkg, State: "NOT_ACCEPTED", Authority: []i18n.GlossaryArchiveReference{}, Targets: []i18n.AcceptedCommentTarget{}, Rows: []commentAuditRow{}, Diagnostics: []string{}, ReviewUnits: []string{}}
	// Completion discovers scope only. Existing authorization readers below
	// prove current accepted bytes independently, including candidate freshness.
	b, err := os.ReadFile(filepath.Join(root, sitecontent.LocalePath(locale)))
	if os.IsNotExist(err) {
		return p
	}
	if err != nil {
		p.State = "CURRENT_TARGET_UNAVAILABLE"
		p.Error = err.Error()
		return p
	}
	var l sitecontent.Locale
	if err = sitecontent.StrictJSON(b, &l); err != nil {
		p.State = "CURRENT_TARGET_UNAVAILABLE"
		p.Error = err.Error()
		return p
	}
	if l.Locale != locale {
		p.State = "CURRENT_TARGET_UNAVAILABLE"
		p.Error = "locale authority identity mismatch"
		return p
	}
	found := false
	for _, v := range l.Packages {
		if v.Package == pkg && v.State == "complete" {
			found = true
		}
	}
	if !found {
		return p
	}
	copy := *c
	if pkg != "tour-v1" {
		if err = copy.UseTranslationPackage(pkg); err != nil {
			p.State = "CURRENT_TARGET_UNAVAILABLE"
			p.Error = err.Error()
			return p
		}
	}
	targets, refs, err := i18n.CurrentAcceptedCommentTargets(root, &copy, locale)
	if err != nil {
		p.State = "CURRENT_TARGET_UNAVAILABLE"
		p.Error = err.Error()
		return p
	}
	p.State = "CURRENT"
	p.Authority = refs
	p.Targets = targets
	byID := map[string]i18n.AcceptedCommentTarget{}
	for _, t := range targets {
		byID[t.UnitID] = t
	}
	for _, u := range inventory {
		if u.Package != pkg {
			continue
		}
		t, ok := byID[u.ID]
		if !ok {
			p.State = "CURRENT_TARGET_UNAVAILABLE"
			p.Error = "accepted exact set missing " + u.ID
			return p
		}
		blocks, err := commentBlocks(u, t.Bytes)
		if err != nil {
			p.State = "CURRENT_TARGET_UNAVAILABLE"
			p.Error = err.Error()
			return p
		}
		blocks = i18n.SourceSelectedTargetCommentBlocks(u.Blocks, blocks)
		n := &p.Page
		if u.Kind == i18n.UnitKindExample {
			n = &p.Example
		}
		n.Blocks += len(u.Blocks)
		rows, diagnostics := i18n.CompareCommentBlocks(pkg+"/"+u.Path+"/"+string(u.Kind)+"/"+u.ID, base, u.Blocks, blocks)
		for _, d := range diagnostics {
			p.Diagnostics = append(p.Diagnostics, u.ID+": "+d)
		}
		review := len(diagnostics) > 0
		for _, r := range rows {
			p.Rows = append(p.Rows, commentAuditRow{pkg, locale, u.Kind, u.ID, u.Route, u.Path, r})
			n := &p.Page
			if u.Kind == i18n.UnitKindExample {
				n = &p.Example
			}
			n.Slots++
			if r.SemanticKind == "unclassified" {
				n.Unclassified++
			} else if r.ProtectedSemantics {
				n.Protected++
			}
			if r.SemanticKind == "teaching-comment" {
				n.Teaching++
				if r.ExactSourceCarry {
					n.Exact++
				}
			}
			if r.Missing {
				n.Missing++
			}
			if r.Extra {
				n.Extra++
			}
			if !r.ProtectedUnchanged && r.ProtectedSemantics {
				n.ProtectedMismatch++
			}
			if !r.StructureMatch {
				n.StructureMismatch++
			}
			if r.ReviewRequired || r.EnglishResidueSuspect || r.Missing || r.Extra || !r.StructureMatch || !r.ProtectedUnchanged {
				review = true
			}
		}
		if review {
			p.ReviewUnits = append(p.ReviewUnits, string(u.Kind)+":"+u.ID)
		}
	}
	return p
}

func codeCommentsCommand(root string, args []string) error {
	if len(args) > 0 && (args[0] == "unsupported" || args[0] == "coverage") {
		return commentCoverageCommand(root, args)
	}
	if len(args) == 0 || args[0] != "audit" {
		return fmt.Errorf("usage: code-comments <unsupported|coverage|audit> --output-dir <directory>")
	}
	fs := flag.NewFlagSet("code-comments audit", flag.ContinueOnError)
	output := fs.String("output-dir", "/tmp", "diagnostic evidence directory")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	c, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		return err
	}
	if err = sitecontent.AttachRegisteredTranslationPackages(root, c); err != nil {
		return err
	}
	sitecontent.AttachIntegratedSurfaceAuthority(c)
	site.AttachPreviewFailureAuthority(c)
	inventory, err := commentSourceInventory(c)
	if err != nil {
		return err
	}
	inventory, _, err = closeCommentInventory(c, inventory)
	if err != nil {
		return err
	}
	if err := requireCompleteCommentInventory(inventory); err != nil {
		return err
	}
	if err := writeCommentFrontierPreflight(root, func(string, any) error { return nil }); err != nil {
		return err
	}
	locales, err := commentRoadmapLocales(root)
	if err != nil {
		return err
	}
	counts := map[string]commentCounts{}
	languages := map[string]int{}
	unsupported := map[string]int{}
	for _, u := range inventory {
		k := u.Package + "/" + string(u.Kind)
		n := counts[k]
		for _, b := range u.Blocks {
			n.Blocks++
			languages[b.Language]++
			if b.Unsupported {
				unsupported[b.Language]++
			}
			for _, s := range b.Comments {
				n.Slots++
				switch s.Kind {
				case "teaching-comment":
					n.Teaching++
				case "unclassified":
					n.Unclassified++
				default:
					n.Protected++
				}
			}
		}
		counts[k] = n
	}
	audits := []commentLocaleAudit{}
	for _, l := range locales {
		metadataBytes, err := os.ReadFile(filepath.Join(root, "locales", l, "locale.json"))
		if err != nil {
			return err
		}
		var metadata i18n.Locale
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return err
		}
		if metadata.Locale != l || metadata.HTMLLang != l {
			return fmt.Errorf("locale metadata/registry identity mismatch: %s", l)
		}
		base, err := i18n.CommentBaseLanguage(metadata.HTMLLang)
		if err != nil {
			return err
		}
		a := commentLocaleAudit{l, base, []commentPackageAudit{}}
		for _, pkg := range []string{"tour-v1", "learn-docs-v1"} {
			a.Packages = append(a.Packages, auditCommentPackage(root, c, l, base, pkg, inventory))
		}
		audits = append(audits, a)
	}
	date := time.Now().Format("20060102")
	write := func(name string, v any) error {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		b = append(b, '\n')
		return os.WriteFile(filepath.Join(*output, name+"-"+date+".json"), b, 0600)
	}
	if err := os.MkdirAll(*output, 0700); err != nil {
		return err
	}
	source := struct {
		Schema      string                   `json:"schema"`
		Counts      map[string]commentCounts `json:"counts"`
		Languages   map[string]int           `json:"languages"`
		Unsupported map[string]int           `json:"unsupported_languages"`
		Units       []commentInventoryUnit   `json:"units"`
	}{i18n.CommentInventorySchema, counts, languages, unsupported, inventory}
	if err = write("code-comment-source-inventory", source); err != nil {
		return err
	}
	sourceBytes, _ := json.Marshal(source)
	audit := struct {
		Schema    string               `json:"schema"`
		SourceSHA string               `json:"source_inventory_identity_sha256"`
		Locales   []commentLocaleAudit `json:"locales"`
	}{i18n.CommentInventorySchema, contentidentity.SHA256Hex(sourceBytes), audits}
	if err = write("code-comment-current-locale-audit", audit); err != nil {
		return err
	}
	unitBytes, err := json.Marshal(inventory)
	if err != nil {
		return err
	}
	auditBytes, err := json.Marshal(audit)
	if err != nil {
		return err
	}
	if err := write("code-comment-review-scope", map[string]any{"schema": "go-learning/code-comment-review-scope/v1", "source_inventory_identity_sha256": contentidentity.SHA256Hex(unitBytes), "audit_identity_sha256": contentidentity.SHA256Hex(auditBytes), "scope": "machine suspects only; proposal, no Reviewer ratings or lifecycle mutation", "locales": commentReviewerScope(audits, inventory)}); err != nil {
		return err
	}
	for _, a := range audits {
		if a.Locale == "zh-CN" {
			if err := write("code-comment-zh-CN-closure-audit", map[string]any{"schema": i18n.CommentInventorySchema, "source_inventory_identity_sha256": contentidentity.SHA256Hex(unitBytes), "source_comment_extraction_coverage": "COMPLETE", "ambiguous_unsupported_blocks": 0, "counts_scope": "current Tour and current 49 Learn/Docs Pages only; future source preflight excluded", "locale": a}); err != nil {
				return err
			}
		}
	}
	var summary, zh strings.Builder
	discovered, tourScanned, learnScanned, tourSuspects, learnSuspects, comparisons, carry, missing, protected, structure := 0, 0, 0, 0, 0, 0, 0, 0, 0, 0
	affected := []string{}
	unavailable := []string{}
	for _, a := range audits {
		current, suspect := false, false
		localeUnavailable := false
		for _, p := range a.Packages {
			fmt.Fprintf(&summary, "%s %s %s %s Page=%+v Example=%+v %s\n", a.Locale, a.Base, p.Package, p.State, p.Page, p.Example, p.Error)
			if p.State == "CURRENT_TARGET_UNAVAILABLE" {
				localeUnavailable = true
			}
			if p.State != "CURRENT" {
				continue
			}
			if p.State == "CURRENT" {
				current = true
				if p.Package == "tour-v1" {
					tourScanned++
				} else {
					learnScanned++
				}
			}
			if len(p.ReviewUnits) > 0 {
				suspect = true
				if p.Package == "tour-v1" {
					tourSuspects++
				} else {
					learnSuspects++
				}
			}
			comparisons += len(p.Rows)
			if a.Base != "en" {
				carry += p.Page.Exact + p.Example.Exact
			}
			missing += p.Page.Missing + p.Example.Missing
			protected += p.Page.ProtectedMismatch + p.Example.ProtectedMismatch
			structure += p.Page.StructureMismatch + p.Example.StructureMismatch
			if a.Locale == "zh-CN" {
				fmt.Fprintf(&zh, "%s state=%s Page=%+v Example=%+v\nreview_units=%s\n", p.Package, p.State, p.Page, p.Example, strings.Join(p.ReviewUnits, ","))
			}
		}
		if current {
			discovered++
		}
		if suspect {
			affected = append(affected, a.Locale)
		}
		if localeUnavailable {
			unavailable = append(unavailable, a.Locale)
		}
	}
	fmt.Fprintf(&summary, "CURRENT_LOCALES_DISCOVERED=%d\nTOUR_CURRENT_LOCALES_SCANNED=%d\nLEARN_DOCS_CURRENT_LOCALES_SCANNED=%d\nLOCALES_WITH_TOUR_COMMENT_SUSPECTS=%d\nLOCALES_WITH_LEARN_DOCS_COMMENT_SUSPECTS=%d\nTOTAL_CURRENT_TARGET_COMMENT_COMPARISONS=%d\nTOTAL_NON_ENGLISH_EXACT_SOURCE_CARRY=%d\nTOTAL_MISSING_COMMENT_SLOTS=%d\nTOTAL_PROTECTED_COMMENT_MISMATCHES=%d\nTOTAL_STRUCTURE_MISMATCHES=%d\nLOCALES_WITH_COMMENT_SUSPECTS=%s\n", discovered, tourScanned, learnScanned, tourSuspects, learnSuspects, comparisons, carry, missing, protected, structure, strings.Join(affected, ","))
	fmt.Fprintf(&summary, "CURRENT_TARGET_UNAVAILABLE_LOCALES=%s\n", strings.Join(unavailable, ","))
	for name, value := range map[string]string{"code-comment-cross-locale-summary": summary.String(), "code-comment-zh-CN-summary": zh.String()} {
		if err = os.WriteFile(filepath.Join(*output, name+"-"+date+".txt"), []byte(value), 0600); err != nil {
			return err
		}
	}
	fmt.Printf("source_units=%d current_locales=%d comparisons=%d non_english_exact_carry=%d missing=%d protected_mismatch=%d structure_mismatch=%d\nsource_inventory=%s\naudit=%s\n", len(inventory), discovered, comparisons, carry, missing, protected, structure, filepath.Join(*output, "code-comment-source-inventory-"+date+".json"), filepath.Join(*output, "code-comment-current-locale-audit-"+date+".json"))
	return nil
}
