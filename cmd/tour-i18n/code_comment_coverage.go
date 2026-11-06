package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/site"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
	"github.com/shuijingwan/go-tour-i18n/internal/sourcestorage"
)

type commentBlockRecord struct {
	Package          string        `json:"package"`
	ParentKind       i18n.UnitKind `json:"parent_kind"`
	ParentID         string        `json:"parent_unit_id"`
	Route            string        `json:"route"`
	SourcePath       string        `json:"source_path"`
	StableIndex      int           `json:"stable_index"`
	Coordinate       string        `json:"coordinate"`
	EnclosingKind    string        `json:"enclosing_kind"`
	DeclaredLanguage string        `json:"declared_language"`
	SourceSHA        string        `json:"source_sha256"`
	BlockSHA         string        `json:"block_sha256"`
	FirstLine        string        `json:"first_nonempty_line"`
	LineCount        int           `json:"line_count"`
	ByteCount        int           `json:"byte_count"`
	Extractor        string        `json:"current_extractor"`
	Diagnostic       string        `json:"current_diagnostic"`
	SourceBefore     string        `json:"source_before"`
	Code             string        `json:"code"`
}

type commentDispositionRecord struct {
	commentBlockRecord
	BaselineUnsupported bool   `json:"baseline_unsupported"`
	Language            string `json:"resolved_language"`
	Disposition         string `json:"disposition"`
	Authority           string `json:"classification_evidence"`
	ResidualClass       string `json:"residual_class,omitempty"`
	ResolvedExtractor   string `json:"resolved_extractor,omitempty"`
}

func closeCommentInventory(c *i18n.Catalog, inventory []commentInventoryUnit) ([]commentInventoryUnit, []commentDispositionRecord, error) {
	closed := append([]commentInventoryUnit{}, inventory...)
	records := []commentDispositionRecord{}
	for i, u := range inventory {
		unit, err := c.Unit(u.ID)
		if err != nil {
			return nil, nil, err
		}
		title := ""
		examples := []i18n.Example{}
		if u.Package == "tour-v1" {
			for _, e := range c.Examples {
				for _, id := range e.ReferencedBy {
					if id == u.ID {
						examples = append(examples, e)
					}
				}
			}
		} else {
			_, title, _, err = sitecontent.PageRenderInput(unit.Source)
			if err != nil {
				return nil, nil, err
			}
		}
		closed[i].Blocks = append([]i18n.CommentCodeBlock{}, u.Blocks...)
		for j, b := range u.Blocks {
			resolved := i18n.SourceCommentBlockDisposition(b, title, examples)
			for k := range resolved.Comments {
				resolved.Comments[k].SlotID = i18n.CommentSlotIdentity(u.Package+"/"+u.Path+"/"+string(u.Kind)+"/"+u.ID, resolved, resolved.Comments[k])
			}
			closed[i].Blocks[j] = resolved
			r := commentDispositionRecord{commentBlockRecord: commentBlockEvidence(u, b, unit.Source), BaselineUnsupported: b.Unsupported, Language: resolved.Language, Disposition: resolved.Disposition, Authority: resolved.DispositionEvidence}
			r.ResolvedExtractor = resolved.Extractor
			if resolved.Unsupported {
				r.ResidualClass = "undeclared-format-or-output-without-parser-proof"
				if _, ok := i18n.ModuleCommentAuthority(b.DeclaredLanguage, title); ok {
					r.ResidualClass = "source-declared-module-reference-nonconcrete-or-other-format"
				}
				if u.Package == "tour-v1" {
					r.ResidualClass = "tour-fragment-without-referenced-Go-AST-proof"
				}
				if b.DeclaredLanguage != "" {
					r.ResidualClass = "explicit-language-without-supported-complete-grammar"
				}
			}
			records = append(records, r)
		}
		if err := i18n.ValidateSourceCommentBlockSet(u.Blocks, closed[i].Blocks); err != nil {
			return nil, nil, err
		}
	}
	return closed, records, nil
}

func commentInventoryCounts(inventory []commentInventoryUnit) map[string]commentCounts {
	counts := map[string]commentCounts{}
	for _, u := range inventory {
		k := u.Package + "/" + string(u.Kind)
		n := counts[k]
		for _, b := range u.Blocks {
			n.Blocks++
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
	return counts
}

func requireCompleteCommentInventory(inventory []commentInventoryUnit) error {
	for _, u := range inventory {
		if err := i18n.ValidateCommentBlockDispositions(u.Blocks); err != nil {
			return err
		}
		for _, b := range u.Blocks {
			if b.Unsupported {
				return fmt.Errorf("source coverage partial: %s/%s/%s; cross-locale audit not run", u.Package, u.ID, b.Coordinate)
			}
		}
	}
	return nil
}

func writeCommentCoverage(root, output, primaryLocale, predecessorPath string, c *i18n.Catalog, inventory []commentInventoryUnit) error {
	closed, records, err := closeCommentInventory(c, inventory)
	if err != nil {
		return err
	}
	predecessor := map[string]any{}
	if predecessorPath != "" {
		raw, err := os.ReadFile(predecessorPath)
		if err != nil {
			return err
		}
		previous, err := readCommentPredecessor(raw)
		if err != nil {
			return err
		}
		counts, err := compareCommentPredecessor(previous, records)
		if err != nil {
			return err
		}
		predecessor = map[string]any{"path": predecessorPath, "sha256": contentidentity.SHA256Hex(raw), "counts": counts}
	}
	summary := map[string]int{"UNSUPPORTED_BLOCKS_INPUT": 0, "EXPLICIT_LANGUAGE_BLOCKS": 0, "STRICTLY_INFERRED_LANGUAGE_BLOCKS": 0, "TERMINAL_TRANSCRIPT_BLOCKS": 0, "NON_CODE_OUTPUT_BLOCKS": 0, "COMMENT_FREE_GRAMMAR_BLOCKS": 0, "MACHINE_OPAQUE_BLOCKS": 0, "AMBIGUOUS_UNSUPPORTED_BLOCKS": 0}
	languages := map[string]int{}
	residuals := map[string]int{}
	for _, r := range records {
		if !r.BaselineUnsupported {
			continue
		}
		summary["UNSUPPORTED_BLOCKS_INPUT"]++
		language := r.Language
		if language == "" {
			language = "unspecified"
		}
		languages[language]++
		switch r.Disposition {
		case "comment-extracted":
			if r.DeclaredLanguage != "" {
				summary["EXPLICIT_LANGUAGE_BLOCKS"]++
			} else {
				summary["STRICTLY_INFERRED_LANGUAGE_BLOCKS"]++
			}
		case "comment-free-by-grammar":
			summary["COMMENT_FREE_GRAMMAR_BLOCKS"]++
		case "terminal-transcript":
			summary["TERMINAL_TRANSCRIPT_BLOCKS"]++
		case "non-code-output-or-literal":
			summary["NON_CODE_OUTPUT_BLOCKS"]++
		case "machine-semantic-opaque":
			summary["MACHINE_OPAQUE_BLOCKS"]++
		case "ambiguous-unsupported":
			summary["AMBIGUOUS_UNSUPPORTED_BLOCKS"]++
			residuals[r.ResidualClass]++
		}
	}
	date := time.Now().Format("20060102")
	write := func(name string, v any) error {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(output, name+"-"+date+".json"), append(b, '\n'), 0600)
	}
	if err := requireCompleteCommentInventory(closed); err != nil {
		return err
	}
	if err := write("code-comment-block-disposition", map[string]any{"schema": i18n.CommentInventorySchema, "taxonomy": summary, "language_counts": languages, "residual_classes": residuals, "predecessor": predecessor, "blocks": records}); err != nil {
		return err
	}
	inventoryBytes, err := json.Marshal(closed)
	if err != nil {
		return err
	}
	if err := write("code-comment-source-inventory-closure", map[string]any{"schema": i18n.CommentInventorySchema, "inventory_identity_sha256": contentidentity.SHA256Hex(inventoryBytes), "counts": commentInventoryCounts(closed), "units": closed, "ambiguous_unsupported_blocks": summary["AMBIGUOUS_UNSUPPORTED_BLOCKS"]}); err != nil {
		return err
	}
	if err := writeCommentFrontierPreflight(root, write); err != nil {
		return err
	}
	if primaryLocale != "" {
		if err := writeCommentPrimaryAudit(root, primaryLocale, closed, summary["AMBIGUOUS_UNSUPPORTED_BLOCKS"], write); err != nil {
			return err
		}
	}
	fmt.Printf("taxonomy=%s\nlanguages=%s\nresidual_classes=%s\n", mustCommentJSON(summary), mustCommentJSON(languages), mustCommentJSON(residuals))
	if summary["AMBIGUOUS_UNSUPPORTED_BLOCKS"] > 0 {
		return fmt.Errorf("source coverage partial: %d ambiguous blocks; cross-locale audit not run", summary["AMBIGUOUS_UNSUPPORTED_BLOCKS"])
	}
	return nil
}

// The primary-locale report is explicitly partial when source grammar is
// unknown. It cannot overwrite the last complete cross-locale evidence.
func writeCommentPrimaryAudit(root, locale string, inventory []commentInventoryUnit, ambiguous int, write func(string, any) error) error {
	raw, err := os.ReadFile(filepath.Join(root, "locales", locale, "locale.json"))
	if err != nil {
		return err
	}
	var metadata i18n.Locale
	if err := sitecontent.StrictJSON(raw, &metadata); err != nil {
		return err
	}
	if metadata.Locale != locale || metadata.HTMLLang != locale {
		return fmt.Errorf("primary locale metadata identity mismatch")
	}
	base, err := i18n.CommentBaseLanguage(metadata.HTMLLang)
	if err != nil {
		return err
	}
	c, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		return err
	}
	if err := sitecontent.AttachRegisteredTranslationPackages(root, c); err != nil {
		return err
	}
	sitecontent.AttachIntegratedSurfaceAuthority(c)
	site.AttachPreviewFailureAuthority(c)
	a := commentLocaleAudit{Locale: locale, Base: base, Packages: []commentPackageAudit{}}
	for _, pkg := range []string{"tour-v1", "learn-docs-v1"} {
		a.Packages = append(a.Packages, auditCommentPackage(root, c, locale, base, pkg, inventory))
	}
	source, err := json.Marshal(inventory)
	if err != nil {
		return err
	}
	coverage := "COMPLETE"
	if ambiguous != 0 {
		coverage = "PARTIAL_UNSUPPORTED"
	}
	return write("code-comment-"+locale+"-closure-audit", map[string]any{"schema": i18n.CommentInventorySchema, "source_inventory_identity_sha256": contentidentity.SHA256Hex(source), "source_comment_extraction_coverage": coverage, "ambiguous_unsupported_blocks": ambiguous, "counts_scope": "only parser-proved source slots; unknown blocks excluded, never treated as comment-free", "locale": a})
}

func mustCommentJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

func writeCommentFrontierPreflight(root string, write func(string, any) error) error {
	m, err := sourcestorage.LoadManifest(root)
	if err != nil {
		return err
	}
	manifest, err := sitecontent.ReferenceFile(root, sitecontent.SourceManifestPath)
	if err != nil {
		return err
	}
	pages := []map[string]any{}
	totalAmbiguous := 0
	for _, entry := range []struct{ path, route string }{{"_content/doc/index.html", "/doc/"}, {"_content/doc/install.html", "/doc/install"}} {
		var file *sourcestorage.FrozenSourceFile
		for i := range m.Files {
			if m.Files[i].Path == entry.path {
				f := m.Files[i]
				file = &f
				break
			}
		}
		if file == nil {
			return fmt.Errorf("frontier missing frozen source: %s", entry.path)
		}
		raw, err := sourcestorage.ReadAdoptedSource(root, *file)
		if err != nil {
			return err
		}
		blocks, err := sitecontent.SourceHTMLCommentBlocks(raw)
		if err != nil {
			return err
		}
		u := commentInventoryUnit{Package: "future-source-preflight", Kind: i18n.UnitKind("source-page"), Route: entry.route, Path: entry.path, SourceSHA: file.SHA256}
		records := []commentDispositionRecord{}
		ambiguous := 0
		for i, b := range blocks {
			resolved := i18n.SourceCommentBlockDisposition(b, "", nil)
			blocks[i] = resolved
			r := commentDispositionRecord{commentBlockRecord: commentBlockEvidence(u, b, raw), BaselineUnsupported: b.Unsupported, Language: resolved.Language, Disposition: resolved.Disposition, Authority: resolved.DispositionEvidence}
			if resolved.Unsupported {
				ambiguous++
				r.ResidualClass = "future-adoption-blocker-no-complete-source-grammar-proof"
			}
			records = append(records, r)
		}
		baseline, err := sitecontent.SourceHTMLCommentBlocks(raw)
		if err != nil {
			return err
		}
		if err := i18n.ValidateSourceCommentBlockSet(baseline, blocks); err != nil {
			return err
		}
		u.Blocks = blocks
		counts := commentInventoryCounts([]commentInventoryUnit{u})["future-source-preflight/source-page"]
		pages = append(pages, map[string]any{"scope": "future-source-only-not-current-package", "route": entry.route, "source_path": entry.path, "source_sha256": file.SHA256, "formal_unit_created": false, "current_local_coverage": false, "counts": counts, "ambiguous_unsupported_blocks": ambiguous, "blocks": records})
		totalAmbiguous += ambiguous
	}
	if err := write("code-comment-frontier-source-preflight", map[string]any{"schema": i18n.CommentInventorySchema, "frozen_manifest": manifest, "manifest_identity_sha256": m.Identity, "pages": pages}); err != nil {
		return err
	}
	if totalAmbiguous > 0 {
		return fmt.Errorf("future adoption comment preflight failed: %d ambiguous blocks", totalAmbiguous)
	}
	return nil
}

func commentBlockEvidence(u commentInventoryUnit, b i18n.CommentCodeBlock, raw []byte) commentBlockRecord {
	r := commentBlockRecord{Package: u.Package, ParentKind: u.Kind, ParentID: u.ID, Route: u.Route, SourcePath: u.Path, StableIndex: u.StableIndex, Coordinate: b.Coordinate, EnclosingKind: b.EnclosingKind, DeclaredLanguage: b.DeclaredLanguage, SourceSHA: u.SourceSHA, BlockSHA: b.SHA256, ByteCount: len(b.Code), Extractor: b.Extractor, Diagnostic: b.Diagnostic, Code: b.Code}
	for _, line := range strings.Split(b.Code, "\n") {
		if strings.TrimSpace(line) != "" {
			r.FirstLine = strings.TrimSpace(line)
			break
		}
	}
	if b.Code != "" {
		r.LineCount = strings.Count(strings.TrimSuffix(b.Code, "\n"), "\n") + 1
	}
	if b.Start >= 0 && b.Start <= len(raw) {
		a := max(0, b.Start-1000)
		for a < b.Start && !utf8.RuneStart(raw[a]) {
			a++
		}
		r.SourceBefore = string(raw[a:b.Start])
	}
	return r
}

func commentCoverageCommand(root string, args []string) error {
	fs := flag.NewFlagSet("code-comments "+args[0], flag.ContinueOnError)
	output := fs.String("output-dir", "/tmp", "diagnostic evidence directory")
	locale := fs.String("locale", "", "optional single-locale known-slot comparison (coverage only)")
	predecessor := fs.String("predecessor", "", "optional source disposition predecessor exact-set evidence (coverage only)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	if *predecessor != "" && args[0] != "coverage" {
		return fmt.Errorf("--predecessor requires coverage")
	}
	if *locale != "" {
		if args[0] != "coverage" {
			return fmt.Errorf("--locale requires coverage")
		}
		if err := i18n.ValidateLocaleName(*locale); err != nil {
			return err
		}
	}
	c, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		return err
	}
	if _, err := sitecontent.AttachPageTranslationUnits(root, c); err != nil {
		return err
	}
	inventory, err := commentSourceInventory(c)
	if err != nil {
		return err
	}
	records := []commentBlockRecord{}
	kinds := map[string]int{}
	languages := map[string]int{}
	for _, u := range inventory {
		unit, err := c.Unit(u.ID)
		if err != nil {
			return err
		}
		for _, b := range u.Blocks {
			if b.Unsupported {
				records = append(records, commentBlockEvidence(u, b, unit.Source))
				kinds[b.EnclosingKind]++
				languages[b.DeclaredLanguage]++
			}
		}
	}
	if err := os.MkdirAll(*output, 0700); err != nil {
		return err
	}
	path := filepath.Join(*output, "code-comment-unsupported-blocks-"+time.Now().Format("20060102")+".json")
	b, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(path, append(b, '\n'), 0600); err != nil {
		return err
	}
	if args[0] == "coverage" {
		return writeCommentCoverage(root, *output, *locale, *predecessor, c, inventory)
	}
	counts, _ := json.Marshal(struct {
		Blocks     int            `json:"unsupported_blocks"`
		Enclosures map[string]int `json:"enclosing_kinds"`
		Languages  map[string]int `json:"declared_languages"`
	}{len(records), kinds, languages})
	fmt.Printf("%s\nevidence=%s\n", counts, path)
	return nil
}
