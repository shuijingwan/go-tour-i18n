package i18n

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
)

// CommentRevisionBody is a parser-owned coordinate, never a TranslationUnit.
// Its adapter protects/restores only the comment payload, including bindings.
type CommentRevisionBody struct {
	Coordinate string
	Position   int
	Start, End int
	Adapter    TranslationUnitAdapter
}

type CommentRevisionAdapter interface {
	CommentRevisionBodies([]byte) ([]CommentRevisionBody, error)
}

// The inventory and revision transport use the same scanner payload boundary.
func CommentPayloadBounds(comment string) (int, int) {
	a, z, _ := goCommentPayload([]byte(comment), 0, len(comment), comment)
	return a, z
}

type CommentRevisionTransportSlot struct {
	SlotID         string   `json:"slot_id"`
	Start          int      `json:"start"`
	End            int      `json:"end"`
	Text           string   `json:"protected_text"`
	Tokens         []string `json:"tokens"`
	OriginalTokens []string `json:"original_tokens"`
	body           TranslationUnitAdapter
}

// Derived Bundle input only: no new output schema, receipt or lifecycle state.
type CommentRevisionTransport struct {
	Schema          string                         `json:"schema"`
	UnitID          string                         `json:"unit_id"`
	SourceSHA256    string                         `json:"source_sha256"`
	CandidateSHA256 string                         `json:"candidate_sha256"`
	Reopen          GlossaryArchiveReference       `json:"reopen"`
	Slots           []CommentRevisionTransportSlot `json:"slots"`
	Boundaries      []string                       `json:"boundaries"`
	Values          []string                       `json:"protected_values"`
	Text            string                         `json:"text"`
}

func buildCommentRevisionTransport(root string, c *Catalog, g *Glossary, locale string, u *TranslationUnit, r *QualityCheckSurfaceReopen, candidate []byte) (*CommentRevisionTransport, error) {
	if r == nil || r.AuthoritySource != CurrentValidationAuthority || r.CurrentValidation == nil || u.Kind != UnitKindPage {
		return nil, fmt.Errorf("comment revision requires current-validation Page authority")
	}
	var finding *CurrentValidationReopenFinding
	for i := range r.CurrentValidation.Findings {
		if r.CurrentValidation.Findings[i].UnitID == u.ID {
			if finding != nil {
				return nil, fmt.Errorf("duplicate per-Unit comment authority")
			}
			finding = &r.CurrentValidation.Findings[i]
		}
	}
	if finding == nil {
		return nil, fmt.Errorf("missing per-Unit current-validation authority")
	}
	if len(finding.CommentSlots) == 0 {
		if strings.HasPrefix(finding.Reason, "code-comment gate:") {
			return nil, fmt.Errorf("comment finding has no exact slot authority")
		}
		return nil, nil
	}
	rows, gateErr := ValidateTranslationUnitCommentCoverage(root, c, u, locale, candidate)
	var gate *CommentGateError
	if !errors.As(gateErr, &gate) || gate.Reason != "non-English teaching-comment exact source carry" {
		return nil, fmt.Errorf("comment revision requires exact carry failure without structural defects")
	}
	bad := []CommentSlotComparison{}
	for _, row := range rows {
		if row.ReviewRequired {
			bad = append(bad, row)
		}
	}
	if !reflect.DeepEqual(bad, finding.CommentSlots) || ValidateCommentCoverage(rows, nil, false) != nil || finding.Reason != gateErr.Error() {
		return nil, fmt.Errorf("comment revision exact finding/slot scope mismatch")
	}
	a, ok := u.Adapter.(CommentRevisionAdapter)
	if !ok {
		return nil, fmt.Errorf("Page adapter lacks comment revision coordinates")
	}
	bodies, err := a.CommentRevisionBodies(candidate)
	if err != nil {
		return nil, err
	}
	ca := u.Adapter.(TranslationCommentAdapter)
	source, title, err := ca.CommentSourceBlocks()
	if err != nil {
		return nil, err
	}
	wanted := map[string]CommentSlotComparison{}
	for _, row := range bad {
		wanted[row.SlotID] = row
	}
	coordinates := map[string]string{}
	for _, block := range source {
		block = SourceCommentBlockDisposition(block, title, nil)
		for _, comment := range block.Comments {
			id := CommentSlotIdentity(r.CurrentValidation.Package+"/"+u.SourcePath+"/"+string(u.Kind)+"/"+u.ID, block, comment)
			if _, ok := wanted[id]; ok {
				if comment.Kind != "teaching-comment" {
					return nil, fmt.Errorf("protected comment cannot be exposed")
				}
				coordinates[fmt.Sprintf("%s/%d", block.Coordinate, comment.Position)] = id
			}
		}
	}
	p := &CommentRevisionTransport{Schema: "go-learning/comment-revision-transport/v1", UnitID: u.ID, SourceSHA256: u.SourceSHA256, CandidateSHA256: sum(candidate), Reopen: r.revisionEvidence()}
	next := 0
	token := func() string {
		next++
		return fmt.Sprintf("⟪GTI18N_%s_%06d⟫", sum(mustJSON([]string{u.ID, p.CandidateSHA256, p.Reopen.SHA256}))[:8], next)
	}
	pos := 0
	seen := map[string]bool{}
	for _, body := range bodies {
		id, ok := coordinates[fmt.Sprintf("%s/%d", body.Coordinate, body.Position)]
		if !ok {
			continue
		}
		if seen[id] || body.Start < pos || body.End <= body.Start || body.End > len(candidate) || body.Adapter == nil {
			return nil, fmt.Errorf("ambiguous comment revision coordinate")
		}
		seen[id] = true
		left, right := CommentPayloadBounds(wanted[id].TargetComment)
		if string(candidate[body.Start:body.End]) != wanted[id].TargetComment[left:right] {
			return nil, fmt.Errorf("comment payload/source coordinate mismatch")
		}
		text, tokens, err := body.Adapter.ProtectedInput(g)
		if err != nil {
			return nil, err
		}
		check, err := body.Adapter.Restore(text)
		if err != nil || !bytes.Equal(check, candidate[body.Start:body.End]) {
			return nil, fmt.Errorf("comment body projection is not exact")
		}
		boundary := token()
		p.Boundaries = append(p.Boundaries, boundary)
		p.Values = append(p.Values, string(candidate[pos:body.Start]))
		p.Text += boundary
		slot := CommentRevisionTransportSlot{SlotID: id, Start: body.Start, End: body.End, Text: text, OriginalTokens: tokens, body: body.Adapter}
		replacements := []string{}
		for _, old := range tokens {
			fresh := token()
			slot.Tokens = append(slot.Tokens, fresh)
			replacements = append(replacements, old, fresh)
		}
		if len(replacements) > 0 {
			slot.Text = strings.NewReplacer(replacements...).Replace(text)
		}
		p.Text += slot.Text
		p.Slots = append(p.Slots, slot)
		pos = body.End
	}
	if len(p.Slots) != len(wanted) || len(coordinates) != len(wanted) {
		return nil, fmt.Errorf("authorized comment hidden or duplicated in transport")
	}
	p.Boundaries = append(p.Boundaries, token())
	p.Values = append(p.Values, string(candidate[pos:]))
	p.Text += p.Boundaries[len(p.Boundaries)-1]
	return p, nil
}

func (p *CommentRevisionTransport) ProtectedInput(*Glossary) (string, []string, error) {
	tokens := append([]string(nil), p.Boundaries...)
	for _, s := range p.Slots {
		tokens = append(tokens, s.Tokens...)
	}
	return p.Text, tokens, nil
}
func (p *CommentRevisionTransport) Validate([]byte, *Glossary) error {
	return fmt.Errorf("transport is not a formal validator")
}
func (p *CommentRevisionTransport) Restore(output string) ([]byte, error) {
	output = strings.TrimSuffix(output, "\n") // existing single-LF artifact framing
	_, tokens, _ := p.ProtectedInput(nil)
	if len(translationTokenRE.FindAllString(output, -1)) != len(tokens) {
		return nil, fmt.Errorf("comment transport token exact-set mismatch")
	}
	for _, t := range tokens {
		if strings.Count(output, t) != 1 {
			return nil, fmt.Errorf("comment transport token missing/duplicated: %s", t)
		}
	}
	if !strings.HasPrefix(output, p.Boundaries[0]) || !strings.HasSuffix(output, p.Boundaries[len(p.Boundaries)-1]) {
		return nil, fmt.Errorf("comment transport external text forbidden")
	}
	var restored strings.Builder
	for i, s := range p.Slots {
		start := strings.Index(output, p.Boundaries[i]) + len(p.Boundaries[i])
		end := strings.Index(output, p.Boundaries[i+1])
		if start > end {
			return nil, fmt.Errorf("comment transport boundary order changed")
		}
		text := output[start:end]
		pairs := []string{}
		for j, t := range s.Tokens {
			if strings.Count(text, t) != 1 {
				return nil, fmt.Errorf("comment binding crossed slot")
			}
			pairs = append(pairs, t, s.OriginalTokens[j])
		}
		if len(pairs) > 0 {
			text = strings.NewReplacer(pairs...).Replace(text)
		}
		b, err := s.body.Restore(text)
		if err != nil {
			return nil, err
		}
		restored.WriteString(p.Values[i])
		restored.Write(b)
	}
	restored.WriteString(p.Values[len(p.Values)-1])
	return []byte(restored.String()), nil
}

// Inputs retain their immutable source representation. Only the response
// decoder is selected by the existing exact revision authority.
func prepareRetranslationOutput(root string, c *Catalog, g *Glossary, locale string, selected []RetranslationBatchUnit, record RetranslationBatchUnit) (protectedTranslation, error) {
	u, err := c.Unit(record.UnitID)
	if err != nil {
		return protectedTranslation{}, err
	}
	if p, lineage, err := surfaceTransportOutput(root, c, locale, record); err != nil || lineage != nil {
		return p, err
	}
	if record.RepresentationBridge != nil {
		return bridgeOutput(root, c, g, locale, record)
	}
	if record.RevisionFeedbackSource == CurrentValidationRevisionFeedback {
		projections := map[string]protectedTranslation{}
		_, err := revisionGenerationInputs(root, c, locale, selected, projections)
		if err != nil {
			return protectedTranslation{}, err
		}
		if p := projections[record.UnitID]; p.Adapter != nil {
			return p, nil
		}
	}
	return prepareTranslationUnitInput(u, g)
}

func commentTransportPath(record RetranslationBatchUnit) string {
	return "formal/comment-revision-transport/" + filepath.Base(record.InputPath) + ".json"
}
