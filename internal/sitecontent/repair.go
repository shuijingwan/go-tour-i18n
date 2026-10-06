package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
)

// RepairUncommittedV2A replaces only the still-untracked V2-A scope artifacts.
// It never rewrites source snapshots or historical language/review evidence.
// This is not a future upstream-sync or completion-activation workflow.
func RepairUncommittedV2A(root string) (int, error) {
	snapshot, err := readRegular(root, SnapshotPath)
	if err != nil {
		return 0, err
	}
	oldBytes, err := readRegular(root, GlobalPath)
	if err != nil {
		return 0, err
	}
	var old Global
	if err := StrictJSON(oldBytes, &old); err != nil {
		return 0, err
	}
	if old.Schema != GlobalSchema || old.SnapshotSHA256 != digest(snapshot) {
		return 0, fmt.Errorf("repair requires existing V2-A authority and unchanged snapshot")
	}
	g, err := BuildGlobal(root, snapshot)
	if err != nil {
		return 0, err
	}
	states, err := LegacyPlan(root, g)
	if err != nil {
		return 0, err
	}
	outputs := map[string][]byte{}
	outputs[GlobalPath], err = Encode(g)
	if err != nil {
		return 0, err
	}
	paths := []string{GlobalPath}
	for _, l := range states {
		p := LocalePath(l.Locale)
		b, err := readRegular(root, p)
		if err != nil {
			return 0, err
		}
		var prior Locale
		if err := StrictJSON(b, &prior); err != nil {
			return 0, err
		}
		if prior.Schema != LocaleSchema || prior.Locale != l.Locale || len(prior.Packages) == 0 {
			return 0, fmt.Errorf("not an existing V2-A bootstrap: %s", p)
		}
		seen := map[string]bool{"tour-v1": true}
		for _, c := range prior.Packages[1:] {
			if seen[c.Package] || (c.Package != "site-v2-shell" && c.Package != "learn-docs-v1") || c.State != "incomplete" || len(c.Evidence)+len(c.Routes)+len(c.Surfaces)+len(c.RouteFamilies) != 0 || c.EvidenceKind != "" || c.EvidenceIdentity != "" || c.ContextIdentity != "" || c.LegacySurfaceState != "" {
				return 0, fmt.Errorf("repair refuses non-bootstrap package record: %s", p)
			}
			seen[c.Package] = true
		}
		// Regeneration may correct scope projection, not substitute evidence.
		c, want := prior.Packages[0], l.Packages[0]
		if c.Package != "tour-v1" || c.State != "complete" || c.EvidenceKind != want.EvidenceKind || c.EvidenceIdentity != want.EvidenceIdentity || c.ContextIdentity != want.ContextIdentity || c.LegacySurfaceState != want.LegacySurfaceState || !reflect.DeepEqual(c.Evidence, want.Evidence) {
			return 0, fmt.Errorf("repair would change legacy evidence/context: %s", p)
		}
		outputs[p], err = Encode(l)
		if err != nil {
			return 0, err
		}
		paths = append(paths, p)
	}
	// Refuse even modified tracked files: this command is exclusively for the
	// incorrect uncommitted initial V2-A implementation, not historical authority.
	args := append([]string{"ls-files", "-z", "--"}, paths...)
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	tracked, err := cmd.Output()
	if err != nil || len(tracked) != 0 {
		return 0, fmt.Errorf("repair requires all scope outputs untracked (git error=%v)", err)
	}
	for _, p := range paths {
		current, err := readRegular(root, p)
		if err != nil {
			return 0, err
		}
		if bytes.Equal(current, outputs[p]) {
			continue
		}
		f, err := os.CreateTemp(filepath.Dir(filepath.Join(root, p)), ".content-scope-repair-*")
		if err != nil {
			return 0, err
		}
		name := f.Name()
		err = f.Chmod(0644)
		if err == nil {
			_, err = f.Write(outputs[p])
		}
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		}
		if err == nil {
			err = os.Rename(name, filepath.Join(root, p))
		}
		if err != nil {
			os.Remove(name)
			return 0, err
		}
	}
	return len(states), nil
}
