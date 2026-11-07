package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestApprovedIntervalSelectionContract(t *testing.T) {
	root, c, batch := generationBundleFixture(t, "zh-CN", RetranslationGeneratorChatGPT)
	m, err := readRetranslationProcessManifest(root+"/data/retranslation-runs/zh-CN/"+batch, "zh-CN", batch)
	if err != nil {
		t.Fatal(err)
	}
	for i := range m.Units {
		m.Units[i].RevisionFeedbackSource = CurrentValidationRevisionFeedback
	}
	pkg, _, _, _, err := translationAuthorizationScope(c)
	if err != nil {
		t.Fatal(err)
	}
	s := ApprovedIntervalSelection{Schema: ApprovedIntervalSelectionSchema, Locale: m.Locale, BatchID: m.BatchID, ContentPackage: pkg, Units: []ApprovedIntervalSelectionUnit{{CurrentUnitID: m.Units[0].UnitID, Mode: ApprovedIntervalMode}}}
	encoded, _ := json.Marshal(s)
	if _, err := parseApprovedIntervalSelectionBytes(root, encoded, c, m); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*ApprovedIntervalSelection){
		"schema":    func(s *ApprovedIntervalSelection) { s.Schema += "unknown" },
		"mode":      func(s *ApprovedIntervalSelection) { s.Units[0].Mode = "ordinary" },
		"unit":      func(s *ApprovedIntervalSelection) { s.Units[0].CurrentUnitID = "unknown" },
		"duplicate": func(s *ApprovedIntervalSelection) { s.Units = append(s.Units, s.Units[0]) },
		"reverse-order": func(s *ApprovedIntervalSelection) {
			s.Units = append([]ApprovedIntervalSelectionUnit{{CurrentUnitID: m.Units[1].UnitID, Mode: ApprovedIntervalMode}}, s.Units...)
		},
		"empty": func(s *ApprovedIntervalSelection) { s.Units = nil },
	} {
		t.Run(name, func(t *testing.T) {
			var bad ApprovedIntervalSelection
			json.Unmarshal(encoded, &bad)
			mutate(&bad)
			b, _ := json.Marshal(bad)
			if _, err := parseApprovedIntervalSelectionBytes(root, b, c, m); err == nil {
				t.Fatal("invalid selection accepted")
			}
		})
	}
	if _, err := parseApprovedIntervalSelectionBytes(root, append([]byte(`{"target_mapping":{},`), encoded[1:]...), c, m); err == nil {
		t.Fatal("target mapping accepted")
	}
	if err := os.WriteFile(filepath.Join(root, "selection.json"), encoded, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseApprovedIntervalSelection(root, "selection.json", c, m); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(root, "selection.json"), "../selection.json", `requests\selection.json`} {
		if _, err := ParseApprovedIntervalSelection(root, path, c, m); err == nil {
			t.Fatal("unsafe selector path accepted", path)
		}
	}
	if err := os.Symlink(filepath.Join(root, "selection.json"), filepath.Join(root, "linked.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseApprovedIntervalSelection(root, "linked.json", c, m); err == nil {
		t.Fatal("symlink selector accepted")
	}
}
