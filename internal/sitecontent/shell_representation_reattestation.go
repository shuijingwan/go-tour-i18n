package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
)

const ShellRepresentationReattestationSchema = "go-learning/shell-representation-dependency-reattestation/v1"
const ShellRepresentationReattestationRoot = "data/site-shell/representation-reattestations/"
const ShellRepresentationReattestationPending = "data/.shell-representation-reattestation-pending"

var shellReattestationValidationPaths = []string{
	"internal/sitecontent/shell_representation_reattestation.go",
	"internal/sitecontent/shell_representation_reattestation_publication.go",
	"internal/sitecontent/shell_representation_context.go",
	"docs/SHELL_REPRESENTATION_REATTESTATION.md",
}

type ShellRepresentationDependencyChange struct {
	Old     ShellRepresentationInput `json:"old"`
	Current ShellRepresentationInput `json:"current"`
}

// This authority binds dependency evolution, never representation evolution or
// language acceptance. Representation contains the complete slot/protection tree.
type ShellRepresentationReattestation struct {
	Schema           string                                `json:"schema"`
	BasePointer      Reference                             `json:"base_pointer"`
	BaseReceipt      Reference                             `json:"base_receipt"`
	BaseIdentity     string                                `json:"base_identity_sha256"`
	Basis            ShellRepresentationBasis              `json:"source_authority"`
	Representation   ShellRepresentation                   `json:"representation"`
	Released         []ShellTeachingRelease                `json:"released_teaching_bodies"`
	OldInputs        []ShellRepresentationInput            `json:"old_dependencies"`
	CurrentInputs    []ShellRepresentationInput            `json:"current_dependencies"`
	Changes          []ShellRepresentationDependencyChange `json:"changed_dependencies"`
	ValidationInputs []ShellRepresentationInput            `json:"validation_inputs"`
	Previous         *Reference                            `json:"previous,omitempty"`
	Identity         string                                `json:"identity_sha256"`
}

type ShellRepresentationReattestationPlan struct {
	Schema          string                           `json:"schema"`
	Authority       ShellRepresentationReattestation `json:"authority"`
	Receipt         Reference                        `json:"planned_receipt"`
	CurrentPath     string                           `json:"current_path"`
	PriorPointerSHA string                           `json:"prior_pointer_sha256"`
	BaseCurrent     bool                             `json:"base_receipt_current"`
	Eligible        bool                             `json:"reattestation_eligible"`
	State           string                           `json:"state"`
	Identity        string                           `json:"identity_sha256"`
}

type ShellRepresentationReattestationResult struct {
	Plan           *ShellRepresentationReattestationPlan `json:"plan"`
	AlreadyCurrent bool                                  `json:"already_current"`
	Applied        bool                                  `json:"applied"`
	Mutation       string                                `json:"mutation"`
}

func shellReattestationPath(base string) string {
	return ShellRepresentationReattestationRoot + "current/" + base + ".json"
}
func shellReattestationReceipt(r *ShellRepresentationReattestation) Reference {
	b, _ := Encode(r)
	return Reference{ShellRepresentationReattestationRoot + "receipts/" + r.Identity + ".json", digest(b)}
}

// Only the existing explicit parser/contract dependency set may evolve. Source,
// adoption, foundation, inventory, heads, modes and absence remain frozen.
func shellDependencyChanges(old, current []ShellRepresentationInput) ([]ShellRepresentationDependencyChange, error) {
	if len(old) != len(current) {
		return nil, fmt.Errorf("representation dependency path set changed")
	}
	changes := []ShellRepresentationDependencyChange{}
	for i, a := range old {
		b := current[i]
		if a.Reference.Path != b.Reference.Path || a.Mode != b.Mode || a.Absent != b.Absent {
			return nil, fmt.Errorf("representation dependency path/mode/absence changed")
		}
		if a == b {
			continue
		}
		allowed := false
		for _, path := range shellRepresentationParserPaths {
			allowed = allowed || path == a.Reference.Path
		}
		if !allowed || a.Absent || !validSHA(a.Reference.SHA256) || !validSHA(b.Reference.SHA256) {
			return nil, fmt.Errorf("non-parser representation authority drift: %s", a.Reference.Path)
		}
		changes = append(changes, ShellRepresentationDependencyChange{a, b})
	}
	return changes, nil
}

// Reconstruct the immutable base without declaring its old dependency bindings
// current. All non-dependency fields must match byte-for-byte deterministic
// reconstruction. The ordinary receipt checker retains its strict contract.
func reconstructShellRepresentationBase(root string, g *Global, ref Reference, v *structuredSuccessorValidation, active map[Reference]bool, depth int) (*ShellRepresentationSuccessor, []ShellRepresentationInput, error) {
	if depth > 32 || active[ref] {
		return nil, nil, fmt.Errorf("representation successor cycle/depth")
	}
	active[ref] = true
	defer delete(active, ref)
	var base ShellRepresentationSuccessor
	if err := v.guard.readReference(root, ref, &base); err != nil {
		return nil, nil, err
	}
	copy := base
	copy.Identity = ""
	if base.Schema != ShellRepresentationSchema || base.Identity != identity(copy) || shellRepresentationReceipt(&base) != ref {
		return nil, nil, fmt.Errorf("invalid representation base receipt")
	}
	var old ShellRepresentation
	if base.Predecessor != nil {
		prior, _, err := reconstructShellRepresentationBase(root, g, *base.Predecessor, v, active, depth+1)
		if err != nil {
			return nil, nil, err
		}
		old = prior.New
	} else {
		raw, err := v.guard.readRegular(root, base.Basis.Source.Path)
		if err != nil {
			return nil, nil, err
		}
		source, err := ExtractShellPageRepresentation(base.Basis.Source.Path, raw)
		if err != nil {
			return nil, nil, err
		}
		old = shellRepresentationValue(shellRepresentationLegacy, source)
	}
	if !reflect.DeepEqual(old, base.Old) || base.PredecessorIdentity != old.Identity {
		return nil, nil, fmt.Errorf("representation base predecessor discontinuity")
	}
	want, err := buildShellRepresentationAuthority(root, g, base.Predecessor, old, base.New.Parser, v)
	if err != nil {
		return nil, nil, err
	}
	current := want.Inputs
	want.Inputs, want.Identity = base.Inputs, base.Identity
	if !reflect.DeepEqual(*want, base) {
		return nil, nil, fmt.Errorf("representation/source/topology changed; dependency re-attestation forbidden")
	}
	if _, err := shellDependencyChanges(base.Inputs, current); err != nil {
		return nil, nil, err
	}
	return &base, current, nil
}

func readShellReattestationHead(root, path string, v *structuredSuccessorValidation) (*Reference, string, error) {
	if err := v.guard.optional(path); err != nil {
		return nil, "", err
	}
	if v.guard.absent[path] {
		return nil, digest(nil), nil
	}
	raw, err := v.guard.readRegular(root, path)
	if err != nil {
		return nil, "", err
	}
	var ref Reference
	if err := StrictJSON(raw, &ref); err != nil {
		return nil, "", err
	}
	canonical, _ := Encode(ref)
	if !bytes.Equal(canonical, raw) {
		return nil, "", fmt.Errorf("noncanonical dependency re-attestation pointer")
	}
	return &ref, digest(raw), nil
}

func shellReattestationBase(root string, g *Global, v *structuredSuccessorValidation) (Reference, Reference, *ShellRepresentationSuccessor, []ShellRepresentationInput, error) {
	raw, err := v.guard.readRegular(root, ShellRepresentationCurrent)
	if err != nil {
		return Reference{}, Reference{}, nil, nil, err
	}
	var ref Reference
	if err := StrictJSON(raw, &ref); err != nil {
		return Reference{}, Reference{}, nil, nil, err
	}
	canonical, _ := Encode(ref)
	if !bytes.Equal(canonical, raw) {
		return Reference{}, Reference{}, nil, nil, fmt.Errorf("noncanonical representation pointer")
	}
	base, current, err := reconstructShellRepresentationBase(root, g, ref, v, map[Reference]bool{}, 0)
	return Reference{ShellRepresentationCurrent, digest(raw)}, ref, base, current, err
}

func buildShellReattestation(root string, pointer, ref Reference, base *ShellRepresentationSuccessor, current []ShellRepresentationInput, previous *Reference, v *structuredSuccessorValidation) (*ShellRepresentationReattestation, error) {
	changes, err := shellDependencyChanges(base.Inputs, current)
	if err != nil {
		return nil, err
	}
	r := &ShellRepresentationReattestation{Schema: ShellRepresentationReattestationSchema, BasePointer: pointer, BaseReceipt: ref, BaseIdentity: base.Identity, Basis: base.Basis, Representation: base.New, Released: base.Released, OldInputs: base.Inputs, CurrentInputs: current, Changes: changes, Previous: previous, ValidationInputs: []ShellRepresentationInput{}}
	for _, path := range shellReattestationValidationPaths {
		b, err := v.guard.readRegular(root, path)
		if err != nil {
			return nil, err
		}
		st, err := os.Stat(pathJoin(root, path))
		if err != nil {
			return nil, err
		}
		r.ValidationInputs = append(r.ValidationInputs, ShellRepresentationInput{Reference: Reference{path, digest(b)}, Mode: uint32(st.Mode())})
	}
	r.Identity = identity(*r)
	return r, nil
}

// Historical re-attestation lineage is immutable and hash-bound; only its head
// must match current dependency bytes. Every node must bind the same base and
// exact representation and an admissible dependency delta.
func readShellReattestationLineage(root string, ref Reference, base *ShellRepresentationReattestation, v *structuredSuccessorValidation, active map[Reference]bool, depth int) (*ShellRepresentationReattestation, error) {
	if depth > 32 || active[ref] {
		return nil, fmt.Errorf("dependency re-attestation cycle/depth")
	}
	active[ref] = true
	defer delete(active, ref)
	var r ShellRepresentationReattestation
	if err := v.guard.readReference(root, ref, &r); err != nil {
		return nil, err
	}
	c := r
	c.Identity = ""
	changes, err := shellDependencyChanges(r.OldInputs, r.CurrentInputs)
	if err != nil || len(changes) == 0 || !reflect.DeepEqual(changes, r.Changes) || r.Schema != ShellRepresentationReattestationSchema || r.Identity != identity(c) || shellReattestationReceipt(&r) != ref ||
		r.BasePointer != base.BasePointer || r.BaseReceipt != base.BaseReceipt || r.BaseIdentity != base.BaseIdentity || !reflect.DeepEqual(r.Basis, base.Basis) || !reflect.DeepEqual(r.Representation, base.Representation) || !reflect.DeepEqual(r.Released, base.Released) || !reflect.DeepEqual(r.OldInputs, base.OldInputs) {
		return nil, fmt.Errorf("invalid dependency re-attestation lineage")
	}
	if r.Previous != nil {
		if _, err := readShellReattestationLineage(root, *r.Previous, base, v, active, depth+1); err != nil {
			return nil, err
		}
	}
	return &r, nil
}

func currentShellRepresentationDependencies(root string, g *Global, ref Reference, v *structuredSuccessorValidation) (*ShellRepresentationSuccessor, error) {
	pointer, actual, base, current, err := shellReattestationBase(root, g, v)
	if err != nil {
		return nil, err
	}
	if actual != ref {
		return nil, fmt.Errorf("representation pointer changed")
	}
	changes, err := shellDependencyChanges(base.Inputs, current)
	if err != nil {
		return nil, err
	}
	if len(changes) == 0 {
		return base, nil
	}
	want, err := buildShellReattestation(root, pointer, ref, base, current, nil, v)
	if err != nil {
		return nil, err
	}
	head, _, err := readShellReattestationHead(root, shellReattestationPath(base.Identity), v)
	if err != nil {
		return nil, err
	}
	if head == nil {
		return nil, fmt.Errorf("canonical representation dependencies stale; exact dependency re-attestation required")
	}
	r, err := readShellReattestationLineage(root, *head, want, v, map[Reference]bool{}, 0)
	if err != nil {
		return nil, err
	}
	want.Previous = r.Previous
	want.Identity = ""
	want.Identity = identity(*want)
	if !reflect.DeepEqual(*want, *r) {
		return nil, fmt.Errorf("dependency re-attestation is stale")
	}
	return base, nil
}

func planShellReattestationAt(root string, g *Global, v *structuredSuccessorValidation, ownedLock bool) (*ShellRepresentationReattestationPlan, error) {
	if err := shellRepresentationReady(root, ownedLock); err != nil {
		return nil, err
	}
	pointer, ref, base, current, err := shellReattestationBase(root, g, v)
	if err != nil {
		return nil, err
	}
	path := shellReattestationPath(base.Identity)
	prior, sha, err := readShellReattestationHead(root, path, v)
	if err != nil {
		return nil, err
	}
	r, err := buildShellReattestation(root, pointer, ref, base, current, prior, v)
	if err != nil {
		return nil, err
	}
	if prior != nil {
		previous, err := readShellReattestationLineage(root, *prior, r, v, map[Reference]bool{}, 0)
		if err != nil {
			return nil, err
		}
		probe := *r
		probe.Previous, probe.Identity = previous.Previous, ""
		probe.Identity = identity(probe)
		if reflect.DeepEqual(probe, *previous) {
			return nil, fmt.Errorf("dependency re-attestation already current; inspect the original exact plan")
		}
	}
	p := &ShellRepresentationReattestationPlan{Schema: ShellRepresentationReattestationSchema + "/plan", Authority: *r, Receipt: shellReattestationReceipt(r), CurrentPath: path, PriorPointerSHA: sha, BaseCurrent: len(r.Changes) == 0, Eligible: len(r.Changes) > 0, State: "generation-required"}
	p.Identity = identity(*p)
	return p, nil
}

func PlanShellRepresentationReattestation(root string) (*ShellRepresentationReattestationPlan, error) {
	var p *ShellRepresentationReattestationPlan
	err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		g, err := loadSourceCurrent(root, v)
		if err != nil {
			return err
		}
		p, err = planShellReattestationAt(root, g, v, false)
		return err
	})
	return p, err
}

func shellReattestationContextReferences(root string, r *ShellRepresentationSuccessor) ([]Reference, error) {
	var refs []Reference
	err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		g, err := loadSourceCurrent(root, v)
		if err != nil {
			return err
		}
		refs, err = shellReattestationContextReferencesAt(root, g, r, v)
		return err
	})
	return refs, err
}

func shellReattestationContextReferencesAt(root string, g *Global, r *ShellRepresentationSuccessor, v *structuredSuccessorValidation) ([]Reference, error) {
	_, ref, base, current, err := shellReattestationBase(root, g, v)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(*base, *r) {
		return nil, fmt.Errorf("representation context base changed")
	}
	changes, err := shellDependencyChanges(base.Inputs, current)
	if err != nil || len(changes) == 0 {
		return nil, err // Unused additive authorities remain inert.
	}
	if _, err := currentShellRepresentationDependencies(root, g, ref, v); err != nil {
		return nil, err
	}
	path := shellReattestationPath(r.Identity)
	head, sha, err := readShellReattestationHead(root, path, v)
	if err != nil || head == nil {
		return nil, fmt.Errorf("missing dependency re-attestation context: %v", err)
	}
	return []Reference{{path, sha}, *head}, nil
}
