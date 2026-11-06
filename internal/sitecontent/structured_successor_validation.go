package sitecontent

import (
	"fmt"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Explicit, invocation-owned immutable proof graph. Current pointer/CAS and
// publication eligibility remain in their existing callers. The guard uses the
// existing closure byte/mode/absence checks, including on every proof reuse.
type structuredSuccessorValidation struct {
	registrationReuse       bool // explicit readonly registration subtree only
	registrationAdoptions   map[registrationAdoptionKey][]byte
	readonlyOwner           bool // explicit readonly registration/source subphase only
	disableHistoricalReuse  bool // differential test only
	historical              *historicalGlobalProof
	currentSource           *currentSourceProjection
	currentSourcePhase      bool // explicit installed-corpus reconstruction only
	sharedSourceObservation bool // explicit whole readonly bridge construction
	guard                   *closureValidation
	base                    *Global
	proofs                  map[Reference]successorProof
	active                  map[Reference]bool
	frames                  []*successorDepth
	disabled                bool              // differential test only
	builds                  map[Reference]int // nil outside focused tests
	historicalMetrics       map[string]int    // nil outside focused tests
	historicalIdentities    map[string]int    // nil outside focused tests
}

// Only an explicit readonly caller owns this lifetime. A context never remains
// deferred after return, and publication/default contexts stay eager.
func withReadonlyStructuredValidation(root string, work func(*structuredSuccessorValidation) error) error {
	v := successorValidation(root, nil)
	v.readonlyOwner = true
	defer func() { v.readonlyOwner = false }()
	if err := work(v); err != nil {
		return err
	}
	return v.aggregateVerify()
}

func (v *structuredSuccessorValidation) verifyObserved() error {
	if v.readonlyOwner {
		return nil
	}
	return v.aggregateVerify()
}

func (v *structuredSuccessorValidation) aggregateVerify() error {
	if v.historicalMetrics != nil {
		v.historicalMetrics["aggregate-verifies"]++
		reads := len(v.guard.deps) + len(v.guard.absent)
		for _, members := range v.guard.dirs {
			reads += len(members)
		}
		v.historicalMetrics["aggregate-rereads"] += reads
	}
	return v.guard.verify()
}

func (v *structuredSuccessorValidation) countHistorical(name string) {
	if v.historicalMetrics != nil {
		v.historicalMetrics[name]++
	}
}

type successorProof struct {
	bytes  []byte
	height int
}
type successorDepth struct{ depth, maximum int }

func successorValidation(root string, contexts []*structuredSuccessorValidation) *structuredSuccessorValidation {
	if len(contexts) != 0 && contexts[0] != nil {
		return contexts[0]
	}
	return &structuredSuccessorValidation{guard: newClosureValidation(root), proofs: map[Reference]successorProof{}, active: map[Reference]bool{}}
}

func (v *structuredSuccessorValidation) depth(depth int) {
	for _, f := range v.frames {
		if depth > f.maximum {
			f.maximum = depth
		}
	}
}

func (v *structuredSuccessorValidation) check(ref Reference, depth int) (*StructuredSourceSuccessor, error) {
	if depth > 32 || v.active[ref] {
		return nil, fmt.Errorf("source successor ancestry limit/cycle")
	}
	v.depth(depth)
	if p, ok := v.proofs[ref]; ok && !v.disabled {
		if depth+p.height > 32 {
			return nil, fmt.Errorf("source successor ancestry limit")
		}
		if err := v.guard.bind(ref); err != nil {
			return nil, err
		}
		if err := v.verifyObserved(); err != nil {
			return nil, err
		}
		v.depth(depth + p.height)
		var copy StructuredSourceSuccessor
		if err := StrictJSON(p.bytes, &copy); err != nil {
			return nil, err
		}
		return &copy, nil
	}
	v.active[ref] = true
	defer delete(v.active, ref)
	f := &successorDepth{depth, depth}
	v.frames = append(v.frames, f)
	defer func() { v.frames = v.frames[:len(v.frames)-1] }()
	if v.builds != nil {
		v.builds[ref]++
	}
	var declared StructuredSourceSuccessor
	if err := v.guard.readReference(v.guard.root, ref, &declared); err != nil {
		return nil, err
	}
	if err := v.observe(declared); err != nil {
		return nil, err
	}
	a, err := rebuildStructuredSourceSuccessor(v.guard.root, ref, depth, v)
	if err != nil {
		return nil, err
	}
	if err := v.guard.bind(ref); err != nil {
		return nil, err
	}
	if err := v.observe(*a); err != nil {
		return nil, err
	}
	if err := v.verifyObserved(); err != nil {
		return nil, err
	}
	b, err := Encode(a)
	if err != nil {
		return nil, err
	}
	if !v.disabled {
		v.proofs[ref] = successorProof{b, f.maximum - f.depth}
	}
	return a, nil
}

func (v *structuredSuccessorValidation) adoptionBase() (*Global, error) {
	if v.base != nil && !v.disabled {
		if v.readonlyOwner {
			if err := v.guard.bind(Reference{GlobalPath, v.guard.deps[GlobalPath]}); err != nil {
				return nil, err
			}
		}
		return v.base, v.verifyObserved()
	}
	for _, name := range []string{GlobalPath, SourceManifestPath, SnapshotPath, ShellAdoptionCurrent, i18n.GlossaryCorpusPath} {
		if err := v.guard.optional(name); err != nil {
			return nil, err
		}
	}
	// Freeze declared input paths before their canonical reconstruction, not
	// afterwards: a mutation between verification and observation must not be
	// mistaken for the bytes that supplied the proof.
	for _, item := range []struct {
		path  string
		value any
	}{{GlobalPath, &Global{}}, {i18n.GlossaryCorpusPath, &i18n.GlossarySourceCorpus{}}} {
		if _, absent := v.guard.absent[item.path]; !absent {
			b, err := v.guard.readRegular(v.guard.root, item.path)
			if err != nil {
				return nil, err
			}
			if err := StrictJSON(b, item.value); err != nil {
				return nil, err
			}
			if err := v.observe(item.value); err != nil {
				return nil, err
			}
		}
	}
	if _, absent := v.guard.absent[ShellAdoptionCurrent]; !absent {
		var ref Reference
		b, err := v.guard.readRegular(v.guard.root, ShellAdoptionCurrent)
		if err != nil {
			return nil, err
		}
		if err := StrictJSON(b, &ref); err != nil {
			return nil, err
		}
		var a ShellAdoption
		if err := v.guard.readReference(v.guard.root, ref, &a); err != nil {
			return nil, err
		}
		var foundation ShellSuccessorAuthority
		if err := v.guard.readReference(v.guard.root, a.Foundation, &foundation); err != nil {
			return nil, err
		}
		if err := v.observe(a); err != nil {
			return nil, err
		}
		if err := v.observe(foundation); err != nil {
			return nil, err
		}
	}
	g, err := adoptionBase(v.guard.root, v)
	if err != nil {
		return nil, err
	}
	if _, absent := v.guard.absent[SourceManifestPath]; !absent {
		m, err := loadSourceManifest(v.guard.root)
		if err != nil {
			return nil, err
		}
		for _, file := range m.Files {
			if err := observeManifestSourcePaths(file.Path, v.observeSourcePath); err != nil {
				return nil, err
			}
		}
	}
	if err := v.observe(*g); err != nil {
		return nil, err
	}
	if g.ShellAdoption != nil {
		var a ShellAdoption
		if err := v.guard.readReference(v.guard.root, *g.ShellAdoption, &a); err != nil {
			return nil, err
		}
		if err := v.observe(a); err != nil {
			return nil, err
		}
		var foundation ShellSuccessorAuthority
		if err := v.guard.readReference(v.guard.root, a.Foundation, &foundation); err != nil {
			return nil, err
		}
		if err := v.observe(foundation); err != nil {
			return nil, err
		}
	}
	v.base = g
	return g, v.verifyObserved()
}

// Observe typed references and source/contract paths, not a filesystem scan.
// Historical prior pointers are derived virtual bytes; the immutable verifier
// validates those bindings. Observe their actual current presence/bytes without
// mistaking an old pointer SHA for current eligibility.
func (v *structuredSuccessorValidation) observe(value any) error {
	var walk func(reflect.Value) error
	walk = func(x reflect.Value) error {
		if !x.IsValid() {
			return nil
		}
		if x.Kind() == reflect.Pointer || x.Kind() == reflect.Interface {
			if x.IsNil() {
				return nil
			}
			return walk(x.Elem())
		}
		if x.Kind() == reflect.Struct {
			// All source governance structs declare repository paths as Path;
			// route and semantic identities have different field names.
			if path := x.FieldByName("Path"); path.IsValid() && path.Kind() == reflect.String && path.String() != "" {
				if err := v.observeSourcePath(path.String()); err != nil {
					return err
				}
			}
			for i := 0; i < x.NumField(); i++ {
				if err := walk(x.Field(i)); err != nil {
					return err
				}
			}
		} else if x.Kind() == reflect.Slice || x.Kind() == reflect.Array {
			for i := 0; i < x.Len(); i++ {
				if err := walk(x.Index(i)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(reflect.ValueOf(value))
}

// Dependency capture belongs to the existing closed readonly source or
// registration phase. Repeated traversal of a derived graph must not reopen
// every captured file. The owner still verifies the complete bytes/modes,
// absence and discovery set before any result escapes. Explicit reference/hash
// readers are unchanged; default/eager callers retain per-access observation.
func (v *structuredSuccessorValidation) observeSourcePath(path string) error {
	if !v.disabled && v.readonlyOwner && (v.currentSourcePhase || v.registrationReuse) {
		_, captured := v.guard.deps[path]
		if captured || v.guard.absent[path] {
			v.countHistorical("source-path-capture-reuse")
			return nil
		}
	}
	v.countHistorical("source-path-capture")
	return v.guard.optional(path)
}
