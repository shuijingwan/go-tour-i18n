package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"
)

const (
	HistoricalConstructionBytesSchema     = "go-learning/historical-construction-bytes/v1"
	HistoricalConstructionBytesCapability = "attest-exact-declared-historical-construction-bytes"
	historicalBytesIndexSchema            = "go-learning/historical-construction-bytes-index/v1"
	historicalBytesPlanSchema             = "go-learning/historical-construction-bytes-plan/v1"
	historicalBytesTransactionSchema      = "go-learning/historical-construction-bytes-transaction/v1"
	historicalBytesRoot                   = "data/historical-construction-bytes"
	historicalBytesMaxSource              = 4 << 20
	historicalBytesMaxEvidence            = 32 << 20
	historicalBytesMaxAuthority           = 64 << 10
	historicalBytesMaxTransaction         = 64 << 20
)

type HistoricalBytesPackage struct {
	ID                  string               `json:"id"`
	AuthoritySHA256     string               `json:"authority_sha256"`
	ScopeIdentitySHA256 string               `json:"scope_identity_sha256"`
	AuthorityReferences []RecoveryDependency `json:"authority_references"`
}

type HistoricalBytesInvocation struct {
	BatchID             string                `json:"batch_id"`
	TaskKind            string                `json:"task_kind"`
	Attempt             int                   `json:"attempt"`
	Manifest            RecoveryFileReference `json:"manifest"`
	ImportReceipt       RecoveryFileReference `json:"import_receipt"`
	InputIdentitySHA256 string                `json:"input_identity_sha256"`
	Provider            string                `json:"provider"`
	Model               string                `json:"model"`
	UnitID              string                `json:"unit_id"`
	UnitKind            UnitKind              `json:"unit_kind"`
	SourceSHA256        string                `json:"source_sha256"`
	InputSHA256         string                `json:"input_sha256"`
	LineageReferences   []RecoveryDependency  `json:"lineage_references"`
}

type HistoricalBytesMember struct {
	Container RecoveryFileReference `json:"container"`
	Member    RecoveryFileReference `json:"member"`
}

type HistoricalBytesDeclarationRef struct {
	Container RecoveryFileReference `json:"container"`
	Member    RecoveryFileReference `json:"member"`
	Selector  string                `json:"selector"`
}

type HistoricalBytesProducer struct {
	ID         string `json:"id"`
	Version    int    `json:"contract_version"`
	RoleOrigin string `json:"role_binding_origin"`
}

// This closed structure is both the declaration selector and K's identity
// input. In particular it contains no mutable final result or acquisition path.
type HistoricalBytesDeclaration struct {
	Locale         string                        `json:"locale"`
	ContentPackage HistoricalBytesPackage        `json:"content_package"`
	Invocation     HistoricalBytesInvocation     `json:"historical_invocation_ref"`
	Archive        RecoveryFileReference         `json:"historical_archive_ref"`
	Declaration    HistoricalBytesDeclarationRef `json:"historical_declaration_ref"`
	Proof          HistoricalBytesMember         `json:"historical_proof_member_ref"`
	LogicalPath    string                        `json:"logical_dependency_path"`
	SHA256         string                        `json:"declared_historical_sha256"`
	Size           *int64                        `json:"declared_historical_size"`
	Role           string                        `json:"dependency_role"`
	Producer       HistoricalBytesProducer       `json:"producer_contract"`
}

// Explicit fields preserve the v1 flattened wire layout and allow the strict
// JSON reader to validate every nested field without anonymous-field inference.
type HistoricalBytesStatement struct {
	Locale          string                        `json:"locale"`
	ContentPackage  HistoricalBytesPackage        `json:"content_package"`
	Invocation      HistoricalBytesInvocation     `json:"historical_invocation_ref"`
	Archive         RecoveryFileReference         `json:"historical_archive_ref"`
	Declaration     HistoricalBytesDeclarationRef `json:"historical_declaration_ref"`
	Proof           HistoricalBytesMember         `json:"historical_proof_member_ref"`
	LogicalPath     string                        `json:"logical_dependency_path"`
	SHA256          string                        `json:"declared_historical_sha256"`
	Size            *int64                        `json:"declared_historical_size"`
	Role            string                        `json:"dependency_role"`
	Producer        HistoricalBytesProducer       `json:"producer_contract"`
	RecoveredSHA256 string                        `json:"recovered_bytes_sha256"`
	RecoveredSize   int64                         `json:"recovered_bytes_size"`
	Blob            RecoveryFileReference         `json:"published_bytes_ref"`
}

func (s HistoricalBytesStatement) declaration() HistoricalBytesDeclaration {
	return HistoricalBytesDeclaration{s.Locale, s.ContentPackage, s.Invocation, s.Archive, s.Declaration, s.Proof, s.LogicalPath, s.SHA256, s.Size, s.Role, s.Producer}
}
func historicalBytesStatement(d HistoricalBytesDeclaration, sha string, size int64, blob RecoveryFileReference) HistoricalBytesStatement {
	return HistoricalBytesStatement{d.Locale, d.ContentPackage, d.Invocation, d.Archive, d.Declaration, d.Proof, d.LogicalPath, d.SHA256, d.Size, d.Role, d.Producer, sha, size, blob}
}

type HistoricalBytesObservation struct {
	SHA256    string `json:"sha256"`
	Size      int64  `json:"size"`
	Device    uint64 `json:"device"`
	Inode     uint64 `json:"inode"`
	Mode      uint32 `json:"mode"`
	LinkCount uint64 `json:"link_count"`
	MtimeNS   int64  `json:"mtime_ns"`
	CtimeNS   int64  `json:"ctime_ns"`
}

type HistoricalBytesAcquisition struct {
	Method         string                     `json:"acquisition_method"`
	SourcePath     string                     `json:"original_source_path"`
	Source         HistoricalBytesObservation `json:"source_observation"`
	EvidencePath   string                     `json:"original_evidence_path"`
	Evidence       RecoveryFileReference      `json:"published_evidence_ref"`
	EvidenceSHA256 string                     `json:"observed_evidence_sha256"`
	EvidenceSize   int64                      `json:"observed_evidence_size"`
}

type HistoricalConstructionBytesAuthority struct {
	Schema      string                     `json:"schema"`
	Capability  string                     `json:"capability"`
	Statement   HistoricalBytesStatement   `json:"statement"`
	Identity    string                     `json:"identity"`
	SourceKind  string                     `json:"recovery_source_kind"`
	Acquisition HistoricalBytesAcquisition `json:"recovery_acquisition_evidence"`
}

type historicalBytesIdentityCore struct {
	Schema     string                   `json:"schema"`
	Capability string                   `json:"capability"`
	Statement  HistoricalBytesStatement `json:"statement"`
	Identity   string                   `json:"identity"`
}

type historicalBytesIndex struct {
	Schema            string                `json:"schema"`
	DeclarationKey    string                `json:"declaration_key"`
	AuthorityIdentity string                `json:"authority_identity"`
	Authority         RecoveryFileReference `json:"authority_ref"`
	Blob              RecoveryFileReference `json:"published_bytes_ref"`
}

type HistoricalBytesOptions struct {
	Locale, ContentPackage, BatchID, UnitID, LogicalPath string
	Source, SourceKind, Evidence, EvidenceSHA256         string
}

type HistoricalBytesBlocker struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

type HistoricalBytesPlan struct {
	Schema                      string                                `json:"schema"`
	Mode                        string                                `json:"mode"`
	Status                      string                                `json:"status"`
	Locale                      string                                `json:"locale"`
	ContentPackage              string                                `json:"content_package"`
	BatchID                     *string                               `json:"batch_id"`
	UnitID                      *string                               `json:"unit_id"`
	Declaration                 *HistoricalBytesDeclaration           `json:"declaration"`
	DeclarationKey              *string                               `json:"declaration_key"`
	BytesSHA256                 *string                               `json:"bytes_sha256"`
	AuthorityIdentity           *string                               `json:"authority_identity"`
	AuthorityPath               *string                               `json:"authority_path"`
	PlannedAuthority            *HistoricalConstructionBytesAuthority `json:"planned_authority"`
	Blob                        *RecoveryFileReference                `json:"blob_ref"`
	Authority                   *RecoveryFileReference                `json:"authority_ref"`
	Index                       *RecoveryFileReference                `json:"index_ref"`
	Evidence                    *RecoveryFileReference                `json:"acquisition_evidence_ref"`
	SourceVerification          string                                `json:"source_verification"`
	EvidenceVerified            bool                                  `json:"evidence_verified"`
	BlobVerified                bool                                  `json:"blob_verified"`
	HistoricalIntegrity         bool                                  `json:"historical_integrity"`
	TransportRecoveryAuthorized bool                                  `json:"transport_recovery_authorized"`
	CAS                         *string                               `json:"cas_token"`
	Blockers                    []HistoricalBytesBlocker              `json:"blockers"`
	Recorded                    bool                                  `json:"recorded"`
	Mutation                    string                                `json:"mutation"`
	Pending                     *string                               `json:"pending_path"`
	AllowedNextOperation        *string                               `json:"allowed_next_operation"`
}

type historicalBytesError struct {
	code string
	err  error
}

func (e historicalBytesError) Error() string { return e.code + ": " + e.err.Error() }
func hbError(code, detail string) error      { return historicalBytesError{code, fmt.Errorf("%s", detail)} }
func hbWrap(code string, err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(historicalBytesError); ok {
		return err
	}
	return historicalBytesError{code, err}
}
func hbPlan(o HistoricalBytesOptions, mode string) *HistoricalBytesPlan {
	return &HistoricalBytesPlan{Schema: historicalBytesPlanSchema, Mode: mode, Status: "BLOCKED", Locale: o.Locale, ContentPackage: o.ContentPackage, BatchID: hbString(o.BatchID), UnitID: hbString(o.UnitID), SourceVerification: "unknown", Blockers: []HistoricalBytesBlocker{}, Mutation: "none"}
}
func hbString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
func hbFail(p *HistoricalBytesPlan, err error) (*HistoricalBytesPlan, error) {
	code := "IO_FAILURE"
	if e, ok := err.(historicalBytesError); ok {
		code = e.code
	}
	p.Status = "BLOCKED"
	p.PlannedAuthority = nil
	p.AuthorityIdentity = nil
	p.AuthorityPath = nil
	p.CAS = nil
	p.Recorded = false
	p.Blockers = append(p.Blockers, HistoricalBytesBlocker{code, err.Error()})
	return p, err
}
func historicalBytesKey(d HistoricalBytesDeclaration) string { return sum(mustJSON(d)) }
func HistoricalConstructionBytesIdentity(s HistoricalBytesStatement) string {
	return sum(mustJSON(historicalBytesIdentityCore{HistoricalConstructionBytesSchema, HistoricalConstructionBytesCapability, s, ""}))
}
func historicalBytesBlobPath(sha string) string {
	return historicalBytesRoot + "/blobs/sha256/" + sha + ".bin"
}
func historicalBytesAuthorityDir(locale, id string) string {
	return historicalBytesRoot + "/authorities/" + locale + "/" + id
}
func historicalBytesIndexPath(locale, key string) string {
	return historicalBytesRoot + "/declarations/" + locale + "/" + key + ".json"
}
func historicalBytesPendingPath(locale, key string) string {
	return historicalBytesRoot + "/transactions/" + locale + "/" + key + ".pending"
}
func historicalBytesStagePath(locale, key string) string {
	return historicalBytesRoot + "/transactions/" + locale + "/" + key + ".stage"
}
func historicalBytesKind(s string) bool {
	return s == "codex-artifact" || s == "editor-history" || s == "tmp-copy" || s == "backup"
}

func validateHistoricalBytesAuthority(a *HistoricalConstructionBytesAuthority) error {
	d := a.Statement.declaration()
	i := d.Invocation
	if a.Schema != HistoricalConstructionBytesSchema || a.Capability != HistoricalConstructionBytesCapability || a.Identity != HistoricalConstructionBytesIdentity(a.Statement) || !validSHA256(a.Identity) || !historicalBytesKind(a.SourceKind) {
		return hbError("CORRUPT_RECORDED_EVIDENCE", "schema/capability/identity/acquisition kind mismatch")
	}
	if ValidateLocaleName(d.Locale) != nil || d.ContentPackage.ID == "" || d.ContentPackage.ID == tourNativeTranslationPackage || !validSHA256(d.ContentPackage.AuthoritySHA256) || !validSHA256(d.ContentPackage.ScopeIdentitySHA256) || validateBatchID(i.BatchID) != nil || !validSHA256(i.UnitID) || i.UnitKind != UnitKindPage || i.Attempt != 1 || !validSHA256(i.InputIdentitySHA256) || !validSHA256(i.SourceSHA256) || !validSHA256(i.InputSHA256) || (i.TaskKind != "translation-unit-revision" && i.TaskKind != TransportRecoveryTaskKind) {
		return hbError("WRONG_SCOPE", "incomplete historical invocation/package")
	}
	if err := requireFormalGenerationIdentity(i.Provider, i.Model, i.BatchID); err != nil {
		return hbWrap("WRONG_SCOPE", err)
	}
	base := "data/retranslation-runs/" + d.Locale + "/" + i.BatchID + "/"
	if i.Manifest.Path != base+"manifest.json" || i.ImportReceipt.Path != base+"generation-imports/initial-manifest.json" || d.Archive.Path != importedGenerationArchive(d.Locale, d.Archive.SHA256) || d.Declaration.Container != d.Archive || d.Proof.Container != d.Archive || d.Proof.Member.Path != approvedIntervalProofMember(i.UnitID) {
		return hbError("WRONG_SCOPE", "historical container/ref mismatch")
	}
	for _, r := range []RecoveryFileReference{i.Manifest, i.ImportReceipt, d.Archive, d.Proof.Member, d.Declaration.Member} {
		if !validRecoveryFileReference(r) {
			return hbError("UNSAFE_PATH", "invalid historical reference")
		}
	}
	if validateGenerationInstallPath(d.LogicalPath) != nil || !validSHA256(d.SHA256) || (d.Size != nil && (*d.Size < 0 || *d.Size > historicalBytesMaxSource)) {
		return hbError("NO_DECLARATION", "invalid declaration bytes")
	}
	if !historicalBytesRole(d.Producer, d.Role, d.LogicalPath) {
		return hbError("UNSUPPORTED_ROLE", "unsupported versioned implementation role")
	}
	if d.Producer.RoleOrigin == "versioned-reader" {
		if d.Producer.Version != 1 || d.Declaration.Selector != "approved-proof.dependencies" || d.Declaration.Member != d.Proof.Member || d.Size != nil {
			return hbError("UNSUPPORTED_PRODUCER", "legacy declaration cannot claim explicit fields")
		}
	} else if d.Producer.RoleOrigin != "explicit" || d.Declaration.Selector != "construction.producer.implementation" || d.Declaration.Member.Path != transportRecoveryConstructionMember || d.Size == nil {
		return hbError("UNSUPPORTED_PRODUCER", "explicit producer declaration mismatch")
	}
	for _, set := range []struct {
		refs []RecoveryDependency
		pkg  bool
	}{{d.ContentPackage.AuthorityReferences, true}, {i.LineageReferences, false}} {
		prev := ""
		seenPath := map[string]bool{}
		roles := map[string]int{}
		for _, r := range set.refs {
			key := r.Role + "\x00" + r.Ref.Path + "\x00" + r.Ref.SHA256
			if !validRecoveryFileReference(r.Ref) || key <= prev || seenPath[r.Role+"\x00"+r.Ref.Path] {
				return hbError("WRONG_SCOPE", "unsafe/unordered history references")
			}
			prev = key
			seenPath[r.Role+"\x00"+r.Ref.Path] = true
			roles[r.Role]++
			if set.pkg {
				if r.Role != "package-authority" {
					return hbError("WRONG_SCOPE", "unknown package reference role")
				}
			} else {
				switch r.Role {
				case "initial-raw", "predecessor-snapshot", "original-reopen", "representation-authority", "historical-provenance-authority":
				default:
					return hbError("WRONG_SCOPE", "unknown lineage role")
				}
			}
		}
		if !set.pkg {
			for _, role := range []string{"initial-raw", "predecessor-snapshot", "original-reopen", "representation-authority"} {
				if roles[role] != 1 {
					return hbError("WRONG_SCOPE", "missing/duplicate required historical lineage")
				}
			}
		}
	}
	st := a.Statement
	if st.RecoveredSHA256 != d.SHA256 || st.RecoveredSize < 0 || st.RecoveredSize > historicalBytesMaxSource || (d.Size != nil && st.RecoveredSize != *d.Size) || st.Blob != (RecoveryFileReference{historicalBytesBlobPath(d.SHA256), d.SHA256, st.RecoveredSize}) {
		return hbError("HASH_MISMATCH", "published bytes differ from historical declaration")
	}
	ac := a.Acquisition
	if ac.Method != "existing-whole-file-copy" || !historicalBytesAbsolutePath(ac.SourcePath) || !historicalBytesAbsolutePath(ac.EvidencePath) || ac.Source.SHA256 != d.SHA256 || ac.Source.Size != st.RecoveredSize || ac.Source.LinkCount != 1 || ac.Source.Mode&0170000 != 0100000 || !validSHA256(ac.EvidenceSHA256) || ac.EvidenceSize < 1 || ac.EvidenceSize > historicalBytesMaxEvidence || ac.Evidence != (RecoveryFileReference{historicalBytesAuthorityDir(d.Locale, a.Identity) + "/acquisition-evidence.bin", ac.EvidenceSHA256, ac.EvidenceSize}) {
		return hbError("INVALID_ACQUISITION", "invalid frozen acquisition audit")
	}
	return nil
}

func historicalBytesRole(p HistoricalBytesProducer, role, path string) bool {
	if p.ID != ApprovedIntervalTransportSchema || (p.Version != 1 && p.Version != 2) {
		return false
	}
	if role == "approved-construction-implementation" {
		_, ok := approvedConstructionDeclarationsV1[path]
		return ok
	}
	if role == "approved-construction-reader" && p.Version == 2 {
		for _, r := range transportRecoveryImplementationPaths {
			if r == path {
				return true
			}
		}
	}
	return false
}

func decodeHistoricalBytesAuthority(b []byte) (*HistoricalConstructionBytesAuthority, error) {
	if len(b) > historicalBytesMaxAuthority {
		return nil, hbError("SIZE_MISMATCH", "authority exceeds bound")
	}
	var a HistoricalConstructionBytesAuthority
	if err := historicalBytesStrictJSON(b, &a); err != nil {
		return nil, hbWrap("CORRUPT_RECORDED_EVIDENCE", err)
	}
	if err := validateHistoricalBytesAuthority(&a); err != nil {
		return nil, err
	}
	return &a, nil
}

func historicalBytesSameDeclaration(a, b HistoricalBytesDeclaration) bool {
	return bytes.Equal(mustJSON(a), mustJSON(b))
}

func historicalBytesSortedRefs(in []RecoveryDependency) ([]RecoveryDependency, error) {
	out, err := canonicalRecoveryDependencySet(in)
	if out == nil {
		out = []RecoveryDependency{}
	}
	if err != nil {
		return nil, err
	}
	for _, r := range out {
		if strings.TrimSpace(r.Role) != r.Role {
			return nil, hbError("WRONG_SCOPE", "invalid role")
		}
	}
	return out, nil
}

// This family's closed wire schema requires every field, including the nullable
// legacy size. The general historical reader rejects null *int64; keep this
// version-owned shape reader local rather than changing other protocol readers.
func historicalBytesStrictJSON(b []byte, out any) error {
	if !utf8.Valid(b) {
		return fmt.Errorf("invalid JSON UTF-8")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	var walk func(reflect.Type) error
	walk = func(typ reflect.Type) error {
		nullable := typ.Kind() == reflect.Pointer
		for typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		token, err := d.Token()
		if err != nil {
			return err
		}
		if token == nil {
			if nullable {
				return nil
			}
			return fmt.Errorf("null required field")
		}
		if typ.Kind() == reflect.Struct {
			if token != json.Delim('{') {
				return fmt.Errorf("object required")
			}
			fields := map[string]reflect.Type{}
			for i := 0; i < typ.NumField(); i++ {
				f := typ.Field(i)
				if f.PkgPath != "" {
					continue
				}
				name := strings.Split(f.Tag.Get("json"), ",")[0]
				if name == "-" {
					continue
				}
				if name == "" {
					name = f.Name
				}
				fields[name] = f.Type
			}
			seen := map[string]bool{}
			for d.More() {
				k, e := d.Token()
				if e != nil {
					return e
				}
				name, ok := k.(string)
				child, exists := fields[name]
				if !ok || seen[name] || !exists {
					return fmt.Errorf("unknown/duplicate JSON member %v", k)
				}
				seen[name] = true
				if e = walk(child); e != nil {
					return e
				}
			}
			if len(seen) != len(fields) {
				return fmt.Errorf("missing required JSON field in %s", typ.Name())
			}
			_, err = d.Token()
			return err
		}
		if typ.Kind() == reflect.Slice && typ.Elem().Kind() != reflect.Uint8 {
			if token != json.Delim('[') {
				return fmt.Errorf("array required")
			}
			for d.More() {
				if err = walk(typ.Elem()); err != nil {
					return err
				}
			}
			_, err = d.Token()
			return err
		}
		if _, delim := token.(json.Delim); delim {
			return fmt.Errorf("scalar required")
		}
		return nil
	}
	typ := reflect.TypeOf(out)
	if typ == nil || typ.Kind() != reflect.Pointer {
		return fmt.Errorf("typed destination required")
	}
	if err := walk(typ.Elem()); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("trailing JSON")
	}
	d = json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	return d.Decode(out)
}
