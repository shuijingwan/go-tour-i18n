package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type snapshotFixtureInverse struct{ protected protectedTranslation }

func (a snapshotFixtureInverse) ProtectedInput(*Glossary) (string, []string, error) {
	return a.protected.Text, a.protected.Tokens, nil
}
func (a snapshotFixtureInverse) Restore(s string) ([]byte, error) {
	text, failures := a.protected.restore(s)
	if len(failures) != 0 {
		return nil, fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	return []byte(text), nil
}
func (a snapshotFixtureInverse) Validate([]byte, *Glossary) error {
	return fmt.Errorf("decoder grants no candidate eligibility")
}

// This tests only the nested decoder boundary, using a synthetic sealed
// archive. Full Snapshot admission/fresh construction is separately exercised
// by TestApprovedIntervalBundleLifecycle with the real source-owned Page codec.
func checkSnapshotFrozenApprovedPredecessor(t *testing.T, root string, c *Catalog, g *Glossary, manifest *RetranslationBatchManifest, initial GenerationBundleManifest, original map[string][]byte) {
	t.Helper()
	dir := filepath.Join(root, "data/retranslation-runs", manifest.Locale, manifest.BatchID)
	m := *manifest
	m.Units = append([]RetranslationBatchUnit(nil), manifest.Units...)
	for i := range m.Units {
		m.Units[i].RevisionFeedbackSource = CurrentValidationRevisionFeedback
	}
	mb, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), mb, 0644); err != nil {
			t.Error(err)
		}
	}()
	if err := writeTranslationJSON(filepath.Join(dir, "manifest.json"), m); err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for p, b := range original {
		files[p] = b
	}
	files["formal/manifest.json"], err = os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	pkg, _, _, _, err := translationAuthorizationScope(c)
	if err != nil {
		t.Fatal(err)
	}
	r := m.Units[0]
	sel := ApprovedIntervalSelection{Schema: ApprovedIntervalSelectionSchema, Locale: m.Locale, BatchID: m.BatchID, ContentPackage: pkg, Units: []ApprovedIntervalSelectionUnit{{CurrentUnitID: r.UnitID, Mode: ApprovedIntervalMode}}}
	files[approvedIntervalSelectionMember], err = json.Marshal(sel)
	if err != nil {
		t.Fatal(err)
	}
	codePath := approvedIntervalImplementationPaths[0]
	copyBundleAuthority(t, root, []string{codePath})
	code, err := os.ReadFile(filepath.Join(root, codePath))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.WriteFile(filepath.Join(root, codePath), code, 0644); err != nil {
			t.Error(err)
		}
	}()
	proof := ApprovedIntervalTransport{Schema: ApprovedIntervalTransportSchema, Mode: ApprovedIntervalMode, Record: r, SelectionSHA256: sum(files[approvedIntervalSelectionMember]), CurrentAuthority: sum([]byte("historical fixture validator")), Dependencies: []GlossaryArchiveReference{{Path: codePath, SHA256: sum(code)}}, Projection: ApprovedIntervalProjection{UnitID: r.UnitID, SourceSHA256: r.SourceSHA256, Codec: json.RawMessage(`{"schema":"fixture-frozen-inverse/v1"}`), Children: json.RawMessage(`[]`)}}
	proof.Identity = approvedIntervalIdentity(proof)
	files[approvedIntervalProofMember(r.UnitID)] = mustJSON(proof)
	gm := initial
	gm.TaskKind = "translation-unit-revision"
	gm.Files = append([]TransportBundleFile(nil), initial.Files...)
	for _, p := range []string{approvedIntervalSelectionMember, approvedIntervalProofMember(r.UnitID)} {
		gm.Files = append(gm.Files, NewTransportBundleFile(p, "", files[p]))
	}
	sort.Slice(gm.Files, func(i, j int) bool { return gm.Files[i].BundlePath < gm.Files[j].BundlePath })
	var identity strings.Builder
	entries := []TransportBundleEntry{}
	for i, f := range gm.Files {
		gm.Files[i] = NewTransportBundleFile(f.BundlePath, f.RepositoryPath, files[f.BundlePath])
		f = gm.Files[i]
		identity.WriteString(f.BundlePath + "\x00" + f.SHA256 + "\x00")
		entries = append(entries, TransportBundleEntry{Path: f.BundlePath, Data: files[f.BundlePath]})
	}
	gm.InputIdentitySHA256 = sum([]byte(identity.String()))
	gb, err := json.Marshal(gm)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := writeTranslationUnitGenerationBundle(gb, entries)
	if err != nil {
		t.Fatal(err)
	}
	ip := filepath.Join(dir, "generation-imports/initial-manifest.json")
	ib, err := os.ReadFile(ip)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.WriteFile(ip, ib, 0644); err != nil {
			t.Error(err)
		}
	}()
	var receipt GenerationResultBundleManifest
	if err := json.Unmarshal(ib, &receipt); err != nil {
		t.Fatal(err)
	}
	receipt.TaskKind = gm.TaskKind
	receipt.GenerationBundleSHA256 = sum(archive)
	receipt.InputIdentitySHA256 = gm.InputIdentitySHA256
	receipt.GenerationManifest = NewTransportBundleFile("generation-manifest.json", "", gb)
	if err := writeTranslationJSON(ip, receipt); err != nil {
		t.Fatal(err)
	}
	ap := filepath.Join(root, importedGenerationArchive(m.Locale, sum(archive)))
	if err := os.WriteFile(ap, archive, 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(ap)
	if err := os.WriteFile(filepath.Join(root, codePath), append(bytes.Clone(code), []byte("\n// fixture code-owned drift\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	current := *c
	type codec struct {
		Schema string `json:"schema"`
	}
	type child struct {
		Role string `json:"role"`
	}
	u, err := c.Unit(r.UnitID)
	if err != nil {
		t.Fatal(err)
	}
	ordinary, err := prepareTranslationUnitInput(u, g)
	if err != nil {
		t.Fatal(err)
	}
	current.DerivedTransportCodecReader = func(_ TranslationUnitAdapter, b, inventory []byte) (any, any, TranslationUnitAdapter, error) {
		var p codec
		var children []child
		if err := json.Unmarshal(b, &p); err != nil {
			return nil, nil, nil, err
		}
		if err := json.Unmarshal(inventory, &children); err != nil {
			return nil, nil, nil, err
		}
		if p.Schema != "fixture-frozen-inverse/v1" {
			return nil, nil, nil, fmt.Errorf("unknown fixture inverse")
		}
		return p, children, snapshotFixtureInverse{ordinary}, nil
	}
	observed := map[string]string{}
	outputs, check, err := snapshotFrozenPredecessorOutputs(root, dir, &current, g, &m, observed)
	if err != nil {
		t.Fatal("nested historical decoder must not require ordinary current-proof equality", err)
	}
	result, err := readPromotionResult(dir, m.Locale, m.BatchID, len(m.Units))
	if err != nil {
		t.Fatal(err)
	}
	v, err := readPromotionValidation(dir, m.BatchID, m.Locale, r, result.Units[0])
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, v.RawResponsePath))
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := os.ReadFile(filepath.Join(dir, result.Units[0].CandidatePath))
	if err != nil {
		t.Fatal(err)
	}
	restored, failures := outputs[r.UnitID].restore(string(raw))
	if len(failures) != 0 || !bytes.Equal(canonicalizeRetranslationArtifactEOF([]byte(restored)), candidate) {
		t.Fatal("frozen inverse failed exact candidate restore", failures)
	}
	if err := check(); err != nil {
		t.Fatal(err)
	}
	if len(observed) == 0 {
		t.Fatal("ancestor observations were not bound into Snapshot commit recheck")
	}
	bad := append(bytes.Clone(raw), 'x')
	if err := os.WriteFile(filepath.Join(dir, v.RawResponsePath), bad, 0644); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.WriteFile(filepath.Join(dir, v.RawResponsePath), raw, 0644); err != nil {
			t.Error(err)
		}
	}()
	if err := check(); err == nil {
		t.Fatal("frozen predecessor mutation passed commit recheck")
	}
	if _, _, err := snapshotFrozenPredecessorOutputs(root, dir, &current, g, &m, map[string]string{}); err == nil {
		t.Fatal("tampered imported ancestor output accepted")
	}
}
