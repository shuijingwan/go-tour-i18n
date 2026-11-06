package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// A historical validator epoch, not a different Page representation. Only the
// comment completeness policy changes when enabled; source/input/UnitID do not.
type transportEpochPage struct {
	PageAdapter
	enabled bool
}

func (a transportEpochPage) ReattestationAuthority(root string) (string, error) {
	identity, err := a.PageAdapter.ReattestationAuthority(root)
	return digest([]byte(fmt.Sprintf("%s/epoch/%t", identity, a.enabled))), err
}
func (a transportEpochPage) CommentSourceBlocks() ([]i18n.CommentCodeBlock, string, error) {
	if !a.enabled {
		return nil, "", nil
	}
	return a.PageAdapter.CommentSourceBlocks()
}
func (a transportEpochPage) CommentTargetBlocks(b []byte) ([]i18n.CommentCodeBlock, error) {
	if !a.enabled {
		return nil, nil
	}
	return a.PageAdapter.CommentTargetBlocks(b)
}

func TestCommentRevisionTransportLifecycle(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	finish := contentidentity.BeginMemoization()
	defer finish()
	const code = "# Comment transport\n\nKeep unrelated prose.\n\n```go\npackage p\nconst message = \"literal\"\nvar name string\n// Associate message with name.\n// Capture connection properties.\n//go:generate echo keep\nfunc f() string { return message }\n```\n"
	root, _, c, _ := replacementPageFixtureSize(t, 1, 1, func(int, []byte) []byte {
		return []byte(code + "\n<pre>\n{\n  // The module path.\n  \"path\": \"example.com/module\"\n}\n</pre>\n")
	})
	for _, path := range []string{"docs/CURRENT_TRANSLATION_AUTHORIZATION.md", "docs/CODE_COMMENT_INTEGRITY.md"} {
		b, err := os.ReadFile(filepath.Join(repositoryRoot(t), path))
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, path, b)
	}
	binding := c.Adapted[0]
	a := transportEpochPage{PageAdapter: binding.Adapter.(PageAdapter)}
	binding.Adapter = a
	c.Adapted[0], c.TranslationPackages[0].Units[0] = binding, binding
	ids := append([]string(nil), c.TranslationScope...)
	initial, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: ids})
	if err != nil {
		t.Fatal(err)
	}
	bundle, m, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: initial.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	files, err := i18n.ReadTransportBundle(bundle, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(m.ExpectedOutputs[0].BundlePath)
	initialRaw := bytes.Replace(files["formal/inputs/"+name], []byte("Capture connection properties."), []byte("保留另一条说明。"), 1)
	if bytes.Equal(initialRaw, files["formal/inputs/"+name]) {
		t.Fatal("fixture secondary teaching comment not editable")
	}
	if files["formal/comment-revision-transport/"+name+".json"] != nil {
		t.Fatal("initial Generation acquired revision projection")
	}
	surfaceImport(t, root, c, bundle, map[string][]byte{ids[0]: initialRaw})
	processed, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: initial.BatchID})
	if err != nil || processed.ValidationPassed != 1 {
		t.Fatal(processed, err)
	}
	f := surfaceFinalize(t, root, c, "comment-predecessor", "", ids)
	if _, _, err := i18n.EstablishCurrentTranslationAuthorization(root, c, "zh-CN", f.SnapshotID); err != nil {
		t.Fatal(err)
	}
	finalPath := "data/quality-check-snapshots/zh-CN/" + f.SnapshotID + "/finalization.json"
	finalBefore, err := os.ReadFile(filepath.Join(root, finalPath))
	if err != nil {
		t.Fatal(err)
	}
	a.enabled = true
	binding.Adapter = a
	c.Adapted[0], c.TranslationPackages[0].Units[0] = binding, binding
	assessment, err := i18n.AssessCurrentTranslation(root, c, "zh-CN", f.SnapshotID)
	if err != nil || assessment.Affected != 1 || assessment.Compatible+assessment.Stale+assessment.Unknown != 0 {
		t.Fatal("epoch impact", assessment, err)
	}
	assessmentBytes, _ := json.MarshalIndent(assessment, "", "  ")
	assessmentRef := i18n.GlossaryArchiveReference{Path: "engineering/comment-reattest.json", SHA256: digest(assessmentBytes)}
	writeFixture(t, root, assessmentRef.Path, assessmentBytes)
	manifestBytes, err := os.ReadFile(filepath.Join(root, "data/quality-check-snapshots/zh-CN/"+f.SnapshotID+"/manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	o := i18n.CurrentValidationReopenOptions{ContentPackage: c.TranslationPackages[0].ID, Locale: "zh-CN", ReopenID: "comment-revision", PreviousSnapshotID: f.SnapshotID, SnapshotSHA256: digest(manifestBytes), Evidence: assessmentRef, AssessmentIdentity: assessment.Identity, UnitIDs: ids}
	r, _, err := i18n.RecordQualityCheckSurfaceReopen(root, c, i18n.QualityCheckSurfaceReopenOptions{AuthoritySource: i18n.CurrentValidationAuthority, Locale: o.Locale, ReopenID: o.ReopenID, PreviousSnapshotID: o.PreviousSnapshotID, CurrentValidation: &o})
	if err != nil || len(r.CurrentValidation.Findings) != 1 || len(r.CurrentValidation.Findings[0].CommentSlots) != 2 {
		t.Fatal("exact authorized carry", err)
	}
	e, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: o.Locale, Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: ids, UnitKind: i18n.UnitKindPage, AllowReexport: true, PreviousSnapshotID: o.PreviousSnapshotID, SurfaceReopenID: o.ReopenID})
	if err != nil {
		t.Fatal(err)
	}
	revision, rm, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: o.Locale, BatchID: e.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	files, err = i18n.ReadTransportBundle(revision, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	var p i18n.CommentRevisionTransport
	if err := json.Unmarshal(files["formal/comment-revision-transport/"+name+".json"], &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Slots) != 2 || p.Slots[0].SlotID != r.CurrentValidation.Findings[0].CommentSlots[0].SlotID || len(p.Slots[0].Tokens) != 2 {
		t.Fatal("wrong editable scope/bindings", p.Slots)
	}
	raw := string(files["formal/protected-candidates/"+name])
	if !bytes.Equal(files["formal/historical-protected-candidates/"+name], initialRaw) {
		t.Fatal("historical passed raw lost")
	}
	for _, locked := range []string{"const message", "\"literal\"", "//go:generate", "保留另一条说明", "Keep unrelated prose", "// Associate"} {
		if strings.Contains(raw, locked) {
			t.Fatal("unauthorized editable content", locked)
		}
	}
	translated := "将" + p.Slots[0].Tokens[1] + "与" + p.Slots[0].Tokens[0] + "关联。"
	good := []byte(strings.Replace(strings.Replace(raw, p.Slots[0].Text, translated, 1), p.Slots[1].Text, "模块路径。", 1))
	dir := t.TempDir()
	write := func(b []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(good)
	if _, _, err := i18n.PackGenerationResultBundle(root, c, revision, "chatgpt", i18n.FormalChatGPTGenerationModel, dir); err != nil {
		t.Fatal("legal projected output", err)
	}
	badOutputs := map[string]string{
		"delete-token":    strings.Replace(string(good), p.Boundaries[0], "", 1),
		"duplicate-token": p.Boundaries[0] + string(good),
		"mutate-token":    strings.Replace(string(good), p.Boundaries[0], "⟪GTI18N_00000000_999999⟫", 1),
		"identifier":      strings.Replace(string(good), p.Slots[0].Tokens[0], "newName", 1),
		"delimiter":       strings.Replace(string(good), translated, translated+" // inserted", 1),
		"newline":         strings.Replace(string(good), translated, translated+"\nnew code", 1),
	}
	for name, edit := range map[string][2]string{
		"code":                 {"const message", "var changed"},
		"literal":              {"\"literal\"", "\"changed\""},
		"protected-comment":    {"//go:generate echo keep", "//go:generate echo changed"},
		"unauthorized-comment": {"保留另一条说明。", "擅自改写另一条说明。"},
		"unrelated-prose":      {"Keep unrelated prose.", "Changed unrelated prose."},
	} {
		// Restoring an opaque region by hand still cannot authorize its edit.
		found := false
		for i, value := range p.Values {
			if strings.Contains(value, edit[0]) {
				badOutputs[name] = strings.Replace(string(good), p.Boundaries[i], strings.Replace(value, edit[0], edit[1], 1), 1)
				found = true
				break
			}
		}
		if !found {
			t.Fatal("negative control not bound to actual protected bytes", name)
		}
	}
	for name, bad := range badOutputs {
		t.Run(name, func(t *testing.T) {
			write([]byte(bad))
			if _, _, err := i18n.PackGenerationResultBundle(root, c, revision, "chatgpt", i18n.FormalChatGPTGenerationModel, dir); err == nil {
				t.Fatal("unsafe output accepted")
			}
		})
	}
	write(good)
	t.Run("projection-dependency-currentness", func(t *testing.T) {
		path := "internal/sitecontent/comment_revision_transport.go"
		before, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, path, append(bytes.Clone(before), []byte("\n// changed transport dependency\n")...))
		defer writeFixture(t, root, path, before)
		if _, _, err := i18n.PackGenerationResultBundle(root, c, revision, "chatgpt", i18n.FormalChatGPTGenerationModel, dir); err == nil || !strings.Contains(err.Error(), "stale") {
			t.Fatal("old transport remained current", err)
		}
	})
	if _, err := i18n.ImportGenerationOutputDirectory(root, c, revision, "chatgpt", i18n.FormalChatGPTGenerationModel, dir); err != nil {
		t.Fatal(err)
	}
	processed, err = i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: o.Locale, BatchID: e.BatchID})
	if err != nil || processed.RestorePassed != 1 || processed.ValidationPassed != 1 {
		t.Fatal("projected import/process", processed, err)
	}
	candidate, err := os.ReadFile(filepath.Join(root, e.BatchPath, processed.Units[0].CandidatePath))
	if err != nil {
		t.Fatal(err)
	}
	baseline := files["formal/candidates/"+name]
	want := bytes.Replace(baseline, []byte("// Associate message with name."), []byte("// 将name与message关联。"), 1)
	want = bytes.Replace(want, []byte("The module path."), []byte("模块路径。"), 1)
	if !bytes.Equal(candidate, want) {
		t.Fatal("non-authorized bytes changed", string(candidate))
	}
	if err := i18n.ValidateTranslationUnitCandidate(root, c, ids[0], o.Locale, candidate); err != nil {
		t.Fatal("active canonical validator", err)
	}
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: o.Locale, SnapshotID: "comment-successor"}); err != nil {
		t.Fatal("shared Snapshot raw/candidate evidence reader", err)
	}
	after, err := os.ReadFile(filepath.Join(root, finalPath))
	if err != nil || !bytes.Equal(after, finalBefore) {
		t.Fatal("historical A changed", err)
	}
	t.Run("same-unit-bridge", func(t *testing.T) { testSameUnitBridgeLifecycle(t, root, c, "comment-successor", ids[0]) })
	if len(rm.ExpectedOutputs) != 1 || rm.TaskKind != "translation-unit-revision" {
		t.Fatal("formal output boundary changed")
	}
}
