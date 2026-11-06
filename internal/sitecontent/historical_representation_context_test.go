package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const surface009HistoricalCheckpoint = "data/translation-authorizations/zh-CN/a03a8d8174fbafd8a01589a162af42f4c3adbb816f4131f2f2e0abbbe16d8ca2/site-v2-shell-qc-004/3e525d19caf38b758445a4af65b4b41a99800329fc932ae9f765635631e2c653.json"

// Only immutable encodings in a disposable repository: no lifecycle commands
// or real authority publication are needed to exercise historical addressing.
func historicalReattestationFixture(t *testing.T) (string, i18n.GlossaryArchiveReference, Reference, *ShellRepresentationReattestation) {
	t.Helper()
	root := t.TempDir()
	old := ShellRepresentationInput{Reference: Reference{shellRepresentationParserPaths[0], digest([]byte("original parser"))}, Mode: 0644}
	base := ShellRepresentationSuccessor{
		Schema: ShellRepresentationSchema,
		Basis:  ShellRepresentationBasis{LocaleScope: "all-locales", Package: Package{ID: "synthetic-shell"}, Source: Reference{"synthetic-source.md", digest([]byte("raw source"))}},
		New:    ShellRepresentation{Parser: "synthetic-parser", Identity: digest([]byte("frozen representation"))},
		Inputs: []ShellRepresentationInput{old},
	}
	base.Identity = identity(base)
	baseRef := shellRepresentationReceipt(&base)
	baseBytes, _ := Encode(base)
	writeFixture(t, root, baseRef.Path, baseBytes)
	basePointer, _ := Encode(baseRef)
	makeNode := func(version string, previous *Reference) *ShellRepresentationReattestation {
		next := old
		next.Reference.SHA256 = digest([]byte(version))
		r := &ShellRepresentationReattestation{
			Schema:      ShellRepresentationReattestationSchema,
			BasePointer: Reference{ShellRepresentationCurrent, digest(basePointer)},
			BaseReceipt: baseRef, BaseIdentity: base.Identity, Basis: base.Basis,
			Representation: base.New, Released: base.Released, OldInputs: base.Inputs,
			CurrentInputs: []ShellRepresentationInput{next},
			Changes:       []ShellRepresentationDependencyChange{{Old: old, Current: next}}, Previous: previous,
		}
		r.Identity = identity(*r)
		b, _ := Encode(r)
		writeFixture(t, root, shellReattestationReceipt(r).Path, b)
		return r
	}
	prior := makeNode("first parser update", nil)
	priorRef := shellReattestationReceipt(prior)
	priorPointer, _ := Encode(priorRef)
	next := makeNode("second parser update", &priorRef)
	head := shellReattestationReceipt(next)
	b, _ := Encode(head)
	path := shellReattestationPath(base.Identity)
	writeFixture(t, root, path, b)
	return root, i18n.GlossaryArchiveReference{Path: path, SHA256: digest(priorPointer)}, priorRef, next
}

func TestHistoricalShellReattestationContext(t *testing.T) {
	root, ref, old, current := historicalReattestationFixture(t)
	paths := []string{ref.Path, old.Path, shellReattestationReceipt(current).Path, current.BaseReceipt.Path}
	before := map[string][]byte{}
	for _, p := range paths {
		var err error
		before[p], err = readRegular(root, p)
		if err != nil {
			t.Fatal(err)
		}
	}
	b, err := readHistoricalLanguageContext(root, "zh-CN", ref)
	want, _ := Encode(old)
	if err != nil || !bytes.Equal(b, want) || digest(b) != ref.SHA256 || bytes.Equal(b, before[ref.Path]) {
		t.Fatal("did not prove exact old pointer bytes through immutable previous lineage", err)
	}
	for p, original := range before {
		after, err := readRegular(root, p)
		if err != nil || !bytes.Equal(after, original) {
			t.Fatal("historical proof mutated filesystem evidence", p, err)
		}
	}
}

func TestHistoricalShellReattestationContextFailClosed(t *testing.T) {
	for _, name := range []string{"missing-head", "malformed-head", "noncanonical-head", "missing-base", "missing-ancestor", "corrupt-ancestor", "wrong-base", "wrong-package", "wrong-representation", "wrong-base-pointer", "invalid-delta", "disconnected-ancestor", "wrong-frozen-sha", "unrelated-path", "pending"} {
		t.Run(name, func(t *testing.T) {
			root, ref, prior, head := historicalReattestationFixture(t)
			reseal := false
			switch name {
			case "missing-head":
				if err := os.Remove(pathJoin(root, ref.Path)); err != nil {
					t.Fatal(err)
				}
			case "malformed-head":
				writeFixture(t, root, ref.Path, []byte("{invalid"))
			case "noncanonical-head":
				b, _ := readRegular(root, ref.Path)
				writeFixture(t, root, ref.Path, append(b, '\n'))
			case "missing-base", "missing-ancestor":
				path := prior.Path
				if name == "missing-base" {
					path = head.BaseReceipt.Path
				}
				if err := os.Remove(pathJoin(root, path)); err != nil {
					t.Fatal(err)
				}
			case "corrupt-ancestor":
				writeFixture(t, root, prior.Path, []byte("changed historical receipt"))
			case "wrong-base":
				head.BaseIdentity = digest([]byte("different base"))
				reseal = true
			case "wrong-package":
				head.Basis.Package.ID = "unrelated-package"
				reseal = true
			case "wrong-representation":
				head.Representation.Identity = digest([]byte("different topology"))
				reseal = true
			case "wrong-base-pointer":
				head.BasePointer.SHA256 = digest([]byte("different base pointer"))
				reseal = true
			case "invalid-delta":
				head.Changes = nil
				reseal = true
			case "disconnected-ancestor":
				head.Previous = nil // Old receipt still exists, but is an orphan.
				reseal = true
			case "wrong-frozen-sha":
				ref.SHA256 = digest([]byte("unproven historical pointer"))
			case "unrelated-path":
				ref.Path = "data/synthetic-unrelated/current.json"
			case "pending":
				writeFixture(t, root, ShellRepresentationReattestationPending, []byte("pending publication"))
			}
			if reseal {
				head.Identity = ""
				head.Identity = identity(*head)
				b, _ := Encode(head)
				writeFixture(t, root, shellReattestationReceipt(head).Path, b)
				p, _ := Encode(shellReattestationReceipt(head))
				writeFixture(t, root, ref.Path, p)
			}
			if _, err := readHistoricalLanguageContext(root, "zh-CN", ref); err == nil {
				t.Fatal("unproven historical context accepted")
			}
		})
	}
}

func TestSurface009HistoricalRepresentationEvidenceReadonly(t *testing.T) {
	if os.Getenv("GO_TOUR_SURFACE009_READONLY") != "1" {
		t.Skip("explicit Surface009 readonly historical provenance proof")
	}
	root := repositoryRoot(t)
	var a i18n.CurrentTranslationAuthorization
	b, err := readRegular(root, surface009HistoricalCheckpoint)
	if err != nil {
		t.Fatal(err)
	}
	if err := StrictJSON(b, &a); err != nil {
		t.Fatal(err)
	}
	path := shellReattestationPath("1c80e2c83c4225b1bea8834a92e77e5340f01cab1166fa67b4046cb07f2c096d")
	var bound i18n.GlossaryArchiveReference
	for _, ref := range a.Historical.References {
		if ref.Path == path {
			bound = ref
		}
	}
	if bound.SHA256 != "387cb88adfafaa5585ec04e670eab64eb078fe157947c029f0741b7cd4a548bd" {
		t.Fatal("unexpected historical checkpoint binding", bound)
	}
	frozen, err := readHistoricalLanguageContext(root, "zh-CN", bound)
	if err != nil || digest(frozen) != bound.SHA256 {
		t.Fatal("exact checkpoint context could not be proved", err)
	}
	var old Reference
	if err := StrictJSON(frozen, &old); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, ref := range a.Historical.References {
		found = found || (ref.Path == old.Path && ref.SHA256 == old.SHA256)
	}
	if !found {
		t.Fatal("resolved old receipt not sealed in checkpoint", old)
	}
	current, err := readRegular(root, path)
	if err != nil || bytes.Equal(current, frozen) {
		t.Fatal("current successor was substituted for historical context", err)
	}
	t.Logf("CHECKPOINT_BINDING=%s HISTORICAL_RECEIPT=%s CURRENT_POINTER_SHA=%s", bound.SHA256, old.Path, digest(current))
}

// A successful historical proof cannot make an old machine-authority epoch
// current. Keep this diagnosis separate from the full selection proof, which
// must continue to FAIL until the canonical current gate actually passes.
func TestSurface009CurrentAuthorizationReadonlyDiagnosis(t *testing.T) {
	if os.Getenv("GO_TOUR_SURFACE009_READONLY") != "1" {
		t.Skip("explicit Surface009 readonly checkpoint diagnosis")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	var sealed i18n.CurrentTranslationAuthorization
	b, err := readRegular(root, surface009HistoricalCheckpoint)
	if err != nil {
		t.Fatal(err)
	}
	if err := StrictJSON(b, &sealed); err != nil {
		t.Fatal(err)
	}
	c, err := newClosureValidation(root).packageCatalog(sealed.Package)
	if err != nil {
		t.Fatal(err)
	}
	current, err := i18n.AssessCurrentTranslation(root, c, sealed.Locale, sealed.Historical.Finalization.SnapshotID)
	if err != nil {
		t.Fatal("current assessment after exact historical integrity proof", err)
	}
	var oldMap, currentMap map[string]any
	if err := json.Unmarshal(b, &oldMap); err != nil {
		t.Fatal(err)
	}
	now, _ := Encode(current)
	if err := json.Unmarshal(now, &currentMap); err != nil {
		t.Fatal(err)
	}
	for _, diff := range exactScopeDifferences(oldMap, currentMap, "authorization") {
		t.Logf("CURRENT_AUTHORIZATION_DIFF=%s OLD=%v CURRENT=%v", diff.Field, diff.Old, diff.Current)
	}
	t.Logf("CURRENT_ASSESSMENT compatible=%d affected=%d stale=%d unknown=%d", current.Compatible, current.Affected, current.Stale, current.Unknown)
	if _, err := i18n.CheckCurrentTranslationAuthorization(root, c, surface009HistoricalCheckpoint); err != nil {
		t.Logf("CURRENT_AUTHORIZATION_GATE_REJECTED=%v; no new checkpoint published", err)
	}
}
