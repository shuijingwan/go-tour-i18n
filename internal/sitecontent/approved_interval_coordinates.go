package sitecontent

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	gmtext "github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type approvedTraceWriter struct{ bytes.Buffer }

func (w *approvedTraceWriter) Available() int { return int(^uint(0)>>1) - w.Len() }
func (w *approvedTraceWriter) Buffered() int  { return w.Len() }
func (w *approvedTraceWriter) Flush() error   { return nil }

type approvedRenderRange struct {
	HTMLStart, HTMLEnd, SourceStart, SourceEnd int
	Kind                                       string
}
type approvedTraceRenderer struct {
	ranges []approvedRenderRange
	open   map[ast.Node]int
	offset int
}
type approvedTraceRegister struct {
	register renderer.NodeRendererFuncRegisterer
	owner    *approvedTraceRenderer
}

func (r *approvedTraceRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	gmhtml.NewRenderer(gmhtml.WithUnsafe()).RegisterFuncs(approvedTraceRegister{reg, r})
}
func (r approvedTraceRegister) Register(kind ast.NodeKind, f renderer.NodeRendererFunc) {
	if kind != ast.KindParagraph && kind != ast.KindHeading && kind != ast.KindTextBlock {
		return
	}
	r.register.Register(kind, func(w util.BufWriter, source []byte, n ast.Node, enter bool) (ast.WalkStatus, error) {
		trace, ok := w.(*approvedTraceWriter)
		if !ok {
			return ast.WalkStop, fmt.Errorf("approved interval renderer lost byte coordinate owner")
		}
		if enter {
			r.owner.open[n] = trace.Len()
		}
		status, err := f(w, source, n, enter)
		if !enter && err == nil {
			lines := n.Lines()
			if lines.Len() == 0 {
				return ast.WalkStop, fmt.Errorf("approved interval Markdown block has no source coordinates")
			}
			start, end := lines.At(0).Start, lines.At(lines.Len()-1).Stop
			// Cover the entire parser-owned block, including its Markdown delimiters
			// and local spacing context, rather than narrowing to a visible occurrence.
			for start > 0 && source[start-1] != '\n' {
				start--
			}
			for end < len(source) && source[end] != '\n' {
				end++
			}
			if end < len(source) {
				end++
			}
			r.owner.ranges = append(r.owner.ranges, approvedRenderRange{r.owner.open[n], trace.Len(), start + r.owner.offset, end + r.owner.offset, n.Kind().String()})
		}
		return status, err
	})
}
func approvedMarkdownEngine() goldmark.Markdown {
	return goldmark.New(goldmark.WithParserOptions(parser.WithHeadingAttribute(), parser.WithAutoHeadingID()), goldmark.WithRendererOptions(gmhtml.WithUnsafe()), goldmark.WithExtensions(extension.Typographer, extension.Linkify, extension.DefinitionList, extension.Table))
}

func (a PageAdapter) approvedFindingCoordinates(candidate []byte, audit languageaudit.Result) ([]i18n.ApprovedIntervalFindingRegion, error) {
	if languageaudit.Blocking(audit) == nil {
		return nil, fmt.Errorf("approved interval missing machine finding")
	}
	if !strings.HasSuffix(a.Document.Path, ".md") {
		return nil, fmt.Errorf("approved interval unsupported typed coordinate syntax")
	}
	offset, err := parseMetadata(candidate, func(int, int, string) {})
	if err != nil {
		return nil, err
	}
	source := candidate[offset:]
	tracing := &approvedTraceRenderer{open: map[ast.Node]int{}, offset: offset}
	md := approvedMarkdownEngine()
	md.Renderer().AddOptions(renderer.WithNodeRenderers(util.Prioritized(tracing, 500)))
	tree := md.Parser().Parse(gmtext.NewReader(source))
	w := &approvedTraceWriter{}
	if err = md.Renderer().Render(w, source, tree); err != nil {
		return nil, err
	}
	// The traced renderer must produce the exact canonical Markdown audit HTML.
	var canonical bytes.Buffer
	if err = approvedMarkdownEngine().Convert(source, &canonical); err != nil || !bytes.Equal(canonical.Bytes(), w.Bytes()) {
		return nil, fmt.Errorf("approved interval renderer conversion differs from canonical audit")
	}
	htmlBlocks, err := auditHTMLBlocks(w.Bytes(), false)
	if err != nil {
		return nil, err
	}
	fullView, err := ProjectLanguageAudit(a.Document.Path, candidate, AuditMarkdown)
	if err != nil {
		return nil, err
	}
	byLocator := map[string]languageaudit.Block{}
	for _, b := range htmlBlocks {
		b.Locator = "markdown-rendered/" + b.Locator
		byLocator[b.Locator] = b
	}
	canonicalBlocks := map[string]languageaudit.Block{}
	for _, b := range fullView.Blocks {
		canonicalBlocks[b.Locator] = b
	}
	out := []i18n.ApprovedIntervalFindingRegion{}
	for _, d := range audit.Diagnostics {
		if d.Decision != languageaudit.Defect {
			continue
		}
		// Other defect classes need an independently defined complete edit footprint.
		if d.Class != "script-adjacency" || d.Owner != a.Document.Path || d.Start != -1 || d.End != -1 {
			return nil, fmt.Errorf("approved interval finding has unsupported/incomplete typed edit footprint")
		}
		block, ok := byLocator[d.Locator]
		original, ok2 := canonicalBlocks[d.Locator]
		if !ok || !ok2 || !reflect.DeepEqual(block.Pieces, original.Pieces) || block.Context != original.Context || d.VisibleStart < 0 || d.VisibleEnd <= d.VisibleStart {
			return nil, fmt.Errorf("approved interval typed diagnostic/rendered binding mismatch")
		}
		var selected *approvedRenderRange
		for i := range tracing.ranges {
			r := &tracing.ranges[i]
			if block.Start >= r.HTMLStart && block.End <= r.HTMLEnd {
				if selected != nil {
					return nil, fmt.Errorf("approved interval coordinate conversion ambiguous")
				}
				selected = r
			}
		}
		if selected == nil || selected.SourceStart < offset || selected.SourceEnd > len(candidate) || selected.SourceEnd <= selected.SourceStart {
			return nil, fmt.Errorf("approved interval coordinate conversion unknown")
		}
		binding := struct {
			CandidateSHA string
			Diagnostic   languageaudit.Diagnostic
			Block        languageaudit.Block
			Range        approvedRenderRange
			HTMLSHA      string
		}{digest(candidate), d, block, *selected, digest(w.Bytes())}
		out = append(out, i18n.ApprovedIntervalFindingRegion{Diagnostic: d, Visible: i18n.ApprovedIntervalCoordinate{Space: d.Locator, Unit: "rune", Start: d.VisibleStart, End: d.VisibleEnd}, Rendered: i18n.ApprovedIntervalCoordinate{Space: "canonical-markdown-rendered-html", Unit: "byte", Start: block.Start, End: block.End}, Candidate: i18n.ApprovedIntervalCoordinate{Space: "candidate-markdown", Unit: "byte", Start: selected.SourceStart, End: selected.SourceEnd}, ConversionSHA256: identity(binding)})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("approved interval editable footprint incomplete")
	}
	return out, nil
}
