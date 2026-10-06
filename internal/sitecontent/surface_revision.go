package sitecontent

import (
	"fmt"
	"reflect"
)

// FindingScope is explicit; free prose never mechanically authorizes arbitrary
// replacement files or Page units. This is not a synthetic QC rating.
type SurfaceFinding struct {
	ID      string `json:"id"`
	Finding string `json:"finding"`
}

func validatedSurfaceFinding(root, locale string, ref Reference) (*IntegratedSurfaceReceipt, error) {
	var r IntegratedSurfaceReceipt
	if err := readReference(root, ref, &r); err != nil {
		return nil, err
	}
	copy := r
	copy.Identity = ""
	if r.Identity != identity(copy) || r.Schema != IntegratedSurfaceSchema || r.Scope.Locale != locale || r.Decision != "failed" || len(r.ExactFindings) == 0 {
		return nil, fmt.Errorf("requires exact independent failed Surface finding")
	}
	if ref.Path != "data/locale-language/"+locale+"/surface/"+r.ReviewID+".json" || !component(r.Reviewer) || !component(r.ReviewID) {
		return nil, fmt.Errorf("invalid Surface finding path/session")
	}
	if err := requireIndependentSession(root, locale, r.Reviewer, "reviewer"); err != nil {
		return nil, err
	}
	b, err := readRegular(root, r.Evidence.Path)
	if err != nil || digest(b) != r.Evidence.SHA256 || textBytes(b) != nil {
		return nil, fmt.Errorf("Surface finding evidence changed")
	}
	want, err := BuildIntegratedSurfaceScope(root, locale, r.Scope.Closures)
	if err != nil || !reflect.DeepEqual(r.Scope, *want) {
		return nil, fmt.Errorf("Surface finding context no longer current")
	}
	allowed, seen := map[string]bool{}, map[string]bool{}
	for _, a := range r.Scope.Reviewed {
		allowed[a.ID] = true
	}
	for _, f := range r.ExactFindings {
		if !allowed[f.ID] || seen[f.ID] || f.Finding == "" {
			return nil, fmt.Errorf("invalid exact Surface finding")
		}
		seen[f.ID] = true
	}
	return &r, nil
}

func surfaceRevisionPages(root, locale, pkg string, ref Reference, docs []Document) ([]string, error) {
	r, err := validatedSurfaceFinding(root, locale, ref)
	if err != nil {
		return nil, err
	}
	allowed := map[string]bool{}
	for _, a := range r.Scope.Reviewed {
		if a.Package == pkg {
			allowed[a.ID] = true
		}
	}
	ids := []string{}
	for _, finding := range r.ExactFindings {
		if !allowed[finding.ID] {
			return nil, fmt.Errorf("Surface finding outside reviewed scope")
		}
		for _, d := range docs {
			if d.Path == finding.ID {
				ids = append(ids, d.Units[0].ID)
			}
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("no formal Page finding")
	}
	return uniqueStrings(ids), nil
}
