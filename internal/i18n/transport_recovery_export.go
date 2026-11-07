package i18n

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

type recoveryExportTransaction struct {
	Schema    string                     `json:"schema"`
	Authority TransportRecoveryReference `json:"authority"`
	Manifest  RetranslationBatchManifest `json:"manifest"`
	Members   []recoverySealedMember     `json:"members"`
	Seal      string                     `json:"seal"`
}

func recoveryExportSeal(t recoveryExportTransaction) string { t.Seal = ""; return sum(mustJSON(t)) }

func buildRecoveryExportTransaction(root string, a *InstalledDerivedTransportRecovery, m *RetranslationBatchManifest, ref TransportRecoveryReference) (recoveryExportTransaction, error) {
	tx := recoveryExportTransaction{Schema: "go-learning/transport-recovery-export-transaction/v1", Authority: ref, Manifest: *m}
	for _, r := range m.Units {
		b, err := readCompatibilityFile(root, "data/retranslation-runs/"+m.Locale+"/"+a.Predecessor.BatchID+"/"+r.InputPath)
		if err != nil || sum(b) != r.InputSHA256 {
			return tx, fmt.Errorf("recovery predecessor input changed")
		}
		tx.Members = append(tx.Members, recoverySealedMember{recoveryMemberReference(r.InputPath, b), b})
	}
	b := mustJSON(m)
	tx.Members = append(tx.Members, recoverySealedMember{recoveryMemberReference("manifest.json", b), b})
	tx.Seal = recoveryExportSeal(tx)
	return tx, nil
}

func publishTransportRecoveryBatch(root string, c *Catalog, a *InstalledDerivedTransportRecovery, m *RetranslationBatchManifest, ref TransportRecoveryReference) (*RetranslationExportResult, error) {
	tx, err := buildRecoveryExportTransaction(root, a, m, ref)
	if err != nil {
		return nil, err
	}
	return continueTransportRecoveryExport(root, c, a, tx, false)
}

func resumeTransportRecoveryBatch(root string, c *Catalog, a *InstalledDerivedTransportRecovery, ref TransportRecoveryReference) (*RetranslationExportResult, error) {
	b, err := readCompatibilityFile(root, transportRecoveryDirectory(a.Locale, a.Identity)+"/.export.pending")
	if err != nil {
		return nil, err
	}
	var tx recoveryExportTransaction
	if err = contentidentity.StrictJSON(b, &tx); err != nil {
		return nil, err
	}
	if tx.Schema != "go-learning/transport-recovery-export-transaction/v1" || tx.Seal != recoveryExportSeal(tx) || tx.Authority != ref {
		return nil, fmt.Errorf("mutation=unknown: recovery export seal/owner mismatch")
	}
	if err = validateRecoveryChildManifest(a, &tx.Manifest); err != nil {
		return nil, err
	}
	expected, err := buildRecoveryExportTransaction(root, a, &tx.Manifest, ref)
	if err != nil || !bytes.Equal(mustJSON(expected), mustJSON(tx)) {
		return nil, fmt.Errorf("mutation=unknown: recovery export sealed inputs changed: %v", err)
	}
	return continueTransportRecoveryExport(root, c, a, tx, true)
}

func continueTransportRecoveryExport(root string, c *Catalog, a *InstalledDerivedTransportRecovery, tx recoveryExportTransaction, resume bool) (result *RetranslationExportResult, resultErr error) {
	c, err := recoveryPackageCatalog(c, a.ContentPackage.ID)
	if err != nil {
		return nil, err
	}
	pending := transportRecoveryDirectory(a.Locale, a.Identity) + "/.export.pending"
	if err = checkRecoveryParents(root, pending); err != nil {
		return nil, err
	}
	flags := os.O_RDWR | syscall.O_NOFOLLOW
	if !resume {
		flags |= os.O_CREATE | os.O_EXCL
	}
	f, err := os.OpenFile(filepath.Join(root, pending), flags, 0600)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return nil, fmt.Errorf("active recovery export owner")
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	defer func() {
		if resultErr != nil {
			resultErr = fmt.Errorf("mutation=unknown: pending=%s: %w", pending, resultErr)
		}
	}()
	if err = recoveryRequireAbsent(root, recoveryRecordPending(a)); err != nil {
		return nil, err
	}
	if err = recoveryRequireAbsent(root, recoveryRootDirectory(a.Locale)+"/supersessions/"+a.Identity+".json"); err != nil {
		return nil, fmt.Errorf("recovery export authority is no longer the leaf: %w", err)
	}
	if !resume {
		if _, err = f.Write(mustJSON(tx)); err != nil {
			return nil, err
		}
		if err = f.Sync(); err != nil {
			return nil, err
		}
	}
	base := filepath.Join(root, "data/retranslation-runs", a.Locale)
	fresh, _, err := prepareTransportRecovery(root, c, TransportRecoveryOptions{Locale: a.Locale, ContentPackage: a.ContentPackage.ID, PredecessorBatch: a.Predecessor.BatchID, Supersedes: a.Supersedes})
	if err != nil || fresh == nil || fresh.Identity != a.Identity {
		return nil, fmt.Errorf("recovery export epoch/CAS mismatch: %v", err)
	}
	final := filepath.Join(base, tx.Manifest.BatchID)
	if info, err := os.Lstat(final); err == nil {
		if !resume || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("mutation=unknown: recovery child destination occupied")
		}
		for _, entry := range tx.Members {
			b, err := readCompatibilityFile(root, "data/retranslation-runs/"+a.Locale+"/"+tx.Manifest.BatchID+"/"+entry.Reference.Path)
			if err != nil || !bytes.Equal(b, entry.Data) {
				return nil, fmt.Errorf("mutation=unknown: installed recovery child mismatch")
			}
		}
	} else {
		if !os.IsNotExist(err) {
			return nil, err
		}
		_, max, err := scanRetranslationBatches(base, a.Locale, c)
		if err != nil {
			return nil, err
		}
		want := a.AllowedAction.Provider + "-" + a.Locale + "-" + fmt.Sprintf("%03d", max)
		if want != tx.Manifest.BatchID {
			return nil, fmt.Errorf("mutation=unknown: sealed recovery ordinal/CAS changed")
		}
		stage := filepath.Join(base, ".transport-recovery-"+a.Identity+".stage")
		if err = os.Mkdir(stage, 0700); err != nil && !(resume && os.IsExist(err)) {
			return nil, err
		}
		for _, entry := range tx.Members {
			if !validRecoveryFileReference(entry.Reference) || entry.Reference.SHA256 != sum(entry.Data) || entry.Reference.Size != int64(len(entry.Data)) {
				return nil, fmt.Errorf("invalid sealed export member")
			}
			rel, err := repositoryRelativePath(root, filepath.Join(stage, entry.Reference.Path))
			if err != nil {
				return nil, err
			}
			if err = publishRecoverySealedFile(root, rel, entry.Data, resume); err != nil {
				return nil, err
			}
		}
		expected := map[string]bool{}
		for _, entry := range tx.Members {
			expected[filepath.ToSlash(filepath.Join("data/retranslation-runs", a.Locale, filepath.Base(stage), entry.Reference.Path))] = true
		}
		if err = verifyRecoveryManagedDirectory(root, filepath.ToSlash(filepath.Join("data/retranslation-runs", a.Locale, filepath.Base(stage))), expected, ""); err != nil {
			return nil, err
		}
		fresh, _, err = prepareTransportRecovery(root, c, TransportRecoveryOptions{Locale: a.Locale, ContentPackage: a.ContentPackage.ID, PredecessorBatch: a.Predecessor.BatchID, Supersedes: a.Supersedes})
		if err != nil || fresh == nil || fresh.Identity != a.Identity {
			return nil, fmt.Errorf("mutation=unknown: recovery export publication CAS changed: %v", err)
		}
		if err = recoveryRequireAbsent(root, recoveryRecordPending(a)); err != nil {
			return nil, err
		}
		if err = renameGenerationDirNoReplace(stage, final); err != nil {
			return nil, fmt.Errorf("mutation=unknown: recovery child publication: %w", err)
		}
	}
	expectedFinal := map[string]bool{}
	for _, entry := range tx.Members {
		expectedFinal["data/retranslation-runs/"+a.Locale+"/"+tx.Manifest.BatchID+"/"+entry.Reference.Path] = true
	}
	if err = verifyRecoveryManagedDirectory(root, "data/retranslation-runs/"+a.Locale+"/"+tx.Manifest.BatchID, expectedFinal, ""); err != nil {
		return nil, err
	}
	if err = os.Remove(filepath.Join(root, pending)); err != nil {
		return nil, fmt.Errorf("recovery child committed, pending cleanup failed: %w", err)
	}
	r := &RetranslationExportResult{Locale: a.Locale, BatchID: tx.Manifest.BatchID, BatchPath: filepath.ToSlash(filepath.Join("data/retranslation-runs", a.Locale, tx.Manifest.BatchID)), UnitKind: tx.Manifest.UnitKind, UnitCount: tx.Manifest.UnitCount}
	r.TransportRecovery = &tx.Authority
	r.PredecessorBatchID = a.Predecessor.BatchID
	for _, u := range tx.Manifest.Units {
		r.UnitIDs = append(r.UnitIDs, u.UnitID)
	}
	return r, nil
}
