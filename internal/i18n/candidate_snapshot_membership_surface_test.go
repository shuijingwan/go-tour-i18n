package i18n

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSnapshotMembershipSurfaceUnitIDJoin(t *testing.T) {
	root, c, ids, snapshot := checkpointFixture(t)
	_, options := aggregateSurfaceAuthorityFixture(t, root, c, snapshot)
	reopen, reopenPath, err := RecordQualityCheckSurfaceReopen(root, c, options)
	if err != nil {
		t.Fatal(err)
	}
	if len(reopen.Units) < 3 {
		t.Fatal("three frozen Surface Units required")
	}
	exported, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: options.Locale, Generator: RetranslationGeneratorChatGPT, UnitKind: UnitKindPage, UnitIDs: ids, PreviousSnapshotID: snapshot, SurfaceReopenID: options.ReopenID, AllowReexport: true})
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "data/retranslation-runs", options.Locale, exported.BatchID)
	m, err := readRetranslationProcessManifest(dir, options.Locale, exported.BatchID)
	if err != nil {
		t.Fatal(err)
	}
	// Freeze a different, legal manifest order before export/import/process.
	// Neither the Surface receipt nor its per-Unit findings are reordered.
	m.Units = append(append([]RetranslationBatchUnit{}, m.Units[1:]...), m.Units[0])
	if m.Units[0].UnitID == reopen.Units[0].UnitID {
		t.Fatal("fixture order did not differ")
	}
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := writeTranslationJSON(manifestPath, m); err != nil {
		t.Fatal(err)
	}
	// Reuse the existing ordinary Surface process path; this fixture needs no
	// optional Surface decoder capability or new imported transport authority.
	if err := os.MkdirAll(filepath.Join(dir, "raw-responses"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, record := range m.Units {
		input, err := os.ReadFile(filepath.Join(dir, record.InputPath))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "raw-responses", filepath.Base(record.InputPath)), input, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if result, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: options.Locale, BatchID: exported.BatchID}); err != nil || result.ValidationPassed != len(ids) {
		t.Fatal("processed Surface fixture", result, err)
	}
	latest, err := selectLatestRetranslationUnits(root, c, options.Locale)
	if err != nil {
		t.Fatal(err)
	}
	g, err := LoadGlossary(root, options.Locale)
	if err != nil {
		t.Fatal(err)
	}
	current := *c
	members := map[string]*Catalog{}
	for _, id := range ids {
		members[id] = c
	}
	prepared := PrepareGlossaryCompatibility(root, options.Locale, c)
	validate := func(id string) error {
		u, err := c.Unit(id)
		if err != nil {
			return err
		}
		_, err = validateSnapshotMembershipCandidate(root, &current, options.Locale, g, u, latest.selectedByID[id], nil, members, nil, prepared)
		return err
	}
	t.Run("exact-set-different-order", func(t *testing.T) {
		for _, id := range ids {
			if err := validate(id); err != nil {
				t.Fatal("identity-aligned Surface membership must pass", err)
			}
		}
	})
	for _, mode := range []string{"missing-extra", "duplicate-manifest", "count", "source-path", "source-sha", "finding", "authority", "previous-snapshot", "review-sha", "review-path", "rating", "kind", "feedback"} {
		t.Run(mode, func(t *testing.T) {
			changed := *m
			changed.Units = append([]RetranslationBatchUnit{}, m.Units...)
			switch mode {
			case "missing-extra":
				changed.Units[1].UnitID = "outside-reopen"
			case "duplicate-manifest":
				changed.Units[1] = changed.Units[0]
			case "count":
				changed.Units = changed.Units[:len(changed.Units)-1]
				changed.UnitCount--
			case "source-path":
				changed.Units[1].SourcePath = "different-source.xml"
			case "source-sha":
				changed.Units[1].SourceSHA256 = sum([]byte("different source"))
			case "finding":
				changed.Units[1].PreviousFinding = "unrelated finding"
			case "authority":
				changed.Units[1].RevisionAuthorizationID = "different-authority"
			case "previous-snapshot":
				changed.Units[1].PreviousSnapshotID = "different-snapshot"
			case "review-sha":
				changed.Units[1].PreviousReviewSHA256 = sum([]byte("different review"))
			case "review-path":
				changed.Units[1].PreviousReviewPath = "data/wrong-review.json"
			case "rating":
				changed.Units[1].PreviousRating = "B"
			case "kind":
				changed.Units[1].UnitKind = UnitKindExample
			case "feedback":
				changed.Units[1].RevisionFeedbackSource = "quality_check"
			}
			if err := writeTranslationJSON(manifestPath, changed); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := writeTranslationJSON(manifestPath, m); err != nil {
					t.Fatal(err)
				}
			}()
			if err := validate(ids[0]); err == nil {
				t.Fatal("invalid Surface lineage accepted", mode)
			} else if (mode == "source-path" || mode == "source-sha" || mode == "finding" || mode == "authority" || mode == "review-sha") && !strings.Contains(err.Error(), "Snapshot membership Surface revision frozen lineage mismatch") {
				t.Fatal("expected exact lineage gate", err)
			}
		})
	}
	t.Run("duplicate-reopen", func(t *testing.T) {
		changed := *reopen
		changed.Units = append([]QualityCheckSurfaceReopenUnit{}, reopen.Units...)
		changed.Units[1] = changed.Units[0]
		if err := writeTranslationJSON(filepath.Join(root, reopenPath), changed); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := writeTranslationJSON(filepath.Join(root, reopenPath), reopen); err != nil {
				t.Fatal(err)
			}
		}()
		if err := validate(ids[0]); err == nil {
			t.Fatal("duplicate frozen Surface Unit accepted")
		}
	})
	// The manifest ordering is process provenance and stays unchanged by join.
	after, err := readRetranslationProcessManifest(dir, options.Locale, exported.BatchID)
	if err != nil || !reflect.DeepEqual(after, m) {
		t.Fatal("membership verifier rewrote manifest order", err)
	}
}
