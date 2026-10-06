package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
)

const StructuredSchema = "go-learning/locale-structured-assets/v1"

type StructuredAsset struct {
	ID          string      `json:"id"`
	Package     string      `json:"package"`
	SourcePaths []string    `json:"source_paths"`
	SourceSHA   string      `json:"source_sha256"`
	TargetPath  string      `json:"target_path"`
	State       string      `json:"state"`
	TargetSHA   string      `json:"target_sha256,omitempty"`
	Proof       []Reference `json:"carry_evidence"`
	Document    *Document   `json:"document,omitempty"`
}
type StructuredPlan struct {
	Schema      string            `json:"schema"`
	Locale      string            `json:"locale"`
	CorpusSHA   string            `json:"corpus_identity_sha256"`
	GlossarySHA string            `json:"glossary_sha256"`
	Review      Reference         `json:"unified_glossary_review"`
	Assets      []StructuredAsset `json:"assets"`
	Expected    []string          `json:"expected_outputs"`
	Identity    string            `json:"identity_sha256"`
}

// Registry membership is shared with the terminology corpus, not another list.
func PlanStructuredAssets(root, locale string) (*StructuredPlan, error) {
	if !component(locale) {
		return nil, fmt.Errorf("invalid locale")
	}
	if err := requireUnifiedReview(root, locale); err != nil {
		return nil, err
	}
	corpus, err := i18n.LoadUnifiedGlossaryCorpus(root)
	if err != nil {
		return nil, err
	}
	glossary, err := readRegular(root, "locales/"+locale+"/glossary.yaml")
	if err != nil {
		return nil, err
	}
	review, err := i18n.RequireUnifiedGlossaryReview(root, locale)
	if err != nil {
		return nil, err
	}
	p := &StructuredPlan{Schema: StructuredSchema, Locale: locale, CorpusSHA: corpus.Identity, GlossarySHA: digest(glossary), Review: Reference{review.Path, review.SHA256}, Assets: []StructuredAsset{}, Expected: []string{}}
	groups := map[string][]i18n.LanguageContributor{}
	for _, c := range corpus.Contributors {
		if !c.Structured {
			continue
		}
		id := c.ID
		if c.Parser == "ui-visible/v1" {
			id = "tour-ui"
		}
		if c.Parser == "article-header/v1" {
			id = "article-metadata"
		}
		groups[id] = append(groups[id], c)
	}
	g, err := LoadCurrent(root)
	if err != nil {
		return nil, err
	}
	docs := map[string]Document{}
	for _, pkg := range []string{"site-v2-shell", "learn-docs-v1"} {
		ds, _, err := PackageDocuments(root, g, pkg)
		if err != nil {
			return nil, err
		}
		for _, d := range ds {
			docs[d.Path] = d
		}
	}
	var completion *Completion
	completed := map[string]Completion{}
	b, err := readRegular(root, LocalePath(locale))
	if err == nil {
		var l Locale
		if err := StrictJSON(b, &l); err != nil {
			return nil, err
		}
		if err := ValidateLocale(g, l); err != nil {
			return nil, err
		}
		for i := range l.Packages {
			completed[l.Packages[i].Package] = l.Packages[i]
			if l.Packages[i].Package == "tour-v1" {
				completion = &l.Packages[i]
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	reuse := map[string]string{}
	if completion != nil {
		plan, err := PlanLocaleLanguage(root, locale)
		if err != nil {
			return nil, err
		}
		for _, w := range plan.Work {
			reuse[w.ID] = w.State
		}
	}

	ids := []string{}
	for id := range groups {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		cs := groups[id]
		paths := []string{}
		for _, c := range cs {
			paths = append(paths, c.Path)
		}
		paths = uniqueStrings(paths)
		a := StructuredAsset{ID: id, Package: cs[0].Package, SourcePaths: paths, SourceSHA: identity(cs), State: "generation-required", Proof: []Reference{}}
		switch id {
		case "tour-ui":
			a.TargetPath = "internal/tour/ui/" + locale + ".json"
		case "article-metadata":
			a.TargetPath = "locales/" + locale + "/article-metadata.json"
		default:
			a.TargetPath = "locales/" + locale + "/structured/" + cs[0].Path
			d, ok := docs[cs[0].Path]
			if !ok {
				return nil, fmt.Errorf("unknown structured parser")
			}
			a.Document = &d
		}
		if c, ok := completed[a.Package]; ok && a.Document != nil {
			if err := checkPackageCompletion(root, g, locale, c); err != nil {
				return nil, err
			}
			_, files, err := CheckPackageClosure(root, c.Evidence[0])
			if err != nil {
				return nil, err
			}
			file, ok := files[a.Document.Path]
			if !ok {
				return nil, fmt.Errorf("completed structured asset missing from exact closure")
			}
			a.State, a.TargetSHA, a.Proof = "carry", digest(file), c.Evidence
			p.Assets = append(p.Assets, a)
			continue
		}
		target, err := readRegular(root, a.TargetPath)
		if err == nil {
			a.TargetSHA = digest(target)
			a.State = "stale"
			if completion != nil && a.Package == "tour-v1" {
				for _, r := range completion.Evidence {
					if r.Path == a.TargetPath && r.SHA256 == a.TargetSHA {
						a.Proof = append(a.Proof, r)
					}
				}
				if len(a.Proof) > 0 {
					cat, err := checkedCatalog(root)
					if err != nil {
						return nil, err
					}
					unchanged := true
					oldGlossary := ""
					for _, r := range completion.Evidence {
						if r.Path == "locales/"+locale+"/glossary.yaml" {
							oldGlossary = r.SHA256
						}
					}
					for _, c := range cs {
						scope := c.ID
						if reuse[scope] != "carry" {
							unchanged = false
						}
						if oldGlossary != p.GlossarySHA {
							result := i18n.ResolveGlossaryCompatibility(root, locale, oldGlossary, p.GlossarySHA, scope, cat)
							if result.Status != "compatible" && result.Status != "exact" {
								unchanged = false
							}
						}
						for _, r := range completion.Evidence {
							if r.Path == c.Path {
								source, e := readRegular(root, c.Path)
								if e != nil || digest(source) != r.SHA256 {
									unchanged = false
								}
							}
						}
					}
					if unchanged {
						a.State = "carry"
					}
				}
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		} else {
			a.State = "missing"
		}
		// New/missing/stale share one exact generation-required output set.
		if a.State != "carry" && a.State != "context-only" {
			p.Expected = append(p.Expected, a.TargetPath)
		}
		p.Assets = append(p.Assets, a)
	}
	sort.Strings(p.Expected)
	p.Identity = identity(*p)
	return p, nil
}
func uniqueStrings(xs []string) []string {
	sort.Strings(xs)
	out := []string{}
	for _, x := range xs {
		if len(out) == 0 || out[len(out)-1] != x {
			out = append(out, x)
		}
	}
	return out
}

type StructuredBundle struct {
	RevisionFinding *Reference                 `json:"revision_finding,omitempty"`
	Schema          string                     `json:"schema"`
	Plan            StructuredPlan             `json:"plan"`
	Files           []i18n.TransportBundleFile `json:"files"`
	Identity        string                     `json:"identity_sha256"`
}

func ExportStructuredAssets(root, locale string) ([]byte, *StructuredBundle, error) {
	return exportStructuredAssets(root, locale, nil)
}
func ExportStructuredAssetsRevision(root, locale string, finding Reference) ([]byte, *StructuredBundle, error) {
	return exportStructuredAssets(root, locale, &finding)
}
func exportStructuredAssets(root, locale string, finding *Reference) ([]byte, *StructuredBundle, error) {
	p, err := PlanStructuredAssets(root, locale)
	if err != nil {
		return nil, nil, err
	}
	if finding != nil {
		r, err := validatedSurfaceFinding(root, locale, *finding)
		if err != nil {
			return nil, nil, err
		}
		expected := []string{}
		for i := range p.Assets {
			a := &p.Assets[i]
			needed := false
			for _, f := range r.ExactFindings {
				if a.Document != nil && a.Document.Path == f.ID {
					needed = true
				}
				if a.ID == "tour-ui" && strings.HasPrefix(f.ID, "ui:") {
					needed = true
				}
				if a.ID == "article-metadata" && strings.HasPrefix(f.ID, "article:") {
					needed = true
				}
			}
			if needed {
				expected = append(expected, a.TargetPath)
				a.State = "generation-required"
			} else {
				a.State = "context-only"
			}
		}
		sort.Strings(expected)
		p.Expected = expected
		p.Identity = ""
		p.Identity = identity(*p)
	}
	if len(p.Expected) == 0 {
		return nil, nil, fmt.Errorf("no generation-required structured assets")
	}
	m := &StructuredBundle{Schema: StructuredSchema, RevisionFinding: finding, Plan: *p, Files: []i18n.TransportBundleFile{}}
	entries := []i18n.TransportBundleEntry{}
	paths := []string{i18n.GlossaryCorpusPath, "locales/" + locale + "/glossary.yaml", "locales/" + locale + "/locale.json", p.Review.Path}
	paths = append(paths, PackageAuthorities...)
	if finding != nil {
		paths = append(paths, finding.Path)
	}
	for _, a := range p.Assets {
		if a.State != "carry" && a.State != "context-only" {
			for _, path := range a.SourcePaths {
				if a.Document == nil {
					paths = append(paths, path)
				}
			}
		}
	}
	for _, path := range uniqueStrings(paths) {
		b, err := readRegular(root, path)
		if err != nil {
			return nil, nil, err
		}
		name := "inputs/" + path
		entries = append(entries, i18n.TransportBundleEntry{Path: name, Data: b})
		m.Files = append(m.Files, i18n.NewTransportBundleFile(name, path, b))
	}
	global, err := LoadCurrent(root)
	if err != nil {
		return nil, nil, err
	}
	raw := map[string][]byte{}
	for _, pkg := range []string{"site-v2-shell", "learn-docs-v1"} {
		_, files, err := PackageDocuments(root, global, pkg)
		if err != nil {
			return nil, nil, err
		}
		for k, b := range files {
			raw[k] = b
		}
	}
	for _, a := range p.Assets {
		if (a.State == "carry" || a.State == "context-only") || a.Document == nil {
			continue
		}
		b := raw[a.Document.Path]
		name := "source/" + a.Document.Path
		entries = append(entries, i18n.TransportBundleEntry{Path: name, Data: b})
		m.Files = append(m.Files, i18n.NewTransportBundleFile(name, a.Document.Path, b))
	}
	if finding != nil {
		var review IntegratedSurfaceReceipt
		if err := readReference(root, *finding, &review); err != nil {
			return nil, nil, err
		}
		for _, ref := range review.Scope.Closures {
			_, previous, err := CheckPackageClosure(root, ref)
			if err != nil {
				return nil, nil, err
			}
			for _, a := range p.Assets {
				if a.State != "generation-required" || a.Document == nil {
					continue
				}
				if b, ok := previous[a.Document.Path]; ok {
					name := "previous/" + a.Document.Path
					entries = append(entries, i18n.TransportBundleEntry{Path: name, Data: b})
					m.Files = append(m.Files, i18n.NewTransportBundleFile(name, "", b))
				}
			}
		}
	}
	sort.Slice(m.Files, func(i, j int) bool { return m.Files[i].BundlePath < m.Files[j].BundlePath })
	m.Identity = identity(*m)
	manifest, err := Encode(m)
	if err != nil {
		return nil, nil, err
	}
	b, err := i18n.WriteDeterministicTransportBundle(manifest, entries)
	return b, m, err
}
func CheckStructuredAssetsBundle(root string, b []byte) (*StructuredBundle, error) {
	fs, err := i18n.ReadTransportBundle(b, 256, 32<<20)
	if err != nil {
		return nil, err
	}
	var m StructuredBundle
	if err := StrictJSON(fs["manifest.json"], &m); err != nil {
		return nil, err
	}
	if err := i18n.ValidateTransportBundleInventory(fs, m.Files, true); err != nil {
		return nil, err
	}
	want, _, err := exportStructuredAssets(root, m.Plan.Locale, m.RevisionFinding)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(want, b) {
		return nil, fmt.Errorf("structured bundle stale/noncanonical")
	}
	return &m, nil
}

type StructuredTarget struct {
	Path  string   `json:"path"`
	Data  string   `json:"data"`
	Slots []Target `json:"slots"`
}
type StructuredOutput struct {
	Schema    string             `json:"schema"`
	BundleSHA string             `json:"bundle_identity_sha256"`
	Outputs   []StructuredTarget `json:"outputs"`
}
type StructuredGeneration struct {
	Schema     string           `json:"schema"`
	Plan       StructuredPlan   `json:"plan"`
	Output     StructuredOutput `json:"output"`
	Provenance Provenance       `json:"generation"`
	Bundle     Reference        `json:"bundle"`
	Identity   string           `json:"identity_sha256"`
}

func validateStructuredOutput(root string, p StructuredPlan, o StructuredOutput) (map[string][]byte, error) {
	if o.Schema != StructuredSchema || len(o.Outputs) != len(p.Expected) {
		return nil, fmt.Errorf("structured output exact-set mismatch")
	}
	all := map[string]StructuredAsset{}
	for _, a := range p.Assets {
		all[a.TargetPath] = a
	}
	files := map[string][]byte{}
	g, err := LoadCurrent(root)
	if err != nil {
		return nil, err
	}
	raw := map[string][]byte{}
	for _, pkg := range []string{"site-v2-shell", "learn-docs-v1"} {
		_, rs, err := PackageDocuments(root, g, pkg)
		if err != nil {
			return nil, err
		}
		for k, v := range rs {
			raw[k] = v
		}
	}
	glossary, err := i18n.LoadGlossary(root, p.Locale)
	if err != nil {
		return nil, err
	}
	for i, t := range o.Outputs {
		if t.Path != p.Expected[i] {
			return nil, fmt.Errorf("structured output outside generation-required exact set")
		}
		a := all[t.Path]
		if a.Document != nil {
			if t.Data != "" || len(t.Slots) != len(a.Document.Units) {
				return nil, fmt.Errorf("structured parser slot exact-set required")
			}
			values := map[string]string{}
			for _, v := range t.Slots {
				if values[v.ID] != "" {
					return nil, fmt.Errorf("duplicate structured slot")
				}
				values[v.ID] = v.Text
			}
			for _, u := range a.Document.Units {
				if err := i18n.ValidateParsedGlossary(u.Source, values[u.ID], glossary); err != nil {
					return nil, err
				}
			}
			b, err := Reconstruct(documentSource(*a.Document), raw[a.Document.Path], values)
			if err != nil {
				return nil, err
			}
			files[t.Path] = b
		} else {
			if len(t.Slots) != 0 {
				return nil, fmt.Errorf("Tour structured asset is complete file")
			}
			b := []byte(t.Data)
			if err := textBytes(b); err != nil {
				return nil, err
			}
			switch a.ID {
			case "tour-ui":
				en, err := readRegular(root, "internal/tour/ui/en.json")
				if err != nil {
					return nil, err
				}
				if err := ui.ValidateLocalizedBytes(p.Locale, en, b); err != nil {
					return nil, err
				}
			case "article-metadata":
				cat, err := checkedCatalog(root)
				if err != nil {
					return nil, err
				}
				if _, err := i18n.ValidateArticleMetadataBytes(p.Locale, cat, b); err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("unknown structured validator")
			}
			files[t.Path] = b
		}
	}
	return files, nil
}
func ImportStructuredAssets(root string, bundle, output []byte, provenance Provenance) (*StructuredGeneration, string, error) {
	m, err := CheckStructuredAssetsBundle(root, bundle)
	if err != nil {
		return nil, "", err
	}
	if err := validateProvenance(provenance); err != nil {
		return nil, "", err
	}
	if err := requireIndependentSession(root, m.Plan.Locale, provenance.Session, "generation"); err != nil {
		return nil, "", err
	}
	var o StructuredOutput
	if err := StrictJSON(output, &o); err != nil {
		return nil, "", err
	}
	if o.BundleSHA != m.Identity {
		return nil, "", fmt.Errorf("structured bundle identity mismatch")
	}
	if _, err := validateStructuredOutput(root, m.Plan, o); err != nil {
		return nil, "", err
	}
	ref := Reference{Path: "data/locale-language/" + m.Plan.Locale + "/structured-bundles/" + digest(bundle) + ".zip", SHA256: digest(bundle)}
	r := &StructuredGeneration{Schema: StructuredSchema, Plan: m.Plan, Output: o, Provenance: provenance, Bundle: ref}
	r.Identity = identity(*r)
	p := "data/locale-language/" + m.Plan.Locale + "/structured-generation/" + r.Identity + ".json"
	if err := SaveArtifact(root, ref.Path, bundle); err != nil {
		return nil, "", err
	}
	if err := saveImmutable(root, p, r); err != nil {
		return nil, "", err
	}
	return r, p, nil
}
func CheckStructuredGeneration(root string, ref Reference) (*StructuredGeneration, map[string][]byte, error) {
	return checkStructuredGeneration(root, ref, "")
}
func checkStructuredGeneration(root string, ref Reference, pkg string) (*StructuredGeneration, map[string][]byte, error) {
	var r StructuredGeneration
	if err := readReference(root, ref, &r); err != nil {
		return nil, nil, err
	}
	copy := r
	copy.Identity = ""
	if r.Schema != StructuredSchema || r.Identity != identity(copy) || ref.Path != "data/locale-language/"+r.Plan.Locale+"/structured-generation/"+r.Identity+".json" {
		return nil, nil, fmt.Errorf("invalid structured evidence identity")
	}
	if err := validateProvenance(r.Provenance); err != nil {
		return nil, nil, err
	}
	b, err := readRegular(root, r.Bundle.Path)
	if err != nil || digest(b) != r.Bundle.SHA256 {
		return nil, nil, fmt.Errorf("structured archived bundle hash mismatch")
	}
	fs, err := i18n.ReadTransportBundle(b, 256, 32<<20)
	if err != nil {
		return nil, nil, err
	}
	var manifest StructuredBundle
	if err := StrictJSON(fs["manifest.json"], &manifest); err != nil {
		return nil, nil, err
	}
	m := &manifest
	if err := i18n.ValidateTransportBundleInventory(fs, m.Files, true); err != nil {
		return nil, nil, err
	}
	mc := *m
	mc.Identity = ""
	if m.Identity != identity(mc) {
		return nil, nil, fmt.Errorf("structured archived input identity mismatch")
	}

	if identity(m.Plan) != identity(r.Plan) || r.Output.BundleSHA != m.Identity {
		return nil, nil, fmt.Errorf("structured evidence input mismatch")
	}
	p := r.Plan
	o := r.Output
	if pkg != "" {
		assets := []StructuredAsset{}
		wanted := map[string]bool{}
		for _, a := range p.Assets {
			if a.Package == pkg {
				assets = append(assets, a)
				wanted[a.TargetPath] = true
			}
		}
		p.Assets = assets
		p.Expected = []string{}
		o.Outputs = []StructuredTarget{}
		for _, t := range r.Output.Outputs {
			if wanted[t.Path] {
				p.Expected = append(p.Expected, t.Path)
				o.Outputs = append(o.Outputs, t)
			}
		}
	}
	// Per-asset semantic reuse; corpus growth elsewhere is not a language gate.
	current, err := readRegular(root, "locales/"+p.Locale+"/glossary.yaml")
	if err != nil {
		return nil, nil, err
	}
	if digest(current) != p.GlossarySHA {
		old := fs["inputs/locales/"+p.Locale+"/glossary.yaml"]
		delta, err := i18n.ParsedGlossaryDelta(p.Locale, old, current)
		if err != nil {
			return nil, nil, err
		}
		for _, a := range p.Assets {
			if a.Document == nil {
				return nil, nil, fmt.Errorf("Tour structured carry requires original V2-B scope proof")
			}
			for _, t := range o.Outputs {
				if t.Path != a.TargetPath {
					continue
				}
				target := map[string]string{}
				for _, v := range t.Slots {
					target[v.ID] = v.Text
				}
				for _, u := range a.Document.Units {
					ok, _ := i18n.ParsedGlossaryImpact(u.Source, target[u.ID], delta)
					if !ok {
						return nil, nil, fmt.Errorf("affected structured asset %s", a.ID)
					}
				}
			}
		}
	}
	files, err := validateStructuredOutput(root, p, o)
	return &r, files, err
}
func structuredSupersedes(root string, g *StructuredGeneration, parent Reference, path string) bool {
	b, err := readRegular(root, g.Bundle.Path)
	if err != nil {
		return false
	}
	fs, err := i18n.ReadTransportBundle(b, 256, 32<<20)
	if err != nil {
		return false
	}
	var m StructuredBundle
	if StrictJSON(fs["manifest.json"], &m) != nil || m.RevisionFinding == nil {
		return false
	}
	var r IntegratedSurfaceReceipt
	if readReference(root, *m.RevisionFinding, &r) != nil || r.Decision != "failed" {
		return false
	}
	exact := false
	for _, f := range r.ExactFindings {
		if f.ID == path {
			exact = true
		}
	}
	if !exact {
		return false
	}
	for _, ref := range r.Scope.Closures {
		var c PackageClosure
		if readReference(root, ref, &c) != nil {
			return false
		}
		for _, old := range c.Structured {
			if old == parent {
				return true
			}
		}
	}
	return false
}
