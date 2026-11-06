package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
	"github.com/shuijingwan/go-tour-i18n/internal/languageview"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

// Synthetic plain-text representation, used only in TempDir CLI fixtures.
type authorizationDiagnosisAdapter struct {
	source         string
	validationFail bool
	unknown        bool
}

func (a *authorizationDiagnosisAdapter) LanguageAuditDocuments(target []byte) (languageaudit.Document, languageaudit.Document, error) {
	project := func(text string) languageaudit.Document {
		return languageaudit.Document{Owner: "synthetic-diagnostic", Blocks: []languageaudit.Block{{Locator: "message", Context: "paragraph", Pieces: languageview.Plain(text)}}}
	}
	return project(a.source), project(string(target)), nil
}

func (a *authorizationDiagnosisAdapter) ProtectedInput(*i18n.Glossary) (string, []string, error) {
	return a.source, nil, nil
}
func (a *authorizationDiagnosisAdapter) Restore(raw string) ([]byte, error) {
	return []byte(raw), nil
}
func (a *authorizationDiagnosisAdapter) Validate(b []byte, g *i18n.Glossary) error {
	if a.validationFail {
		return fmt.Errorf("synthetic current validator failure")
	}
	return i18n.ValidateParsedGlossary(a.source, string(b), g)
}
func (a *authorizationDiagnosisAdapter) ReattestationAuthority(string) (string, error) {
	if a.unknown {
		return "", fmt.Errorf("synthetic unknown adapter")
	}
	return courseContextSHA([]byte("synthetic-authorization-diagnosis/v1")), nil
}
func (a *authorizationDiagnosisAdapter) GlossaryVisiblePair(b []byte) (string, string, error) {
	return a.source, string(b), nil
}
func (*authorizationDiagnosisAdapter) CommentSourceBlocks() ([]i18n.CommentCodeBlock, string, error) {
	return nil, "", nil
}
func (*authorizationDiagnosisAdapter) CommentTargetBlocks([]byte) ([]i18n.CommentCodeBlock, error) {
	return nil, nil
}

func authorizationDiagnosisMachineRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	// Copy executable/command authority bytes, never real locale evidence.
	for _, dir := range []string{"docs", "internal/i18n", "internal/sitecontent"} {
		if err := filepath.WalkDir(filepath.Join(repo, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || (filepath.Ext(path) != ".md" && filepath.Ext(path) != ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, err := filepath.Rel(repo, path)
			if err != nil {
				return err
			}
			courseContextWrite(t, root, rel, courseContextRead(t, repo, rel))
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{"AGENTS.md", "go.mod", "go.sum", "internal/sitecontent/same_unit_bridge.go", "internal/sitecontent/comment_revision_transport.go"} {
		courseContextWrite(t, root, path, courseContextRead(t, repo, path))
	}
	// Current authority includes the shared scanner/view implementation. These
	// are exact copied code inputs, never a fallback policy or validator bypass.
	for _, path := range i18n.LanguageAuditAuthorityPaths {
		courseContextWrite(t, root, path, courseContextRead(t, repo, path))
	}
	return root
}

func authorizationDiagnosisFixture(t *testing.T, count int) (string, *i18n.Catalog, string) {
	t.Helper()
	root := authorizationDiagnosisMachineRoot(t)
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	courseContextWrite(t, root, "locales/zh-CN/locale.json", courseContextRead(t, repo, "locales/zh-CN/locale.json"))
	courseContextWrite(t, root, "locales/zh-CN/glossary.yaml", []byte("locale: zh-CN\nmandatory:\n  Go: Go\n"))
	corpus := i18n.GlossarySourceCorpus{Schema: i18n.GlossaryCorpusSchema, Sources: []i18n.GlossaryArchiveReference{}, Contributors: []i18n.LanguageContributor{{ID: "synthetic-diagnostic", Parser: "present-visible/v1", Package: "tour-v1", Path: "fixtures/diagnostic.txt", Contexts: []i18n.LanguageContext{{ID: "synthetic", Text: "Go", Protected: []string{}}}}}}
	corpus.Contributors[0].SourceSHA = i18n.ContributorIdentity(corpus.Contributors[0])
	corpus.Identity = i18n.CorpusIdentity(corpus)
	courseContextWrite(t, root, i18n.GlossaryCorpusPath, courseContextJSON(t, corpus))
	if _, _, err := i18n.RecordGlossaryReview(root, "zh-CN", "synthetic-diagnostic", "synthetic-reviewer", "passed", nil); err != nil {
		t.Fatal(err)
	}
	bundle, _, err := i18n.ExportUnifiedGlossaryBundle(root, "zh-CN", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.RecordUnifiedGlossaryReview(root, bundle, "synthetic-diagnostic", "synthetic-reviewer", "synthetic-generator", "passed", nil); err != nil {
		t.Fatal(err)
	}
	c := &i18n.Catalog{}
	d := i18n.TranslationPackageDescriptor{ID: "diagnostic-fixture", AuthoritySHA256: courseContextSHA([]byte("synthetic-original-package"))}
	ids := []string{}
	for i := 1; i <= count; i++ {
		id, path := fmt.Sprintf("diagnostic/page-%03d", i), fmt.Sprintf("fixtures/page-%03d.txt", i)
		source := []byte(fmt.Sprintf("Read the message number %d.\n", i))
		courseContextWrite(t, root, path, source)
		d.Units = append(d.Units, i18n.AdaptedTranslationUnit{Unit: i18n.TranslationUnit{ID: id, Kind: i18n.UnitKindPage, SourcePath: path, Source: source, SourceSHA256: courseContextSHA(source)}, Adapter: &authorizationDiagnosisAdapter{source: string(source)}, StableIndex: i, Route: fmt.Sprintf("/diagnostic/%d", i), WorkingSet: "initial", WorkingSetLimit: 60})
		ids = append(ids, id)
	}
	if err := c.RegisterTranslationPackage(d); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage(d.ID); err != nil {
		t.Fatal(err)
	}
	e, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorCodex, UnitIDs: ids})
	if err != nil {
		t.Fatal(err)
	}
	var m i18n.RetranslationBatchManifest
	if err := json.Unmarshal(courseContextRead(t, root, e.BatchPath+"/manifest.json"), &m); err != nil {
		t.Fatal(err)
	}
	for _, u := range m.Units {
		courseContextWrite(t, root, e.BatchPath+"/raw-responses/"+filepath.Base(u.InputPath), courseContextRead(t, root, e.BatchPath+"/"+u.InputPath))
	}
	if p, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID}); err != nil || p.ValidationPassed != count {
		t.Fatal("synthetic processed fixture", p, err)
	}
	const snapshot = "diagnostic-finalized"
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: snapshot}); err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: snapshot, UnitIDs: ids, Rating: "A", ReviewerSession: "synthetic-independent-reviewer"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.FinalizeQualityCheck(root, c, i18n.QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: snapshot}); err != nil {
		t.Fatal(err)
	}
	c.TranslationScope = nil
	return root, c, snapshot
}

func authorizationDiagnosisOutput(t *testing.T, call func() error) ([]byte, error) {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "stdout-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	original := os.Stdout
	os.Stdout = f
	defer func() { os.Stdout = original }()
	callErr := call()
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	return b, callErr
}

func authorizationDiagnosisInventory(t *testing.T, root string) map[string]string {
	t.Helper()
	refs := map[string]string{}
	for _, dir := range []string{"data", "locales"} {
		if err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				refs[path] = courseContextSHA(courseContextRead(t, root, strings.TrimPrefix(path, root+string(filepath.Separator))))
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	return refs
}

func TestQualityCheckAuthorizationRegisteredDiagnosis(t *testing.T) {
	for _, classification := range []string{"compatible", "affected", "stale", "unknown"} {
		t.Run(classification, func(t *testing.T) {
			root, c, snapshot := authorizationDiagnosisFixture(t, 2)
			a := c.Adapted[0].Adapter.(*authorizationDiagnosisAdapter)
			reason, validation := "semantic_delta_empty", "passed"
			switch classification {
			case "affected":
				a.validationFail = true
				reason, validation = "current_validation_failed: synthetic current validator failure", "failed"
			case "stale":
				courseContextWrite(t, root, c.Adapted[0].Unit.SourcePath, []byte("changed current source\n"))
				reason, validation = "current_source_file_changed", "not_run"
			case "unknown":
				a.unknown = true
				reason, validation = "unknown_adapter_authority", "not_run"
			}
			before := authorizationDiagnosisInventory(t, root)
			b, err := authorizationDiagnosisOutput(t, func() error {
				return qualityCheckAuthorizationCommand(root, c, []string{"--locale", "zh-CN", "--snapshot-id", snapshot})
			})
			var result i18n.CurrentTranslationAuthorization
			if decodeErr := json.Unmarshal(b, &result); decodeErr != nil || len(result.Units) != 2 || result.Schema != i18n.CurrentTranslationAuthorizationSchema || result.Historical.Finalization.SnapshotID != snapshot || result.Identity == "" {
				t.Fatalf("missing complete assessment JSON: %s: %v", b, decodeErr)
			}
			x := result.Units[0]
			if x.Classification != classification || x.Validation != validation || !reflect.DeepEqual(result.Units[1].Reasons, []string{"semantic_delta_empty"}) || !strings.Contains(strings.Join(x.Reasons, "\n"), reason) || x.ValidationIdentity == "" {
				t.Fatal("classification/reasons/identity changed", x)
			}
			if classification == "compatible" {
				if err != nil || !result.Ready() {
					t.Fatal("compatible diagnosis failed", err)
				}
			} else {
				want := fmt.Sprintf("current authorization blocked: affected=%d stale=%d unknown=%d", result.Affected, result.Stale, result.Unknown)
				if err == nil || err.Error() != want || result.Ready() || result.Affected+result.Stale+result.Unknown != 1 {
					t.Fatal("blocked JSON did not retain aggregate nonzero error", err)
				}
			}
			if !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, root)) {
				t.Fatal("readonly diagnosis changed formal paths/bytes")
			}
			if _, err := os.Stat(filepath.Join(root, "data/translation-authorizations")); !os.IsNotExist(err) {
				t.Fatal("readonly diagnosis created checkpoint", err)
			}
		})
	}
}

func TestQualityCheckAuthorizationHistorical49Scope(t *testing.T) {
	root, c, snapshot := authorizationDiagnosisFixture(t, 49)
	old := c.TranslationPackages[0]
	c.HistoricalTranslationPackages = []i18n.TranslationPackageDescriptor{old}
	current := old
	current.AuthoritySHA256 = courseContextSHA([]byte("synthetic-51-page-membership"))
	current.Units = append([]i18n.AdaptedTranslationUnit{}, old.Units...)
	for i := 50; i <= 51; i++ {
		u := old.Units[0]
		u.Unit.ID, u.Unit.SourcePath, u.StableIndex, u.Route = fmt.Sprintf("diagnostic/page-%03d", i), fmt.Sprintf("fixtures/page-%03d.txt", i), i, fmt.Sprintf("/diagnostic/%d", i)
		current.Units = append(current.Units, u)
		c.Adapted = append(c.Adapted, u)
	}
	c.TranslationPackages = []i18n.TranslationPackageDescriptor{current}
	b, err := authorizationDiagnosisOutput(t, func() error {
		return qualityCheckAuthorizationCommand(root, c, []string{"--locale", "zh-CN", "--snapshot-id", snapshot})
	})
	var result i18n.CurrentTranslationAuthorization
	if json.Unmarshal(b, &result) != nil || err != nil || len(result.Units) != 49 || result.PackageAuthoritySHA != old.AuthoritySHA256 || !result.Ready() || len(c.TranslationScope) != 0 {
		t.Fatal("historical finalized scope expanded to current 51 Pages", len(result.Units), err)
	}
	for _, scope := range [][]string{{old.Units[0].Unit.ID}, {"wrong/package-unit"}} {
		copy := *c
		copy.TranslationScope = scope
		b, err := authorizationDiagnosisOutput(t, func() error {
			return qualityCheckAuthorizationCommand(root, &copy, []string{"--locale", "zh-CN", "--snapshot-id", snapshot})
		})
		if err == nil || len(b) != 0 || !strings.Contains(err.Error(), "scope mismatch") {
			t.Fatal("explicit wrong/partial scope accepted", err)
		}
	}
}

func TestQualityCheckAuthorizationSealedCheckpoint(t *testing.T) {
	root, c, snapshot := authorizationDiagnosisFixture(t, 2)
	scoped, _, err := i18n.SelectFinalizedTranslationPackage(root, c, "zh-CN", snapshot)
	if err != nil {
		t.Fatal(err)
	}
	want, path, err := i18n.EstablishCurrentTranslationAuthorization(root, scoped, "zh-CN", snapshot)
	if err != nil {
		t.Fatal(err)
	}
	before := authorizationDiagnosisInventory(t, root)
	b, err := authorizationDiagnosisOutput(t, func() error {
		return qualityCheckAuthorizationCommand(root, c, []string{"--locale", "zh-CN", "--snapshot-id", snapshot})
	})
	var got i18n.CurrentTranslationAuthorization
	if err != nil || json.Unmarshal(b, &got) != nil || !reflect.DeepEqual(*want, got) || !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, root)) {
		t.Fatal("sealed authorization identity/bytes not preserved", err)
	}
	courseContextWrite(t, root, path, []byte("{tampered checkpoint\n"))
	b, err = authorizationDiagnosisOutput(t, func() error {
		return qualityCheckAuthorizationCommand(root, c, []string{"--locale", "zh-CN", "--snapshot-id", snapshot})
	})
	if err == nil || len(b) != 0 {
		t.Fatal("corrupt checkpoint fell back to a fresh assessment", err)
	}
}

func TestQualityCheckAuthorizationNativeTourDiagnosis(t *testing.T) {
	root, c, _, _, _ := courseActivationContextFixture(t)
	sitecontent.AttachIntegratedSurfaceAuthority(c)
	want, err := i18n.DiagnoseQualityCheckAuthorization(root, c, "zh-CN", "qc-002")
	if err != nil {
		t.Fatal(err)
	}
	before := authorizationDiagnosisInventory(t, root)
	b, err := authorizationDiagnosisOutput(t, func() error {
		return qualityCheckAuthorizationCommand(root, c, []string{"--locale", "zh-CN", "--snapshot-id", "qc-002"})
	})
	var got, expected any
	if json.Unmarshal(b, &got) != nil || json.Unmarshal(courseContextJSON(t, want), &expected) != nil || !reflect.DeepEqual(got, expected) {
		t.Fatal("native Tour stdout changed")
	}
	if (err == nil) != want.CanonicalAuthorized || (err != nil && err.Error() != "current canonical authorization blocked: "+want.Reason) || !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, root)) {
		t.Fatal("native Tour failure/mutation behavior changed", err)
	}
}

func TestQualityCheckAuthorizationFlagsFailClosed(t *testing.T) {
	for _, args := range [][]string{nil, {"--locale", "zh-CN"}, {"--snapshot-id", "accepted"}, {"--locale", "zh-CN", "--snapshot-id", "accepted", "extra"}, {"--locale", "zh-CN", "--snapshot-id", "accepted", "--apply"}, {"--locale", "zh-CN", "--snapshot-id", "../outside"}, {"--locale", "zh-CN", "--snapshot-id", "/tmp/accepted.json"}} {
		b, err := authorizationDiagnosisOutput(t, func() error { return qualityCheckAuthorizationCommand(t.TempDir(), &i18n.Catalog{}, args) })
		if err == nil || len(b) != 0 {
			t.Fatal("invalid input produced assessment", args, err)
		}
	}
}
