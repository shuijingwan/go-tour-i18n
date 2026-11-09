package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"sort"
)

const ShellRepresentationSchema = "go-learning/shell-canonical-representation-successor/v1"
const ShellRepresentationPlanSchema = ShellRepresentationSchema + "/plan"
const ShellRepresentationRoot = "data/site-shell/representation-successors/"
const ShellRepresentationCurrent = ShellRepresentationRoot + "current.json"
const ShellRepresentationPending = "data/.shell-representation-successor-pending"
const shellRepresentationLegacy = "go-learning/shell-page-legacy-representation/v1"

// Canonical representation is global across locales. No translated bytes or
// locale acceptance are inputs to this source-side authority.
type ShellRepresentationBasis struct {
	LocaleScope           string      `json:"locale_scope"`
	HistoricalSnapshotSHA string      `json:"historical_snapshot_sha256"`
	Package               Package     `json:"source_package"`
	Source                Reference   `json:"canonical_raw_source"`
	Adoption              Reference   `json:"shell_adoption"`
	Foundation            Reference   `json:"shell_foundation"`
	SourceHeads           []Reference `json:"source_heads"`
}
type ShellRepresentation struct {
	Parser            string              `json:"parser_contract"`
	Source            ShellLanguageSource `json:"representation"`
	ProtectedInputSHA string              `json:"protected_input_sha256"`
	Identity          string              `json:"identity_sha256"`
}
type ShellRepresentationInput struct {
	Reference Reference `json:"reference"`
	Mode      uint32    `json:"mode"`
	Absent    bool      `json:"absent,omitempty"`
}
type ShellTeachingRelease struct {
	Start     int    `json:"start"`
	End       int    `json:"end"`
	SourceSHA string `json:"source_sha256"`
	Role      string `json:"role"`
}
type ShellRepresentationSuccessor struct {
	Schema              string                     `json:"schema"`
	Basis               ShellRepresentationBasis   `json:"source_authority"`
	Predecessor         *Reference                 `json:"predecessor,omitempty"`
	PredecessorIdentity string                     `json:"predecessor_representation_identity_sha256"`
	Old                 ShellRepresentation        `json:"old"`
	New                 ShellRepresentation        `json:"new"`
	Released            []ShellTeachingRelease     `json:"released_teaching_bodies"`
	Inputs              []ShellRepresentationInput `json:"currentness_inputs"`
	Identity            string                     `json:"identity_sha256"`
}
type ShellRepresentationPlan struct {
	Schema            string                       `json:"schema"`
	Authority         ShellRepresentationSuccessor `json:"authority"`
	Receipt           Reference                    `json:"planned_receipt"`
	PriorIdentity     string                       `json:"prior_identity_sha256"`
	PriorPointerSHA   string                       `json:"prior_pointer_sha256"`
	RawSourceChanged  bool                         `json:"raw_source_changed"`
	AffectedPackage   string                       `json:"affected_package"`
	AffectedRoute     string                       `json:"affected_route"`
	AffectedPageCount int                          `json:"affected_page_count"`
	State             string                       `json:"state"`
	Identity          string                       `json:"identity_sha256"`
}
type ShellRepresentationResult struct {
	Plan           *ShellRepresentationPlan `json:"plan"`
	AlreadyCurrent bool                     `json:"already_current"`
	Applied        bool                     `json:"applied"`
	Mutation       string                   `json:"mutation"`
}

var shellRepresentationParserPaths = []string{
	repositoryModeImplementation,
	"internal/sitecontent/shell_representation_successor.go", "docs/SHELL_REPRESENTATION_SUCCESSOR.md",
	"internal/sitecontent/shell_representation_parser.go", "internal/i18n/semantic_teaching_body.go",
	"internal/sitecontent/shell_inline.go", "internal/sitecontent/shell_language_source.go",
	"internal/sitecontent/shell_reconstruction.go", "internal/sitecontent/formal_page.go",
	"internal/sitecontent/linguistic.go", "internal/sitecontent/teaching.go",
	"internal/i18n/preformatted.go", "internal/i18n/comment_inventory.go", "internal/i18n/go_example_protect.go",
}

func shellRepresentationValue(parser string, s *ShellLanguageSource) ShellRepresentation {
	v := ShellRepresentation{Parser: parser, Source: *s, ProtectedInputSHA: digest([]byte(s.Page.Units[0].Source))}
	v.Identity = identity(v)
	return v
}
func shellRepresentationReceipt(r *ShellRepresentationSuccessor) Reference {
	b, _ := Encode(r)
	return Reference{ShellRepresentationRoot + "receipts/" + r.Identity + ".json", digest(b)}
}
func shellRepresentationReady(root string, ownedLock bool) error {
	for _, name := range []string{ShellRepresentationPending, ShellRepresentationReattestationPending, StructuredSourceReplacementPending, "data/.corpus-successor-pending"} {
		if _, err := os.Lstat(pathJoin(root, name)); !os.IsNotExist(err) {
			return fmt.Errorf("mutation=unknown: pending publication %s", name)
		}
	}
	if !ownedLock {
		if _, err := os.Lstat(pathJoin(root, "data/.glossary-corpus.lock")); !os.IsNotExist(err) {
			return fmt.Errorf("publication lock held; inspect before proceeding")
		}
	}
	return nil
}
func shellRepresentationBasis(root string, g *Global, v *structuredSuccessorValidation) (ShellRepresentationBasis, error) {
	for _, dir := range []string{StructuredSourceSuccessorRoot + "current", PagePackageAdoptionRoot + "current"} {
		if err := v.guard.directory(dir); err != nil {
			return ShellRepresentationBasis{}, err
		}
	}
	if g.ShellAdoption == nil {
		return ShellRepresentationBasis{}, fmt.Errorf("current Shell adoption required")
	}
	var pointer Reference
	b, err := v.guard.readRegular(root, ShellAdoptionCurrent)
	if err != nil {
		return ShellRepresentationBasis{}, err
	}
	if err := StrictJSON(b, &pointer); err != nil || pointer != *g.ShellAdoption {
		return ShellRepresentationBasis{}, fmt.Errorf("Shell source pointer mismatch")
	}
	var a ShellAdoption
	if err := v.guard.readReference(root, pointer, &a); err != nil {
		return ShellRepresentationBasis{}, err
	}
	copy := a
	copy.Identity = ""
	if a.Schema != ShellAdoptionSchema || a.Identity != identity(copy) || pointer.Path != shellAdoptionPath(&a) {
		return ShellRepresentationBasis{}, fmt.Errorf("invalid Shell adoption authority")
	}
	var f ShellSuccessorAuthority
	if err := v.guard.readReference(root, a.Foundation, &f); err != nil {
		return ShellRepresentationBasis{}, err
	}
	fc := f
	fc.Identity = ""
	if f.Schema != ShellSuccessorSchema || f.Identity != identity(fc) || f.Package != "site-v2-shell" || f.ExtractionContract != ShellExtractionContract {
		return ShellRepresentationBasis{}, fmt.Errorf("invalid Shell foundation authority")
	}
	p, err := packageByID(g, "site-v2-shell")
	if err != nil {
		return ShellRepresentationBasis{}, err
	}
	var rawRef Reference
	for _, s := range g.Sources {
		if s.Package == p.ID && s.Surface == "homepage" && s.Route == "/" && s.Kind == "page" {
			if rawRef.Path != "" {
				return ShellRepresentationBasis{}, fmt.Errorf("ambiguous Homepage source")
			}
			rawRef = Reference{s.Path, s.SHA256}
		}
	}
	if rawRef.Path != "_content/index.md" || !validSHA(rawRef.SHA256) {
		return ShellRepresentationBasis{}, fmt.Errorf("canonical Homepage source required")
	}
	if err := v.guard.bind(rawRef); err != nil {
		return ShellRepresentationBasis{}, err
	}
	found := false
	for _, s := range f.Sources {
		if s.Path == rawRef.Path && s.SourceSHA == rawRef.SHA256 && s.Role == "homepage-page" {
			found = true
		}
	}
	if !found {
		return ShellRepresentationBasis{}, fmt.Errorf("raw source not bound by Shell foundation")
	}
	return ShellRepresentationBasis{LocaleScope: "all-locales", HistoricalSnapshotSHA: g.SnapshotSHA256, Package: p, Source: rawRef, Adoption: pointer, Foundation: a.Foundation, SourceHeads: append([]Reference(nil), g.StructuredSuccessors...)}, nil
}
func shellRepresentationInputs(root string, b ShellRepresentationBasis, v *structuredSuccessorValidation, frozen ...[]ShellRepresentationInput) ([]ShellRepresentationInput, error) {
	paths := append([]string{}, shellRepresentationParserPaths...)
	// One explicit historical inventory boundary: old bases predate this
	// dependency. Preserve their exact path set; additive validation_inputs
	// always bind the current mode implementation. New bases bind it directly.
	modeDependency := len(frozen) == 0
	if len(frozen) > 0 {
		for _, in := range frozen[0] {
			modeDependency = modeDependency || in.Reference.Path == repositoryModeImplementation
		}
	}
	if modeDependency {
		paths = append(paths, repositoryModeImplementation)
	} else {
		for i, path := range paths {
			if path == repositoryModeImplementation {
				paths = append(paths[:i], paths[i+1:]...)
				break
			}
		}
	}
	paths = append(paths, b.Source.Path, ShellAdoptionCurrent, b.Adoption.Path, b.Foundation.Path, GlobalPath, SourceManifestPath, SnapshotPath)
	for _, r := range b.SourceHeads {
		paths = append(paths, r.Path)
	}
	for _, d := range b.Package.Dependencies {
		paths = append(paths, d.Path)
	}
	var foundation ShellSuccessorAuthority
	if err := v.guard.readReference(root, b.Foundation, &foundation); err != nil {
		return nil, err
	}
	for _, ref := range foundation.ParserReferences {
		paths = append(paths, ref.Path)
	}
	for _, dir := range []string{StructuredSourceSuccessorRoot + "current", PagePackageAdoptionRoot + "current"} {
		entries, err := os.ReadDir(pathJoin(root, dir))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			paths = append(paths, dir+"/"+entry.Name())
		}
	}
	sort.Strings(paths)
	var out []ShellRepresentationInput
	for i, name := range paths {
		if i > 0 && name == paths[i-1] {
			continue
		}
		if name == SnapshotPath {
			if err := v.guard.optional(name); err != nil {
				return nil, err
			}
			if v.guard.absent[name] {
				out = append(out, ShellRepresentationInput{Reference: Reference{name, digest(nil)}, Absent: true})
				continue
			}
		}
		raw, err := v.guard.readRegular(root, name)
		if err != nil {
			return nil, err
		}
		st, err := os.Stat(pathJoin(root, name))
		if err != nil {
			return nil, err
		}
		mode, err := representationFileMode(st.Mode())
		if err != nil {
			return nil, err
		}
		out = append(out, ShellRepresentationInput{Reference: Reference{name, digest(raw)}, Mode: mode})
	}
	return out, nil
}
func buildShellRepresentationAuthority(root string, g *Global, prior *Reference, old ShellRepresentation, rule string, v *structuredSuccessorValidation, frozen ...[]ShellRepresentationInput) (*ShellRepresentationSuccessor, error) {
	basis, err := shellRepresentationBasis(root, g, v)
	if err != nil {
		return nil, err
	}
	raw, err := v.guard.readRegular(root, basis.Source.Path)
	if err != nil {
		return nil, err
	}
	inputs, err := shellRepresentationInputs(root, basis, v, frozen...)
	if err != nil {
		return nil, err
	}
	if prior != nil {
		if err := v.guard.bind(*prior); err != nil {
			return nil, err
		}
		st, err := os.Stat(pathJoin(root, prior.Path))
		if err != nil {
			return nil, err
		}
		mode, err := representationFileMode(st.Mode())
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, ShellRepresentationInput{Reference: *prior, Mode: mode})
		sort.Slice(inputs, func(i, j int) bool { return inputs[i].Reference.Path < inputs[j].Reference.Path })
	}
	next, err := prospectiveShellRepresentation(basis.Source.Path, raw, rule)
	if err != nil {
		return nil, err
	}
	newRep := shellRepresentationValue(rule, next)
	if old.Source.Path != basis.Source.Path || old.Source.SourceSHA != basis.Source.SHA256 || old.Source.Page == nil || old.Source.Page.SourceSHA != basis.Source.SHA256 {
		return nil, fmt.Errorf("same raw source/predecessor required")
	}
	if old.Source.Page.Units[0].ID == next.Page.Units[0].ID || old.ProtectedInputSHA == newRep.ProtectedInputSHA {
		return nil, fmt.Errorf("canonical representation successor must change protected topology")
	}
	releases := []ShellTeachingRelease{}
	for _, n := range next.Slots {
		found := false
		for _, o := range old.Source.Slots {
			found = found || reflect.DeepEqual(n, o)
		}
		if !found {
			if n.Role != "teaching-comment" || n.Encoding != "go-comment-interior" {
				return nil, fmt.Errorf("unbounded representation change")
			}
			releases = append(releases, ShellTeachingRelease{n.Unit.Start, n.Unit.End, digest(raw[n.Unit.Start:n.Unit.End]), n.Role})
		}
	}
	if len(releases) == 0 {
		return nil, fmt.Errorf("no bounded teaching release")
	}
	r := &ShellRepresentationSuccessor{Schema: ShellRepresentationSchema, Basis: basis, Predecessor: prior, PredecessorIdentity: old.Identity, Old: old, New: newRep, Released: releases, Inputs: inputs}
	r.Identity = identity(*r)
	return r, nil
}

// Frozen lineage is exact; an unrelated semantic-review/reopen receipt has a
// different schema and cannot enter this verifier. No target equivalence exists.
func checkShellRepresentation(root string, g *Global, ref Reference, v *structuredSuccessorValidation, active map[Reference]bool, depth int) (*ShellRepresentationSuccessor, error) {
	if depth > 32 || active[ref] {
		return nil, fmt.Errorf("representation successor cycle/depth")
	}
	active[ref] = true
	defer delete(active, ref)
	var r ShellRepresentationSuccessor
	if err := v.guard.readReference(root, ref, &r); err != nil {
		return nil, err
	}
	copy := r
	copy.Identity = ""
	if r.Schema != ShellRepresentationSchema || r.Identity != identity(copy) || shellRepresentationReceipt(&r) != ref {
		return nil, fmt.Errorf("invalid canonical representation receipt")
	}
	var old ShellRepresentation
	if r.Predecessor != nil {
		previous, err := checkShellRepresentation(root, g, *r.Predecessor, v, active, depth+1)
		if err != nil {
			return nil, err
		}
		old = previous.New
	} else {
		raw, err := v.guard.readRegular(root, r.Basis.Source.Path)
		if err != nil {
			return nil, err
		}
		s, err := ExtractShellPageRepresentation(r.Basis.Source.Path, raw)
		if err != nil {
			return nil, err
		}
		old = shellRepresentationValue(shellRepresentationLegacy, s)
	}
	if !reflect.DeepEqual(old, r.Old) || r.PredecessorIdentity != old.Identity {
		return nil, fmt.Errorf("representation predecessor discontinuity")
	}
	want, err := buildShellRepresentationAuthority(root, g, r.Predecessor, old, r.New.Parser, v, r.Inputs)
	if err != nil {
		return nil, err
	}
	if !sameRepresentationInputs(want.Inputs, r.Inputs) {
		return nil, fmt.Errorf("canonical representation/source/dependencies stale")
	}
	want.Inputs, want.Identity = r.Inputs, r.Identity
	if !reflect.DeepEqual(*want, r) {
		return nil, fmt.Errorf("canonical representation/source/dependencies stale")
	}
	return &r, nil
}
func currentShellRepresentation(root string, g *Global, v *structuredSuccessorValidation, ownedLock bool) (*ShellRepresentationSuccessor, *Reference, error) {
	if err := shellRepresentationReady(root, ownedLock); err != nil {
		return nil, nil, err
	}
	if err := v.guard.optional(ShellRepresentationCurrent); err != nil {
		return nil, nil, err
	}
	raw, err := readRegular(root, ShellRepresentationCurrent)
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	var ref Reference
	if err := StrictJSON(raw, &ref); err != nil {
		return nil, nil, err
	}
	canonical, _ := Encode(ref)
	if !bytes.Equal(raw, canonical) {
		return nil, nil, fmt.Errorf("noncanonical representation current pointer")
	}
	r, err := currentShellRepresentationDependencies(root, g, ref, v)
	return r, &ref, err
}
func shellRepresentationPlan(r *ShellRepresentationSuccessor, priorSHA string) *ShellRepresentationPlan {
	p := &ShellRepresentationPlan{Schema: ShellRepresentationPlanSchema, Authority: *r, Receipt: shellRepresentationReceipt(r), PriorIdentity: r.Old.Identity, PriorPointerSHA: priorSHA, AffectedPackage: r.Basis.Package.ID, AffectedRoute: "/", AffectedPageCount: 1, State: "generation-required"}
	p.Identity = identity(*p)
	return p
}
func planShellRepresentationAt(root string, g *Global, rule string, v *structuredSuccessorValidation, ownedLock bool) (*ShellRepresentationPlan, error) {
	current, ref, err := currentShellRepresentation(root, g, v, ownedLock)
	if err != nil {
		return nil, err
	}
	if current != nil && current.New.Parser == rule {
		return nil, fmt.Errorf("representation already current; inspect the original exact plan for idempotent replay")
	}
	var old ShellRepresentation
	priorSHA := digest(nil)
	if current != nil {
		old = current.New
		b, _ := Encode(ref)
		priorSHA = digest(b)
	} else {
		basis, err := shellRepresentationBasis(root, g, v)
		if err != nil {
			return nil, err
		}
		raw, err := v.guard.readRegular(root, basis.Source.Path)
		if err != nil {
			return nil, err
		}
		s, err := ExtractShellPageRepresentation(basis.Source.Path, raw)
		if err != nil {
			return nil, err
		}
		old = shellRepresentationValue(shellRepresentationLegacy, s)
	}
	r, err := buildShellRepresentationAuthority(root, g, ref, old, rule, v)
	if err != nil {
		return nil, err
	}
	return shellRepresentationPlan(r, priorSHA), nil
}
func PlanShellRepresentationSuccessor(root, rule string) (*ShellRepresentationPlan, error) {
	var p *ShellRepresentationPlan
	err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		g, err := loadSourceCurrent(root, v)
		if err != nil {
			return err
		}
		p, err = planShellRepresentationAt(root, g, rule, v, false)
		return err
	})
	return p, err
}
func inspectShellRepresentationAt(root string, g *Global, input Reference, v *structuredSuccessorValidation, ownedLock bool) (*ShellRepresentationResult, error) {
	if err := shellRepresentationReady(root, ownedLock); err != nil {
		return nil, err
	}
	var p ShellRepresentationPlan
	if err := v.guard.readReference(root, input, &p); err != nil {
		return nil, err
	}
	copy := p
	copy.Identity = ""
	if p.Schema != ShellRepresentationPlanSchema || p.Identity != identity(copy) || p.RawSourceChanged || p.AffectedPackage != "site-v2-shell" || p.AffectedRoute != "/" || p.AffectedPageCount != 1 || p.State != "generation-required" {
		return nil, fmt.Errorf("invalid representation plan")
	}
	// Reconstruct from the exact predecessor rather than treating a new parser
	// result, semantic approval, or a caller-supplied representation as authority.
	var old ShellRepresentation
	if p.Authority.Predecessor != nil {
		r, err := checkShellRepresentation(root, g, *p.Authority.Predecessor, v, map[Reference]bool{}, 0)
		if err != nil {
			return nil, err
		}
		old = r.New
	} else {
		basis, err := shellRepresentationBasis(root, g, v)
		if err != nil {
			return nil, err
		}
		raw, err := v.guard.readRegular(root, basis.Source.Path)
		if err != nil {
			return nil, err
		}
		s, err := ExtractShellPageRepresentation(basis.Source.Path, raw)
		if err != nil {
			return nil, err
		}
		old = shellRepresentationValue(shellRepresentationLegacy, s)
	}
	r, err := buildShellRepresentationAuthority(root, g, p.Authority.Predecessor, old, p.Authority.New.Parser, v)
	if err != nil {
		return nil, err
	}
	priorSHA := digest(nil)
	if r.Predecessor != nil {
		b, _ := Encode(r.Predecessor)
		priorSHA = digest(b)
	}
	expected := shellRepresentationPlan(r, priorSHA)
	if !reflect.DeepEqual(*expected, p) {
		return nil, fmt.Errorf("representation plan/predecessor/parser/dependencies mismatch")
	}
	current, ref, err := currentShellRepresentation(root, g, v, ownedLock)
	if err != nil {
		return nil, err
	}
	if ref != nil && *ref == p.Receipt {
		return &ShellRepresentationResult{Plan: &p, AlreadyCurrent: true, Mutation: "none"}, nil
	}
	if (ref == nil) != (r.Predecessor == nil) || (ref != nil && *ref != *r.Predecessor) {
		return nil, fmt.Errorf("stale representation prior pointer")
	}
	if current != nil && current.New.Identity != p.PriorIdentity {
		return nil, fmt.Errorf("stale representation predecessor identity")
	}
	return &ShellRepresentationResult{Plan: &p, Mutation: "none"}, nil
}
func InspectShellRepresentationSuccessor(root string, input Reference) (*ShellRepresentationResult, error) {
	var result *ShellRepresentationResult
	err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		g, err := loadSourceCurrent(root, v)
		if err != nil {
			return err
		}
		result, err = inspectShellRepresentationAt(root, g, input, v, false)
		return err
	})
	return result, err
}

// The public publisher always reloads source authority and verifies the exact
// reviewed plan and CAS. Tests use the same transaction with an isolated source
// loader, never a validation bypass in a production reader.
func ApplyShellRepresentationSuccessor(root string, input Reference, priorIdentity, priorSHA string) (*ShellRepresentationResult, error) {
	return applyShellRepresentationSuccessor(root, input, priorIdentity, priorSHA, func(v *structuredSuccessorValidation) (*Global, error) { return loadSourceCurrent(root, v) }, nil)
}
func applyShellRepresentationSuccessor(root string, input Reference, priorIdentity, priorSHA string, load func(*structuredSuccessorValidation) (*Global, error), beforeCommit func()) (*ShellRepresentationResult, error) {
	if err := shellRepresentationReady(root, false); err != nil {
		return nil, err
	}
	lockName := pathJoin(root, "data/.glossary-corpus.lock")
	lock, err := os.OpenFile(lockName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("publication lock held: %w", err)
	}
	if err := lock.Close(); err != nil {
		return nil, fmt.Errorf("mutation=unknown: %w", err)
	}
	pending := false
	defer func() {
		if !pending {
			_ = os.Remove(lockName)
		}
	}()
	v := successorValidation(root, nil)
	v.readonlyOwner = true
	defer func() { v.readonlyOwner = false }()
	g, err := load(v)
	if err != nil {
		return nil, err
	}
	result, err := inspectShellRepresentationAt(root, g, input, v, true)
	if err != nil {
		return nil, err
	}
	p := result.Plan
	if priorIdentity != p.PriorIdentity || priorSHA != p.PriorPointerSHA {
		return nil, fmt.Errorf("exact representation predecessor CAS required")
	}
	if result.AlreadyCurrent {
		if err := v.aggregateVerify(); err != nil {
			return nil, err
		}
		return result, nil
	}
	if beforeCommit != nil {
		beforeCommit()
	}
	// A second independent graph rejects changes after preflight, including
	// changes to source, dependencies, mode, plan or publication pointer.
	fresh := successorValidation(root, nil)
	fresh.readonlyOwner = true
	freshG, err := load(fresh)
	if err != nil {
		return nil, err
	}
	again, err := inspectShellRepresentationAt(root, freshG, input, fresh, true)
	if err != nil || !reflect.DeepEqual(result, again) {
		return nil, fmt.Errorf("concurrent representation input mutation: %v", err)
	}
	if err := v.aggregateVerify(); err != nil {
		return nil, err
	}
	if err := fresh.aggregateVerify(); err != nil {
		return nil, err
	}
	if err := saveImmutable(root, p.Receipt.Path, &p.Authority); err != nil {
		return nil, err
	}
	// Make the immutable receipt directory durable before publishing a pointer
	// that depends on it. An uncertain I/O result leaves the owned lock inert.
	for _, dir := range []string{ShellRepresentationRoot + "receipts", ShellRepresentationRoot, "data/site-shell"} {
		if err := syncSourceDirectory(root, dir); err != nil {
			pending = true
			return nil, fmt.Errorf("mutation=unknown: immutable receipt durability: %w", err)
		}
	}
	next, _ := Encode(p.Receipt)
	f, err := os.CreateTemp(pathJoin(root, ShellRepresentationRoot), ".representation-current-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f.Name())
	if err := f.Chmod(0644); err != nil {
		f.Close()
		return nil, err
	}
	if _, err := f.Write(next); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	marker, _ := Encode(struct {
		Plan   Reference `json:"plan"`
		Prior  string    `json:"prior_pointer_sha256"`
		Result Reference `json:"result"`
	}{input, priorSHA, p.Receipt})
	if err := SaveArtifact(root, ShellRepresentationPending, marker); err != nil {
		pending = true
		return nil, fmt.Errorf("mutation=unknown: %w", err)
	}
	pending = true
	if err := syncSourceDirectory(root, "data"); err != nil {
		return nil, fmt.Errorf("mutation=unknown: %w", err)
	}
	if err := v.aggregateVerify(); err != nil {
		return nil, fmt.Errorf("mutation=unknown: %w", err)
	}
	raw, err := readRegular(root, ShellRepresentationCurrent)
	if os.IsNotExist(err) {
		raw = nil
		err = nil
	}
	if err != nil || digest(raw) != priorSHA {
		return nil, fmt.Errorf("mutation=unknown: representation pointer CAS drift")
	}
	if err := os.Rename(f.Name(), pathJoin(root, ShellRepresentationCurrent)); err != nil {
		return nil, fmt.Errorf("mutation=unknown: %w", err)
	}
	if err := syncSourceDirectory(root, ShellRepresentationRoot); err != nil {
		return nil, fmt.Errorf("mutation=unknown: %w", err)
	}
	installed, err := readRegular(root, ShellRepresentationCurrent)
	if err != nil || !bytes.Equal(installed, next) {
		return nil, fmt.Errorf("mutation=unknown: representation pointer mismatch")
	}
	if err := os.Remove(pathJoin(root, ShellRepresentationPending)); err != nil {
		return nil, fmt.Errorf("mutation=unknown: %w", err)
	}
	if err := syncSourceDirectory(root, "data"); err != nil {
		return nil, fmt.Errorf("mutation=unknown: %w", err)
	}
	pending = false
	result.Applied = true
	result.Mutation = "known"
	return result, nil
}

func representationPackage(source Package, ref Reference) Package {
	p := source
	p.Dependencies = append(append([]Dependency{}, source.Dependencies...), Dependency{Kind: "contract", Path: ref.Path, SHA256: ref.SHA256})
	p.Identity = ""
	p.Identity = identity(p)
	return p
}
func applyShellRepresentationAuthority(root string, g *Global, contexts ...*structuredSuccessorValidation) (*Global, error) {
	if _, err := os.Lstat(pathJoin(root, ShellRepresentationPending)); !os.IsNotExist(err) {
		return nil, fmt.Errorf("mutation=unknown: representation publication pending")
	}
	if _, err := os.Lstat(pathJoin(root, ShellRepresentationCurrent)); os.IsNotExist(err) {
		if len(contexts) > 0 && contexts[0] != nil {
			if err := contexts[0].guard.optional(ShellRepresentationCurrent); err != nil {
				return nil, err
			}
		}
		return g, nil
	} else if err != nil {
		return nil, err
	}
	v := successorValidation(root, contexts)
	r, ref, err := currentShellRepresentation(root, g, v, false)
	if err != nil || r == nil {
		return g, err
	}
	next := *g
	next.Packages = append([]Package{}, g.Packages...)
	next.HistoricalPackages = append([]Package{}, g.HistoricalPackages...)
	for i, p := range next.Packages {
		if p.ID == r.Basis.Package.ID {
			next.HistoricalPackages = append(next.HistoricalPackages, p)
			next.Packages[i] = representationPackage(p, *ref)
		}
	}
	next.ShellRepresentation = ref
	next.Identity = ""
	next.Identity = identity(next)
	if err := v.verifyObserved(); err != nil {
		return nil, err
	}
	return &next, ValidateGlobal(next)
}
func selectShellRepresentation(root string, raw []byte, contexts ...*structuredSuccessorValidation) (*ShellLanguageSource, *ShellRepresentationSuccessor, error) {
	if _, err := os.Lstat(pathJoin(root, ShellRepresentationPending)); !os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("mutation=unknown: representation publication pending")
	}
	if _, err := os.Lstat(pathJoin(root, ShellRepresentationCurrent)); os.IsNotExist(err) {
		s, err := ExtractShellPageRepresentation("_content/index.md", raw)
		return s, nil, err
	} else if err != nil {
		return nil, nil, err
	}
	v := successorValidation(root, contexts)
	g, err := loadSourceCurrent(root, v)
	if err != nil {
		return nil, nil, err
	}
	return selectShellRepresentationAt(root, g, raw, v)
}

func selectShellRepresentationAt(root string, g *Global, raw []byte, v *structuredSuccessorValidation) (*ShellLanguageSource, *ShellRepresentationSuccessor, error) {
	r, _, err := currentShellRepresentation(root, g, v, false)
	if err != nil {
		return nil, nil, err
	}
	if r == nil {
		s, err := ExtractShellPageRepresentation("_content/index.md", raw)
		return s, nil, err
	}
	if digest(raw) != r.Basis.Source.SHA256 {
		return nil, nil, fmt.Errorf("canonical representation raw source mismatch")
	}
	if err := v.verifyObserved(); err != nil {
		return nil, nil, err
	}
	b, _ := Encode(r.New.Source)
	var copy ShellLanguageSource
	if err := StrictJSON(b, &copy); err != nil {
		return nil, nil, err
	}
	return &copy, r, nil
}

// Keep an inert orphan receipt separate from the sole current pointer. A check
// never scans receipts to infer "latest", nor adopts semantic-review evidence.
func CheckShellRepresentationSuccessor(root string, ref Reference) (*ShellRepresentationSuccessor, error) {
	var r *ShellRepresentationSuccessor
	err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		if err := shellRepresentationReady(root, false); err != nil {
			return err
		}
		g, err := loadSourceCurrent(root, v)
		if err != nil {
			return err
		}
		r, err = checkShellRepresentation(root, g, ref, v, map[Reference]bool{}, 0)
		return err
	})
	return r, err
}
