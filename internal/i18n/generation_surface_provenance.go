package i18n

import "fmt"

// This post-import guard has no pre-import/current-Bundle fallback. Initial
// raw responses remain immutable even when a later retry has its own output.
func checkImportedSurfaceOutputHashes(root string, m *RetranslationBatchManifest) error {
	needed := false
	for _, r := range m.Units {
		needed = needed || (r.RevisionFeedbackSource == "surface_review" && hasSurfaceTransport(root, m.Locale, r.RevisionAuthorizationID))
	}
	if !needed {
		return nil
	}
	base := "data/retranslation-runs/" + m.Locale + "/" + m.BatchID
	b, err := readSnapshotReferencedFile(root, base+"/generation-imports/initial-manifest.json")
	if err != nil {
		return fmt.Errorf("Surface process requires sealed import: %w", err)
	}
	var imported GenerationResultBundleManifest
	if err := decodeStrictBundleJSON(b, &imported); err != nil {
		return err
	}
	archive, err := readSnapshotReferencedFile(root, importedGenerationArchive(m.Locale, imported.GenerationBundleSHA256))
	if err != nil {
		return err
	}
	gm, files, err := readGenerationBundle(archive)
	if err != nil {
		return err
	}
	if err := checkImportedGenerationIdentity(imported, gm, archive, files); err != nil {
		return err
	}
	if gm.Locale != m.Locale || gm.BatchID != m.BatchID || gm.Attempt != 1 || gm.TaskKind != "translation-unit-revision" {
		return fmt.Errorf("Surface initial import batch/attempt mismatch")
	}
	outputs := map[string]TransportBundleFile{}
	for _, o := range imported.Outputs {
		if _, ok := outputs[o.BundlePath]; ok {
			return fmt.Errorf("duplicate Surface imported output")
		}
		outputs[o.BundlePath] = o
	}
	for _, expected := range gm.ExpectedOutputs {
		out, ok := outputs[expected.BundlePath]
		if err := validateGenerationInstallPath(expected.InstallPath); err != nil {
			return err
		}
		raw, err := readSnapshotReferencedFile(root, base+"/"+expected.InstallPath)
		if !ok || err != nil || sum(raw) != out.SHA256 || int64(len(raw)) != out.Size {
			return fmt.Errorf("Surface imported raw output provenance mismatch: %s", expected.UnitID)
		}
	}
	return nil
}
