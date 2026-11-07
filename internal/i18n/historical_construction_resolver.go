package i18n

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

type historicalConstructionMaterial struct {
	Bytes    []byte
	Origin   string
	Evidence []RecoveryFileReference
}

func strictHistoricalSelection(b []byte, s *ApprovedIntervalSelection) error {
	return contentidentity.StrictJSON(b, s)
}

// One verified installed invocation owns this lookup context. No global SHA
// cache, archive map augmentation, latest scan, or external source lookup.
type historicalConstructionResolver struct {
	root         string
	catalog      *Catalog
	manifest     *RetranslationBatchManifest
	files        map[string][]byte
	graphs       map[string]*historicalBytesGraph
	materials    map[string]historicalConstructionMaterial
	observed     map[string]RecoveryFileReference
	supplemental map[string]RecoveryFileReference
}

func newHistoricalConstructionResolver(root string, c *Catalog, m *RetranslationBatchManifest, files map[string][]byte) *historicalConstructionResolver {
	return &historicalConstructionResolver{root: root, catalog: c, manifest: m, files: files, graphs: map[string]*historicalBytesGraph{}, materials: map[string]historicalConstructionMaterial{}, observed: map[string]RecoveryFileReference{}, supplemental: map[string]RecoveryFileReference{}}
}

func (r *historicalConstructionResolver) graph(id, path string) (*historicalBytesGraph, error) {
	base := r.graphs[id]
	if base == nil {
		var request ApprovedIntervalSelection
		if err := strictHistoricalSelection(r.files[approvedIntervalSelectionMember], &request); err != nil {
			return nil, err
		}
		g, err := loadHistoricalBytesGraph(r.root, r.catalog, HistoricalBytesOptions{Locale: r.manifest.Locale, ContentPackage: request.ContentPackage, BatchID: r.manifest.BatchID, UnitID: id, LogicalPath: path})
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(g.Files["manifest.json"], r.files["manifest.json"]) || !bytes.Equal(g.Files["formal/manifest.json"], r.files["formal/manifest.json"]) || !bytes.Equal(g.Files[approvedIntervalProofMember(id)], r.files[approvedIntervalProofMember(id)]) {
			return nil, hbError("WRONG_SCOPE", "material resolver context is not the exact installed archive")
		}
		base = g
		r.graphs[id] = base
		for _, ref := range g.Observed {
			r.observed[ref.Ref.Path] = ref.Ref
		}
	}
	if base.Declaration.LogicalPath == path {
		return base, nil
	}
	g := *base
	d := base.Declaration
	d.LogicalPath = path
	d.SHA256 = ""
	d.Size = nil
	d.Role = ""
	if d.Producer.RoleOrigin == "explicit" {
		x, err := decodeDerivedTransportConstruction(r.catalog, base.Files[transportRecoveryConstructionMember])
		if err != nil {
			return nil, err
		}
		for _, ref := range x.Producer.Implementation {
			if ref.Ref.Path == path {
				d.SHA256 = ref.Ref.SHA256
				n := ref.Ref.Size
				d.Size = &n
				d.Role = ref.Role
			}
		}
	} else {
		p, err := recoveryProofFromFiles(base.Files, id)
		if err != nil {
			return nil, err
		}
		for _, ref := range p.Dependencies {
			if ref.Path == path {
				d.SHA256 = ref.SHA256
				d.Role = "approved-construction-implementation"
			}
		}
	}
	if !validSHA256(d.SHA256) || !historicalBytesRole(d.Producer, d.Role, path) {
		return nil, hbError("NO_DECLARATION", "no declaration in verified resolver inventory")
	}
	g.Declaration = d
	return &g, nil
}

func (r *historicalConstructionResolver) Read(id string, ref GlossaryArchiveReference, role string, version int) ([]byte, error) {
	key := id + "\x00" + role + "\x00" + ref.Path + "\x00" + ref.SHA256 + "\x00" + fmt.Sprint(version)
	if m, ok := r.materials[key]; ok {
		return m.Bytes, nil
	}
	g, err := r.graph(id, ref.Path)
	if err != nil {
		return nil, err
	}
	d := g.Declaration
	if d.SHA256 != ref.SHA256 || d.Role != role || d.Producer.Version != version {
		return nil, hbError("WRONG_SCOPE", "material declaration/ref/role/version mismatch")
	}
	b, origin, evidence, err := resolveHistoricalBytesMaterial(r.root, r.catalog, g, false)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, fmt.Errorf("missing declared historical construction bytes: %s", ref.Path)
	}
	for _, ref := range evidence {
		r.supplemental[ref.Path] = ref
	}
	r.materials[key] = historicalConstructionMaterial{b, origin, evidence}
	return b, nil
}

func (r *historicalConstructionResolver) ReadProducer(ref GlossaryArchiveReference, role string, version int) ([]byte, error) {
	// Every participating approved Unit needs its own exact declaration. A
	// matching blob under another Unit's index is not transferable authority.
	var out []byte
	for _, record := range r.manifest.Units {
		if len(r.files[approvedIntervalProofMember(record.UnitID)]) == 0 {
			continue
		}
		b, err := r.Read(record.UnitID, ref, role, version)
		if err != nil {
			return nil, err
		}
		if out != nil && !bytes.Equal(out, b) {
			return nil, hbError("CORRUPT_RECORDED_EVIDENCE", "shared producer declarations conflict")
		}
		out = b
	}
	if out == nil {
		return nil, hbError("NO_DECLARATION", "producer has no approved declaration")
	}
	return out, nil
}

func (r *historicalConstructionResolver) References() ([]RecoveryDependency, error) {
	var out []RecoveryDependency
	for _, ref := range r.observed {
		out = append(out, RecoveryDependency{"historical-bytes-material-context", ref})
	}
	for _, ref := range r.supplemental {
		out = append(out, RecoveryDependency{"historical-bytes-material-evidence", ref})
	}
	if len(r.materials) > 0 {
		refs, err := historicalBytesVerifierRefs(r.root)
		if err != nil {
			return nil, err
		}
		for _, ref := range refs {
			out = append(out, RecoveryDependency{"historical-bytes-material-reader", ref})
		}
	}
	return sortedRecoveryDependencies(out)
}

func (r *historicalConstructionResolver) Verify() error {
	var paths []string
	for p := range r.supplemental {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		ref := r.supplemental[p]
		max := int64(historicalBytesMaxEvidence)
		if _, err := readHistoricalBytesStored(r.root, ref, max); err != nil {
			return err
		}
	}
	refs := []RecoveryFileReference{}
	for _, ref := range r.observed {
		refs = append(refs, ref)
	}
	return verifyRecoveryByteReferences(r.root, refs)
}

func historicalBytesGenerationInputs(root string, c *Catalog, a *InstalledDerivedTransportRecovery) ([]generationBundleInput, error) {
	var out []generationBundleInput
	expected := map[string]RecoveryFileReference{}
	for _, r := range a.CurrentEpoch {
		if r.Role == "historical-bytes-material-evidence" {
			expected[r.Ref.Path] = r.Ref
		}
	}
	seen := map[string]bool{}
	for _, r := range a.CurrentEpoch {
		if r.Role == "historical-bytes-material-reader" {
			b, err := readCompatibilityReference(root, GlossaryArchiveReference{r.Ref.Path, r.Ref.SHA256})
			if err != nil || int64(len(b)) != r.Ref.Size {
				return nil, fmt.Errorf("historical byte verifier implementation drift")
			}
			out = append(out, generationBundleInput{"authority/historical-construction-bytes/" + r.Ref.Path, r.Ref.Path, b})
		}
		if r.Role != "historical-bytes-material-evidence" || !strings.HasPrefix(r.Ref.Path, historicalBytesRoot+"/authorities/") || filepath.Base(r.Ref.Path) != "authority.json" {
			continue
		}
		b, err := readHistoricalBytesStored(root, r.Ref, historicalBytesMaxAuthority)
		if err != nil {
			return nil, err
		}
		authority, err := decodeHistoricalBytesAuthority(b)
		if err != nil {
			return nil, err
		}
		_, _, refs, err := loadCommittedHistoricalBytes(root, c, authority.Statement.declaration(), false)
		if err != nil {
			return nil, err
		}
		names := []string{"index.json", "authority.json", "bytes.bin", "acquisition-evidence.bin"}
		for i, ref := range refs {
			if expected[ref.Path] != ref {
				return nil, hbError("WRONG_SCOPE", "recovery epoch does not bind complete supplemental group")
			}
			data, err := readHistoricalBytesStored(root, ref, historicalBytesMaxEvidence)
			if err != nil {
				return nil, err
			}
			out = append(out, generationBundleInput{"formal/historical-construction-bytes/" + authority.Identity + "/" + names[i], ref.Path, data})
			seen[ref.Path] = true
		}
	}
	if len(seen) != len(expected) {
		return nil, hbError("WRONG_SCOPE", "unbound supplemental inventory member")
	}
	return out, nil
}
