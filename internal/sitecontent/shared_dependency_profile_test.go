package sitecontent

import (
	"bytes"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const profileExtendedReceipt = SharedRepresentationRoot + "receipts/267dcc42ec34b85435b06fbeae10b3f495a05abf3a1c5b24dec798003bb7dd51.json"
const profileExtendedReceiptSHA = "b23f992f3c55b235178d8d8a8112814bc7eb5e510fd9f09d4b5b3d505522506d"

// Reuse immutable, exact recorded inventories as historical profile fixtures.
// The consumed Bundle coordinate is parameterized, as it is in the protocol;
// no historical receipt is rewritten or installed as synthetic authority.
func recordedSharedProfilePaths(t *testing.T, ref Reference, bundle string) []string {
	t.Helper()
	var saved SharedRepresentationSuccessor
	if err := readReference(repositoryRoot(t), ref, &saved); err != nil {
		t.Fatal(err)
	}
	var generation StructuredGeneration
	if err := readReference(repositoryRoot(t), saved.Request.Generation, &generation); err != nil {
		t.Fatal(err)
	}
	paths := []string{}
	for _, in := range saved.Inputs {
		p := in.Reference.Path
		if p == generation.Bundle.Path {
			p = bundle
		}
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

func sharedProfileFixture(t *testing.T) (string, SharedRepresentationSuccessor, string) {
	t.Helper()
	real, root := repositoryRoot(t), t.TempDir()
	var base SharedRepresentationSuccessor
	if err := readReference(real, Reference{profileExtendedReceipt, profileExtendedReceiptSHA}, &base); err != nil {
		t.Fatal(err)
	}
	var generation StructuredGeneration
	if err := readReference(real, base.Request.Generation, &generation); err != nil {
		t.Fatal(err)
	}
	paths := uniqueStrings(append(append(append([]string{}, sharedRepresentationParserPaths...), sharedReattestationValidationPaths...), i18n.GlossaryCorpusPath, base.Request.Generation.Path, generation.Bundle.Path))
	for _, p := range paths {
		b, err := readRegular(real, p)
		if err != nil {
			t.Fatal(err)
		}
		st, err := os.Stat(pathJoin(real, p))
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, p, b)
		if err := os.Chmod(pathJoin(root, p), st.Mode().Perm()); err != nil {
			t.Fatal(err)
		}
	}
	return root, base, generation.Bundle.Path
}

func snapshotSharedProfile(t *testing.T, root string, paths []string) []ShellRepresentationInput {
	t.Helper()
	inputs := []ShellRepresentationInput{}
	for _, p := range paths {
		ref, err := ReferenceFile(root, p)
		if err != nil {
			t.Fatal(err)
		}
		st, err := os.Stat(pathJoin(root, p))
		if err != nil {
			t.Fatal(err)
		}
		inputs = append(inputs, ShellRepresentationInput{Reference: ref, Mode: uint32(st.Mode())})
	}
	return inputs
}

func sharedProfileValidationIndex(t *testing.T, inputs []ShellRepresentationInput) int {
	t.Helper()
	for i, in := range inputs {
		for _, path := range sharedReattestationValidationPaths {
			if in.Reference.Path == path {
				return i
			}
		}
	}
	t.Fatal("fixture has no frozen validation dependency")
	return -1
}

func TestSharedFrozenDependencyProfiles(t *testing.T) {
	root, base, bundle := sharedProfileFixture(t)
	contract := mustSharedFixtureBytes(t, repositoryRoot(t), "docs/SHARED_STRUCTURED_REPRESENTATION_REATTESTATION.md")
	latest, err := sharedRepresentationCurrentInputs(root, SharedRepresentationSuccessor{Request: base.Request}, successorValidation(root, nil))
	if err != nil {
		t.Fatal("new creation profile must be explicitly recognized", err)
	}
	latestPaths := []string{}
	for _, in := range latest {
		latestPaths = append(latestPaths, in.Reference.Path)
	}
	for _, tc := range []struct {
		name, fingerprint string
		paths             []string
	}{
		{"legacy", sharedLegacyProfileSHA, recordedSharedProfilePaths(t, Reference{installedShared008Receipt, installedShared008ReceiptSHA}, bundle)},
		{"historical-extended", sharedExtendedV1ProfileSHA, recordedSharedProfilePaths(t, Reference{profileExtendedReceipt, profileExtendedReceiptSHA}, bundle)},
		{"latest", sharedAuditV2ProfileSHA, latestPaths},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := base
			r.Inputs = snapshotSharedProfile(t, root, tc.paths)
			implementation := []string{}
			for _, p := range tc.paths {
				if p != i18n.GlossaryCorpusPath && p != bundle {
					implementation = append(implementation, p)
				}
			}
			if identity(implementation) != tc.fingerprint {
				t.Fatal("canonical epoch fingerprint drifted")
			}
			if !bytes.Contains(contract, []byte(tc.fingerprint)) {
				t.Fatal("canonical docs and recognized profile differ")
			}
			// SHA evolution in an existing implementation path is eligible;
			// source/corpus/Bundle and every path/mode remain unchanged.
			driftPath := sharedRepresentationParserPaths[0]
			if tc.name == "historical-extended" {
				driftPath = r.Inputs[sharedProfileValidationIndex(t, r.Inputs)].Reference.Path
			}
			for i := range r.Inputs {
				if r.Inputs[i].Reference.Path == driftPath {
					r.Inputs[i].Reference.SHA256 = digest([]byte("synthetic earlier implementation"))
				}
			}
			v := successorValidation(root, nil)
			current, err := sharedRepresentationCurrentInputs(root, r, v)
			if err != nil {
				t.Fatal(err)
			}
			want := snapshotSharedProfile(t, root, tc.paths)
			if !reflect.DeepEqual(current, want) {
				t.Fatal("frozen profile expanded or reconstruction changed modes")
			}
			attestation, err := buildSharedReattestation(root, Reference{}, Reference{}, &r, current, nil, v)
			if err != nil || len(attestation.Changes) != 1 {
				t.Fatal("same-profile SHA drift not eligible", err)
			}
			if !reflect.DeepEqual(attestation.Base, r) || !reflect.DeepEqual(attestation.OldInputs, r.Inputs) || !reflect.DeepEqual(attestation.CurrentInputs, want) || len(attestation.Added) != 0 || len(attestation.Removed) != 0 {
				t.Fatal("base/representation/frozen envelope changed")
			}
			validationPaths := append([]string{}, sharedReattestationValidationPaths...)
			sort.Strings(validationPaths)
			if !reflect.DeepEqual(attestation.ValidationInputs, snapshotSharedProfile(t, root, validationPaths)) {
				t.Fatal("today's validation_inputs were not independently bound")
			}
			if tc.name != "latest" && len(current) >= len(latest) {
				t.Fatal("historical inputs grew to today's creation profile")
			}
			if err := v.guard.verify(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSharedFrozenDependencyProfileRejectsTampering(t *testing.T) {
	root, base, bundle := sharedProfileFixture(t)
	frozen := snapshotSharedProfile(t, root, recordedSharedProfilePaths(t, Reference{profileExtendedReceipt, profileExtendedReceiptSHA}, bundle))
	for _, name := range []string{"mixed-legacy", "missing-path", "missing-validation-path", "unknown-path", "duplicate", "wrong-mode", "absent", "unsorted", "foreign-corpus", "foreign-bundle", "invalid-sha", "missing-inventory", "physical-mode", "physical-absence", "corpus-sha"} {
		t.Run(name, func(t *testing.T) {
			r := base
			r.Inputs = append([]ShellRepresentationInput{}, frozen...)
			path := sharedRepresentationParserPaths[0]
			index := sort.Search(len(r.Inputs), func(i int) bool { return r.Inputs[i].Reference.Path >= path })
			switch name {
			case "mixed-legacy":
				paths := recordedSharedProfilePaths(t, Reference{installedShared008Receipt, installedShared008ReceiptSHA}, bundle)
				paths = append(paths, sharedReattestationValidationPaths[0])
				sort.Strings(paths)
				r.Inputs = snapshotSharedProfile(t, root, paths)
			case "missing-path":
				r.Inputs = append(r.Inputs[:index], r.Inputs[index+1:]...)
			case "missing-validation-path":
				i := sharedProfileValidationIndex(t, r.Inputs)
				r.Inputs = append(r.Inputs[:i], r.Inputs[i+1:]...)
			case "unknown-path":
				// Existing readable bytes must not make an unknown path legitimate.
				writeFixture(t, root, "internal/unknown.go", []byte("foreign"))
				r.Inputs = append(r.Inputs, ShellRepresentationInput{Reference: Reference{"internal/unknown.go", digest([]byte("foreign"))}, Mode: 0644})
				sort.Slice(r.Inputs, func(i, j int) bool { return r.Inputs[i].Reference.Path < r.Inputs[j].Reference.Path })
			case "duplicate":
				r.Inputs = append(r.Inputs[:1], append([]ShellRepresentationInput{r.Inputs[0]}, r.Inputs[1:]...)...)
			case "wrong-mode":
				r.Inputs[index].Mode ^= 0111
			case "absent":
				r.Inputs[index].Absent = true
			case "unsorted":
				r.Inputs[0], r.Inputs[1] = r.Inputs[1], r.Inputs[0]
			case "foreign-corpus", "foreign-bundle", "corpus-sha":
				for i := range r.Inputs {
					p := r.Inputs[i].Reference.Path
					if name == "corpus-sha" && p == i18n.GlossaryCorpusPath {
						r.Inputs[i].Reference.SHA256 = digest([]byte("foreign corpus"))
					} else if (name == "foreign-corpus" && p == i18n.GlossaryCorpusPath) || (name == "foreign-bundle" && p == bundle) {
						writeFixture(t, root, "data/foreign.json", mustSharedFixtureBytes(t, root, p))
						if err := os.Chmod(pathJoin(root, "data/foreign.json"), os.FileMode(r.Inputs[i].Mode).Perm()); err != nil {
							t.Fatal(err)
						}
						r.Inputs[i].Reference.Path = "data/foreign.json"
					}
				}
				sort.Slice(r.Inputs, func(i, j int) bool { return r.Inputs[i].Reference.Path < r.Inputs[j].Reference.Path })
			case "invalid-sha":
				r.Inputs[index].Reference.SHA256 = "invalid"
			case "missing-inventory":
				r.Inputs = nil
			case "physical-mode":
				st, _ := os.Stat(pathJoin(root, path))
				if err := os.Chmod(pathJoin(root, path), st.Mode().Perm()^0111); err != nil {
					t.Fatal(err)
				}
				defer os.Chmod(pathJoin(root, path), st.Mode().Perm())
			case "physical-absence":
				b := mustSharedFixtureBytes(t, root, path)
				st, _ := os.Stat(pathJoin(root, path))
				if err := os.Remove(pathJoin(root, path)); err != nil {
					t.Fatal(err)
				}
				defer func() { writeFixture(t, root, path, b); _ = os.Chmod(pathJoin(root, path), st.Mode().Perm()) }()
			}
			_, err := sharedRepresentationCurrentInputs(root, r, successorValidation(root, nil))
			if err == nil {
				t.Fatal("forged or drifted profile accepted", name)
			}
			switch name {
			case "mixed-legacy", "missing-path", "missing-validation-path", "unknown-path":
				if !strings.Contains(err.Error(), "unrecognized canonical profile") {
					t.Fatal("profile was not rejected by its legitimacy gate", err)
				}
			case "foreign-corpus", "foreign-bundle":
				if !strings.Contains(err.Error(), "corpus/consumed Bundle") {
					t.Fatal("foreign path was not rejected by mandatory authority binding", err)
				}
			}
		})
	}
}

func TestSharedReattestationActive267Readonly(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	ref := Reference{profileExtendedReceipt, profileExtendedReceiptSHA}
	before := mustSharedFixtureBytes(t, root, ref.Path)
	var base SharedRepresentationSuccessor
	if err := readReference(root, ref, &base); err != nil {
		t.Fatal(err)
	}
	headPath := sharedRepresentationHead(base.Request.Package, base.Request.AssetID)
	headBefore := mustSharedFixtureBytes(t, root, headPath)
	p, err := PlanSharedRepresentationReattestation(root, ref)
	if err != nil {
		t.Fatal("readonly historical-profile prospective rejected", err)
	}
	if !p.Eligible || p.BaseCurrent || len(p.Authority.Changes) == 0 || !reflect.DeepEqual(p.Authority.Base, base) || !reflect.DeepEqual(p.Authority.OldInputs, base.Inputs) || len(p.Authority.Added) != 0 || len(p.Authority.Removed) != 0 {
		t.Fatal("real dependency drift or frozen base invariant lost")
	}
	if _, err := sharedDependencyChanges(base.Inputs, p.Authority.CurrentInputs); err != nil {
		t.Fatal(err)
	}
	validationPaths := append([]string{}, sharedReattestationValidationPaths...)
	sort.Strings(validationPaths)
	if !reflect.DeepEqual(p.Authority.ValidationInputs, snapshotSharedProfile(t, root, validationPaths)) {
		t.Fatal("real prospective omitted current validation authority")
	}
	if !bytes.Equal(before, mustSharedFixtureBytes(t, root, ref.Path)) || !bytes.Equal(headBefore, mustSharedFixtureBytes(t, root, headPath)) {
		t.Fatal("readonly prospective modified base or head")
	}
	for _, in := range p.Authority.CurrentInputs {
		if strings.HasPrefix(in.Reference.Path, "internal/languageaudit/") {
			t.Fatal("new checker path leaked into historical currentness inventory")
		}
	}
	t.Logf("READONLY_PROFILE_PASS old=%d current=%d validation=%d changes=%d", len(base.Inputs), len(p.Authority.CurrentInputs), len(p.Authority.ValidationInputs), len(p.Authority.Changes))
}
