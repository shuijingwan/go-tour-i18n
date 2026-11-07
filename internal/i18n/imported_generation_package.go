package i18n

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Resolve downstream scope solely from this batch's exact imported invocation.
// This is package preparation, not an approved-interval proof/currentness PASS:
// process/retry still rebuild the decoder and compare the frozen proof before
// restoring/validating outputs, and repeat those guards before publication.
func reconstructImportedApprovedIntervalPackage(root string, c *Catalog, m *RetranslationBatchManifest) (*Catalog, error) {
	base := "data/retranslation-runs/" + m.Locale + "/" + m.BatchID + "/"
	data, err := readCompatibilityFile(root, base+"generation-imports/initial-manifest.json")
	if os.IsNotExist(err) {
		return c, nil // Legacy/manual ordinary batches have no imported selector.
	}
	if err != nil {
		return nil, err
	}
	var receipt GenerationResultBundleManifest
	if err := contentidentity.StrictJSON(data, &receipt); err != nil {
		return nil, err
	}
	if receipt.Locale != m.Locale || receipt.BatchID != m.BatchID || receipt.Attempt != 1 || !validSHA256(receipt.GenerationBundleSHA256) || !validSHA256(receipt.InputIdentitySHA256) {
		return nil, fmt.Errorf("imported package receipt identity mismatch")
	}
	archive, err := readCompatibilityFile(root, importedGenerationArchive(m.Locale, receipt.GenerationBundleSHA256))
	if err != nil {
		return nil, err
	}
	if sum(archive) != receipt.GenerationBundleSHA256 {
		return nil, fmt.Errorf("imported package archive SHA256 mismatch")
	}
	gm, files, err := readGenerationBundle(archive)
	if err != nil {
		return nil, err
	}
	if err := checkImportedGenerationIdentity(receipt, gm, archive, files); err != nil {
		return nil, err
	}
	request, present := files[approvedIntervalSelectionMember]
	if !present {
		return c, nil // Ordinary imported bundles retain their caller's scope.
	}
	disk, err := readCompatibilityFile(root, base+"manifest.json")
	if err != nil || !bytes.Equal(disk, files["formal/manifest.json"]) {
		return nil, fmt.Errorf("imported package batch manifest mismatch")
	}
	var selection ApprovedIntervalSelection
	if err := contentidentity.StrictJSON(request, &selection); err != nil {
		return nil, err
	}
	if selection.Schema != ApprovedIntervalSelectionSchema || selection.Locale != m.Locale || selection.BatchID != m.BatchID {
		return nil, fmt.Errorf("imported package frozen selection identity mismatch")
	}
	scoped, err := SelectCurrentValidationReopenPackage(c, selection.ContentPackage)
	if err != nil {
		return nil, err
	}
	// Preserve the existing strict parser, including ContentPackage != pkg.
	if _, err := parseApprovedIntervalSelectionBytes(root, request, scoped, m); err != nil {
		return nil, err
	}
	return scoped, nil
}

// Bind an approved batch's next response to its actual retry import. Ordinary
// manual retries keep their existing contract. The returned identity is local
// to this validation invocation, not a new receipt or authorization.
func importedApprovedIntervalRetryIdentity(root string, c *Catalog, m *RetranslationBatchManifest, record RetranslationBatchUnit, attempt int, rawPath string, raw []byte) (string, error) {
	selection, initial, err := importedApprovedIntervalSelection(root, c, m)
	if err != nil || selection == nil {
		return "", err
	}
	base := "data/retranslation-runs/" + m.Locale + "/" + m.BatchID + "/"
	expected := GenerationBundleExpectedOutput{BundlePath: "outputs/" + filepath.Base(record.InputPath), InstallPath: rawPath, UnitID: record.UnitID}
	request := GenerationBundleManifest{TaskKind: "translation-unit-retry", Attempt: attempt, ExpectedOutputs: []GenerationBundleExpectedOutput{expected}}
	receiptPath := filepath.ToSlash(generationImportReceiptPath(filepath.Join(root, base), request))
	receiptPath, err = repositoryRelativePath(root, receiptPath)
	if err != nil {
		return "", err
	}
	b, err := readCompatibilityFile(root, receiptPath)
	if err != nil {
		return "", err
	}
	var receipt GenerationResultBundleManifest
	if err := contentidentity.StrictJSON(b, &receipt); err != nil {
		return "", err
	}
	if !validSHA256(receipt.GenerationBundleSHA256) || !validSHA256(receipt.InputIdentitySHA256) {
		return "", fmt.Errorf("invalid approved retry import identity")
	}
	archive, err := readCompatibilityFile(root, importedGenerationArchive(m.Locale, receipt.GenerationBundleSHA256))
	if err != nil {
		return "", err
	}
	if sum(archive) != receipt.GenerationBundleSHA256 {
		return "", fmt.Errorf("approved retry archive SHA256 mismatch")
	}
	gm, files, err := readGenerationBundle(archive)
	if err != nil {
		return "", err
	}
	if err := checkImportedGenerationIdentity(receipt, gm, archive, files); err != nil {
		return "", err
	}
	if gm.Locale != m.Locale || gm.BatchID != m.BatchID || gm.Attempt != attempt || gm.TaskKind != request.TaskKind || gm.UnitKind != record.UnitKind || len(gm.ExpectedOutputs) != 1 || gm.ExpectedOutputs[0] != expected {
		return "", fmt.Errorf("approved retry invocation/attempt/output mismatch")
	}
	if !bytes.Equal(files["formal/manifest.json"], initial["formal/manifest.json"]) || !bytes.Equal(files[approvedIntervalSelectionMember], selection.bytes) {
		return "", fmt.Errorf("approved retry changed frozen batch/selection")
	}
	proofs := []generationBundleInput{}
	for path, data := range initial {
		if strings.HasPrefix(path, "formal/approved-interval-transport/") {
			proofs = append(proofs, generationBundleInput{path: path, data: data})
		}
	}
	if err := compareInstalledApprovedProofs(proofs, files); err != nil {
		return "", err
	}
	if len(receipt.Outputs) != 1 || receipt.Outputs[0] != NewTransportBundleFile(expected.BundlePath, "", raw) {
		return "", fmt.Errorf("approved retry raw/import provenance mismatch")
	}
	disk, err := readCompatibilityFile(root, base+rawPath)
	if err != nil || !bytes.Equal(disk, raw) {
		return "", fmt.Errorf("approved retry raw changed during validation")
	}
	return sum([]byte(sum(b) + "\x00" + receipt.GenerationBundleSHA256 + "\x00" + sum(raw))), nil
}
