package i18n

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type changedSnapshotInputAdapter struct{ frozenCardAdapter }

func (a changedSnapshotInputAdapter) ProtectedInput(g *Glossary) (string, []string, error) {
	text, tokens, err := a.frozenCardAdapter.ProtectedInput(g)
	return "Changed formal input " + text, tokens, err
}

type glossaryChangedSnapshotInputAdapter struct{ frozenCardAdapter }

func (a glossaryChangedSnapshotInputAdapter) ProtectedInput(g *Glossary) (string, []string, error) {
	text, tokens, err := a.frozenCardAdapter.ProtectedInput(g)
	if g != nil && g.Preferred["unmentioned"] != "" {
		text = "Current glossary formal instruction " + text
	}
	return text, tokens, err
}

func TestSnapshotArchivedCandidateCompatibleCarry(t *testing.T) {
	for _, kind := range []string{"compatible", "no authorization", "affected", "stale", "unknown", "formal input changed", "glossary changes formal input", "candidate", "source identity", "validation", "archive", "full Review", "partial scope"} {
		t.Run(kind, func(t *testing.T) {
			root, c, ids, accepted := frozenCardFinalizedFixture(t, "carry-fixture")
			checkpointGlossary(t, root, "# byte-only glossary refresh\n", "carry-current-review")
			var authorization *CurrentTranslationAuthorization
			if kind != "no authorization" {
				var err error
				authorization, _, err = EstablishCurrentTranslationAuthorization(root, c, "zh-CN", accepted)
				if err != nil {
					t.Fatal(err)
				}
			}
			latest, err := selectLatestRetranslationUnits(root, c, "zh-CN")
			if err != nil {
				t.Fatal(err)
			}
			choice := latest.selectedByID[ids[0]]
			v := &c.Adapted[0]
			classification := ""
			switch kind {
			case "affected":
				checkpointGlossary(t, root, "preferred:\n  apple: 苹果\n", "carry-affected-review")
				classification = "affected"
			case "stale":
				writeSnapshotSource(t, root, v.Unit.SourcePath, append(bytes.Clone(v.Unit.Source), '\n'))
				classification = "stale"
			case "unknown":
				// Keep both provenance verifiers but remove visibility/authority.
				v.Adapter = struct {
					TranslationUnitAdapter
					ArchivedGenerationAdapter
					FrozenGenerationAdapter
				}{v.Adapter, v.Adapter.(ArchivedGenerationAdapter), v.Adapter.(FrozenGenerationAdapter)}
				classification = "unknown"
			case "formal input changed":
				v.Adapter = changedSnapshotInputAdapter{v.Adapter.(frozenCardAdapter)}
			case "glossary changes formal input":
				checkpointGlossary(t, root, "preferred:\n  unmentioned: 测试术语\n", "carry-formal-input-review")
				v.Adapter = glossaryChangedSnapshotInputAdapter{v.Adapter.(frozenCardAdapter)}
			case "candidate":
				writeSnapshotSource(t, root, authorization.Units[0].Snapshot.CandidatePath, []byte("changed candidate\n"))
			case "source identity":
				choice.manifest.SourceSHA256 = strings.Repeat("0", 64)
			case "validation":
				p := authorization.Units[0].Snapshot.ValidationPath
				b, err := readCompatibilityFile(root, p)
				if err != nil {
					t.Fatal(err)
				}
				writeSnapshotSource(t, root, p, append(b, '\n'))
			case "archive":
				writeSnapshotSource(t, root, choice.manifest.GenerationOrigin.Bundle.Path, []byte("corrupt invocation\n"))
			case "full Review":
				writeSnapshotSource(t, root, authorization.UnifiedReview.Path, []byte("corrupt full Review\n"))
			case "partial scope":
				if err := c.UseTranslationUnitScope(ids[:1]); err != nil {
					t.Fatal(err)
				}
			}
			c.TranslationPackages[0].Units[0] = *v
			if kind == "glossary changes formal input" {
				// Even a real current semantic authorization cannot substitute
				// for exact formal-input identity in archived candidate carry.
				if a, _, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", accepted); err != nil || !a.Ready() {
					t.Fatalf("semantic compatibility fixture: %+v %v", a, err)
				}
			}
			if classification != "" {
				a, err := AssessCurrentTranslation(root, c, "zh-CN", accepted)
				if err != nil || a.Units[0].Classification != classification {
					t.Fatalf("classification=%s assessment=%+v err=%v", classification, a, err)
				}
			}
			u, err := c.Unit(ids[0])
			if err != nil {
				t.Fatal(err)
			}
			g, err := LoadGlossary(root, "zh-CN")
			if err != nil {
				t.Fatal(err)
			}
			before := snapshotDirectoryBytes(t, root)
			e, err := validateSelectedRetranslationCandidate(root, c, "zh-CN", g, u, choice)
			if kind == "compatible" {
				if err != nil || sum(e.candidate) != authorization.Units[0].Snapshot.CandidateSHA256 || e.attempt != 1 {
					t.Fatalf("compatible exact accepted candidate: %+v %v", e, err)
				}
				s, err := prepareQualityCheckCandidateSnapshot(root, c, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "readonly-compatible"})
				if err != nil || s.UnitCount != len(ids) {
					t.Fatalf("complete readonly Snapshot selection: %+v %v", s, err)
				}
			} else if err == nil {
				t.Fatal("unsafe carry accepted", kind)
			}
			if kind == "glossary changes formal input" {
				if _, err := prepareQualityCheckCandidateSnapshot(root, c, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "readonly-input-drift"}); err == nil || !strings.Contains(err.Error(), "formal input changed") {
					t.Fatalf("complete-set re-attestation bypassed input drift: %v", err)
				}
			}
			if !reflect.DeepEqual(before, snapshotDirectoryBytes(t, root)) {
				t.Fatal("readonly candidate selection mutated evidence")
			}
		})
	}
}

func TestSnapshotArchivedCandidateRejectsRouteDrift(t *testing.T) {
	root, c, ids, accepted := frozenCardFinalizedFixture(t, "route-carry-fixture")
	checkpointGlossary(t, root, "# current glossary\n", "route-carry-review")
	if _, _, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", accepted); err != nil {
		t.Fatal(err)
	}
	// A representation owns route identity inside its frozen invocation check.
	// Keep the old strict verifier intact and make the current route differ.
	v := &c.Adapted[0]
	v.Adapter = routeDriftSnapshotAdapter{v.Adapter.(frozenCardAdapter)}
	c.TranslationPackages[0].Units[0] = *v
	u, _ := c.Unit(ids[0])
	latest, err := selectLatestRetranslationUnits(root, c, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	g, _ := LoadGlossary(root, "zh-CN")
	if _, err := validateSelectedRetranslationCandidate(root, c, "zh-CN", g, u, latest.selectedByID[ids[0]]); err == nil || !strings.Contains(err.Error(), "route changed") {
		t.Fatalf("route drift error=%v", err)
	}
}

type routeDriftSnapshotAdapter struct{ frozenCardAdapter }

func (a routeDriftSnapshotAdapter) VerifyFrozenGenerationOrigin(string, string, TranslationUnitGenerationOrigin, []byte) error {
	return fmt.Errorf("frozen invocation route changed")
}

func TestSnapshotReadonlyPreparationDoesNotCreateArtifact(t *testing.T) {
	root, c, _, _ := frozenCardFinalizedFixture(t, "readonly-cards")
	before := snapshotDirectoryBytes(t, root)
	s, err := prepareQualityCheckCandidateSnapshot(root, c, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "readonly-candidate"})
	if err != nil || s.UnitCount != 3 {
		t.Fatalf("readonly selector: %+v %v", s, err)
	}
	if _, err := os.Lstat(filepath.Join(root, "data/quality-check-snapshots/zh-CN/readonly-candidate")); !os.IsNotExist(err) || !reflect.DeepEqual(before, snapshotDirectoryBytes(t, root)) {
		t.Fatal("readonly preparation created an artifact", err)
	}
}
