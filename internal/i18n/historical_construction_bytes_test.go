package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func historicalBytesTestAuthority(t *testing.T) (*HistoricalConstructionBytesAuthority, []byte, []byte) {
	t.Helper()
	blob := []byte("package historical\n// exact binary material\x00\xff\n")
	evidence := []byte("synthetic acquisition audit\n")
	locale := "en-GB"
	batch := "chatgpt-en-GB-001"
	uid := sum([]byte("unit"))
	base := "data/retranslation-runs/" + locale + "/" + batch + "/"
	archive := recoveryMemberReference(importedGenerationArchive(locale, sum([]byte("archive"))), []byte("archive"))
	proof := recoveryMemberReference(approvedIntervalProofMember(uid), []byte("proof"))
	d := HistoricalBytesDeclaration{Locale: locale, ContentPackage: HistoricalBytesPackage{ID: "learn-docs-v1", AuthoritySHA256: sum([]byte("package")), ScopeIdentitySHA256: sum([]byte("scope")), AuthorityReferences: []RecoveryDependency{}}, Invocation: HistoricalBytesInvocation{BatchID: batch, TaskKind: "translation-unit-revision", Attempt: 1, Manifest: recoveryMemberReference(base+"manifest.json", []byte("manifest")), ImportReceipt: recoveryMemberReference(base+"generation-imports/initial-manifest.json", []byte("receipt")), InputIdentitySHA256: sum([]byte("input-identity")), Provider: "chatgpt", Model: FormalChatGPTGenerationModel, UnitID: uid, UnitKind: UnitKindPage, SourceSHA256: sum([]byte("source")), InputSHA256: sum([]byte("input")), LineageReferences: []RecoveryDependency{}}, Archive: archive, Proof: HistoricalBytesMember{archive, proof}, LogicalPath: "internal/i18n/approved_interval_transport.go", SHA256: sum(blob), Role: "approved-construction-implementation", Producer: HistoricalBytesProducer{ApprovedIntervalTransportSchema, 1, "versioned-reader"}, Declaration: HistoricalBytesDeclarationRef{archive, proof, "approved-proof.dependencies"}}
	d.Invocation.LineageReferences = []RecoveryDependency{{"initial-raw", recoveryMemberReference(base+"raw-responses/"+uid+".article", []byte("raw"))}, {"original-reopen", recoveryMemberReference("data/quality-check-surface-reopens/en-GB/example.json", []byte("reopen"))}, {"predecessor-snapshot", recoveryMemberReference("data/quality-check-snapshots/en-GB/old/manifest.json", []byte("snapshot"))}, {"representation-authority", recoveryMemberReference("data/representation/authority.json", []byte("representation"))}}
	st := historicalBytesStatement(d, sum(blob), int64(len(blob)), recoveryMemberReference(historicalBytesBlobPath(sum(blob)), blob))
	id := HistoricalConstructionBytesIdentity(st)
	a := &HistoricalConstructionBytesAuthority{Schema: HistoricalConstructionBytesSchema, Capability: HistoricalConstructionBytesCapability, Statement: st, Identity: id, SourceKind: "codex-artifact", Acquisition: HistoricalBytesAcquisition{Method: "existing-whole-file-copy", SourcePath: "/tmp/historical-source.go", Source: HistoricalBytesObservation{SHA256: sum(blob), Size: int64(len(blob)), Device: 1, Inode: 123, Mode: 0100644, LinkCount: 1, MtimeNS: 12, CtimeNS: 13}, EvidencePath: "/tmp/acquisition.json", Evidence: recoveryMemberReference(historicalBytesAuthorityDir(locale, id)+"/acquisition-evidence.bin", evidence), EvidenceSHA256: sum(evidence), EvidenceSize: int64(len(evidence))}}
	if err := validateHistoricalBytesAuthority(a); err != nil {
		t.Fatal(err)
	}
	return a, blob, evidence
}

func TestHistoricalConstructionBytesStrictIdentity(t *testing.T) {
	a, _, _ := historicalBytesTestAuthority(t)
	before := mustJSON(a)
	id := a.Identity
	key := historicalBytesKey(a.Statement.declaration())
	a.SourceKind = "backup"
	a.Acquisition.SourcePath = "/tmp/different-codex-session/source.go"
	a.Acquisition.EvidencePath = "/tmp/different-evidence"
	a.Acquisition.Source.Inode++
	a.Acquisition.Source.MtimeNS++
	if HistoricalConstructionBytesIdentity(a.Statement) != id || historicalBytesKey(a.Statement.declaration()) != key || sum(before) == sum(mustJSON(a)) {
		t.Fatal("audit metadata entered identity or lost physical binding")
	}
	if _, err := decodeHistoricalBytesAuthority(mustJSON(a)); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		edit func(*HistoricalConstructionBytesAuthority)
	}{
		{"schema", func(a *HistoricalConstructionBytesAuthority) { a.Schema += "-unknown" }},
		{"capability", func(a *HistoricalConstructionBytesAuthority) { a.Capability = "arbitrary-injection" }},
		{"producer-version", func(a *HistoricalConstructionBytesAuthority) { a.Statement.Producer.Version = 99 }},
		{"producer-role", func(a *HistoricalConstructionBytesAuthority) { a.Statement.Role = "arbitrary-source" }},
		{"wrong-logical-path", func(a *HistoricalConstructionBytesAuthority) {
			a.Statement.LogicalPath = "internal/i18n/unregistered.go"
		}},
		{"wrong-archive-lineage", func(a *HistoricalConstructionBytesAuthority) {
			a.Statement.Archive.Path = "data/generation-archives/other.zip"
		}},
		{"wrong-unit", func(a *HistoricalConstructionBytesAuthority) {
			a.Statement.Invocation.UnitID = sum([]byte("other-unit"))
		}},
		{"wrong-batch", func(a *HistoricalConstructionBytesAuthority) { a.Statement.Invocation.BatchID = "chatgpt-en-GB-002" }},
		{"missing-lineage", func(a *HistoricalConstructionBytesAuthority) {
			a.Statement.Invocation.LineageReferences = []RecoveryDependency{}
		}},
		{"wrong-size", func(a *HistoricalConstructionBytesAuthority) { n := int64(2); a.Statement.Size = &n }},
		{"wrong-old-sha", func(a *HistoricalConstructionBytesAuthority) { a.Statement.SHA256 = sum([]byte("other")) }},
		{"traversal", func(a *HistoricalConstructionBytesAuthority) { a.Statement.LogicalPath = "../secret" }},
		{"absolute-formal-ref", func(a *HistoricalConstructionBytesAuthority) { a.Statement.Invocation.Manifest.Path = "/tmp/manifest" }},
		{"unknown-source-kind", func(a *HistoricalConstructionBytesAuthority) { a.SourceKind = "reconstructed-equivalent" }},
		{"audit-hardlink", func(a *HistoricalConstructionBytesAuthority) { a.Acquisition.Source.LinkCount = 2 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var c HistoricalConstructionBytesAuthority
			_ = json.Unmarshal(before, &c)
			tc.edit(&c)
			c.Identity = HistoricalConstructionBytesIdentity(c.Statement)
			if _, err := decodeHistoricalBytesAuthority(mustJSON(c)); err == nil {
				t.Fatal("invalid authority accepted")
			}
		})
	}
	unknown := bytes.Replace(before, []byte(`"schema":`), []byte(`"undeclared":true,"schema":`), 1)
	if _, err := decodeHistoricalBytesAuthority(unknown); err == nil {
		t.Fatal("unknown authority field accepted")
	}
	var wire map[string]any
	_ = json.Unmarshal(before, &wire)
	statement := wire["statement"].(map[string]any)
	delete(statement, "declared_historical_size")
	if _, err := decodeHistoricalBytesAuthority(mustJSON(wire)); err == nil {
		t.Fatal("missing required nullable declaration accepted")
	}
	duplicate := bytes.Replace(before, []byte(`"schema":`), []byte(`"schema":"duplicate","schema":`), 1)
	if _, err := decodeHistoricalBytesAuthority(duplicate); err == nil {
		t.Fatal("duplicate field accepted")
	}

}

func TestHistoricalConstructionBytesExternalBoundary(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	p := filepath.Join(external, "exact.bin")
	b := []byte{0, 255, 13, 10, 0, 10}
	if err := os.WriteFile(p, b, 0600); err != nil {
		t.Fatal(err)
	}
	got, obs, err := readHistoricalBytesExternal(root, p, 100)
	if err != nil || !bytes.Equal(got, b) || obs.SHA256 != sum(b) || obs.Size != 6 {
		t.Fatal("raw byte round trip", err)
	}
	if _, _, err = readHistoricalBytesExternal(root, p, 5); err == nil {
		t.Fatal("source bound ignored")
	}
	link := filepath.Join(external, "symlink")
	if err = os.Symlink(p, link); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Join(external, "alias")
	if err = os.Symlink(external, parent); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(external, "fifo")
	if err = syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	inside := filepath.Join(root, "current.go")
	if err = os.WriteFile(inside, b, 0644); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{link, filepath.Join(parent, "exact.bin"), external, fifo, inside, filepath.Join(external, "missing"), "relative.go", external + "/../exact.bin", external + "/bad\npath"} {
		if _, _, err := readHistoricalBytesExternal(root, path, 100); err == nil {
			t.Fatal("invalid acquisition accepted", path)
		}
	}
	hard := filepath.Join(external, "hard")
	if err = os.Link(p, hard); err != nil {
		t.Fatal(err)
	}
	if _, _, err = readHistoricalBytesExternal(root, p, 100); err == nil {
		t.Fatal("multiply linked source accepted")
	}
}

func TestHistoricalConstructionBytesResolverPriority(t *testing.T) {
	root := t.TempDir()
	a, blob, evidence := historicalBytesTestAuthority(t)
	d := a.Statement.declaration()
	g := &historicalBytesGraph{Declaration: d, Files: map[string][]byte{}}
	write := func(path string, b []byte) {
		t.Helper()
		if err := publishHistoricalBytesFile(root, path, b, false); err != nil {
			t.Fatal(err)
		}
	}
	b, origin, _, err := resolveHistoricalBytesMaterial(root, nil, g, false)
	if err != nil || b != nil || origin != "" {
		t.Fatal("missing declaration bytes must stay unavailable", err)
	}
	// A loose blob is never evidence for this declaration.
	write(a.Statement.Blob.Path, blob)
	b, _, _, err = resolveHistoricalBytesMaterial(root, nil, g, false)
	if err != nil || b != nil {
		t.Fatal("loose blob supplied authority", err)
	}
	write(a.Acquisition.Evidence.Path, evidence)
	write(historicalBytesAuthorityDir(d.Locale, a.Identity)+"/authority.json", mustJSON(a))
	write(historicalBytesIndexPath(d.Locale, historicalBytesKey(d)), historicalBytesExpectedIndexBytes(a))
	b, origin, refs, err := resolveHistoricalBytesMaterial(root, nil, g, false)
	if err != nil || !bytes.Equal(b, blob) || origin != "recorded-supplemental" || len(refs) != 4 {
		t.Fatal("supplemental exact declaration", err, origin)
	}
	recovery := &InstalledDerivedTransportRecovery{}
	for _, ref := range refs {
		recovery.CurrentEpoch = append(recovery.CurrentEpoch, RecoveryDependency{"historical-bytes-material-evidence", ref})
	}
	inputs, e := historicalBytesGenerationInputs(root, nil, recovery)
	if e != nil || len(inputs) != 4 {
		t.Fatal("supplemental future Bundle inventory", e, len(inputs))
	}
	names := map[string]bool{}
	for _, in := range inputs {
		names[in.path] = true
	}
	for _, name := range []string{"index.json", "authority.json", "bytes.bin", "acquisition-evidence.bin"} {
		if !names["formal/historical-construction-bytes/"+a.Identity+"/"+name] {
			t.Fatal("missing independent supplemental namespace", name)
		}
	}
	recovery.CurrentEpoch = recovery.CurrentEpoch[:3]
	if _, e = historicalBytesGenerationInputs(root, nil, recovery); e == nil {
		t.Fatal("incomplete epoch evidence silently frozen")
	}

	if err = os.MkdirAll(filepath.Dir(filepath.Join(root, d.LogicalPath)), 0755); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, d.LogicalPath), blob, 0644); err != nil {
		t.Fatal(err)
	}
	_, origin, _, err = resolveHistoricalBytesMaterial(root, nil, g, false)
	if err != nil || origin != "surviving-repository" {
		t.Fatal("surviving source priority", err, origin)
	}
	g.Files["old-archive-member"] = blob
	_, origin, _, err = resolveHistoricalBytesMaterial(root, nil, g, false)
	if err != nil || origin != "verified-archive" {
		t.Fatal("archive priority", err, origin)
	}
	// Another Unit cannot borrow a global content-addressed blob/authority.
	other := *g
	other.Files = map[string][]byte{}
	other.Declaration.Invocation.UnitID = sum([]byte("different-unit"))
	_ = os.WriteFile(filepath.Join(root, d.LogicalPath), []byte("current changed"), 0644)
	b, _, _, err = resolveHistoricalBytesMaterial(root, nil, &other, false)
	if err != nil || b != nil {
		t.Fatal("cross-Unit authority was borrowed", err)
	}
	// A known corrupt index must block even while the archive has exact bytes.
	ip := filepath.Join(root, historicalBytesIndexPath(d.Locale, historicalBytesKey(d)))
	_ = os.Chmod(ip, 0644)
	_ = os.WriteFile(ip, []byte("{}"), 0444)
	_ = os.Chmod(ip, 0444)
	if _, _, _, err = resolveHistoricalBytesMaterial(root, nil, g, false); err == nil {
		t.Fatal("higher priority masked corrupt recorded claim")
	}
	_ = os.Chmod(ip, 0644)
	_ = os.WriteFile(ip, historicalBytesExpectedIndexBytes(a), 0444)
	_ = os.Chmod(ip, 0444)
	bp := filepath.Join(root, a.Statement.Blob.Path)
	_ = os.Chmod(bp, 0644)
	_ = os.WriteFile(bp, bytes.Repeat([]byte{'x'}, len(blob)), 0444)
	_ = os.Chmod(bp, 0444)
	if _, _, _, err = resolveHistoricalBytesMaterial(root, nil, g, false); err == nil {
		t.Fatal("higher priority masked corrupt blob")
	}
}

func TestHistoricalConstructionBytesSealedOwner(t *testing.T) {
	a, blob, evidence := historicalBytesTestAuthority(t)
	o := HistoricalBytesOptions{Locale: a.Statement.Locale, ContentPackage: a.Statement.ContentPackage.ID, BatchID: a.Statement.Invocation.BatchID, UnitID: a.Statement.Invocation.UnitID, LogicalPath: a.Statement.LogicalPath, Source: a.Acquisition.SourcePath, SourceKind: a.SourceKind, Evidence: a.Acquisition.EvidencePath, EvidenceSHA256: sum(evidence)}
	var index historicalBytesIndex
	_ = json.Unmarshal(historicalBytesExpectedIndexBytes(a), &index)
	prep := &historicalBytesPrepared{Graph: &historicalBytesGraph{Declaration: a.Statement.declaration()}, Authority: a, Blob: blob, Evidence: evidence, Index: index, CAS: historicalBytesCAS{Options: o, Source: a.Acquisition.Source, Evidence: HistoricalBytesObservation{SHA256: sum(evidence), Size: int64(len(evidence))}}}
	prep.Token = sum(mustJSON(prep.CAS))
	tx := historicalBytesTransactionFrom(prep)
	if err := validateHistoricalBytesTransaction(&tx, o, a.Identity, prep.Token); err != nil {
		t.Fatal(err)
	}
	original := mustJSON(tx)
	for _, tc := range []struct {
		name string
		edit func(*historicalBytesTransaction)
	}{{"payload", func(tx *historicalBytesTransaction) { tx.Blob = []byte("reconstructed") }}, {"seal", func(tx *historicalBytesTransaction) { tx.Seal = strings.Repeat("0", 64) }}, {"owner", func(tx *historicalBytesTransaction) { tx.Owner = strings.Repeat("0", 64) }}, {"index", func(tx *historicalBytesTransaction) { tx.Index.DeclarationKey = strings.Repeat("0", 64) }}} {
		t.Run(tc.name, func(t *testing.T) {
			var changed historicalBytesTransaction
			_ = json.Unmarshal(original, &changed)
			tc.edit(&changed)
			if err := validateHistoricalBytesTransaction(&changed, o, a.Identity, prep.Token); err == nil {
				t.Fatal("unsafe sealed state accepted")
			}
		})
	}
	wrong := o
	wrong.Source = "/tmp/other-whole-file"
	if err := validateHistoricalBytesTransaction(&tx, wrong, a.Identity, prep.Token); err == nil {
		t.Fatal("resume changed owner flags")
	}
}

func TestHistoricalConstructionBytesUnknownStagePreserved(t *testing.T) {
	root := t.TempDir()
	stage := historicalBytesStagePath("en-GB", sum([]byte("declaration")))
	unknown := filepath.Join(root, stage, "unknown-empty-directory")
	if err := os.MkdirAll(unknown, 0700); err != nil {
		t.Fatal(err)
	}
	if err := cleanupHistoricalBytesStage(root, stage, &historicalBytesTransaction{}); err == nil {
		t.Fatal("unknown stage removed")
	}
	if _, err := os.Stat(unknown); err != nil {
		t.Fatal("unknown evidence deleted", err)
	}
}
