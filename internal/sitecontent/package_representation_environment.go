package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

type packageRepresentationInput struct {
	Package     Package
	Sources     []Source
	SnapshotSHA string
	Raw         []Reference
}

type packageRepresentationData struct {
	Documents []Document
	Raw       map[string][]byte
}

// Only the immutable base representation is derived here. The caller still
// applies/validates current structured successors after this node, and every
// membership/adoption reference is read by its original authority reader.
func sharedPackageRepresentation(root string, g *Global, id string, contexts ...*structuredSuccessorValidation) ([]Document, map[string][]byte, error) {
	canonical := func() ([]Document, map[string][]byte, error) { return PackageDocuments(root, g, id, contexts...) }
	if !contentidentity.MemoizationActive() {
		return canonical()
	}
	p, err := packageByID(g, id)
	if err != nil {
		return nil, nil, err
	}
	if g.Membership != nil {
		var m PackageMembership
		if err := readReference(root, *g.Membership, &m); err != nil {
			return nil, nil, err
		}
		if id == m.Package {
			base, err := loadSourceCurrent(root, contexts...)
			if err != nil {
				return nil, nil, err
			}
			docs := []Document{}
			raw := map[string][]byte{}
			for _, old := range m.Predecessors {
				part, files, err := sharedPackageRepresentation(root, base, old.Package, contexts...)
				if err != nil {
					return nil, nil, err
				}
				docs = append(docs, part...)
				for name, b := range files {
					if _, ok := raw[name]; ok {
						return nil, nil, fmt.Errorf("membership output overlap")
					}
					raw[name] = b
				}
			}
			return docs, raw, nil
		}
	}
	if id != "site-v2-shell" && id != "learn-docs-v1" {
		return canonical()
	}
	input := packageRepresentationInput{Package: p}
	raw := map[string][]byte{}
	var snapshot []byte
	if id == "learn-docs-v1" {
		snapshot, err = ReadSourceSnapshot(root)
		if err != nil {
			return nil, nil, err
		}
		input.SnapshotSHA = digest(snapshot)
	}
	for _, s := range g.Sources {
		if s.Package != id || (id == "learn-docs-v1" && s.Kind != "page" && s.Kind != "data") {
			continue
		}
		input.Sources = append(input.Sources, s)
		if id == "site-v2-shell" {
			b, err := readRegularReference(root, currentSourceReference(s))
			if err != nil {
				return nil, nil, err
			}
			raw[s.Path] = b
			input.Raw = append(input.Raw, Reference{s.Path, digest(b)})
		}
	}
	key, err := json.Marshal(input)
	if err != nil {
		return canonical()
	}
	var copyInput packageRepresentationInput
	if json.Unmarshal(key, &copyInput) != nil || !reflect.DeepEqual(input, copyInput) {
		return canonical()
	}
	compute := func() ([]byte, error) {
		if len(contexts) > 0 && contexts[0] != nil {
			contexts[0].countHistorical("package-representation-build")
		}
		data, err := derivePackageRepresentation(input, snapshot, raw)
		if err != nil {
			return nil, err
		}
		b, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		var copy packageRepresentationData
		if json.Unmarshal(b, &copy) != nil || !reflect.DeepEqual(data, &copy) {
			return nil, fmt.Errorf("package representation codec is not lossless")
		}
		return b, nil
	}
	b, err := contentidentity.Transform("sitecontent/package-representation-environment/v1", key, compute)
	if err != nil {
		return canonical()
	}
	var data packageRepresentationData
	if json.Unmarshal(b, &data) != nil || len(data.Documents) != len(input.Sources) || len(data.Raw) != len(input.Sources) {
		return canonical()
	}
	if id == "learn-docs-v1" {
		fresh, err := ReadSourceSnapshot(root)
		if err != nil || !bytes.Equal(fresh, snapshot) {
			return canonical()
		}
	} else {
		for _, s := range input.Sources {
			b, err := readRegularReference(root, currentSourceReference(s))
			if err != nil || !bytes.Equal(b, raw[s.Path]) {
				return canonical()
			}
		}
	}
	return data.Documents, data.Raw, nil
}

// No root, currentness callback or authority reader enters this calculation.
func derivePackageRepresentation(input packageRepresentationInput, snapshot []byte, raw map[string][]byte) (*packageRepresentationData, error) {
	data := &packageRepresentationData{Documents: []Document{}, Raw: map[string][]byte{}}
	var frozen fs.FS
	var err error
	if input.Package.ID == "learn-docs-v1" {
		frozen, err = snapshotFS(snapshot)
		if err != nil {
			return nil, err
		}
	}
	for _, s := range input.Sources {
		b := raw[s.Path]
		if frozen != nil {
			b, err = fs.ReadFile(frozen, s.Path)
			if err != nil {
				return nil, err
			}
		}
		d, err := ParseDocument(s, b)
		if err != nil {
			if frozen != nil {
				return nil, fmt.Errorf("%s: %w", s.Path, err)
			}
			return nil, err
		}
		data.Documents = append(data.Documents, *d)
		data.Raw[s.Path] = b
	}
	// Preserve the existing base-provider exact-set contracts. Membership
	// composition and new providers are handled by their authority readers.
	if input.Package.ID == "site-v2-shell" && len(data.Documents) != 2 {
		return nil, fmt.Errorf("shell requires exact homepage + translation sources")
	}
	if input.Package.ID == "learn-docs-v1" && len(data.Documents) != 56 {
		return nil, fmt.Errorf("frozen localizable exact-set must be 49 page + 7 data")
	}
	return data, nil
}
