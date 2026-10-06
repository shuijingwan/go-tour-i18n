package sitecontent

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

type ReviewRow struct {
	Unit         Unit       `json:"unit"`
	Target       Target     `json:"target"`
	DocumentPath string     `json:"document_path"`
	DocumentSHA  string     `json:"document_sha256"`
	Generation   Provenance `json:"generation"`
}

// Context metadata has no duplicated Unit array. The full source is one ZIP
// member; selected source/target/protection is present exactly in Rows.
type ReviewDocument struct {
	Contract     string       `json:"contract"`
	StableIndex  int          `json:"stable_index"`
	UnitScope    string       `json:"unit_scope"`
	Path         string       `json:"path"`
	Kind         string       `json:"kind"`
	Route        string       `json:"route"`
	SourceSHA    string       `json:"source_sha256"`
	StructureSHA string       `json:"structure_sha256"`
	Dependencies []Dependency `json:"dependencies"`
	Identity     string       `json:"document_sha256"`
}
type PackageReviewerBundle struct {
	Schema    string                     `json:"schema"`
	Scope     Review                     `json:"scope"`
	Rows      []ReviewRow                `json:"rows"`
	Documents []ReviewDocument           `json:"documents"`
	Files     []i18n.TransportBundleFile `json:"files"`
	Identity  string                     `json:"identity_sha256"`
}

func ExportPackageReview(root string, scope *Review) ([]byte, error) {
	want, err := ReviewScope(root, scope.Locale, scope.Package, scope.Generations, scope.Selected)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(scope, want) {
		return nil, fmt.Errorf("review scope is not current")
	}
	_, _, docs, raw, glossary, err := currentPackage(root, scope.Locale, scope.Package)
	if err != nil {
		return nil, err
	}
	values, err := selections(root, scope.Locale, scope.Package, scope.Generations, docs)
	if err != nil {
		return nil, err
	}
	m := &PackageReviewerBundle{Schema: "go-learning/package-reviewer-bundle/v2", Scope: *scope, Rows: []ReviewRow{}, Documents: []ReviewDocument{}, Files: []i18n.TransportBundleFile{}}
	for _, d := range selectedDocuments(docs, scope.Selected) {
		m.Documents = append(m.Documents, ReviewDocument{d.Contract, d.StableIndex, d.UnitScope, d.Path, d.Kind, d.Route, d.SourceSHA, d.StructureSHA, d.Dependencies, identity(d)})
	}
	entries := []i18n.TransportBundleEntry{}
	seen := map[string]bool{}
	add := func(name, repo string, b []byte) {
		if seen[name] {
			return
		}
		seen[name] = true
		entries = append(entries, i18n.TransportBundleEntry{Path: name, Data: b})
		m.Files = append(m.Files, i18n.NewTransportBundleFile(name, repo, b))
	}
	for _, id := range scope.Selected {
		s := values[id]
		m.Rows = append(m.Rows, ReviewRow{s.Unit, s.Target, s.Document.Path, identity(s.Document), s.Generation.Provenance})
		add("context/"+s.Document.Path, s.Document.Path, raw[s.Document.Path])
	}
	add("formal/glossary.yaml", "locales/"+scope.Locale+"/glossary.yaml", glossary)
	// Generation refs and per-row provenance bind immutable evidence without
	// duplicating the entire Generation Document inventory in model context.
	for _, d := range selectedDocuments(docs, scope.Selected) {
		u := d.Units[0]
		candidate, err := ReconstructWorkflow(d, raw[d.Path], map[string]string{u.ID: values[u.ID].Target.Text})
		if err != nil {
			return nil, err
		}
		add("candidate/"+d.Path, "", candidate)
	}

	for _, name := range append(append([]string{}, PackageAuthorities...), "docs/LOCALE_SURFACE_REVIEW.md", scope.GlossaryReview.Path) {
		b, err := readRegular(root, name)
		if err != nil {
			return nil, err
		}
		add("authority/"+name, name, b)
	}
	sort.Slice(m.Files, func(i, j int) bool { return m.Files[i].BundlePath < m.Files[j].BundlePath })
	m.Identity = identity(*m)
	b, err := Encode(m)
	if err != nil {
		return nil, err
	}
	return i18n.WriteDeterministicTransportBundle(b, entries)
}
func CheckPackageReviewBundle(root string, b []byte) error {
	files, err := i18n.ReadTransportBundle(b, 256, 32<<20)
	if err != nil {
		return err
	}
	var m PackageReviewerBundle
	if err := StrictJSON(files["manifest.json"], &m); err != nil {
		return err
	}
	if err := i18n.ValidateTransportBundleInventory(files, m.Files, true); err != nil {
		return err
	}
	want, err := ExportPackageReview(root, &m.Scope)
	if err != nil {
		return err
	}
	if !bytes.Equal(b, want) {
		return fmt.Errorf("Reviewer bundle non-current")
	}
	return nil
}
