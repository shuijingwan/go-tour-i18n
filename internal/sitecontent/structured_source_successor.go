package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

const StructuredSourceSuccessorSchema = "go-learning/structured-source-content-successor/v1"
const StructuredSourceSuccessorRoot = "data/structured-source-successors/"

// Append-only source governance. Logical source/route/package stay fixed;
// the historical file remains intact and the new body has an exact reference.
type StructuredSourceProposal struct {
	Schema   string    `json:"schema"`
	Package  string    `json:"package"`
	Previous Reference `json:"previous_source"`
	Body     Reference `json:"successor_body"`
	InsertAt int       `json:"insert_at"`
}
type StructuredSourceSuccessor struct {
	Replacement  *ConsumedSourceReplacement `json:"consumed_replacement,omitempty"`
	Schema       string                     `json:"schema"`
	Proposal     Reference                  `json:"proposal"`
	BaseIdentity string                     `json:"predecessor_scope_identity_sha256"`
	Package      string                     `json:"package"`
	Previous     Reference                  `json:"previous_source"`
	Body         Reference                  `json:"successor_body"`
	InsertAt     int                        `json:"insert_at"`
	Old          Document                   `json:"previous_document"`
	Current      Document                   `json:"current_document"`
	CarryIDs     []string                   `json:"carry_slot_ids"`
	AddedIDs     []string                   `json:"added_slot_ids"`
	Identity     string                     `json:"identity_sha256"`
}

func BuildStructuredSourceSuccessor(root string, ref Reference) (*StructuredSourceSuccessor, error) {
	return buildStructuredSourceSuccessor(root, ref, 0)
}

func buildStructuredSourceSuccessor(root string, ref Reference, depth int, contexts ...*structuredSuccessorValidation) (*StructuredSourceSuccessor, error) {
	v := successorValidation(root, contexts)
	if depth > 32 {
		return nil, fmt.Errorf("source successor ancestry limit")
	}
	b, err := readRegularReference(root, ref)
	if err != nil {
		return nil, err
	}
	var header struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(b, &header); err != nil {
		return nil, err
	}
	if header.Schema == StructuredSourceReplacementSchema {
		return buildConsumedSourceReplacement(root, ref, depth, v)
	}
	var p StructuredSourceProposal
	if err := readReference(root, ref, &p); err != nil {
		return nil, err
	}
	if p.Schema != StructuredSourceSuccessorSchema || !component(p.Package) || p.Previous.Path == p.Body.Path {
		return nil, fmt.Errorf("explicit distinct source successor body required")
	}
	g, err := v.adoptionBase()
	if err != nil {
		return nil, err
	}
	repositoryOwned := false
	for _, s := range g.Sources {
		if s.Path == p.Previous.Path && s.Package == p.Package && s.Origin == "repository" {
			repositoryOwned = true
		}
	}
	if !repositoryOwned {
		return nil, fmt.Errorf("source-owned expansion requires a registered repository source; frozen upstream needs its source-sync authority")
	}
	// Rebuild the registered source predecessor, before representation heads.
	// A subsequently adopted representation is not an input to this immutable
	// source authority and cannot validate itself against this earlier graph.
	docs, raw, err := baseCurrentPackageDocuments(root, g, p.Package, v)
	if err != nil {
		return nil, err
	}
	var old *Document
	for i := range docs {
		if docs[i].Path == p.Previous.Path {
			old = &docs[i]
		}
	}
	if old == nil || old.UnitScope != "" || old.Contract != UnitContract || digest(raw[old.Path]) != p.Previous.SHA256 {
		return nil, fmt.Errorf("source successor requires exact registered structured predecessor")
	}
	body, err := readRegularReference(root, p.Body)
	if err != nil {
		return nil, err
	}
	before := raw[old.Path]
	if p.InsertAt < 0 || p.InsertAt > len(before) || len(body) <= len(before) {
		return nil, fmt.Errorf("invalid append insertion")
	}
	addedBytes := len(body) - len(before)
	if !bytes.Equal(body[:p.InsertAt], before[:p.InsertAt]) || !bytes.Equal(body[p.InsertAt+addedBytes:], before[p.InsertAt:]) {
		return nil, fmt.Errorf("source successor may only insert source-owned content; historical bytes changed")
	}
	for _, u := range old.Units {
		if u.End > p.InsertAt {
			return nil, fmt.Errorf("source insertion must follow every existing semantic slot")
		}
	}
	source := documentSource(*old)
	source.SHA256 = p.Body.SHA256
	current, err := ParseDocument(source, body)
	if err != nil {
		return nil, err
	}
	current, err = appendFrozenStructuredPrefix(*old, before, *current, body, p.InsertAt, addedBytes)
	if err != nil {
		return nil, err
	}
	carry, added, err := sourceExpansionSlots(*old, *current)
	if err != nil {
		return nil, err
	}
	a := &StructuredSourceSuccessor{Schema: StructuredSourceSuccessorSchema, Proposal: ref, BaseIdentity: g.Identity, Package: p.Package, Previous: p.Previous, Body: p.Body, InsertAt: p.InsertAt, Old: *old, Current: *current, CarryIDs: carry, AddedIDs: added}
	a.Identity = identity(*a)
	return a, nil
}

// Preserve the source-adopted prefix representation, including any historical
// protector version. New parser discovery is confined to the explicit inserted
// source range. No old target/protection is inferred or widened.
func appendFrozenStructuredPrefix(old Document, oldRaw []byte, parsed Document, raw []byte, at, inserted int) (*Document, error) {
	if err := verifyFrozenStructuredRepresentation(old, oldRaw); err != nil {
		return nil, err
	}
	current := parsed
	current.Units = append([]Unit{}, old.Units...)
	for _, u := range parsed.Units {
		if u.End <= at {
			continue
		}
		if u.Start < at || u.End > at+inserted {
			return nil, fmt.Errorf("source insertion crossed semantic slot boundary")
		}
		current.Units = append(current.Units, u)
	}
	seen := map[string]int{}
	var structure bytes.Buffer
	pos := 0
	for i := range current.Units {
		u := &current.Units[i]
		ordinal := strings.LastIndexByte(u.Locator, ':')
		if ordinal < 0 {
			return nil, fmt.Errorf("invalid structured source locator")
		}
		base := u.Locator[:ordinal]
		seen[base]++
		if i >= len(old.Units) {
			u.Locator = base + ":" + strconv.Itoa(seen[base])
			u.ID = digest([]byte(UnitContract + "\n" + u.Locator))
		}
		prefix, suffix := current.Path+"#", ":"+u.SourceSHA
		if !strings.HasPrefix(base, prefix) || !strings.HasSuffix(base, suffix) {
			return nil, fmt.Errorf("invalid structured source context")
		}
		context := base[len(prefix) : len(base)-len(suffix)]
		structure.Write(raw[pos:u.Start])
		structure.WriteString("\x00" + context + "\x00" + protectionIdentity(u.Protected))
		pos = u.End
	}
	structure.Write(raw[pos:])
	current.StructureSHA = identity(struct {
		Structure    string
		Dependencies []Dependency
	}{digest(structure.Bytes()), current.Dependencies})
	if err := verifyFrozenStructuredRepresentation(current, raw); err != nil {
		return nil, err
	}
	return &current, nil
}

func sourceExpansionSlots(old, current Document) ([]string, []string, error) {
	if old.Contract != UnitContract || current.Contract != old.Contract || old.UnitScope != "" || current.UnitScope != "" || old.Path != current.Path || old.Route != current.Route || old.Kind != current.Kind || !reflect.DeepEqual(old.Dependencies, current.Dependencies) {
		return nil, nil, fmt.Errorf("source expansion parser/context drift")
	}
	carry, added := []string{}, []string{}
	seen := map[string]Unit{}
	for _, u := range current.Units {
		if _, ok := seen[u.ID]; ok {
			return nil, nil, fmt.Errorf("duplicate source semantic slot")
		}
		seen[u.ID] = u
	}
	oldIDs := map[string]bool{}
	for _, u := range old.Units {
		if oldIDs[u.ID] || !reflect.DeepEqual(u, seen[u.ID]) {
			return nil, nil, fmt.Errorf("source expansion changed/removed historical slot: %s", u.ID)
		}
		oldIDs[u.ID] = true
		carry = append(carry, u.ID)
	}
	for _, u := range current.Units {
		if !oldIDs[u.ID] {
			added = append(added, u.ID)
		}
	}
	if len(added) == 0 {
		return nil, nil, fmt.Errorf("strict semantic source expansion required")
	}
	return carry, added, nil
}

func StructuredSourceSuccessorPath(a *StructuredSourceSuccessor) string {
	return StructuredSourceSuccessorRoot + "receipts/" + a.Identity + ".json"
}
func structuredSourceCurrent(source string) string {
	return StructuredSourceSuccessorRoot + "current/" + digest([]byte(source)) + ".json"
}
func CheckStructuredSourceSuccessor(root string, ref Reference) (*StructuredSourceSuccessor, error) {
	return checkStructuredSourceSuccessor(root, ref, 0)
}
func checkStructuredSourceSuccessor(root string, ref Reference, depth int, contexts ...*structuredSuccessorValidation) (*StructuredSourceSuccessor, error) {
	return successorValidation(root, contexts).check(ref, depth)
}

func rebuildStructuredSourceSuccessor(root string, ref Reference, depth int, v *structuredSuccessorValidation) (*StructuredSourceSuccessor, error) {
	var a StructuredSourceSuccessor
	if err := readReference(root, ref, &a); err != nil {
		return nil, err
	}
	want, err := buildStructuredSourceSuccessor(root, a.Proposal, depth, v)
	if err != nil {
		return nil, err
	}
	if ref.Path != StructuredSourceSuccessorPath(want) || !reflect.DeepEqual(a, *want) {
		return nil, fmt.Errorf("structured source successor stale/noncanonical")
	}
	return &a, nil
}
func AdoptStructuredSourceSuccessor(root string, ref Reference, prior string) (*StructuredSourceSuccessor, error) {
	lock, err := os.OpenFile(pathJoin(root, "data/.glossary-corpus.lock"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	if err := lock.Close(); err != nil {
		return nil, err
	}
	defer os.Remove(lock.Name())
	if err := replacementPublicationGuard(root); err != nil {
		return nil, err
	}
	a, err := BuildStructuredSourceSuccessor(root, ref)
	if err != nil {
		return nil, err
	}
	if a.Replacement != nil {
		return nil, fmt.Errorf("consumed replacement requires explicit replacement CAS")
	}
	if !validSHA(prior) || prior != a.BaseIdentity {
		return nil, fmt.Errorf("exact predecessor scope identity CAS required")
	}
	if err := saveImmutable(root, StructuredSourceSuccessorPath(a), a); err != nil {
		return nil, err
	}
	authority, err := ReferenceFile(root, StructuredSourceSuccessorPath(a))
	if err != nil {
		return nil, err
	}
	pointer, _ := Encode(authority)
	if b, err := readRegular(root, structuredSourceCurrent(a.Previous.Path)); err == nil {
		if !bytes.Equal(b, pointer) {
			return nil, fmt.Errorf("another structured source successor is current")
		}
		return a, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	fresh, err := BuildStructuredSourceSuccessor(root, ref)
	if err != nil || !reflect.DeepEqual(a, fresh) {
		return nil, fmt.Errorf("concurrent structured source mutation")
	}
	return a, SaveArtifact(root, structuredSourceCurrent(a.Previous.Path), pointer)
}

func applyStructuredSourceSuccessors(root string, base *Global, contexts ...*structuredSuccessorValidation) (*Global, error) {
	if _, err := os.Lstat(pathJoin(root, StructuredSourceReplacementPending)); !os.IsNotExist(err) {
		return nil, fmt.Errorf("mutation=unknown: source replacement publication pending")
	}
	entries, err := os.ReadDir(pathJoin(root, StructuredSourceSuccessorRoot+"current"))
	if os.IsNotExist(err) {
		return base, nil
	}
	if err != nil {
		return nil, err
	}
	next := *base
	next.Sources = append([]Source{}, base.Sources...)
	next.Packages = append([]Package{}, base.Packages...)
	next.HistoricalPackages = append([]Package{}, base.HistoricalPackages...)
	next.StructuredSuccessors = append([]Reference{}, base.StructuredSuccessors...)
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			return nil, fmt.Errorf("unsafe source successor pointer")
		}
		b, err := readRegular(root, StructuredSourceSuccessorRoot+"current/"+entry.Name())
		if err != nil {
			return nil, err
		}
		var ref Reference
		if err := StrictJSON(b, &ref); err != nil {
			return nil, err
		}
		a, err := checkStructuredSourceSuccessor(root, ref, 0, contexts...)
		if err != nil {
			return nil, err
		}
		for ancestor := a; ancestor.Replacement != nil; {
			next.HistoricalPackages = append(next.HistoricalPackages, ancestor.Replacement.PredecessorPackage)
			ancestor, err = checkStructuredSourceSuccessor(root, ancestor.Replacement.Predecessor, 0, contexts...)
			if err != nil {
				return nil, err
			}
		}
		if entry.Name() != digest([]byte(a.Previous.Path))+".json" {
			return nil, fmt.Errorf("source successor pointer mismatch")
		}
		found := false
		for i, s := range next.Sources {
			if s.Path == a.Previous.Path && s.Package == a.Package && s.SHA256 == a.Previous.SHA256 {
				next.Sources[i].SHA256, next.Sources[i].Body = a.Body.SHA256, &a.Body
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("source successor predecessor membership mismatch")
		}
		next.StructuredSuccessors = append(next.StructuredSuccessors, ref)
	}
	for i, p := range next.Packages {
		part := []Source{}
		changed := false
		for _, s := range next.Sources {
			if s.Package == p.ID {
				part = append(part, s)
				changed = changed || s.Body != nil
			}
		}
		if !changed {
			continue
		}
		next.HistoricalPackages = append(next.HistoricalPackages, p)
		p.SourceIdentity = identity(struct {
			Sources   []Source
			Contracts []Dependency
		}{part, p.Dependencies})
		p.Identity = ""
		p.Identity = identity(p)
		next.Packages[i] = p
	}
	next.Identity = ""
	next.Identity = identity(next)
	return &next, ValidateGlobal(next)
}

func currentSourceReference(s Source) Reference {
	if s.Body != nil {
		return *s.Body
	}
	return Reference{s.Path, s.SHA256}
}

func applyStructuredDocumentSuccessors(root string, g *Global, pkg string, docs []Document, raw map[string][]byte, contexts ...*structuredSuccessorValidation) ([]Document, map[string][]byte, error) {
	for _, ref := range g.StructuredSuccessors {
		a, err := checkStructuredSourceSuccessor(root, ref, 0, contexts...)
		if err != nil {
			return nil, nil, err
		}
		if a.Package != pkg {
			continue
		}
		body, err := readRegularReference(root, a.Body)
		if err != nil {
			return nil, nil, err
		}
		found := false
		for i := range docs {
			if docs[i].Path == a.Previous.Path {
				docs[i] = a.Current
				raw[a.Previous.Path] = body
				found = true
			}
		}
		if !found {
			return nil, nil, fmt.Errorf("source successor document membership mismatch")
		}
	}
	return docs, raw, nil
}
