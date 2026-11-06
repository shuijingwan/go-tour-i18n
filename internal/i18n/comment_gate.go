package i18n

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// TranslationCommentAdapter supplies representation-owned coordinates only.
// Source blocks are unresolved parser output; target blocks never supply a
// source title, companion Example or semantic classification authority.
type TranslationCommentAdapter interface {
	CommentSourceBlocks() ([]CommentCodeBlock, string, error)
	CommentTargetBlocks([]byte) ([]CommentCodeBlock, error)
}

const CommentGateActive = true

var commentGateAuthorityPaths = []string{
	"internal/i18n/comment_gate.go", "internal/i18n/comment_inventory.go",
	"internal/i18n/comment_disposition.go", "internal/i18n/comment_residual.go",
	"internal/i18n/comment_command.go", "internal/i18n/comment_transcript.go",
	"internal/i18n/comment_literal.go", "internal/i18n/comment_module.go",
	"internal/i18n/comment_projection.go", "internal/i18n/comment_template_go.go",
	"internal/i18n/comment_json_documentation.go", "internal/i18n/comment_sql.go",
	"internal/i18n/preformatted.go", "internal/i18n/go_example_protect.go",
	"internal/i18n/go_example_validate.go", "internal/i18n/language_corpus.go",
	"internal/i18n/status.go",
}

// CommentLocaleBaseLanguage uses the current workflow locale metadata, never a
// locale allowlist or the translated Page's language attributes.
func CommentLocaleBaseLanguage(root, locale string) (string, GlossaryArchiveReference, error) {
	var empty GlossaryArchiveReference
	if err := ValidateLocaleName(locale); err != nil {
		return "", empty, err
	}
	path := filepath.ToSlash(filepath.Join("locales", locale, "locale.json"))
	b, err := readCompatibilityFile(root, path)
	if err != nil {
		return "", empty, err
	}
	var metadata Locale
	if err := contentidentity.StrictJSON(b, &metadata); err != nil {
		return "", empty, err
	}
	if metadata.Locale != locale || metadata.HTMLLang != locale {
		return "", empty, fmt.Errorf("comment gate locale metadata identity mismatch: %s", locale)
	}
	base, err := CommentBaseLanguage(metadata.HTMLLang)
	return base, GlossaryArchiveReference{Path: path, SHA256: sum(b)}, err
}

// CommentGateError distinguishes current machine failure from a language
// judgment. All offending slot identities are retained, not just the first.
type CommentGateError struct {
	Reason  string
	SlotIDs []string
	cause   error
}

func (e *CommentGateError) Error() string {
	return fmt.Sprintf("code-comment gate: %s: %s: %v", e.Reason, strings.Join(e.SlotIDs, ", "), e.cause)
}
func (e *CommentGateError) Unwrap() error { return e.cause }

// ValidateTranslationUnitCommentCoverage reuses the source inventory contract
// for exactly this Unit. It neither scans a package nor loads target evidence.
// The caller has already validated the representation and selected candidate.
func ValidateTranslationUnitCommentCoverage(root string, c *Catalog, u *TranslationUnit, locale string, candidate []byte) ([]CommentSlotComparison, error) {
	if u == nil || c == nil || sum(u.Source) != u.SourceSHA256 {
		return nil, fmt.Errorf("code-comment gate: source identity mismatch")
	}
	var source, target []CommentCodeBlock
	var title string
	var examples []Example
	var err error
	if u.Adapter != nil {
		a, ok := u.Adapter.(TranslationCommentAdapter)
		if !ok {
			return nil, fmt.Errorf("code-comment gate: adapter lacks source comment authority: %s", u.ID)
		}
		source, title, err = a.CommentSourceBlocks()
		if err == nil {
			target, err = a.CommentTargetBlocks(candidate)
		}
	} else if u.Kind == UnitKindPage {
		source, target = TourPageCommentBlocks(u.Source), TourPageCommentBlocks(candidate)
		for _, e := range c.Examples {
			for _, id := range e.ReferencedBy {
				if id == u.ID {
					examples = append(examples, e)
				}
			}
		}
	} else if u.Kind == UnitKindExample {
		source = []CommentCodeBlock{AnalyzeCommentCode(string(u.Source), "go", "example/1", true)}
		target = []CommentCodeBlock{AnalyzeCommentCode(string(candidate), "go", "example/1", true)}
	} else {
		err = fmt.Errorf("unsupported Unit kind: %s", u.Kind)
	}
	if err != nil {
		return nil, fmt.Errorf("code-comment gate: extraction unavailable: %w", err)
	}
	resolved := make([]CommentCodeBlock, len(source))
	for i, b := range source {
		resolved[i] = SourceCommentBlockDisposition(b, title, examples)
	}
	if err := ValidateSourceCommentBlockSet(source, resolved); err != nil {
		return nil, fmt.Errorf("code-comment gate: source disposition: %w", err)
	}
	target = SourceSelectedTargetCommentBlocks(resolved, target)
	base := ""
	// With no comments, structural comparison still runs. Empty projections do
	// not require a language policy or claim any exception for an unknown locale.
	for _, b := range resolved {
		if len(b.Comments) > 0 {
			base, _, err = CommentLocaleBaseLanguage(root, locale)
			if err != nil {
				return nil, fmt.Errorf("code-comment gate: base-language authority: %w", err)
			}
			break
		}
	}
	pkg := "tour-v1"
	for _, p := range c.TranslationPackages {
		for _, a := range p.Units {
			if a.Unit.ID == u.ID {
				pkg = p.ID
			}
		}
	}
	rows, failures := CompareCommentBlocks(pkg+"/"+u.SourcePath+"/"+string(u.Kind)+"/"+u.ID, base, resolved, target)
	if err := ValidateCommentCoverage(rows, failures, true); err != nil {
		reason := "comment structure/protection/coverage failure"
		if ValidateCommentCoverage(rows, failures, false) == nil {
			reason = "non-English teaching-comment exact source carry"
		}
		ids := []string{}
		for _, r := range rows {
			if r.Missing || r.Extra || !r.StructureMatch || !r.ProtectedUnchanged || r.SemanticKind == "unclassified" || r.ReviewRequired {
				ids = append(ids, r.SlotID)
			}
		}
		return rows, &CommentGateError{Reason: reason, SlotIDs: ids, cause: err}
	}
	return rows, nil
}
