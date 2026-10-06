package sitecontent

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

const PackageCompletionKind = "independent-package-closure/v1"

func routeFamilies(pkg string) []string {
	if pkg == "site-v2-shell" {
		return []string{"/", "/translation/"}
	}
	return []string{"/learn/**", "/doc/tutorial/**", "/doc/database/**", "/doc/modules/**", "/doc/security/**"}
}

func packageCompletion(root string, g *Global, ref Reference) (Completion, error) {
	f, _, err := CheckPackageClosure(root, ref)
	if err != nil {
		return Completion{}, err
	}
	p, err := packageByID(g, f.Package)
	if err != nil {
		return Completion{}, err
	}
	if f.PackageSHA != p.Identity {
		return Completion{}, fmt.Errorf("package authority stale")
	}
	gate, err := RequireIntegratedSurface(root, f.Locale, ref)
	if err != nil {
		return Completion{}, err
	}
	refs := []Reference{ref, gate}
	return Completion{Package: p.ID, State: "complete", PackageIdentity: p.Identity, SourceIdentity: p.SourceIdentity, Surfaces: p.Surfaces, Routes: p.Routes, RouteFamilies: routeFamilies(p.ID), EvidenceKind: PackageCompletionKind, Evidence: refs, EvidenceIdentity: identity(refs), ContextIdentity: f.Identity}, nil
}
func checkPackageCompletion(root string, g *Global, locale string, c Completion) error {
	if len(c.Evidence) != 2 {
		return fmt.Errorf("closure + integrated Surface evidence required")
	}
	f, _, err := CheckPackageClosure(root, c.Evidence[0])
	if err != nil {
		return err
	}
	if f.Locale != locale {
		return fmt.Errorf("closure locale mismatch")
	}
	if err := validateHistoricalIntegratedSurface(root, locale, c.Evidence[1], c.Evidence[0]); err != nil {
		return err
	}
	p, err := packageByID(g, c.Package)
	if err != nil {
		return err
	}
	want := Completion{Package: p.ID, State: "complete", PackageIdentity: p.Identity, SourceIdentity: p.SourceIdentity, Surfaces: p.Surfaces, Routes: p.Routes, RouteFamilies: routeFamilies(p.ID), EvidenceKind: PackageCompletionKind, Evidence: c.Evidence, EvidenceIdentity: identity(c.Evidence), ContextIdentity: f.Identity}
	if !reflect.DeepEqual(c, want) {
		return fmt.Errorf("package completion identity mismatch")
	}
	return nil
}

type ActivationPlan struct {
	Locale        string     `json:"locale"`
	Package       string     `json:"package"`
	PriorSHA      string     `json:"prior_sha256"`
	Completion    Completion `json:"completion"`
	ResultSHA     string     `json:"result_sha256"`
	AlreadyActive bool       `json:"already_active"`
	result        []byte
}

// ActivationPreflight verifies full package gates before any coverage mutation.
func ActivationPreflight(root string, ref Reference) (*ActivationPlan, error) {
	f, _, err := CheckPackageClosure(root, ref)
	if err != nil {
		return nil, err
	}
	g, err := LoadCurrent(root)
	if err != nil {
		return nil, err
	}
	l, err := CheckLocale(root, g, f.Locale)
	if err != nil {
		return nil, err
	}
	c, err := packageCompletion(root, g, ref)
	if err != nil {
		return nil, err
	}
	old, err := readRegular(root, LocalePath(f.Locale))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	plan := &ActivationPlan{Locale: f.Locale, Package: f.Package, PriorSHA: digest(old), Completion: c}
	if os.IsNotExist(err) {
		plan.PriorSHA = ""
	}
	for _, existing := range l.Packages {
		if existing.Package == f.Package {
			if !reflect.DeepEqual(existing, c) {
				return nil, fmt.Errorf("existing different completion cannot be overwritten")
			}
			plan.AlreadyActive = true
			plan.result = old
			plan.ResultSHA = digest(old)
			return plan, nil
		}
	}
	l.Packages = append(l.Packages, c)
	if err := ValidateLocale(g, *l); err != nil {
		return nil, err
	}
	b, err := Encode(l)
	if err != nil {
		return nil, err
	}
	plan.result = b
	plan.ResultSHA = digest(b)
	return plan, nil
}

// ApplyActivation is the only intentional update to sparse locale authority.
// Historical artifacts and existing completion records remain byte-identical.
// The lock and expected prior hash prevent replay over an unexpected state.
func ApplyActivation(root string, ref Reference, expectedPriorSHA string) (*ActivationPlan, error) {
	plan, err := ActivationPreflight(root, ref)
	if err != nil {
		return nil, err
	}
	if plan.PriorSHA != expectedPriorSHA {
		return nil, fmt.Errorf("activation prior state changed")
	}
	if plan.AlreadyActive {
		return plan, nil
	}
	name := LocalePath(plan.Locale)
	dir := filepath.Dir(filepath.Join(root, name))
	lock, err := os.OpenFile(filepath.Join(dir, ".site-activation.lock"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	if err := lock.Close(); err != nil {
		return nil, err
	}
	defer os.Remove(lock.Name())
	old, err := readRegular(root, name)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	missing := os.IsNotExist(err)
	if (missing && expectedPriorSHA != "") || (!missing && digest(old) != expectedPriorSHA) {
		return nil, fmt.Errorf("activation concurrent mutation")
	}
	f, err := os.CreateTemp(dir, ".site-activation-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f.Name())
	if err := f.Chmod(0644); err != nil {
		f.Close()
		return nil, err
	}
	if _, err := f.Write(plan.result); err != nil {
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
	if missing {
		// Hard-link creation is atomic and cannot overwrite a concurrently created
		// sparse authority. Existing locale identity is never reinitialized.
		if err := os.Link(f.Name(), filepath.Join(root, name)); err != nil {
			return nil, err
		}
		return plan, nil
	}
	if err := os.Rename(f.Name(), filepath.Join(root, name)); err != nil {
		return nil, err
	}
	return plan, nil
}

// CompletedPackageFiles reconstructs validated targets for an opt-in runtime or
// later projection. It cannot consume arbitrary translated directories.
func CompletedPackageFiles(root, locale, pkg string) (map[string][]byte, error) {
	g, l, err := LocalCoverage(root, locale)
	if err != nil {
		return nil, err
	}
	for _, c := range l.Packages {
		if c.Package != pkg {
			continue
		}
		if err := checkPackageCompletion(root, g, locale, c); err != nil {
			return nil, err
		}
		_, files, err := CheckPackageClosure(root, c.Evidence[0])
		if err != nil {
			return nil, err
		}
		return files, nil
	}
	return nil, fmt.Errorf("package incomplete")
}
