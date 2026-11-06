package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
)

func inspectSharedReattestation(root string, g *Global, input Reference, v *structuredSuccessorValidation, owned bool) (*SharedRepresentationReattestationResult, error) {
	if err := sharedRepresentationReady(root, owned); err != nil {
		return nil, err
	}
	var p SharedRepresentationReattestationPlan
	if err := v.guard.readReference(root, input, &p); err != nil {
		return nil, err
	}
	c := p
	c.Identity = ""
	if p.Schema != SharedReattestationSchema+"/plan" || p.Identity != identity(c) || !p.Eligible || p.BaseCurrent {
		return nil, fmt.Errorf("eligible exact shared dependency re-attestation plan required")
	}
	pointer, base, current, err := sharedReattestationBase(root, g, p.Authority.BaseReceipt, v)
	if err != nil {
		return nil, err
	}
	r, err := buildSharedReattestation(root, pointer, p.Authority.BaseReceipt, base, current, p.Authority.Previous, v)
	if err != nil {
		return nil, err
	}
	priorSHA := digest(nil)
	if r.Previous != nil {
		if _, err := readSharedReattestationLineage(root, *r.Previous, r, v, map[Reference]bool{}, 0); err != nil {
			return nil, err
		}
		b, _ := Encode(r.Previous)
		priorSHA = digest(b)
	}
	want := SharedRepresentationReattestationPlan{Schema: SharedReattestationSchema + "/plan", Authority: *r, Receipt: sharedReattestationReceipt(r), CurrentPath: sharedReattestationPath(base.Identity), PriorPointerSHA: priorSHA, BaseCurrent: len(r.Changes) == 0, Eligible: len(r.Changes) > 0, State: "generation-required"}
	want.Identity = identity(want)
	if !reflect.DeepEqual(want, p) {
		return nil, fmt.Errorf("shared re-attestation plan/source/topology/dependencies changed")
	}
	head, sha, err := readShellReattestationHead(root, p.CurrentPath, v)
	if err != nil {
		return nil, err
	}
	result := &SharedRepresentationReattestationResult{Plan: &p, Mutation: "none"}
	if head != nil && *head == p.Receipt {
		actual, err := readSharedReattestationLineage(root, *head, r, v, map[Reference]bool{}, 0)
		if err != nil || !reflect.DeepEqual(*actual, *r) {
			return nil, fmt.Errorf("installed shared re-attestation mismatch: %v", err)
		}
		result.AlreadyCurrent = true
		return result, nil
	}
	if sha != p.PriorPointerSHA || !reflect.DeepEqual(head, r.Previous) {
		return nil, fmt.Errorf("stale shared dependency re-attestation prior CAS")
	}
	return result, nil
}

func InspectSharedRepresentationReattestation(root string, input Reference) (*SharedRepresentationReattestationResult, error) {
	var result *SharedRepresentationReattestationResult
	err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		g, err := loadSourceCurrent(root, v)
		if err != nil {
			return err
		}
		result, err = inspectSharedReattestation(root, g, input, v, false)
		return err
	})
	return result, err
}

func ApplySharedRepresentationReattestation(root string, input Reference, prior string) (*SharedRepresentationReattestationResult, error) {
	return applySharedReattestation(root, input, prior, nil)
}

// The same write-once/CAS/fsync pattern as the Homepage authority, with its own
// selector and marker. No base representation bytes are in the write set.
func applySharedReattestation(root string, input Reference, prior string, beforeCommit func()) (*SharedRepresentationReattestationResult, error) {
	if err := sharedRepresentationReady(root, false); err != nil {
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
	g, err := loadSourceCurrent(root, v)
	if err != nil {
		return nil, err
	}
	result, err := inspectSharedReattestation(root, g, input, v, true)
	if err != nil {
		return nil, err
	}
	p := result.Plan
	if prior != p.PriorPointerSHA {
		return nil, fmt.Errorf("exact shared re-attestation prior-pointer CAS required")
	}
	if result.AlreadyCurrent {
		return result, v.aggregateVerify()
	}
	if beforeCommit != nil {
		beforeCommit()
	}
	fresh := successorValidation(root, nil)
	fresh.readonlyOwner = true
	freshG, err := loadSourceCurrent(root, fresh)
	if err != nil {
		return nil, err
	}
	again, err := inspectSharedReattestation(root, freshG, input, fresh, true)
	if err != nil || !reflect.DeepEqual(result, again) {
		return nil, fmt.Errorf("concurrent shared re-attestation mutation: %v", err)
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
	for _, dir := range []string{SharedReattestationRoot + "receipts", SharedReattestationRoot, "data/site-shell"} {
		if err := syncSourceDirectory(root, dir); err != nil {
			pending = true
			return nil, fmt.Errorf("mutation=unknown: shared re-attestation receipt durability: %w", err)
		}
	}
	directory := pathJoin(root, SharedReattestationRoot+"current")
	if err := os.Mkdir(directory, 0755); err != nil && !os.IsExist(err) {
		return nil, err
	}
	st, err := os.Lstat(directory)
	if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("unsafe shared re-attestation directory")
	}
	if err := v.aggregateVerify(); err != nil {
		return nil, err
	}
	pointer, _ := Encode(p.Receipt)
	f, err := os.CreateTemp(directory, ".shared-reattestation-*")
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
	}{input, prior, p.CurrentPath, p.Receipt})
	if err := SaveArtifact(root, SharedReattestationPending, marker); err != nil {
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
	if err != nil || digest(raw) != prior {
		return nil, fmt.Errorf("mutation=unknown: shared re-attestation pointer CAS drift")
	}
	if err := os.Rename(f.Name(), pathJoin(root, p.CurrentPath)); err != nil {
		return nil, fmt.Errorf("mutation=unknown: %w", err)
	}
	for _, dir := range []string{SharedReattestationRoot + "current", SharedReattestationRoot} {
		if err := syncSourceDirectory(root, dir); err != nil {
			return nil, fmt.Errorf("mutation=unknown: %w", err)
		}
	}
	installed, err := readRegular(root, p.CurrentPath)
	if err != nil || !bytes.Equal(installed, pointer) {
		return nil, fmt.Errorf("mutation=unknown: shared re-attestation pointer mismatch")
	}
	b, err := readRegular(root, p.Receipt.Path)
	if err != nil || digest(b) != p.Receipt.SHA256 {
		return nil, fmt.Errorf("mutation=unknown: shared re-attestation receipt mismatch")
	}
	if err := os.Remove(pathJoin(root, SharedReattestationPending)); err != nil {
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
