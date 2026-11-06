package sitecontent

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Workflow consumers share the existing package observation owner. The original
// identity-bound WorkflowDocuments remains the canonical compatibility reader;
// the source parser/FormalPage algorithms are called unchanged here.
func sharedWorkflowDocuments(root string, g *Global, pkg string, contexts ...*structuredSuccessorValidation) ([]Document, map[string][]byte, error) {
	canonical := func() ([]Document, map[string][]byte, error) { return WorkflowDocuments(root, g, pkg, contexts...) }
	if !contentidentity.MemoizationActive() {
		return canonical()
	}
	if _, err := packageByID(g, pkg); err != nil {
		return nil, nil, err
	}
	for _, ref := range g.PackageAdoptions {
		var a PagePackageAdoption
		if err := readReference(root, ref, &a); err != nil {
			return nil, nil, err
		}
		if a.Package.ID == pkg {
			return adoptedPageDocuments(root, g, pkg, contexts...)
		}
	}
	docs, raw, err := sharedPackageRepresentation(root, g, pkg, contexts...)
	if err != nil {
		return nil, nil, err
	}
	indexes := map[string]int{}
	if pkg == "learn-docs-v1" {
		r, err := LoadPageRegistry(root, g)
		if err != nil {
			return nil, nil, err
		}
		for _, p := range r.Pages {
			if !p.Retired {
				indexes[p.Path] = p.Index
			}
		}
	}
	input := struct {
		Documents []Document
		Raw       map[string][]byte
		Indexes   map[string]int
	}{docs, raw, indexes}
	key, err := json.Marshal(input)
	if err != nil {
		return canonical()
	}
	var copyInput struct {
		Documents []Document
		Raw       map[string][]byte
		Indexes   map[string]int
	}
	if json.Unmarshal(key, &copyInput) != nil || !reflect.DeepEqual(input, copyInput) {
		return canonical()
	}
	compute := func() ([]byte, error) {
		if len(contexts) > 0 && contexts[0] != nil {
			contexts[0].countHistorical("workflow-package-graph-build")
		}
		out := []Document{}
		for _, d := range docs {
			if d.Kind != "page" {
				continue
			}
			if d.UnitScope == PageContract {
				d.StableIndex = indexes[d.Path]
				out = append(out, d)
				continue
			}
			v, err := FormalPage(d, raw[d.Path], indexes[d.Path])
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i].StableIndex != out[j].StableIndex {
				return out[i].StableIndex < out[j].StableIndex
			}
			return out[i].Path < out[j].Path
		})
		b, err := json.Marshal(out)
		if err != nil {
			return nil, err
		}
		var copy []Document
		if json.Unmarshal(b, &copy) != nil || !reflect.DeepEqual(out, copy) {
			return nil, fmt.Errorf("workflow projection codec is not lossless")
		}
		return b, nil
	}
	b, err := contentidentity.Transform("sitecontent/workflow-package-graph/v1", key, compute)
	if err != nil {
		return canonical()
	}
	var out []Document
	if json.Unmarshal(b, &out) != nil {
		return canonical()
	}
	pages := 0
	for _, d := range docs {
		if d.Kind == "page" {
			pages++
		}
	}
	if len(out) != pages {
		return canonical()
	}
	return out, raw, nil
}
