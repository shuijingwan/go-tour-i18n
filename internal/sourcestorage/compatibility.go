package sourcestorage

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

const CompatibilitySchema = "go-learning/source-storage-compatibility/v1"
const CompatibilityRoot = "data/source-storage-compatibility"

type CorpusBinding struct {
	Reference        contentidentity.Reference `json:"corpus"`
	SemanticIdentity string                    `json:"corpus_identity"`
	RequiredEntries  []string                  `json:"required_entries"`
}
type EquivalentEntry struct {
	OldPath string `json:"old_entry_path"`
	OldSHA  string `json:"old_entry_sha256"`
	NewPath string `json:"new_repo_path"`
	NewSHA  string `json:"new_entry_sha256"`
	Role    string `json:"role"`
}
type Compatibility struct {
	Schema           string                    `json:"schema"`
	OldStorage       contentidentity.Reference `json:"old_storage"`
	Manifest         contentidentity.Reference `json:"site_source_manifest"`
	ManifestIdentity string                    `json:"site_source_manifest_identity"`
	Corpora          []CorpusBinding           `json:"corpora"`
	Entries          []EquivalentEntry         `json:"equivalent_entries"`
	Equality         string                    `json:"equality"`
	Identity         string                    `json:"identity_sha256"`
}

func SealCompatibility(p *Compatibility) {
	p.Identity = ""
	p.Identity = identity(*p)
}
func CompatibilityPath(p Compatibility) string {
	return CompatibilityRoot + "/" + p.OldStorage.SHA256 + "/" + p.Identity + ".json"
}

func ValidateCompatibility(p Compatibility) error {
	copy := p
	copy.Identity = ""
	if p.Schema != CompatibilitySchema || p.Identity != identity(copy) || p.Equality != "byte-exact" || p.OldStorage.Path != LegacyZIPPath || !validSHA(p.OldStorage.SHA256) || p.Manifest.Path != ManifestPath || !validSHA(p.Manifest.SHA256) || !validSHA(p.ManifestIdentity) || len(p.Corpora) == 0 || len(p.Entries) == 0 {
		return fmt.Errorf("invalid storage compatibility identity")
	}
	union := map[string]bool{}
	previous := ""
	for _, c := range p.Corpora {
		path := c.Reference.Path
		if path <= previous || !validPath(path) || !validSHA(c.Reference.SHA256) || !validSHA(c.SemanticIdentity) || len(c.RequiredEntries) == 0 {
			return fmt.Errorf("invalid/duplicate corpus binding")
		}
		if path != "data/glossary-source-corpus.json" && path != "data/glossary-corpus-history/"+c.Reference.SHA256+".json" {
			return fmt.Errorf("non-canonical corpus binding")
		}
		previous = path
		last := ""
		for _, entry := range c.RequiredEntries {
			if !validPath(entry) || !strings.HasPrefix(entry, "_content/") || entry <= last {
				return fmt.Errorf("ambiguous/duplicate required entry")
			}
			last = entry
			union[entry] = true
		}
	}
	previous = ""
	for _, e := range p.Entries {
		if e.OldPath <= previous || !union[e.OldPath] || e.NewPath != e.OldPath || !validSHA(e.OldSHA) || e.OldSHA != e.NewSHA || (e.Role != "website-source" && e.Role != "website-data") {
			return fmt.Errorf("invalid/incomplete/non-equivalent storage entry")
		}
		previous = e.OldPath
		delete(union, e.OldPath)
	}
	if len(union) != 0 {
		return fmt.Errorf("compatibility does not cover the exact required entry set")
	}
	return nil
}

// ReadOldZIP retains the original physical hash/path gate. A present corrupt
// ZIP must never be replaced by proof or current bytes.
func ReadOldZIP(root string, ref contentidentity.Reference) ([]byte, error) {
	if ref.Path != LegacyZIPPath || !validSHA(ref.SHA256) {
		return nil, fmt.Errorf("unexpected old storage reference")
	}
	b, err := readRegular(root, ref.Path)
	if err != nil {
		return nil, err
	}
	if digest(b) != ref.SHA256 {
		return nil, fmt.Errorf("old physical ZIP hash changed")
	}
	return b, nil
}

func ZIPEntries(b []byte) (map[string][]byte, error) {
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return nil, err
	}
	result := map[string][]byte{}
	total := int64(0)
	for _, f := range z.File {
		if !validPath(f.Name) || !f.Mode().IsRegular() || f.UncompressedSize64 > 16<<20 {
			return nil, fmt.Errorf("unsafe old ZIP entry")
		}
		if _, ok := result[f.Name]; ok {
			return nil, fmt.Errorf("duplicate old ZIP entry")
		}
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(io.LimitReader(r, (16<<20)+1))
		closeErr := r.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		total += int64(len(data))
		if len(data) > 16<<20 || total > 64<<20 {
			return nil, fmt.Errorf("old ZIP exceeds registered transport boundary")
		}
		result[f.Name] = data
	}
	return result, nil
}

func readCompatibility(root, path string) (*Compatibility, error) {
	b, err := readRegular(root, path)
	if err != nil {
		return nil, err
	}
	var p Compatibility
	if err := contentidentity.StrictJSON(b, &p); err != nil {
		return nil, err
	}
	if err := ValidateCompatibility(p); err != nil {
		return nil, err
	}
	canonical, _ := Encode(p)
	if path != CompatibilityPath(p) || !bytes.Equal(b, canonical) {
		return nil, fmt.Errorf("non-canonical storage compatibility proof")
	}
	return &p, nil
}

// VerifyCompatibility validates individual adopted bytes, corpus memberships
// and the exact current manifest. It deliberately does not read unused ZIP
// entries from current storage: they are not glossary semantic dependencies.
func VerifyCompatibility(root string, p Compatibility, physical []byte) error {
	return verifyCompatibility(root, p, physical, nil)
}

func verifyCompatibility(root string, p Compatibility, physical []byte, selected *CorpusBinding) error {
	return verifyCompatibilityAt(root, p, physical, selected, "")
}

func verifyCompatibilityAt(root string, p Compatibility, physical []byte, selected *CorpusBinding, archivedPath string) error {
	if err := ValidateCompatibility(p); err != nil {
		return err
	}
	m, err := LoadManifest(root)
	if err != nil {
		return err
	}
	manifest, err := readRegular(root, ManifestPath)
	if err != nil {
		return err
	}
	if m.Identity != p.ManifestIdentity || digest(manifest) != p.Manifest.SHA256 || m.Legacy.Path != p.OldStorage.Path || m.Legacy.SHA256 != p.OldStorage.SHA256 {
		return fmt.Errorf("storage compatibility manifest is stale")
	}
	if _, err := os.Lstat(pathJoin(root, MigrationLock)); !os.IsNotExist(err) {
		return fmt.Errorf("source migration pending")
	}
	for _, c := range p.Corpora {
		if selected != nil && c.Reference != selected.Reference {
			continue
		}
		name := c.Reference.Path
		if archivedPath != "" {
			if selected == nil || c.Reference.Path != "data/glossary-source-corpus.json" || archivedPath != "data/glossary-corpus-history/"+c.Reference.SHA256+".json" {
				return fmt.Errorf("invalid exact archival storage alias")
			}
			name = archivedPath
		}
		b, err := readRegular(root, name)
		if err != nil {
			return err
		}
		if digest(b) != c.Reference.SHA256 {
			return fmt.Errorf("bound corpus bytes changed")
		}
	}
	files := map[string]FrozenSourceFile{}
	registered := map[string]bool{}
	for _, f := range m.Files {
		files[f.Path] = f
	}
	for _, name := range m.Legacy.Paths {
		registered[name] = true
	}
	var old map[string][]byte
	if physical != nil {
		if digest(physical) != p.OldStorage.SHA256 {
			return fmt.Errorf("old ZIP/proof hash mismatch")
		}
		old, err = ZIPEntries(physical)
		if err != nil {
			return err
		}
	}
	for _, e := range p.Entries {
		if selected != nil {
			i := sort.SearchStrings(selected.RequiredEntries, e.OldPath)
			if i == len(selected.RequiredEntries) || selected.RequiredEntries[i] != e.OldPath {
				continue
			}
		}
		f, ok := files[e.NewPath]
		if !ok || !registered[e.OldPath] || f.UpstreamPath != e.OldPath || f.Classification != "upstream-exact" || f.Role != e.Role || f.SHA256 != e.NewSHA || f.UpstreamSHA256 != e.OldSHA {
			return fmt.Errorf("entry path/role/provenance is not exact: %s", e.OldPath)
		}
		current, err := ReadAdoptedSource(root, f)
		if err != nil {
			return err
		}
		if physical != nil {
			prior, ok := old[e.OldPath]
			if !ok || digest(prior) != e.OldSHA || !bytes.Equal(prior, current) {
				return fmt.Errorf("old/new entry bytes differ: %s", e.OldPath)
			}
		}
	}
	return nil
}

// Historical proofs may bind an older manifest. While the physical ZIP exists,
// even those proofs must agree with its exact entries before being skipped as
// non-current storage authorization.
func verifyPhysicalEntries(p Compatibility, physical []byte) error {
	if digest(physical) != p.OldStorage.SHA256 {
		return fmt.Errorf("old ZIP/proof hash mismatch")
	}
	entries, err := ZIPEntries(physical)
	if err != nil {
		return err
	}
	for _, e := range p.Entries {
		b, ok := entries[e.OldPath]
		if !ok || digest(b) != e.OldSHA {
			return fmt.Errorf("proof disagrees with physical old entry: %s", e.OldPath)
		}
	}
	return nil
}

// CheckCorpusReference only resolves this specific retired storage reference,
// through a proof binding the actual corpus file and its complete requirements.
// Other references retain their own original readers and hash gates.
func CheckCorpusReference(root string, corpus contentidentity.Reference, semantic string, storage contentidentity.Reference, required []string) error {
	return checkCorpusReference(root, corpus, semantic, storage, required, false)
}

func checkCorpusReference(root string, corpus contentidentity.Reference, semantic string, storage contentidentity.Reference, required []string, requireProof bool) error {
	physical, physicalErr := ReadOldZIP(root, storage)
	if physicalErr != nil && !os.IsNotExist(physicalErr) {
		return physicalErr
	}
	dir := CompatibilityRoot + "/" + storage.SHA256
	entries, err := os.ReadDir(pathJoin(root, dir))
	if os.IsNotExist(err) && physicalErr == nil && !requireProof {
		return nil
	}
	if err != nil {
		return fmt.Errorf("exact storage compatibility proof missing: %w", err)
	}
	current, err := LoadManifest(root)
	if err != nil {
		return err
	}
	matches := 0
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".json") {
			return fmt.Errorf("unsafe storage compatibility artifact")
		}
		p, err := readCompatibility(root, dir+"/"+entry.Name())
		if err != nil {
			return err
		}
		if p.OldStorage != storage {
			return fmt.Errorf("proof old storage binding differs")
		}
		if physical != nil {
			if err := verifyPhysicalEntries(*p, physical); err != nil {
				return err
			}
		}
		if p.ManifestIdentity != current.Identity {
			continue
		}
		hasDirectBinding := false
		for _, c := range p.Corpora {
			if c.Reference == corpus {
				hasDirectBinding = true
			}
		}
		for _, c := range p.Corpora {
			// Canonical content-addressed archival of the exact original current
			// bytes inherits its existing storage proof, never later current bytes.
			archivedExact := !hasDirectBinding && corpus.Path == "data/glossary-corpus-history/"+corpus.SHA256+".json" && c.Reference.Path == "data/glossary-source-corpus.json" && c.Reference.SHA256 == corpus.SHA256
			if c.Reference == corpus || archivedExact {
				if c.SemanticIdentity != semantic || !samePaths(c.RequiredEntries, required) {
					return fmt.Errorf("proof corpus membership/semantic identity differs")
				}
				// An archive verifies its own exact membership and provenance;
				// later current corpus bytes are not an archive dependency.
				archivePath := ""
				if archivedExact {
					archivePath = corpus.Path
				}
				if err := verifyCompatibilityAt(root, *p, physical, &c, archivePath); err != nil {
					return err
				}
				matches++
			}
		}
	}
	if matches == 0 && physical != nil && !requireProof {
		return nil // Original physical verification remains sufficient.
	}
	if matches != 1 {
		return fmt.Errorf("current exact storage compatibility proof missing/ambiguous")
	}
	return nil
}

func samePaths(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// HasCurrentProof is also used by retirement preflight, where proof is required
// even while the physical ZIP still exists.
func HasCurrentProof(root string, corpus contentidentity.Reference, semantic string, storage contentidentity.Reference, required []string) error {
	return checkCorpusReference(root, corpus, semantic, storage, required, true)
}

func SortBindings(bindings []CorpusBinding) {
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].Reference.Path < bindings[j].Reference.Path })
}
