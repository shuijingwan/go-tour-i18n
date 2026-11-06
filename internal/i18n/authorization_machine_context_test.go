package i18n

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestAuthorizationMachineContextReuse(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, current, _, plan, ref, _ := authorizationMachineFixture(t)
	resolve := current.ReplacementResolver
	calls := 0
	current.ReplacementResolver = func(root string, c *Catalog, ref GlossaryArchiveReference) (*TranslationUnitReplacementResolution, error) {
		calls++
		return resolve(root, c, ref)
	}
	machines := map[string]*machineSelection{}
	var shared *machineReconciliationContext
	for i, item := range plan.Receipt.Units {
		s := machineSelection{receipt: plan.Receipt, unit: item, ref: ref}
		u, err := current.Unit(item.Replacement.CurrentUnitID)
		if err != nil {
			t.Fatal(err)
		}
		candidate, err := readSnapshotReferencedFile(root, item.Candidate.Path)
		if err != nil {
			t.Fatal(err)
		}
		x := s.snapshot(i + 1)
		if err := verifyAcceptedFrozenInputWithMachines(root, "zh-CN", u, x, candidate, "", machines, current); err != nil {
			t.Fatal(err)
		}
		selected := machines[ref.SHA256+u.ID]
		if shared != nil && shared != selected.ctx {
			t.Fatal("same receipt prepared a second context")
		}
		shared = selected.ctx
		// Differential full verifier, without sharing preparation, remains valid.
		if err := verifyAcceptedFrozenInput(root, "zh-CN", u, x, candidate, "", current); err != nil {
			t.Fatal(err)
		}
	}
	if calls != len(plan.Receipt.Units)+1 {
		t.Fatal("expected one shared preparation plus one fresh per differential Unit", calls)
	}
	first := plan.Receipt.Units[0]
	s := (&machineSelection{receipt: plan.Receipt, unit: first, ref: ref}).snapshot(1)
	u, _ := current.Unit(s.UnitID)
	candidate, _ := readSnapshotReferencedFile(root, first.Candidate.Path)
	for label, path := range map[string]string{"receipt": ref.Path, "validation": first.Validation.Path, "candidate": first.Candidate.Path, "historical-candidate": first.Predecessor.CandidatePath} {
		t.Run(label, func(t *testing.T) {
			before, err := readSnapshotReferencedFile(root, path)
			if err != nil {
				t.Fatal(err)
			}
			defer writeSnapshotSource(t, root, path, before)
			writeSnapshotSource(t, root, path, append(bytes.Clone(before), '\n'))
			if err := verifyAcceptedFrozenInputWithMachines(root, "zh-CN", u, s, candidate, "", machines, current); err == nil {
				t.Fatal("warm context concealed changed machine evidence")
			}
		})
	}
	bad := s
	badRef := ref
	badRef.SHA256 = strings.Repeat("0", 64)
	bad.RepresentationReconciliation = &badRef
	if err := verifyAcceptedFrozenInputWithMachines(root, "zh-CN", u, bad, candidate, "", machines, current); err == nil {
		t.Fatal("warm context accepted different expected SHA")
	}
}

func TestAuthorizationMachineAssessmentFreshGuards(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, c, _, snapshot := checkpointFixture(t)
	a, err := AssessCurrentTranslation(root, c, "zh-CN", snapshot)
	if err != nil || !a.Ready() {
		t.Fatal(a, err)
	}
	_, _, _, ordered, err := translationAuthorizationScope(c)
	if err != nil {
		t.Fatal(err)
	}
	shared, err := assessCurrentTranslationProofWithMachines(root, c, "zh-CN", a.Historical, map[string]*machineSelection{})
	if err != nil || !reflect.DeepEqual(a, shared) {
		t.Fatal("unchanged assessment identity/semantics", err)
	}
	for label, path := range map[string]string{
		"source": ordered[0].SourcePath, "glossary": a.Glossary.Path,
		"review": a.UnifiedReview.Path, "historical-reference": a.Historical.Finalization.QCResults[0].Path,
	} {
		t.Run(label, func(t *testing.T) {
			before, err := readSnapshotReferencedFile(root, path)
			if err != nil {
				t.Fatal(err)
			}
			defer writeSnapshotSource(t, root, path, before)
			writeSnapshotSource(t, root, path, append(bytes.Clone(before), '\n'))
			// Review JSON whitespace does not change identity; corrupt its schema.
			if label == "review" {
				writeSnapshotSource(t, root, path, bytes.Replace(before, []byte(UnifiedGlossaryReviewSchema), []byte("unsupported-schema"), 1))
			}
			if err := verifyMachineAssessmentInputs(root, c, a, ordered); err == nil {
				t.Fatal("changed assessment dependency accepted")
			}
		})
	}
	if err := verifyMachineAssessmentInputs(root, c, a, ordered); err != nil {
		t.Fatal("restored exact dependencies", err)
	}
}
