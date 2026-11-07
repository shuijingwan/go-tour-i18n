package i18n

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

type historicalBytesGraph struct {
	Declaration HistoricalBytesDeclaration
	Files       map[string][]byte // Only authenticated original ZIP members.
	Observed    []RecoveryDependency
}

// Authenticate the declaration without deriving the construction whose missing
// material is the very reason for adoption. Never call the pending exporter or
// a current language re-assessment from this historical-only boundary.
func loadHistoricalBytesGraph(root string, c *Catalog, o HistoricalBytesOptions) (*historicalBytesGraph, error) {
	if ValidateLocaleName(o.Locale) != nil || validateBatchID(o.BatchID) != nil || !validSHA256(o.UnitID) || validateGenerationInstallPath(o.LogicalPath) != nil {
		return nil, hbError("WRONG_SCOPE", "explicit safe locale/batch/Unit/logical path required")
	}
	selected, err := recoveryPackageCatalog(c, o.ContentPackage)
	if err != nil {
		return nil, hbWrap("WRONG_SCOPE", err)
	}
	g := &historicalBytesGraph{}
	observe := func(role string, r GlossaryArchiveReference) (RecoveryFileReference, []byte, error) {
		ref, b, e := recoveryDiskReference(root, r.Path)
		if e != nil {
			return ref, nil, e
		}
		if r.SHA256 != "" && ref.SHA256 != r.SHA256 {
			return ref, nil, fmt.Errorf("historical reference SHA mismatch: %s", r.Path)
		}
		g.Observed = append(g.Observed, RecoveryDependency{role, ref})
		return ref, b, nil
	}
	base := "data/retranslation-runs/" + o.Locale + "/" + o.BatchID + "/"
	if err = ensureNoGenerationImportPending(filepath.Join(root, base)); err != nil {
		return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", err)
	}
	mr, mb, err := observe("historical-batch", GlossaryArchiveReference{Path: base + "manifest.json"})
	if err != nil {
		return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", err)
	}
	m, err := readRetranslationProcessManifest(filepath.Join(root, base), o.Locale, o.BatchID)
	if err != nil {
		return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", err)
	}
	var record *RetranslationBatchUnit
	for i := range m.Units {
		if m.Units[i].UnitID == o.UnitID {
			record = &m.Units[i]
		}
	}
	if record == nil || record.UnitKind != UnitKindPage || record.RevisionFeedbackSource != CurrentValidationRevisionFeedback {
		return nil, hbError("WRONG_SCOPE", "Unit is not an approved current-validation revision member")
	}
	ir, ib, err := observe("historical-import", GlossaryArchiveReference{Path: base + "generation-imports/initial-manifest.json"})
	if err != nil {
		return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", err)
	}
	var receipt GenerationResultBundleManifest
	if err = contentidentity.StrictJSON(ib, &receipt); err != nil {
		return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", err)
	}
	if !validSHA256(receipt.GenerationBundleSHA256) {
		return nil, hbError("UNVERIFIED_HISTORICAL_GRAPH", "invalid archive digest")
	}
	ar, ab, err := observe("historical-archive", GlossaryArchiveReference{importedGenerationArchive(o.Locale, receipt.GenerationBundleSHA256), receipt.GenerationBundleSHA256})
	if err != nil {
		return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", err)
	}
	gm, files, err := readGenerationBundle(ab)
	if err != nil {
		return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", err)
	}
	g.Files = files
	if err = checkImportedGenerationIdentity(receipt, gm, ab, files); err != nil {
		return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", err)
	}
	if gm.Locale != o.Locale || gm.BatchID != o.BatchID || gm.Attempt != 1 || (gm.TaskKind != "translation-unit-revision" && gm.TaskKind != TransportRecoveryTaskKind) || !bytes.Equal(files["formal/manifest.json"], mb) {
		return nil, hbError("UNVERIFIED_HISTORICAL_GRAPH", "initial invocation/manifest mismatch")
	}
	if err = verifyRecoveryHistoricalBundleScope(m, gm, "", 1); err != nil {
		return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", err)
	}
	selection, err := parseApprovedIntervalSelectionBytes(root, files[approvedIntervalSelectionMember], selected, m)
	if err != nil || selection.entry(o.UnitID) == nil {
		return nil, hbError("NO_DECLARATION", fmt.Sprintf("no exact approved selection: %v", err))
	}
	outputs := map[string]TransportBundleFile{}
	for _, r := range receipt.Outputs {
		if _, ok := outputs[r.BundlePath]; ok {
			return nil, hbError("UNVERIFIED_HISTORICAL_GRAPH", "duplicate imported output")
		}
		outputs[r.BundlePath] = r
	}
	var unitRaw RecoveryFileReference
	for _, out := range gm.ExpectedOutputs {
		r, b, e := observe("historical-initial-raw", GlossaryArchiveReference{Path: base + out.InstallPath})
		if e != nil {
			return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", e)
		}
		if outputs[out.BundlePath] != NewTransportBundleFile(out.BundlePath, "", b) {
			return nil, hbError("UNVERIFIED_HISTORICAL_GRAPH", "initial raw/import output mismatch")
		}
		if out.UnitID == o.UnitID {
			unitRaw = r
		}
	}
	p, err := recoveryProofFromFiles(files, o.UnitID)
	if err != nil || p == nil {
		return nil, hbError("NO_DECLARATION", fmt.Sprintf("no strict frozen approved proof: %v", err))
	}
	if !reflect.DeepEqual(p.Record, *record) || p.SelectionSHA256 != sum(selection.bytes) || p.Projection.UnitID != o.UnitID || p.Projection.SourceSHA256 != record.SourceSHA256 || p.Reopen.Path != record.PreviousReviewPath || p.Reopen.SHA256 != record.PreviousReviewSHA256 {
		return nil, hbError("WRONG_SCOPE", "frozen proof/selection/revision mismatch")
	}
	rr, rb, err := observe("historical-reopen", p.Reopen)
	if err != nil {
		return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", err)
	}
	var reopen QualityCheckSurfaceReopen
	if err = contentidentity.StrictJSON(rb, &reopen); err != nil {
		return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", err)
	}
	if reopen.Locale != o.Locale || reopen.ReopenID != record.RevisionAuthorizationID || reopen.PreviousSnapshotID != record.PreviousSnapshotID || reopen.CurrentValidation == nil {
		return nil, hbError("WRONG_SCOPE", "historical reopen binding mismatch")
	}
	assessment, err := decodeCurrentValidationAssessment(reopen.CurrentValidation)
	if err != nil {
		return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", err)
	}
	if assessment.Package != o.ContentPackage {
		return nil, hbError("WRONG_SCOPE", "historical package selector mismatch")
	}
	h, err := HistoricalTranslationPackageCatalog(selected, assessment.Package, assessment.PackageAuthoritySHA)
	if err != nil {
		return nil, hbWrap("WRONG_SCOPE", err)
	}
	_, authority, scope, _, err := translationAuthorizationScope(h)
	if err != nil || authority != assessment.PackageAuthoritySHA || scope != assessment.ScopeIdentity {
		return nil, hbError("WRONG_SCOPE", "historical complete descriptor mismatch")
	}
	inventory := *selected
	inventory.TranslationScope = nil
	snapshotCatalog, _, err := selectSnapshotTranslationPackage(root, &inventory, o.Locale, reopen.PreviousSnapshotID)
	if err != nil {
		return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", err)
	}
	if err = verifyCurrentValidationReopen(root, snapshotCatalog, &reopen, false); err != nil {
		return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", err)
	}
	found := false
	for _, u := range reopen.Units {
		if u.UnitID == o.UnitID && qualityCheckSnapshotIdentityMatches(u.Snapshot, p.Snapshot) && u.Finding == record.PreviousFinding {
			found = true
		}
	}
	if !found {
		return nil, hbError("WRONG_SCOPE", "approved proof has no exact original finding/Snapshot")
	}
	sr, _, err := observe("historical-snapshot", reopen.CurrentValidation.Snapshot)
	if err != nil {
		return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", err)
	}
	sem, _, err := observe("historical-representation", p.SemanticAuthority)
	if err != nil {
		return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", err)
	}
	lineage := []RecoveryDependency{{"initial-raw", unitRaw}, {"predecessor-snapshot", sr}, {"original-reopen", rr}, {"representation-authority", sem}}
	for _, ref := range assessment.Historical.References {
		if _, _, err = observe("historical-original-A", ref); err != nil {
			return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", err)
		}
	}
	if p.LegacyReference != nil {
		v, e := VerifyLegacyGenerationProvenance(root, h, *p.LegacyReference, p.RequestedOptions)
		if e != nil {
			return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", e)
		}
		if p.HistoricalVerification == nil || v.Authority.Identity != p.HistoricalVerification.Authority.Identity {
			return nil, hbError("UNVERIFIED_HISTORICAL_GRAPH", "historical legacy authority mismatch")
		}
		ref, _, e := observe("historical-legacy", *p.LegacyReference)
		if e != nil {
			return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", e)
		}
		lineage = append(lineage, RecoveryDependency{"historical-provenance-authority", ref})
	} else if p.ProvenanceSource != "normal-archive" {
		return nil, hbError("UNVERIFIED_HISTORICAL_GRAPH", "unknown response provenance")
	}
	u, err := h.Unit(o.UnitID)
	if err != nil || u.SourceSHA256 != record.SourceSHA256 {
		return nil, hbError("WRONG_SCOPE", "frozen Unit/source does not belong to historical descriptor")
	}
	projection, decoder, err := typedApprovedConstruction(h, u, p)
	if err != nil {
		return nil, hbWrap("UNSUPPORTED_PRODUCER", err)
	}
	if projection == nil || decoder == nil {
		return nil, hbError("UNSUPPORTED_PRODUCER", "missing known inverse reader")
	}
	reviewed := files["formal/candidates/"+filepath.Base(record.InputPath)]
	restored, err := decoder.Restore(p.Projection.Text)
	if err != nil || sum(reviewed) != p.Snapshot.CandidateSHA256 || !bytes.Equal(canonicalizeRetranslationArtifactEOF(restored), canonicalizeRetranslationArtifactEOF(reviewed)) || !bytes.Equal(files["formal/protected-candidates/"+filepath.Base(record.InputPath)], []byte(p.Projection.Text)) {
		return nil, hbError("UNVERIFIED_HISTORICAL_GRAPH", "frozen approved inverse/reviewed candidate mismatch")
	}
	if sum(files["formal/"+record.InputPath]) != record.InputSHA256 {
		return nil, hbError("UNVERIFIED_HISTORICAL_GRAPH", "formal source input mismatch")
	}
	if m.TransportRecovery != nil {
		if err = verifyRecoveryInstalledExecution(root, h, m); err != nil {
			return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", err)
		}
	}
	d := HistoricalBytesDeclaration{Locale: o.Locale, ContentPackage: HistoricalBytesPackage{ID: o.ContentPackage, AuthoritySHA256: authority, ScopeIdentitySHA256: scope, AuthorityReferences: []RecoveryDependency{}}, Invocation: HistoricalBytesInvocation{BatchID: o.BatchID, TaskKind: gm.TaskKind, Attempt: 1, Manifest: mr, ImportReceipt: ir, InputIdentitySHA256: gm.InputIdentitySHA256, Provider: receipt.Provider, Model: receipt.Model, UnitID: o.UnitID, UnitKind: record.UnitKind, SourceSHA256: record.SourceSHA256, InputSHA256: record.InputSHA256}, Archive: ar, Proof: HistoricalBytesMember{ar, recoveryMemberReference(approvedIntervalProofMember(o.UnitID), files[approvedIntervalProofMember(o.UnitID)])}, LogicalPath: o.LogicalPath}
	d.Invocation.LineageReferences, err = historicalBytesSortedRefs(lineage)
	if err != nil {
		return nil, err
	}
	// Membership hydration is current registration evidence. Only immutable refs
	// already corroborated by this archive enter the old declaration's identity.
	for _, desc := range h.TranslationPackages {
		if desc.ID == d.ContentPackage.ID && desc.AuthoritySHA256 == authority && desc.Membership != nil {
			for _, ref := range desc.Membership.References {
				r, b, e := observe("historical-package-registration", ref)
				if e != nil {
					return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", e)
				}
				if _, e = recoveryArchiveReference(files, GlossaryArchiveReference{ref.Path, sum(b)}); e == nil {
					d.ContentPackage.AuthorityReferences = append(d.ContentPackage.AuthorityReferences, RecoveryDependency{"package-authority", r})
				}
			}
		}
	}
	d.ContentPackage.AuthorityReferences, err = historicalBytesSortedRefs(d.ContentPackage.AuthorityReferences)
	if err != nil {
		return nil, err
	}
	if cb := files[transportRecoveryConstructionMember]; len(cb) > 0 {
		construction, e := decodeDerivedTransportConstruction(h, cb)
		if e != nil {
			return nil, hbWrap("UNSUPPORTED_PRODUCER", e)
		}
		if construction.SemanticContext.Locale != o.Locale || construction.SemanticContext.Package.ID != o.ContentPackage {
			return nil, hbError("WRONG_SCOPE", "construction scope mismatch")
		}
		d.Producer = HistoricalBytesProducer{construction.Producer.ID, construction.Producer.Version, "explicit"}
		seen := map[string]bool{}
		matches := 0
		for _, dep := range construction.Producer.Implementation {
			if seen[dep.Ref.Path] || !validRecoveryFileReference(dep.Ref) || !historicalBytesRole(d.Producer, dep.Role, dep.Ref.Path) {
				return nil, hbError("UNSUPPORTED_PRODUCER", "invalid/duplicate frozen producer inventory")
			}
			seen[dep.Ref.Path] = true
			if dep.Ref.Path == o.LogicalPath {
				d.SHA256 = dep.Ref.SHA256
				size := dep.Ref.Size
				d.Size = &size
				d.Role = dep.Role
				matches++
			}
		}
		want := len(approvedConstructionDeclarationsV1)
		if construction.Producer.Version == 2 {
			want += len(transportRecoveryImplementationPaths)
		}
		if len(seen) != want {
			return nil, hbError("UNSUPPORTED_PRODUCER", "incomplete explicit producer inventory")
		}
		if matches != 1 {
			return nil, hbError("NO_DECLARATION", "logical dependency not in frozen producer")
		}
		d.Declaration = HistoricalBytesDeclarationRef{ar, recoveryMemberReference(transportRecoveryConstructionMember, cb), "construction.producer.implementation"}
	} else {
		d.Producer = HistoricalBytesProducer{ApprovedIntervalTransportSchema, 1, "versioned-reader"}
		if !historicalBytesRole(d.Producer, "approved-construction-implementation", o.LogicalPath) {
			return nil, hbError("UNSUPPORTED_ROLE", "logical dependency is not a legacy producer implementation")
		}
		d.Role = "approved-construction-implementation"
		d.Declaration = HistoricalBytesDeclarationRef{ar, d.Proof.Member, "approved-proof.dependencies"}
	}
	seen := map[string]string{}
	known := map[string]bool{}
	matches := 0
	for _, ref := range p.Dependencies {
		if validateGenerationInstallPath(ref.Path) != nil || !validSHA256(ref.SHA256) {
			return nil, hbError("NO_DECLARATION", "invalid approved dependency")
		}
		if prior, ok := seen[ref.Path]; ok {
			if prior != ref.SHA256 {
				return nil, hbError("NO_DECLARATION", "conflicting approved dependency")
			}
			if _, impl := approvedConstructionDeclarationsV1[ref.Path]; impl {
				return nil, hbError("NO_DECLARATION", "duplicate producer declaration")
			}
			continue
		}
		seen[ref.Path] = ref.SHA256
		if _, ok := approvedConstructionDeclarationsV1[ref.Path]; ok {
			known[ref.Path] = true
		}
		if ref.Path == o.LogicalPath {
			matches++
			if d.SHA256 != "" && d.SHA256 != ref.SHA256 {
				return nil, hbError("NO_DECLARATION", "proof/construction declaration conflict")
			}
			d.SHA256 = ref.SHA256
		}
	}
	if len(known) != len(approvedConstructionDeclarationsV1) {
		return nil, hbError("UNSUPPORTED_PRODUCER", "incomplete versioned producer declaration")
	}
	if (d.Role == "approved-construction-implementation" && matches != 1) || (d.Role == "approved-construction-reader" && matches != 0) || !validSHA256(d.SHA256) {
		return nil, hbError("NO_DECLARATION", "no unique formal logical dependency declaration")
	}
	g.Declaration = d
	g.Observed, err = canonicalRecoveryDependencySet(g.Observed)
	if err != nil {
		return nil, err
	}
	refs := []RecoveryFileReference{}
	for _, r := range g.Observed {
		refs = append(refs, r.Ref)
	}
	if err = verifyRecoveryByteReferences(root, refs); err != nil {
		return nil, hbWrap("UNVERIFIED_HISTORICAL_GRAPH", err)
	}
	return g, nil
}
