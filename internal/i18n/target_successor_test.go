package i18n

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func historicalRecoveryFixture(t *testing.T, configure ...func(*Catalog)) (string, *Catalog, string) {
	t.Helper()
	root, cat, g1 := compatibilityFixture(t, configure...)
	s, err := readQualityCheckSnapshot(root, "zh-CN", "qc-001", false)
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{"locales/zh-CN/glossary.yaml", "locales/zh-CN/status.tsv", "locales/zh-CN/course-metadata.json", "internal/tour/ui/zh-CN.json", "data/quality-check-snapshots/zh-CN/qc-001/manifest.json", "data/quality-check-snapshots/zh-CN/qc-001/quality-check-results.json", "data/quality-check-snapshots/zh-CN/qc-001/finalization.json"}
	for _, u := range s.Units {
		paths = append(paths, u.CandidatePath, u.ValidationPath, canonicalCandidatePath("zh-CN", u.UnitID))
	}
	sort.Strings(paths)
	refs := []contentidentity.Reference{}
	for _, p := range paths {
		b, err := readCompatibilityFile(root, p)
		if err != nil {
			t.Fatal(err)
		}
		refs = append(refs, contentidentity.Reference{Path: p, SHA256: sum(b)})
	}
	c := contentidentity.Completion{Package: "tour-v1", State: "complete", EvidenceKind: "legacy-tour-closure/v1", Evidence: refs, EvidenceIdentity: sum(mustJSON(refs))}
	writeSnapshotSource(t, root, "locales/zh-CN/content-scope.json", mustJSON(contentidentity.Locale{Schema: contentidentity.LocaleSchema, Locale: "zh-CN", Packages: []contentidentity.Completion{c}}))
	batch, err := ExportRetranslationBatch(root, cat, RetranslationExportOptions{Locale: "zh-CN", BatchID: "codex-zh-CN-002", UnitIDs: []string{"lesson/1"}, AllowReexport: true})
	if err != nil {
		t.Fatal(err)
	}
	m := readRetranslationManifest(t, root, "codex-zh-CN-002")
	input, err := os.ReadFile(filepath.Join(root, batch.BatchPath, m.Units[0].InputPath))
	if err != nil {
		t.Fatal(err)
	}
	raw := strings.ReplaceAll(string(input), "* Page", "* 修订页面")
	raw = strings.ReplaceAll(raw, "Use ", "本页使用 ")
	raw = strings.ReplaceAll(raw, " on this page.", "。")
	writeSnapshotSource(t, root, batch.BatchPath+"/raw-responses/lesson-1.article", []byte(raw))
	if _, err := ProcessRetranslationBatch(root, cat, RetranslationProcessOptions{Locale: "zh-CN", BatchID: "codex-zh-CN-002"}); err != nil {
		t.Fatal(err)
	}
	compatibilityChange(t, root, "# successor fixture\n"+compatibilityFixtureGlossary, "next")
	_, compat, err := AssessGlossaryCompatibility(root, "zh-CN", g1, cat)
	if err != nil {
		t.Fatal(err)
	}
	s2, _, err := CreateQualityCheckCandidateSnapshot(root, cat, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-002"})
	if err != nil {
		t.Fatal(err)
	}
	recordQualityCheckRatings(t, root, cat, "qc-002", "qc-001", "A", []string{"lesson/1"})
	if _, _, err := FinalizeQualityCheck(root, cat, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "qc-002"}); err != nil {
		t.Fatal(err)
	}
	b, err := readCompatibilityFile(root, s2.Units[0].CandidatePath)
	if err != nil {
		t.Fatal(err)
	}
	writeSnapshotSource(t, root, canonicalCandidatePath("zh-CN", "lesson/1"), canonicalizeCandidateEOF(b))
	return root, cat, compat
}

func TestHistoricalPromotionRecoveryProofAndConsumerBoundaries(t *testing.T) {
	root, cat, compat := historicalRecoveryFixture(t)
	immutable := []string{"locales/zh-CN/content-scope.json", compat, "data/quality-check-snapshots/zh-CN/qc-002/manifest.json", "data/quality-check-snapshots/zh-CN/qc-002/quality-check-results.json", "data/quality-check-snapshots/zh-CN/qc-002/finalization.json"}
	before := map[string][]byte{}
	for _, p := range immutable {
		before[p], _ = readCompatibilityFile(root, p)
	}
	e, path, err := RecoverHistoricalPromotion(root, cat, "zh-CN", "qc-002", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Units) != 1 || !e.Units[0].Changed || len(e.Membership) != 3 {
		t.Fatalf("incorrect changed membership: %+v", e)
	}
	if _, err := os.Stat(filepath.Join(root, path)); !os.IsNotExist(err) {
		t.Fatal("dry-run created artifact")
	}
	e2, path2, err := RecoverHistoricalPromotion(root, cat, "zh-CN", "qc-002", false)
	if err != nil || path != path2 || !bytes.Equal(mustJSON(e), mustJSON(e2)) {
		t.Fatalf("nondeterministic recovery: %v", err)
	}
	if err := AuthorizeHistoricalCanonicalReference(root, "zh-CN", e.Units[0].CanonicalPath, e.Units[0].OldSHA, cat); err == nil {
		t.Fatal("current bytes self-authorized without recovery")
	}
	diagnostic, err := DiagnoseQualityCheckAuthorization(root, cat, "zh-CN", "qc-002")
	if err != nil || diagnostic.ReviewerACount != 3 || diagnostic.CanonicalAuthorized {
		t.Fatalf("historical judgment confused with authorization: %+v %v", diagnostic, err)
	}
	scope, err := BuildQualityCheckScope(root, cat, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "qc-002"})
	if err != nil {
		t.Fatal(err)
	}
	if scope.ACount == 3 {
		t.Fatal("stale full inventory silently authorized")
	}
	if _, _, err := RecoverHistoricalPromotion(root, cat, "zh-CN", "qc-002", true); err != nil {
		t.Fatal(err)
	}
	if _, _, err := RecoverHistoricalPromotion(root, cat, "zh-CN", "qc-002", true); err == nil {
		t.Fatal("duplicate recovery apply overwrote artifact")
	}
	if _, err := verifyTargetSuccessor(root, path, cat); err != nil {
		t.Fatal(err)
	}
	diagnostic, err = DiagnoseQualityCheckAuthorization(root, cat, "zh-CN", "qc-002")
	if err != nil || diagnostic.ReviewerACount != 3 || !diagnostic.CanonicalAuthorized {
		t.Fatalf("successor did not authorize unchanged judgment: %+v %v", diagnostic, err)
	}
	scope, err = BuildQualityCheckScope(root, cat, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "qc-002"})
	if err != nil || scope.ACount != 3 || scope.PendingCount != 0 {
		t.Fatalf("lost immutable A judgment: %+v %v", scope, err)
	}
	if _, err := VerifyQualityCheckFinalization(root, cat, "zh-CN", "qc-002"); err != nil {
		t.Fatal(err)
	}
	if err := AuthorizeHistoricalCanonicalReference(root, "zh-CN", e.Units[0].CanonicalPath, e.Units[0].OldSHA, cat); err != nil {
		t.Fatal(err)
	}
	g1 := readSnapshotGlossarySHA(t, root, "qc-001")
	g2 := e.GlossarySHA
	if r := ResolveGlossaryCompatibility(root, "zh-CN", g1, g2, "seo-page:lesson/1", cat); r.Status == "compatible" {
		t.Fatal("TU successor carried derived SEO")
	}
	for _, p := range immutable {
		after, _ := readCompatibilityFile(root, p)
		if !bytes.Equal(before[p], after) {
			t.Fatalf("historical bytes changed %s", p)
		}
	}
	statuses, err := ReadStatuses(filepath.Join(root, "locales/zh-CN/status.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	for i := range statuses {
		statuses[i].State = "published"
		statuses[i].Note = "synthetic subsequent state transition"
	}
	if err := writeStatuses(filepath.Join(root, "locales/zh-CN/status.tsv"), statuses); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyTargetSuccessor(root, path, cat); err != nil {
		t.Fatalf("status update invalidated recovery target relation: %v", err)
	}
	writeSnapshotSource(t, root, e.Units[0].CanonicalPath, []byte("* 手工修改\n\n非法当前 target。\n"))
	if _, err := verifyTargetSuccessor(root, path, cat); err == nil {
		t.Fatal("manual canonical edit authorized")
	}
}

func readSnapshotGlossarySHA(t *testing.T, root, id string) string {
	t.Helper()
	s, err := readQualityCheckSnapshot(root, "zh-CN", id, false)
	if err != nil {
		t.Fatal(err)
	}
	return s.GlossarySHA256
}

func TestHistoricalPromotionRecoveryCompatibleGlossaryRefresh(t *testing.T) {
	root, cat, _ := historicalRecoveryFixture(t)
	e, path, err := RecoverHistoricalPromotion(root, cat, "zh-CN", "qc-002", true)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := readCompatibilityFile(root, path)
	data := strings.Replace(compatibilityFixtureGlossary, "  Go: Go\n", "  Go: Go\n  Welcome: 欢迎\n", 1)
	compatibilityChange(t, root, "# subsequent refresh\n"+data, "subsequent")
	edge, _, err := AssessGlossaryCompatibility(root, "zh-CN", e.GlossarySHA, cat)
	if err != nil {
		t.Fatal(err)
	}
	if len(edge.Affected) == 0 {
		t.Fatal("fixture lacks independent affected consumer")
	}
	if _, err := verifyTargetSuccessor(root, path, cat); err != nil {
		t.Fatalf("historical event expired: %v", err)
	}
	d, err := DiagnoseQualityCheckAuthorization(root, cat, "zh-CN", "qc-002")
	if err != nil || d.ReviewerACount != 3 || d.CanonicalAuthorized {
		t.Fatalf("unfinalized current language authorized: %+v %v", d, err)
	}
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, cat, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-003"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := FinalizeQualityCheck(root, cat, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "qc-003", PreviousSnapshotID: "qc-002"}); err != nil {
		t.Fatal(err)
	}
	d, err = DiagnoseQualityCheckAuthorization(root, cat, "zh-CN", "qc-002")
	if err != nil || !d.CanonicalAuthorized || d.ReviewerACount != 3 {
		t.Fatalf("current carry not authorized: %+v %v", d, err)
	}
	for _, input := range []string{"internal/tour/ui/zh-CN.json", "internal/tour/ui/en.json", "locales/zh-CN/course-metadata.json", "locales/zh-CN/article-metadata.json", "internal/tour/languages.go", "internal/tour/project.go", "internal/tour/seo.go"} {
		original, err := readCompatibilityFile(root, input)
		if err != nil {
			t.Fatal(err)
		}
		writeSnapshotSource(t, root, input, append(append([]byte(nil), original...), '\n'))
		d, err := DiagnoseQualityCheckAuthorization(root, cat, "zh-CN", "qc-002")
		writeSnapshotSource(t, root, input, original)
		if err != nil || !d.CanonicalAuthorized {
			t.Fatalf("non-TU current ref revoked compatible finalized TU: %s: %+v %v", input, d, err)
		}
	}
	if r := ResolveGlossaryCompatibility(root, "zh-CN", e.GlossarySHA, edge.NewGlossary.SHA256, "ui:test.one", cat); r.Status == "compatible" {
		t.Fatal("TU carried affected UI")
	}
	after, _ := readCompatibilityFile(root, path)
	if !bytes.Equal(before, after) {
		t.Fatal("recovery rewritten")
	}
	for _, input := range []string{path, "data/quality-check-snapshots/zh-CN/qc-002/finalization.json", "data/quality-check-snapshots/zh-CN/qc-003/finalization.json", "data/quality-check-snapshots/zh-CN/qc-003/manifest.json"} {
		original, err := readCompatibilityFile(root, input)
		if err != nil {
			t.Fatal(err)
		}
		writeSnapshotSource(t, root, input, append(append([]byte(nil), original...), []byte("\n{}")...))
		if d, err := DiagnoseQualityCheckAuthorization(root, cat, "zh-CN", "qc-002"); err == nil && d.CanonicalAuthorized {
			t.Fatalf("tampered current/historical authority accepted: %s", input)
		}
		writeSnapshotSource(t, root, input, original)
	}
	writeSnapshotSource(t, root, e.Units[0].CanonicalPath, []byte("* manual edit\n"))
	if _, err := verifyTargetSuccessor(root, path, cat); err == nil {
		t.Fatal("manual edit authorized")
	}
}

func TestHistoricalPromotionRecoveryRejectsBrokenProof(t *testing.T) {
	for _, kind := range []string{"old-sha", "canonical", "candidate", "validation", "source", "glossary", "finalization", "reviewer", "unaffected-evidence", "wrong-snapshot", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			root, cat, _ := historicalRecoveryFixture(t)
			s, err := readQualityCheckSnapshot(root, "zh-CN", "qc-002", false)
			if err != nil {
				t.Fatal(err)
			}
			path := ""
			id := "qc-002"
			switch kind {
			case "old-sha":
				b, _ := readCompatibilityFile(root, "locales/zh-CN/content-scope.json")
				var l contentidentity.Locale
				if err := contentidentity.StrictJSON(b, &l); err != nil {
					t.Fatal(err)
				}
				for i := range l.Packages[0].Evidence {
					r := &l.Packages[0].Evidence[i]
					if r.Path == canonicalCandidatePath("zh-CN", "lesson/1") {
						r.SHA256 = sum([]byte("invented old target"))
					}
				}
				l.Packages[0].EvidenceIdentity = sum(mustJSON(l.Packages[0].Evidence))
				writeSnapshotSource(t, root, "locales/zh-CN/content-scope.json", mustJSON(l))
			case "canonical":
				path = canonicalCandidatePath("zh-CN", "lesson/1")
			case "candidate":
				path = s.Units[0].CandidatePath
			case "validation":
				path = s.Units[0].ValidationPath
			case "source":
				path = s.Units[0].SourcePath
			case "glossary":
				path = "locales/zh-CN/glossary.yaml"
			case "finalization":
				path = "data/quality-check-snapshots/zh-CN/qc-002/finalization.json"
			case "reviewer":
				p := qualityCheckResultsPath(root, "zh-CN", "qc-002")
				b, _ := os.ReadFile(p)
				b = bytes.Replace(b, []byte(`"rating": "A"`), []byte(`"rating": "B", "finding": "real finding"`), 1)
				writeSnapshotSource(t, root, filepath.ToSlash(strings.TrimPrefix(p, root+"/")), b)
			case "unaffected-evidence":
				path = "internal/tour/ui/zh-CN.json"
			case "wrong-snapshot":
				id = "qc-001"
			case "symlink":
				p := filepath.Join(root, s.Units[0].ValidationPath)
				b, _ := os.ReadFile(p)
				target := filepath.Join(t.TempDir(), "validation.json")
				if err := os.WriteFile(target, b, 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(p); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, p); err != nil {
					t.Fatal(err)
				}
			}
			if path != "" {
				b, _ := readCompatibilityFile(root, path)
				writeSnapshotSource(t, root, path, append(b, []byte(" changed")...))
			}
			if _, _, err := RecoverHistoricalPromotion(root, cat, "zh-CN", id, false); err == nil {
				t.Fatal("accepted broken proof")
			}
		})
	}
}

func TestHistoricalPromotionRecoveryStrictArtifact(t *testing.T) {
	root, cat, _ := historicalRecoveryFixture(t)
	e, path, err := RecoverHistoricalPromotion(root, cat, "zh-CN", "qc-002", true)
	if err != nil {
		t.Fatal(err)
	}
	original, _ := readCompatibilityFile(root, path)
	for _, b := range [][]byte{append(original, []byte(` {}`)...), bytes.Replace(original, []byte(`"locale": "zh-CN"`), []byte(`"locale": "zh-CN", "locale": "zh-CN"`), 1), bytes.Replace(original, []byte(`"locale": "zh-CN"`), []byte(`"locale": "zh-CN", "unknown": true`), 1)} {
		writeSnapshotSource(t, root, path, b)
		if _, err := verifyTargetSuccessor(root, path, cat); err == nil {
			t.Fatal("accepted invalid strict JSON")
		}
	}
	e.Schema = NativePromotionReceiptSchema
	e.Kind = "native-promotion-receipt"
	e.Reason = "formal-promotion-apply"
	e.Identity = successorIdentity(*e)
	writeSnapshotSource(t, root, path, mustJSON(e))
	if _, err := verifyTargetSuccessor(root, path, cat); err == nil {
		t.Fatal("recovery impersonated native receipt")
	}
}

func nativeReceiptFixture(t *testing.T) (string, *Catalog) {
	t.Helper()
	root, cat, _ := compatibilityFixture(t)
	p := canonicalCandidatePath("zh-CN", "lesson/1")
	b, err := readCompatibilityFile(root, p)
	if err != nil {
		t.Fatal(err)
	}
	writeSnapshotSource(t, root, p, bytes.Replace(b, []byte("* 新页面"), []byte("* 旧页面"), 1))
	return root, cat
}

func TestNativePromotionReceiptAndAtomicFailures(t *testing.T) {
	for _, stage := range []int{0, 1, 2, 3, 4, 5, 6} {
		t.Run(string(rune('0'+stage)), func(t *testing.T) {
			root, cat := nativeReceiptFixture(t)
			before, _ := readCompatibilityFile(root, "locales/zh-CN/status.tsv")
			canonicalBefore, _ := readCompatibilityFile(root, canonicalCandidatePath("zh-CN", "lesson/1"))
			options := RetranslationPromoteOptions{Locale: "zh-CN", SnapshotID: "qc-001", Apply: true}
			if stage == 5 {
				options.stageReceipt = func(*TargetSuccessorEvidence, string, string) error {
					return errors.New("injected receipt write failure")
				}
			}
			if stage == 6 {
				options.publishReceipt = func(a, b string) error { return errors.New("injected final receipt publication failure") }
			}
			calls := 0
			options.rename = func(a, b string) error {
				calls++
				if calls == stage {
					return errors.New("injected rename failure")
				}
				return os.Rename(a, b)
			}
			_, err := PromoteRetranslation(root, cat, options)
			if stage != 0 {
				if err == nil {
					t.Fatal("failure not returned")
				}
				after, _ := readCompatibilityFile(root, "locales/zh-CN/status.tsv")
				if !bytes.Equal(before, after) {
					t.Fatal("failure changed status")
				}
				canonicalAfter, _ := readCompatibilityFile(root, canonicalCandidatePath("zh-CN", "lesson/1"))
				if !bytes.Equal(canonicalBefore, canonicalAfter) {
					t.Fatal("failed apply changed canonical")
				}
				entries, _ := os.ReadDir(filepath.Join(root, "locales/zh-CN/candidates"))
				for _, e := range entries {
					if strings.HasPrefix(e.Name(), ".promotion-") {
						t.Fatal("failed apply published receipt")
					}
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			proof, err := findCurrentTargetSuccessor(root, "zh-CN", cat)
			if err != nil || proof == nil {
				t.Fatalf("native receipt invalid: %v", err)
			}
			if proof.evidence.Schema != NativePromotionReceiptSchema || len(proof.evidence.Units) != 3 || len(proof.evidence.Membership) != 3 {
				t.Fatal("native membership mismatch")
			}
			if !proof.evidence.Units[0].Changed || proof.evidence.Units[0].OldSHA == proof.evidence.Units[0].NewSHA || proof.evidence.Units[1].Changed {
				t.Fatal("wrong native old→new binding")
			}
			if _, _, err := RecoverHistoricalPromotion(root, cat, "zh-CN", "qc-001", false); err == nil || !strings.Contains(err.Error(), "native promotion receipt already exists") {
				t.Fatalf("recovery reason accepted existing native receipt: %v", err)
			}
			statusApplied, _ := readCompatibilityFile(root, "locales/zh-CN/status.tsv")
			if _, err := PromoteRetranslation(root, cat, options); err == nil || !strings.Contains(err.Error(), "native promotion receipt already exists") {
				t.Fatalf("duplicate Snapshot apply accepted: %v", err)
			}
			statusAfterDuplicate, _ := readCompatibilityFile(root, "locales/zh-CN/status.tsv")
			if !bytes.Equal(statusApplied, statusAfterDuplicate) {
				t.Fatal("duplicate Snapshot apply mutated status")
			}
			statuses, err := ReadStatuses(filepath.Join(root, "locales/zh-CN/status.tsv"))
			if err != nil {
				t.Fatal(err)
			}
			for i := range statuses {
				statuses[i].State = "published"
				statuses[i].Note = "synthetic subsequent state transition"
			}
			if err := writeStatuses(filepath.Join(root, "locales/zh-CN/status.tsv"), statuses); err != nil {
				t.Fatal(err)
			}
			if _, err := verifyTargetSuccessor(root, mustSuccessorPath(t, proof.evidence), cat); err != nil {
				t.Fatalf("mutable status retroactively invalidated TU receipt: %v", err)
			}
			writeSnapshotSource(t, root, proof.evidence.Units[0].CanonicalPath, []byte("manual mutation"))
			if _, err := verifyTargetSuccessor(root, mustSuccessorPath(t, proof.evidence), cat); err == nil {
				t.Fatal("native receipt authorized manual mutation")
			}
		})
	}
}

func TestNativePromotionReceiptCrashBoundaries(t *testing.T) {
	for _, stage := range []int{1, 2, 3, 4, 5} {
		t.Run(string(rune('0'+stage)), func(t *testing.T) {
			root, cat := nativeReceiptFixture(t)
			before, _ := readCompatibilityFile(root, canonicalCandidatePath("zh-CN", "lesson/1"))
			calls := 0
			opts := RetranslationPromoteOptions{Locale: "zh-CN", SnapshotID: "qc-001", Apply: true}
			opts.rename = func(a, b string) error {
				if err := os.Rename(a, b); err != nil {
					return err
				}
				calls++
				if calls == stage {
					panic("simulated interruption")
				}
				return nil
			}
			opts.publishReceipt = func(a, b string) error {
				if err := os.Link(a, b); err != nil {
					return err
				}
				if stage == 5 {
					panic("simulated interruption after receipt commit")
				}
				return nil
			}
			interrupted := false
			func() {
				defer func() {
					if recover() != nil {
						interrupted = true
					}
				}()
				_, _ = PromoteRetranslation(root, cat, opts)
			}()
			if !interrupted {
				t.Fatal("interruption not reached")
			}
			staging := filepath.Join(root, "locales/zh-CN/.promotion-staging-transaction")
			backup, err := os.ReadFile(filepath.Join(staging, "old-candidates/lesson-1.article"))
			if err != nil || !bytes.Equal(backup, before) {
				t.Fatalf("crash lost old canonical bytes: %v", err)
			}
			if err := CheckPromotionTransactionState(root, "zh-CN"); err == nil {
				t.Fatal("interrupted transaction treated as complete")
			}
			if proof, err := findCurrentTargetSuccessor(root, "zh-CN", cat); err == nil || proof != nil {
				t.Fatal("interrupted staging authorized current target")
			}
			if _, err := PromoteRetranslation(root, cat, opts); err == nil || !strings.Contains(err.Error(), "pending/unrecovered") {
				t.Fatalf("interrupted transaction replayed: %v", err)
			}
			if _, _, err := RecoverHistoricalPromotion(root, cat, "zh-CN", "qc-001", false); err == nil || !strings.Contains(err.Error(), "pending/unrecovered") {
				t.Fatalf("recovery bypassed interrupted transaction: %v", err)
			}
		})
	}
}

func TestNativePromotionReceiptInstalledOutputPrepublication(t *testing.T) {
	for _, mutation := range []string{"canonical", "status", "receipt"} {
		t.Run(mutation, func(t *testing.T) {
			root, cat := nativeReceiptFixture(t)
			path := canonicalCandidatePath("zh-CN", "lesson/1")
			before, _ := readCompatibilityFile(root, path)
			statusBefore, _ := readCompatibilityFile(root, "locales/zh-CN/status.tsv")
			calls := 0
			opts := RetranslationPromoteOptions{Locale: "zh-CN", SnapshotID: "qc-001", Apply: true}
			opts.rename = func(a, b string) error {
				if err := os.Rename(a, b); err != nil {
					return err
				}
				calls++
				if calls == 4 {
					p := path
					if mutation == "status" {
						p = "locales/zh-CN/status.tsv"
					}
					if mutation == "receipt" {
						p = "locales/zh-CN/.promotion-staging-transaction/promotion-receipt.json"
					}
					writeSnapshotSource(t, root, p, []byte("injected partial installed output"))
				}
				return nil
			}
			if _, err := PromoteRetranslation(root, cat, opts); err == nil || !strings.Contains(err.Error(), "publish native promotion receipt") {
				t.Fatalf("published partial output: %v", err)
			}
			after, _ := readCompatibilityFile(root, path)
			statusAfter, _ := readCompatibilityFile(root, "locales/zh-CN/status.tsv")
			if !bytes.Equal(before, after) || !bytes.Equal(statusBefore, statusAfter) {
				t.Fatal("failed publication did not restore exact old tree/status")
			}
			if proof, err := findCurrentTargetSuccessor(root, "zh-CN", cat); err != nil || proof != nil {
				t.Fatalf("failed publication left authority/state: %v", err)
			}
		})
	}
}

func mustSuccessorPath(t *testing.T, e *TargetSuccessorEvidence) string {
	t.Helper()
	p, err := successorPath(e)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestNativePromotionReceiptPrecommitAndRetainedRecovery(t *testing.T) {
	t.Run("committed-cleanup-failure-is-explicit", func(t *testing.T) {
		root, cat := nativeReceiptFixture(t)
		opts := RetranslationPromoteOptions{Locale: "zh-CN", SnapshotID: "qc-001", Apply: true, removeStaging: func(string) error { return errors.New("cleanup failed") }}
		if _, err := PromoteRetranslation(root, cat, opts); err == nil || !strings.Contains(err.Error(), "mutation=known; committed=true; retained") {
			t.Fatalf("silent transaction cleanup failure: %v", err)
		}
		if proof, err := findCurrentTargetSuccessor(root, "zh-CN", cat); err == nil || proof != nil {
			t.Fatal("unclosed commit authorized without inspecting retained staging")
		}
	})
	t.Run("no-unreceipted-public-apply", func(t *testing.T) {
		root, cat := nativeReceiptFixture(t)
		if _, err := PromoteRetranslation(root, cat, RetranslationPromoteOptions{Locale: "zh-CN", Apply: true}); err == nil {
			t.Fatal("public apply without finalized Snapshot accepted")
		}
	})
	t.Run("precommit-input-change", func(t *testing.T) {
		root, cat := nativeReceiptFixture(t)
		before, _ := readCompatibilityFile(root, canonicalCandidatePath("zh-CN", "lesson/1"))
		opts := RetranslationPromoteOptions{Locale: "zh-CN", SnapshotID: "qc-001", Apply: true}
		opts.stageReceipt = func(e *TargetSuccessorEvidence, candidates, status string) error {
			if err := stageNativePromotionReceipt(e, candidates, status); err != nil {
				return err
			}
			writeSnapshotSource(t, root, e.Units[0].Candidate.Path, []byte("concurrent input mutation"))
			return nil
		}
		if _, err := PromoteRetranslation(root, cat, opts); err == nil || !strings.Contains(err.Error(), "precommit input changed") {
			t.Fatalf("did not reject changed input before install: %v", err)
		}
		after, _ := readCompatibilityFile(root, canonicalCandidatePath("zh-CN", "lesson/1"))
		if !bytes.Equal(before, after) {
			t.Fatal("precommit failure installed canonical")
		}
	})
	t.Run("rollback-failure-preserves-backups", func(t *testing.T) {
		root, cat := nativeReceiptFixture(t)
		before, _ := readCompatibilityFile(root, canonicalCandidatePath("zh-CN", "lesson/1"))
		calls := 0
		opts := RetranslationPromoteOptions{Locale: "zh-CN", SnapshotID: "qc-001", Apply: true}
		opts.rename = func(a, b string) error {
			calls++
			if calls == 3 {
				return errors.New("status backup failure")
			}
			return os.Rename(a, b)
		}
		opts.rollbackRename = func(a, b string) error { return errors.New("rollback failure") }
		if _, err := PromoteRetranslation(root, cat, opts); err == nil || !strings.Contains(err.Error(), "mutation=unknown; retained") {
			t.Fatalf("lost explicit recovery state: %v", err)
		}
		staging, err := filepath.Glob(filepath.Join(root, "locales/zh-CN/.promotion-staging-*"))
		if err != nil || len(staging) != 1 {
			t.Fatalf("lost staging backup: %v %v", staging, err)
		}
		backup, err := os.ReadFile(filepath.Join(staging[0], "old-candidates/lesson-1.article"))
		if err != nil || !bytes.Equal(backup, before) {
			t.Fatalf("lost exact old canonical backup: %v", err)
		}
		if p, err := findCurrentTargetSuccessor(root, "zh-CN", cat); err == nil && p != nil {
			t.Fatal("partial status install authorized receipt")
		}
	})
}
