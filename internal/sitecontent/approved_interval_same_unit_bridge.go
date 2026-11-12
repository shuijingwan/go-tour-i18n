package sitecontent

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// A response-only composition, never a new formal source or target protector.
// The approved whole parent remains opaque: its newly protected children have
// no historical target token correspondence and must not be guessed.
type approvedOrdinaryBridge struct {
	old                 PageAdapter
	text                string
	tokens              []string
	parentToken, parent string
	left, right         string
}

func (p *approvedOrdinaryBridge) ProtectedInput(*i18n.Glossary) (string, []string, error) {
	return p.text, append([]string(nil), p.tokens...), nil
}
func (p *approvedOrdinaryBridge) UnrestrictedRevisionTransport() bool { return true }
func (p *approvedOrdinaryBridge) Validate([]byte, *i18n.Glossary) error {
	return fmt.Errorf("bridge response is not a formal validator")
}
func (p *approvedOrdinaryBridge) Restore(raw string) ([]byte, error) {
	if strings.Count(raw, p.parentToken) != 1 {
		return nil, fmt.Errorf("approved bridge opaque parent missing/duplicate")
	}
	start, end := 0, len(raw)
	if p.left != "" {
		if strings.Count(raw, p.left) != 1 {
			return nil, fmt.Errorf("approved bridge left source boundary changed")
		}
		start = strings.Index(raw, p.left) + len(p.left)
	}
	if p.right != "" {
		if strings.Count(raw, p.right) != 1 {
			return nil, fmt.Errorf("approved bridge right source boundary changed")
		}
		end = strings.Index(raw, p.right)
	}
	if start > end || raw[start:end] != p.parentToken {
		return nil, fmt.Errorf("approved bridge parent moved or edited")
	}
	return p.old.Restore(strings.Replace(raw, p.parentToken, p.parent, 1))
}

func (a PageAdapter) BridgeApprovedIntervalRevision(projection i18n.ApprovedIntervalProjection, raw, reviewed []byte) (i18n.TranslationUnitAdapter, []byte, []byte, error) {
	typed, _, decoder, err := derivedApprovedCodec(a, projection.Codec, projection.Children)
	if err != nil {
		return nil, nil, nil, err
	}
	p := typed.(approvedPageCodec)
	if projection.Namespace != "GTI18N_AI1/v1" || len(a.Document.Units) != 1 || len(p.Old.Document.Units) != 1 || projection.UnitID != a.Document.Units[0].ID || projection.SourceSHA256 != digest(a.Source) || !bytes.Equal(a.Source, p.Old.Source) || projection.OldRepresentationSHA256 != identity(p.Old.Document) || projection.CurrentRepresentationSHA256 != identity(a.Document) || projection.ParentToken != p.ParentToken || projection.Text != bridgeStringEOF(p.Text) || !reflect.DeepEqual(projection.Tokens, p.Tokens) {
		return nil, nil, nil, fmt.Errorf("approved bridge source/representation binding mismatch")
	}
	baseline, err := p.Old.Restore(p.Baseline)
	if err != nil || digest(bridgeEOF(baseline)) != projection.CandidateSHA256 {
		return nil, nil, nil, fmt.Errorf("approved bridge baseline candidate mismatch")
	}
	got, err := decoder.Restore(string(raw))
	if err != nil || !bytes.Equal(bridgeEOF(got), bridgeEOF(reviewed)) {
		return nil, nil, nil, fmt.Errorf("approved bridge reviewed inverse mismatch: %v", err)
	}
	// Decode response boundaries/tokens only, using the original exact inverse.
	// No search for candidate words or technical-atom occurrences is performed.
	response := strings.TrimSuffix(string(raw), "\n")
	var ordinary strings.Builder
	for i, s := range p.Slots {
		start := strings.Index(response, p.Boundaries[i]) + len(p.Boundaries[i])
		end := strings.Index(response, p.Boundaries[i+1])
		pairs := []string{}
		for j, t := range s.Tokens {
			pairs = append(pairs, t, s.OldTokens[j])
		}
		ordinary.WriteString(p.Values[i])
		ordinary.WriteString(strings.NewReplacer(pairs...).Replace(response[start:end]))
	}
	ordinary.WriteString(p.Values[len(p.Values)-1])
	responsePage, ordinaryBytes, err := p.Old.bridgeReleaseNaturalAtoms(a, []byte(ordinary.String()))
	if err != nil {
		return nil, nil, nil, err
	}
	source, err := protectionPieces(responsePage.Document.Units[0])
	if err != nil {
		return nil, nil, nil, err
	}
	current, err := protectionPieces(a.Document.Units[0])
	if err != nil {
		return nil, nil, nil, err
	}
	parent := -1
	for i, piece := range source {
		if piece.token == nil && piece.start == projection.ParentSource.Start && piece.end == projection.ParentSource.End {
			if parent >= 0 {
				return nil, nil, nil, fmt.Errorf("ambiguous approved parent")
			}
			parent = i
		}
	}
	if projection.ParentSource.Space != "canonical-source" || projection.ParentSource.Unit != "byte" || parent < 0 {
		return nil, nil, nil, fmt.Errorf("approved bridge parent source coordinate missing")
	}
	for _, child := range current {
		if child.token == nil || (child.start >= projection.ParentSource.Start && child.end <= projection.ParentSource.End) {
			continue
		}
		covered := false
		for _, prior := range source {
			if prior.token != nil && child.start >= prior.start && child.end <= prior.end {
				covered = true
			}
		}
		if !covered {
			return nil, nil, nil, fmt.Errorf("approved bridge unbound protection outside parent")
		}
	}
	if projection.ParentCandidate.Space != "candidate-markdown" || projection.ParentCandidate.Unit != "byte" || projection.ParentCandidate.Start < 0 || projection.ParentCandidate.End <= projection.ParentCandidate.Start || projection.ParentCandidate.End > len(baseline) || digest(baseline[projection.ParentCandidate.Start:projection.ParentCandidate.End]) != projection.ParentSHA256 {
		return nil, nil, nil, fmt.Errorf("approved bridge parent candidate coordinate mismatch")
	}
	// Source-owned adjacent tokens delimit this complete historical prose piece.
	// Their identity/uniqueness was already proved by the old decoder above.
	ordinaryRaw := string(ordinaryBytes)
	start, end := 0, len(ordinaryRaw)
	if parent > 0 {
		start = strings.Index(ordinaryRaw, source[parent-1].text) + len(source[parent-1].text)
	}
	if parent+1 < len(source) {
		end = strings.Index(ordinaryRaw, source[parent+1].text)
	}
	if start > end || digest([]byte(ordinaryRaw[start:end])) != projection.ParentSHA256 {
		return nil, nil, nil, fmt.Errorf("approved bridge frozen parent drift")
	}
	token := "⟦GTI18N_BRIDGE_" + digest([]byte(projection.UnitID + "\n" + projection.ParentSHA256))[:16] + "⟧"
	if strings.Contains(ordinaryRaw, token) {
		return nil, nil, nil, fmt.Errorf("approved bridge parent token collision")
	}
	text := ordinaryRaw[:start] + token + ordinaryRaw[end:]
	tokens := []string{}
	for _, t := range responsePage.Document.Units[0].Protected {
		tokens = append(tokens, t.Token)
	}
	tokens = append(tokens, token)
	out := &approvedOrdinaryBridge{old: responsePage, text: text, tokens: tokens, parentToken: token, parent: ordinaryRaw[start:end]}
	if parent > 0 {
		out.left = source[parent-1].text
	}
	if parent+1 < len(source) {
		out.right = source[parent+1].text
	}
	restored, err := out.Restore(text)
	if err != nil || !bytes.Equal(bridgeEOF(restored), bridgeEOF(reviewed)) {
		return nil, nil, nil, fmt.Errorf("approved bridge ordinary restore mismatch: %v", err)
	}
	return out, bridgeEOF([]byte(text)), []byte(p.Baseline), nil
}

// The source owner gives codec/inventory their typed meaning, including through
// existing Page decorators. Lifecycle code never normalizes arbitrary JSON.
func (a PageAdapter) CompareApprovedIntervalBridge(old, current i18n.ApprovedIntervalProjection) error {
	ac, ai, _, err := derivedApprovedCodec(a, old.Codec, old.Children)
	if err != nil {
		return err
	}
	bc, bi, _, err := derivedApprovedCodec(a, current.Codec, current.Children)
	if err != nil {
		return err
	}
	old.Codec, current.Codec = nil, nil
	old.Children, current.Children = nil, nil
	if !reflect.DeepEqual(old, current) || !reflect.DeepEqual(ac, bc) || !reflect.DeepEqual(ai, bi) {
		return fmt.Errorf("approved bridge source-owned semantic projection changed")
	}
	return nil
}

// Existing observation decorators are not historical representation identities.
// Keep the exact registered source/package world and its real underlying gates.
func (a PageAdapter) ApprovedIntervalBridgeCatalog(c *i18n.Catalog) *i18n.Catalog {
	return originalBridgeCatalog(c)
}
