package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestMembershipSnapshotSuccessor49Plus2(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	f, options, historical, batchID := membershipRevisionSnapshotFixture(t)
	root, c := f.root, f.current
	s, err := i18n.PrepareQualityCheckCandidateSnapshot(root, c, options)
	if err != nil {
		t.Fatal("49+2 current Snapshot preflight", err)
	}
	if s.UnitCount != 51 || s.PageCount != 51 || s.ExampleCount != 0 || s.ContentPackage == nil || !reflect.DeepEqual(s.ContentPackage.UnitIDs, f.ids) || s.CurrentReattestation != nil {
		t.Fatal("current full scope or language judgment changed", s)
	}
	for i, u := range s.Units {
		if u.Index != i+1 || u.UnitID != f.ids[i] {
			t.Fatal("ordered membership changed", i)
		}
		if i < 5 {
			if u.SelectedBatchID != batchID {
				t.Fatal("revision successor not selected")
			}
		} else {
			pi, index := 0, i
			if i >= 49 {
				pi, index = 1, i-49
			}
			if u.CandidatePath != f.finals[pi].Units[index].Snapshot.CandidatePath || u.CandidateSHA256 != f.finals[pi].Units[index].Snapshot.CandidateSHA256 {
				t.Fatal("unaffected predecessor changed", i)
			}
		}
	}
	bound := false
	for _, ref := range s.ContentPackage.AuthorityReferences {
		bound = bound || ref.Path == f.membershipRef.Path
	}
	if !bound {
		t.Fatal("Snapshot did not bind formal membership authority")
	}
	if _, err := os.Stat(filepath.Join(root, "data/quality-check-snapshots/zh-CN", options.SnapshotID)); !os.IsNotExist(err) {
		t.Fatal("preflight published Snapshot", err)
	}
	for _, name := range []string{"missing-membership", "tampered-membership", "wrong-predecessor-identity", "predecessor-order", "missing-predecessor-unit", "missing-sibling-unit", "current-membership-changed", "source-drift", "candidate-drift", "validation-attempt", "unauthorized-extra"} {
		t.Run(name, func(t *testing.T) {
			local := *c
			local.TranslationScope = append([]string(nil), c.TranslationScope...)
			local.HistoricalTranslationPackages = append([]i18n.TranslationPackageDescriptor(nil), c.HistoricalTranslationPackages...)
			for i := range local.HistoricalTranslationPackages {
				local.HistoricalTranslationPackages[i].Units = append([]i18n.AdaptedTranslationUnit(nil), local.HistoricalTranslationPackages[i].Units...)
			}
			path := ""
			var before []byte
			switch name {
			case "missing-membership", "tampered-membership":
				path = f.membershipRef.Path
			case "source-drift":
				path = s.Units[7].SourcePath
			case "candidate-drift":
				path = s.Units[0].CandidatePath
			case "validation-attempt":
				path = s.Units[0].ValidationPath
			case "wrong-predecessor-identity":
				local.HistoricalTranslationPackages[0].AuthoritySHA256 = digest([]byte("wrong authority"))
			case "predecessor-order":
				p := &local.HistoricalTranslationPackages[0]
				p.Units[0], p.Units[1] = p.Units[1], p.Units[0]
			case "missing-predecessor-unit":
				local.HistoricalTranslationPackages[0].Units = local.HistoricalTranslationPackages[0].Units[:48]
			case "missing-sibling-unit":
				local.HistoricalTranslationPackages[1].Units = local.HistoricalTranslationPackages[1].Units[:1]
			case "current-membership-changed":
				local.TranslationScope[1], local.TranslationScope[2] = local.TranslationScope[2], local.TranslationScope[1]
			case "unauthorized-extra":
				local.TranslationScope = append(local.TranslationScope, "not-authorized")
			}
			if path != "" {
				var err error
				before, err = readRegular(root, path)
				if err != nil {
					t.Fatal(err)
				}
				defer writeFixture(t, root, path, before)
				if name == "missing-membership" {
					if err := os.Remove(filepath.Join(root, path)); err != nil {
						t.Fatal(err)
					}
				} else if name == "validation-attempt" {
					var v i18n.RetranslationValidation
					if json.Unmarshal(before, &v) != nil {
						t.Fatal("validation fixture")
					}
					v.Attempt++
					bad, _ := json.Marshal(v)
					writeFixture(t, root, path, bad)
				} else {
					writeFixture(t, root, path, append(bytes.Clone(before), ' '))
				}
			}
			if _, err := i18n.PrepareQualityCheckCandidateSnapshot(root, &local, options); err == nil {
				t.Fatal("invalid membership/candidate accepted", name)
			}
		})
	}
	for path, before := range historical {
		after, err := readRegular(root, path)
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal("historical Snapshot mutated", path, err)
		}
	}
}

func membershipRevisionSnapshotFixture(t *testing.T, revisionIndices ...int) (surfaceComposedAuthorityFixture, i18n.QualityCheckSnapshotOptions, map[string][]byte, string) {
	t.Helper()
	f := newSurfaceComposedAuthorityFixture(t, true)
	root, c := f.root, f.current
	historical := map[string][]byte{}
	for _, final := range f.finals {
		path := "data/quality-check-snapshots/zh-CN/" + final.SnapshotID + "/manifest.json"
		b, err := readRegular(root, path)
		if err != nil {
			t.Fatal(err)
		}
		historical[path] = b
	}
	if len(revisionIndices) == 0 {
		revisionIndices = []int{0, 1, 2, 3, 4}
	}
	selected := []i18n.QualityCheckFinalizationUnit{}
	selectedIDs := []string{}
	for _, i := range revisionIndices {
		selected = append(selected, f.finals[0].Units[i])
		selectedIDs = append(selectedIDs, f.ids[i])
	}
	findings := []SurfaceFinding{}
	texts := []string{}
	for _, x := range selected {
		text := "Correct exact successor prose: " + x.UnitID
		texts = append(texts, text)
		findings = append(findings, SurfaceFinding{ID: x.Snapshot.SourcePath, Finding: text, Class: i18n.SurfaceTranslationUnitLanguage, Artifacts: []i18n.SurfaceCandidateArtifact{{SnapshotID: f.finals[0].SnapshotID, Unit: x.Snapshot}}})
	}
	r, _ := surfaceReceipt(t, root, c, f.build, []Reference{f.closureRef}, "membership-snapshot-failed", "failed", findings)
	o := i18n.QualityCheckSurfaceReopenOptions{Locale: "zh-CN", ReopenID: "membership-snapshot-reopen", PreviousSnapshotID: f.finals[0].SnapshotID, SurfaceReviewID: r.ReviewID, Findings: texts, UnitIDs: selectedIDs, IntegratedSurface: true}
	if _, _, err := i18n.RecordQualityCheckSurfaceReopen(root, c, o); err != nil {
		t.Fatal(err)
	}
	e, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, BatchID: "chatgpt-zh-CN-060", UnitIDs: o.UnitIDs, UnitKind: i18n.UnitKindPage, PreviousSnapshotID: o.PreviousSnapshotID, SurfaceReopenID: o.ReopenID, AllowReexport: true})
	if err != nil {
		t.Fatal(err)
	}
	// Materialize the already-passed revision state this Snapshot consumer
	// receives. This is the legacy ordinary Surface transport fixture; sealed
	// bounded-transport import/process is covered by its dedicated regression.
	for _, old := range selected {
		var v i18n.RetranslationValidation
		b, err := readRegular(root, old.Snapshot.ValidationPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatal(err)
		}
		base := filepath.ToSlash(filepath.Dir(filepath.Dir(old.Snapshot.ValidationPath)))
		raw, err := readRegular(root, base+"/"+v.RawResponsePath)
		if err != nil {
			t.Fatal(err)
		}
		revised := bytes.Replace(raw, []byte("Read this message."), []byte("阅读修订后的消息。"), 1)
		if bytes.Equal(raw, revised) {
			t.Fatal("fixture prose not editable")
		}
		writeFixture(t, root, e.BatchPath+"/raw-responses/"+filepath.Base(v.RawResponsePath), revised)
	}
	processed, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID})
	if err != nil || processed.ValidationPassed != 5 {
		t.Fatal("fixture passed revision", processed, err)
	}
	options := i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "membership-successor", ContentPackage: c.TranslationPackages[0].ID}
	return f, options, historical, e.BatchID
}
