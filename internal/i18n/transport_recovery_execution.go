package i18n

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

type TransportRecoveryExecutionOutput struct {
	Output   TransportBundleFile   `json:"output"`
	Evidence RecoveryFileReference `json:"response_artifact"`
}

type TransportRecoveryModelInvocation struct {
	Session             string                             `json:"session_reference"`
	Invocation          string                             `json:"invocation_reference"`
	BundleSHA256        string                             `json:"generation_bundle_sha256"`
	InputIdentitySHA256 string                             `json:"input_identity_sha256"`
	TaskKind            string                             `json:"task_kind"`
	Attempt             int                                `json:"attempt"`
	Provider            string                             `json:"provider"`
	Model               string                             `json:"model"`
	InputEvidence       RecoveryFileReference              `json:"input_bundle_artifact"`
	Outputs             []TransportRecoveryExecutionOutput `json:"outputs"`
	Evidence            []RecoveryFileReference            `json:"execution_evidence"`
}

type TransportRecoveryGenerationExecution struct {
	Schema              string                             `json:"schema"`
	Kind                string                             `json:"kind"`
	Locale              string                             `json:"locale"`
	ContentPackage      string                             `json:"content_package"`
	BatchID             string                             `json:"batch_id"`
	TaskKind            string                             `json:"task_kind"`
	Attempt             int                                `json:"attempt"`
	Authority           TransportRecoveryReference         `json:"transport_recovery"`
	BundleSHA256        string                             `json:"generation_bundle_sha256"`
	InputIdentitySHA256 string                             `json:"input_identity_sha256"`
	Provider            string                             `json:"provider"`
	Model               string                             `json:"model"`
	Invocations         []TransportRecoveryModelInvocation `json:"invocations"`
	Identity            string                             `json:"identity"`
}

type RecoveryExecutionArchivedEvidence struct {
	Original  RecoveryFileReference `json:"original"`
	Installed RecoveryFileReference `json:"installed"`
}

type RecoveryExecutionReceipt struct {
	Identity string                              `json:"identity"`
	Object   RecoveryFileReference               `json:"object"`
	Evidence []RecoveryExecutionArchivedEvidence `json:"evidence"`
}

type recoveryExecutionInstall struct {
	Check   func() error
	Receipt RecoveryExecutionReceipt
	Members map[string][]byte
}

func recoveryExecutionIdentity(e TransportRecoveryGenerationExecution) string {
	e.Identity = ""
	return sum(mustJSON(e))
}

func recoveryEvidenceIsHistorical(path string) bool {
	// This rejects a forbidden provenance origin, not byte equality. A real new
	// model call is allowed to produce the same language bytes.
	return strings.HasPrefix(path, "data/retranslation-runs/") || strings.HasPrefix(path, "data/translation-transport-recoveries/")
}

func validateRecoveryExecution(root string, e *TransportRecoveryGenerationExecution, a *InstalledDerivedTransportRecovery, ref TransportRecoveryReference, m GenerationBundleManifest, bundle []byte, provider, model string, outputs map[string][]byte, read func(RecoveryFileReference) ([]byte, error)) (map[string][]byte, error) {
	if e.Schema != TransportRecoveryExecutionSchema || e.Kind != "model-generation" || e.Identity != recoveryExecutionIdentity(*e) || e.Locale != m.Locale || e.BatchID != m.BatchID || e.ContentPackage != a.ContentPackage.ID || e.TaskKind != m.TaskKind || e.Attempt != m.Attempt || e.Authority != ref || e.BundleSHA256 != sum(bundle) || e.InputIdentitySHA256 != m.InputIdentitySHA256 || e.Provider != provider || e.Model != model || provider != a.AllowedAction.Provider || model != a.AllowedAction.Model || len(e.Invocations) == 0 {
		return nil, fmt.Errorf("recovery Generation execution invocation/provenance mismatch")
	}
	material := map[string][]byte{}
	readEvidence := func(r RecoveryFileReference) ([]byte, error) {
		if !validRecoveryFileReference(r) || recoveryEvidenceIsHistorical(r.Path) {
			return nil, fmt.Errorf("new model execution requires nonhistorical exact evidence")
		}
		b, err := read(r)
		if err != nil || sum(b) != r.SHA256 || int64(len(b)) != r.Size {
			return nil, fmt.Errorf("recovery execution evidence mismatch: %s", r.Path)
		}
		if prev, ok := material[r.Path]; ok && !bytes.Equal(prev, b) {
			return nil, fmt.Errorf("conflicting execution evidence reference")
		}
		material[r.Path] = b
		return b, nil
	}
	expected := map[string]bool{}
	positions := map[string]int{}
	for i, o := range m.ExpectedOutputs {
		expected[o.BundlePath] = true
		positions[o.BundlePath] = i
	}
	origins := map[string]bool{}
	calls := map[string]bool{}
	for _, call := range e.Invocations {
		key := call.Session + "\x00" + call.Invocation
		if call.Session == "" || strings.TrimSpace(call.Session) != call.Session || strings.ContainsAny(call.Session, "\x00\r\n") || call.Invocation == "" || strings.TrimSpace(call.Invocation) != call.Invocation || strings.ContainsAny(call.Invocation, "\x00\r\n") || calls[key] || call.BundleSHA256 != e.BundleSHA256 || call.InputIdentitySHA256 != e.InputIdentitySHA256 || call.TaskKind != e.TaskKind || call.Attempt != e.Attempt || call.Provider != provider || call.Model != model || len(call.Outputs) == 0 || len(call.Evidence) == 0 {
			return nil, fmt.Errorf("invalid/duplicate new model invocation origin")
		}
		calls[key] = true
		input, err := readEvidence(call.InputEvidence)
		if err != nil || !bytes.Equal(input, bundle) {
			return nil, fmt.Errorf("model call did not bind exact new input Bundle")
		}
		for _, r := range call.Evidence {
			if _, err := readEvidence(r); err != nil {
				return nil, err
			}
		}
		last := -1
		for _, out := range call.Outputs {
			b, ok := outputs[out.Output.BundlePath]
			if !ok || !expected[out.Output.BundlePath] || origins[out.Output.BundlePath] || out.Output != NewTransportBundleFile(out.Output.BundlePath, "", b) {
				return nil, fmt.Errorf("recovery execution output coverage/origin mismatch")
			}
			if positions[out.Output.BundlePath] <= last {
				return nil, fmt.Errorf("execution outputs must preserve ExpectedOutputs order")
			}
			last = positions[out.Output.BundlePath]
			evidence, err := readEvidence(out.Evidence)
			if err != nil || !bytes.Equal(evidence, b) {
				return nil, fmt.Errorf("new model response artifact does not bind output bytes")
			}
			origins[out.Output.BundlePath] = true
		}
	}
	if len(origins) != len(expected) {
		return nil, fmt.Errorf("incomplete recovery execution output origins")
	}
	return material, nil
}

func ImportTransportRecoveryOutputDirectory(root string, c *Catalog, bundle []byte, provider, model, inputDir, executionPath, executionSHA string) (*GenerationImportResult, error) {
	var err error
	c, err = recoveryGenerationImportCatalog(root, c, bundle)
	if err != nil {
		return nil, err
	}
	m, files, err := currentGenerationBundle(root, c, bundle)
	if err != nil {
		return nil, err
	}
	var batch RetranslationBatchManifest
	if err = contentidentity.StrictJSON(files["formal/manifest.json"], &batch); err != nil {
		return nil, err
	}
	if batch.TransportRecovery == nil {
		return nil, fmt.Errorf("generation execution flags only support recovery invocation")
	}
	a, _, err := readRecoveryExactAuthority(root, *batch.TransportRecovery)
	if err != nil {
		return nil, err
	}
	if m.TaskKind != TransportRecoveryTaskKind && m.TaskKind != "translation-unit-retry" {
		return nil, fmt.Errorf("recovery task/attempt mismatch")
	}
	if !validSHA256(executionSHA) {
		return nil, fmt.Errorf("exact generation execution SHA required")
	}
	b, err := readCompatibilityFile(root, executionPath)
	if err != nil || sum(b) != executionSHA {
		return nil, fmt.Errorf("generation execution object SHA mismatch")
	}
	var e TransportRecoveryGenerationExecution
	if err = contentidentity.StrictJSON(b, &e); err != nil {
		return nil, err
	}
	if err = rejectRecoveryExecutionOriginReplay(root, &e, a); err != nil {
		return nil, err
	}
	rawFiles, err := readExactGenerationOutputDirectory(inputDir, m.ExpectedOutputs)
	if err != nil {
		return nil, err
	}
	outputs := map[string][]byte{}
	for _, o := range m.ExpectedOutputs {
		outputs[o.BundlePath] = rawFiles[filepath.Base(o.BundlePath)]
	}
	if err = rejectHistoricalRecoveryOutputDirectory(root, inputDir, a, m); err != nil {
		return nil, err
	}
	material, err := validateRecoveryExecution(root, &e, a, *batch.TransportRecovery, m, bundle, provider, model, outputs, func(r RecoveryFileReference) ([]byte, error) { return readCompatibilityFile(root, r.Path) })
	if err != nil {
		return nil, err
	}
	if err = validateGenerationOutputSet(root, c, m, outputs); err != nil {
		return nil, err
	}
	base := "data/retranslation-runs/" + m.Locale + "/" + m.BatchID + "/generation-imports/"
	stem := "initial"
	if m.Attempt > 1 {
		stem = retryUnitID(m) + fmt.Sprintf("-attempt-%03d", m.Attempt)
	}
	installedBase := base + stem + "-execution/"
	x := &recoveryExecutionInstall{Receipt: RecoveryExecutionReceipt{Identity: e.Identity, Object: recoveryMemberReference(installedBase+"execution.json", b)}, Members: map[string][]byte{installedBase + "execution.json": b}}
	x.Check = func() error { return checkRecoveryInstallCAS(root, c, &batch) }
	var names []string
	for name := range material {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		data := material[name]
		oldRef := recoveryMemberReference(name, data)
		dest := installedBase + "evidence/" + sum([]byte(name)) + ".bin"
		x.Receipt.Evidence = append(x.Receipt.Evidence, RecoveryExecutionArchivedEvidence{oldRef, recoveryMemberReference(dest, data)})
		x.Members[dest] = data
	}
	return installGenerationOutputs(root, m, bundle, files["manifest.json"], provider, model, outputs, x)
}

func rejectHistoricalRecoveryOutputDirectory(root, dir string, a *InstalledDerivedTransportRecovery, m GenerationBundleManifest) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	formal := filepath.Join(root, "data/retranslation-runs")
	if pathWithinRoot(formal, abs) {
		return fmt.Errorf("recovery outputs cannot originate from installed raw/retry directories")
	}
	priorPaths := map[string]bool{}
	receiptPaths := map[string]bool{}
	for _, u := range a.Predecessor.FinalUnits {
		priorPaths[u.Raw.Path] = true
		for _, h := range u.History {
			if h.Role == "invocation-receipt" {
				receiptPaths[h.Ref.Path] = true
			}
		}
	}
	var receipts []string
	for path := range receiptPaths {
		receipts = append(receipts, path)
	}
	sort.Strings(receipts)
	for _, path := range receipts {
		b, err := readCompatibilityFile(root, path)
		if err != nil {
			return err
		}
		var receipt GenerationResultBundleManifest
		if err = contentidentity.StrictJSON(b, &receipt); err != nil {
			return err
		}
		archive, err := readCompatibilityFile(root, importedGenerationArchive(receipt.Locale, receipt.GenerationBundleSHA256))
		if err != nil || sum(archive) != receipt.GenerationBundleSHA256 {
			return fmt.Errorf("historical execution archive changed")
		}
		gm, _, err := readGenerationBundle(archive)
		if err != nil {
			return err
		}
		for _, out := range gm.ExpectedOutputs {
			priorPaths["data/retranslation-runs/"+receipt.Locale+"/"+receipt.BatchID+"/"+out.InstallPath] = true
		}
	}
	for _, out := range m.ExpectedOutputs {
		info, err := os.Lstat(filepath.Join(dir, filepath.Base(out.BundlePath)))
		if err != nil {
			return err
		}
		for path := range priorPaths {
			prior, err := os.Lstat(filepath.Join(root, path))
			if err != nil {
				return err
			}
			if os.SameFile(info, prior) {
				return fmt.Errorf("old raw inode/hardlink cannot become a new model response")
			}
		}
	}
	return nil
}

func verifyRecoveryInstalledExecution(root string, c *Catalog, batch *RetranslationBatchManifest) error {
	if batch.TransportRecovery == nil {
		return nil
	}
	a, _, err := readRecoveryExactAuthority(root, *batch.TransportRecovery)
	if err != nil {
		return err
	}
	base := "data/retranslation-runs/" + batch.Locale + "/" + batch.BatchID + "/"
	if err = ensureNoGenerationImportPending(filepath.Join(root, base)); err != nil {
		return err
	}
	initialBytes, err := readCompatibilityFile(root, base+"generation-imports/initial-manifest.json")
	if err != nil {
		return err
	}
	var initialReceipt GenerationResultBundleManifest
	if err = contentidentity.StrictJSON(initialBytes, &initialReceipt); err != nil {
		return err
	}
	initialArchive, err := readCompatibilityFile(root, importedGenerationArchive(batch.Locale, initialReceipt.GenerationBundleSHA256))
	if err != nil {
		return err
	}
	initialManifest, initialFiles, err := readGenerationBundle(initialArchive)
	if err != nil {
		return err
	}
	if err = checkImportedGenerationIdentity(initialReceipt, initialManifest, initialArchive, initialFiles); err != nil {
		return err
	}
	var frozenProofs []generationBundleInput
	for path, b := range initialFiles {
		if isApprovedProofMember(path) {
			frozenProofs = append(frozenProofs, generationBundleInput{path: path, data: b})
		}
	}
	// Check every installed receipt, including next-attempt receipts awaiting
	// retry processing. No external staging file remains an installed dependency.
	entries, err := os.ReadDir(filepath.Join(root, base, "generation-imports"))
	if err != nil {
		return err
	}
	count := 0
	initialSeen := false
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "-manifest.json") {
			continue
		}
		b, err := readCompatibilityFile(root, base+"generation-imports/"+entry.Name())
		if err != nil {
			return err
		}
		var receipt GenerationResultBundleManifest
		if err = contentidentity.StrictJSON(b, &receipt); err != nil {
			return err
		}
		if receipt.RecoveryExecution == nil {
			return fmt.Errorf("installed recovery execution provenance missing")
		}
		bundle, err := readCompatibilityFile(root, importedGenerationArchive(batch.Locale, receipt.GenerationBundleSHA256))
		if err != nil {
			return err
		}
		gm, files, err := readGenerationBundle(bundle)
		if err != nil {
			return err
		}
		if err = checkImportedGenerationIdentity(receipt, gm, bundle, files); err != nil {
			return err
		}
		if entry.Name() != filepath.Base(generationImportReceiptPath(filepath.Join(root, base), gm)) || (gm.Attempt == 1 && gm.TaskKind != TransportRecoveryTaskKind) || (gm.Attempt > 1 && gm.TaskKind != "translation-unit-retry") || gm.Attempt < 1 {
			return fmt.Errorf("installed recovery receipt task/path mismatch")
		}
		if gm.Attempt == 1 {
			initialSeen = true
		}
		if err = verifyRecoveryHistoricalBundleScope(batch, gm, retryUnitID(gm), gm.Attempt); err != nil {
			return err
		}
		if !bytes.Equal(files[approvedIntervalSelectionMember], initialFiles[approvedIntervalSelectionMember]) || !bytes.Equal(files[transportRecoveryConstructionMember], initialFiles[transportRecoveryConstructionMember]) {
			return fmt.Errorf("retry changed frozen recovery selection/construction")
		}
		if err = compareInstalledApprovedProofs(frozenProofs, files); err != nil {
			return err
		}
		if gm.Locale != batch.Locale || gm.BatchID != batch.BatchID || !bytes.Equal(files["formal/manifest.json"], mustReadRecoveryManifest(root, base)) {
			return fmt.Errorf("installed recovery invocation scope mismatch")
		}
		x := receipt.RecoveryExecution
		stem := "initial"
		if gm.Attempt > 1 {
			stem = retryUnitID(gm) + fmt.Sprintf("-attempt-%03d", gm.Attempt)
		}
		executionBase := base + "generation-imports/" + stem + "-execution/"
		if x.Object.Path != executionBase+"execution.json" {
			return fmt.Errorf("installed execution must use canonical archived object")
		}
		eb, err := readCompatibilityFile(root, x.Object.Path)
		if err != nil || x.Object.SHA256 != sum(eb) || x.Object.Size != int64(len(eb)) {
			return fmt.Errorf("installed execution object drift")
		}
		var e TransportRecoveryGenerationExecution
		if err = contentidentity.StrictJSON(eb, &e); err != nil {
			return err
		}
		if e.Identity != x.Identity {
			return fmt.Errorf("installed execution identity mismatch")
		}
		if err = rejectRecoveryExecutionOriginReplay(root, &e, a); err != nil {
			return err
		}
		archived := map[string]RecoveryExecutionArchivedEvidence{}
		for _, v := range x.Evidence {
			if v.Installed.Path != executionBase+"evidence/"+sum([]byte(v.Original.Path))+".bin" || v.Installed.SHA256 != v.Original.SHA256 || v.Installed.Size != v.Original.Size {
				return fmt.Errorf("execution archived evidence location/hash mismatch")
			}
			if _, ok := archived[v.Original.Path]; ok {
				return fmt.Errorf("duplicate archived execution evidence")
			}
			archived[v.Original.Path] = v
		}
		outputs := map[string][]byte{}
		for _, out := range gm.ExpectedOutputs {
			raw, err := readCompatibilityFile(root, base+out.InstallPath)
			if err != nil {
				return err
			}
			outputs[out.BundlePath] = raw
		}
		material, validationErr := validateRecoveryExecution(root, &e, a, *batch.TransportRecovery, gm, bundle, receipt.Provider, receipt.Model, outputs, func(r RecoveryFileReference) ([]byte, error) {
			v, ok := archived[r.Path]
			if !ok || v.Original != r {
				return nil, fmt.Errorf("execution archived evidence binding missing")
			}
			b, err := readCompatibilityFile(root, v.Installed.Path)
			if err != nil || sum(b) != v.Installed.SHA256 || int64(len(b)) != v.Installed.Size {
				return nil, fmt.Errorf("execution evidence archive drift")
			}
			return b, nil
		})
		if validationErr != nil {
			return validationErr
		}
		if len(material) != len(archived) {
			return fmt.Errorf("extra archived recovery execution evidence")
		}
		expected := map[string]bool{x.Object.Path: true}
		for _, r := range x.Evidence {
			expected[r.Installed.Path] = true
		}
		if err = verifyRecoveryManagedDirectory(root, strings.TrimSuffix(executionBase, "/"), expected, ""); err != nil {
			return err
		}
		count++
	}
	if count == 0 || !initialSeen {
		return fmt.Errorf("initial recovery import receipt missing")
	}
	return nil
}

func mustReadRecoveryManifest(root, base string) []byte {
	b, _ := readCompatibilityFile(root, base+"manifest.json")
	return b
}
