package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// A source-setup correction, before any corpus authority consumes the source.
// Receipts/bodies remain immutable; this cannot revise accepted language or
// reinterpret an old generation. Default adoption still refuses replacement.
func CorrectUnconsumedStructuredSource(root string, proposal Reference, priorBase, priorPointer string) (*StructuredSourceSuccessor, error) {
	if err := replacementPublicationGuard(root); err != nil {
		return nil, err
	}
	if _, err := os.Lstat(pathJoin(root, "data/.corpus-successor-pending")); !os.IsNotExist(err) {
		return nil, fmt.Errorf("mutation=unknown: corpus publication pending")
	}
	if !validSHA(priorPointer) {
		return nil, fmt.Errorf("exact source pointer CAS required")
	}
	a, err := BuildStructuredSourceSuccessor(root, proposal)
	if err != nil {
		return nil, err
	}
	if a.Replacement != nil {
		return nil, fmt.Errorf("consumed replacement cannot use unconsumed correction")
	}
	if priorBase != a.BaseIdentity {
		return nil, fmt.Errorf("predecessor source scope CAS mismatch")
	}
	lock, err := os.OpenFile(pathJoin(root, "data/.glossary-corpus.lock"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	if err = lock.Close(); err != nil {
		return nil, err
	}
	defer os.Remove(lock.Name())
	name := structuredSourceCurrent(a.Previous.Path)
	b, err := readRegular(root, name)
	if err != nil {
		return nil, err
	}
	encoded, _ := Encode(a)
	ref := Reference{StructuredSourceSuccessorPath(a), digest(encoded)}
	next, _ := Encode(ref)
	if bytes.Equal(b, next) {
		return a, nil
	}
	if digest(b) != priorPointer {
		return nil, fmt.Errorf("source pointer CAS mismatch")
	}
	var oldRef Reference
	if err := StrictJSON(b, &oldRef); err != nil {
		return nil, err
	}
	old, err := CheckStructuredSourceSuccessor(root, oldRef)
	if err != nil {
		return nil, err
	}
	if old.Package != a.Package || old.BaseIdentity != a.BaseIdentity || old.Previous != a.Previous {
		return nil, fmt.Errorf("source correction predecessor scope mismatch")
	}
	if err := requireUnconsumedSource(root, oldRef); err != nil {
		return nil, err
	}
	if err := saveImmutable(root, ref.Path, a); err != nil {
		return nil, err
	}
	f, err := os.CreateTemp(pathJoin(root, StructuredSourceSuccessorRoot), ".source-pointer-*")
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
	fresh, err := BuildStructuredSourceSuccessor(root, proposal)
	if err != nil || fresh.Identity != a.Identity {
		return nil, fmt.Errorf("concurrent source correction dependency mutation")
	}
	current, err := readRegular(root, name)
	if err != nil || digest(current) != priorPointer {
		return nil, fmt.Errorf("concurrent source pointer mutation")
	}
	if err := os.Rename(f.Name(), filepath.Join(root, filepath.FromSlash(name))); err != nil {
		return nil, fmt.Errorf("mutation=unknown: source pointer CAS: %w", err)
	}
	return a, nil
}

func requireUnconsumedSource(root string, ref Reference) error {
	c, err := i18n.LoadUnifiedGlossaryCorpus(root)
	if err != nil {
		return err
	}
	for _, source := range c.Sources {
		if source.Path == ref.Path {
			return fmt.Errorf("source authority already consumed by current corpus")
		}
	}
	entries, err := os.ReadDir(pathJoin(root, CorpusSuccessorRoot))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unsafe corpus successor evidence")
		}
		name := CorpusSuccessorRoot + entry.Name()
		b, err := readRegular(root, name)
		if err != nil {
			return err
		}
		var p CorpusSuccessor
		if err := StrictJSON(b, &p); err != nil {
			return err
		}
		copy := p
		copy.Identity = ""
		if p.Schema != CorpusSuccessorSchema || p.Identity != identity(copy) || CorpusSuccessorPath(&p) != name {
			return fmt.Errorf("invalid corpus successor evidence")
		}
		for _, used := range p.ProjectionAuthorities {
			if used.Path == ref.Path {
				return fmt.Errorf("source authority consumed by immutable corpus successor")
			}
		}
	}
	return nil
}
