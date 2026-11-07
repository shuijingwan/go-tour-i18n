package i18n

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

type historicalBytesCAS struct {
	Options        HistoricalBytesOptions     `json:"options"`
	Observed       []RecoveryDependency       `json:"historical_observations"`
	Implementation []RecoveryFileReference    `json:"verifier_implementation"`
	Source         HistoricalBytesObservation `json:"source"`
	Evidence       HistoricalBytesObservation `json:"evidence"`
	BlobPresent    bool                       `json:"blob_present"`
}

type historicalBytesPrepared struct {
	Graph          *historicalBytesGraph
	Authority      *HistoricalConstructionBytesAuthority
	Blob, Evidence []byte
	CAS            historicalBytesCAS
	Token          string
	Index          historicalBytesIndex
}

func historicalBytesVerifierRefs(root string) ([]RecoveryFileReference, error) {
	var refs []RecoveryFileReference
	for _, path := range historicalBytesImplementationPaths {
		ref, _, err := recoveryDiskReference(root, path)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

func historicalBytesRootAdmissible(root string, d HistoricalBytesDeclaration) error {
	for _, p := range []string{recoveryRootDirectory(d.Locale) + "/roots/" + d.Archive.SHA256 + ".json", recoveryRootDirectory(d.Locale) + "/roots/" + d.Archive.SHA256 + ".record.pending"} {
		if err := recoveryRequireAbsent(root, p); err != nil {
			return hbWrap("ROOT_NOT_ADMISSIBLE", err)
		}
	}
	return nil
}

func readHistoricalBytesAt(root, path string, max int64) ([]byte, RecoveryFileReference, error) {
	var zero RecoveryFileReference
	if validateGenerationInstallPath(path) != nil {
		return nil, zero, hbError("UNSAFE_PATH", "invalid repository reference")
	}
	if err := checkRecoveryParents(root, path); err != nil {
		return nil, zero, err
	}
	info, err := os.Lstat(filepath.Join(root, path))
	if err != nil {
		return nil, zero, err
	}
	if !info.Mode().IsRegular() || info.Size() > max {
		return nil, zero, hbError("CORRUPT_RECORDED_EVIDENCE", "unsafe stored artifact")
	}
	// Bound the first read too; a concurrent swap cannot turn it into an
	// unbounded ReadFile. The exact descriptor/physical digest is checked again.
	b, err := readHistoricalBytesBoundedRepository(root, path, max)
	if err != nil {
		return nil, zero, err
	}
	ref := recoveryMemberReference(path, b)
	b, err = readHistoricalBytesStored(root, ref, max)
	return b, ref, err
}

func loadCommittedHistoricalBytes(root string, c *Catalog, d HistoricalBytesDeclaration, verifyGraph bool) (*HistoricalConstructionBytesAuthority, []byte, []RecoveryFileReference, error) {
	key := historicalBytesKey(d)
	indexPath := historicalBytesIndexPath(d.Locale, key)
	ib, indexRef, err := readHistoricalBytesAt(root, indexPath, historicalBytesMaxAuthority)
	if err != nil {
		return nil, nil, nil, err
	}
	var index historicalBytesIndex
	if err = historicalBytesStrictJSON(ib, &index); err != nil {
		return nil, nil, nil, hbWrap("CORRUPT_RECORDED_EVIDENCE", err)
	}
	if index.Schema != historicalBytesIndexSchema || index.DeclarationKey != key || !validSHA256(index.AuthorityIdentity) || index.Authority.Path != historicalBytesAuthorityDir(d.Locale, index.AuthorityIdentity)+"/authority.json" {
		return nil, nil, nil, hbError("CORRUPT_RECORDED_EVIDENCE", "index/declaration identity mismatch")
	}
	ab, err := readHistoricalBytesStored(root, index.Authority, historicalBytesMaxAuthority)
	if err != nil {
		return nil, nil, nil, err
	}
	a, err := decodeHistoricalBytesAuthority(ab)
	if err != nil {
		return nil, nil, nil, err
	}
	if !historicalBytesSameDeclaration(d, a.Statement.declaration()) || a.Identity != index.AuthorityIdentity || a.Statement.Blob != index.Blob {
		return nil, nil, nil, hbError("CORRUPT_RECORDED_EVIDENCE", "supplemental authority belongs to another declaration")
	}
	if verifyGraph {
		o := HistoricalBytesOptions{Locale: d.Locale, ContentPackage: d.ContentPackage.ID, BatchID: d.Invocation.BatchID, UnitID: d.Invocation.UnitID, LogicalPath: d.LogicalPath}
		g, e := loadHistoricalBytesGraph(root, c, o)
		if e != nil {
			return nil, nil, nil, e
		}
		if !historicalBytesSameDeclaration(d, g.Declaration) {
			return nil, nil, nil, hbError("CORRUPT_RECORDED_EVIDENCE", "recorded historical declaration no longer verifies")
		}
	}
	blob, err := readHistoricalBytesStored(root, index.Blob, historicalBytesMaxSource)
	if err != nil {
		return nil, nil, nil, err
	}
	if _, err = readHistoricalBytesStored(root, a.Acquisition.Evidence, historicalBytesMaxEvidence); err != nil {
		return nil, nil, nil, err
	}
	expected := map[string]bool{index.Authority.Path: true, a.Acquisition.Evidence.Path: true}
	if err = verifyHistoricalBytesDirectory(root, historicalBytesAuthorityDir(d.Locale, a.Identity), expected, ""); err != nil {
		return nil, nil, nil, hbWrap("CORRUPT_RECORDED_EVIDENCE", err)
	}
	return a, blob, []RecoveryFileReference{indexRef, index.Authority, index.Blob, a.Acquisition.Evidence}, nil
}

func prepareHistoricalBytes(root string, c *Catalog, o HistoricalBytesOptions) (*historicalBytesPrepared, error) {
	g, err := loadHistoricalBytesGraph(root, c, o)
	if err != nil {
		return nil, err
	}
	if !historicalBytesKind(o.SourceKind) || !validSHA256(o.EvidenceSHA256) {
		return nil, hbError("INVALID_ACQUISITION", "explicit known source kind and exact evidence SHA required")
	}
	blob, source, err := readHistoricalBytesExternal(root, o.Source, historicalBytesMaxSource)
	if err != nil {
		return nil, err
	}
	if source.SHA256 != g.Declaration.SHA256 {
		return nil, hbError("HASH_MISMATCH", "source is not the declared historical byte sequence")
	}
	if g.Declaration.Size != nil && source.Size != *g.Declaration.Size {
		return nil, hbError("SIZE_MISMATCH", "source differs from old explicit size")
	}
	current, err := os.Lstat(filepath.Join(root, g.Declaration.LogicalPath))
	if err != nil && !os.IsNotExist(err) {
		return nil, hbWrap("INVALID_ACQUISITION", err)
	}
	input, err := os.Lstat(o.Source)
	if err != nil {
		return nil, hbWrap("INVALID_ACQUISITION", err)
	}
	if current != nil && os.SameFile(current, input) {
		return nil, hbError("INVALID_ACQUISITION", "current worktree inode cannot be adopted as history")
	}
	evidence, evidenceObs, err := readHistoricalBytesExternal(root, o.Evidence, historicalBytesMaxEvidence)
	if err != nil {
		return nil, err
	}
	if evidenceObs.SHA256 != o.EvidenceSHA256 || len(evidence) == 0 {
		return nil, hbError("INVALID_ACQUISITION", "evidence SHA/size mismatch")
	}
	if source.Device == evidenceObs.Device && source.Inode == evidenceObs.Inode {
		return nil, hbError("INVALID_ACQUISITION", "source itself is not acquisition provenance")
	}
	st := historicalBytesStatement(g.Declaration, source.SHA256, source.Size, recoveryMemberReference(historicalBytesBlobPath(source.SHA256), blob))
	id := HistoricalConstructionBytesIdentity(st)
	a := &HistoricalConstructionBytesAuthority{Schema: HistoricalConstructionBytesSchema, Capability: HistoricalConstructionBytesCapability, Statement: st, Identity: id, SourceKind: o.SourceKind, Acquisition: HistoricalBytesAcquisition{Method: "existing-whole-file-copy", SourcePath: o.Source, Source: source, EvidencePath: o.Evidence, Evidence: recoveryMemberReference(historicalBytesAuthorityDir(o.Locale, id)+"/acquisition-evidence.bin", evidence), EvidenceSHA256: evidenceObs.SHA256, EvidenceSize: evidenceObs.Size}}
	if err = validateHistoricalBytesAuthority(a); err != nil {
		return nil, err
	}
	if len(mustJSON(a)) > historicalBytesMaxAuthority {
		return nil, hbError("SIZE_MISMATCH", "authority audit exceeds bound")
	}
	impl, err := historicalBytesVerifierRefs(root)
	if err != nil {
		return nil, hbWrap("IO_FAILURE", err)
	}
	p := &historicalBytesPrepared{Graph: g, Authority: a, Blob: blob, Evidence: evidence, CAS: historicalBytesCAS{Options: o, Observed: g.Observed, Implementation: impl, Source: source, Evidence: evidenceObs}}
	p.Index = historicalBytesIndex{historicalBytesIndexSchema, historicalBytesKey(g.Declaration), id, recoveryMemberReference(historicalBytesAuthorityDir(o.Locale, id)+"/authority.json", mustJSON(a)), st.Blob}
	if info, e := os.Lstat(filepath.Join(root, st.Blob.Path)); e == nil {
		if info.Mode().Perm() != 0444 {
			return nil, hbError("CORRUPT_RECORDED_EVIDENCE", "reused published blob mode must be 0444")
		}
		b, e := readHistoricalBytesStored(root, st.Blob, historicalBytesMaxSource)
		if e != nil || !bytes.Equal(b, blob) {
			return nil, hbError("CORRUPT_RECORDED_EVIDENCE", "pre-existing blob conflicts with acquired bytes")
		}
		p.CAS.BlobPresent = true
	} else if !os.IsNotExist(e) {
		return nil, hbWrap("IO_FAILURE", e)
	}
	p.Token = sum(mustJSON(p.CAS))
	return p, nil
}

func PreflightHistoricalConstructionBytes(root string, c *Catalog, o HistoricalBytesOptions) (*HistoricalBytesPlan, error) {
	p := hbPlan(o, "preflight")
	prep, err := prepareHistoricalBytes(root, c, o)
	if err != nil {
		return hbFail(p, err)
	}
	return historicalBytesAdmission(root, c, p, prep)
}

func historicalBytesAdmission(root string, c *Catalog, p *HistoricalBytesPlan, prep *historicalBytesPrepared) (*HistoricalBytesPlan, error) {
	d := prep.Graph.Declaration
	key := historicalBytesKey(d)
	p.Declaration = &d
	p.DeclarationKey = &key
	p.BytesSHA256 = &d.SHA256
	p.SourceVerification = "verified-live"
	p.EvidenceVerified = true
	p.HistoricalIntegrity = true
	if _, err := os.Lstat(filepath.Join(root, historicalBytesIndexPath(d.Locale, key))); err == nil {
		a, b, refs, e := loadCommittedHistoricalBytes(root, c, d, false)
		if e != nil {
			return hbFail(p, hbWrap("CORRUPT_RECORDED_EVIDENCE", e))
		}
		if !bytes.Equal(b, prep.Blob) || !bytes.Equal(mustJSON(a.Statement), mustJSON(prep.Authority.Statement)) {
			return hbFail(p, hbError("CORRUPT_RECORDED_EVIDENCE", "same declaration has conflicting published bytes"))
		}
		p.Status = "ALREADY_RECORDED"
		p.Authority = &refs[1]
		p.Blob = &refs[2]
		p.Index = &refs[0]
		p.Evidence = &refs[3]
		p.BlobVerified = true
		return p, nil
	} else if !os.IsNotExist(err) {
		return hbFail(p, hbWrap("IO_FAILURE", err))
	}
	if err := historicalBytesRootAdmissible(root, d); err != nil {
		return hbFail(p, err)
	}
	if err := recoveryRequireAbsent(root, historicalBytesPendingPath(d.Locale, key)); err != nil {
		return hbFail(p, hbWrap("TRANSACTION_PENDING", err))
	}
	if err := recoveryRequireAbsent(root, historicalBytesStagePath(d.Locale, key)); err != nil {
		return hbFail(p, hbWrap("TRANSACTION_MISMATCH", err))
	}
	if err := recoveryRequireAbsent(root, historicalBytesAuthorityDir(d.Locale, prep.Authority.Identity)); err != nil {
		return hbFail(p, hbWrap("TRANSACTION_MISMATCH", err))
	}
	available, _, _, err := resolveHistoricalBytesMaterial(root, c, prep.Graph, false)
	if err != nil {
		return hbFail(p, err)
	}
	if available != nil {
		return hbFail(p, hbError("SUPPLEMENT_NOT_NEEDED", "exact archive/surviving bytes already available"))
	}
	a := prep.Authority
	p.Status = "READY"
	p.PlannedAuthority = a
	p.AuthorityIdentity = &a.Identity
	p.AuthorityPath = &prep.Index.Authority.Path
	p.Blob = &a.Statement.Blob
	p.CAS = &prep.Token
	return p, nil
}

func CheckHistoricalConstructionBytes(root string, c *Catalog, locale, pkg, path, sha string) (*HistoricalBytesPlan, error) {
	p := hbPlan(HistoricalBytesOptions{Locale: locale, ContentPackage: pkg}, "check")
	p.SourceVerification = "not-required"
	if !validSHA256(sha) || validateGenerationInstallPath(path) != nil {
		return hbFail(p, hbError("UNSAFE_PATH", "exact authority path/SHA required"))
	}
	b, ref, err := readHistoricalBytesAt(root, path, historicalBytesMaxAuthority)
	if err != nil {
		return hbFail(p, hbWrap("CORRUPT_RECORDED_EVIDENCE", err))
	}
	if ref.SHA256 != sha {
		return hbFail(p, hbError("HASH_MISMATCH", "authority SHA mismatch"))
	}
	a, err := decodeHistoricalBytesAuthority(b)
	if err != nil {
		return hbFail(p, err)
	}
	d := a.Statement.declaration()
	if d.Locale != locale || d.ContentPackage.ID != pkg || path != historicalBytesAuthorityDir(locale, a.Identity)+"/authority.json" {
		return hbFail(p, hbError("WRONG_SCOPE", "authority request mismatch"))
	}
	g, err := loadHistoricalBytesGraph(root, c, HistoricalBytesOptions{Locale: locale, ContentPackage: pkg, BatchID: d.Invocation.BatchID, UnitID: d.Invocation.UnitID, LogicalPath: d.LogicalPath})
	if err != nil {
		return hbFail(p, err)
	}
	_, blob, refs, err := loadCommittedHistoricalBytes(root, c, g.Declaration, false)
	if err != nil {
		return hbFail(p, hbWrap("CORRUPT_RECORDED_EVIDENCE", err))
	}
	if refs[1] != ref || !historicalBytesSameDeclaration(d, g.Declaration) {
		return hbFail(p, hbError("CORRUPT_RECORDED_EVIDENCE", "authority is not the committed declaration"))
	}
	if _, _, _, err = resolveHistoricalBytesMaterial(root, c, g, true); err != nil {
		return hbFail(p, err)
	}
	key := historicalBytesKey(d)
	p.Status = "VERIFIED_HISTORICAL_EVIDENCE"
	p.BatchID = &d.Invocation.BatchID
	p.UnitID = &d.Invocation.UnitID
	p.Declaration = &d
	p.DeclarationKey = &key
	p.BytesSHA256 = &d.SHA256
	p.AuthorityIdentity = &a.Identity
	p.AuthorityPath = &ref.Path
	p.Authority = &ref
	p.Blob = &a.Statement.Blob
	p.Index = &refs[0]
	p.Evidence = &refs[3]
	p.BlobVerified = sum(blob) == d.SHA256
	p.EvidenceVerified = true
	p.HistoricalIntegrity = true
	return p, nil
}

// No callers may treat the existence of a loose CAS blob as recorded evidence.
// This is deliberately separate from the old archive's inventory lookup.
func resolveHistoricalBytesMaterial(root string, c *Catalog, g *historicalBytesGraph, requireSupplement bool) ([]byte, string, []RecoveryFileReference, error) {
	d := g.Declaration
	var archive []byte
	for _, b := range g.Files {
		if sum(b) == d.SHA256 {
			if d.Size != nil && int64(len(b)) != *d.Size {
				return nil, "", nil, hbError("SIZE_MISMATCH", "archive bytes disagree with old size")
			}
			if archive != nil && !bytes.Equal(archive, b) {
				return nil, "", nil, hbError("CORRUPT_RECORDED_EVIDENCE", "archive hash collision/conflict")
			}
			archive = b
		}
	}
	var surviving []byte
	if b, err := readCompatibilityReference(root, GlossaryArchiveReference{d.LogicalPath, d.SHA256}); err == nil {
		if d.Size != nil && int64(len(b)) != *d.Size {
			return nil, "", nil, hbError("SIZE_MISMATCH", "surviving bytes disagree with old size")
		}
		surviving = b
	}
	var supplemental []byte
	var refs []RecoveryFileReference
	indexPath := historicalBytesIndexPath(d.Locale, historicalBytesKey(d))
	if err := checkRecoveryParents(root, indexPath); err != nil {
		return nil, "", nil, hbWrap("CORRUPT_RECORDED_EVIDENCE", err)
	}
	if _, err := os.Lstat(filepath.Join(root, indexPath)); err == nil {
		_, b, r, e := loadCommittedHistoricalBytes(root, c, d, false)
		if e != nil {
			return nil, "", nil, hbWrap("CORRUPT_RECORDED_EVIDENCE", e)
		}
		supplemental = b
		refs = r
	} else if !os.IsNotExist(err) {
		return nil, "", nil, hbWrap("CORRUPT_RECORDED_EVIDENCE", err)
	} else if requireSupplement {
		return nil, "", nil, hbError("CORRUPT_RECORDED_EVIDENCE", "recorded declaration index missing")
	}
	var chosen []byte
	origin := ""
	for _, x := range []struct {
		b      []byte
		origin string
	}{{archive, "verified-archive"}, {surviving, "surviving-repository"}, {supplemental, "recorded-supplemental"}} {
		if x.b == nil {
			continue
		}
		if chosen == nil {
			chosen = x.b
			origin = x.origin
		} else if !bytes.Equal(chosen, x.b) {
			return nil, "", nil, hbError("CORRUPT_RECORDED_EVIDENCE", "sources conflict for same historical declaration")
		}
	}
	return chosen, origin, refs, nil
}

func verifyHistoricalBytesPreparedCAS(root string, c *Catalog, p *historicalBytesPrepared) error {
	g, err := loadHistoricalBytesGraph(root, c, p.CAS.Options)
	if err != nil {
		return err
	}
	if !historicalBytesSameDeclaration(g.Declaration, p.Graph.Declaration) || !bytes.Equal(mustJSON(g.Observed), mustJSON(p.CAS.Observed)) {
		return hbError("CAS_MISMATCH", "historical declaration/registry observations changed")
	}
	impl, err := historicalBytesVerifierRefs(root)
	if err != nil {
		return err
	}
	if !bytes.Equal(mustJSON(impl), mustJSON(p.CAS.Implementation)) {
		return hbError("CAS_MISMATCH", "verifier implementation changed")
	}
	return historicalBytesRootAdmissible(root, g.Declaration)
}

func historicalBytesExistingIndex(root string, prep *historicalBytesPrepared) error {
	return recoveryRequireAbsent(root, historicalBytesIndexPath(prep.Graph.Declaration.Locale, historicalBytesKey(prep.Graph.Declaration)))
}

func historicalBytesExpectedIndexBytes(a *HistoricalConstructionBytesAuthority) []byte {
	return mustJSON(historicalBytesIndex{historicalBytesIndexSchema, historicalBytesKey(a.Statement.declaration()), a.Identity, recoveryMemberReference(historicalBytesAuthorityDir(a.Statement.Locale, a.Identity)+"/authority.json", mustJSON(a)), a.Statement.Blob})
}

func historicalBytesVerifyStatement(d HistoricalBytesDeclaration, blob []byte) error {
	if sum(blob) != d.SHA256 {
		return hbError("HASH_MISMATCH", "sealed source differs from declaration")
	}
	if d.Size != nil && int64(len(blob)) != *d.Size {
		return hbError("SIZE_MISMATCH", "sealed source size differs from declaration")
	}
	return nil
}

func historicalBytesLabelError(format string, args ...any) error {
	return hbError("TRANSACTION_MISMATCH", fmt.Sprintf(format, args...))
}
