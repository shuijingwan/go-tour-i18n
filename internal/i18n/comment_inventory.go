package i18n

// This is a diagnostic projection of the existing Tour comment contract. It
// does not register TranslationUnits or change historical validation policy.
import (
	"fmt"
	"go/scanner"
	"go/token"
	"go/types"
	"regexp"
	"strings"
)

const CommentInventorySchema = "go-learning/code-comment-inventory/v1"

type CodeComment struct {
	SlotID        string `json:"slot_id"`
	Position      int    `json:"position"`
	TokenPosition int    `json:"token_position"`
	Start         int    `json:"start"`
	End           int    `json:"end"`
	Kind          string `json:"semantic_kind"`
	Policy        string `json:"protection_policy"`
	Text          string `json:"text"`
	Body          string `json:"body"`
	Style         string `json:"style"`
}

type CommentCodeBlock struct {
	SourceContext       string             `json:"source_context,omitempty"`
	VariableMarkup      bool               `json:"variable_markup,omitempty"`
	Transcript          []TranscriptRecord `json:"transcript,omitempty"`
	Disposition         string             `json:"disposition,omitempty"`
	DispositionEvidence string             `json:"disposition_evidence,omitempty"`
	EnclosingKind       string             `json:"enclosing_kind"`
	DeclaredLanguage    string             `json:"declared_language"`
	Start               int                `json:"source_start"`
	End                 int                `json:"source_end"`
	Coordinate          string             `json:"coordinate"`
	Language            string             `json:"code_language"`
	Extractor           string             `json:"extractor"`
	Unsupported         bool               `json:"unsupported"`
	Diagnostic          string             `json:"diagnostic,omitempty"`
	SHA256              string             `json:"sha256"`
	Comments            []CodeComment      `json:"comments"`
	Code                string             `json:"-"`
	Skeleton            string             `json:"-"`
}

// TourPageCommentBlocks shares the Present indentation parser exactly.
func TourPageCommentBlocks(source []byte) []CommentCodeBlock {
	out := []CommentCodeBlock{}
	for i, b := range preformattedBlocks(string(source)) {
		block := AnalyzeCommentCode(b.Text, "", fmt.Sprintf("present/%d", i+1), false)
		block.EnclosingKind, block.Start, block.End = "present-preformatted", b.Start, b.End
		prefix := strings.TrimSpace(string(source[:b.Start]))
		if n := strings.LastIndex(prefix, "\n\n"); n >= 0 {
			prefix = prefix[n+2:]
		}
		block.SourceContext = prefix
		out = append(out, block)
	}
	return out
}

// AnalyzeCommentCode uses Go's lexer and existing source classifiers. Explicit
// JSON fences use the restricted JSON/schema grammar below, never Go inference.
// Unknown languages are reported, rather than searched with comment regexes.
func AnalyzeCommentCode(code, lang, coordinate string, example bool) CommentCodeBlock {
	return analyzeCommentCodeMode(code, lang, coordinate, example, false)
}

func analyzeCommentCodeMode(code, lang, coordinate string, example, documentationJSON bool) CommentCodeBlock {
	b := CommentCodeBlock{Coordinate: coordinate, Language: lang, DeclaredLanguage: lang, Code: code, SHA256: sum([]byte(code)), Comments: []CodeComment{}}
	jsonCode := lang == "json" || lang == "jsonc" || lang == "json-schema"
	goCode := lang == "go" || lang == "golang" || example || (lang == "" && isSafeGoFragment(code))
	if !jsonCode && !goCode {
		b.Unsupported = true
		b.Diagnostic = "UNSUPPORTED_COMMENT_LANGUAGE=" + lang
		if lang == "" {
			b.Diagnostic = "UNSUPPORTED_COMMENT_LANGUAGE=unspecified"
		}
		return b
	}
	if goCode {
		b.Language, b.Extractor = "go", "tour-go-scanner/v1"
	} else {
		b.Extractor = "tour-scanner-json-schema/v1"
	}
	fset := token.NewFileSet()
	f := fset.AddFile("code", -1, len(code))
	var scan scanner.Scanner
	var scanErr string
	scan.Init(f, []byte(code), func(_ token.Position, message string) {
		if scanErr == "" {
			scanErr = message
		}
	}, scanner.ScanComments)
	var grammar []jsonSchemaToken
	var skeleton strings.Builder
	previous, tokenPosition := 0, 0
	a := analyzePreformattedGo(code)
	open := map[int]bool{}
	for _, c := range a.Comments {
		if !a.Static && c.Translatable {
			open[c.Start] = true
		}
	}
	for {
		pos, tok, literal := scan.Scan()
		if tok == token.EOF {
			break
		}
		if tok != token.COMMENT {
			if tok != token.SEMICOLON || literal != "\n" {
				grammar = append(grammar, jsonSchemaToken{tok, literal})
				tokenPosition++
			}
			continue
		}
		start, end := f.Offset(pos), f.Offset(pos)+len(literal)
		_, _, body := goCommentPayload([]byte(code), start, end, literal)
		kind := classifyGoExampleComment(literal, body)
		semantic, policy := "unclassified", "protected"
		structural := !jsonCode && !example && !open[start]
		for _, word := range englishWords(body) {
			if !a.CodeIdentifiers[word] && types.Universe.Lookup(word) == nil {
				structural = false
				break
			}
		}
		if !jsonCode && !example && !open[start] && isSafeGoFragment(body) {
			structural = true
		}
		switch kind {
		case goExampleCommentNatural, goExampleCommentNonNatural:
			if (isTranslatableTeachingComment(body) || kind == goExampleCommentNatural) && !structural {
				semantic = "teaching-comment"
			} else {
				semantic = "non-language/structural-comment"
			}
		default:
			semantic = "machine-semantic-comment"
		}
		if jsonCode && semantic == "machine-semantic-comment" {
			semantic = "unclassified"
		}
		if semantic == "teaching-comment" && !jsonCode && (open[start] || (example && kind == goExampleCommentNatural)) {
			policy = "translatable"
		}
		// Block comments remain protected in Page representation, as before.
		c := CodeComment{Position: len(b.Comments) + 1, TokenPosition: tokenPosition, Start: start, End: end, Kind: semantic, Policy: policy, Text: code[start:end], Body: body, Style: goCommentStyle(literal)}
		b.Comments = append(b.Comments, c)
		skeleton.WriteString(code[previous:start])
		skeleton.WriteString("⟦comment:" + c.Style + "⟧")
		previous = end
	}
	skeleton.WriteString(code[previous:])
	b.Skeleton = skeleton.String()
	if scanErr != "" || (jsonCode && !validJSONSchemaTokensMode(grammar, documentationJSON)) {
		b.Unsupported = true
		b.Diagnostic = "unsupported code grammar: " + scanErr
		b.Comments = []CodeComment{}
	}
	return b
}

type jsonSchemaToken struct {
	kind    token.Token
	literal string
}

func CommentSlotIdentity(prefix string, b CommentCodeBlock, c CodeComment) string {
	return fmt.Sprintf("%s/%s/comment/%d/token/%d/%s", prefix, b.Coordinate, c.Position, c.TokenPosition, c.Kind)
}

var commentEnglishPhrase = regexp.MustCompile(`[A-Za-z][A-Za-z0-9_]*(?:[ \t]+[A-Za-z][A-Za-z0-9_]*){2,}`)

// A retained source phrase with a teaching-language marker is a review hint,
// never a validator failure. Individual technical tokens are not evidence.
func commentResidueSuspect(source, target string) bool {
	for _, phrase := range commentEnglishPhrase.FindAllString(target, -1) {
		if strings.Contains(source, phrase) && isTranslatableTeachingComment(phrase) {
			return true
		}
	}
	return false
}

// JSON plus comments, trailing commas and schema type atoms used by frozen
// docs (string/number/integer/boolean/object/array). This grammar does not
// interpret quoted strings, URLs, keys, or type atoms as comments.
func validJSONSchemaTokens(ts []jsonSchemaToken) bool {
	return validJSONSchemaTokensMode(ts, false)
}

func validJSONSchemaTokensMode(ts []jsonSchemaToken, documentation bool) bool {
	i := 0
	var value func() bool
	value = func() bool {
		if i >= len(ts) {
			return false
		}
		t := ts[i]
		i++
		switch t.kind {
		case token.STRING:
			return strings.HasPrefix(t.literal, "\"")
		case token.INT, token.FLOAT:
			return true
		case token.SUB:
			return i < len(ts) && (ts[i].kind == token.INT || ts[i].kind == token.FLOAT) && func() bool { i++; return true }()
		case token.IDENT:
			switch t.literal {
			case "true", "false", "null", "string", "number", "integer", "boolean", "object", "array":
				return true
			}
			return false
		case token.LBRACE, token.LBRACK:
			close := token.RBRACE
			if t.kind == token.LBRACK {
				close = token.RBRACK
			}
			for i < len(ts) && ts[i].kind != close {
				if documentation && ts[i].kind == token.ELLIPSIS {
					i++
					if i < len(ts) && ts[i].kind == token.COMMA {
						i++
					}
					continue
				}
				if t.kind == token.LBRACE {
					if ts[i].kind != token.STRING {
						return false
					}
					i++
					if i >= len(ts) || ts[i].kind != token.COLON {
						return false
					}
					i++
				}
				if !value() || i >= len(ts) {
					return false
				}
				if ts[i].kind == close {
					break
				}
				if ts[i].kind != token.COMMA {
					return false
				}
				i++
			}
			if i >= len(ts) {
				return false
			}
			i++
			return true
		}
		return false
	}
	return value() && i == len(ts)
}

type CommentSlotComparison struct {
	SlotID                string `json:"slot_id"`
	CodeLanguage          string `json:"code_language"`
	SemanticKind          string `json:"semantic_kind"`
	SourceComment         string `json:"source_comment"`
	TargetComment         string `json:"target_comment"`
	TargetPresent         bool   `json:"target_present"`
	ExactSourceCarry      bool   `json:"exact_source_carry"`
	StructureMatch        bool   `json:"structure_match"`
	ProtectedSemantics    bool   `json:"protected_semantics"`
	ProtectedUnchanged    bool   `json:"protected_comment_unchanged"`
	Missing               bool   `json:"missing_comment_slot"`
	Extra                 bool   `json:"extra_comment_slot"`
	ReviewRequired        bool   `json:"review_required"`
	EnglishResidueSuspect bool   `json:"english_residue_suspect"`
}

func CommentBaseLanguage(locale string) (string, error) {
	if err := ValidateLocaleName(locale); err != nil {
		return "", err
	}
	return strings.ToLower(strings.Split(locale, "-")[0]), nil
}

// CompareCommentBlocks freezes source classification; target text never
// rediscovers eligibility. Structure errors are diagnostics, never fuzzy joins.
func CompareCommentBlocks(prefix, base string, source, target []CommentCodeBlock) ([]CommentSlotComparison, []string) {
	out, failures := []CommentSlotComparison{}, []string{}
	if len(source) != len(target) {
		failures = append(failures, "code block mapping count mismatch")
	}
	for i, s := range source {
		var t CommentCodeBlock
		if i < len(target) {
			t = target[i]
		}
		blockMatch := i < len(target) && s.Coordinate == t.Coordinate && s.Language == t.Language && s.Skeleton == t.Skeleton && !t.Unsupported
		if s.Unsupported {
			failures = append(failures, s.Coordinate+": "+s.Diagnostic)
			continue
		}
		if !blockMatch {
			failures = append(failures, s.Coordinate+": comment/code structure mismatch")
		}
		for j, c := range s.Comments {
			r := CommentSlotComparison{SlotID: CommentSlotIdentity(prefix, s, c), CodeLanguage: s.Language, SemanticKind: c.Kind, SourceComment: c.Text, StructureMatch: blockMatch, ProtectedSemantics: c.Kind != "teaching-comment"}
			if j < len(t.Comments) {
				a := t.Comments[j]
				r.TargetPresent = true
				r.TargetComment = a.Text
				r.StructureMatch = blockMatch && c.Style == a.Style && c.TokenPosition == a.TokenPosition
			}
			r.Missing = !r.TargetPresent
			r.ExactSourceCarry = r.TargetPresent && c.Text == r.TargetComment
			r.ProtectedUnchanged = !r.ProtectedSemantics || r.ExactSourceCarry
			// No language-correctness inference is made from unequal bytes.
			r.ReviewRequired = c.Kind == "teaching-comment" && base != "en" && r.ExactSourceCarry
			r.EnglishResidueSuspect = c.Kind == "teaching-comment" && base != "en" && !r.ExactSourceCarry && commentResidueSuspect(c.Text, r.TargetComment)
			out = append(out, r)
		}
		for j := len(s.Comments); j < len(t.Comments); j++ {
			out = append(out, CommentSlotComparison{SlotID: fmt.Sprintf("%s/%s/extra/%d", prefix, s.Coordinate, j+1), CodeLanguage: s.Language, TargetPresent: true, TargetComment: t.Comments[j].Text, Extra: true})
		}
	}
	if len(target) > len(source) {
		for _, t := range target[len(source):] {
			for j, c := range t.Comments {
				out = append(out, CommentSlotComparison{SlotID: fmt.Sprintf("%s/%s/extra-block/%d", prefix, t.Coordinate, j+1), CodeLanguage: t.Language, TargetPresent: true, TargetComment: c.Text, Extra: true})
			}
		}
	}
	return out, failures
}

// ValidateCommentCoverage is shared by diagnostics and the active current TU
// gate. Historical integrity does not acquire today's completeness policy.
func ValidateCommentCoverage(rows []CommentSlotComparison, failures []string, enforceCarry bool) error {
	if len(failures) > 0 {
		return fmt.Errorf("comment coverage unavailable: %s", strings.Join(failures, "; "))
	}
	for _, r := range rows {
		if r.Missing || r.Extra || !r.StructureMatch || !r.ProtectedUnchanged || r.SemanticKind == "unclassified" || (enforceCarry && r.ReviewRequired) {
			return fmt.Errorf("comment coverage failed: %s", r.SlotID)
		}
	}
	return nil
}
