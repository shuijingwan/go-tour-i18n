package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const installedShared008Receipt = "data/site-shell/shared-representation-successors/receipts/758c4e4888a3bb1d7fdbc324ea3cfd0c7ced531c6514deab75520bed0db27fa3.json"
const installedShared008ReceiptSHA = "0167896bdcab112a6eb7b6810aa70c341be8e411554a116b37e2a3ea9fca0da6"

func logSharedBasisDifference(t *testing.T, r SharedRepresentationSuccessor, pkg Package, registry string, refs []Reference) {
	t.Helper()
	show := func(key string, v any) { b, _ := json.Marshal(v); t.Logf("%s=%s", key, b) }
	show("RECORDED_SOURCE_PACKAGE", r.SourcePackage)
	show("CURRENT_BASIS_SOURCE_PACKAGE", pkg)
	t.Logf("RECORDED_SOURCE_PACKAGE_IDENTITY=%s", r.SourcePackage.Identity)
	t.Logf("CURRENT_BASIS_SOURCE_PACKAGE_IDENTITY=%s", pkg.Identity)
	t.Logf("RECORDED_REGISTRY_SHA256=%s", r.RegistrySHA)
	t.Logf("CURRENT_BASIS_REGISTRY_SHA256=%s", registry)
	show("RECORDED_SOURCE_AUTHORITY", r.SourceAuthority)
	show("CURRENT_BASIS_SOURCE_AUTHORITY", refs)
	t.Logf("PACKAGE_EQUAL=%t REGISTRY_EQUAL=%t SOURCE_AUTHORITY_EQUAL=%t", reflect.DeepEqual(pkg, r.SourcePackage), registry == r.RegistrySHA, reflect.DeepEqual(refs, r.SourceAuthority))
	a, b := reflect.ValueOf(r.SourcePackage), reflect.ValueOf(pkg)
	for i := 0; i < a.NumField(); i++ {
		if !reflect.DeepEqual(a.Field(i).Interface(), b.Field(i).Interface()) {
			show("PACKAGE_FIELD_"+a.Type().Field(i).Name+"_RECORDED", a.Field(i).Interface())
			show("PACKAGE_FIELD_"+a.Type().Field(i).Name+"_CURRENT", b.Field(i).Interface())
		}
	}
	old, current := map[string]string{}, map[string]string{}
	for _, x := range r.SourceAuthority {
		old[x.Path] = x.SHA256
	}
	for _, x := range refs {
		current[x.Path] = x.SHA256
	}
	for _, x := range refs {
		if sha, ok := old[x.Path]; !ok {
			show("ADDED_SOURCE_AUTHORITY", x)
		} else if sha != x.SHA256 {
			show("CHANGED_SOURCE_AUTHORITY", []Reference{{x.Path, sha}, x})
		}
	}
	for _, x := range r.SourceAuthority {
		if _, ok := current[x.Path]; !ok {
			show("REMOVED_SOURCE_AUTHORITY", x)
		}
	}
}

// Read the installed authority without executing a real lifecycle command or
// publishing anything. In checkouts without this retained evidence, skip it.
func TestSharedRepresentationInstalledBasisDiagnosis(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	b, err := readRegular(root, installedShared008Receipt)
	if os.IsNotExist(err) {
		t.Skip("installed first-use evidence unavailable", err)
	}
	if err != nil {
		t.Fatal(err)
	}
	if digest(b) != installedShared008ReceiptSHA {
		t.Fatal("installed receipt changed")
	}
	var r SharedRepresentationSuccessor
	if err := StrictJSON(b, &r); err != nil {
		t.Fatal(err)
	}
	if err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		g, err := v.adoptionBase()
		if err != nil {
			return err
		}
		t.Log("DIAG_STAGE=structured-source-predecessor-reconstruction")
		_, pkg, registry, refs, _, err := sharedRepresentationBasis(root, g, r.Request, v)
		if err != nil {
			return err
		}
		logSharedBasisDifference(t, r, pkg, registry, refs)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		g, err := loadSourceCurrent(root, v)
		if err != nil {
			return fmt.Errorf("current source graph: %w", err)
		}
		_, pkg, registry, refs, _, err := sharedRepresentationBasis(root, g, r.Request, v)
		if err != nil {
			return err
		}
		t.Log("DIAG_STAGE=current-source-graph")
		logSharedBasisDifference(t, r, pkg, registry, refs)
		g, err = LoadCurrent(root, v)
		if err != nil {
			return fmt.Errorf("full current graph: %w", err)
		}
		_, pkg, registry, refs, _, err = sharedRepresentationBasis(root, g, r.Request, v)
		if err != nil {
			return err
		}
		t.Log("DIAG_STAGE=current-representation-projection")
		logSharedBasisDifference(t, r, pkg, registry, refs)
		if !reflect.DeepEqual(pkg, r.SourcePackage) || registry != r.RegistrySHA || !reflect.DeepEqual(refs, r.SourceAuthority) {
			return fmt.Errorf("source-only projection mismatch remains")
		}
		if err := checkSharedRepresentationSource(root, g, r, v); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSharedRepresentationPostApplyBasisIsolation(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	real, request, q := shared008Request(t)
	if _, err := os.Stat(pathJoin(real, installedShared008Receipt)); os.IsNotExist(err) {
		t.Skip("installed authority unavailable")
	}
	root, extend := mirrorShared008Authority(t, real, request)
	head := sharedRepresentationHead(q.Package, q.AssetID)
	extend(head)
	extend("data/glossary-corpus-history")
	for _, path := range sharedReattestationValidationPaths {
		extend(path)
	}
	var r SharedRepresentationSuccessor
	if err := readReference(root, Reference{installedShared008Receipt, installedShared008ReceiptSHA}, &r); err != nil {
		t.Fatal(err)
	}
	before, _ := readRegular(root, installedShared008Receipt)
	pointer, _ := readRegular(root, head)
	var source *Global
	if err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		var err error
		source, err = loadSourceCurrent(root, v)
		if err != nil {
			return err
		}
		if err := checkSharedRepresentationSource(root, source, r, v); err != nil {
			return fmt.Errorf("immutable current source basis: %w", err)
		}
		projected, err := LoadCurrent(root, v)
		if err != nil {
			return err
		}
		_, pkg, registry, refs, _, err := sharedRepresentationBasis(root, projected, q, v)
		if err != nil {
			return err
		}
		t.Log("DIAG_STAGE=isolated-current-representation-projection")
		logSharedBasisDifference(t, r, pkg, registry, refs)
		if !reflect.DeepEqual(pkg, r.SourcePackage) || registry != r.RegistrySHA || !reflect.DeepEqual(refs, r.SourceAuthority) {
			return fmt.Errorf("unexpected mismatch components")
		}
		full, err := packageByID(projected, r.Request.Package)
		if err != nil {
			return err
		}
		if want := representationPackage(r.SourcePackage, *projected.ShellRepresentation); !reflect.DeepEqual(full, want) {
			return fmt.Errorf("independent Homepage package projection changed")
		}
		if err := checkSharedRepresentationCurrent(root, projected, r, v); err == nil {
			return fmt.Errorf("dependency drift accepted before re-attestation")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	check := func(g *Global) error {
		return withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error { return checkSharedRepresentationCurrent(root, g, r, v) })
	}
	t.Run("source-package-drift", func(t *testing.T) {
		g := *source
		g.Packages = append([]Package{}, source.Packages...)
		for i, p := range g.Packages {
			if p.ID == q.Package {
				p.SourceIdentity = strings.Repeat("a", 64)
				p.Identity = ""
				p.Identity = identity(p)
				g.Packages[i] = p
			}
		}
		if err := check(&g); err == nil {
			t.Fatal("source semantic package drift accepted")
		}
	})
	t.Run("registry-contributor-drift", func(t *testing.T) {
		b, err := readRegular(root, i18n.GlossaryCorpusPath)
		if err != nil {
			t.Fatal(err)
		}
		defer writeFixture(t, root, i18n.GlossaryCorpusPath, b)
		var corpus i18n.GlossarySourceCorpus
		if err := StrictJSON(b, &corpus); err != nil {
			t.Fatal(err)
		}
		for i := range corpus.Contributors {
			c := &corpus.Contributors[i]
			if c.ID == q.AssetID {
				c.Contexts[0].Text += " synthetic source prose drift"
				c.SourceSHA = i18n.ContributorIdentity(*c)
			}
		}
		corpus.Identity = i18n.CorpusIdentity(corpus)
		if err := i18n.ValidateGlossaryCorpus(&corpus); err != nil {
			t.Fatal("invalid negative fixture", err)
		}
		changed, _ := Encode(corpus)
		writeFixture(t, root, i18n.GlossaryCorpusPath, changed)
		if err := check(source); err == nil {
			t.Fatal("registered contributor drift accepted")
		}
	})
	for label, path := range map[string]string{"source-authority-ref-drift": r.SourceAuthority[0].Path, "raw-source-drift": q.Path} {
		t.Run(label, func(t *testing.T) {
			b, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, path, b)
			writeFixture(t, root, path, append(bytes.Clone(b), ' '))
			if err := check(source); err == nil {
				t.Fatal("exact source binding drift accepted")
			}
		})
	}
	t.Run("head-tamper", func(t *testing.T) {
		defer writeFixture(t, root, head, pointer)
		writeFixture(t, root, head, []byte(`{"path":"unknown","sha256":"unknown"}`))
		if _, _, err := CurrentPackageDocuments(root, source, q.Package); err == nil {
			t.Fatal("tampered head accepted")
		}
	})

	// The installed currentness envelope is exact. An engineering edit to the
	// checker cannot be silently authorized by unchanged representation topology.
	parser := sharedRepresentationParserPaths[0]
	code, err := readRegular(root, parser)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, parser, append(bytes.Clone(code), []byte("\n// isolated engineering repair epoch\n")...))
	if err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		return checkSharedRepresentationCurrent(root, source, r, v)
	}); err == nil {
		t.Fatal("installed parser binding was loosened", err)
	}
	if _, _, _, err := sharedRepresentationConsumedGeneration(root, q.Generation, newClosureValidation(root)); err != nil {
		t.Fatal("frozen consumed source depended on today's parser epoch", err)
	}
	if b, _ := readRegular(root, installedShared008Receipt); !bytes.Equal(b, before) {
		t.Fatal("receipt rewritten")
	}
	if b, _ := readRegular(root, head); !bytes.Equal(b, pointer) {
		t.Fatal("head rewritten")
	}
}

// Engineering cannot silently adopt re-attestation in the real checkout.
// The positive acceptance path is covered by isolated publication tests.
// This optional checkout diagnostic is not a deterministic first-real-use
// gate. Both settled current dependencies and canonical stale dependencies are
// valid observations; corruption, pending publication and unrelated errors fail.
func installedRepresentationDependencyStaleSignal(err error) bool {
	if err == nil {
		return false
	}
	switch err.Error() {
	case "dependency re-attestation is stale", "shared dependency re-attestation stale", "canonical representation dependencies stale; exact dependency re-attestation required":
		return true
	}
	const prefix = "shared representation dependencies stale; exact shared dependency re-attestation required: "
	return strings.HasPrefix(err.Error(), prefix) && validPath(strings.TrimPrefix(err.Error(), prefix))
}

func TestSharedRepresentationInstalledNeedsReattestation(t *testing.T) {
	for _, example := range []struct {
		message string
		stale   bool
	}{
		{"dependency re-attestation is stale", true}, {"shared dependency re-attestation stale", true},
		{"canonical representation dependencies stale; exact dependency re-attestation required", true},
		{"shared representation dependencies stale; exact shared dependency re-attestation required: internal/sitecontent/structured.go", true},
		{"unexpected EOF", false}, {"current authorization stale", false}, {"mutation=unknown: pending publication", false},
		{"unrelated error: dependency re-attestation is stale", false},
		{"shared representation dependencies stale; exact shared dependency re-attestation required: ../outside", false},
	} {
		if got := installedRepresentationDependencyStaleSignal(fmt.Errorf("%s", example.message)); got != example.stale {
			t.Fatalf("currentness signal classification: %q -> %t", example.message, got)
		}
	}
	root := repositoryRoot(t)
	if _, err := os.Stat(pathJoin(root, installedShared008Receipt)); os.IsNotExist(err) {
		t.Skip("installed authority unavailable")
	} else if err != nil {
		t.Fatal(err)
	}
	var observed error
	// Inspect representation/source currentness only, not the whole locale's
	// language plan, completion, or current Translation authorization assessment.
	err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		_, observed = LoadCurrent(root, v)
		if observed != nil && !installedRepresentationDependencyStaleSignal(observed) {
			return observed
		}
		return nil
	})
	if err != nil {
		t.Fatal("installed representation currentness diagnostic failed", err)
	}
	if observed == nil {
		t.Log("INSTALLED_REPRESENTATION_DEPENDENCIES=current")
	} else {
		t.Logf("INSTALLED_REPRESENTATION_DEPENDENCIES=stale CANONICAL_ERROR=%v", observed)
	}
}
