package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// The adapter owns the closed, versioned inverse and child inventory. The two
// returned values must be typed structures; incoming JSON spelling is never an
// identity input. This interface confers no lifecycle or language authority.
type DerivedTransportCodecReader func(TranslationUnitAdapter, []byte, []byte) (any, any, TranslationUnitAdapter, error)

type DerivedTransportProducer struct {
	ID             string               `json:"id"`
	Version        int                  `json:"contract_version"`
	Implementation []RecoveryDependency `json:"implementation"`
	Identity       string               `json:"implementation_identity"`
}

type DerivedTransportSemanticContext struct {
	Locale       string                    `json:"locale"`
	Package      RecoveryPackage           `json:"content_package"`
	OrderedScope []RecoverySourceUnit      `json:"ordered_recovery_scope"`
	Language     RecoveryLanguageAuthority `json:"language_authority"`
	Dependencies []RecoveryDependency      `json:"semantic_dependencies"`
}

type DerivedApprovedConstruction struct {
	Namespace             string                          `json:"namespace"`
	UnitID                string                          `json:"unit_id"`
	SourceSHA256          string                          `json:"source_sha256"`
	OldRepresentation     string                          `json:"old_representation_sha256"`
	CurrentRepresentation string                          `json:"current_representation_sha256"`
	CandidateSHA256       string                          `json:"candidate_sha256"`
	ParentSource          ApprovedIntervalCoordinate      `json:"parent_source"`
	ParentCandidate       ApprovedIntervalCoordinate      `json:"parent_candidate"`
	ParentSHA256          string                          `json:"parent_target_sha256"`
	ParentToken           string                          `json:"parent_token"`
	Children              any                             `json:"current_children_source_inventory"`
	Findings              []ApprovedIntervalFindingRegion `json:"typed_finding_regions"`
	Text                  string                          `json:"text"`
	Tokens                []string                        `json:"complete_projection_token_inventory"`
	Codec                 any                             `json:"inverse_restore_contract"`
	SemanticAuthority     GlossaryArchiveReference        `json:"semantic_authority"`
	ProvenanceSource      string                          `json:"provenance_source"`
	LegacyReference       *GlossaryArchiveReference       `json:"legacy_reference,omitempty"`
}

type DerivedDecoderConstruction struct {
	UnitID            string                        `json:"unit_id"`
	Variant           string                        `json:"variant"`
	Input             string                        `json:"formal_protected_input"`
	ProtectedBaseline string                        `json:"protected_response_baseline"`
	Tokens            []string                      `json:"tokens"`
	Approved          *DerivedApprovedConstruction  `json:"approved,omitempty"`
	Rebase            *CurrentValidationPredecessor `json:"deterministic_rebase,omitempty"`
	Comment           *CommentRevisionTransport     `json:"comment_transport,omitempty"`
}

type DerivedTransportConstruction struct {
	Schema          string                          `json:"schema"`
	Producer        DerivedTransportProducer        `json:"producer"`
	SemanticContext DerivedTransportSemanticContext `json:"semantic_context"`
	Decoders        []DerivedDecoderConstruction    `json:"decoder_constructions"`
	Dependencies    []RecoveryDependency            `json:"construction_dependencies"`
	Identity        string                          `json:"construction_identity"`
}

func derivedConstructionIdentity(c DerivedTransportConstruction) string {
	c.Identity = ""
	return sum(mustJSON(c))
}

func typedApprovedConstruction(c *Catalog, u *TranslationUnit, p *ApprovedIntervalTransport) (*DerivedApprovedConstruction, TranslationUnitAdapter, error) {
	if c.DerivedTransportCodecReader == nil {
		return nil, nil, fmt.Errorf("unsupported versioned approved inverse reader")
	}
	codec, children, decoder, err := c.DerivedTransportCodecReader(u.Adapter, p.Projection.Codec, p.Projection.Children)
	if err != nil {
		return nil, nil, err
	}
	for _, v := range []any{codec, children} {
		t := reflect.TypeOf(v)
		if t == nil || t == reflect.TypeOf(json.RawMessage{}) || t.Kind() == reflect.Map || t.Kind() == reflect.String {
			return nil, nil, fmt.Errorf("inverse reader must return source-owned typed values")
		}
	}
	x := p.Projection
	return &DerivedApprovedConstruction{
		Namespace: x.Namespace, UnitID: x.UnitID, SourceSHA256: x.SourceSHA256,
		OldRepresentation: x.OldRepresentationSHA256, CurrentRepresentation: x.CurrentRepresentationSHA256,
		CandidateSHA256: x.CandidateSHA256, ParentSource: x.ParentSource, ParentCandidate: x.ParentCandidate,
		ParentSHA256: x.ParentSHA256, ParentToken: x.ParentToken, Children: children, Findings: x.Findings,
		Text: x.Text, Tokens: x.Tokens, Codec: codec, SemanticAuthority: p.SemanticAuthority,
		ProvenanceSource: p.ProvenanceSource, LegacyReference: p.LegacyReference,
	}, decoder, nil
}

// Legacy v1 recorded full implementation references. This table is the
// versioned producer declaration; callers cannot add paths or assign roles.
// A change outside these declared constructor declarations is NOT classified
// merely because it occurs in the same Go file.
// generationApprovedIntervalSelection owns the explicit/frozen/imported/recovery
// selector that determines the approved decoder construction. Its recovery
// branch derives that selector and checks exact scope; it performs no artifact
// encoding/publication. This binding classifies a construction DEPENDENCY delta,
// not a claim that every branch changed the old invocation's decoder semantics.
// approvedResponseInputs has separate installed execution/provenance duties and
// deliberately does not acquire ownership from this adjacent declaration.
var approvedConstructionDeclarationsV1 = map[string][]string{
	"internal/i18n/approved_interval_transport.go":          {"func:buildApprovedIntervalTransport", "func:generationApprovedIntervalSelection", "func:approvedIntervalIdentity", "type:ApprovedIntervalProjection", "type:ApprovedIntervalTransport"},
	"internal/i18n/current_validation_approved_interval.go": {"func:InspectApprovedIntervalReadiness", "func:InspectApprovedIntervalPredecessors"},
	"internal/i18n/generation_revision_inputs.go":           {"func:revisionGenerationInputsCore"},
	"internal/i18n/generation_bundle.go":                    {"func:ExportTranslationUnitGenerationBundle", "func:validatePreparedGenerationOutput"},
	"internal/i18n/retranslation_process_context.go":        {"func:prepareProcessOutputContext"},
	"internal/i18n/generation_import_archive.go":            {"func:verifyImportedRevisionInvocation"},
	"internal/i18n/bridge_invocation.go":                    {"func:inheritedBridgeOutput", "func:frozenBridgeOutput"},
	"internal/i18n/candidate_snapshot_outputs.go":           {"func:prepareSnapshotRevisionOutputs", "func:snapshotRevisionEvidenceCatalog"},
	"internal/i18n/retranslation_promote.go":                {"func:validateRetranslationCandidateWithOutput"},
	"internal/sitecontent/approved_interval_transport.go":   {"type:approvedPageCodec", "type:approvedPageSlot", "func:ApprovedIntervalProjection", "func:Restore", "func:ProtectedInput"},
	"internal/sitecontent/approved_interval_coordinates.go": {"func:approvedMarkdownEngine", "func:approvedFindingCoordinates:PageAdapter", "func:RegisterFuncs:*approvedTraceRenderer", "func:Register:approvedTraceRegister"},
}

// Entire files are retained for identity/integrity. AST declarations are used
// only to classify the delta; comments/formatting do not prove construction
// drift. No old source is executed and no current facts fill an old archive.
func recoveryGoDeclarations(path string, b []byte) (map[string]string, error) {
	fs := token.NewFileSet()
	var directives []string
	var scan scanner.Scanner
	scan.Init(fs.AddFile(path, -1, len(b)), b, nil, scanner.ScanComments)
	for {
		_, tok, text := scan.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.COMMENT && (strings.HasPrefix(text, "//go:") || strings.HasPrefix(text, "// +build") || strings.HasPrefix(text, "//line ") || strings.HasPrefix(text, "/*line ")) {
			directives = append(directives, text)
		}
	}
	f, err := parser.ParseFile(fs, path, b, 0)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for i, d := range directives {
		out[fmt.Sprintf("compiler-directive:%04d", i)] = d
	}
	put := func(name string, n ast.Node) error {
		if _, ok := out[name]; ok {
			return fmt.Errorf("ambiguous construction declaration %s", name)
		}
		var buf bytes.Buffer
		if err := format.Node(&buf, fs, n); err != nil {
			return err
		}
		out[name] = buf.String()
		return nil
	}
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			name := "func:" + d.Name.Name
			if d.Recv != nil {
				var r bytes.Buffer
				if err := format.Node(&r, fs, d.Recv.List[0].Type); err != nil {
					return nil, err
				}
				name += ":" + r.String()
			}
			if err := put(name, d); err != nil {
				return nil, err
			}
		case *ast.GenDecl:
			for _, s := range d.Specs {
				switch s := s.(type) {
				case *ast.TypeSpec:
					if err := put("type:"+s.Name.Name, s); err != nil {
						return nil, err
					}
				case *ast.ValueSpec:
					for _, n := range s.Names {
						if err := put("value:"+n.Name, s); err != nil {
							return nil, err
						}
					}
				case *ast.ImportSpec:
					if err := put("import:"+s.Path.Value, s); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	return out, nil
}

func classifyApprovedDependencyDelta(path string, old, current []byte) ([]string, error) {
	allowed, ok := approvedConstructionDeclarationsV1[path]
	if !ok {
		return nil, fmt.Errorf("UNCLASSIFIED_PROOF_MISMATCH: undeclared construction dependency")
	}
	a, err := recoveryGoDeclarations(path, old)
	if err != nil {
		return nil, err
	}
	b, err := recoveryGoDeclarations(path, current)
	if err != nil {
		return nil, err
	}
	keys := map[string]bool{}
	for k := range a {
		keys[k] = true
	}
	for k := range b {
		keys[k] = true
	}
	var changed []string
	for k := range keys {
		if a[k] == b[k] {
			continue
		}
		owned := false
		for _, decl := range allowed {
			if k == decl {
				owned = true
			}
			// Explicit receiver declarations are version-owned. A same-named
			// method on another receiver does not acquire the role.
			if path == "internal/sitecontent/approved_interval_transport.go" && (k == decl+":*approvedPageCodec" || k == decl+":PageAdapter") {
				owned = true
			}
		}
		if !owned {
			return nil, fmt.Errorf("UNCLASSIFIED_PROOF_MISMATCH: mixed-role declaration %s", k)
		}
		changed = append(changed, k)
	}
	sort.Strings(changed)
	if len(changed) == 0 {
		return nil, fmt.Errorf("ENCODING_ONLY_MISMATCH: no construction declaration delta")
	}
	return changed, nil
}

func recoveryArchiveReference(files map[string][]byte, ref GlossaryArchiveReference) ([]byte, error) {
	if validateGenerationInstallPath(ref.Path) != nil || !validSHA256(ref.SHA256) {
		return nil, fmt.Errorf("invalid declared old dependency reference")
	}
	// Content lookup is restricted to the already verified archive inventory and
	// the exact declared SHA. It is not receipt discovery or a path heuristic.
	for _, b := range files {
		if sum(b) == ref.SHA256 {
			return b, nil
		}
	}
	return nil, fmt.Errorf("missing declared old construction dependency: %s", ref.Path)
}

func recoveryProofFromFiles(files map[string][]byte, id string) (*ApprovedIntervalTransport, error) {
	b, ok := files[approvedIntervalProofMember(id)]
	if !ok {
		return nil, nil
	}
	var p ApprovedIntervalTransport
	if err := contentidentity.StrictJSON(b, &p); err != nil {
		return nil, err
	}
	if p.Schema != ApprovedIntervalTransportSchema || p.Mode != ApprovedIntervalMode || p.Record.UnitID != id || p.Identity != approvedIntervalIdentity(p) {
		return nil, fmt.Errorf("invalid frozen approved proof")
	}
	return &p, nil
}

// Build the invocation-locator-independent recipe from a complete, verified
// member graph. The caller supplies the immutable original language scope for
// both old and current constructions, never the new batch's provenance fields.
func deriveRecoveryConstruction(root string, c *Catalog, m *RetranslationBatchManifest, files map[string][]byte, semantic DerivedTransportSemanticContext, fromFrozen bool, materials ...*historicalConstructionResolver) (*DerivedTransportConstruction, map[string]TranslationUnitAdapter, error) {
	x := &DerivedTransportConstruction{Schema: DerivedTransportConstructionSchema, Producer: DerivedTransportProducer{ID: ApprovedIntervalTransportSchema, Version: 1}, SemanticContext: semantic}
	var saved *DerivedTransportConstruction
	if fromFrozen && len(files[transportRecoveryConstructionMember]) > 0 {
		var err error
		saved, err = decodeDerivedTransportConstruction(c, files[transportRecoveryConstructionMember])
		if err != nil {
			return nil, nil, err
		}
		x.Producer.Version = saved.Producer.Version
	} else if !fromFrozen {
		x.Producer.Version = 2
	}
	decoders := map[string]TranslationUnitAdapter{}
	impl := map[string]RecoveryDependency{}
	deps := map[string]RecoveryDependency{}
	g, err := LoadGlossary(root, m.Locale)
	if err != nil {
		return nil, nil, err
	}
	content := map[string][]byte{}
	for _, b := range files {
		content[sum(b)] = b
	}
	for _, r := range m.Units {
		u, err := c.Unit(r.UnitID)
		if err != nil {
			return nil, nil, err
		}
		baseline, ok := files["formal/protected-candidates/"+filepath.Base(r.InputPath)]
		if !ok {
			return nil, nil, fmt.Errorf("missing protected response baseline")
		}
		input, ok := files["formal/"+r.InputPath]
		if !ok || sum(input) != r.InputSHA256 {
			return nil, nil, fmt.Errorf("missing exact formal source input")
		}
		d := DerivedDecoderConstruction{UnitID: r.UnitID, Variant: "ordinary", Input: string(input), ProtectedBaseline: string(baseline)}
		normal, err := prepareTranslationUnitInput(u, g)
		if err != nil {
			return nil, nil, err
		}
		if sum(canonicalizeRetranslationArtifactEOF([]byte(normal.Text))) != r.InputSHA256 {
			return nil, nil, fmt.Errorf("formal representation identity drift")
		}
		d.Tokens = append([]string(nil), normal.Tokens...)
		p, err := recoveryProofFromFiles(files, r.UnitID)
		if err != nil {
			return nil, nil, err
		}
		if p != nil {
			d.Variant = "approved-interval"
			d.Approved, decoders[r.UnitID], err = typedApprovedConstruction(c, u, p)
			if err != nil {
				return nil, nil, err
			}
			d.Tokens = append([]string(nil), p.Projection.Tokens...)
			for _, ref := range p.Dependencies {
				role := "immutable-approved-dependency"
				if _, ok := approvedConstructionDeclarationsV1[ref.Path]; ok {
					role = "approved-construction-implementation"
				}
				b, found := content[ref.SHA256]
				if fromFrozen && role == "approved-construction-implementation" && len(materials) > 0 {
					b, err = materials[0].Read(r.UnitID, ref, role, x.Producer.Version)
					if err != nil {
						return nil, nil, err
					}
					found = true
				}
				if !found {
					// A surviving declared immutable dependency is an old fact only
					// when its bytes match the OLD declared digest. Never substitute
					// today's reference or fill a missing changed implementation.
					var err error
					b, err = readCompatibilityReference(root, ref)
					if err != nil {
						return nil, nil, fmt.Errorf("missing declared historical construction bytes: %s", ref.Path)
					}
					content[ref.SHA256] = b
				}
				v := RecoveryDependency{Role: role, Ref: recoveryMemberReference(ref.Path, b)}
				deps[role+"\x00"+ref.Path] = v
				if role == "approved-construction-implementation" {
					impl[ref.Path] = v
				}
			}
		} else if b, ok := files["formal/current-validation-predecessor-transport/"+filepath.Base(r.InputPath)+".json"]; ok {
			d.Variant = "deterministic-rebase"
			var p CurrentValidationPredecessor
			if err := contentidentity.StrictJSON(b, &p); err != nil {
				return nil, nil, err
			}
			if p.Schema != "go-learning/current-validation-predecessor-transport/v1" || p.RecoveryClass != "DETERMINISTIC_REBASE_PASS" || p.UnitID != r.UnitID || p.Template != string(baseline) || p.TemplateSHA256 != sum(baseline) {
				return nil, nil, fmt.Errorf("invalid exact rebase construction")
			}
			d.Rebase = &p
			d.Rebase.Selected = append([]RetranslationBatchUnit(nil), semantic.Language.Scope...)
		} else if b, ok := files[commentTransportPath(r)]; ok {
			d.Variant = "comment-transport"
			var p CommentRevisionTransport
			if err := contentidentity.StrictJSON(b, &p); err != nil {
				return nil, nil, err
			}
			if p.Schema != "go-learning/comment-revision-transport/v1" || p.UnitID != r.UnitID || p.Text != string(baseline) {
				return nil, nil, fmt.Errorf("invalid comment construction")
			}
			d.Comment = &p
		}
		x.Decoders = append(x.Decoders, d)
	}
	for _, d := range impl {
		x.Producer.Implementation = append(x.Producer.Implementation, d)
	}
	if x.Producer.Version == 2 {
		for _, path := range transportRecoveryImplementationPaths {
			var ref RecoveryFileReference
			if fromFrozen {
				for _, r := range saved.Producer.Implementation {
					if r.Role == "approved-construction-reader" && r.Ref.Path == path {
						ref = r.Ref
					}
				}
				if !validRecoveryFileReference(ref) {
					return nil, nil, fmt.Errorf("missing frozen producer-v2 reader declaration")
				}
				b, ok := content[ref.SHA256]
				if len(materials) > 0 {
					b, err = materials[0].ReadProducer(GlossaryArchiveReference{ref.Path, ref.SHA256}, "approved-construction-reader", x.Producer.Version)
					if err != nil {
						return nil, nil, err
					}
					ok = true
				}
				if !ok || int64(len(b)) != ref.Size {
					return nil, nil, fmt.Errorf("missing frozen producer-v2 implementation bytes")
				}
			} else {
				var err error
				ref, _, err = recoveryDiskReference(root, path)
				if err != nil {
					return nil, nil, err
				}
			}
			r := RecoveryDependency{Role: "approved-construction-reader", Ref: ref}
			x.Producer.Implementation = append(x.Producer.Implementation, r)
			deps[r.Role+"\x00"+path] = r
		}
	}
	for _, d := range deps {
		x.Dependencies = append(x.Dependencies, d)
	}
	x.Producer.Implementation, err = sortedRecoveryDependencies(x.Producer.Implementation)
	if err != nil {
		return nil, nil, err
	}
	x.Dependencies, err = sortedRecoveryDependencies(x.Dependencies)
	if err != nil {
		return nil, nil, err
	}
	wantImplementations := len(approvedConstructionDeclarationsV1)
	if x.Producer.Version == 2 {
		wantImplementations += len(transportRecoveryImplementationPaths)
	}
	if len(x.Producer.Implementation) != wantImplementations {
		return nil, nil, fmt.Errorf("incomplete versioned approved producer implementation")
	}
	x.Producer.Identity = sum(mustJSON(x.Producer.Implementation))
	x.Identity = derivedConstructionIdentity(*x)
	if saved != nil && !bytes.Equal(mustJSON(saved), mustJSON(x)) {
		return nil, nil, fmt.Errorf("frozen construction does not match verified old invocation facts")
	}
	return x, decoders, nil
}

func decodeDerivedTransportConstruction(c *Catalog, b []byte) (*DerivedTransportConstruction, error) {
	var x DerivedTransportConstruction
	if err := contentidentity.StrictJSON(b, &x); err != nil {
		return nil, err
	}
	if x.Schema != DerivedTransportConstructionSchema || x.Producer.ID != ApprovedIntervalTransportSchema || (x.Producer.Version != 1 && x.Producer.Version != 2) || len(x.Decoders) != len(x.SemanticContext.OrderedScope) {
		return nil, fmt.Errorf("unsupported construction schema/producer/scope")
	}
	for i := range x.Decoders {
		d := &x.Decoders[i]
		if d.UnitID != x.SemanticContext.OrderedScope[i].UnitID {
			return nil, fmt.Errorf("construction decoder order mismatch")
		}
		switch d.Variant {
		case "ordinary":
			if d.Approved != nil || d.Rebase != nil || d.Comment != nil {
				return nil, fmt.Errorf("ordinary decoder has extra payload")
			}
		case "deterministic-rebase":
			if d.Rebase == nil || d.Approved != nil || d.Comment != nil {
				return nil, fmt.Errorf("incomplete rebase payload")
			}
		case "comment-transport":
			if d.Comment == nil || d.Approved != nil || d.Rebase != nil {
				return nil, fmt.Errorf("incomplete comment payload")
			}
		case "approved-interval":
			if d.Approved == nil || d.Rebase != nil || d.Comment != nil || c.DerivedTransportCodecReader == nil {
				return nil, fmt.Errorf("incomplete approved construction")
			}
			u, err := c.Unit(d.UnitID)
			if err != nil {
				return nil, err
			}
			codec, children, _, err := c.DerivedTransportCodecReader(u.Adapter, mustJSON(d.Approved.Codec), mustJSON(d.Approved.Children))
			if err != nil {
				return nil, err
			}
			d.Approved.Codec, d.Approved.Children = codec, children
		default:
			return nil, fmt.Errorf("unsupported construction decoder variant")
		}
	}
	if x.Identity != derivedConstructionIdentity(x) || x.Producer.Identity != sum(mustJSON(x.Producer.Implementation)) {
		return nil, fmt.Errorf("typed construction identity mismatch")
	}
	return &x, nil
}
