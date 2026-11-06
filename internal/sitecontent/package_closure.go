package sitecontent

import (
	"fmt"
	"path"
	"reflect"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const ClosureSchema = "go-learning/package-language-closure/v1"

type PackageClosure struct {
	Membership         *Reference                          `json:"package_membership_successor,omitempty"`
	Schema             string                              `json:"schema"`
	Locale             string                              `json:"locale"`
	Package            string                              `json:"package"`
	PackageSHA         string                              `json:"package_sha256"`
	PageFinalization   *Reference                          `json:"page_finalization,omitempty"`
	PageComposition    []i18n.TranslationFinalizationInput `json:"page_finalization_composition,omitempty"`
	Structured         []Reference                         `json:"structured_generation"`
	StructuredBaseline *Reference                          `json:"structured_baseline_closure,omitempty"`
	Files              []Reference                         `json:"localized_files"`
	Identity           string                              `json:"identity_sha256"`
}

func (validation *closureValidation) buildPackageClosure(root, locale, pkg string, page *Reference, structured []Reference, frozenBaseline ...*Reference) (*PackageClosure, map[string][]byte, error) {
	return validation.buildPackageClosureInputs(root, locale, pkg, page, structured, nil, frozenBaseline...)
}

func (validation *closureValidation) buildPackageClosureInputs(root, locale, pkg string, page *Reference, structured []Reference, composition []i18n.TranslationFinalizationInput, frozenBaseline ...*Reference) (*PackageClosure, map[string][]byte, error) {
	return validation.buildPackageClosureMaterialized(root, locale, pkg, page, structured, composition, validation.mergeStructuredBaselineDelta, frozenBaseline...)
}

type structuredClosureMaterializer func(string, string, string, []Reference, ...*Reference) (map[string][]byte, *Reference, bool, error)

func (validation *closureValidation) buildPackageClosureMaterialized(root, locale, pkg string, page *Reference, structured []Reference, composition []i18n.TranslationFinalizationInput, materialize structuredClosureMaterializer, frozenBaseline ...*Reference) (*PackageClosure, map[string][]byte, error) {
	if !component(locale) || !component(pkg) {
		return nil, nil, fmt.Errorf("invalid package closure")
	}
	if err := validation.preflightClosureInputs(locale, pkg, page, structured, composition, frozenBaseline...); err != nil {
		return nil, nil, err
	}
	if pkg != "site-v2-shell" && pkg != "learn-docs-v1" && len(composition) == 0 {
		if page == nil || len(structured) != 0 {
			return nil, nil, fmt.Errorf("descriptor closure requires shared finalization and exact TU-only inventory")
		}
		catalog, err := checkedCatalog(root)
		if err != nil {
			return nil, nil, err
		}
		if err := AttachRegisteredTranslationPackages(root, catalog); err != nil {
			return nil, nil, err
		}
		g, err := validation.loadCurrent(root)
		if err != nil {
			return nil, nil, err
		}
		p, err := packageByID(g, pkg)
		if err != nil {
			return nil, nil, err
		}
		bound := false
		for _, d := range catalog.TranslationPackages {
			if d.ID == pkg && d.AuthoritySHA256 == p.Identity {
				bound = true
			}
		}
		if !bound {
			return nil, nil, fmt.Errorf("descriptor must bind current Content Scope package identity")
		}
		var f i18n.QualityCheckFinalization
		if err := readReference(root, *page, &f); err != nil {
			return nil, nil, err
		}
		if f.Locale != locale {
			return nil, nil, fmt.Errorf("descriptor finalization locale mismatch")
		}
		if page.Path != "data/quality-check-snapshots/"+locale+"/"+f.SnapshotID+"/finalization.json" {
			return nil, nil, fmt.Errorf("descriptor finalization path mismatch")
		}
		return BuildTranslationPackageClosure(root, catalog, locale, pkg, f.SnapshotID)
	}
	if err := validation.requireUnifiedReview(root, locale); err != nil {
		return nil, nil, err
	}
	g, err := validation.loadCurrent(root)
	if err != nil {
		return nil, nil, err
	}
	p, err := packageByID(g, pkg)
	if err != nil {
		return nil, nil, err
	}
	docs, raw, err := CurrentPackageDocuments(root, g, pkg)
	if err != nil {
		return nil, nil, err
	}
	out := map[string][]byte{}
	if len(composition) > 0 {
		if page != nil {
			return nil, nil, fmt.Errorf("composition cannot replace/merge a single Snapshot")
		}
		c, err := validation.packageCatalog(pkg)
		if err != nil {
			return nil, nil, err
		}
		_, files, err := validation.composePackage(root, c, locale, composition)
		if err != nil {
			return nil, nil, err
		}
		for path, b := range files {
			out[path] = b
		}
	} else if pkg == "learn-docs-v1" || (pkg == "site-v2-shell" && g.ShellAdoption != nil) {
		if page == nil {
			return nil, nil, fmt.Errorf("current package Page A-only finalization required")
		}
		if strings.HasPrefix(page.Path, "data/quality-check-snapshots/") {
			if err := validation.projectSharedPackageFinalization(root, locale, pkg, *page, out); err != nil {
				return nil, nil, err
			}
		} else {
			if pkg == "site-v2-shell" {
				return nil, nil, fmt.Errorf("Homepage requires shared Page finalization")
			}
			f, err := CheckFinalization(root, *page)
			if err != nil {
				return nil, nil, err
			}
			if f.Locale != locale || f.Package != pkg {
				return nil, nil, fmt.Errorf("Page closure scope mismatch")
			}
			values := map[string]string{}
			for _, v := range f.Targets {
				values[v.ID] = v.Text
			}
			for _, d := range f.Documents {
				b, err := ReconstructWorkflow(d, raw[d.Path], map[string]string{d.Units[0].ID: values[d.Units[0].ID]})
				if err != nil {
					return nil, nil, err
				}
				out[d.Path] = b
			}
		}
	} else if page != nil {
		return nil, nil, fmt.Errorf("shell has no formal content-QC TU")
	}
	merged, baseline, handled, err := materialize(root, locale, pkg, structured, frozenBaseline...)
	if err != nil {
		return nil, nil, err
	}
	for path, b := range merged {
		out[path] = b
	}
	chosen := map[string]Reference{}
	owners := map[string]*StructuredGeneration{}
	for _, ref := range structured {
		if handled {
			break
		}
		r, files, err := validation.checkStructuredGeneration(root, ref, pkg)
		if err != nil {
			return nil, nil, err
		}
		if r.Plan.Locale != locale {
			return nil, nil, fmt.Errorf("structured locale mismatch")
		}
		for _, a := range r.Plan.Assets {
			if a.Package != pkg || a.Document == nil {
				continue
			}
			b, ok := files[a.TargetPath]
			if !ok {
				continue
			}
			if _, duplicate := out[a.Document.Path]; duplicate {
				if structuredSupersedes(root, r, chosen[a.Document.Path], a.Document.Path) {
				} else if structuredSupersedes(root, owners[a.Document.Path], ref, a.Document.Path) {
					continue
				} else {
					return nil, nil, fmt.Errorf("ambiguous structured replacement")
				}
			}
			out[a.Document.Path] = b
			chosen[a.Document.Path] = ref
			owners[a.Document.Path] = r
		}
	}
	if len(out) != len(docs) {
		return nil, nil, fmt.Errorf("atomic package incomplete: localized=%d expected=%d", len(out), len(docs))
	}
	for _, d := range docs {
		if _, ok := out[d.Path]; !ok {
			return nil, nil, fmt.Errorf("missing package file %s", d.Path)
		}
	}
	c := &PackageClosure{Schema: ClosureSchema, Locale: locale, Package: pkg, PackageSHA: p.Identity, PageFinalization: page, PageComposition: composition, Structured: structured, StructuredBaseline: baseline, Files: []Reference{}}
	if g.Membership != nil {
		var m PackageMembership
		if err := validation.readReference(root, *g.Membership, &m); err != nil {
			return nil, nil, err
		}
		if m.Package == pkg {
			c.Membership = g.Membership
		}
	}
	for path, b := range out {
		c.Files = append(c.Files, Reference{Path: path, SHA256: digest(b)})
	}
	sort.Slice(c.Files, func(i, j int) bool { return c.Files[i].Path < c.Files[j].Path })
	c.Identity = identity(*c)
	return c, out, nil
}

// This is package projection after the shared A-only gate, never a second
// finalization/QC state machine. Legacy package artifacts remain readable.
func projectSharedPageFinalization(root, locale string, ref Reference, out map[string][]byte) error {
	return projectSharedPackageFinalization(root, locale, "learn-docs-v1", ref, out)
}

func projectSharedPackageFinalization(root, locale, pkg string, ref Reference, out map[string][]byte) error {
	var f i18n.QualityCheckFinalization
	if err := readReference(root, ref, &f); err != nil {
		return err
	}
	if f.Locale != locale || ref.Path != path.Join("data/quality-check-snapshots", locale, f.SnapshotID, "finalization.json") {
		return fmt.Errorf("shared finalization scope/path mismatch")
	}
	c, err := i18n.ReadCatalog(root)
	if err != nil {
		return err
	}
	current, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		return err
	}
	if err := i18n.HydrateCatalogSources(c, current); err != nil {
		return err
	}
	err = AttachRegisteredTranslationPackages(root, c)
	if err != nil {
		return err
	}
	AttachIntegratedSurfaceAuthority(c)
	if err := c.UseTranslationPackage(pkg); err != nil {
		return err
	}
	files, err := i18n.FinalizedTranslationPackageCandidates(root, c, locale, f.SnapshotID)
	if err != nil {
		return err
	}
	for p, b := range files {
		out[p] = b
	}
	return nil
}
func FinalizePackageClosure(root, locale, pkg string, page *Reference, structured []Reference) (*PackageClosure, string, error) {
	c, _, err := BuildPackageClosure(root, locale, pkg, page, structured)
	if err != nil {
		return nil, "", err
	}
	p := workflowPath(locale, pkg, "closure", c.Identity)
	return c, p, saveImmutable(root, p, c)
}
func (validation *closureValidation) computePackageClosure(root string, ref Reference) (*PackageClosure, map[string][]byte, error) {
	var c PackageClosure
	if err := validation.readReference(root, ref, &c); err != nil {
		return nil, nil, err
	}
	if ref.Path != workflowPath(c.Locale, c.Package, "closure", c.Identity) {
		return nil, nil, fmt.Errorf("closure path mismatch")
	}
	global, err := validation.loadCurrent(root)
	if err != nil {
		return nil, nil, err
	}
	for _, old := range global.HistoricalPackages {
		if c.Package == old.ID && c.PackageSHA == old.Identity {
			baseline, err := validation.historicalStructuredClosure(root, ref, map[string]bool{})
			if err != nil {
				return nil, nil, err
			}
			return &baseline.Closure, baseline.Files, nil
		}
	}
	want, files, err := validation.buildPackageClosureInputs(root, c.Locale, c.Package, c.PageFinalization, c.Structured, c.PageComposition, c.StructuredBaseline)
	if err != nil {
		return nil, nil, err
	}
	if !reflect.DeepEqual(c, *want) {
		return nil, nil, fmt.Errorf("package closure stale")
	}
	return &c, files, nil
}
