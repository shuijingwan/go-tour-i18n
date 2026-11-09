package sitecontent

import (
	"fmt"
	"os"
	"reflect"
	"sort"
)

const SharedReattestationSchema = "go-learning/shared-structured-representation-reattestation/v1"
const SharedReattestationRoot = "data/site-shell/shared-representation-reattestations/"
const SharedReattestationPending = "data/.shared-structured-representation-reattestation-pending"

var sharedReattestationValidationPaths = []string{
	repositoryModeImplementation,
	"internal/languageaudit/audit.go", "internal/languageaudit/policy.go", "internal/languageview/html.go", "internal/languageview/identity.go", "internal/languageview/semantic_role.go", "internal/languageview/semantic_go_packages.go", "internal/i18n/language_audit.go", "internal/i18n/language_audit_tour.go", "internal/i18n/language_audit_ui.go", "internal/i18n/language_audit_surface.go", "internal/sitecontent/language_audit_formal.go", "internal/sitecontent/language_audit_view.go", "internal/sitecontent/language_audit_tour.go", "docs/LANGUAGE_DEFECT_EXHAUSTION.md",
	"internal/sitecontent/shared_representation_reattestation.go",
	"internal/sitecontent/shared_representation_reattestation_publication.go",
	"internal/sitecontent/structured_source_successor.go",
	"internal/sitecontent/scope.go",
	"internal/sitecontent/structured.go",
	"cmd/tour-i18n/shared_representation_reattestation.go",
	"cmd/tour-i18n/shared_representation_reattestation_test.go",
	"docs/SHARED_STRUCTURED_REPRESENTATION_REATTESTATION.md",
}

// Full frozen base is an invariant, including its original dependency envelope.
// This authority attests implementation evolution, never source or language.
type SharedRepresentationReattestation struct {
	Schema                 string                                `json:"schema"`
	BasePointer            Reference                             `json:"base_pointer"`
	BaseReceipt            Reference                             `json:"base_receipt"`
	BaseIdentity           string                                `json:"base_identity_sha256"`
	RepresentationIdentity string                                `json:"base_representation_identity_sha256"`
	Base                   SharedRepresentationSuccessor         `json:"base"`
	OldInputs              []ShellRepresentationInput            `json:"old_currentness_inputs"`
	CurrentInputs          []ShellRepresentationInput            `json:"current_currentness_inputs"`
	Changes                []ShellRepresentationDependencyChange `json:"changed_dependencies"`
	Added                  []ShellRepresentationInput            `json:"added_dependencies"`
	Removed                []ShellRepresentationInput            `json:"removed_dependencies"`
	ValidationInputs       []ShellRepresentationInput            `json:"validation_inputs"`
	Previous               *Reference                            `json:"previous,omitempty"`
	Identity               string                                `json:"identity_sha256"`
}
type SharedRepresentationReattestationPlan struct {
	Schema          string                            `json:"schema"`
	Authority       SharedRepresentationReattestation `json:"authority"`
	Receipt         Reference                         `json:"planned_receipt"`
	CurrentPath     string                            `json:"current_path"`
	PriorPointerSHA string                            `json:"prior_pointer_sha256"`
	BaseCurrent     bool                              `json:"base_receipt_current"`
	Eligible        bool                              `json:"reattestation_eligible"`
	State           string                            `json:"state"`
	Identity        string                            `json:"identity_sha256"`
}
type SharedRepresentationReattestationResult struct {
	Plan           *SharedRepresentationReattestationPlan `json:"plan"`
	AlreadyCurrent bool                                   `json:"already_current"`
	Applied        bool                                   `json:"applied"`
	Mutation       string                                 `json:"mutation"`
}

func sharedReattestationPath(base string) string {
	return SharedReattestationRoot + "current/" + base + ".json"
}
func sharedReattestationReceipt(r *SharedRepresentationReattestation) Reference {
	b, _ := Encode(r)
	return Reference{SharedReattestationRoot + "receipts/" + r.Identity + ".json", digest(b)}
}

// Inventory/mode/absence changes are forbidden, as in the Homepage family.
// Listing a file here grants only audited re-attestation eligibility, not an
// automatic currentness PASS, and grants nothing to any other family.
func sharedDependencyChanges(old, current []ShellRepresentationInput) ([]ShellRepresentationDependencyChange, error) {
	if len(old) != len(current) {
		return nil, fmt.Errorf("shared dependency inventory changed")
	}
	changes := []ShellRepresentationDependencyChange{}
	for i, a := range old {
		b := current[i]
		if a.Reference.Path != b.Reference.Path || !sameRepresentationMode(a, b) || a.Absent != b.Absent || a.Absent || !validSHA(a.Reference.SHA256) || !validSHA(b.Reference.SHA256) || (i > 0 && old[i-1].Reference.Path >= a.Reference.Path) {
			return nil, fmt.Errorf("shared dependency path/mode/absence changed: %s", a.Reference.Path)
		}
		if a.Reference == b.Reference {
			continue
		}
		allowed := false
		for _, p := range append(append([]string{}, sharedRepresentationParserPaths...), sharedReattestationValidationPaths...) {
			allowed = allowed || p == a.Reference.Path
		}
		if !allowed {
			return nil, fmt.Errorf("non-parser shared representation authority drift: %s", a.Reference.Path)
		}
		changes = append(changes, ShellRepresentationDependencyChange{Old: a, Current: b})
	}
	return changes, nil
}

// Bootstrap does not call the dependency-current checker. It verifies the
// sealed history and independently rebuilds source/owner/registry and topology.
func sharedReattestationBase(root string, g *Global, ref Reference, v *structuredSuccessorValidation) (Reference, *SharedRepresentationSuccessor, []ShellRepresentationInput, error) {
	var header SharedRepresentationSuccessor
	if err := v.guard.readReference(root, ref, &header); err != nil {
		return Reference{}, nil, nil, err
	}
	head, sha, err := readSharedRepresentationHead(root, header.Request.Package, header.Request.AssetID, v)
	if err != nil {
		return Reference{}, nil, nil, err
	}
	if head == nil || *head != ref {
		return Reference{}, nil, nil, fmt.Errorf("exact installed shared base receipt required")
	}
	raw, err := v.guard.readRegular(root, header.Request.Path)
	if err != nil {
		return Reference{}, nil, nil, err
	}
	base, err := readSharedRepresentationReceipt(root, ref, raw, v, map[Reference]bool{})
	if err != nil {
		return Reference{}, nil, nil, err
	}
	if err := checkSharedRepresentationSource(root, g, *base, v); err != nil {
		return Reference{}, nil, nil, err
	}
	if _, _, _, err := sharedRepresentationConsumedGeneration(root, base.Request.Generation, v.guard); err != nil {
		return Reference{}, nil, nil, err
	}
	current, err := sharedRepresentationCurrentInputs(root, *base, v)
	if err != nil {
		return Reference{}, nil, nil, err
	}
	if _, err := sharedDependencyChanges(base.Inputs, current); err != nil {
		return Reference{}, nil, nil, err
	}
	return Reference{sharedRepresentationHead(base.Request.Package, base.Request.AssetID), sha}, base, current, nil
}

func buildSharedReattestation(root string, pointer, ref Reference, base *SharedRepresentationSuccessor, current []ShellRepresentationInput, previous *Reference, v *structuredSuccessorValidation) (*SharedRepresentationReattestation, error) {
	changes, err := sharedDependencyChanges(base.Inputs, current)
	if err != nil {
		return nil, err
	}
	r := &SharedRepresentationReattestation{Schema: SharedReattestationSchema, BasePointer: pointer, BaseReceipt: ref, BaseIdentity: base.Identity, RepresentationIdentity: base.NewIdentity, Base: *base, OldInputs: base.Inputs, CurrentInputs: current, Changes: changes, Added: []ShellRepresentationInput{}, Removed: []ShellRepresentationInput{}, Previous: previous, ValidationInputs: []ShellRepresentationInput{}}
	paths := append([]string{}, sharedReattestationValidationPaths...)
	sort.Strings(paths)
	for _, path := range paths {
		b, err := v.guard.readRegular(root, path)
		if err != nil {
			return nil, err
		}
		st, err := os.Stat(pathJoin(root, path))
		if err != nil {
			return nil, err
		}
		mode, err := representationFileMode(st.Mode())
		if err != nil {
			return nil, err
		}
		r.ValidationInputs = append(r.ValidationInputs, ShellRepresentationInput{Reference: Reference{path, digest(b)}, Mode: mode})
	}
	r.Identity = identity(*r)
	return r, nil
}

func readSharedReattestationLineage(root string, ref Reference, want *SharedRepresentationReattestation, v *structuredSuccessorValidation, active map[Reference]bool, depth int) (*SharedRepresentationReattestation, error) {
	if depth > 32 || active[ref] {
		return nil, fmt.Errorf("shared dependency re-attestation cycle/depth")
	}
	active[ref] = true
	defer delete(active, ref)
	var r SharedRepresentationReattestation
	if err := v.guard.readReference(root, ref, &r); err != nil {
		return nil, err
	}
	c := r
	c.Identity = ""
	changes, err := sharedDependencyChanges(r.OldInputs, r.CurrentInputs)
	if err != nil || len(changes) == 0 || !reflect.DeepEqual(changes, r.Changes) || len(r.Added) != 0 || len(r.Removed) != 0 || r.Schema != SharedReattestationSchema || r.Identity != identity(c) || sharedReattestationReceipt(&r) != ref || r.BasePointer != want.BasePointer || r.BaseReceipt != want.BaseReceipt || r.BaseIdentity != want.BaseIdentity || r.RepresentationIdentity != want.RepresentationIdentity || !reflect.DeepEqual(r.Base, want.Base) || !reflect.DeepEqual(r.OldInputs, want.OldInputs) {
		return nil, fmt.Errorf("invalid shared dependency re-attestation lineage/invariant")
	}
	if r.Previous != nil {
		if _, err := readSharedReattestationLineage(root, *r.Previous, want, v, active, depth+1); err != nil {
			return nil, err
		}
	}
	return &r, nil
}

func currentSharedRepresentationDependencies(root string, base SharedRepresentationSuccessor, current []ShellRepresentationInput, v *structuredSuccessorValidation) error {
	changes, err := sharedDependencyChanges(base.Inputs, current)
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		return nil
	}
	ref := sharedRepresentationReference(base)
	head, sha, err := readSharedRepresentationHead(root, base.Request.Package, base.Request.AssetID, v)
	if err != nil {
		return err
	}
	if head == nil || *head != ref {
		return fmt.Errorf("shared dependency base head changed")
	}
	want, err := buildSharedReattestation(root, Reference{sharedRepresentationHead(base.Request.Package, base.Request.AssetID), sha}, ref, &base, current, nil, v)
	if err != nil {
		return err
	}
	additive, _, err := readShellReattestationHead(root, sharedReattestationPath(base.Identity), v)
	if err != nil {
		return err
	}
	if additive == nil {
		return fmt.Errorf("shared representation dependencies stale; exact shared dependency re-attestation required: %s", changes[0].Current.Reference.Path)
	}
	r, err := readSharedReattestationLineage(root, *additive, want, v, map[Reference]bool{}, 0)
	if err != nil {
		return err
	}
	want.Previous = r.Previous
	want.Identity = ""
	want.Identity = identity(*want)
	if !sameSharedReattestationCurrentness(*want, *r) {
		return fmt.Errorf("shared dependency re-attestation stale")
	}
	return nil
}

func planSharedReattestationAt(root string, g *Global, ref Reference, v *structuredSuccessorValidation, owned bool) (*SharedRepresentationReattestationPlan, error) {
	if err := sharedRepresentationReady(root, owned); err != nil {
		return nil, err
	}
	pointer, base, current, err := sharedReattestationBase(root, g, ref, v)
	if err != nil {
		return nil, err
	}
	path := sharedReattestationPath(base.Identity)
	prior, sha, err := readShellReattestationHead(root, path, v)
	if err != nil {
		return nil, err
	}
	r, err := buildSharedReattestation(root, pointer, ref, base, current, prior, v)
	if err != nil {
		return nil, err
	}
	if prior != nil {
		previous, err := readSharedReattestationLineage(root, *prior, r, v, map[Reference]bool{}, 0)
		if err != nil {
			return nil, err
		}
		probe := *r
		probe.Previous = previous.Previous
		probe.Identity = ""
		probe.Identity = identity(probe)
		if sameSharedReattestationCurrentness(probe, *previous) {
			return nil, fmt.Errorf("shared re-attestation already current; inspect original exact plan")
		}
	}
	p := &SharedRepresentationReattestationPlan{Schema: SharedReattestationSchema + "/plan", Authority: *r, Receipt: sharedReattestationReceipt(r), CurrentPath: path, PriorPointerSHA: sha, BaseCurrent: len(r.Changes) == 0, Eligible: len(r.Changes) > 0, State: "generation-required"}
	p.Identity = identity(*p)
	return p, nil
}

func PlanSharedRepresentationReattestation(root string, base Reference) (*SharedRepresentationReattestationPlan, error) {
	var p *SharedRepresentationReattestationPlan
	err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		g, err := loadSourceCurrent(root, v)
		if err != nil {
			return err
		}
		p, err = planSharedReattestationAt(root, g, base, v, false)
		return err
	})
	return p, err
}

// A Bundle transports the additive proof actually used, without changing the
// Document or representation identity. Historical decoding binds these bytes.
func sharedReattestationContextPaths(root string, base SharedRepresentationSuccessor, v *structuredSuccessorValidation) ([]string, error) {
	current, err := sharedRepresentationCurrentInputs(root, base, v)
	if err != nil {
		return nil, err
	}
	changes, err := sharedDependencyChanges(base.Inputs, current)
	if err != nil || len(changes) == 0 {
		return nil, err
	}
	if err := currentSharedRepresentationDependencies(root, base, current, v); err != nil {
		return nil, err
	}
	path := sharedReattestationPath(base.Identity)
	head, _, err := readShellReattestationHead(root, path, v)
	if err != nil || head == nil {
		return nil, fmt.Errorf("shared re-attestation context missing: %v", err)
	}
	paths := []string{path}
	for depth := 0; ; depth++ {
		if depth > 32 {
			return nil, fmt.Errorf("shared re-attestation context depth")
		}
		paths = append(paths, head.Path)
		var r SharedRepresentationReattestation
		if err := v.guard.readReference(root, *head, &r); err != nil {
			return nil, err
		}
		if r.Previous == nil {
			break
		}
		head = r.Previous
	}
	return paths, nil
}
