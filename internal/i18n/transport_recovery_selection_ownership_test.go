package i18n

import (
	"bytes"
	"strings"
	"testing"
)

// Exact declaration shape from the supported historical selector and its
// recovery-origin successor. These are standalone producer fragments, not an
// invocation of any locale's installed archive or adoption workflow.
const recoveryHistoricalSelectorFixture = `func generationApprovedIntervalSelection(root string, c *Catalog, m *RetranslationBatchManifest, options GenerationBundleOptions) (*ApprovedIntervalSelection, error) {
	if options.ApprovedIntervalSelection != "" && len(options.approvedIntervalSelection) != 0 {
		return nil, fmt.Errorf("conflicting approved interval requests")
	}
	var s *ApprovedIntervalSelection
	var err error
	if options.ApprovedIntervalSelection != "" {
		s, err = ParseApprovedIntervalSelection(root, options.ApprovedIntervalSelection, c, m)
	}
	if len(options.approvedIntervalSelection) != 0 {
		s, err = parseApprovedIntervalSelectionBytes(root, options.approvedIntervalSelection, c, m)
	}
	if err != nil {
		return nil, err
	}
	if options.UnitID != "" {
		original, _, err := importedApprovedIntervalSelection(root, c, m)
		if err != nil {
			return nil, err
		}
		if s != nil && (original == nil || !bytes.Equal(s.bytes, original.bytes)) {
			return nil, fmt.Errorf("retry must preserve original approved interval selection")
		}
		s = original
	}
	return s, nil
}`
const recoveryCurrentSelectorFixture = `func generationApprovedIntervalSelection(root string, c *Catalog, m *RetranslationBatchManifest, options GenerationBundleOptions) (*ApprovedIntervalSelection, error) {
	if m.TransportRecovery != nil {
		if options.ApprovedIntervalSelection != "" {
			return nil, fmt.Errorf("recovery derives approved selection; external selection forbidden")
		}
		s, err := derivedRecoverySelection(root, c, m)
		if err != nil {
			return nil, err
		}
		if len(options.approvedIntervalSelection) > 0 && !bytes.Equal(options.approvedIntervalSelection, s.bytes) {
			return nil, fmt.Errorf("frozen recovery selection differs from canonical derivation")
		}
		return s, nil
	}
	if options.ApprovedIntervalSelection != "" && len(options.approvedIntervalSelection) != 0 {
		return nil, fmt.Errorf("conflicting approved interval requests")
	}
	var s *ApprovedIntervalSelection
	var err error
	if options.ApprovedIntervalSelection != "" {
		s, err = ParseApprovedIntervalSelection(root, options.ApprovedIntervalSelection, c, m)
	}
	if len(options.approvedIntervalSelection) != 0 {
		s, err = parseApprovedIntervalSelectionBytes(root, options.approvedIntervalSelection, c, m)
	}
	if err != nil {
		return nil, err
	}
	if options.UnitID != "" {
		original, _, err := importedApprovedIntervalSelection(root, c, m)
		if err != nil {
			return nil, err
		}
		if s != nil && (original == nil || !bytes.Equal(s.bytes, original.bytes)) {
			return nil, fmt.Errorf("retry must preserve original approved interval selection")
		}
		s = original
	}
	return s, nil
}`

func TestTransportRecoveryApprovedSelectionOwnership(t *testing.T) {
	const path = "internal/i18n/approved_interval_transport.go"
	source := func(decl string) []byte { return []byte("package i18n\n" + decl + "\n") }
	old, current := source(recoveryHistoricalSelectorFixture), source(recoveryCurrentSelectorFixture)
	decls, err := classifyApprovedDependencyDelta(path, old, current)
	if err != nil || len(decls) != 1 || decls[0] != "func:generationApprovedIntervalSelection" {
		t.Fatal("construction selector ownership missing", decls, err)
	}
	// A construction role follows the version-owned declaration, never a source
	// filename alone. Other changed declarations in the SAME file still block.
	historicalResponse := `func approvedResponseInputs(root string, c *Catalog, m *RetranslationBatchManifest, transports ...map[string]protectedTranslation) ([]generationBundleInput, error) {
	s, files, err := importedApprovedIntervalSelection(root, c, m)
	if err != nil {
		return nil, err
	}
	inputs, err := revisionGenerationInputsCore(root, c, m.Locale, m.Units, transports, false, s)
	if err != nil {
		return nil, err
	}
	inputs, err = applySurfaceTransportInWorld(root, c, m.Locale, m.Units, inputs, true)
	if err != nil {
		return nil, err
	}
	if err = compareInstalledApprovedProofs(inputs, files); err != nil {
		return nil, err
	}
	return inputs, nil
}`
	currentResponse := `func approvedResponseInputs(root string, c *Catalog, m *RetranslationBatchManifest, transports ...map[string]protectedTranslation) ([]generationBundleInput, error) {
	if m.TransportRecovery != nil {
		if err := verifyRecoveryInstalledExecution(root, c, m); err != nil {
			return nil, err
		}
	}
	s, files, err := importedApprovedIntervalSelection(root, c, m)
	if err != nil {
		return nil, err
	}
	if m.TransportRecovery != nil {
		expected, err := derivedRecoverySelection(root, c, m)
		if err != nil || s == nil || !bytes.Equal(s.bytes, expected.bytes) {
			return nil, fmt.Errorf("installed recovery selection differs from authority derivation: %v", err)
		}
	}
	inputs, err := revisionGenerationInputsCore(root, c, m.Locale, m.Units, transports, false, s)
	if err != nil {
		return nil, err
	}
	inputs, err = applySurfaceTransportInWorld(root, c, m.Locale, m.Units, inputs, true)
	if err != nil {
		return nil, err
	}
	if err = compareInstalledApprovedProofs(inputs, files); err != nil {
		return nil, err
	}
	if m.TransportRecovery != nil {
		inputs, err = recoveryGenerationInputs(root, c, m, inputs)
		if err != nil {
			return nil, err
		}
		for _, in := range inputs {
			if in.path == transportRecoveryConstructionMember && !bytes.Equal(in.data, files[in.path]) {
				return nil, fmt.Errorf("installed recovery construction drift")
			}
		}
	}
	return inputs, nil
}`
	cases := []struct {
		name         string
		old, current []byte
		want         string
	}{
		{"actual-adjacent-execution-and-construction", source(recoveryHistoricalSelectorFixture + "\n" + historicalResponse), source(recoveryCurrentSelectorFixture + "\n" + currentResponse), "mixed-role declaration func:approvedResponseInputs"},
		{"artifact-encoder", source(recoveryHistoricalSelectorFixture + "\nfunc encodeEnvelope() string {return \"compact\"}"), source(recoveryCurrentSelectorFixture + "\nfunc encodeEnvelope() string {return \"indented\"}"), "mixed-role declaration func:encodeEnvelope"},
		{"unknown-declaration", old, source(recoveryCurrentSelectorFixture + "\nfunc unownedControl() {}"), "mixed-role declaration func:unownedControl"},
		{"compiler-directive", old, append([]byte("//go:build linux\n\n"), current...), "mixed-role declaration compiler-directive:"},
		{"encoding-only", old, append(bytes.Clone(old), []byte("\n// no construction change\n")...), "ENCODING_ONLY_MISMATCH"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := classifyApprovedDependencyDelta(path, tc.old, tc.current); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatal("fail-closed ownership boundary weakened", err)
			}
		})
	}
	if _, err := classifyApprovedDependencyDelta("unregistered.go", old, current); err == nil {
		t.Fatal("function name granted ownership outside its producer")
	}
	if _, err := recoveryArchiveReference(map[string][]byte{"current-source": current}, GlossaryArchiveReference{path, sum(old)}); err == nil {
		t.Fatal("current bytes replaced missing declared historical material")
	}
}
