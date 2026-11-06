package i18n

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
	"github.com/shuijingwan/go-tour-i18n/internal/languageview"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A test-only XML card source: neither Present nor Markdown/HTML slot packing.
// Only this adapter and its structure fixtures know the card representation.
// No production registry, routes, generation or review semantics use its name.
type futureCard struct {
	XMLName xml.Name `xml:"card"`
	ID      string   `xml:"id,attr"`
	Href    string   `xml:"href,attr"`
	Heading string   `xml:"heading"`
	Message string   `xml:"message"`
}
type futureCardAdapter struct{ source futureCard }

// The fixture's typed XML fields use the same normalized audit contract as
// registered production adapters; IDs/attributes are structure, not prose.
func (a futureCardAdapter) LanguageAuditDocuments(candidate []byte) (languageaudit.Document, languageaudit.Document, error) {
	return testCardLanguageAuditDocuments(a.source, candidate, false)
}

func testCardLanguageAuditDocuments(source futureCard, candidate []byte, code bool) (languageaudit.Document, languageaudit.Document, error) {
	var target futureCard
	if err := xml.Unmarshal(candidate, &target); err != nil {
		return languageaudit.Document{}, languageaudit.Document{}, err
	}
	sourceDoc := languageaudit.Document{Owner: source.ID, Blocks: []languageaudit.Block{{Locator: "heading", Context: "heading", Pieces: languageview.Plain(source.Heading)}}}
	targetDoc := languageaudit.Document{Owner: source.ID, Blocks: []languageaudit.Block{{Locator: "heading", Context: "heading", Pieces: languageview.Plain(target.Heading)}}}
	if code {
		// Reuse the native Go comment/identifier projection, never treat Go syntax
		// as prose or replace the scanner with an empty fixture view.
		example := retranslationTestExample("example:fixture-card.go", "fixtures/card.go", source.Message)
		catalog := &Catalog{Examples: []Example{example}}
		unit, err := catalog.Unit(example.ID)
		if err != nil {
			return sourceDoc, targetDoc, err
		}
		src, dst, err := NativeTourLanguageAuditDocuments("", catalog, unit, []byte(target.Message))
		if err != nil {
			return sourceDoc, targetDoc, err
		}
		for _, pair := range []struct {
			doc    *languageaudit.Document
			blocks []languageaudit.Block
		}{{&sourceDoc, src.Blocks}, {&targetDoc, dst.Blocks}} {
			for _, block := range pair.blocks {
				block.Locator = "message/" + block.Locator
				pair.doc.Blocks = append(pair.doc.Blocks, block)
			}
		}
	} else {
		sourceDoc.Blocks = append(sourceDoc.Blocks, languageaudit.Block{Locator: "message", Context: "paragraph", Pieces: languageview.Plain(source.Message)})
		targetDoc.Blocks = append(targetDoc.Blocks, languageaudit.Block{Locator: "message", Context: "paragraph", Pieces: languageview.Plain(target.Message)})
	}
	return sourceDoc, targetDoc, nil
}

func (a futureCardAdapter) CommentSourceBlocks() ([]CommentCodeBlock, string, error) {
	// The fixture's exact XML roundtrip grammar has no code/preformatted fields.
	return nil, "", nil
}
func (a futureCardAdapter) CommentTargetBlocks([]byte) ([]CommentCodeBlock, error) {
	return nil, nil
}

func (a futureCardAdapter) ReattestationAuthority(string) (string, error) {
	return sum([]byte("test-only-future-card-adapter/v1")), nil
}
func (a futureCardAdapter) GlossaryVisiblePair(candidate []byte) (string, string, error) {
	var target futureCard
	if err := xml.Unmarshal(candidate, &target); err != nil {
		return "", "", err
	}
	return a.source.Heading + "\n" + a.source.Message, target.Heading + "\n" + target.Message, nil
}

var futureCardTokens = []string{"⟬CARD_BEGIN⟭", "⟬CARD_MESSAGE⟭", "⟬CARD_END⟭"}

func (a futureCardAdapter) ProtectedInput(_ *Glossary) (string, []string, error) {
	return futureCardTokens[0] + a.source.Heading + futureCardTokens[1] + a.source.Message + futureCardTokens[2] + "\n", append([]string(nil), futureCardTokens...), nil
}
func (a futureCardAdapter) Restore(raw string) ([]byte, error) {
	raw = strings.TrimSuffix(raw, "\n")
	for _, token := range futureCardTokens {
		if strings.Count(raw, token) != 1 {
			return nil, fmt.Errorf("card token membership")
		}
	}
	if !strings.HasPrefix(raw, futureCardTokens[0]) || !strings.HasSuffix(raw, futureCardTokens[2]) {
		return nil, fmt.Errorf("card envelope")
	}
	parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(raw, futureCardTokens[0]), futureCardTokens[2]), futureCardTokens[1])
	card := a.source
	card.Heading, card.Message = parts[0], parts[1]
	b, err := xml.Marshal(card)
	return append(b, '\n'), err
}
func (a futureCardAdapter) Validate(candidate []byte, g *Glossary) error {
	var card futureCard
	if err := xml.Unmarshal(candidate, &card); err != nil {
		return err
	}
	canonical, err := xml.Marshal(card)
	if err != nil || !bytes.Equal(append(canonical, '\n'), candidate) || card.ID != a.source.ID || card.Href != a.source.Href || strings.TrimSpace(card.Heading) == "" || strings.TrimSpace(card.Message) == "" {
		return fmt.Errorf("card structure/technical identity changed")
	}
	return ValidateParsedGlossary(a.source.Heading+"\n"+a.source.Message, card.Heading+"\n"+card.Message, g)
}

func futureContentFixture(t *testing.T) (string, *Catalog, []string) {
	t.Helper()
	return futureContentFixtureSize(t, 3)
}

func futureContentFixtureSize(t *testing.T, count int) (string, *Catalog, []string) {
	t.Helper()
	root := t.TempDir()
	copyBundleAuthority(t, root, append(append([]string(nil), translationUnitGenerationAuthorityPaths...), "docs/TRANSLATION_QUALITY_REVIEW.md"))
	copyBundleAuthority(t, root, append(append([]string(nil), commentGateAuthorityPaths...), "go.mod", "go.sum", "internal/i18n/glossary.go", "internal/i18n/glossary_context.go", "internal/i18n/glossary_compatibility.go", "internal/i18n/glossary_semantics.go", "internal/i18n/translation_authorization.go"))
	copyTestLanguageAuditMachineAuthority(t, root)
	writeRetranslationTestGlossaryForLocale(t, root, "zh-CN")
	// The shared initialized-locale fixture already owns metadata and policy.
	glossaryPath := filepath.Join(root, "locales/zh-CN/glossary.yaml")
	glossaryBytes, err := os.ReadFile(glossaryPath)
	if err != nil {
		t.Fatal(err)
	}
	writeSnapshotSource(t, root, "locales/zh-CN/glossary.yaml", append([]byte("locale: zh-CN\n"), glossaryBytes...))
	recordSyntheticUnifiedReview(t, root, "synthetic-generation-review", "zh-CN")
	c := &Catalog{}
	d := TranslationPackageDescriptor{ID: "test-future-cards"}
	ids := []string{}
	for i := 1; i <= count; i++ {
		card := futureCard{ID: fmt.Sprintf("card-%d", i), Href: "https://example.invalid/identity", Heading: "A future card", Message: "Read this message."}
		source, err := xml.Marshal(card)
		if err != nil {
			t.Fatal(err)
		}
		source = append(source, '\n')
		id, p := fmt.Sprintf("future/card-%d", i), fmt.Sprintf("fixtures/future/card-%d.xml", i)
		writeSnapshotSource(t, root, p, source)
		a := futureCardAdapter{card}
		d.Units = append(d.Units, AdaptedTranslationUnit{Unit: TranslationUnit{ID: id, Kind: UnitKindPage, SourcePath: p, Source: source, SourceSHA256: sum(source)}, Adapter: a, StableIndex: i, Route: fmt.Sprintf("/fixture/cards/%d", i), WorkingSet: fmt.Sprintf("initial-%d", i), WorkingSetLimit: 1})
		ids = append(ids, id)
	}
	if err := c.RegisterTranslationPackage(d); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage(d.ID); err != nil {
		t.Fatal(err)
	}
	return root, c, ids
}

func futureImport(t *testing.T, root string, c *Catalog, bundle []byte, outputs map[string][]byte) {
	t.Helper()
	m, _, err := readGenerationBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, e := range m.ExpectedOutputs {
		if err := os.WriteFile(filepath.Join(dir, filepath.Base(e.BundlePath)), outputs[e.UnitID], 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ImportGenerationOutputDirectory(root, c, bundle, "chatgpt", FormalChatGPTGenerationModel, dir); err != nil {
		t.Fatal(err)
	}
}
func futureSnapshot(t *testing.T, root string, c *Catalog, id string) *QualityCheckSnapshotManifest {
	t.Helper()
	s, _, err := CreateQualityCheckCandidateSnapshot(root, c, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: id})
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func futureRecord(t *testing.T, root string, c *Catalog, snapshot, previous, rating, finding string, ids []string) {
	t.Helper()
	if _, err := RecordQualityCheckResults(root, c, QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: snapshot, PreviousSnapshotID: previous, UnitIDs: ids, Rating: rating, Finding: finding, ReviewerSession: "synthetic-independent-reviewer"}); err != nil {
		t.Fatal(err)
	}
}

func TestFutureContentSharedLifecycleConformance(t *testing.T) {
	root, c, ids := futureContentFixture(t)
	// Initial scheduling only: each card's policy is deliberately a separate set.
	for i, id := range ids {
		e, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, UnitIDs: []string{id}})
		if err != nil {
			t.Fatal(err)
		}
		b, m, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID})
		if err != nil {
			t.Fatal(err)
		}
		_, files, err := readGenerationBundle(b)
		if err != nil {
			t.Fatal(err)
		}
		good := files["formal/inputs/"+filepath.Base(m.ExpectedOutputs[0].BundlePath)]
		if i == 0 {
			// Recover a valid complete raw file before formal import, via shared API.
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, filepath.Base(m.ExpectedOutputs[0].BundlePath)), good, 0600); err != nil {
				t.Fatal(err)
			}
			r, err := RecoverGenerationOutputs(root, c, b, []string{dir}, filepath.Join(t.TempDir(), "recovered"))
			if err != nil || len(r.Recovered) != 1 || len(r.Missing) != 0 {
				t.Fatalf("recovery: %+v %v", r, err)
			}
		}
		raw := good
		if i == 1 {
			raw = bytes.Replace(good, []byte(futureCardTokens[0]), []byte("broken"), 1)
		}
		if i == 2 {
			raw = []byte(futureCardTokens[0] + futureCardTokens[1] + "Read this message." + futureCardTokens[2] + "\n")
		}
		futureImport(t, root, c, b, map[string][]byte{id: raw})
		r, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID})
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 && r.ValidationPassed != 1 {
			t.Fatal("initial success", r)
		}
		if i == 0 {
			// Exercise the actual historical-glossary evidence branch, not just
			// the adapter's validator. The old branch incorrectly required Page().
			formal := readRetranslationManifest(t, root, e.BatchID)
			batchDir := filepath.Join(root, e.BatchPath)
			unit, err := c.Unit(id)
			if err != nil {
				t.Fatal(err)
			}
			validation, err := readPromotionValidation(batchDir, e.BatchID, "zh-CN", formal.Units[0], r.Units[0])
			if err != nil {
				t.Fatal(err)
			}
			candidate, err := os.ReadFile(filepath.Join(batchDir, r.Units[0].CandidatePath))
			if err != nil {
				t.Fatal(err)
			}
			glossary, err := LoadGlossary(root, "zh-CN")
			if err != nil {
				t.Fatal(err)
			}
			glossaryBytes, err := os.ReadFile(filepath.Join(root, "locales/zh-CN/glossary.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			context := PrepareGlossaryCompatibility(root, "zh-CN", c)
			context.historicalSHA = sum(glossaryBytes)
			if attempt, err := validateUnifiedPromotionEvidence(root, batchDir, c, unit, formal.Units[0], validation, glossary, candidate, "zh-CN", context); err != nil || attempt != 1 {
				t.Fatal("historical adapted evidence dispatch", err)
			}
		}
		if i > 0 {
			if (i == 1 && r.RestoreFailed != 1) || (i == 2 && r.ValidationFailed != 1) {
				t.Fatal("wrong failure representation", r)
			}
			retry, m, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID, UnitID: id})
			if err != nil || m.Attempt != 2 {
				t.Fatal(err)
			}
			futureImport(t, root, c, retry, map[string][]byte{id: good})
			r, err = ProcessRetranslationRetry(root, c, RetranslationRetryOptions{Locale: "zh-CN", BatchID: e.BatchID, UnitID: id})
			if err != nil || r.ValidationPassed != 1 {
				t.Fatal("initial retry", r, err)
			}
		}
	}
	s := futureSnapshot(t, root, c, "future-initial")
	if s.UnitCount != 3 || s.Units[1].Attempt != 2 || s.Units[2].Attempt != 2 {
		t.Fatal("complete Snapshot attempt identities", s)
	}
	for i, id := range ids {
		b, m, err := ExportQualityCheckReviewerBundle(root, c, QualityCheckReviewerBundleOptions{Locale: "zh-CN", SnapshotID: s.SnapshotID, StartIndex: i + 1})
		if err != nil || m.UnitCount != 1 {
			t.Fatal("initial working set", err)
		}
		if _, err := VerifyCurrentQualityCheckReviewerBundle(root, c, b); err != nil {
			t.Fatal(err)
		}
		if i < 2 {
			futureRecord(t, root, c, s.SnapshotID, "", "B", "Fix this card's prose.", []string{id})
		} else {
			futureRecord(t, root, c, s.SnapshotID, "", "A", "", []string{id})
		}
		if _, err := VerifyCurrentQualityCheckReviewerBundle(root, c, b); err == nil {
			t.Fatal("JIT Bundle stayed current after record")
		}
	}
	if _, _, err := FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: s.SnapshotID}); err == nil {
		t.Fatal("non-A finalized")
	}
	// Exact revision spans two initial groups without adding package branches.
	previous := s.SnapshotID
	for round := 0; round < 2; round++ {
		selected := ids[:2]
		reopen := ""
		if round == 1 {
			selected = ids[1:2]
			reopen = "future-surface-reopen"
			finding := "Correct the future card message."
			p := filepath.Join(root, "data/locale-surface-reviews/zh-CN/future-surface.md")
			if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(selected[0]+": "+finding+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, _, err := RecordQualityCheckSurfaceReopen(root, c, QualityCheckSurfaceReopenOptions{Locale: "zh-CN", ReopenID: reopen, PreviousSnapshotID: previous, SurfaceReviewID: "future-surface", Finding: finding, UnitIDs: selected}); err != nil {
				t.Fatal(err)
			}
		}
		e, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, UnitIDs: selected, AllowReexport: true, PreviousSnapshotID: previous, SurfaceReopenID: reopen})
		if err != nil {
			t.Fatal(err)
		}
		b, m, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID})
		if err != nil {
			t.Fatal(err)
		}
		_, files, err := readGenerationBundle(b)
		if err != nil {
			t.Fatal(err)
		}
		good := map[string][]byte{}
		bad := map[string][]byte{}
		for _, out := range m.ExpectedOutputs {
			name := filepath.Base(out.BundlePath)
			baseline := files["formal/protected-candidates/"+name]
			if len(files["formal/candidates/"+name]) == 0 || len(baseline) == 0 || bytes.Equal(files["formal/candidates/"+name], baseline) {
				t.Fatal("predecessor representations confused")
			}
			good[out.UnitID] = bytes.Replace(baseline, []byte("Read this message."), []byte(fmt.Sprintf("Read the revised message %d.", round)), 1)
			if bytes.Equal(good[out.UnitID], baseline) {
				good[out.UnitID] = bytes.Replace(baseline, []byte("message 0"), []byte("message 1"), 1)
			}
			bad[out.UnitID] = bytes.Replace(good[out.UnitID], []byte(futureCardTokens[0]), []byte("broken"), 1)
		}
		futureImport(t, root, c, b, bad)
		r, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID})
		if err != nil || r.RestoreFailed != len(selected) {
			t.Fatal("revision failure", r, err)
		}
		bundles := [][]byte{}
		for _, id := range selected {
			retry, _, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID, UnitID: id})
			if err != nil {
				t.Fatal(err)
			}
			_, inputs, err := readGenerationBundle(retry)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(inputs["generation-context.json"], []byte("Semantic task: translation-unit-revision")) {
				t.Fatal("retry lost origin semantics")
			}
			for name, data := range files {
				if strings.HasPrefix(name, "formal/candidates/") || strings.HasPrefix(name, "formal/protected-candidates/") || strings.HasPrefix(name, "formal/predecessors/") {
					if !bytes.Equal(inputs[name], data) {
						t.Fatal("retry lost frozen predecessor", name)
					}
				}
			}
			bundles = append(bundles, retry)
		}
		handoff, _, err := ExportRetryHandoff(root, c, bundles)
		if err != nil {
			t.Fatal(err)
		}
		_, originals, err := ReadRetryHandoff(handoff)
		if err != nil {
			t.Fatal(err)
		}
		for i, retry := range originals {
			m, _, err := readGenerationBundle(retry)
			if err != nil {
				t.Fatal(err)
			}
			id := m.ExpectedOutputs[0].UnitID
			if _, err := VerifyCurrentTranslationUnitGenerationBundle(root, c, retry); err != nil {
				t.Fatal("sibling independence", err)
			}
			futureImport(t, root, c, retry, map[string][]byte{id: good[id]})
			r, err = ProcessRetranslationRetry(root, c, RetranslationRetryOptions{Locale: "zh-CN", BatchID: e.BatchID, UnitID: id})
			if err != nil || r.ValidationPassed != i+1 {
				t.Fatal("revision retry", r, err)
			}
			if _, err := VerifyCurrentRetryHandoff(root, c, handoff); err == nil {
				t.Fatal("aggregate remained current after member install")
			}
		}
		next := fmt.Sprintf("future-successor-%d", round)
		futureSnapshot(t, root, c, next)
		scope, err := BuildQualityCheckScope(root, c, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: next, PreviousSnapshotID: previous})
		if err != nil || scope.PendingCount != len(selected) || scope.ACount != 3-len(selected) {
			t.Fatalf("successor scope: %+v %v", scope, err)
		}
		for _, pending := range scope.Pending {
			if pending.RequiredAction != QualityCheckActionRequired {
				t.Fatal("retry bypassed re-QC")
			}
		}
		_, review, err := ExportQualityCheckReviewerBundle(root, c, QualityCheckReviewerBundleOptions{Locale: "zh-CN", SnapshotID: next, PreviousSnapshotID: previous})
		if err != nil || review.UnitCount != len(selected) {
			t.Fatal("re-QC inherited initial partition", err)
		}
		futureRecord(t, root, c, next, previous, "A", "", selected)
		if _, _, err := FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: next, PreviousSnapshotID: previous}); err != nil {
			t.Fatal(err)
		}
		outputs, err := FinalizedTranslationPackageCandidates(root, c, "zh-CN", next)
		if err != nil || len(outputs) != 3 {
			t.Fatal("shared package closure authorization", err)
		}
		previous = next
	}
	// Old finalized package cannot authorize newer selected candidates.
	if _, err := FinalizedTranslationPackageCandidates(root, c, "zh-CN", "future-successor-0"); err == nil {
		t.Fatal("stale package authorization accepted")
	}
	t.Log("XML future package: initial/recovery/two machine failure kinds/QC/exact revision/aggregate origin retry/re-QC/A-only/Surface revision retry/package authorization PASS")
}

func TestFutureContentStructureAndRegistrationBoundaries(t *testing.T) {
	root, c, ids := futureContentFixture(t)
	u, err := c.Unit(ids[0])
	if err != nil {
		t.Fatal(err)
	}
	a := u.Adapter.(futureCardAdapter)
	g, err := LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	if err := validateTranslationUnitWithGlossary(root, c, u, "zh-CN", u.Source, g); err != nil {
		t.Fatal("archived glossary dispatch forced XML through Present", err)
	}
	for _, bad := range [][]byte{bytes.Replace(u.Source, []byte("https://example.invalid/identity"), []byte("https://changed.invalid"), 1), bytes.Replace(u.Source, []byte("</card>"), []byte("<unknown/></card>"), 1)} {
		if err := a.Validate(bad, g); err == nil {
			t.Fatal("invalid structure accepted")
		}
	}
	input, _, _ := a.ProtectedInput(g)
	for _, bad := range []string{strings.Replace(input, futureCardTokens[0], "", 1), input + futureCardTokens[0], strings.Replace(input, futureCardTokens[1], futureCardTokens[2], 1)} {
		if _, err := a.Restore(bad); err == nil {
			t.Fatal("bad token set accepted")
		}
	}
	before := len(c.Adapted)
	if err := c.RegisterTranslationPackage(c.TranslationPackages[0]); err == nil || len(c.Adapted) != before {
		t.Fatal("duplicate registration mutation")
	}
	if err := c.UseTranslationPackage("unknown"); err == nil {
		t.Fatal("unknown package")
	}
}
