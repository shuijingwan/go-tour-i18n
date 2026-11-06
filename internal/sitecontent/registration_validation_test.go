package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func registrationCatalogBytes(t *testing.T, c *i18n.Catalog) []byte {
	t.Helper()
	// Historical descriptors are deliberately excluded from Catalog's public
	// JSON, so compare their full ordered representation separately as well.
	b, err := Encode(struct {
		Catalog    *i18n.Catalog
		Historical []i18n.TranslationPackageDescriptor
	}{c, c.HistoricalTranslationPackages})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestReadonlyRegistrationCatalogPreparation(t *testing.T) {
	root, _, _ := plannerPropagationFixture(t)
	want, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackages(root, want); err != nil {
		t.Fatal(err)
	}
	wantBytes := registrationCatalogBytes(t, want)
	for invocation := 0; invocation < 2; invocation++ {
		got, err := checkedCatalog(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := AttachRegisteredTranslationPackagesReadonly(root, got); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(wantBytes, registrationCatalogBytes(t, got)) {
			t.Fatal("readonly preparation changed catalog/ordered package identity")
		}
		got.HistoricalTranslationPackages = nil // must not poison the next call
	}
	g, err := loadSourceCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	// A fresh invocation cannot inherit prior currentness/PASS state.
	old, err := readRegular(root, g.ShellAdoption.Path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathJoin(root, g.ShellAdoption.Path), append(old, ' '), 0644); err != nil {
		t.Fatal(err)
	}
	c, _ := checkedCatalog(root)
	if err := AttachRegisteredTranslationPackagesReadonly(root, c); err == nil {
		t.Fatal("second invocation reused stale adoption currentness")
	}
}

func TestRegistrationInvocationAuthorityReuse(t *testing.T) {
	if root := os.Getenv("GO_TOUR_CURRENT_SOURCE_ROOT"); root != "" {
		registrationCurrentCheckChild(t, root)
		return
	}
	root, parent, replacement := plannerPropagationFixture(t)
	want, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackages(root, want); err != nil {
		t.Fatal(err)
	}
	wantBytes := registrationCatalogBytes(t, want)
	var previous *structuredSuccessorValidation
	for invocation := 0; invocation < 2; invocation++ {
		var owned *structuredSuccessorValidation
		err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
			owned = v
			if v == previous || v.registrationAdoptions != nil {
				return fmt.Errorf("registration state escaped invocation")
			}
			v.historicalMetrics, v.builds = map[string]int{}, map[Reference]int{}
			inventory := newClosureValidation(root)
			inventory.readonlySuccessors = v
			got, err := inventory.inventoryCatalog()
			if err != nil {
				return err
			}
			if !bytes.Equal(wantBytes, registrationCatalogBytes(t, got)) {
				return fmt.Errorf("Catalog/current/historical descriptors changed")
			}
			if v.registrationReuse || len(v.registrationAdoptions) != 2 {
				return fmt.Errorf("registration scope/cache mismatch")
			}
			// Caller mutation must not poison the prepared immutable body.
			v.registrationReuse = true
			defer func() { v.registrationReuse = false }()
			for key := range v.registrationAdoptions {
				if key.kind != "shell" {
					continue
				}
				a, err := CheckShellAdoption(root, key.ref, v)
				if err != nil {
					return err
				}
				original := a.Sources[0].Path
				a.Sources[0].Path = "caller mutation"
				a, err = CheckShellAdoption(root, key.ref, v)
				if err != nil || a.Sources[0].Path != original {
					return fmt.Errorf("prepared adoption escaped private ownership: %v", err)
				}
				wrong := key.ref
				wrong.SHA256 = digest([]byte("wrong receipt SHA"))
				if _, err := CheckShellAdoption(root, wrong, v); err == nil {
					return fmt.Errorf("wrong SHA reused adoption proof")
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		m := owned.historicalMetrics
		if m["manifest-file-observation"] != 1 || m["source-path-capture-reuse"] == 0 {
			t.Fatalf("readonly registration recaptured the same frozen source world: %v", m)
		}
		if m["BuildGlobal"] != 1 || m["snapshotFS"] != m["global-environment-build"] || m["aggregate-verifies"] != 1 || m["registration-shell-adoption-build"] != 1 || m["registration-page-adoption-build"] != 1 || owned.builds[parent] != 1 || owned.builds[replacement] != 1 {
			t.Fatalf("repeated construction: metrics=%v successors=%v", m, owned.builds)
		}
		if owned.readonlyOwner || owned.registrationReuse {
			t.Fatal("readonly ownership escaped")
		}
		t.Logf("invocation=%d metrics=%v predecessor_rebuild=1 catalog_bytes_equal=true", invocation, m)
		previous = owned
	}
	// Standalone inventory owns its final aggregate check too.
	got, err := newClosureValidation(root).inventoryCatalog()
	if err != nil || !bytes.Equal(wantBytes, registrationCatalogBytes(t, got)) {
		t.Fatal("standalone inventory changed", err)
	}
	// An explicit default/public context never enters readonly preparation.
	eager := successorValidation(root, nil)
	c, _ := checkedCatalog(root)
	if err := AttachRegisteredTranslationPackages(root, c, eager); err != nil {
		t.Fatal(err)
	}
	if eager.registrationAdoptions != nil || eager.readonlyOwner || eager.registrationReuse {
		t.Fatal("public eager reader inherited readonly state")
	}
	testRegistrationCurrentCheckProcesses(t, root, want)
}

func TestRegistrationInvocationMutationFailClosed(t *testing.T) {
	root, _, _ := plannerPropagationFixture(t)
	g, err := loadSourceCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	var shell ShellAdoption
	if err := readReference(root, *g.ShellAdoption, &shell); err != nil {
		t.Fatal(err)
	}
	var page PagePackageAdoption
	if err := readReference(root, g.PackageAdoptions[0], &page); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{shell.Sources[0].Path, g.ShellAdoption.Path, shell.Foundation.Path, g.PackageAdoptions[0].Path, page.Proposal.Path, ShellAdoptionCurrent, PackageMembershipPath, StructuredSourceSuccessorRoot + "current/new-pointer.json"} {
		t.Run(name, func(t *testing.T) {
			full := filepath.Join(root, filepath.FromSlash(name))
			old, readErr := os.ReadFile(full)
			if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
				t.Fatal(err)
			}
			mutated := false
			err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
				c, err := checkedCatalog(root)
				if err != nil {
					return err
				}
				if err := AttachRegisteredTranslationPackages(root, c, v); err != nil {
					return err
				}
				// Change only after preparation/reuse, before aggregate success.
				if err := os.WriteFile(full, append(bytes.Clone(old), ' '), 0644); err != nil {
					return err
				}
				mutated = true
				return nil
			})
			if readErr == nil {
				if e := os.WriteFile(full, old, 0644); e != nil {
					t.Fatal(e)
				}
			} else if e := os.Remove(full); e != nil {
				t.Fatal(e)
			}
			if err == nil {
				t.Fatal("mid-invocation dependency mutation accepted")
			}
			if !mutated {
				t.Fatal("fixture did not reach mutation assertion", err)
			}
		})
	}
}
