package i18n

import (
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// A successor reopen copies its predecessor verbatim except for its explicit
// ID/context and per-Unit mapping. Today's resolver cannot redefine that event.
func verifyHistoricalReopenPredecessor(locale string, reopen *QualityCheckSurfaceReopen, data []byte) error {
	var prior QualityCheckSurfaceReopen
	if err := contentidentity.StrictJSON(data, &prior); err != nil {
		return err
	}
	if prior.Locale != locale || prior.ReplacementContext != nil || prior.ReopenID == reopen.ReopenID || reopen.ReplacementContext.PredecessorReopen.Path != "data/quality-check-reopens/"+locale+"/"+prior.ReopenID+".json" || len(prior.Units) != len(reopen.Units) {
		return fmt.Errorf("historical replacement predecessor identity mismatch")
	}
	want := prior
	want.ReopenID, want.ReplacementContext = reopen.ReopenID, reopen.ReplacementContext
	want.Units = append([]QualityCheckSurfaceReopenUnit(nil), prior.Units...)
	changed := 0
	for i, u := range reopen.Units {
		want.Units[i].UnitID, want.Units[i].Replacement = u.UnitID, u.Replacement
		if u.Replacement != nil {
			changed++
		}
	}
	if changed == 0 || !reflect.DeepEqual(&want, reopen) {
		return fmt.Errorf("historical replacement event differs from frozen predecessor")
	}
	return nil
}

// HistoricalCatalog is the frozen old side. resolution.Units is old -> today,
// not old -> event time. Check the recorded event against its frozen source and
// saved formal input; any later edge must already have passed machine successor
// verification in the enclosing finalization reader. No mapping is guessed.
func verifyHistoricalSurfaceMapping(root, locale string, record RetranslationBatchUnit, reopen *QualityCheckSurfaceReopen, resolution *TranslationUnitReplacementResolution, currentSuccessor *TranslationUnitReplacement, bind func(string, string) ([]byte, error)) error {
	for _, u := range reopen.Units {
		if u.UnitID != record.UnitID {
			continue
		}
		p, ok := resolution.Units[u.Snapshot.UnitID]
		if !ok || p.OldUnitID != u.Snapshot.UnitID || p.SourcePath != record.SourcePath || p.SourceSHA256 != record.SourceSHA256 || u.Snapshot.SourcePath != record.SourcePath || u.Snapshot.SourceSHA256 != record.SourceSHA256 || record.UnitKind != UnitKindPage || u.Snapshot.UnitKind != UnitKindPage || u.Index != p.StableIndex {
			return fmt.Errorf("historical mapping source/descriptor mismatch: %s", u.UnitID)
		}
		e, err := readSnapshotUnitRepositoryEvidence(root, resolution.HistoricalCatalog, locale, u.Snapshot)
		if err != nil || e.record.InputSHA256 != p.OldInputSHA256 {
			return fmt.Errorf("historical mapping reviewed input mismatch: %s: %v", u.UnitID, err)
		}
		base := filepath.ToSlash(filepath.Join("data/retranslation-runs", locale, u.Snapshot.SelectedBatchID))
		for _, ref := range []GlossaryArchiveReference{
			{u.Snapshot.CandidatePath, u.Snapshot.CandidateSHA256},
			{u.Snapshot.ValidationPath, u.Snapshot.ValidationSHA256},
			{base + "/" + e.record.InputPath, p.OldInputSHA256},
			{base + "/manifest.json", ""}, {base + "/result.json", ""},
		} {
			if _, err := bind(ref.Path, ref.SHA256); err != nil {
				return err
			}
		}
		eventInput := p.OldInputSHA256
		if old := u.Replacement; old == nil {
			if u.UnitID != p.OldUnitID || record.InputSHA256 != eventInput {
				return fmt.Errorf("historical unchanged mapping changed: %s", u.UnitID)
			}
		} else {
			if old.OldUnitID != p.OldUnitID || old.CurrentUnitID != u.UnitID || old.OldUnitID == old.CurrentUnitID || old.SourcePath != p.SourcePath || old.SourceSHA256 != p.SourceSHA256 || old.OldInputSHA256 != p.OldInputSHA256 || old.CurrentInputSHA256 != record.InputSHA256 || old.OldInputSHA256 == old.CurrentInputSHA256 || old.Package != p.Package || old.StableIndex != p.StableIndex || old.Route != p.Route || !validSHA256(old.ReconciliationSHA256) {
				return fmt.Errorf("historical replacement mapping changed: %s", u.UnitID)
			}
			eventInput = old.CurrentInputSHA256
		}
		if p.CurrentUnitID != u.UnitID || p.CurrentInputSHA256 != eventInput {
			s := currentSuccessor
			if s == nil || s.OldUnitID != u.UnitID || s.CurrentUnitID != p.CurrentUnitID || s.OldInputSHA256 != eventInput || s.CurrentInputSHA256 != p.CurrentInputSHA256 || s.SourcePath != p.SourcePath || s.SourceSHA256 != p.SourceSHA256 || s.Package != p.Package || s.StableIndex != p.StableIndex || s.Route != p.Route || !validSHA256(s.ReconciliationSHA256) {
				return fmt.Errorf("historical mapping lacks verified current successor: %s", u.UnitID)
			}
		}
		return nil
	}
	return fmt.Errorf("historical mapping missing exact revision Unit: %s", record.UnitID)
}
