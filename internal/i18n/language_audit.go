package i18n

import (
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
	"github.com/shuijingwan/go-tour-i18n/internal/languageview"
	"reflect"
	"strings"
)

var LanguageAuditAuthorityPaths = []string{"internal/languageaudit/audit.go", "internal/languageaudit/policy.go", "internal/languageview/html.go", "internal/languageview/identity.go", "internal/languageview/semantic_role.go", "internal/languageview/semantic_go_packages.go", "internal/i18n/language_audit_ui.go", "internal/i18n/language_audit.go", "internal/i18n/language_audit_tour.go", "internal/i18n/language_audit_surface.go", "internal/i18n/status.go"}

func LoadLanguageAuditPolicy(root, locale string, g *Glossary) (languageaudit.Policy, GlossaryArchiveReference, error) {
	var empty languageaudit.Policy
	if err := ValidateLocaleName(locale); err != nil {
		return empty, GlossaryArchiveReference{}, err
	}
	path := "locales/" + locale + "/locale.json"
	b, err := readCompatibilityFile(root, path)
	if err != nil {
		return empty, GlossaryArchiveReference{}, err
	}
	var m Locale
	if err := contentidentity.StrictJSON(b, &m); err != nil {
		return empty, GlossaryArchiveReference{}, err
	}
	if m.Locale != locale || m.HTMLLang != locale || m.LanguageAudit == nil {
		return empty, GlossaryArchiveReference{}, fmt.Errorf("locale %s requires explicit language_audit authority", locale)
	}
	if err := m.LanguageAudit.Validate(locale); err != nil {
		return empty, GlossaryArchiveReference{}, err
	}
	p := m.LanguageAudit.Policy
	if g != nil {
		p.ProtectedIdentities = append([]string(nil), g.Keep...)
	}
	return p, GlossaryArchiveReference{Path: path, SHA256: sum(b)}, nil
}

type LanguageAuditAdapter interface {
	LanguageAuditDocuments([]byte) (languageaudit.Document, languageaudit.Document, error)
}

func TranslationUnitLanguageAudit(root string, c *Catalog, u *TranslationUnit, locale string, target []byte, g *Glossary) (languageaudit.Result, error) {
	p, _, err := LoadLanguageAuditPolicy(root, locale, g)
	if err != nil {
		return languageaudit.Result{}, err
	}
	var source, localized languageaudit.Document
	if u.Adapter != nil {
		a, ok := u.Adapter.(LanguageAuditAdapter)
		if !ok {
			return languageaudit.Result{}, fmt.Errorf("adapter lacks normalized language audit view: %s", u.ID)
		}
		source, localized, err = a.LanguageAuditDocuments(target)
	} else {
		source, localized, err = NativeTourLanguageAuditDocuments(root, c, u, target)
	}
	if err != nil {
		return languageaudit.Result{}, err
	}
	return languageaudit.Scan(p, source, localized)
}
func NativeTourLanguageAuditDocuments(root string, c *Catalog, u *TranslationUnit, target []byte) (languageaudit.Document, languageaudit.Document, error) {
	source, localized, err := TourLanguageAuditFragments(root, c, u, target)
	if err != nil {
		return languageaudit.Document{}, languageaudit.Document{}, err
	}
	project := func(fs []TourAuditFragment) (languageaudit.Document, error) {
		d := languageaudit.Document{Owner: u.SourcePath, Blocks: []languageaudit.Block{}}
		for _, f := range fs {
			blocks := []languageaudit.Block{}
			if f.HTML != "" {
				var err error
				blocks, err = languageview.HTML([]byte(f.HTML), languageview.Options{})
				if err != nil {
					return d, err
				}
			} else {
				ps, err := languageaudit.BindIdentities(languageview.Plain(f.Text), f.Identities)
				if err != nil {
					return d, err
				}
				blocks = []languageaudit.Block{{Locator: "text", Context: f.Context, Pieces: ps}}
			}
			for _, b := range blocks {
				b.Locator = f.Locator + "/" + b.Locator
				if f.Context != "" {
					b.Context = f.Context
				}
				b.Start, b.End = f.Start, f.End
				d.Blocks = append(d.Blocks, b)
			}
		}
		return d, nil
	}
	s, err := project(source)
	if err != nil {
		return s, languageaudit.Document{}, err
	}
	t, err := project(localized)
	return s, t, err
}

const LanguageAttentionPath = "machine/language-review-candidates.json"
const LanguageAttentionSchema = "go-learning/language-review-attention/v1"

type LanguageAttentionCandidate struct {
	Locale         string                 `json:"locale"`
	Surface        string                 `json:"surface"`
	Owner          string                 `json:"owner"`
	UnitID         string                 `json:"unit_id,omitempty"`
	UnitKind       UnitKind               `json:"unit_kind,omitempty"`
	Class          string                 `json:"class"`
	Locator        string                 `json:"locator"`
	Reason         string                 `json:"reason"`
	Context        string                 `json:"bounded_context"`
	Classification languageaudit.Decision `json:"classification"`
}
type LanguageAttention struct {
	Glossary       GlossaryArchiveReference     `json:"glossary"`
	Schema         string                       `json:"schema"`
	Locale         string                       `json:"locale"`
	Policy         GlossaryArchiveReference     `json:"policy"`
	Implementation string                       `json:"implementation_identity_sha256"`
	Instruction    string                       `json:"instruction"`
	Candidates     []LanguageAttentionCandidate `json:"candidates"`
}

func NewLanguageAttention(root, locale string) (LanguageAttention, error) {
	_, ref, err := LoadLanguageAuditPolicy(root, locale, nil)
	if err != nil {
		return LanguageAttention{}, err
	}
	machine, err := TranslationMachineAuthority(root, nil)
	if err != nil {
		return LanguageAttention{}, err
	}
	gb, err := readCompatibilityFile(root, "locales/"+locale+"/glossary.yaml")
	if err != nil {
		return LanguageAttention{}, err
	}
	return LanguageAttention{Glossary: GlossaryArchiveReference{Path: "locales/" + locale + "/glossary.yaml", SHA256: sum(gb)}, Schema: LanguageAttentionSchema, Locale: locale, Policy: ref, Implementation: machine, Instruction: "review-candidate is not a finding or failure. Read complete source/target/glossary; do not review snippets alone.", Candidates: []LanguageAttentionCandidate{}}, nil
}
func (a *LanguageAttention) Add(surface, owner, id string, kind UnitKind, r languageaudit.Result) error {
	if err := languageaudit.Blocking(r); err != nil {
		return err
	}
	for _, d := range r.Diagnostics {
		if d.Decision != languageaudit.Candidate {
			continue
		}
		context := []rune(d.Evidence)
		if len(context) > 240 {
			context = context[:240]
		}
		a.Candidates = append(a.Candidates, LanguageAttentionCandidate{a.Locale, surface, owner, id, kind, d.Class, d.Locator, d.Class + " under formal locale policy", strings.TrimSpace(string(context)), languageaudit.Candidate})
	}
	return nil
}

func (a LanguageAttention) Verify(root string) error {
	current, err := NewLanguageAttention(root, a.Locale)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(a.Policy, current.Policy) || !reflect.DeepEqual(a.Glossary, current.Glossary) || a.Implementation != current.Implementation {
		return fmt.Errorf("language audit dependency changed during invocation")
	}
	return nil
}
