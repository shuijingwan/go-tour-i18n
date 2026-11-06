package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
)

func inspectSharedRepresentation(root string, g *Global, input Reference, v *structuredSuccessorValidation, owned bool) (*SharedRepresentationResult, error) {
	if err := sharedRepresentationReady(root, owned); err != nil {
		return nil, err
	}
	var p SharedRepresentationPlan
	if err := v.guard.readReference(root, input, &p); err != nil {
		return nil, err
	}
	x := p
	x.Identity = ""
	if p.Schema != SharedRepresentationSchema+"/plan" || p.Identity != identity(x) || p.Receipt != sharedRepresentationReference(p.Authority) || p.CurrentPath != sharedRepresentationHead(p.Authority.Request.Package, p.Authority.Request.AssetID) || p.State != "generation-required" || p.PriorIdentity != identity(p.Authority.Old) {
		return nil, fmt.Errorf("invalid shared representation plan")
	}
	var request SharedRepresentationRequest
	if err := v.guard.readReference(root, p.Request, &request); err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(request, p.Authority.Request) {
		return nil, fmt.Errorf("shared plan/request mismatch")
	}
	head, sha, err := readSharedRepresentationHead(root, request.Package, request.AssetID, v)
	if err != nil {
		return nil, err
	}
	result := &SharedRepresentationResult{Plan: &p, Mutation: "none"}
	if head != nil && *head == p.Receipt {
		raw, err := v.guard.readRegular(root, request.Path)
		if err != nil {
			return nil, err
		}
		r, err := readSharedRepresentationReceipt(root, *head, raw, v, map[Reference]bool{})
		if err != nil {
			return nil, err
		}
		if !reflect.DeepEqual(*r, p.Authority) {
			return nil, fmt.Errorf("installed shared authority/plan mismatch")
		}
		if err := checkSharedRepresentationCurrent(root, g, *r, v); err != nil {
			return nil, err
		}
		priorBytes := []byte(nil)
		if r.Previous != nil {
			priorBytes, _ = Encode(r.Previous)
		}
		if p.PriorPointerSHA != digest(priorBytes) {
			return nil, fmt.Errorf("shared original prior SHA mismatch")
		}
		result.AlreadyCurrent = true
		return result, nil
	}
	if sha != p.PriorPointerSHA || !reflect.DeepEqual(head, p.Authority.Previous) {
		return nil, fmt.Errorf("stale shared representation predecessor CAS")
	}
	want, err := buildSharedRepresentationPlan(root, g, p.Request, v, owned)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(want, &p) {
		return nil, fmt.Errorf("shared representation plan/current proof changed")
	}
	return result, nil
}

func InspectSharedStructuredRepresentation(root string, input Reference) (*SharedRepresentationResult, error) {
	var result *SharedRepresentationResult
	err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		g, err := loadSourceCurrent(root, v)
		if err != nil {
			return err
		}
		result, err = inspectSharedRepresentation(root, g, input, v, false)
		return err
	})
	return result, err
}

func ApplySharedStructuredRepresentation(root string, input Reference, priorIdentity, priorSHA string) (*SharedRepresentationResult, error) {
	return applySharedStructuredRepresentation(root, input, priorIdentity, priorSHA, func(v *structuredSuccessorValidation) (*Global, error) { return loadSourceCurrent(root, v) }, nil)
}

// Same write-once/CAS/durable publication pattern as Homepage authority, with
// a separate keyed head and marker. Tests inject only a verified source loader.
func applySharedStructuredRepresentation(root string, input Reference, priorIdentity, priorSHA string, load func(*structuredSuccessorValidation) (*Global, error), beforeCommit func()) (*SharedRepresentationResult, error) {
	if err := sharedRepresentationReady(root, false); err != nil {
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
	g, err := load(v)
	if err != nil {
		return nil, err
	}
	result, err := inspectSharedRepresentation(root, g, input, v, true)
	if err != nil {
		return nil, err
	}
	p := result.Plan
	if priorIdentity != p.PriorIdentity || priorSHA != p.PriorPointerSHA {
		return nil, fmt.Errorf("exact shared predecessor identity/pointer CAS required")
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
	fresh := successorValidation(root, nil)
	fresh.readonlyOwner = true
	freshG, err := load(fresh)
	if err != nil {
		return nil, err
	}
	again, err := inspectSharedRepresentation(root, freshG, input, fresh, true)
	if err != nil || !reflect.DeepEqual(result, again) {
		return nil, fmt.Errorf("concurrent shared representation mutation: %v", err)
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
	directory := pathJoin(root, SharedRepresentationRoot+"current")
	if err := os.Mkdir(directory, 0755); err != nil && !os.IsExist(err) {
		return nil, err
	}
	st, err := os.Lstat(directory)
	if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("unsafe shared head directory")
	}
	if err := v.aggregateVerify(); err != nil {
		return nil, err
	}
	for _, dir := range []string{SharedRepresentationRoot + "receipts", SharedRepresentationRoot + "current", SharedRepresentationRoot, "data/site-shell"} {
		if err := syncSourceDirectory(root, dir); err != nil {
			pending = true
			return nil, fmt.Errorf("mutation=unknown: receipt durability: %w", err)
		}
	}
	next, _ := Encode(p.Receipt)
	f, err := os.CreateTemp(pathJoin(root, SharedRepresentationRoot+"current"), ".shared-current-*")
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
		Head   string    `json:"head"`
		Prior  string    `json:"prior_pointer_sha256"`
		Result Reference `json:"result"`
	}{input, p.CurrentPath, priorSHA, p.Receipt})
	if err := SaveArtifact(root, SharedRepresentationPending, marker); err != nil {
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
	current, err := readRegular(root, p.CurrentPath)
	if os.IsNotExist(err) {
		current = nil
		err = nil
	}
	if err != nil || digest(current) != priorSHA {
		return nil, fmt.Errorf("mutation=unknown: shared head CAS drift")
	}
	if err := os.Rename(f.Name(), pathJoin(root, p.CurrentPath)); err != nil {
		return nil, fmt.Errorf("mutation=unknown: %w", err)
	}
	if err := syncSourceDirectory(root, SharedRepresentationRoot+"current"); err != nil {
		return nil, fmt.Errorf("mutation=unknown: %w", err)
	}
	installed, err := readRegular(root, p.CurrentPath)
	if err != nil || !bytes.Equal(installed, next) {
		return nil, fmt.Errorf("mutation=unknown: installed shared head mismatch")
	}
	if err := os.Remove(pathJoin(root, SharedRepresentationPending)); err != nil {
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
