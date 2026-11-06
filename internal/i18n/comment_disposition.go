package i18n

import "fmt"

// ValidateSourceCommentBlockSet binds the disposition exact set to the parser's
// original source blocks. A classifier may not drop, insert, reorder or rewrite
// source bytes, declarations, or structural coordinates.
func ValidateSourceCommentBlockSet(source, resolved []CommentCodeBlock) error {
	if len(source) != len(resolved) {
		return fmt.Errorf("source block disposition exact-set count mismatch")
	}
	for i, s := range source {
		r := resolved[i]
		if s.Coordinate != r.Coordinate || s.SHA256 != r.SHA256 || s.Code != r.Code || s.Start != r.Start || s.End != r.End || s.EnclosingKind != r.EnclosingKind || s.DeclaredLanguage != r.DeclaredLanguage || s.SourceContext != r.SourceContext || s.VariableMarkup != r.VariableMarkup {
			return fmt.Errorf("source block disposition identity mismatch: %s", s.Coordinate)
		}
	}
	return ValidateCommentBlockDispositions(resolved)
}

// SourceSelectedTargetCommentBlocks freezes the extractor selected from source.
// Target prose/metadata never supplies language authority or companion proofs.
// CompareCommentBlocks still rejects count, coordinate and skeleton drift.
func SourceSelectedTargetCommentBlocks(source, target []CommentCodeBlock) []CommentCodeBlock {
	out := append([]CommentCodeBlock{}, target...)
	for i, t := range out {
		if i >= len(source) {
			continue
		}
		s := source[i]
		if s.Coordinate != t.Coordinate || s.DeclaredLanguage != t.DeclaredLanguage {
			out[i].Unsupported = true
			out[i].Diagnostic = "source/target declaration or coordinate mismatch"
			continue
		}
		var selected CommentCodeBlock
		switch {
		case s.Unsupported:
			out[i].Unsupported = true
			out[i].Comments = nil
			out[i].Diagnostic = "source coverage unknown; target cannot resolve it"
			continue
		case s.Extractor == "modfile-strict/v1":
			selected = AnalyzeModuleCommentCode(t.Code, t.Coordinate)
		case s.Extractor == "modfile-work-strict/v1":
			selected = AnalyzeWorkCommentCode(t.Code, t.Coordinate)
		case s.Extractor == "tour-go-template-literal/v1":
			var ok bool
			selected, ok = AnalyzeTemplateGoComments(t.Code, t.Coordinate)
			if !ok {
				selected = t
				selected.Unsupported = true
			}
		case s.Extractor == "tour-scanner-json-documentation/v1":
			var ok bool
			selected, ok = AnalyzeJSONDocumentation(t.Code, t.Coordinate)
			if !ok {
				selected = t
				selected.Unsupported = true
			}
		case s.Disposition == "terminal-transcript" || s.Disposition == "non-code-output-or-literal" || s.Disposition == "machine-semantic-opaque" || s.Extractor == "mysql-tutorial-grammar/v1":
			selected = s
			selected.Code, selected.SHA256 = t.Code, t.SHA256
			if t.Code != s.Code {
				selected.Unsupported = true
				selected.Diagnostic = "protected source semantic record changed"
			}
		case s.Extractor == "referenced-go-ast-projection/v1":
			selected = s
			if t.Code != s.Code {
				selected.Unsupported = true
				selected.Diagnostic = "comment-free source projection bytes changed"
			}
			selected.Code, selected.SHA256 = t.Code, t.SHA256
		case s.Extractor == "tour-go-scanner/v1":
			selected = AnalyzeCommentCode(t.Code, "go", t.Coordinate, false)
		default:
			selected = t
		}
		selected.Start, selected.End = t.Start, t.End
		selected.EnclosingKind, selected.DeclaredLanguage = t.EnclosingKind, t.DeclaredLanguage
		out[i] = selected
	}
	return out
}

// SourceCommentBlockDisposition consumes source authority only. The target
// cannot supply a metadata title, language declaration, or companion Example.
func SourceCommentBlockDisposition(b CommentCodeBlock, metadataTitle string, tourExamples []Example) CommentCodeBlock {
	metadata := b
	proof := "existing source parser acceptance: " + b.Extractor
	if b.Unsupported {
		if authority, ok := ModuleCommentAuthority(b.DeclaredLanguage, metadataTitle); ok {
			module := AnalyzeModuleCommentCode(b.Code, b.Coordinate)
			if !module.Unsupported {
				b = module
				proof = authority + "; strict full go.mod grammar and parser-owned comment positions"
			}
		}
		if b.Unsupported && (b.DeclaredLanguage == "" || b.DeclaredLanguage == "go" || b.DeclaredLanguage == "golang") {
			if authority, ok := TourCommentFreeProjection(b.Code, tourExamples); ok {
				b.Unsupported = false
				b.Language = "go"
				b.Extractor = "referenced-go-ast-projection/v1"
				b.Skeleton = b.Code
				b.Diagnostic = ""
				proof = authority
			}
		}
	}
	if b.Unsupported {
		b = resolveCommentResidual(b, metadataTitle)
		if !b.Unsupported {
			proof = b.DispositionEvidence
		}
	}
	b.Start, b.End, b.EnclosingKind, b.DeclaredLanguage = metadata.Start, metadata.End, metadata.EnclosingKind, metadata.DeclaredLanguage
	b.SourceContext, b.VariableMarkup = metadata.SourceContext, metadata.VariableMarkup
	if b.Unsupported {
		b.Disposition = "ambiguous-unsupported"
		b.DispositionEvidence = "no explicit supported language or complete source-owned grammar/projection proof; " + b.Diagnostic
		return b
	}
	if b.Disposition == "" {
		b.Disposition = "comment-free-by-grammar"
		if len(b.Comments) > 0 {
			b.Disposition = "comment-extracted"
		}
	}
	b.DispositionEvidence = proof
	return b
}

// ValidateCommentBlockDispositions enforces exactly one disposition per input
// coordinate. Unknown/contradictory claims cannot silently satisfy coverage.
func ValidateCommentBlockDispositions(blocks []CommentCodeBlock) error {
	seen := map[string]bool{}
	for _, b := range blocks {
		if b.Coordinate == "" || seen[b.Coordinate] || b.DispositionEvidence == "" {
			return fmt.Errorf("missing/duplicate/unproved block disposition: %s", b.Coordinate)
		}
		seen[b.Coordinate] = true
		switch b.Disposition {
		case "comment-extracted":
			if b.Unsupported || len(b.Comments) == 0 || b.Extractor == "" {
				return fmt.Errorf("invalid extracted disposition: %s", b.Coordinate)
			}
		case "comment-free-by-grammar", "terminal-transcript", "non-code-output-or-literal", "machine-semantic-opaque":
			if b.Unsupported || len(b.Comments) != 0 || b.Skeleton != b.Code {
				return fmt.Errorf("invalid non-comment disposition: %s", b.Coordinate)
			}
			if b.Disposition == "terminal-transcript" {
				pos := 0
				for _, r := range b.Transcript {
					if r.Start != pos || r.End < r.Start || r.End > len(b.Code) || r.Text != b.Code[r.Start:r.End] || (r.Kind != "command" && r.Kind != "output") {
						return fmt.Errorf("invalid transcript exact-set: %s", b.Coordinate)
					}
					pos = r.End
				}
				if len(b.Transcript) == 0 || pos != len(b.Code) {
					return fmt.Errorf("incomplete transcript exact-set: %s", b.Coordinate)
				}
			}
		case "ambiguous-unsupported":
			if !b.Unsupported || len(b.Comments) != 0 {
				return fmt.Errorf("invalid ambiguous disposition: %s", b.Coordinate)
			}
		default:
			return fmt.Errorf("unknown block disposition: %s", b.Coordinate)
		}
	}
	return nil
}
