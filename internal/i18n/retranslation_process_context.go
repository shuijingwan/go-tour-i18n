package i18n

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

// One process/retry/revalidate invocation owns this graph. It is not a PASS
// cache: publication requires a second complete authority reconstruction and
// exact dependency check after all restore/current-validation work.
func prepareProcessOutputContext(batchDir string, c *Catalog, g *Glossary, m *RetranslationBatchManifest) (map[string]protectedTranslation, func() error, error) {
	root := filepath.Clean(filepath.Join(batchDir, "../../../.."))
	if err := checkImportedSurfaceOutputHashes(root, m); err != nil {
		return nil, nil, err
	}
	if err := checkSurfaceTransportBatch(root, m.Locale, m.BatchID, m.Units, false); err != nil {
		return nil, nil, err
	}
	needed := false
	for _, r := range m.Units {
		needed = needed || r.RevisionFeedbackSource == CurrentValidationRevisionFeedback || r.RepresentationBridge != nil || (r.RevisionFeedbackSource == "surface_review" && hasSurfaceTransport(root, m.Locale, r.RevisionAuthorizationID))
		if !needed {
			_, lineage, err := inheritedBridgeOutput(root, c, g, m.Locale, r)
			if err != nil {
				return nil, nil, err
			}
			needed = lineage != nil
		}
	}
	if !needed {
		return nil, nil, nil
	}

	projections := map[string]protectedTranslation{}
	inputs, err := approvedResponseInputs(root, c, m, projections)
	if err != nil {
		return nil, nil, err
	}
	refs := map[string][]byte{}
	paths := append([]string(nil), translationUnitGenerationAuthorityPaths...)
	paths = append(paths, currentValidationReopenDependencies...)
	currentGlossary, err := LoadGlossary(root, m.Locale)
	if err != nil || !reflect.DeepEqual(currentGlossary, g) {
		return nil, nil, fmt.Errorf("process glossary changed before preparation")
	}
	review, err := RequireUnifiedGlossaryReview(root, m.Locale)
	if err != nil {
		return nil, nil, err
	}
	paths = append(paths, review.Path, GlossaryCorpusPath)
	rel, err := repositoryRelativePath(root, batchDir)
	if err != nil {
		return nil, nil, err
	}
	paths = append(paths, rel+"/manifest.json", glossaryReviewGlossaryPath(m.Locale))
	provenance := rel + "/generation-imports/initial-manifest.json"
	provenancePresent := false
	if _, err := os.Lstat(filepath.Join(root, provenance)); err == nil {
		provenancePresent = true
		paths = append(paths, provenance)
	} else if !os.IsNotExist(err) {
		return nil, nil, err
	}
	identities := map[string]string{}
	for _, r := range m.Units {
		u, err := c.Unit(r.UnitID)
		if err != nil {
			return nil, nil, err
		}
		disk, err := readSnapshotReferencedFile(root, u.SourcePath)
		if err != nil || validateSnapshotSourceIdentity(c, u, disk) != nil {
			return nil, nil, fmt.Errorf("process source changed: %s", r.UnitID)
		}
		paths = append(paths, u.SourcePath, rel+"/raw-responses/"+filepath.Base(r.InputPath), rel+"/"+r.InputPath)
		identity, err := currentTranslationAuthority(root, u, m.Locale)
		if err != nil {
			return nil, nil, err
		}
		identities[u.ID] = identity
	}
	for _, path := range paths {
		b, err := readSnapshotReferencedFile(root, path)
		if err != nil {
			return nil, nil, err
		}
		refs[path] = bytes.Clone(b)
	}
	manifest, err := readRetranslationProcessManifest(batchDir, m.Locale, m.BatchID)
	if err != nil || !reflect.DeepEqual(manifest, m) {
		return nil, nil, fmt.Errorf("process manifest changed while preparing context")
	}
	check := func() error {
		if err := checkImportedSurfaceOutputHashes(root, m); err != nil {
			return err
		}
		if err := ensureNoGenerationImportPending(batchDir); err != nil {
			return err
		}
		_, provenanceErr := os.Lstat(filepath.Join(root, provenance))
		if (provenanceErr == nil) != provenancePresent || (provenanceErr != nil && !os.IsNotExist(provenanceErr)) {
			return fmt.Errorf("process import provenance presence changed")
		}
		for path, want := range refs {
			got, err := readSnapshotReferencedFile(root, path)
			if err != nil || !bytes.Equal(got, want) {
				return fmt.Errorf("process dependency changed during invocation: %s", path)
			}
		}
		for id, want := range identities {
			u, err := c.Unit(id)
			if err != nil {
				return err
			}
			got, err := currentTranslationAuthority(root, u, m.Locale)
			if err != nil || got != want {
				return fmt.Errorf("process validator/source authority changed: %s", id)
			}
		}
		if err := RequireNewLanguageGlossaryReview(root, m.Locale); err != nil {
			return err
		}
		fresh, err := approvedResponseInputs(root, c, m)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(inputs, fresh) {
			return fmt.Errorf("process authority graph changed during invocation")
		}
		return nil
	}
	return projections, check, nil
}

func checkPreparedRetranslationContext(items []preparedRetranslationPage) error {
	if len(items) > 0 && items[0].checkOutputContext != nil {
		return items[0].checkOutputContext()
	}
	return nil
}
