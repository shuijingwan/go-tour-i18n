package sitecontent

import (
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
	"path/filepath"
	"sort"
)

func languageAuditSyntax(path string) (LanguageAuditSyntax, error) {
	switch filepath.Ext(path) {
	case ".md":
		return AuditMarkdown, nil
	case ".html":
		return AuditHTML, nil
	case ".tmpl":
		return AuditTemplate, nil
	case ".yaml":
		return AuditShellYAML, nil
	}
	return "", fmt.Errorf("unsupported language representation %s", path)
}
func formalLanguagePair(path string, source, target []byte) (languageaudit.Document, languageaudit.Document, error) {
	syntax, err := languageAuditSyntax(path)
	if err != nil {
		return languageaudit.Document{}, languageaudit.Document{}, err
	}
	s, err := ProjectLanguageAudit(path, source, syntax)
	if err != nil {
		return s, languageaudit.Document{}, err
	}
	t, err := ProjectLanguageAudit(path, target, syntax)
	return s, t, err
}
func (a PageAdapter) LanguageAuditDocuments(target []byte) (languageaudit.Document, languageaudit.Document, error) {
	return formalLanguagePair(a.Document.Path, a.Source, target)
}
func (a ShellPageAdapter) LanguageAuditDocuments(target []byte) (languageaudit.Document, languageaudit.Document, error) {
	s, err := projectShellPageLanguageAudit(a.Source.Path, a.Raw)
	if err != nil {
		return s, languageaudit.Document{}, err
	}
	t, err := projectShellPageLanguageAudit(a.Source.Path, target)
	return s, t, err
}
func scanFormalLanguagePair(root, locale, path string, source, target []byte) (languageaudit.Result, error) {
	g, err := i18n.LoadGlossary(root, locale)
	if err != nil {
		return languageaudit.Result{}, err
	}
	p, _, err := i18n.LoadLanguageAuditPolicy(root, locale, g)
	if err != nil {
		return languageaudit.Result{}, err
	}
	s, t, err := formalLanguagePair(path, source, target)
	if err != nil {
		return languageaudit.Result{}, err
	}
	return languageaudit.Scan(p, s, t)
}

func structuredLanguageAttention(root string, p StructuredPlan, raw, files map[string][]byte) (i18n.LanguageAttention, error) {
	attention, err := i18n.NewLanguageAttention(root, p.Locale)
	if err != nil {
		return attention, err
	}
	glossary, err := i18n.LoadGlossary(root, p.Locale)
	if err != nil {
		return attention, err
	}
	policy, _, err := i18n.LoadLanguageAuditPolicy(root, p.Locale, glossary)
	if err != nil {
		return attention, err
	}
	for _, a := range p.Assets {
		target, ok := files[a.TargetPath]
		if !ok {
			continue
		}
		if a.Document != nil {
			source, ok := raw[a.Document.Path]
			if !ok {
				return attention, fmt.Errorf("missing exact structured language source %s", a.ID)
			}
			result, err := scanFormalLanguageDocument(root, p.Locale, *a.Document, source, target)
			if err != nil {
				return attention, err
			}
			if err := attention.Add("structured", a.Document.Path, "", "", result); err != nil {
				return attention, err
			}
			continue
		}
		switch a.ID {
		case "tour-ui":
			source, err := readRegular(root, "internal/tour/ui/en.json")
			if err != nil {
				return attention, err
			}
			var src, dst ui.Catalog
			if err := ui.ValidateLocalizedBytes(p.Locale, source, target); err != nil {
				return attention, err
			}
			if err := StrictJSON(source, &src); err != nil {
				return attention, err
			}
			if err := StrictJSON(target, &dst); err != nil {
				return attention, err
			}
			keys := []string{}
			for k := range src.Messages {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, key := range keys {
				s, err := ProjectTourUIMessageAudit(a.TargetPath, key, src.Messages[key], src.Messages[key])
				if err != nil {
					return attention, err
				}
				t, err := ProjectTourUIMessageAudit(a.TargetPath, key, dst.Messages[key], src.Messages[key])
				if err != nil {
					return attention, err
				}
				r, err := languageaudit.Scan(policy, s, t)
				if err != nil {
					return attention, err
				}
				if err := attention.Add("tour-ui", a.TargetPath, "", "", r); err != nil {
					return attention, err
				}
			}
		case "article-metadata":
			c, err := checkedCatalog(root)
			if err != nil {
				return attention, err
			}
			src, err := i18n.ReadTourArticleAuditHeaders(root, c)
			if err != nil {
				return attention, err
			}
			dst, err := i18n.ValidateArticleMetadataBytes(p.Locale, c, target)
			if err != nil {
				return attention, err
			}
			names := []string{}
			for n := range src {
				names = append(names, n)
			}
			sort.Strings(names)
			for _, n := range names {
				for _, v := range []struct{ field, s, t string }{{"title", src[n].Title, dst[n].Title}, {"subtitle", src[n].Subtitle, dst[n].Subtitle}} {
					s, err := ProjectLanguageAuditText(a.TargetPath, n+"/"+v.field, "metadata", v.s, nil)
					if err != nil {
						return attention, err
					}
					t, err := ProjectLanguageAuditText(a.TargetPath, n+"/"+v.field, "metadata", v.t, nil)
					if err != nil {
						return attention, err
					}
					r, err := languageaudit.Scan(policy, s, t)
					if err != nil {
						return attention, err
					}
					if err := attention.Add("tour-metadata", a.TargetPath, "", "", r); err != nil {
						return attention, err
					}
				}
			}
		default:
			return attention, fmt.Errorf("structured asset lacks language adapter: %s", a.ID)
		}
	}
	if err := attention.Verify(root); err != nil {
		return attention, err
	}
	return attention, nil
}

var siteLanguageAuditPaths = []string{"internal/sitecontent/language_audit_formal.go", "internal/sitecontent/language_audit_view.go", "internal/sitecontent/language_audit_tour.go", "internal/sitecontent/semantic_role.go", "internal/sitecontent/semantic_go_packages.go", "internal/sitecontent/linguistic.go"}

func (v *closureValidation) bindLanguageAudit(root, locale string) error {
	_, ref, err := i18n.LoadLanguageAuditPolicy(root, locale, nil)
	if err != nil {
		return err
	}
	if err := v.bind(Reference{ref.Path, ref.SHA256}); err != nil {
		return err
	}
	for _, p := range append(append([]string(nil), i18n.LanguageAuditAuthorityPaths...), siteLanguageAuditPaths...) {
		if _, err := v.readRegular(root, p); err != nil {
			return err
		}
	}
	return nil
}

func scanFormalLanguageDocument(root, locale string, d Document, source, target []byte) (languageaudit.Result, error) {
	if d.Kind != "data" || d.Contract == ShellExtractionContract {
		return scanFormalLanguagePair(root, locale, d.Path, source, target)
	}
	g, err := i18n.LoadGlossary(root, locale)
	if err != nil {
		return languageaudit.Result{}, err
	}
	p, _, err := i18n.LoadLanguageAuditPolicy(root, locale, g)
	if err != nil {
		return languageaudit.Result{}, err
	}
	s, err := ProjectLanguageAudit(d.Path, source, AuditData)
	if err != nil {
		return languageaudit.Result{}, err
	}
	t, err := ProjectLanguageAudit(d.Path, target, AuditData)
	if err != nil {
		return languageaudit.Result{}, err
	}
	return languageaudit.Scan(p, s, t)
}
