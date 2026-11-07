package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Exact historical execution objects are evidence, never new model calls. The
// provider's real session/response references must not acquire a second origin
// by changing a Bundle hash or copying outputs into another staging directory.
func rejectRecoveryExecutionOriginReplay(root string, e *TransportRecoveryGenerationExecution, a *InstalledDerivedTransportRecovery) error {
	paths := map[string]bool{a.Predecessor.ImportReceipt.Path: true}
	for _, u := range a.Predecessor.FinalUnits {
		for _, r := range u.History {
			if r.Role == "invocation-receipt" {
				paths[r.Ref.Path] = true
			}
		}
	}
	base := "data/retranslation-runs/" + e.Locale + "/" + e.BatchID + "/generation-imports/"
	entries, err := os.ReadDir(filepath.Join(root, base))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), "-manifest.json") {
			paths[base+entry.Name()] = true
		}
	}
	origins := map[string]bool{}
	for path := range paths {
		b, err := readCompatibilityFile(root, path)
		if err != nil {
			return err
		}
		var receipt GenerationResultBundleManifest
		if err = contentidentity.StrictJSON(b, &receipt); err != nil {
			return err
		}
		if receipt.RecoveryExecution == nil {
			continue
		}
		ref := receipt.RecoveryExecution.Object
		b, err = readCompatibilityFile(root, ref.Path)
		if err != nil || sum(b) != ref.SHA256 || int64(len(b)) != ref.Size {
			return fmt.Errorf("prior execution object integrity mismatch")
		}
		var old TransportRecoveryGenerationExecution
		if err = contentidentity.StrictJSON(b, &old); err != nil {
			return err
		}
		if old.Identity != recoveryExecutionIdentity(old) {
			return fmt.Errorf("prior execution identity mismatch")
		}
		if receipt.Locale == e.Locale && receipt.BatchID == e.BatchID && receipt.Attempt == e.Attempt && receipt.GenerationBundleSHA256 == e.BundleSHA256 && receipt.RecoveryExecution.Identity == e.Identity {
			continue
		}
		for _, call := range old.Invocations {
			origins[call.Session+"\x00"+call.Invocation] = true
		}
	}
	for _, call := range e.Invocations {
		if origins[call.Session+"\x00"+call.Invocation] {
			return fmt.Errorf("old model call cannot become a new invocation output origin")
		}
	}
	return nil
}
