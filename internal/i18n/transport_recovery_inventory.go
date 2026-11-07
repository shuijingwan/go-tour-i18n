package i18n

import (
	"fmt"
	"path/filepath"
)

// Preserve the old bytes required by the versioned construction reader. A
// digest alone cannot reconstruct a changed implementation after installation.
// Existing members remain unchanged; this adds explicit dependency evidence to
// newly exported approved Bundles and never repairs a historical archive.
func freezeApprovedConstructionDependencies(root string, inputs []generationBundleInput) ([]generationBundleInput, error) {
	present := map[string]bool{}
	for _, in := range inputs {
		present[sum(in.data)] = true
	}
	var refs []GlossaryArchiveReference
	for _, in := range inputs {
		if !isApprovedProofMember(in.path) {
			continue
		}
		p, err := recoveryProofFromFiles(map[string][]byte{in.path: in.data}, filepath.Base(in.path)[:len(filepath.Base(in.path))-len(".json")])
		if err != nil {
			return nil, err
		}
		if p == nil {
			return nil, fmt.Errorf("missing approved proof")
		}
		refs = append(refs, p.Dependencies...)
	}
	for _, ref := range refs {
		if present[ref.SHA256] {
			continue
		}
		b, err := readCompatibilityReference(root, ref)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, generationBundleInput{"formal/derived-transport-dependencies/" + ref.SHA256 + "-" + filepath.Base(ref.Path), ref.Path, b})
		present[ref.SHA256] = true
	}
	return inputs, nil
}

func isApprovedProofMember(path string) bool {
	return filepath.ToSlash(filepath.Dir(path)) == "formal/approved-interval-transport" && filepath.Ext(path) == ".json"
}
