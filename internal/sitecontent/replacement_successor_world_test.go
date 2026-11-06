package sitecontent

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestShellReplacementSuccessorWorld(t *testing.T) {
	root, g, r := shellReattestationFixture(t)
	makeCatalog := func(s ShellLanguageSource, authority string) *i18n.Catalog {
		raw, err := readRegular(root, s.Path)
		if err != nil {
			t.Fatal(err)
		}
		c := &i18n.Catalog{}
		d := i18n.TranslationPackageDescriptor{ID: r.Basis.Package.ID, AuthoritySHA256: authority, Units: []i18n.AdaptedTranslationUnit{{Unit: i18n.TranslationUnit{ID: s.Page.Units[0].ID, Kind: i18n.UnitKindPage, SourcePath: s.Path, SourceSHA256: s.SourceSHA, Source: raw}, Adapter: ShellPageAdapter{s, raw}, StableIndex: s.Page.StableIndex, Route: s.Page.Route}}}
		if err := c.RegisterTranslationPackage(d); err != nil {
			t.Fatal(err)
		}
		if err := c.UseTranslationPackage(d.ID); err != nil {
			t.Fatal(err)
		}
		return c
	}
	baseRef := shellRepresentationReceipt(r)
	// Match the real shape: a source-only package successor occurred after
	// the immutable A→B reconciliation, without changing Homepage bytes.
	priorPackage := r.Basis.Package
	priorPackage.SourceIdentity = digest([]byte("earlier exact source inventory"))
	priorPackage.Identity = ""
	priorPackage.Identity = identity(priorPackage)
	g.HistoricalPackages = append(g.HistoricalPackages, priorPackage)
	g.Identity = ""
	g.Identity = identity(*g)
	event := makeCatalog(r.Old.Source, priorPackage.Identity)
	current := makeCatalog(r.New.Source, representationPackage(r.Basis.Package, baseRef).Identity)
	check := func(c, e *i18n.Catalog) ([]i18n.GlossaryArchiveReference, error) {
		var refs []i18n.GlossaryArchiveReference
		err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
			var err error
			refs, err = shellReplacementSuccessorWorldAt(root, c, e, g, v)
			return err
		})
		return refs, err
	}
	first, err := check(current, event)
	if err != nil || len(first) < 3 {
		t.Fatal("exact B to C bridge", first, err)
	}
	first[0].Path = "caller mutation"
	second, err := check(current, event)
	if err != nil || second[0].Path == "caller mutation" {
		t.Fatal("invocation not isolated", err)
	}
	for _, kind := range []string{"missing successor", "tampered receipt", "discontinuous lineage", "cycle", "source drift", "wrong intermediate", "arbitrary current", "wrong package", "wrong route", "ambiguous terminal"} {
		t.Run(kind, func(t *testing.T) {
			c := cloneReplacementValue(reflect.ValueOf(current)).Interface().(*i18n.Catalog)
			e := cloneReplacementValue(reflect.ValueOf(event)).Interface().(*i18n.Catalog)
			path := ""
			var before []byte
			write := func(p string, b []byte) {
				path = p
				var err error
				before, err = readRegular(root, p)
				if err != nil {
					t.Fatal(err)
				}
				writeFixture(t, root, p, b)
			}
			defer func() {
				if path != "" {
					writeFixture(t, root, path, before)
				}
			}()
			switch kind {
			case "missing successor":
				write(ShellRepresentationCurrent, []byte("null\n"))
			case "tampered receipt":
				b, _ := readRegular(root, baseRef.Path)
				write(baseRef.Path, append(bytes.Clone(b), ' '))
			case "discontinuous lineage":
				bad := *r
				bad.PredecessorIdentity = digest([]byte("wrong predecessor"))
				bad.Identity = ""
				bad.Identity = identity(bad)
				if err := saveImmutable(root, shellRepresentationReceipt(&bad).Path, bad); err != nil {
					t.Fatal(err)
				}
				b, _ := Encode(shellRepresentationReceipt(&bad))
				write(ShellRepresentationCurrent, b)
			case "cycle":
				v := successorValidation(root, nil)
				if _, _, err := reconstructShellRepresentationBase(root, g, baseRef, v, map[Reference]bool{baseRef: true}, 0); err == nil {
					t.Fatal("cycle accepted")
				}
				return
			case "source drift":
				write(r.Basis.Source.Path, []byte("different source"))
			case "wrong intermediate":
				e.TranslationPackages[0].Units[0].Unit.ID = digest([]byte("impostor B"))
			case "arbitrary current":
				c.TranslationPackages[0].Units[0].Unit.ID = digest([]byte("impostor C"))
			case "wrong package":
				e.TranslationPackages[0].AuthoritySHA256 = digest([]byte("wrong authority"))
			case "wrong route":
				e.TranslationPackages[0].Units[0].Route = "/elsewhere"
			case "ambiguous terminal":
				c.TranslationPackages = append(c.TranslationPackages, c.TranslationPackages[0])
			}
			if _, err := check(c, e); err == nil {
				t.Fatal("accepted", kind)
			}
		})
	}
	// Additive dependency currentness is consumed through the existing verifier,
	// not by declaring the immutable stale base receipt current.
	driftShellContract(t, root)
	if _, err := check(current, event); err == nil {
		t.Fatal("unattested drift accepted")
	}
	p, input := reattestationFixturePlan(t, root, g)
	if _, err := applyShellReattestation(root, input, p.PriorPointerSHA, func(*structuredSuccessorValidation) (*Global, error) { return g, nil }, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := check(current, event); err != nil {
		t.Fatal("valid additive currentness rejected", err)
	}
	// The bridge's outer guard must detect changes after it was prepared.
	err = withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		if _, err := shellReplacementSuccessorWorldAt(root, current, event, g, v); err != nil {
			return err
		}
		b, err := readRegular(root, baseRef.Path)
		if err != nil {
			return err
		}
		writeFixture(t, root, baseRef.Path, append(b, ' '))
		return nil
	})
	if err == nil {
		t.Fatal(fmt.Errorf("mid-invocation dependency mutation accepted"))
	}
}
