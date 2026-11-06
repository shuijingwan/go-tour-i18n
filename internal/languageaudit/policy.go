package languageaudit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"io"
	"strings"
)

const PolicySchema = "go-learning/locale-language-audit-policy/v1"

type Authority struct {
	Schema  string `json:"schema"`
	Version int    `json:"policy_version"`
	Policy  Policy `json:"policy"`
}

func (a Authority) Validate(locale string) error {
	if a.Schema != PolicySchema || a.Version < 1 || a.Policy.Locale != locale || len(a.Policy.ProtectedIdentities) != 0 {
		return fmt.Errorf("invalid locale language policy authority")
	}
	return a.Policy.validate()
}
func DecodeAuthority(b []byte, locale string) (Authority, error) {
	var a Authority
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&a); err != nil {
		return a, err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return a, fmt.Errorf("policy must contain one JSON value")
	}
	return a, a.Validate(locale)
}

type DefectError struct{ Diagnostics []Diagnostic }

func (e *DefectError) Error() string {
	items := []string{}
	for _, d := range e.Diagnostics {
		items = append(items, fmt.Sprintf("%s %s %s [%d:%d]", d.Class, d.Owner, d.Locator, d.VisibleStart, d.VisibleEnd))
	}
	return "language audit deterministic defects: " + strings.Join(items, "; ")
}
func Blocking(r Result) error {
	defects := []Diagnostic{}
	for _, d := range r.Diagnostics {
		if d.Decision == Defect {
			defects = append(defects, d)
		}
	}
	if len(defects) > 0 {
		return &DefectError{Diagnostics: defects}
	}
	return nil
}

// Required fields distinguish an explicit empty rule set from omitted policy.
func (a *Authority) UnmarshalJSON(b []byte) error {
	type wire Authority
	var w wire
	if err := contentidentity.StrictJSON(b, &w); err != nil {
		return err
	}
	var envelope struct {
		Policy map[string]json.RawMessage `json:"policy"`
	}
	if err := json.Unmarshal(b, &envelope); err != nil {
		return err
	}
	for _, key := range []string{"locale", "profile", "prose_scripts", "spacing", "sentence_end", "closing_quotes", "sentence_terminators", "forbidden_clause_punctuation", "heading_symbols", "residual_min_words", "punctuation_only_paragraph_is_defect"} {
		v, ok := envelope.Policy[key]
		if !ok || bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return fmt.Errorf("explicit language policy field required: %s", key)
		}
	}
	*a = Authority(w)
	return nil
}

func (a Authority) MarshalJSON() ([]byte, error) {
	type wire Authority
	p := a.Policy
	if p.Spacing == nil {
		p.Spacing = []SpacingRule{}
	}
	if p.SentenceEnd == nil {
		p.SentenceEnd = map[string]string{}
	}
	if p.ForbiddenClausePunctuation == nil {
		p.ForbiddenClausePunctuation = []string{}
	}
	a.Policy = p
	return json.Marshal(wire(a))
}
