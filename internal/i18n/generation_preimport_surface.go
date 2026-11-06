package i18n

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
)

// Only the result-pack caller supplies files from currentGenerationBundle.
// This is a pre-import transport decoder, never a historical provenance
// fallback. Process/retry/history still use surfaceTransportOutput's seal.
func preimportSurfaceGenerationOutput(c *Catalog, m GenerationBundleManifest, files map[string][]byte, unitID string) (protectedTranslation, bool, error) {
	var batch RetranslationBatchManifest
	if err := decodeStrictBundleJSON(files["formal/manifest.json"], &batch); err != nil {
		return protectedTranslation{}, false, err
	}
	for _, record := range batch.Units {
		if record.UnitID != unitID || record.RevisionFeedbackSource != "surface_review" {
			continue
		}
		fail := func() (protectedTranslation, bool, error) {
			return protectedTranslation{}, true, fmt.Errorf("pre-import Surface Bundle binding mismatch: %s", unitID)
		}
		b := files["formal/surface-transport-authority/"+unitID+".json"]
		var a SurfaceTransportAuthority
		if err := decodeStrictBundleJSON(b, &a); err != nil || a.Schema != SurfaceTransportSchema || a.Identity != surfaceTransportIdentity(a) || !bytes.Equal(b, surfaceTransportBytes(&a)) || a.Request.Locale != m.Locale || a.Request.BatchID != m.BatchID || a.Request.ManifestSHA256 != sum(files["formal/manifest.json"]) {
			return fail()
		}
		if a.Reopen.Path != "data/quality-check-reopens/"+m.Locale+"/"+record.RevisionAuthorizationID+".json" {
			return fail()
		}
		u, err := c.Unit(unitID)
		if err != nil {
			return protectedTranslation{}, true, err
		}
		factory, ok := u.Adapter.(SurfaceTransportAdapter)
		if !ok {
			return fail()
		}
		for _, binding := range a.Bindings {
			if binding.UnitID != unitID {
				continue
			}
			name := filepath.Base(record.InputPath)
			candidate := files["formal/candidates/"+name]
			if binding.FindingSHA256 != sum([]byte(record.PreviousFinding)) || binding.Predecessor.UnitID != unitID || binding.Predecessor.SourceSHA256 != record.SourceSHA256 || !bytes.Equal(files["formal/surface-transport/"+unitID+".json"], mustJSON(binding.Projection)) || !bytes.Equal(files["formal/protected-candidates/"+name], []byte(binding.Projection.Template)) || sum(candidate) != binding.Predecessor.CandidateSHA256 || sum(files["formal/historical-protected-candidates/"+name]) != binding.Raw.SHA256 {
				return fail()
			}
			adapter, err := factory.ReadSurfaceTransportProjection(binding.Projection.Codec)
			if err != nil {
				return protectedTranslation{}, true, err
			}
			got, err := adapter.Restore(binding.Projection.Template)
			if err != nil || !bytes.Equal(canonicalizeRetranslationArtifactEOF(got), candidate) {
				return fail()
			}
			return protectedTranslation{Text: binding.Projection.Template, Adapter: adapter}, true, nil
		}
		return fail()
	}
	return protectedTranslation{}, false, nil
}

// Fresh final guards for Bundle-local decoding. Do not carry the entry's
// currentness decision across output validation or Result ZIP serialization.
func checkPreimportSurfaceGenerationContext(root string, c *Catalog, m GenerationBundleManifest, files map[string][]byte) error {
	seen := map[string]bool{}
	for _, expected := range m.ExpectedOutputs {
		b := files["formal/surface-transport-authority/"+expected.UnitID+".json"]
		if len(b) == 0 {
			continue
		}
		var a SurfaceTransportAuthority
		if err := decodeStrictBundleJSON(b, &a); err != nil {
			return err
		}
		if seen[a.Identity] {
			continue
		}
		seen[a.Identity] = true
		u, err := c.Unit(expected.UnitID)
		if err != nil {
			return err
		}
		factory, ok := u.Adapter.(SurfaceTransportAdapter)
		if !ok {
			return fmt.Errorf("Surface current context adapter missing")
		}
		if err := factory.WithCurrentSurfaceTransportContext(root, c, func(context string) error {
			if context != a.CurrentContextIdentity {
				return fmt.Errorf("Surface pre-import current package changed")
			}
			for _, ref := range append(append([]GlossaryArchiveReference{}, a.Dependencies...), GlossaryArchiveReference{surfaceTransportPath(m.Locale, strings.TrimSuffix(filepath.Base(a.Reopen.Path), ".json")), sum(b)}) {
				got, e := readSnapshotReferencedFile(root, ref.Path)
				if e != nil || sum(got) != ref.SHA256 {
					return fmt.Errorf("Surface pre-import dependency changed: %s", ref.Path)
				}
			}
			review, err := RequireUnifiedGlossaryReview(root, m.Locale)
			if err != nil || review != a.CurrentReview {
				return fmt.Errorf("Surface pre-import current Review changed")
			}
			for _, binding := range a.Bindings {
				u, err := c.Unit(binding.UnitID)
				if err != nil {
					return err
				}
				authority, err := currentTranslationAuthority(root, u, m.Locale)
				if err != nil || authority != binding.CurrentAuthority {
					return fmt.Errorf("Surface pre-import validator changed")
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}
