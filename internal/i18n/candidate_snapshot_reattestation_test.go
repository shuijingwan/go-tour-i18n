package i18n

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// This generic representation has a genuine frozen invocation glossary, as
// distinct from the shared Generation Bundle fixtures' current-only inputs.
type frozenCardAdapter struct{ futureCardAdapter }
type frozenCardInvocation struct {
	SourceSHA string
	Glossary  []byte
	Input     string
}

func (a frozenCardAdapter) VerifyGenerationOrigin(root, locale string, origin TranslationUnitGenerationOrigin, raw []byte) error {
	return a.verifyInvocation(root, locale, origin, raw, false)
}
func (a frozenCardAdapter) VerifyFrozenGenerationOrigin(root, locale string, origin TranslationUnitGenerationOrigin, raw []byte) error {
	return a.verifyInvocation(root, locale, origin, raw, true)
}
func (a frozenCardAdapter) verifyInvocation(root, locale string, origin TranslationUnitGenerationOrigin, raw []byte, frozen bool) error {
	b, err := readCompatibilityFile(root, origin.Bundle.Path)
	if err != nil || sum(b) != origin.Bundle.SHA256 {
		return fmt.Errorf("frozen invocation missing/changed")
	}
	var invocation frozenCardInvocation
	if err := json.Unmarshal(b, &invocation); err != nil {
		return err
	}
	source, _ := xml.Marshal(a.source)
	input, _, _ := a.ProtectedInput(nil)
	if invocation.SourceSHA != sum(append(source, '\n')) || invocation.Input != input {
		return fmt.Errorf("frozen invocation source/input changed")
	}
	b, err = readCompatibilityFile(root, origin.Result.Path)
	if err != nil || sum(b) != origin.Result.SHA256 || !bytes.Equal(b, raw) {
		return fmt.Errorf("frozen invocation output changed")
	}
	if !frozen {
		g, err := readCompatibilityFile(root, glossaryReviewGlossaryPath(locale))
		if err != nil || !bytes.Equal(g, invocation.Glossary) {
			return fmt.Errorf("original invocation glossary is no longer byte-identical")
		}
	}
	return nil
}

func frozenCardFinalizedFixture(t *testing.T, pkg string, affected ...int) (string, *Catalog, []string, string) {
	t.Helper()
	root, c, _ := futureContentFixture(t)
	d := c.TranslationPackages[0]
	d.ID = pkg
	ids := []string{}
	glossary, err := readCompatibilityFile(root, glossaryReviewGlossaryPath("zh-CN"))
	if err != nil {
		t.Fatal(err)
	}
	for i := range d.Units {
		v := &d.Units[i]
		card := v.Adapter.(futureCardAdapter).source
		card.Message = fmt.Sprintf("Read message number %d.", i+1)
		if i == 0 {
			card.Message = "Read the unique apple message."
		}
		if len(affected) > 0 && i < affected[0] {
			card.Message = fmt.Sprintf("Read apple message number %d.", i+1)
		}
		b, _ := xml.Marshal(card)
		v.Unit.Source = append(b, '\n')
		v.Unit.SourceSHA256 = sum(v.Unit.Source)
		v.Unit.ID = fmt.Sprintf("%s/card-%d", pkg, i+1)
		v.Unit.SourcePath = fmt.Sprintf("fixtures/%s/card-%d.xml", pkg, i+1)
		v.Adapter = frozenCardAdapter{futureCardAdapter{card}}
		writeSnapshotSource(t, root, v.Unit.SourcePath, v.Unit.Source)
		ids = append(ids, v.Unit.ID)
	}
	c = &Catalog{}
	if err := c.RegisterTranslationPackage(d); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage(pkg); err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		u, _ := c.Unit(id)
		raw, _, _ := u.Adapter.ProtectedInput(nil)
		invocation := frozenCardInvocation{u.SourceSHA256, glossary, raw}
		p := "data/generation-archives/" + id
		bundle, output := mustJSON(invocation), []byte(raw)
		writeSnapshotSource(t, root, p+".json", bundle)
		writeSnapshotSource(t, root, p+".txt", output)
		origin := TranslationUnitGenerationOrigin{GlossaryArchiveReference{p + ".json", sum(bundle)}, GlossaryArchiveReference{p + ".txt", sum(output)}}
		e, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, UnitIDs: []string{id}})
		if err != nil {
			t.Fatal(err)
		}
		if err := LandExistingGenerationBatch(root, c, "zh-CN", e.BatchID, map[string]TranslationUnitGenerationOrigin{id: origin}, map[string][]byte{id: output}); err != nil {
			t.Fatal(err)
		}
		if result, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID}); err != nil || result.ValidationPassed != 1 {
			t.Fatalf("frozen origin process: %+v %v", result, err)
		}
	}
	accepted := "frozen-accepted"
	futureSnapshot(t, root, c, accepted)
	for _, id := range ids {
		futureRecord(t, root, c, accepted, "", "A", "", []string{id})
	}
	if _, _, err := FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: accepted}); err != nil {
		t.Fatal(err)
	}
	return root, c, ids, accepted
}

func TestCurrentTranslationSnapshotFrozenInvocationMigration(t *testing.T) {
	for _, pkg := range []string{"registered-cards-alpha", "unrelated-cards-beta"} {
		t.Run(pkg, func(t *testing.T) {
			root, c, ids, accepted := frozenCardFinalizedFixture(t, pkg)
			checkpointGlossary(t, root, "preferred:\n  apple: 苹果\n", "snapshot-current-review")
			a, err := AssessCurrentTranslation(root, c, "zh-CN", accepted)
			if err != nil || a.Compatible != 2 || a.Affected != 1 || a.Stale != 0 || a.Unknown != 0 || a.Units[0].Validation != "passed" {
				t.Fatalf("current assessment: %+v %v", a, err)
			}
			if _, _, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", accepted); err == nil {
				t.Fatal("affected complete authorization accepted")
			}
			latest, err := selectLatestRetranslationUnits(root, c, "zh-CN")
			if err != nil {
				t.Fatal(err)
			}
			g, _ := LoadGlossary(root, "zh-CN")
			compatibility := PrepareGlossaryCompatibility(root, "zh-CN", c)
			if r := compatibility.Resolve(a.Historical.Finalization.GlossarySHA256, a.Glossary.SHA256, "tu:"+ids[0]); r.Status == "compatible" || r.Status == "exact" {
				t.Fatal("fixture unexpectedly has historical adapted compatibility")
			}
			if _, err := validateSelectedRetranslationCandidate(root, c, "zh-CN", g, latest.ordered[0], latest.selectedByID[ids[0]], compatibility); err == nil || !strings.Contains(err.Error(), "invocation glossary") {
				t.Fatalf("old path did not reproduce real failure: %v", err)
			}
			s := futureSnapshot(t, root, c, "frozen-current")
			if s.CurrentReattestation == nil || !reflect.DeepEqual(s.CurrentReattestation, a) {
				t.Fatal("Snapshot did not freeze exact current validation/assessment")
			}
			p, err := BuildQualityCheckPreflight(root, c, QualityCheckPreflightOptions{Locale: "zh-CN", SnapshotID: s.SnapshotID, PreviousSnapshotID: accepted})
			if err != nil || p.CarryForwardCount != 2 || p.QualityCheckRequiredCount != 1 || p.RevisionRequiredCount != 0 || p.Pending[0].UnitID != ids[0] {
				t.Fatalf("affected-only preflight: %+v %v", p, err)
			}
			zip, m, err := ExportQualityCheckReviewerBundle(root, c, QualityCheckReviewerBundleOptions{Locale: "zh-CN", SnapshotID: s.SnapshotID, PreviousSnapshotID: accepted})
			if err != nil || m.UnitCount != 1 || m.WorkingSetKind != UnitKindPage || m.StartIndex != 1 || m.EndIndex != 1 {
				t.Fatalf("only affected Reviewer work: %+v %v", m, err)
			}
			if _, err := VerifyCurrentQualityCheckReviewerBundle(root, c, zip); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(qualityCheckResultsPath(root, "zh-CN", s.SnapshotID)); !os.IsNotExist(err) {
				t.Fatal("machine assessment manufactured a Reviewer decision")
			}
			after, err := selectLatestRetranslationUnits(root, c, "zh-CN")
			if err != nil || !reflect.DeepEqual(latest.selectedByID, after.selectedByID) {
				t.Fatal("successor planning required Generation/replacement")
			}
			// A new glossary makes this frozen review input stale immediately;
			// its reader must not attempt pre-checkpoint compatibility recovery.
			checkpointGlossary(t, root, "# next byte identity\n", "snapshot-next-review")
			if _, err := readQualityCheckSnapshotForReview(root, "zh-CN", s.SnapshotID, c); err == nil || !strings.Contains(err.Error(), "current re-attestation glossary is stale") {
				t.Fatalf("frozen Snapshot re-entered historical glossary recovery: %v", err)
			}
		})
	}
}

type snapshotRejectingCardAdapter struct{ frozenCardAdapter }

func (a snapshotRejectingCardAdapter) Validate([]byte, *Glossary) error {
	return fmt.Errorf("current validator rejects old candidate")
}

func TestCurrentTranslationSnapshotMigrationFailClosed(t *testing.T) {
	for _, kind := range []string{"stale", "unknown", "validation", "candidate", "reviewer", "finalization"} {
		t.Run(kind, func(t *testing.T) {
			root, c, _, accepted := frozenCardFinalizedFixture(t, "negative-registered-cards")
			checkpointGlossary(t, root, "preferred:\n  apple: 苹果\n", "snapshot-current-review")
			s := futureSnapshot(t, root, c, "frozen-current")
			v := &c.Adapted[0]
			switch kind {
			case "stale":
				writeSnapshotSource(t, root, v.Unit.SourcePath, append(v.Unit.Source, '\n'))
			case "unknown":
				v.Adapter = struct{ TranslationUnitAdapter }{v.Adapter}
			case "validation":
				v.Adapter = snapshotRejectingCardAdapter{v.Adapter.(frozenCardAdapter)}
			case "candidate":
				writeSnapshotSource(t, root, s.Units[0].CandidatePath, []byte("changed candidate\n"))
			case "reviewer":
				writeSnapshotSource(t, root, filepath.ToSlash(strings.TrimPrefix(qualityCheckResultsPath(root, "zh-CN", accepted), root+string(filepath.Separator))), []byte("{corrupt\n"))
			case "finalization":
				writeSnapshotSource(t, root, filepath.ToSlash(strings.TrimPrefix(qualityCheckFinalizationPath(root, "zh-CN", accepted), root+string(filepath.Separator))), []byte("{corrupt\n"))
			}
			c.TranslationPackages[0].Units[0] = *v
			if _, _, err := CreateQualityCheckCandidateSnapshot(root, c, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "rejected"}); err == nil {
				t.Fatal("unsafe snapshot created")
			}
			if _, err := os.Stat(filepath.Join(root, "data/quality-check-snapshots/zh-CN/rejected")); !os.IsNotExist(err) {
				t.Fatal("rejected snapshot mutated storage")
			}
			if _, err := BuildQualityCheckPreflight(root, c, QualityCheckPreflightOptions{Locale: "zh-CN", SnapshotID: s.SnapshotID, PreviousSnapshotID: accepted}); err == nil {
				t.Fatal("unsafe successor planning accepted")
			}
		})
	}
}
