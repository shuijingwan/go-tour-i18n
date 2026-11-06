package sitecontent

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Complete data inputs of BuildGlobal, not a prior/current Global acceptance.
// The registered Global and successor authorities are compared by the existing
// callers after this projection; they never enter this node as a PASS verdict.
type globalSourceObservation struct {
	Routes, LearnRoots, Contracts []string
	Shell, Tour                   []Source
}

func observeGlobalSource(root string) (globalSourceObservation, error) {
	o := globalSourceObservation{LearnRoots: append([]string(nil), learnRoots...), Contracts: append([]string(nil), contractPaths...)}
	c, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		return o, err
	}
	for _, p := range c.Pages {
		o.Routes = append(o.Routes, p.Route)
	}
	for _, s := range ShellSources() {
		b, err := readRegular(root, s.Path)
		if err != nil {
			return o, err
		}
		s.Origin, s.SHA256, s.Dependencies = "repository", digest(b), []Dependency{}
		o.Shell = append(o.Shell, s)
	}
	files, err := listFiles(os.DirFS(root), "_content/tour")
	if err != nil {
		return o, err
	}
	for _, name := range files {
		b, err := readRegular(root, name)
		if err != nil {
			return o, err
		}
		kind := "asset"
		if path.Ext(name) == ".article" {
			kind = "page"
		}
		o.Tour = append(o.Tour, Source{Path: name, Kind: kind, Origin: "repository", Package: "tour-v1", Surface: "tour", SHA256: digest(b), Dependencies: []Dependency{}})
	}
	return o, nil
}

func observedGlobalSource(root string, snapshot []byte, contexts ...*structuredSuccessorValidation) (*Global, error) {
	if !contentidentity.MemoizationActive() {
		return buildGlobalCanonical(root, snapshot, contexts...)
	}
	o, err := observeGlobalSource(root)
	if err != nil {
		return buildGlobalCanonical(root, snapshot, contexts...)
	}
	key, err := json.Marshal(struct {
		Snapshot string
		Inputs   globalSourceObservation
	}{digest(snapshot), o})
	if err != nil {
		return buildGlobalCanonical(root, snapshot, contexts...)
	}
	var copied struct {
		Snapshot string
		Inputs   globalSourceObservation
	}
	if json.Unmarshal(key, &copied) != nil || copied.Snapshot != digest(snapshot) || !reflect.DeepEqual(o, copied.Inputs) {
		// Lossy encodings are never semantic cache keys (including nil/empty
		// distinctions or unsupported strings). Keep the canonical reader.
		return buildGlobalCanonical(root, snapshot, contexts...)
	}
	if len(contexts) > 0 && contexts[0] != nil {
		contexts[0].countHistorical("BuildGlobal")
		contexts[0].countHistorical("BuildSourceCatalog")
	}
	compute := func() ([]byte, error) {
		if len(contexts) > 0 && contexts[0] != nil {
			contexts[0].countHistorical("global-environment-build")
			contexts[0].countHistorical("snapshotFS")
		}
		g, err := deriveGlobalSource(snapshot, o)
		if err != nil {
			return nil, err
		}
		return encodeGlobalEnvironment(g)
	}
	b, err := contentidentity.Transform("sitecontent/global-source-environment/v1", key, compute)
	if err != nil {
		return buildGlobalCanonical(root, snapshot, contexts...)
	}
	g, err := decodeGlobalEnvironment(b)
	if err != nil || ValidateGlobal(*g) != nil {
		return buildGlobalCanonical(root, snapshot, contexts...)
	}
	fresh, err := observeGlobalSource(root)
	if err != nil || !reflect.DeepEqual(o, fresh) {
		// Preserve canonical diagnostics and the existing outer freshness gates.
		return buildGlobalCanonical(root, snapshot, contexts...)
	}
	return g, nil
}

func encodeGlobalEnvironment(g *Global) ([]byte, error) {
	b, err := json.Marshal(g)
	if err != nil {
		return nil, err
	}
	copy, err := decodeGlobalEnvironment(b)
	if err != nil || !reflect.DeepEqual(g, copy) {
		return nil, fmt.Errorf("Global data codec is not lossless")
	}
	return b, nil
}

func decodeGlobalEnvironment(b []byte) (*Global, error) {
	var g Global
	if err := json.Unmarshal(b, &g); err != nil {
		return nil, err
	}
	return &g, nil
}
