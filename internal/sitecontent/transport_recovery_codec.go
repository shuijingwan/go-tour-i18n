package sitecontent

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// These values are owned by the source representation, just like the inverse
// codec itself. The lifecycle must not assign meaning to RawMessage spelling.
type derivedApprovedChild struct {
	SourceStart int            `json:"source_start"`
	SourceEnd   int            `json:"source_end"`
	Identity    string         `json:"identity"`
	Role        string         `json:"role"`
	Protection  ProtectedToken `json:"protection"`
}

func derivedApprovedCodec(adapter i18n.TranslationUnitAdapter, codec, inventory []byte) (any, any, i18n.TranslationUnitAdapter, error) {
	a, ok := adapter.(PageAdapter)
	if !ok {
		return nil, nil, nil, fmt.Errorf("unsupported source-owned derived inverse reader")
	}
	var p approvedPageCodec
	var children []derivedApprovedChild
	if err := contentidentity.StrictJSON(codec, &p); err != nil {
		return nil, nil, nil, err
	}
	if err := contentidentity.StrictJSON(inventory, &children); err != nil {
		return nil, nil, nil, err
	}
	if p.Schema != "go-learning/approved-interval-response/v1" || len(p.Slots) == 0 || len(p.Values) != len(p.Slots)+1 || len(p.Boundaries) != len(p.Values) || p.ParentToken == "" {
		return nil, nil, nil, fmt.Errorf("invalid approved response inverse contract")
	}
	last := 0
	for _, s := range p.Slots {
		if s.RawStart < last || s.RawEnd <= s.RawStart || s.RawEnd > len(p.Baseline) || len(s.Tokens) != len(s.OldTokens) {
			return nil, nil, nil, fmt.Errorf("invalid approved inverse slot")
		}
		pairs := []string{}
		for i, token := range s.Tokens {
			pairs = append(pairs, token, s.OldTokens[i])
		}
		if strings.NewReplacer(pairs...).Replace(s.Text) != p.Baseline[s.RawStart:s.RawEnd] {
			return nil, nil, nil, fmt.Errorf("invalid approved inverse slot token binding")
		}
		last = s.RawEnd
	}
	for _, c := range children {
		if c.SourceStart < 0 || c.SourceEnd <= c.SourceStart || c.SourceEnd > len(a.Source) || c.Identity == "" || c.Role == "" {
			return nil, nil, nil, fmt.Errorf("invalid approved source child")
		}
	}
	restored, err := p.Restore(p.Text)
	want, oldErr := p.Old.Restore(p.Baseline)
	if err != nil || oldErr != nil || !bytes.Equal(restored, want) {
		return nil, nil, nil, fmt.Errorf("approved inverse baseline mismatch")
	}
	return p, children, &p, nil
}
