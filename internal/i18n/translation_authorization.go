package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

const CurrentTranslationAuthorizationSchema = "go-learning/current-translation-authorization/v1"

// Native Tour is the unscoped catalog's complete workflow, not a registered
// TranslationPackageDescriptor. Selectors must not manufacture a descriptor.
const tourNativeTranslationPackage = "tour-v1"

// TranslationReattestationAdapter extends representation behavior, not the
// lifecycle. Unknown representations cannot claim reliable semantic absence.
type TranslationReattestationAdapter interface {
	ReattestationAuthority(string) (string, error)
	GlossaryVisiblePair([]byte) (string, string, error)
}

type CurrentTranslationAssessmentUnit struct {
	Snapshot            QualityCheckSnapshotUnit `json:"accepted_snapshot_unit"`
	ReviewerGlossarySHA string                   `json:"reviewer_glossary_sha256"`
	CurrentSourceSHA    string                   `json:"current_source_sha256"`
	AuthoritySHA        string                   `json:"adapter_authority_sha256"`
	ProtectedInputSHA   string                   `json:"current_protected_input_sha256"`
	Classification      string                   `json:"classification"`
	Reasons             []string                 `json:"reasons"`
	Validation          string                   `json:"current_validation"`
	ValidationIdentity  string                   `json:"current_validation_identity_sha256"`
}

type CurrentTranslationAuthorization struct {
	Schema              string                             `json:"schema"`
	Locale              string                             `json:"locale"`
	Package             string                             `json:"package"`
	PackageAuthoritySHA string                             `json:"package_authority_sha256"`
	ScopeIdentity       string                             `json:"scope_identity_sha256"`
	Historical          HistoricalFinalizationIntegrity    `json:"historical_integrity"`
	Glossary            GlossaryArchiveReference           `json:"current_glossary"`
	UnifiedReview       GlossaryArchiveReference           `json:"current_unified_review"`
	Canonical           []GlossaryArchiveReference         `json:"current_canonical_targets,omitempty"`
	Algorithm           string                             `json:"algorithm"`
	Normalization       string                             `json:"normalization"`
	Units               []CurrentTranslationAssessmentUnit `json:"units"`
	Compatible          int                                `json:"compatible_count"`
	Affected            int                                `json:"affected_count"`
	Stale               int                                `json:"stale_count"`
	Unknown             int                                `json:"unknown_count"`
	Identity            string                             `json:"identity_sha256"`
}

func authorizationIdentity(a CurrentTranslationAuthorization) string {
	a.Identity = ""
	return sum(mustJSON(a))
}

func translationAuthorizationScope(c *Catalog) (string, string, string, []*TranslationUnit, error) {
	if c == nil {
		return "", "", "", nil, fmt.Errorf("translation catalog required")
	}
	ordered, _, _, err := localeWorkflowUnitList(c)
	if err != nil {
		return "", "", "", nil, err
	}
	pkg, authority := tourNativeTranslationPackage, ""
	if len(c.TranslationScope) > 0 {
		found := false
		for _, d := range c.TranslationPackages {
			if len(d.Units) != len(ordered) {
				continue
			}
			match := true
			for i, u := range d.Units {
				if u.Unit.ID != ordered[i].ID {
					match = false
				}
			}
			if match {
				if found {
					return "", "", "", nil, fmt.Errorf("ambiguous registered scope")
				}
				pkg, authority, found = d.ID, d.AuthoritySHA256, true
			}
		}
		if !found {
			return "", "", "", nil, fmt.Errorf("authorization requires complete registered package scope")
		}
	}
	// Stable routing/source coordinates locate the package's checkpoint; content
	// hashes and registered authority are independently checked inside it.
	paths := []string{pkg}
	for _, u := range ordered {
		paths = append(paths, string(u.Kind)+":"+u.SourcePath)
	}
	if authority == "" {
		identities := []string{pkg}
		for _, u := range ordered {
			identities = append(identities, u.ID+":"+string(u.Kind)+":"+u.SourcePath+":"+u.SourceSHA256)
		}
		authority = sum(mustJSON(identities))
	}
	return pkg, authority, sum(mustJSON(paths)), ordered, nil
}

// TranslationAdapterAuthority projects only this adapter's type and methods.
// Adding an unrelated package loader/helper in the same source file cannot
// change its language contract. Parser dependencies remain separately bound.
func TranslationAdapterAuthority(root, path, typeName string, dependencies []string) (string, error) {
	b, err := readCompatibilityFile(root, path)
	if err != nil {
		return "", err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, b, 0)
	if err != nil {
		return "", err
	}
	decls := map[string]string{}
	for _, decl := range file.Decls {
		var name string
		switch d := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if s, ok := spec.(*ast.TypeSpec); ok && s.Name.Name == typeName {
					var out bytes.Buffer
					if err := format.Node(&out, fset, s); err != nil {
						return "", err
					}
					decls["type:"+typeName] = out.String()
				}
			}
		case *ast.FuncDecl:
			if d.Recv != nil && len(d.Recv.List) == 1 {
				t := d.Recv.List[0].Type
				if p, ok := t.(*ast.StarExpr); ok {
					t = p.X
				}
				if id, ok := t.(*ast.Ident); ok && id.Name == typeName {
					name = d.Name.Name
				}
			}
			if name != "" {
				var out bytes.Buffer
				if err := format.Node(&out, fset, d); err != nil {
					return "", err
				}
				decls["method:"+name] = out.String()
			}
		}
	}
	if len(decls) < 2 {
		return "", fmt.Errorf("unknown adapter declarations")
	}
	deps, err := TranslationMachineAuthority(root, dependencies)
	if err != nil {
		return "", err
	}
	return sum(mustJSON(struct {
		Declarations map[string]string
		Dependencies string
	}{decls, deps})), nil
}

func currentTranslationAuthority(root string, u *TranslationUnit, locale string) (string, error) {
	if _, _, err := LoadLanguageAuditPolicy(root, locale, nil); err != nil {
		return "", err
	}
	_, metadata, err := CommentLocaleBaseLanguage(root, locale)
	if err != nil {
		return "", err
	}
	var representation string
	if a, ok := u.Adapter.(TranslationReattestationAdapter); ok {
		representation, err = a.ReattestationAuthority(root)
	} else if u.Adapter != nil {
		return "", fmt.Errorf("unknown re-attestation adapter authority")
	} else {
		representation, err = TranslationMachineAuthority(root, []string{"internal/i18n/validate.go", "internal/i18n/go_example_validate.go", "internal/i18n/go_example_protect.go", "internal/i18n/translate_protect.go", "internal/i18n/preformatted.go"})
	}
	if err != nil {
		return "", err
	}
	gate, err := TranslationMachineAuthority(root, commentGateAuthorityPaths)
	if err != nil {
		return "", err
	}
	return sum(mustJSON(struct {
		Representation, CommentGate string
		LocaleMetadata              GlossaryArchiveReference
	}{representation, gate, metadata})), nil
}

// TranslationMachineAuthority binds executable parser/protector/validator
// implementations plus Go dependency versions. No runtime/public config or
// unrelated package/global inventory enters this language authority.
func TranslationMachineAuthority(root string, paths []string) (string, error) {
	paths = append(append(append([]string(nil), paths...), LanguageAuditAuthorityPaths...), "go.mod", "go.sum", "internal/i18n/glossary.go", "internal/i18n/glossary_context.go", "internal/i18n/glossary_compatibility.go", "internal/i18n/glossary_semantics.go", "internal/i18n/translation_authorization.go", "internal/i18n/bridge_successor.go")
	refs := []GlossaryArchiveReference{}
	sort.Strings(paths)
	for _, p := range paths {
		b, err := readCompatibilityFile(root, p)
		if err != nil {
			return "", err
		}
		refs = append(refs, GlossaryArchiveReference{p, sum(b)})
	}
	return sum(mustJSON(refs)), nil
}

func currentTranslationVisiblePair(u *TranslationUnit, candidate []byte) (string, string, error) {
	if a, ok := u.Adapter.(TranslationReattestationAdapter); ok {
		return a.GlossaryVisiblePair(candidate)
	}
	if u.Adapter != nil {
		return "", "", fmt.Errorf("unknown visibility adapter")
	}
	parser := "present-visible/v1"
	if u.Kind == UnitKindExample {
		parser = "go-comment-visible/v1"
	}
	return compatibilityVisiblePair(GlossaryCompatibilityContext{Parser: parser, Source: string(u.Source), Target: string(candidate)})
}

func AssessCurrentTranslation(root string, c *Catalog, locale, snapshotID string) (*CurrentTranslationAuthorization, error) {
	if p, err := loadSealedTranslationHistory(root, c, locale, snapshotID); err != nil {
		return nil, err
	} else if p != nil {
		return assessCurrentTranslationProof(root, c, locale, *p)
	}
	p, err := VerifyHistoricalQualityCheckFinalization(root, locale, snapshotID, c)
	if err != nil {
		return nil, err
	}
	return assessCurrentTranslationProof(root, c, locale, *p)
}

func assessCurrentTranslationProof(root string, c *Catalog, locale string, p HistoricalFinalizationIntegrity) (*CurrentTranslationAuthorization, error) {
	return assessCurrentTranslationProofWithMachines(root, c, locale, p, map[string]*machineSelection{})
}

func assessCurrentTranslationProofWithMachines(root string, c *Catalog, locale string, p HistoricalFinalizationIntegrity, machines map[string]*machineSelection) (*CurrentTranslationAuthorization, error) {
	pkg, authority, scope, ordered, err := translationAuthorizationScope(c)
	if err != nil {
		return nil, err
	}
	if err := CheckPromotionTransactionState(root, locale); err != nil {
		return nil, err
	}
	review, err := RequireUnifiedGlossaryReview(root, locale)
	if err != nil {
		return nil, err
	}
	current, err := readCompatibilityFile(root, glossaryReviewGlossaryPath(locale))
	if err != nil {
		return nil, err
	}
	g, err := LoadGlossary(root, locale)
	if err != nil {
		return nil, err
	}
	a := &CurrentTranslationAuthorization{Schema: CurrentTranslationAuthorizationSchema, Locale: locale, Package: pkg, PackageAuthoritySHA: authority, ScopeIdentity: scope, Historical: p, Glossary: GlossaryArchiveReference{glossaryReviewGlossaryPath(locale), sum(current)}, UnifiedReview: review, Algorithm: glossaryImpactAlgorithmVersion, Normalization: glossaryNormalizationVersion, Units: []CurrentTranslationAssessmentUnit{}}
	if p.Finalization.Locale != locale || len(ordered) != len(p.Finalization.Units) || len(p.ReviewerGlossaries) != len(ordered) {
		return nil, fmt.Errorf("complete historical/current TU scope mismatch: package=%s current=%d historical=%d reviewer=%d snapshot=%s", pkg, len(ordered), len(p.Finalization.Units), len(p.ReviewerGlossaries), p.Finalization.SnapshotID)
	}
	for i, accepted := range p.Finalization.Units {
		x, err := assessCurrentTranslationUnit(root, c, locale, ordered[i], accepted.Snapshot, p.ReviewerGlossaries[i], current, g, machines)
		if err != nil {
			return nil, err
		}
		switch x.Classification {
		case "compatible":
			a.Compatible++
		case "affected":
			a.Affected++
		case "stale":
			a.Stale++
		case "unknown":
			a.Unknown++
		}
		a.Units = append(a.Units, x)
	}
	if len(machines) > 0 {
		if err := verifyMachineAssessmentInputs(root, c, a, ordered); err != nil {
			return nil, err
		}
	}
	a.Identity = authorizationIdentity(*a)
	return a, nil
}

// assessCurrentTranslationUnit is the common current-eligibility primitive.
// Callers must independently establish the accepted historical A integrity;
// this assessment never creates an A or authorizes a partial package scope.
func assessCurrentTranslationUnit(root string, c *Catalog, locale string, u *TranslationUnit, old QualityCheckSnapshotUnit, reviewerGlossarySHA string, current []byte, g *Glossary, machines map[string]*machineSelection) (CurrentTranslationAssessmentUnit, error) {
	if u.SourcePath != old.SourcePath || u.Kind != old.UnitKind {
		return CurrentTranslationAssessmentUnit{}, fmt.Errorf("scope reconciliation required at %d", old.Index)
	}
	x := CurrentTranslationAssessmentUnit{Snapshot: old, ReviewerGlossarySHA: reviewerGlossarySHA, CurrentSourceSHA: u.SourceSHA256, Classification: "compatible", Reasons: []string{}, Validation: "not_run"}
	candidate, err := readCompatibilityFile(root, old.CandidatePath)
	if err != nil || sum(candidate) != old.CandidateSHA256 {
		return CurrentTranslationAssessmentUnit{}, fmt.Errorf("accepted candidate identity drift: %s", old.UnitID)
	}
	if old.UnitID != u.ID || old.SourceSHA256 != u.SourceSHA256 || sum(u.Source) != u.SourceSHA256 {
		x.Classification = "stale"
		x.Reasons = []string{"source_identity_changed"}
	} else {
		// Physical raw source identity is representation-owned (Tour Pages
		// share an article); this is the same formal Catalog source checker.
		source, sourceErr := readCompatibilityFile(root, u.SourcePath)
		if sourceErr != nil || validateSnapshotSourceIdentity(c, u, source) != nil {
			x.Classification = "stale"
			x.Reasons = []string{"current_source_file_changed"}
		} else {
			x.AuthoritySHA, err = currentTranslationAuthority(root, u, locale)
			if err != nil || !validSHA256(x.AuthoritySHA) {
				x.Classification = "unknown"
				x.Reasons = []string{"unknown_adapter_authority"}
			} else {
				protected, inputErr := prepareTranslationUnitInput(u, g)
				if inputErr != nil {
					x.Classification = "unknown"
					x.Reasons = []string{"current_protector_failed"}
				} else {
					x.ProtectedInputSHA = sum([]byte(protected.Text))
				}
				validationErr := validateTranslationUnitWithGlossary(root, c, u, locale, candidate, g)
				if validationErr == nil {
					validationErr = verifyAcceptedFrozenInputWithMachines(root, locale, u, old, candidate, x.ReviewerGlossarySHA, machines, c)
				}
				x.Validation = "passed"
				if validationErr != nil {
					x.Validation = "failed"
					x.Classification = "affected"
					x.Reasons = append(x.Reasons, "current_validation_failed: "+validationErr.Error())
				}
				oldGlossary, oldErr := readArchivedGlossary(root, locale, x.ReviewerGlossarySHA)
				if oldErr != nil {
					return CurrentTranslationAssessmentUnit{}, oldErr
				}
				delta, deltaErr := glossarySemanticDelta(locale, oldGlossary, current)
				if deltaErr != nil {
					return CurrentTranslationAssessmentUnit{}, deltaErr
				}
				s, t, pairErr := currentTranslationVisiblePair(u, candidate)
				if pairErr != nil {
					x.Classification = "unknown"
					x.Reasons = append(x.Reasons, "unknown_visibility_parser")
				} else {
					compatible, reasons := classifyGlossaryContext(GlossaryCompatibilityContext{Parser: "plain-visible/v1", Source: s, Target: t}, delta)
					x.Reasons = append(x.Reasons, reasons...)
					if !compatible && x.Classification == "compatible" {
						x.Classification = "affected"
					}
				}
			}
		}
	}
	x.ValidationIdentity = sum(mustJSON(struct{ Source, Candidate, Glossary, Authority, Input, Status string }{x.CurrentSourceSHA, old.CandidateSHA256, sum(current), x.AuthoritySHA, x.ProtectedInputSHA, x.Validation}))
	return x, nil
}

// Context reuse shares preparation, never a Unit's PASS. Guard the frozen
// dependency graph and the current world again before publishing assessments.
func verifyMachineAssessmentInputs(root string, c *Catalog, a *CurrentTranslationAuthorization, ordered []*TranslationUnit) error {
	if err := checkFlatHistoricalIntegrity(root, c, &a.Historical); err != nil {
		return err
	}
	glossary, err := readCompatibilityFile(root, a.Glossary.Path)
	if err != nil || sum(glossary) != a.Glossary.SHA256 {
		return fmt.Errorf("machine assessment glossary changed during invocation")
	}
	review, err := RequireUnifiedGlossaryReview(root, a.Locale)
	if err != nil || review != a.UnifiedReview {
		return fmt.Errorf("machine assessment Unified Review changed during invocation: %v", err)
	}
	for i, x := range a.Units {
		if x.Validation != "passed" {
			continue
		}
		u := ordered[i]
		raw, err := readCompatibilityFile(root, u.SourcePath)
		if err != nil || validateSnapshotSourceIdentity(c, u, raw) != nil {
			return fmt.Errorf("machine assessment source changed during invocation: %s", u.ID)
		}
		authority, err := currentTranslationAuthority(root, u, a.Locale)
		if err != nil || authority != x.AuthoritySHA {
			return fmt.Errorf("machine assessment adapter authority changed during invocation: %s", u.ID)
		}
	}
	return nil
}

// This is an exact original-input/response relation, never a semantic
// compatibility chain. Only an unchanged source/candidate reaches this check.
func verifyAcceptedFrozenInput(root, locale string, u *TranslationUnit, s QualityCheckSnapshotUnit, candidate []byte, glossarySHA string, catalogs ...*Catalog) error {
	return verifyAcceptedFrozenInputWithMachines(root, locale, u, s, candidate, glossarySHA, nil, catalogs...)
}

func verifyAcceptedFrozenInputWithMachines(root, locale string, u *TranslationUnit, s QualityCheckSnapshotUnit, candidate []byte, glossarySHA string, machines map[string]*machineSelection, catalogs ...*Catalog) error {
	if s.RepresentationReconciliation != nil {
		if len(catalogs) == 0 || catalogs[0] == nil {
			return fmt.Errorf("explicit catalog required for reconciled frozen input")
		}
		// The shared context does not turn the receipt path into an oracle.
		if _, err := readMachineReceipt(root, *s.RepresentationReconciliation); err != nil {
			return err
		}
		// This proves the accepted receipt's original input/response relation.
		// Replaying the current legacy compatibility/successor resolver here
		// couples it back to the historical catalog's old Unit IDs. The caller
		// independently checks current source, candidate, full Review, semantic
		// delta and active validator before granting current TU eligibility.
		e, _, err := machineSnapshotEvidenceInWorld(root, catalogs[0], locale, s, machines, true)
		if err != nil {
			return err
		}
		if !bytes.Equal(e.candidate, candidate) || e.unit.ID != u.ID {
			return fmt.Errorf("reconciled accepted candidate mismatch")
		}
		return nil
	}
	base := filepath.Join(root, "data", "retranslation-runs", locale, s.SelectedBatchID)
	if u.ID != s.UnitID || u.Kind != s.UnitKind || u.SourceSHA256 != s.SourceSHA256 || sum(u.Source) != u.SourceSHA256 || sum(candidate) != s.CandidateSHA256 {
		return fmt.Errorf("original frozen Unit/source/candidate mismatch")
	}
	m, err := readRetranslationProcessManifest(base, locale, s.SelectedBatchID)
	if err != nil {
		return err
	}
	var r RetranslationBatchUnit
	for _, v := range m.Units {
		if v.UnitID == u.ID {
			r = v
		}
	}
	if r.UnitID != u.ID || r.UnitKind != u.Kind || r.SourcePath != u.SourcePath || r.SourceSHA256 != u.SourceSHA256 {
		return fmt.Errorf("original manifest Unit/source mismatch")
	}
	input, err := readCompatibilityFile(root, filepath.ToSlash(filepath.Join("data", "retranslation-runs", locale, s.SelectedBatchID, r.InputPath)))
	if err != nil || sum(input) != r.InputSHA256 {
		return fmt.Errorf("original protected input mismatch")
	}
	b, err := readCompatibilityFile(root, s.ValidationPath)
	if err != nil {
		return err
	}
	if sum(b) != s.ValidationSHA256 {
		return fmt.Errorf("original validation SHA mismatch")
	}
	// The existing schema-v1 Page boundary owns page_id normalization and
	// derives its attempt from the exact original raw-response path. Both
	// legacy and current formats remain strict; never decode v1 as v2.
	v, err := decodeRetranslationValidation(b, u)
	if err != nil {
		return err
	}
	expectedCandidate, err := filepath.Rel(filepath.Join("data", "retranslation-runs", locale, s.SelectedBatchID), filepath.FromSlash(s.CandidatePath))
	if err != nil || v.SchemaVersion != retranslationProcessSchemaVersion || v.BatchID != s.SelectedBatchID || v.Locale != locale || v.UnitID != u.ID || v.UnitKind != u.Kind || v.SourceSHA256 != u.SourceSHA256 || v.Attempt != s.Attempt || v.Status != "passed" || v.Error != "" || v.InputPath != r.InputPath || v.CandidatePath != filepath.ToSlash(expectedCandidate) {
		return fmt.Errorf("original validation Unit/source/attempt/provenance mismatch")
	}
	raw, err := readCompatibilityFile(root, filepath.ToSlash(filepath.Join("data", "retranslation-runs", locale, s.SelectedBatchID, v.RawResponsePath)))
	if err != nil {
		return err
	}
	if r.GenerationOrigin != nil && v.Attempt == 1 {
		a, ok := u.Adapter.(FrozenGenerationAdapter)
		if !ok {
			return fmt.Errorf("unknown frozen generation adapter")
		}
		if err := a.VerifyFrozenGenerationOrigin(root, locale, *r.GenerationOrigin, raw); err != nil {
			return err
		}
	}
	old, err := readArchivedGlossary(root, locale, glossarySHA)
	if err != nil {
		return err
	}
	entries, err := normalizeCompatibilityGlossary(locale, old)
	if err != nil {
		return err
	}
	g := &Glossary{Locale: locale, Mandatory: map[string]string{}, Preferred: map[string]string{}}
	for _, e := range entries {
		switch e.Category {
		case "mandatory":
			g.Mandatory[e.Term] = e.Value
		case "preferred":
			g.Preferred[e.Term] = e.Value
		case "keep":
			g.Keep = append(g.Keep, e.Term)
		case "forbidden":
			g.Forbidden = append(g.Forbidden, e.Term)
		}
	}
	protected, err := prepareTranslationUnitInput(u, g)
	if err != nil || len(protected.Tokens) != r.ProtectedTokenCount || (!bytes.Equal(input, []byte(protected.Text)) && !bytes.Equal(input, canonicalizeRetranslationArtifactEOF([]byte(protected.Text)))) {
		return fmt.Errorf("original frozen protector cannot be reproduced")
	}
	var responseCheck func() error
	if r.RevisionFeedbackSource == CurrentValidationRevisionFeedback {
		if len(catalogs) == 0 || catalogs[0] == nil {
			return fmt.Errorf("comment transport historical catalog required")
		}
		batch, err := readRetranslationProcessManifest(filepath.Join(root, "data/retranslation-runs", locale, s.SelectedBatchID), locale, s.SelectedBatchID)
		if err != nil {
			return err
		}
		historical, check, err := acceptedResponseCatalog(root, catalogs[0], locale, r)
		if err != nil {
			return err
		}
		responseCheck = check
		protected, err = prepareRetranslationOutput(root, historical, g, locale, batch.Units, r)
		if err != nil {
			return err
		}
	}
	if r.RepresentationBridge != nil || (hasInheritedBridgePredecessor(r) && len(catalogs) > 0 && catalogs[0] != nil) {
		if len(catalogs) == 0 || catalogs[0] == nil {
			return fmt.Errorf("historical revision catalog required")
		}
		decoder, lineage, err := surfaceTransportOutput(root, catalogs[0], locale, r)
		if err == nil && lineage == nil {
			decoder, lineage, err = inheritedBridgeOutput(root, catalogs[0], g, locale, r)
		}
		if err != nil {
			return err
		}
		if lineage != nil {
			protected = decoder
		}
	}
	restored, failures := protected.restore(string(raw))
	b = []byte(restored)
	if len(failures) != 0 || (!bytes.Equal(b, candidate) && !bytes.Equal(canonicalizeRetranslationArtifactEOF(b), candidate)) {
		return fmt.Errorf("original raw response/candidate relation changed")
	}
	if responseCheck != nil {
		return responseCheck()
	}
	return nil
}

func (a *CurrentTranslationAuthorization) Ready() bool {
	return len(a.Units) > 0 && a.Compatible == len(a.Units) && a.Affected+a.Stale+a.Unknown == 0
}
func authorizationDirectory(locale, scope, snapshot string) string {
	return "data/translation-authorizations/" + locale + "/" + scope + "/" + snapshot
}
func CurrentTranslationAuthorizationPath(a *CurrentTranslationAuthorization) string {
	return authorizationDirectory(a.Locale, a.ScopeIdentity, a.Historical.Finalization.SnapshotID) + "/" + a.Identity + ".json"
}

// EstablishCurrentTranslationAuthorization is explicit, deterministic and
// no-overwrite. Failed assessments never create a partial authorization.
func EstablishCurrentTranslationAuthorization(root string, c *Catalog, locale, snapshot string, canonical ...bool) (*CurrentTranslationAuthorization, string, error) {
	a, err := AssessCurrentTranslation(root, c, locale, snapshot)
	if err != nil {
		return nil, "", err
	}
	if !a.Ready() {
		return a, "", fmt.Errorf("current authorization blocked: affected=%d stale=%d unknown=%d", a.Affected, a.Stale, a.Unknown)
	}
	if len(canonical) > 0 && canonical[0] {
		refs, err := currentAuthorizationCanonicalTargets(root, c, a)
		if err != nil {
			return a, "", err
		}
		a.Canonical = refs
		a.Identity = authorizationIdentity(*a)
	}
	p := CurrentTranslationAuthorizationPath(a)
	b, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return nil, "", err
	}
	b = append(b, '\n')
	if err := writeCompatibilityImmutable(root, p, b); err != nil {
		return nil, "", err
	}
	return a, p, nil
}

// CheckCurrentTranslationAuthorization checks a sealed, flat historical proof
// and revalidates current exact candidates. It never invokes the historical QC
// evaluation, global compatibility inventory or predecessor/current replay.
func CheckCurrentTranslationAuthorization(root string, c *Catalog, path string) (*CurrentTranslationAuthorization, error) {
	b, err := readCompatibilityFile(root, path)
	if err != nil {
		return nil, err
	}
	var a CurrentTranslationAuthorization
	if err := contentidentity.StrictJSON(b, &a); err != nil {
		return nil, err
	}
	if a.Schema != CurrentTranslationAuthorizationSchema || !a.Ready() || a.Identity != authorizationIdentity(a) || path != CurrentTranslationAuthorizationPath(&a) {
		return nil, fmt.Errorf("current authorization identity/exact-set mismatch")
	}
	if err := checkFlatHistoricalIntegrity(root, c, &a.Historical); err != nil {
		return nil, err
	}
	if err := validateCompatibilityReview(root, a.Locale, a.Glossary.SHA256, a.UnifiedReview); err != nil {
		return nil, err
	}
	want, err := assessCurrentTranslationProof(root, c, a.Locale, a.Historical)
	if err != nil {
		return nil, err
	}
	if len(a.Canonical) > 0 {
		refs, err := currentAuthorizationCanonicalTargets(root, c, want)
		if err != nil {
			return nil, err
		}
		want.Canonical = refs
	}
	// Corpus/package append with byte-identical glossary still requires today's
	// full Unified Review. It need not manufacture another A or rewrite this
	// exact package checkpoint; its original full Review remains hash-bound.
	want.UnifiedReview = a.UnifiedReview
	want.Identity = authorizationIdentity(*want)
	if !reflect.DeepEqual(a, *want) {
		return nil, fmt.Errorf("current authorization stale: affected=%d stale=%d unknown=%d", want.Affected, want.Stale, want.Unknown)
	}
	return &a, nil
}

func currentAuthorizationCanonicalTargets(root string, c *Catalog, a *CurrentTranslationAuthorization) ([]GlossaryArchiveReference, error) {
	refs := []GlossaryArchiveReference{}
	for _, x := range a.Units {
		u, err := c.Unit(x.Snapshot.UnitID)
		if err != nil {
			return nil, err
		}
		path, err := canonicalTranslationUnitCandidatePath(a.Locale, u)
		if err != nil {
			return nil, err
		}
		accepted, err := readCompatibilityFile(root, x.Snapshot.CandidatePath)
		if err != nil || sum(accepted) != x.Snapshot.CandidateSHA256 {
			return nil, fmt.Errorf("accepted canonical candidate drift")
		}
		if u.Kind == UnitKindPage {
			accepted = canonicalizeCandidateEOF(accepted)
		}
		current, err := readCompatibilityFile(root, path)
		if err != nil || !bytes.Equal(current, accepted) {
			return nil, fmt.Errorf("current canonical differs from accepted A: %s", u.ID)
		}
		refs = append(refs, GlossaryArchiveReference{path, sum(current)})
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Path < refs[j].Path })
	return refs, nil
}

// The selected scope's current checkpoint is independent of other packages'
// finalized Snapshots. No legacy successor/compatibility scan occurs here.
func findScopeCurrentAuthorization(root string, c *Catalog, locale string) (*CurrentTranslationAuthorization, error) {
	if err := ValidateLocaleName(locale); err != nil {
		return nil, err
	}
	if _, err := compatibilityPath(root, "data/translation-authorizations/"+locale+"/.inventory", false); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	_, _, scope, _, err := translationAuthorizationScope(c)
	if err != nil {
		return nil, err
	}
	dir := "data/translation-authorizations/" + locale + "/" + scope
	path, err := compatibilityPath(root, dir+"/.inventory", false)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var match *CurrentTranslationAuthorization
	var lastErr error
	for _, e := range entries {
		if !e.IsDir() || e.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("unsafe scope checkpoint member")
		}
		a, err := FindCurrentTranslationAuthorization(root, c, locale, e.Name())
		if err != nil {
			lastErr = err
			continue
		}
		if a == nil {
			continue
		}
		if match != nil && !reflect.DeepEqual(match.Canonical, a.Canonical) {
			return nil, fmt.Errorf("ambiguous current canonical checkpoint")
		}
		match = a
	}
	if match != nil {
		return match, nil
	}
	if len(entries) > 0 {
		return nil, lastErr
	}
	return nil, nil
}

// ReadHistoricalTranslationReference rechecks one exact member of an already
// verified historical integrity proof. Downstream dependency graphs must retain
// this historical read domain instead of treating frozen context as live bytes.
// Callers establish the full proof with ReadHistoricalTranslationIntegrity;
// this reader never grants current candidate or publication eligibility.
func ReadHistoricalTranslationReference(root string, c *Catalog, p *HistoricalFinalizationIntegrity, ref GlossaryArchiveReference) ([]byte, error) {
	if p != nil {
		for _, member := range p.References {
			if member == ref {
				return readHistoricalContextReference(root, c, p.Finalization.Locale, ref)
			}
		}
	}
	return nil, fmt.Errorf("reference outside verified historical translation proof: %s", ref.Path)
}

func checkFlatHistoricalIntegrity(root string, c *Catalog, p *HistoricalFinalizationIntegrity) error {
	seen := map[string]string{}
	for _, r := range p.References {
		if seen[r.Path] != "" {
			return fmt.Errorf("duplicate checkpoint history ref")
		}
		b, err := readHistoricalContextReference(root, c, p.Finalization.Locale, r)
		if err != nil || sum(b) != r.SHA256 {
			return fmt.Errorf("checkpoint historical evidence missing/changed: %s", r.Path)
		}
		seen[r.Path] = r.SHA256
	}
	f := p.Finalization
	path := filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", f.Locale, f.SnapshotID, "finalization.json"))
	b, err := readCompatibilityFile(root, path)
	if err != nil || sum(b) != seen[path] {
		return fmt.Errorf("checkpoint original finalization missing")
	}
	var got QualityCheckFinalization
	if err := contentidentity.StrictJSON(b, &got); err != nil || !bytes.Equal(mustJSON(got), mustJSON(f)) {
		return fmt.Errorf("checkpoint original finalization binding mismatch")
	}
	if seen[f.SnapshotManifestPath] != f.SnapshotManifestSHA256 || len(p.ReviewerGlossaries) != len(f.Units) {
		return fmt.Errorf("checkpoint original Snapshot binding mismatch")
	}
	for _, r := range f.QCResults {
		if seen[r.Path] != r.SHA256 {
			return fmt.Errorf("checkpoint original Reviewer input missing")
		}
	}
	for i, u := range f.Units {
		if seen[u.Snapshot.CandidatePath] != u.Snapshot.CandidateSHA256 || seen[u.Snapshot.ValidationPath] != u.Snapshot.ValidationSHA256 {
			return fmt.Errorf("checkpoint candidate/validation proof missing")
		}
		s, err := readQualityCheckSnapshot(root, f.Locale, u.SourceSnapshotID, false)
		if err != nil || s.GlossarySHA256 != p.ReviewerGlossaries[i] {
			return fmt.Errorf("checkpoint original Reviewer glossary mismatch")
		}
	}
	return nil
}

// FindCurrentTranslationAuthorization only searches this exact scope/Snapshot.
// Unrelated locale/package additions do not enter its authority or replay path.
func FindCurrentTranslationAuthorization(root string, c *Catalog, locale, snapshot string) (*CurrentTranslationAuthorization, error) {
	if err := ValidateLocaleName(locale); err != nil {
		return nil, err
	}
	if _, err := compatibilityPath(root, "data/translation-authorizations/"+locale+"/.inventory", false); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	// Validate every sealed immutable proof before selecting an eligible epoch;
	// a newer record cannot conceal corrupt original Reviewer evidence.
	if _, err := loadSealedTranslationHistory(root, c, locale, snapshot); err != nil {
		return nil, err
	}
	_, _, scope, _, err := translationAuthorizationScope(c)
	if err != nil {
		return nil, err
	}
	if err := ValidateLocaleName(locale); err != nil {
		return nil, err
	}
	if err := validateSnapshotID(snapshot); err != nil {
		return nil, err
	}
	dir := authorizationDirectory(locale, scope, snapshot)
	resolved, err := compatibilityPath(root, dir+"/.inventory", false)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Dir(resolved))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var match *CurrentTranslationAuthorization
	var lastErr error
	for _, e := range entries {
		if !e.Type().IsRegular() || !strings.HasSuffix(e.Name(), ".json") {
			return nil, fmt.Errorf("unsafe authorization member")
		}
		a, err := CheckCurrentTranslationAuthorization(root, c, dir+"/"+e.Name())
		if err != nil {
			lastErr = err
			continue
		}
		if match != nil && !reflect.DeepEqual(match.Units, a.Units) {
			return nil, fmt.Errorf("ambiguous current authorization")
		}
		if match == nil || len(a.Canonical) > 0 {
			match = a
		}
	}
	if match != nil {
		return match, nil
	}
	if len(entries) > 0 {
		return nil, lastErr
	}
	return nil, nil
}

// Re-attestation may read an older sealed proof without accepting its old
// current world. This permits exact affected-only QC planning after drift.
func loadSealedTranslationHistory(root string, c *Catalog, locale, snapshot string) (*HistoricalFinalizationIntegrity, error) {
	if err := ValidateLocaleName(locale); err != nil {
		return nil, err
	}
	if err := validateSnapshotID(snapshot); err != nil {
		return nil, err
	}
	if _, err := compatibilityPath(root, "data/translation-authorizations/"+locale+"/.inventory", false); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	_, _, scope, _, err := translationAuthorizationScope(c)
	if err != nil {
		return nil, err
	}
	dir := authorizationDirectory(locale, scope, snapshot)
	file, err := compatibilityPath(root, dir+"/.inventory", false)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Dir(file))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var proof *HistoricalFinalizationIntegrity
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".json") {
			return nil, fmt.Errorf("unsafe sealed history member")
		}
		path := dir + "/" + entry.Name()
		b, err := readCompatibilityFile(root, path)
		if err != nil {
			return nil, err
		}
		var a CurrentTranslationAuthorization
		if err := contentidentity.StrictJSON(b, &a); err != nil {
			return nil, err
		}
		if a.Schema != CurrentTranslationAuthorizationSchema || a.Locale != locale || a.ScopeIdentity != scope || a.Historical.Finalization.SnapshotID != snapshot || !a.Ready() || a.Identity != authorizationIdentity(a) || path != CurrentTranslationAuthorizationPath(&a) {
			return nil, fmt.Errorf("sealed history identity mismatch")
		}
		if err := checkFlatHistoricalIntegrity(root, c, &a.Historical); err != nil {
			return nil, err
		}
		if err := validateCompatibilityReview(root, locale, a.Glossary.SHA256, a.UnifiedReview); err != nil {
			return nil, err
		}
		if proof != nil && !reflect.DeepEqual(*proof, a.Historical) {
			return nil, fmt.Errorf("conflicting sealed original A proof")
		}
		p := a.Historical
		proof = &p
	}
	return proof, nil
}

func RequireCurrentTranslationAuthorization(root string, c *Catalog, locale, snapshot string) (*CurrentTranslationAuthorization, error) {
	a, err := FindCurrentTranslationAuthorization(root, c, locale, snapshot)
	if err != nil {
		return nil, err
	}
	if a == nil {
		a, err = AssessCurrentTranslation(root, c, locale, snapshot)
		if err != nil {
			return nil, err
		}
	}
	if !a.Ready() {
		return nil, fmt.Errorf("current authorization blocked: affected=%d stale=%d unknown=%d", a.Affected, a.Stale, a.Unknown)
	}
	return a, nil
}
