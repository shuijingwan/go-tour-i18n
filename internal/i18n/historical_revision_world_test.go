package i18n

import (
	"reflect"
	"testing"
)

func TestHistoricalRevisionWorldIsSnapshotAuthority(t *testing.T) {
	root, c, _, plan, _, _ := authorizationMachineFixture(t)
	u := plan.Receipt.Units[0]
	r := &QualityCheckSurfaceReopen{SchemaVersion: qualityCheckSurfaceReopenSchemaVersion, EvidenceType: qualityCheckSurfaceReopenEvidenceType, AuthoritySource: RepresentationMachineAuthority, ReopenID: "historical-world-authority", UnitCount: 1, Locale: "zh-CN", PreviousSnapshotID: plan.Receipt.Options.PreviousSnapshotID, ReplacementContext: &TranslationUnitReplacementContext{HistoricalDocuments: plan.Receipt.Options.HistoricalDocuments, Machine: &RepresentationRevisionAuthority{Options: plan.Receipt.Options, UnitID: u.Replacement.CurrentUnitID, Predecessor: u.Predecessor}}, Units: []QualityCheckSurfaceReopenUnit{{UnitID: u.Replacement.CurrentUnitID, Replacement: &u.Replacement, Snapshot: u.Predecessor}}}
	world, err := historicalRepresentationRevisionWorld(root, c, r)
	if err != nil || world.snapshot != plan.Receipt.Snapshot || world.glossary.SHA256 != plan.Receipt.GlossarySHA256 {
		t.Fatal("world is not bound to revision Snapshot authority", world, err)
	}
	// An internal QC resolver is not an input to this factory. Mutating an
	// unrelated resolver cannot choose the revision's glossary or authority.
	qc := PrepareGlossaryCompatibility(root, "zh-CN", c)
	qc.historicalSHA = sum([]byte("unrelated QC world"))
	qc.historicalEvidenceRefs = map[string]string{"unrelated": sum(nil)}
	checkpointGlossary(t, root, "\n# newer current glossary\n", "revision-world-new-current")
	again, err := historicalRepresentationRevisionWorld(root, c, r)
	if err != nil || !reflect.DeepEqual(world, again) || again.glossary.SHA256 == qc.historicalSHA || world == again {
		t.Fatal("revision authority depended on an incidental QC/current world", again, err)
	}
	copy := *r
	copy.Units = append([]QualityCheckSurfaceReopenUnit(nil), r.Units...)
	bad := *copy.Units[0].Replacement
	bad.OldUnitID = "wrong-old-id"
	copy.Units[0].Replacement = &bad
	if _, err := historicalRepresentationRevisionWorld(root, c, &copy); err == nil {
		t.Fatal("wrong remap authorized a historical world")
	}
	data, err := readCompatibilityFile(root, world.glossary.Path)
	if err != nil {
		t.Fatal(err)
	}
	writeSnapshotSource(t, root, world.glossary.Path, append(data, ' '))
	if _, err := historicalRepresentationRevisionWorld(root, c, r); err == nil {
		t.Fatal("changed glossary archive authorized a historical world")
	}
}
