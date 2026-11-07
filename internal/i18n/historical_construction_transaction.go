package i18n

import (
	"bytes"
	"os"
	"path/filepath"
	"syscall"
)

type historicalBytesTransaction struct {
	Schema            string                               `json:"schema"`
	Owner             string                               `json:"owner"`
	DeclarationKey    string                               `json:"declaration_key"`
	AuthorityIdentity string                               `json:"authority_identity"`
	Token             string                               `json:"cas_token"`
	CAS               historicalBytesCAS                   `json:"preconditions"`
	Authority         HistoricalConstructionBytesAuthority `json:"authority"`
	Blob              []byte                               `json:"sealed_blob"`
	Evidence          []byte                               `json:"sealed_evidence"`
	Index             historicalBytesIndex                 `json:"index"`
	Seal              string                               `json:"seal"`
}

func historicalBytesOwner(key, id, token string) string {
	return sum(mustJSON([]string{historicalBytesTransactionSchema, key, id, token}))
}
func historicalBytesSeal(tx historicalBytesTransaction) string {
	tx.Seal = ""
	return sum(mustJSON(tx))
}

func historicalBytesTransactionFrom(p *historicalBytesPrepared) historicalBytesTransaction {
	tx := historicalBytesTransaction{Schema: historicalBytesTransactionSchema, DeclarationKey: historicalBytesKey(p.Graph.Declaration), AuthorityIdentity: p.Authority.Identity, Token: p.Token, CAS: p.CAS, Authority: *p.Authority, Blob: p.Blob, Evidence: p.Evidence, Index: p.Index}
	tx.Owner = historicalBytesOwner(tx.DeclarationKey, tx.AuthorityIdentity, tx.Token)
	tx.Seal = historicalBytesSeal(tx)
	return tx
}

func validateHistoricalBytesTransaction(tx *historicalBytesTransaction, o HistoricalBytesOptions, id, token string) error {
	if tx.Schema != historicalBytesTransactionSchema || !validSHA256(id) || !validSHA256(token) || tx.AuthorityIdentity != id || tx.Token != token || tx.Owner != historicalBytesOwner(tx.DeclarationKey, id, token) || tx.Seal != historicalBytesSeal(*tx) || !bytes.Equal(mustJSON(tx.CAS.Options), mustJSON(o)) || sum(mustJSON(tx.CAS)) != token || tx.Authority.Identity != id || tx.DeclarationKey != historicalBytesKey(tx.Authority.Statement.declaration()) {
		return hbError("TRANSACTION_MISMATCH", "sealed owner/flags/identity/CAS mismatch")
	}
	if err := validateHistoricalBytesAuthority(&tx.Authority); err != nil {
		return err
	}
	if len(tx.Blob) > historicalBytesMaxSource || len(tx.Evidence) > historicalBytesMaxEvidence || len(mustJSON(&tx.Authority)) > historicalBytesMaxAuthority {
		return hbError("SIZE_MISMATCH", "sealed members exceed bounds")
	}
	if err := historicalBytesVerifyStatement(tx.Authority.Statement.declaration(), tx.Blob); err != nil {
		return err
	}
	a := tx.Authority
	ac := a.Acquisition
	if ac.SourcePath != o.Source || a.SourceKind != o.SourceKind || ac.EvidencePath != o.Evidence || ac.Source != tx.CAS.Source || ac.Source.SHA256 != sum(tx.Blob) || ac.Source.Size != int64(len(tx.Blob)) || ac.EvidenceSHA256 != sum(tx.Evidence) || ac.EvidenceSize != int64(len(tx.Evidence)) || ac.EvidenceSHA256 != o.EvidenceSHA256 || tx.CAS.Evidence.SHA256 != sum(tx.Evidence) || tx.CAS.Evidence.Size != int64(len(tx.Evidence)) || !bytes.Equal(mustJSON(tx.Index), historicalBytesExpectedIndexBytes(&a)) {
		return hbError("TRANSACTION_MISMATCH", "sealed payload/ref/observation mismatch")
	}
	return nil
}

// Record and resume share the official namespace lock. The sealed transaction
// is the only source of already ingested external bytes after interruption.
func RecordHistoricalConstructionBytes(root string, c *Catalog, o HistoricalBytesOptions, id, token string, resume bool) (*HistoricalBytesPlan, error) {
	p := hbPlan(o, "record")
	if !validSHA256(id) || !validSHA256(token) {
		return hbFail(p, hbError("CAS_MISMATCH", "record needs exact identity and CAS token"))
	}
	unlock, err := lockRetranslationBatchNamespace(root)
	if err != nil {
		return hbFail(p, hbWrap("TRANSACTION_PENDING", err))
	}
	defer unlock()
	var tx historicalBytesTransaction
	var prep *historicalBytesPrepared
	if resume {
		g, e := loadHistoricalBytesGraph(root, c, o)
		if e != nil {
			p.Mutation = "unknown"
			return hbFail(p, e)
		}
		pending := historicalBytesPendingPath(o.Locale, historicalBytesKey(g.Declaration))
		p.Pending = &pending
		b, e := readHistoricalBytesBoundedRepository(root, pending, historicalBytesMaxTransaction)
		if e != nil {
			return hbFail(p, hbWrap("TRANSACTION_PENDING", e))
		}
		p.Mutation = "unknown"
		if e = historicalBytesStrictJSON(b, &tx); e != nil {
			return hbFail(p, hbWrap("TRANSACTION_MISMATCH", e))
		}
		if e = validateHistoricalBytesTransaction(&tx, o, id, token); e != nil {
			return hbFail(p, e)
		}
		if !historicalBytesSameDeclaration(g.Declaration, tx.Authority.Statement.declaration()) {
			return hbFail(p, hbError("TRANSACTION_MISMATCH", "sealed declaration no longer matches graph"))
		}
		prep = &historicalBytesPrepared{Graph: g, Authority: &tx.Authority, Blob: tx.Blob, Evidence: tx.Evidence, CAS: tx.CAS, Token: tx.Token, Index: tx.Index}
	} else {
		prep, err = prepareHistoricalBytes(root, c, o)
		if err != nil {
			return hbFail(p, err)
		}
		admission, e := historicalBytesAdmission(root, c, p, prep)
		if e != nil {
			if x, ok := e.(historicalBytesError); ok && x.code == "TRANSACTION_MISMATCH" {
				p.Mutation = "unknown"
			}
			return admission, e
		}
		if admission.Status == "ALREADY_RECORDED" {
			return hbFail(p, hbError("DUPLICATE_RECORD", "declaration already recorded; use check"))
		}
		if prep.Authority.Identity != id || prep.Token != token {
			return hbFail(p, hbError("CAS_MISMATCH", "fresh preflight differs from expected identity/CAS"))
		}
		tx = historicalBytesTransactionFrom(prep)
	}
	key := tx.DeclarationKey
	pending := historicalBytesPendingPath(o.Locale, key)
	p.Pending = &pending
	if err = checkRecoveryParents(root, pending); err != nil {
		return hbFail(p, hbWrap("UNSAFE_PATH", err))
	}
	if !resume {
		if err = makeHistoricalBytesDirectories(root, filepath.Dir(filepath.Join(root, pending)), 0755); err != nil {
			return hbFail(p, err)
		}
	}
	flags := os.O_RDWR | syscall.O_NOFOLLOW | syscall.O_NONBLOCK
	if !resume {
		flags |= os.O_CREATE | os.O_EXCL
	}
	f, err := os.OpenFile(filepath.Join(root, pending), flags, 0600)
	if err != nil {
		return hbFail(p, hbWrap("TRANSACTION_PENDING", err))
	}
	defer f.Close()
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return hbFail(p, hbWrap("TRANSACTION_PENDING", err))
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		return hbFail(p, hbError("TRANSACTION_MISMATCH", "unsafe pending owner"))
	}
	if _, err = historicalBytesStat(info, nil); err != nil {
		return hbFail(p, err)
	}
	if resume {
		b, e := readHistoricalBytesBoundedRepository(root, pending, historicalBytesMaxTransaction)
		if e != nil || !bytes.Equal(b, mustJSON(tx)) {
			return hbFail(p, hbError("TRANSACTION_MISMATCH", "pending changed before owner lock"))
		}
	} else {
		p.Mutation = "unknown"
		// Check the acquisition again immediately before sealing. Own pending
		// progress is not part of the source/historical CAS token.
		fresh, e := prepareHistoricalBytes(root, c, o)
		if e != nil {
			return hbFail(p, e)
		}
		if fresh.Token != token || !bytes.Equal(mustJSON(fresh.Authority), mustJSON(prep.Authority)) {
			return hbFail(p, hbError("CAS_MISMATCH", "source/graph changed before seal"))
		}
		sealed := mustJSON(tx)
		if len(sealed) > historicalBytesMaxTransaction {
			return hbFail(p, hbError("SIZE_MISMATCH", "sealed transaction exceeds bound"))
		}
		if _, err = f.Write(sealed); err != nil {
			return hbFail(p, err)
		}
		if err = f.Sync(); err != nil {
			return hbFail(p, err)
		}
		if err = syncHistoricalBytesDirectory(filepath.Dir(filepath.Join(root, pending))); err != nil {
			return hbFail(p, err)
		}
	}
	p.Mutation = "unknown"
	if err = continueHistoricalBytesPublication(root, c, p, prep, &tx, resume); err != nil {
		return hbFail(p, err)
	}
	p.Status = "RECORDED"
	p.Recorded = true
	p.Mutation = "known"
	p.Pending = nil
	p.PlannedAuthority = nil
	p.AuthorityIdentity = &tx.AuthorityIdentity
	p.AuthorityPath = &tx.Index.Authority.Path
	p.Authority = &tx.Index.Authority
	p.Blob = &tx.Index.Blob
	p.Evidence = &tx.Authority.Acquisition.Evidence
	d := tx.Authority.Statement.declaration()
	p.Declaration = &d
	p.DeclarationKey = &key
	p.BytesSHA256 = &d.SHA256
	p.SourceVerification = "verified-sealed-input"
	p.BlobVerified = true
	p.EvidenceVerified = true
	p.HistoricalIntegrity = true
	p.CAS = nil
	ib := mustJSON(tx.Index)
	r := recoveryMemberReference(historicalBytesIndexPath(o.Locale, key), ib)
	p.Index = &r
	return p, nil
}

func continueHistoricalBytesPublication(root string, c *Catalog, p *HistoricalBytesPlan, prep *historicalBytesPrepared, tx *historicalBytesTransaction, resume bool) error {
	a := &tx.Authority
	d := a.Statement.declaration()
	indexPath := historicalBytesIndexPath(d.Locale, tx.DeclarationKey)
	pending := historicalBytesPendingPath(d.Locale, tx.DeclarationKey)
	stage := historicalBytesStagePath(d.Locale, tx.DeclarationKey)
	indexBytes := mustJSON(tx.Index)
	if _, err := os.Lstat(filepath.Join(root, indexPath)); err == nil {
		if !resume {
			return hbError("DUPLICATE_RECORD", "commit point occupied")
		}
		b, _, e := readHistoricalBytesAt(root, indexPath, historicalBytesMaxAuthority)
		if e != nil || !bytes.Equal(b, indexBytes) {
			return historicalBytesLabelError("occupied commit point differs: %v", e)
		}
		if _, _, _, e = loadCommittedHistoricalBytes(root, c, d, true); e != nil {
			return e
		}
		p.Mutation = "known" // A proven commit does not need an unchanged transport phase.
	} else {
		if !os.IsNotExist(err) {
			return hbWrap("IO_FAILURE", err)
		}
		if err = verifyHistoricalBytesPreparedCAS(root, c, prep); err != nil {
			return err
		}
		if err = checkRecoveryParents(root, stage); err != nil {
			return err
		}
		if err = os.Mkdir(filepath.Join(root, stage), 0700); err != nil && !(resume && os.IsExist(err)) {
			return hbWrap("TRANSACTION_MISMATCH", err)
		}
		stageFiles := map[string][]byte{stage + "/blob.bin": tx.Blob, stage + "/authority/authority.json": mustJSON(a), stage + "/authority/acquisition-evidence.bin": tx.Evidence, stage + "/index.json": indexBytes}
		expected := map[string]bool{}
		for _, name := range []string{stage + "/blob.bin", stage + "/authority/authority.json", stage + "/authority/acquisition-evidence.bin", stage + "/index.json"} {
			expected[name] = true
			if err = publishHistoricalBytesFile(root, name, stageFiles[name], resume); err != nil {
				return hbWrap("TRANSACTION_MISMATCH", err)
			}
		}
		if err = verifyHistoricalBytesDirectory(root, stage, expected, ""); err != nil {
			return hbWrap("TRANSACTION_MISMATCH", err)
		}
		if err = publishHistoricalBytesFile(root, a.Statement.Blob.Path, tx.Blob, true); err != nil {
			return hbWrap("CORRUPT_RECORDED_EVIDENCE", err)
		}
		dir := historicalBytesAuthorityDir(d.Locale, a.Identity)
		if _, err = os.Lstat(filepath.Join(root, dir)); err == nil {
			if !resume {
				return historicalBytesLabelError("unowned authority directory exists")
			}
			for _, ref := range []RecoveryFileReference{tx.Index.Authority, a.Acquisition.Evidence} {
				if _, err = readHistoricalBytesStored(root, ref, historicalBytesMaxEvidence); err != nil {
					return err
				}
			}
		} else {
			if !os.IsNotExist(err) {
				return err
			}
			if err = checkRecoveryParents(root, dir); err != nil {
				return err
			}
			if err = makeHistoricalBytesDirectories(root, filepath.Dir(filepath.Join(root, dir)), 0755); err != nil {
				return err
			}
			if err = renameGenerationDirNoReplace(filepath.Join(root, stage, "authority"), filepath.Join(root, dir)); err != nil {
				return err
			}
			if err = syncHistoricalBytesDirectory(filepath.Dir(filepath.Join(root, dir))); err != nil {
				return err
			}
		}
		expectedDir := map[string]bool{tx.Index.Authority.Path: true, a.Acquisition.Evidence.Path: true}
		if err = verifyHistoricalBytesDirectory(root, dir, expectedDir, ""); err != nil {
			return err
		}
		if err = verifyHistoricalBytesPreparedCAS(root, c, prep); err != nil {
			return err
		}
		for _, ref := range []RecoveryFileReference{a.Statement.Blob, tx.Index.Authority, a.Acquisition.Evidence} {
			if _, err = readHistoricalBytesStored(root, ref, historicalBytesMaxEvidence); err != nil {
				return err
			}
		}
		sealed, e := readHistoricalBytesBoundedRepository(root, pending, historicalBytesMaxTransaction)
		if e != nil || !bytes.Equal(sealed, mustJSON(tx)) {
			return hbError("TRANSACTION_MISMATCH", "pending owner changed before commit")
		}
		if err = historicalBytesExistingIndex(root, prep); err != nil {
			return hbWrap("TRANSACTION_MISMATCH", err)
		}
		if err = publishHistoricalBytesFile(root, indexPath, indexBytes, false); err != nil {
			return hbWrap("TRANSACTION_MISMATCH", err)
		}
		if _, _, _, err = loadCommittedHistoricalBytes(root, c, d, true); err != nil {
			return err
		}
		p.Mutation = "known"
	}
	p.Authority = &tx.Index.Authority
	p.Blob = &tx.Index.Blob
	p.Evidence = &a.Acquisition.Evidence
	indexRef := recoveryMemberReference(indexPath, indexBytes)
	p.Index = &indexRef
	// Only sealed, matching stage members may be removed. Unknown siblings or
	// replacements keep both journal and state intact for readonly diagnosis.
	if err := cleanupHistoricalBytesStage(root, stage, tx); err != nil {
		return hbWrap("TRANSACTION_MISMATCH", err)
	}
	b, err := readHistoricalBytesBoundedRepository(root, pending, historicalBytesMaxTransaction)
	if err != nil || !bytes.Equal(b, mustJSON(tx)) {
		p.Mutation = "unknown"
		return historicalBytesLabelError("pending changed after commit: %v", err)
	}
	if err = os.Remove(filepath.Join(root, pending)); err != nil {
		return hbWrap("IO_FAILURE", err)
	}
	return syncHistoricalBytesDirectory(filepath.Dir(filepath.Join(root, pending)))
}

func cleanupHistoricalBytesStage(root, stage string, tx *historicalBytesTransaction) error {
	if _, err := os.Lstat(filepath.Join(root, stage)); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	expected := map[string][]byte{stage + "/blob.bin": tx.Blob, stage + "/index.json": mustJSON(tx.Index), stage + "/authority/authority.json": mustJSON(tx.Authority), stage + "/authority/acquisition-evidence.bin": tx.Evidence}
	allowed := map[string]bool{}
	for path := range expected {
		allowed[path] = true
	}
	if err := verifyHistoricalBytesDirectory(root, stage, allowed, ""); err != nil {
		return err
	}
	for path, b := range expected {
		if _, err := os.Lstat(filepath.Join(root, path)); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		if _, err := readHistoricalBytesStored(root, recoveryMemberReference(path, b), historicalBytesMaxEvidence); err != nil {
			return err
		}
	}
	// No unknown content survived the complete validation above.
	return os.RemoveAll(filepath.Join(root, stage))
}
