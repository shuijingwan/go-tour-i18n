package i18n

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"golang.org/x/tools/present"
)

// A source observation is data, not a currentness decision. Every entry and
// successful return re-enumerates and reads the complete source domain. No root,
// runtime reader, QC judgment or authorization verdict enters the derived graph.
type sourceCatalogFile struct {
	Path string
	Mode fs.FileMode
	Data []byte
}
type sourceCatalogObservation struct {
	Files                         []sourceCatalogFile
	Directories                   []string
	Articles                      []string
	FixedShape, ProjectStandalone bool
}
type sourceCatalogDeriver func(sourceCatalogObservation) (*Catalog, error)

func observeSourceCatalog(root string, fixed, project bool) (sourceCatalogObservation, []byte, error) {
	o := sourceCatalogObservation{Articles: append([]string(nil), ArticleOrder...), FixedShape: fixed, ProjectStandalone: project}
	for _, name := range o.Articles {
		if !utf8.ValidString(name) {
			return o, nil, fmt.Errorf("unobserved article name encoding")
		}
	}
	total := 0
	err := fs.WalkDir(os.DirFS(root), "_content/tour", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !utf8.ValidString(name) {
			return fmt.Errorf("unobserved source path encoding")
		}
		st, err := d.Info()
		if err != nil {
			return err
		}
		if st.IsDir() {
			o.Directories = append(o.Directories, name)
			return nil
		}
		if !st.Mode().IsRegular() {
			return fmt.Errorf("unobserved source file kind")
		}
		if st.Size() > 32<<20 || int64(total)+st.Size() > 32<<20 {
			return fmt.Errorf("source observation budget exceeded")
		}
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			return err
		}
		total += len(b)
		if total > 32<<20 {
			return fmt.Errorf("source observation budget exceeded")
		}
		o.Files = append(o.Files, sourceCatalogFile{name, st.Mode(), b})
		return nil
	})
	if err != nil {
		return o, nil, err
	}
	name := "UPSTREAM_MANIFEST.tsv"
	st, err := os.Lstat(filepath.Join(root, name))
	if err == nil {
		if !st.Mode().IsRegular() {
			return o, nil, fmt.Errorf("unobserved manifest file kind")
		}
		if st.Size() > 16<<20 {
			return o, nil, fmt.Errorf("manifest observation budget exceeded")
		}
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return o, nil, err
		}
		if len(b) > 16<<20 {
			return o, nil, fmt.Errorf("manifest observation budget exceeded")
		}
		o.Files = append(o.Files, sourceCatalogFile{name, st.Mode(), b})
	} else if !os.IsNotExist(err) {
		return o, nil, err
	}
	type fingerprint struct {
		Path, SHA256 string
		Mode         fs.FileMode
	}
	files := make([]fingerprint, 0, len(o.Files))
	for _, f := range o.Files {
		files = append(files, fingerprint{f.Path, sum(f.Data), f.Mode.Type()})
	}
	key, _ := json.Marshal(struct {
		Contract              string
		Files                 []fingerprint
		Directories, Articles []string
		Fixed, Project        bool
		Shift, Slice, Marker  string
	}{"source-catalog/environment/v1", files, o.Directories, o.Articles, fixed, project,
		goExampleShiftVerbRE.String(), goExampleSliceVerbRE.String(), presentCommentMarkerRE.String()})
	return o, key, nil
}

func observedSourceCatalog(root string, fixed, project bool, derive sourceCatalogDeriver) (*Catalog, error) {
	if !contentidentity.MemoizationActive() {
		return buildCatalog(root, fixed, project)
	}
	// Preserve the canonical parser configuration, including on a warm process.
	present.PlayEnabled = true
	o, key, err := observeSourceCatalog(root, fixed, project)
	if err != nil {
		return buildCatalog(root, fixed, project)
	}
	// Expected upstream hashes are a fresh comparison, never a cache verdict.
	if err := o.checkManifest(); err != nil {
		return buildCatalog(root, fixed, project)
	}
	compute := func() ([]byte, error) {
		c, err := derive(o)
		if err != nil {
			return nil, err
		}
		return encodeSourceCatalog(c)
	}
	b, err := contentidentity.Transform("source-catalog/environment/v1", key, compute)
	if err != nil {
		return buildCatalog(root, fixed, project)
	}
	c, err := decodeSourceCatalog(b)
	if err != nil || validateCatalog(c, fixed) != nil || len(c.Pages) == 0 {
		b, err = compute()
		if err != nil {
			return buildCatalog(root, fixed, project)
		}
		c, err = decodeSourceCatalog(b)
		if err != nil {
			return buildCatalog(root, fixed, project)
		}
	}
	_, fresh, err := observeSourceCatalog(root, fixed, project)
	if err != nil || !bytes.Equal(key, fresh) {
		// The Catalog is a prospective data projection, not an authorization
		// verdict. If its observation changed, use the canonical fresh reader;
		// outer source locks/currentness guards retain their existing decisions.
		return buildCatalog(root, fixed, project)
	}
	return c, nil
}

type sourceCatalogGraph struct {
	Pages                                 []Page
	Conditional                           []ConditionalPage
	Examples                              []Example
	NilPages, NilConditional, NilExamples bool
}

func decodeSourceCatalog(b []byte) (*Catalog, error) {
	var graph sourceCatalogGraph
	if err := gob.NewDecoder(bytes.NewReader(b)).Decode(&graph); err != nil {
		return nil, err
	}
	c := &Catalog{Pages: graph.Pages, Conditional: graph.Conditional, Examples: graph.Examples}
	if !graph.NilPages && c.Pages == nil {
		c.Pages = []Page{}
	}
	if !graph.NilConditional && c.Conditional == nil {
		c.Conditional = []ConditionalPage{}
	}
	if !graph.NilExamples && c.Examples == nil {
		c.Examples = []Example{}
	}
	return c, nil
}

func encodeSourceCatalog(c *Catalog) ([]byte, error) {
	var b bytes.Buffer
	graph := sourceCatalogGraph{c.Pages, c.Conditional, c.Examples, c.Pages == nil, c.Conditional == nil, c.Examples == nil}
	if err := gob.NewEncoder(&b).Encode(graph); err != nil {
		return nil, err
	}
	copy, err := decodeSourceCatalog(b.Bytes())
	if err != nil || !reflect.DeepEqual(c, copy) {
		return nil, fmt.Errorf("source graph codec is not lossless")
	}
	return b.Bytes(), nil
}

func (o sourceCatalogObservation) read(name string) ([]byte, error) {
	for _, f := range o.Files {
		if f.Path == name {
			return f.Data, nil
		}
	}
	return nil, os.ErrNotExist
}
func (o sourceCatalogObservation) examples() []string {
	var paths []string
	for _, f := range o.Files {
		name := strings.TrimPrefix(f.Path, "_content/tour/")
		if name != f.Path && filepath.Ext(name) == ".go" && !strings.HasPrefix(name, "solutions/") {
			paths = append(paths, name)
		}
	}
	sort.Strings(paths)
	return paths
}
func (o sourceCatalogObservation) manifest() (map[string]string, bool, error) {
	b, err := o.read("UPSTREAM_MANIFEST.tsv")
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return parseUpstreamLocalHashes(bytes.NewReader(b), "UPSTREAM_MANIFEST.tsv")
}
func (o sourceCatalogObservation) checkManifest() error {
	hashes, exists, err := o.manifest()
	if err != nil || !exists {
		return err
	}
	for _, name := range o.examples() {
		path := "_content/tour/" + name
		b, err := o.read(path)
		if err != nil || hashes[path] != sum(b) {
			return fmt.Errorf("example manifest/source identity differs")
		}
	}
	return nil
}

// Pure derivation from the observed byte world; no filesystem reads occur here.
func deriveSourceCatalog(o sourceCatalogObservation) (*Catalog, error) {
	parse := func(article string, source []byte) error {
		_, err := parsePresentPageWithReadFile(article, source, func(clean string) ([]byte, error) {
			return o.read(filepath.ToSlash(filepath.Join("_content/tour", clean)))
		})
		return err
	}
	return buildCatalogWith(o.Articles, o.FixedShape, o.ProjectStandalone,
		func(article string) ([]byte, error) { return o.read("_content/tour/" + article) }, parse,
		func(pages []Page) ([]Example, error) {
			return discoverExamplesWith(pages, parse, o.manifest,
				func() ([]string, error) { return o.examples(), nil }, o.read)
		})
}
