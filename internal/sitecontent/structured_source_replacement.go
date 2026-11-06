package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"reflect"
	"sort"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const StructuredSourceReplacementSchema = "go-learning/consumed-structured-source-replacement/v1"
const StructuredSourceReplacementPending = "data/.structured-source-replacement-pending"

// Replaced pairs are source-owner declarations of retirement/replacement,
// never a claim of target equivalence. Only exact parser-owned slot identities
// may be unchanged. No translated bytes enter this source authority.
type SourceSlotReplacement struct {
	Old string `json:"old_slot_id"`
	New string `json:"new_slot_id"`
}
type StructuredSourceReplacementProposal struct {
	Schema              string                  `json:"schema"`
	Package             string                  `json:"package"`
	Predecessor         Reference               `json:"predecessor_authority"`
	PredecessorIdentity string                  `json:"predecessor_identity_sha256"`
	PriorPointer        Reference               `json:"prior_pointer"`
	Consumption         Reference               `json:"consuming_corpus_successor"`
	PredecessorHeads    []Reference             `json:"predecessor_package_source_heads"`
	Body                Reference               `json:"successor_body"`
	Replaced            []SourceSlotReplacement `json:"replaced_slots"`
}
type ConsumedSourceReplacement struct {
	Predecessor         Reference               `json:"predecessor_authority"`
	PredecessorIdentity string                  `json:"predecessor_identity_sha256"`
	PriorPointer        Reference               `json:"prior_pointer"`
	Consumption         Reference               `json:"consuming_corpus_successor"`
	CorpusSHA           string                  `json:"consumed_corpus_sha256"`
	CorpusIdentity      string                  `json:"consumed_corpus_identity_sha256"`
	PredecessorHeads    []Reference             `json:"predecessor_package_source_heads"`
	PredecessorPackage  Package                 `json:"predecessor_package"`
	Replaced            []SourceSlotReplacement `json:"replaced_slots"`
	Removed             []string                `json:"removed_slot_ids"`
}
type StructuredReplacementPlan struct {
	Authority        *StructuredSourceSuccessor `json:"authority"`
	Receipt          Reference                  `json:"planned_receipt"`
	ResultPointerSHA string                     `json:"result_pointer_sha256"`
	AlreadyCurrent   bool                       `json:"already_current"`
	Applied          bool                       `json:"applied"`
	Mutation         string                     `json:"mutation"`
}

func sameSourceSlot(a, b Unit) bool {
	a.Start, a.End, b.Start, b.End = 0, 0, 0, 0
	return reflect.DeepEqual(a, b)
}

func replacementSlotPartition(old, current Document, pairs []SourceSlotReplacement) ([]string, []string, []string, error) {
	if old.UnitScope != "" || current.UnitScope != "" || old.Contract != UnitContract || current.Contract != UnitContract || old.Path != current.Path || old.Kind != current.Kind || old.Route != current.Route || !reflect.DeepEqual(old.Dependencies, current.Dependencies) {
		return nil, nil, nil, fmt.Errorf("replacement structured parser/context mismatch")
	}
	before, after := map[string]Unit{}, map[string]Unit{}
	for _, u := range old.Units {
		if _, ok := before[u.ID]; ok {
			return nil, nil, nil, fmt.Errorf("duplicate old slot")
		}
		before[u.ID] = u
	}
	for _, u := range current.Units {
		if _, ok := after[u.ID]; ok {
			return nil, nil, nil, fmt.Errorf("duplicate new slot")
		}
		after[u.ID] = u
	}
	carry, removed, added := []string{}, []string{}, []string{}
	usedOld, usedNew := map[string]bool{}, map[string]bool{}
	for _, u := range old.Units {
		if v, ok := after[u.ID]; ok {
			if !sameSourceSlot(u, v) {
				return nil, nil, nil, fmt.Errorf("slot identity reused for changed semantics")
			}
			carry = append(carry, u.ID)
			usedOld[u.ID], usedNew[u.ID] = true, true
		}
	}
	for i, p := range pairs {
		_, o := before[p.Old]
		_, n := after[p.New]
		if !o || !n || usedOld[p.Old] || usedNew[p.New] || (i > 0 && pairs[i-1].Old >= p.Old) {
			return nil, nil, nil, fmt.Errorf("replacement pairs require unique existing changed IDs in canonical old-ID order")
		}
		usedOld[p.Old], usedNew[p.New] = true, true
	}
	for _, u := range old.Units {
		if !usedOld[u.ID] {
			removed = append(removed, u.ID)
		}
	}
	for _, u := range current.Units {
		if !usedNew[u.ID] {
			added = append(added, u.ID)
		}
	}
	return carry, removed, added, nil
}

// Immutable validation has no dependency on the current corpus or pointer.
// Current eligibility is checked separately by the preflight/apply entry.
func buildConsumedSourceReplacement(root string, ref Reference, depth int, contexts ...*structuredSuccessorValidation) (*StructuredSourceSuccessor, error) {
	v := successorValidation(root, contexts)
	var p StructuredSourceReplacementProposal
	if err := readReference(root, ref, &p); err != nil {
		return nil, err
	}
	if p.Schema != StructuredSourceReplacementSchema || !component(p.Package) {
		return nil, fmt.Errorf("invalid consumed replacement proposal")
	}
	old, err := checkStructuredSourceSuccessor(root, p.Predecessor, depth+1, v)
	if err != nil {
		return nil, err
	}
	if p.Package != old.Package || p.PredecessorIdentity != old.Identity || p.Body.Path == old.Body.Path || p.Body.Path == old.Previous.Path || p.PriorPointer.Path != structuredSourceCurrent(old.Previous.Path) {
		return nil, fmt.Errorf("replacement predecessor identity/path mismatch")
	}
	priorBytes, _ := Encode(p.Predecessor)
	if digest(priorBytes) != p.PriorPointer.SHA256 {
		return nil, fmt.Errorf("prior pointer does not bind exact predecessor")
	}
	consumed, err := ReadHistoricalAdoptedCorpusSuccessor(root, p.Consumption, v)
	if err != nil {
		return nil, err
	}
	found := false
	for _, r := range consumed.ProjectionAuthorities {
		found = found || r == p.Predecessor
	}
	bound := false
	for _, r := range consumed.Result.Sources {
		bound = bound || (r.Path == p.Predecessor.Path && r.SHA256 == p.Predecessor.SHA256)
	}
	if !found || !bound {
		return nil, fmt.Errorf("predecessor not consumed by exact immutable corpus")
	}
	base, err := v.adoptionBase()
	if err != nil {
		return nil, err
	}
	priorPackage, err := structuredPackageAtHeads(root, base, p.Package, p.PredecessorHeads, depth+1, v)
	if err != nil {
		return nil, err
	}
	parentFound := false
	for _, r := range p.PredecessorHeads {
		parentFound = parentFound || r == p.Predecessor
	}
	if !parentFound {
		return nil, fmt.Errorf("predecessor missing from frozen package heads")
	}
	body, err := readRegularReference(root, p.Body)
	if err != nil {
		return nil, err
	}
	before, err := readRegularReference(root, old.Body)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(before, body) {
		return nil, fmt.Errorf("replacement has no source delta")
	}
	if err := verifyFrozenStructuredRepresentation(old.Current, before); err != nil {
		return nil, err
	}
	s := documentSource(old.Current)
	s.SHA256 = p.Body.SHA256
	current, err := ParseDocument(s, body)
	if err != nil {
		return nil, err
	}
	if current.UnitScope != "" || len(current.Units) == 0 {
		return nil, fmt.Errorf("replacement requires nonempty structured semantic body")
	}
	carry, removed, added, err := replacementSlotPartition(old.Current, *current, p.Replaced)
	if err != nil {
		return nil, err
	}
	values := map[string]string{}
	for _, u := range current.Units {
		values[u.ID] = u.Source
	}
	round, err := reconstructFrozenStructuredRepresentation(*current, body, values)
	if err != nil || !bytes.Equal(round, body) {
		return nil, fmt.Errorf("replacement source round trip mismatch: %v", err)
	}
	a := &StructuredSourceSuccessor{Schema: StructuredSourceSuccessorSchema, Proposal: ref, BaseIdentity: base.Identity, Package: p.Package, Previous: old.Previous, Body: p.Body, Old: old.Current, Current: *current, CarryIDs: carry, AddedIDs: added,
		Replacement: &ConsumedSourceReplacement{Predecessor: p.Predecessor, PredecessorIdentity: old.Identity, PriorPointer: p.PriorPointer, Consumption: p.Consumption, CorpusSHA: consumed.ResultSHA, CorpusIdentity: consumed.Result.Identity, PredecessorHeads: p.PredecessorHeads, PredecessorPackage: priorPackage, Replaced: p.Replaced, Removed: removed}}
	a.Identity = identity(*a)
	return a, nil
}

func structuredPackageAtHeads(root string, base *Global, pkg string, heads []Reference, depth int, contexts ...*structuredSuccessorValidation) (Package, error) {
	v := successorValidation(root, contexts)
	if depth > 32 {
		return Package{}, fmt.Errorf("source ancestry limit")
	}
	p, err := packageByID(base, pkg)
	if err != nil {
		return Package{}, err
	}
	part := []Source{}
	for _, s := range base.Sources {
		if s.Package == pkg {
			part = append(part, s)
		}
	}
	seen := map[string]bool{}
	for i, ref := range heads {
		if i > 0 && heads[i-1].Path >= ref.Path {
			return Package{}, fmt.Errorf("package source heads must be ordered unique references")
		}
		a, err := checkStructuredSourceSuccessor(root, ref, depth+1, v)
		if err != nil {
			return Package{}, err
		}
		if a.Package != pkg || seen[a.Previous.Path] {
			return Package{}, fmt.Errorf("wrong/duplicate package source head")
		}
		seen[a.Previous.Path] = true
		found := false
		for j, s := range part {
			if s.Path == a.Previous.Path && s.SHA256 == a.Previous.SHA256 {
				part[j].SHA256 = a.Body.SHA256
				body := a.Body
				part[j].Body = &body
				found = true
			}
		}
		if !found {
			return Package{}, fmt.Errorf("package head source mismatch")
		}
	}
	p.SourceIdentity = identity(struct {
		Sources   []Source
		Contracts []Dependency
	}{part, p.Dependencies})
	p.Identity = ""
	p.Identity = identity(p)
	return p, nil
}

func replacementPublicationGuard(root string) error {
	for _, name := range []string{StructuredSourceReplacementPending, "data/.corpus-successor-pending"} {
		if _, err := os.Lstat(pathJoin(root, name)); !os.IsNotExist(err) {
			return fmt.Errorf("mutation=unknown: pending publication %s", name)
		}
	}
	return nil
}

func InspectConsumedStructuredSourceReplacement(root string, ref Reference) (*StructuredReplacementPlan, error) {
	if err := replacementPublicationGuard(root); err != nil {
		return nil, err
	}
	a, err := BuildStructuredSourceSuccessor(root, ref)
	if err != nil {
		return nil, err
	}
	if a.Replacement == nil {
		return nil, fmt.Errorf("explicit consumed replacement proposal required")
	}
	b, _ := Encode(a)
	receipt := Reference{StructuredSourceSuccessorPath(a), digest(b)}
	next, _ := Encode(receipt)
	current, err := readRegular(root, a.Replacement.PriorPointer.Path)
	if err != nil {
		return nil, err
	}
	already := bytes.Equal(current, next)
	if already {
		if _, err := CheckStructuredSourceSuccessor(root, receipt); err != nil {
			return nil, fmt.Errorf("mutation=unknown: current pointer has invalid/missing receipt: %w", err)
		}
	}
	if !already && digest(current) != a.Replacement.PriorPointer.SHA256 {
		return nil, fmt.Errorf("source pointer CAS mismatch")
	}
	if !already {
		corpus, err := readRegular(root, i18n.GlossaryCorpusPath)
		if err != nil || digest(corpus) != a.Replacement.CorpusSHA {
			return nil, fmt.Errorf("consuming corpus is not exact current corpus")
		}
		if err := CheckUnifiedCorpus(root); err != nil {
			return nil, err
		}
		g, err := loadSourceCurrent(root)
		if err != nil {
			return nil, err
		}
		heads := []Reference{}
		for _, r := range g.StructuredSuccessors {
			v, err := CheckStructuredSourceSuccessor(root, r)
			if err != nil {
				return nil, err
			}
			if v.Package == a.Package {
				heads = append(heads, r)
			}
		}
		sort.Slice(heads, func(i, j int) bool { return heads[i].Path < heads[j].Path })
		if !reflect.DeepEqual(heads, a.Replacement.PredecessorHeads) {
			return nil, fmt.Errorf("predecessor package authority set changed")
		}
	}
	fresh, err := BuildStructuredSourceSuccessor(root, ref)
	if err != nil || !reflect.DeepEqual(a, fresh) {
		return nil, fmt.Errorf("concurrent replacement input mutation")
	}
	last, err := readRegular(root, a.Replacement.PriorPointer.Path)
	if err != nil || !bytes.Equal(last, current) {
		return nil, fmt.Errorf("concurrent source pointer mutation")
	}
	if !already {
		last, err = readRegular(root, i18n.GlossaryCorpusPath)
		if err != nil || digest(last) != a.Replacement.CorpusSHA {
			return nil, fmt.Errorf("concurrent corpus mutation")
		}
	}
	return &StructuredReplacementPlan{Authority: a, Receipt: receipt, ResultPointerSHA: digest(next), AlreadyCurrent: already, Mutation: "none"}, nil
}

func ApplyConsumedStructuredSourceReplacement(root string, ref Reference, priorIdentity, priorPointer string) (*StructuredReplacementPlan, error) {
	return applyConsumedStructuredSourceReplacement(root, ref, priorIdentity, priorPointer, nil)
}

// The hook is internal and used only for deterministic concurrent-drift tests.
func applyConsumedStructuredSourceReplacement(root string, ref Reference, priorIdentity, priorPointer string, beforeCommit func()) (*StructuredReplacementPlan, error) {
	lockPath := pathJoin(root, "data/.glossary-corpus.lock")
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("publication lock held; stop and inspect: %w", err)
	}
	if err := lock.Close(); err != nil {
		return nil, err
	}
	defer os.Remove(lockPath)
	p, err := InspectConsumedStructuredSourceReplacement(root, ref)
	if err != nil {
		return nil, err
	}
	r := p.Authority.Replacement
	if priorIdentity != r.PredecessorIdentity || priorPointer != r.PriorPointer.SHA256 {
		return nil, fmt.Errorf("exact predecessor identity/pointer CAS required")
	}
	if p.AlreadyCurrent {
		p.Applied = true
		return p, nil
	}
	if err := saveImmutable(root, p.Receipt.Path, p.Authority); err != nil {
		return nil, err
	}
	if err := syncSourceDirectory(root, path.Dir(p.Receipt.Path)); err != nil {
		return nil, err
	}
	if beforeCommit != nil {
		beforeCommit()
	}
	fresh, err := InspectConsumedStructuredSourceReplacement(root, ref)
	if err != nil || !reflect.DeepEqual(p, fresh) {
		return nil, fmt.Errorf("concurrent replacement dependency mutation: %v", err)
	}
	next, _ := Encode(p.Receipt)
	f, err := os.CreateTemp(pathJoin(root, StructuredSourceSuccessorRoot), ".replacement-pointer-*")
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
		Prior    Reference `json:"prior"`
		Result   Reference `json:"result"`
		Proposal Reference `json:"proposal"`
	}{r.PriorPointer, p.Receipt, ref})
	if err := SaveArtifact(root, StructuredSourceReplacementPending, marker); err != nil {
		return nil, err
	}
	if err := syncSourceDirectory(root, "data"); err != nil {
		return nil, fmt.Errorf("mutation=unknown: %w", err)
	}
	last, err := BuildStructuredSourceSuccessor(root, ref)
	if err != nil || !reflect.DeepEqual(last, p.Authority) {
		return nil, fmt.Errorf("mutation=unknown: replacement dependency changed before commit: %v", err)
	}
	corpus, err := readRegular(root, i18n.GlossaryCorpusPath)
	if err != nil || digest(corpus) != r.CorpusSHA {
		return nil, fmt.Errorf("mutation=unknown: corpus changed before commit")
	}
	for _, head := range r.PredecessorHeads {
		a, err := CheckStructuredSourceSuccessor(root, head)
		if err != nil {
			return nil, fmt.Errorf("mutation=unknown: %w", err)
		}
		b, err := readRegular(root, structuredSourceCurrent(a.Previous.Path))
		want, _ := Encode(head)
		if err != nil || !bytes.Equal(b, want) {
			return nil, fmt.Errorf("mutation=unknown: package source heads changed before commit")
		}
	}
	current, err := readRegular(root, r.PriorPointer.Path)
	if err != nil || digest(current) != priorPointer {
		return nil, fmt.Errorf("mutation=unknown: pointer changed before commit")
	}
	if err := os.Rename(f.Name(), pathJoin(root, r.PriorPointer.Path)); err != nil {
		return nil, fmt.Errorf("mutation=unknown: %w", err)
	}
	if err := syncSourceDirectory(root, path.Dir(r.PriorPointer.Path)); err != nil {
		return nil, fmt.Errorf("mutation=unknown: %w", err)
	}
	installed, err := readRegular(root, r.PriorPointer.Path)
	if err != nil || !bytes.Equal(installed, next) {
		return nil, fmt.Errorf("mutation=unknown: result pointer changed during commit")
	}
	if err := os.Remove(pathJoin(root, StructuredSourceReplacementPending)); err != nil {
		return nil, fmt.Errorf("mutation=unknown: %w", err)
	}
	if err := syncSourceDirectory(root, "data"); err != nil {
		return nil, fmt.Errorf("mutation=unknown: %w", err)
	}
	p.Applied = true
	p.Mutation = "known"
	return p, nil
}

func syncSourceDirectory(root, name string) error {
	d, err := os.Open(pathJoin(root, name))
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
