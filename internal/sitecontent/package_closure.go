package sitecontent

import (
	"fmt"
	"reflect"
	"sort"
)

const ClosureSchema = "go-learning/package-language-closure/v1"

type PackageClosure struct {
	Schema           string      `json:"schema"`
	Locale           string      `json:"locale"`
	Package          string      `json:"package"`
	PackageSHA       string      `json:"package_sha256"`
	PageFinalization *Reference  `json:"page_finalization,omitempty"`
	Structured       []Reference `json:"structured_generation"`
	Files            []Reference `json:"localized_files"`
	Identity         string      `json:"identity_sha256"`
}

func BuildPackageClosure(root, locale, pkg string, page *Reference, structured []Reference) (*PackageClosure, map[string][]byte, error) {
	if !component(locale) || (pkg != "site-v2-shell" && pkg != "learn-docs-v1") {
		return nil, nil, fmt.Errorf("invalid package closure")
	}
	if err := requireUnifiedReview(root, locale); err != nil {
		return nil, nil, err
	}
	g, err := LoadCurrent(root)
	if err != nil {
		return nil, nil, err
	}
	p, err := packageByID(g, pkg)
	if err != nil {
		return nil, nil, err
	}
	docs, raw, err := PackageDocuments(root, g, pkg)
	if err != nil {
		return nil, nil, err
	}
	out := map[string][]byte{}
	if pkg == "learn-docs-v1" {
		if page == nil {
			return nil, nil, fmt.Errorf("49 Page A-only finalization required")
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
	} else if page != nil {
		return nil, nil, fmt.Errorf("shell has no formal content-QC TU")
	}
	chosen := map[string]Reference{}
	owners := map[string]*StructuredGeneration{}
	for _, ref := range structured {
		r, files, err := checkStructuredGeneration(root, ref, pkg)
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
	c := &PackageClosure{Schema: ClosureSchema, Locale: locale, Package: pkg, PackageSHA: p.Identity, PageFinalization: page, Structured: structured, Files: []Reference{}}
	for path, b := range out {
		c.Files = append(c.Files, Reference{Path: path, SHA256: digest(b)})
	}
	sort.Slice(c.Files, func(i, j int) bool { return c.Files[i].Path < c.Files[j].Path })
	c.Identity = identity(*c)
	return c, out, nil
}
func FinalizePackageClosure(root, locale, pkg string, page *Reference, structured []Reference) (*PackageClosure, string, error) {
	c, _, err := BuildPackageClosure(root, locale, pkg, page, structured)
	if err != nil {
		return nil, "", err
	}
	p := workflowPath(locale, pkg, "closure", c.Identity)
	return c, p, saveImmutable(root, p, c)
}
func CheckPackageClosure(root string, ref Reference) (*PackageClosure, map[string][]byte, error) {
	var c PackageClosure
	if err := readReference(root, ref, &c); err != nil {
		return nil, nil, err
	}
	if ref.Path != workflowPath(c.Locale, c.Package, "closure", c.Identity) {
		return nil, nil, fmt.Errorf("closure path mismatch")
	}
	want, files, err := BuildPackageClosure(root, c.Locale, c.Package, c.PageFinalization, c.Structured)
	if err != nil {
		return nil, nil, err
	}
	if !reflect.DeepEqual(c, *want) {
		return nil, nil, fmt.Errorf("package closure stale")
	}
	return &c, files, nil
}
