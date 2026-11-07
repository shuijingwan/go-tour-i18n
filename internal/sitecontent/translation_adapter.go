package sitecontent

import (
	"bytes"
	"fmt"
	"html"
	"reflect"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// PageAdapter has no attempts, receipts, QC or recovery state. Those belong to
// i18n. The source-coordinate parser is used only for protection/restoration.
type PageAdapter struct {
	Document Document
	Source   []byte
}

func (a PageAdapter) ReattestationAuthority(root string) (string, error) {
	return i18n.TranslationAdapterAuthority(root, "internal/sitecontent/translation_adapter.go", "PageAdapter", []string{"internal/sitecontent/language_audit_view.go", "internal/sitecontent/language_audit_formal.go", "internal/sitecontent/language_audit_tour.go", "internal/sitecontent/comment_inventory.go", "internal/sitecontent/render_input.go", "internal/sitecontent/formal_page.go", "internal/sitecontent/units.go", "internal/sitecontent/linguistic.go", "internal/sitecontent/semantic_role.go", "internal/sitecontent/semantic_go_packages.go", "internal/sitecontent/page_structure.go", "internal/sitecontent/structural_binding.go", "internal/sitecontent/markdown_references.go", "internal/sitecontent/teaching.go", "internal/sitecontent/emphasis.go"})
}

// Visibility is representation-owned, using the formal Page parser's approved
// prose/metadata/teaching-comment coordinates. URLs, template/code identities
// and opaque preformatted blocks cannot cause artificial language impact.
func (a PageAdapter) GlossaryVisiblePair(candidate []byte) (string, string, error) {
	visible := func(b []byte) (string, error) {
		d, err := parseDocument(Source{Path: a.Document.Path, Kind: "page", Route: a.Document.Route, SHA256: digest(b)}, b, false)
		if err != nil {
			return "", err
		}
		var text []string
		for _, u := range d.Units {
			v := u.Source
			for _, p := range u.Protected {
				v = strings.ReplaceAll(v, p.Token, "")
			}
			text = append(text, html.UnescapeString(v))
		}
		return strings.Join(text, "\n"), nil
	}
	s, err := visible(a.Source)
	if err != nil {
		return "", "", err
	}
	t, err := visible(candidate)
	return s, t, err
}

// Additive package discovery registration is the only CLI extension point.
// A loader supplies frozen descriptors/adapters, never phase-specific logic.
var translationPackageLoaders = []func(string, *i18n.Catalog, ...*structuredSuccessorValidation) ([]string, error){AttachPageTranslationUnits, AttachShellPageTranslationUnits, AttachAdoptedPageTranslationUnits}

func AttachRegisteredTranslationPackages(root string, catalog *i18n.Catalog, contexts ...*structuredSuccessorValidation) error {
	catalog.DerivedTransportCodecReader = derivedApprovedCodec
	if len(contexts) != 0 && contexts[0] != nil && contexts[0].readonlyOwner {
		v := contexts[0]
		if root != v.guard.root {
			return fmt.Errorf("registration validation root changed")
		}
		for _, name := range []string{ShellAdoptionCurrent, PackageMembershipPath, PageRegistryPath, StructuredSourceReplacementPending} {
			if err := v.guard.optional(name); err != nil {
				return err
			}
		}
		for _, name := range []string{StructuredSourceSuccessorRoot + "current", PagePackageAdoptionRoot + "current"} {
			if err := v.guard.directory(name); err != nil {
				return err
			}
		}
		if _, absent := v.guard.absent[PackageMembershipPath]; !absent {
			var ref Reference
			if err := v.guard.readReference(root, Reference{PackageMembershipPath, v.guard.deps[PackageMembershipPath]}, &ref); err != nil {
				return err
			}
			var m PackageMembership
			if err := v.guard.readReference(root, ref, &m); err != nil {
				return err
			}
			if err := v.observe(m); err != nil {
				return err
			}
		}
		previous := v.registrationReuse
		v.registrationReuse = true
		defer func() { v.registrationReuse = previous }()
	}
	for _, load := range translationPackageLoaders {
		if _, err := load(root, catalog, contexts...); err != nil {
			return err
		}
	}
	if err := attachStructuredHistoricalPackages(root, catalog, contexts...); err != nil {
		return err
	}
	if err := attachPackageMembership(root, catalog, contexts...); err != nil {
		return err
	}
	AttachPageReplacementInvocation(root, catalog)
	return nil
}

func (a PageAdapter) ProtectedInput(_ *i18n.Glossary) (string, []string, error) {
	if a.Document.UnitScope != PageContract || len(a.Document.Units) != 1 || digest(a.Source) != a.Document.SourceSHA {
		return "", nil, fmt.Errorf("invalid formal Page adapter")
	}
	u := a.Document.Units[0]
	tokens := []string{}
	for _, p := range u.Protected {
		tokens = append(tokens, p.Token)
	}
	return u.Source, tokens, nil
}
func (a PageAdapter) Restore(output string) ([]byte, error) {
	// Shared text transport adds one LF after the final opaque boundary.
	// It is framing, not Page prose; the original JSON string is unchanged.
	output = string(bytes.TrimSuffix([]byte(output), []byte("\n")))
	return ReconstructWorkflow(a.Document, a.Source, map[string]string{a.Document.Units[0].ID: output})
}
func (a PageAdapter) Validate(candidate []byte, glossary *i18n.Glossary) error {
	if err := validatePageStructure(a.Document, a.Source, candidate); err != nil {
		return err
	}
	// Restore has already checked every coordinate and protected binding.
	// Glossary validation uses language-bearing source/target only.
	return i18n.ValidateParsedGlossary(string(a.Source), string(candidate), glossary)
}

func (a PageAdapter) VerifyGenerationOrigin(root, locale string, origin i18n.TranslationUnitGenerationOrigin, raw []byte) error {
	return a.verifyGenerationOrigin(root, locale, origin, raw, false)
}

func (a PageAdapter) VerifyFrozenGenerationOrigin(root, locale string, origin i18n.TranslationUnitGenerationOrigin, raw []byte) error {
	return a.verifyGenerationOrigin(root, locale, origin, raw, true)
}

func (a PageAdapter) verifyGenerationOrigin(root, locale string, origin i18n.TranslationUnitGenerationOrigin, raw []byte, frozen bool) error {
	b, err := readRegular(root, origin.Bundle.Path)
	if err != nil || digest(b) != origin.Bundle.SHA256 {
		return fmt.Errorf("original Generation bundle changed")
	}
	files, err := i18n.ReadTransportBundle(b, 256, 32<<20)
	if err != nil {
		return err
	}
	var m Batch
	if err := StrictJSON(files["manifest.json"], &m); err != nil {
		return err
	}
	if err := i18n.ValidateTransportBundleInventory(files, m.Files, true); err != nil {
		return err
	}
	copy := m
	copy.Identity = ""
	if m.Schema != WorkflowSchema || m.Identity != identity(copy) || m.Contract != UnitContract || m.Locale != locale {
		return fmt.Errorf("original batch identity mismatch")
	}
	glossary, err := readRegular(root, "locales/"+locale+"/glossary.yaml")
	if frozen {
		glossary, err = files["formal/glossary.yaml"], nil
	}
	if err != nil || !bytes.Equal(glossary, files["formal/glossary.yaml"]) || digest(glossary) != m.GlossarySHA {
		return fmt.Errorf("原 invocation 术语表不再逐字节一致；需要正式 compatibility，不能沿用")
	}
	var out Output
	if err := readReference(root, Reference{origin.Result.Path, origin.Result.SHA256}, &out); err != nil {
		return err
	}
	if out.Schema != WorkflowSchema || out.BatchSHA != m.Identity || len(out.Targets) != len(m.Selected) {
		return fmt.Errorf("original output identity mismatch")
	}
	for i, t := range out.Targets {
		if t.ID != m.Selected[i] {
			return fmt.Errorf("original exact-set mismatch")
		}
	}
	for _, d := range m.Documents {
		if d.Path != a.Document.Path {
			continue
		}
		if d.Route != a.Document.Route || !bytes.Equal(files["source/"+d.Path], a.Source) || !reflect.DeepEqual(d.Units, a.Document.Units) {
			return fmt.Errorf("正式页面输入真实变化，不可沿用：%s", d.Path)
		}
		for _, t := range out.Targets {
			if t.ID == a.Document.Units[0].ID && bytes.Equal(raw, append(bytes.TrimRight([]byte(t.Text), "\n"), '\n')) {
				return nil
			}
		}
	}
	return fmt.Errorf("original invocation does not bind this exact Unit/output")
}

// AttachPageTranslationUnits leaves Tour inventory/history intact and exposes
// one complete physical Page to the same retranslation lifecycle.
func AttachPageTranslationUnits(root string, catalog *i18n.Catalog, contexts ...*structuredSuccessorValidation) ([]string, error) {
	AttachPageReplacementResolver(catalog)
	g, err := loadSourceCurrent(root, contexts...)
	if err != nil {
		return nil, err
	}
	docs, raw, err := WorkflowDocuments(root, g, "learn-docs-v1", contexts...)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	descriptor := i18n.TranslationPackageDescriptor{ID: "learn-docs-v1"}
	packageAuthority, err := packageByID(g, descriptor.ID)
	if err != nil {
		return nil, err
	}
	descriptor.AuthoritySHA256 = packageAuthority.Identity
	for _, d := range docs {
		u := d.Units[0]
		a := PageAdapter{d, bytes.Clone(raw[d.Path])}
		// Fixed membership governs initial Generation/QC, not later revision.
		descriptor.Units = append(descriptor.Units, i18n.AdaptedTranslationUnit{Unit: i18n.TranslationUnit{ID: u.ID, Kind: i18n.UnitKindPage, SourcePath: d.Path, Source: bytes.Clone(raw[d.Path]), SourceSHA256: d.SourceSHA}, Adapter: a, StableIndex: d.StableIndex, Route: d.Route, WorkingSet: fmt.Sprintf("page-batch-%d", (d.StableIndex-1)/30+1), WorkingSetLimit: 30})
		ids = append(ids, u.ID)
	}
	return ids, catalog.RegisterTranslationPackage(descriptor)
}
