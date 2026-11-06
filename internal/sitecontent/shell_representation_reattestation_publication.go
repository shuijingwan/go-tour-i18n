package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
)

func inspectShellReattestationAt(root string, g *Global, input Reference, v *structuredSuccessorValidation, ownedLock bool) (*ShellRepresentationReattestationResult, error) {
	if err := shellRepresentationReady(root, ownedLock); err != nil {
		return nil, err
	}
	var p ShellRepresentationReattestationPlan
	if err := v.guard.readReference(root, input, &p); err != nil {
		return nil, err
	}
	copy := p
	copy.Identity = ""
	if p.Schema != ShellRepresentationReattestationSchema+"/plan" || p.Identity != identity(copy) || !p.Eligible || p.BaseCurrent {
		return nil, fmt.Errorf("eligible exact dependency re-attestation plan required")
	}
	pointer, ref, base, current, err := shellReattestationBase(root, g, v)
	if err != nil {
		return nil, err
	}
	r, err := buildShellReattestation(root, pointer, ref, base, current, p.Authority.Previous, v)
	if err != nil {
		return nil, err
	}
	priorSHA := digest(nil)
	if r.Previous != nil {
		if _, err := readShellReattestationLineage(root, *r.Previous, r, v, map[Reference]bool{}, 0); err != nil {
			return nil, err
		}
		b, _ := Encode(r.Previous)
		priorSHA = digest(b)
	}
	want := ShellRepresentationReattestationPlan{Schema: ShellRepresentationReattestationSchema + "/plan", Authority: *r, Receipt: shellReattestationReceipt(r), CurrentPath: shellReattestationPath(base.Identity), PriorPointerSHA: priorSHA, BaseCurrent: len(r.Changes) == 0, Eligible: len(r.Changes) > 0, State: "generation-required"}
	want.Identity = identity(want)
	if !reflect.DeepEqual(want, p) {
		return nil, fmt.Errorf("dependency re-attestation plan/source/reconstruction changed")
	}
	head, sha, err := readShellReattestationHead(root, p.CurrentPath, v)
	if err != nil {
		return nil, err
	}
	result := &ShellRepresentationReattestationResult{Plan: &p, Mutation: "none"}
	if head != nil && *head == p.Receipt {
		actual, err := readShellReattestationLineage(root, *head, r, v, map[Reference]bool{}, 0)
		if err != nil || !reflect.DeepEqual(*r, *actual) {
			return nil, fmt.Errorf("installed dependency re-attestation mismatch: %v", err)
		}
		result.AlreadyCurrent = true
		return result, nil
	}
	if sha != p.PriorPointerSHA || !reflect.DeepEqual(head, r.Previous) {
		return nil, fmt.Errorf("stale dependency re-attestation prior CAS")
	}
	return result, nil
}

func InspectShellRepresentationReattestation(root string, input Reference) (*ShellRepresentationReattestationResult, error) {
	var result *ShellRepresentationReattestationResult
	err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		g, err := loadSourceCurrent(root, v)
		if err != nil {
			return err
		}
		result, err = inspectShellReattestationAt(root, g, input, v, false)
		return err
	})
	return result, err
}

func ApplyShellRepresentationReattestation(root string, input Reference, priorPointerSHA string) (*ShellRepresentationReattestationResult, error) {
	return applyShellReattestation(root, input, priorPointerSHA, func(v *structuredSuccessorValidation) (*Global, error) { return loadSourceCurrent(root, v) }, nil)
}

// Same durable publication boundary as the canonical successor, with its own
// pointer/receipt/marker. Never writes the base successor pointer or receipt.
func applyShellReattestation(root string, input Reference, priorPointerSHA string, load func(*structuredSuccessorValidation) (*Global, error), beforeCommit func()) (*ShellRepresentationReattestationResult, error) {
	if err := shellRepresentationReady(root, false); err != nil {
		return nil, err
	}
	lockPath := pathJoin(root, "data/.glossary-corpus.lock")
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	if err := lock.Close(); err != nil {
		return nil, fmt.Errorf("mutation=unknown: %w", err)
	}
	pending := false
	defer func() {
		if !pending {
			_ = os.Remove(lockPath)
		}
	}()
	v := successorValidation(root, nil)
	v.readonlyOwner = true
	defer func() { v.readonlyOwner = false }()
	g, err := load(v)
	if err != nil {
		return nil, err
	}
	result, err := inspectShellReattestationAt(root, g, input, v, true)
	if err != nil {
		return nil, err
	}
	p := result.Plan
	if priorPointerSHA != p.PriorPointerSHA {
		return nil, fmt.Errorf("exact dependency re-attestation prior-pointer CAS required")
	}
	if result.AlreadyCurrent {
		return result, v.aggregateVerify()
	}
	if beforeCommit != nil {
		beforeCommit()
	}
	fresh := successorValidation(root, nil)
	fresh.readonlyOwner = true
	freshG, err := load(fresh)
	if err != nil {
		return nil, err
	}
	again, err := inspectShellReattestationAt(root, freshG, input, fresh, true)
	if err != nil || !reflect.DeepEqual(again, result) {
		return nil, fmt.Errorf("concurrent dependency re-attestation mutation: %v", err)
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
	if err := v.guard.bind(p.Receipt); err != nil {
		return nil, err
	}
	for _, dir := range []string{ShellRepresentationReattestationRoot + "receipts", ShellRepresentationReattestationRoot, "data/site-shell"} {
		if err := syncSourceDirectory(root, dir); err != nil {
			pending = true
			return nil, fmt.Errorf("mutation=unknown: receipt durability: %w", err)
		}
	}
	pointer, _ := Encode(p.Receipt)
	if err := os.MkdirAll(pathJoin(root, ShellRepresentationReattestationRoot+"current"), 0755); err != nil {
		return nil, err
	}
	f, err := os.CreateTemp(pathJoin(root, ShellRepresentationReattestationRoot+"current"), ".reattestation-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f.Name())
	if err := f.Chmod(0644); err != nil {
		f.Close()
		return nil, err
	}
	if _, err := f.Write(pointer); err != nil {
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
		Plan    Reference `json:"plan"`
		Prior   string    `json:"prior_pointer_sha256"`
		Current string    `json:"current_path"`
		Result  Reference `json:"result"`
	}{input, priorPointerSHA, p.CurrentPath, p.Receipt})
	if err := SaveArtifact(root, ShellRepresentationReattestationPending, marker); err != nil {
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
	raw, err := readRegular(root, p.CurrentPath)
	if os.IsNotExist(err) {
		raw, err = nil, nil
	}
	if err != nil || digest(raw) != priorPointerSHA {
		return nil, fmt.Errorf("mutation=unknown: dependency re-attestation pointer CAS drift")
	}
	if err := os.Rename(f.Name(), pathJoin(root, p.CurrentPath)); err != nil {
		return nil, fmt.Errorf("mutation=unknown: %w", err)
	}
	for _, dir := range []string{ShellRepresentationReattestationRoot + "current", ShellRepresentationReattestationRoot} {
		if err := syncSourceDirectory(root, dir); err != nil {
			return nil, fmt.Errorf("mutation=unknown: %w", err)
		}
	}
	installed, err := readRegular(root, p.CurrentPath)
	if err != nil || !bytes.Equal(installed, pointer) {
		return nil, fmt.Errorf("mutation=unknown: dependency re-attestation pointer mismatch")
	}
	installedReceipt, err := readRegular(root, p.Receipt.Path)
	if err != nil || digest(installedReceipt) != p.Receipt.SHA256 {
		return nil, fmt.Errorf("mutation=unknown: dependency re-attestation receipt mismatch")
	}
	if err := os.Remove(pathJoin(root, ShellRepresentationReattestationPending)); err != nil {
		return nil, fmt.Errorf("mutation=unknown: %w", err)
	}
	if err := syncSourceDirectory(root, "data"); err != nil {
		return nil, fmt.Errorf("mutation=unknown: %w", err)
	}
	pending = false
	result.Applied, result.Mutation = true, "known"
	return result, nil
}
