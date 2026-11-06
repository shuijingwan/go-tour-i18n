package sitecontent

import (
	"fmt"
	"os"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Frozen historical Global only; current shell/source projections never enter
// this key. Every hit revalidates dependency bytes, inventory and source routes.
type historicalGlobalKey struct {
	Authority, Snapshot, Manifest   Reference
	Mode, Parser, Inventory, Routes string
}

type historicalGlobalProof struct {
	key   historicalGlobalKey
	bytes []byte
}

func (v *structuredSuccessorValidation) historicalGlobal(root string, expected ...Reference) (*Global, error) {
	if root != v.guard.root {
		return nil, fmt.Errorf("historical validation root changed")
	}
	b, err := v.guard.readRegular(root, GlobalPath)
	if err != nil {
		return nil, err
	}
	ref := Reference{GlobalPath, digest(b)}
	if len(expected) != 0 && expected[0] != ref {
		return nil, fmt.Errorf("historical Global authority mismatch")
	}
	var declared Global
	if err := StrictJSON(b, &declared); err != nil {
		return nil, err
	}
	if err := ValidateGlobal(declared); err != nil {
		return nil, err
	}
	if err := v.observe(declared); err != nil {
		return nil, err
	}
	key := historicalGlobalKey{Authority: ref, Snapshot: Reference{SnapshotPath, declared.SnapshotSHA256}, Mode: "frozen-historical-global/v1", Parser: GlobalSchema + "\n" + UnitContract + "\n" + PageContract}
	for _, name := range []string{SourceManifestPath, SnapshotPath} {
		if err := v.guard.optional(name); err != nil {
			return nil, err
		}
	}
	if _, absent := v.guard.absent[SourceManifestPath]; !absent {
		key.Manifest = Reference{SourceManifestPath, v.guard.deps[SourceManifestPath]}
		// A closed installed-corpus phase has already observed this complete
		// manifest's files before constructing its first historical node. The
		// manifest bytes above are freshly compared on EVERY entry; the owner
		// rereads the whole captured file/mode/absence set before returning.
		// Default/eager readers retain the original per-entry observation.
		if (!v.currentSourcePhase && !v.registrationReuse) || !v.readonlyOwner || v.historical == nil {
			v.countHistorical("manifest-file-observation")
			m, err := loadSourceManifest(root)
			if err != nil {
				return nil, err
			}
			for _, file := range m.Files {
				if err := observeManifestSourcePaths(file.Path, v.observeSourcePath); err != nil {
					return nil, err
				}
			}
		}
	}
	if _, absent := v.guard.absent[SnapshotPath]; !absent {
		if err := v.guard.bind(key.Snapshot); err != nil {
			return nil, err
		}
	}
	// BuildGlobal also consumes the live Tour inventory and source-catalog
	// routes. Re-enumeration detects additions, not just changed existing files;
	// the canonical source catalog revalidates its own registry dependencies.
	files, err := listFiles(os.DirFS(root), "_content/tour")
	if err != nil {
		return nil, err
	}
	for _, name := range files {
		if err := v.guard.optional(name); err != nil {
			return nil, err
		}
	}
	key.Inventory = identity(files)
	v.countHistorical("BuildSourceCatalog")
	catalog, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		return nil, err
	}
	routes := []string{}
	for _, page := range catalog.Pages {
		routes = append(routes, page.Route)
	}
	key.Routes = identity(routes)
	if v.historical != nil {
		if key != v.historical.key {
			return nil, fmt.Errorf("historical Global projection authority changed during invocation")
		}
		if err := v.verifyObserved(); err != nil {
			return nil, err
		}
		var copy Global
		if err := StrictJSON(v.historical.bytes, &copy); err != nil {
			return nil, err
		}
		return &copy, nil
	}
	g, err := rebuildHistoricalGlobal(root, v)
	if err != nil {
		return nil, err
	}
	if err := v.verifyObserved(); err != nil {
		return nil, err
	}
	raw, err := Encode(g)
	if err != nil {
		return nil, err
	}
	v.historical = &historicalGlobalProof{key: key, bytes: raw}
	return g, nil
}
