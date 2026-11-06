package i18n

import "go/token"

// Only previously unsupported source blocks enter here. Existing classified
// blocks retain their extractor, semantics and identities unchanged.
func resolveCommentResidual(b CommentCodeBlock, title string) CommentCodeBlock {
	if b.DeclaredLanguage != "" && b.DeclaredLanguage != "go" && b.DeclaredLanguage != "golang" && b.DeclaredLanguage != "gomod" && b.DeclaredLanguage != "go.mod" && b.DeclaredLanguage != "json" && b.DeclaredLanguage != "jsonc" && b.DeclaredLanguage != "json-schema" {
		return b
	}
	finish := func(result CommentCodeBlock, proof string) CommentCodeBlock {
		result.DispositionEvidence = proof
		return result
	}
	// Explicit declarations constrain which grammar may be tried.
	if b.DeclaredLanguage == "" {
		if records, ok := TerminalTranscript(b.Code); ok {
			b.Unsupported = false
			b.Diagnostic = ""
			b.Extractor = "terminal-transcript/v1"
			b.Language = "terminal-transcript"
			b.Disposition = "terminal-transcript"
			b.Skeleton = b.Code
			b.Transcript = records
			return finish(b, "exact source prompt grammar and quote-aware protected command envelope; ordered command/output records")
		}
	}
	if b.DeclaredLanguage == "" || b.DeclaredLanguage == "go" || b.DeclaredLanguage == "golang" {
		if result, ok := AnalyzeTemplateGoComments(b.Code, b.Coordinate); ok {
			return finish(result, "Go template AST literal raw actions, then complete existing Go fragment parser; original byte coordinates preserved")
		}
	}
	if b.DeclaredLanguage == "" {
		if result := AnalyzeModuleCommentCode(b.Code, b.Coordinate); !result.Unsupported {
			return finish(result, "complete official x/mod/modfile.Parse grammar; original tokens and comments")
		}
		if result := AnalyzeWorkCommentCode(b.Code, b.Coordinate); !result.Unsupported {
			return finish(result, "complete official x/mod/modfile.ParseWork grammar; original tokens and comments")
		}
	}
	if b.DeclaredLanguage == "" || b.DeclaredLanguage == "json" || b.DeclaredLanguage == "jsonc" || b.DeclaredLanguage == "json-schema" {
		if result, ok := AnalyzeJSONDocumentation(b.Code, b.Coordinate); ok {
			return finish(result, "existing restricted JSON scanner/grammar, allowing member excerpts and member-position documentation elision")
		}
	}
	if _, ok := ModuleCommentAuthority(b.DeclaredLanguage, title); ok && b.VariableMarkup {
		events, err := scanGoExampleLexicalEvents([]byte(b.Code))
		plain := err == nil
		for _, e := range events {
			if e.token == token.COMMENT {
				plain = false
			}
		}
		if plain {
			b.Unsupported = false
			b.Diagnostic = ""
			b.Extractor = "source-var-notation/v1"
			b.Language = "gomod-notation"
			b.Disposition = "machine-semantic-opaque"
			b.Skeleton = b.Code
			return finish(b, "frozen format-reference metadata and HTML var metavariable markup; no comment tokens; not concrete module semantics")
		}
	}
	if b.DeclaredLanguage == "" {
		if records, ok := BareDocumentationCommand(b.Code); ok {
			b.Unsupported = false
			b.Diagnostic = ""
			b.Extractor = "documentation-command/v1"
			b.Language = "terminal-command"
			b.Disposition = "terminal-transcript"
			b.Skeleton = b.Code
			b.Transcript = records
			return finish(b, "complete finite documentation command grammar, protected original command records")
		}
		if SQLCommentFreeProof(b.Code) {
			b.Unsupported = false
			b.Diagnostic = ""
			b.Extractor = "mysql-tutorial-grammar/v1"
			b.Language = "sql"
			b.Skeleton = b.Code
			return finish(b, "complete restricted MySQL tutorial productions and quote-aware tokenizer; no comment tokens")
		}
		if proof, ok := SourceLiteralProof(b.Code, b.SourceContext); ok {
			b.Unsupported = false
			b.Diagnostic = ""
			b.Extractor = "source-literal-grammar/v1"
			b.Language = "literal"
			b.Disposition = "non-code-output-or-literal"
			b.Skeleton = b.Code
			return finish(b, proof)
		}
	}
	return b
}
