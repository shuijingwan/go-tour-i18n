package i18n

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The lifecycle fixture supplies actual passed source/input/candidate/validation
// evidence. Only the representation mapping is varied by these focused tests.
func TestHistoricalSurfaceMappingTimeBoundaries(t *testing.T) {
	for _, kind := range []string{"unchanged", "unchanged with successor", "unauthorized successor", "false unchanged", "explicit replacement", "explicit replacement with successor", "explicit old input", "explicit current identity", "explicit route", "source path", "source hash", "reviewed input hash", "saved input tamper", "candidate tamper", "validation tamper", "wrong successor old", "wrong successor current", "wrong successor input", "wrong successor source", "wrong successor route"} {
		t.Run(kind, func(t *testing.T) {
			root, c, _, snapshotID := frozenCardFinalizedFixture(t, "historical-mapping-fixture")
			snapshot, err := readQualityCheckSnapshot(root, "zh-CN", snapshotID, false)
			if err != nil {
				t.Fatal(err)
			}
			s := snapshot.Units[0]
			e, err := readSnapshotUnitRepositoryEvidence(root, c, "zh-CN", s)
			if err != nil {
				t.Fatal(err)
			}
			record := e.record
			p := TranslationUnitReplacement{OldUnitID: s.UnitID, CurrentUnitID: s.UnitID, Package: c.TranslationPackages[0].ID, SourcePath: s.SourcePath, SourceSHA256: s.SourceSHA256, StableIndex: 1, Route: c.Adapted[0].Route, OldInputSHA256: record.InputSHA256, CurrentInputSHA256: record.InputSHA256, ReconciliationSHA256: sum([]byte("today's mapping"))}
			u := QualityCheckSurfaceReopenUnit{Index: 1, UnitID: s.UnitID, Snapshot: s}
			var successor *TranslationUnitReplacement
			pass := kind == "unchanged" || kind == "unchanged with successor" || kind == "explicit replacement" || kind == "explicit replacement with successor"
			explicit := kind == "explicit replacement" || kind == "explicit replacement with successor" || kind == "explicit old input" || kind == "explicit current identity" || kind == "explicit route"
			if explicit {
				event := p
				event.CurrentUnitID, event.CurrentInputSHA256 = "event-representation", sum([]byte("event input"))
				u.UnitID, u.Replacement = event.CurrentUnitID, &event
				record.UnitID, record.InputSHA256 = event.CurrentUnitID, event.CurrentInputSHA256
				p.CurrentUnitID, p.CurrentInputSHA256 = event.CurrentUnitID, event.CurrentInputSHA256
			}
			later := kind == "unchanged with successor" || kind == "unauthorized successor" || kind == "explicit replacement with successor" || strings.HasPrefix(kind, "wrong successor")
			if later {
				next := p
				next.OldUnitID, next.OldInputSHA256 = u.UnitID, record.InputSHA256
				next.CurrentUnitID, next.CurrentInputSHA256 = "today-representation", sum([]byte("today input"))
				p.CurrentUnitID, p.CurrentInputSHA256 = next.CurrentUnitID, next.CurrentInputSHA256
				if kind != "unauthorized successor" {
					successor = &next // Already verified by the enclosing reader.
				}
			}
			switch kind {
			case "false unchanged":
				record.InputSHA256 = sum([]byte("different event-time input"))
			case "explicit old input":
				u.Replacement.OldInputSHA256 = sum([]byte("forged original input"))
			case "explicit current identity":
				u.Replacement.CurrentUnitID = "wrong-event-unit"
			case "explicit route":
				u.Replacement.Route = "/wrong-route"
			case "source path":
				u.Snapshot.SourcePath = "fixtures/wrong-source.xml"
			case "source hash":
				record.SourceSHA256 = sum([]byte("wrong source"))
			case "reviewed input hash":
				p.OldInputSHA256 = sum([]byte("wrong reviewed input"))
			case "saved input tamper":
				path := filepath.ToSlash(filepath.Join("data/retranslation-runs/zh-CN", s.SelectedBatchID, record.InputPath))
				writeSnapshotSource(t, root, path, []byte("tampered input\n"))
			case "candidate tamper":
				writeSnapshotSource(t, root, s.CandidatePath, []byte("tampered candidate\n"))
			case "validation tamper":
				writeSnapshotSource(t, root, s.ValidationPath, []byte("tampered validation\n"))
			case "wrong successor old":
				successor.OldUnitID = "wrong-old"
			case "wrong successor current":
				successor.CurrentUnitID = "wrong-current"
			case "wrong successor input":
				successor.OldInputSHA256 = sum([]byte("wrong successor input"))
			case "wrong successor source":
				successor.SourceSHA256 = sum([]byte("wrong successor source"))
			case "wrong successor route":
				successor.Route = "/wrong-successor-route"
			}
			resolution := &TranslationUnitReplacementResolution{HistoricalCatalog: c, Units: map[string]TranslationUnitReplacement{s.UnitID: p}}
			reopen := &QualityCheckSurfaceReopen{Units: []QualityCheckSurfaceReopenUnit{u}}
			bind := func(path, expected string) ([]byte, error) {
				b, err := readCompatibilityFile(root, path)
				if err != nil || expected != "" && sum(b) != expected {
					return nil, fmt.Errorf("historical hash mismatch: %s", path)
				}
				return b, nil
			}
			before := snapshotDirectoryBytes(t, root)
			err = verifyHistoricalSurfaceMapping(root, "zh-CN", record, reopen, resolution, successor, bind)
			if (err == nil) != pass {
				t.Fatalf("pass=%t err=%v", pass, err)
			}
			if !reflect.DeepEqual(before, snapshotDirectoryBytes(t, root)) {
				t.Fatal("historical mapping verifier mutated evidence")
			}
		})
	}
}

func TestHistoricalSurfaceMappingFrozenPredecessor(t *testing.T) {
	for _, kind := range []string{"exact", "finding tamper", "source tamper", "snapshot tamper", "false unchanged", "wrong predecessor"} {
		t.Run(kind, func(t *testing.T) {
			prior := QualityCheckSurfaceReopen{SchemaVersion: 1, EvidenceType: qualityCheckSurfaceReopenEvidenceType, Locale: "zh-CN", ReopenID: "original", Finding: "Exact independent finding", UnitCount: 1, Units: []QualityCheckSurfaceReopenUnit{{UnitID: "old", Snapshot: QualityCheckSnapshotUnit{UnitID: "old", SourcePath: "fixtures/original"}}}}
			r := prior
			r.ReopenID = "successor"
			r.ReplacementContext = &TranslationUnitReplacementContext{PredecessorReopen: GlossaryArchiveReference{Path: "data/quality-check-reopens/zh-CN/original.json"}}
			r.Units = append([]QualityCheckSurfaceReopenUnit(nil), prior.Units...)
			r.Units[0].UnitID, r.Units[0].Replacement = "event", &TranslationUnitReplacement{OldUnitID: "old", CurrentUnitID: "event"}
			switch kind {
			case "finding tamper":
				r.Finding += " altered"
			case "source tamper":
				r.Units[0].Snapshot.SourcePath = "fixtures/forged"
			case "snapshot tamper":
				r.Units[0].Snapshot.CandidateSHA256 = sum([]byte("forged candidate"))
			case "false unchanged":
				r.Units[0].Replacement = nil
			case "wrong predecessor":
				r.ReplacementContext.PredecessorReopen.Path = "data/quality-check-reopens/zh-CN/other.json"
			}
			err := verifyHistoricalReopenPredecessor("zh-CN", &r, mustJSON(prior))
			if (err == nil) != (kind == "exact") {
				t.Fatalf("kind=%s err=%v", kind, err)
			}
		})
	}
}
