package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

const representationReviewSchema = "go-learning/representation-semantic-review/v1"

// This is a correspondence question, not a requested translation or QC rating.
// Coordinates are source bytes within the reconstructed formal Page unit.
type RepresentationSemanticQuestion struct {
	SourceStart           int             `json:"source_start"`
	SourceEnd             int             `json:"source_end"`
	SourceText            string          `json:"source_text"`
	ReviewedTarget        string          `json:"reviewed_target"`
	LeftAnchor            string          `json:"left_anchor"`
	RightAnchor           string          `json:"right_anchor"`
	ChangedAtoms          json.RawMessage `json:"changed_atoms"`
	RepresentationChanges json.RawMessage `json:"representation_changes"`
	OldRepresentation     json.RawMessage `json:"old_representation"`
	CurrentRepresentation json.RawMessage `json:"current_representation"`
}

// An approved correspondence is an explicit exception scoped to one frozen
// prose interval. It never changes the canonical Generation representation.
type RepresentationSemanticAdapter interface {
	ReconciliationQuestion(TranslationUnitAdapter, []byte) (*RepresentationSemanticQuestion, error)
	ReviewedReconciliation(TranslationUnitAdapter, []byte, RepresentationSemanticQuestion) ([]byte, []byte, error)
}

type RepresentationReviewOptions struct {
	Reconciliation RepresentationReconciliationOptions `json:"reconciliation"`
	UnitID         string                              `json:"unit_id"`
	ReviewID       string                              `json:"review_id"`
}

type RepresentationReviewBundle struct {
	Schema            string                         `json:"schema"`
	Options           RepresentationReviewOptions    `json:"options"`
	Replacement       TranslationUnitReplacement     `json:"replacement"`
	Predecessor       QualityCheckSnapshotUnit       `json:"predecessor"`
	PredecessorRating string                         `json:"predecessor_rating"`
	Question          RepresentationSemanticQuestion `json:"question"`
	Files             []TransportBundleFile          `json:"files"`
	Identity          string                         `json:"identity"`
}

const representationReviewInstructions = `# 独立 representation reconciliation semantic authority

本 Bundle 自包含。仅使用其中输入，不扫描 repository。唯一问题：question.json 的冻结 reviewed_target，能否作为同一 source_text 及整篇 source 实际上下文的可信语义对应，保持旧 reviewed A translation 的全部字节不变？

必须区分 Go identifier/API 的 exact identity 与普通自然语言概念。不要仅因中文缺少 literal 就自动批准或拒绝；结合整个 source、旧 reviewed candidate、glossary、old/current protection 和精确 changed atoms 判断。所有机器证据成立不代替本语义判断。

只作独立语言 authority 决定：APPROVE_RECONCILIATION 或 REQUIRE_LANGUAGE_REVISION，附 exact finding/reason。不是新 QC，不给 A/B/C/D，不生成 replacement，不修改 candidate，不建议猜选 target 位置。批准只授权 question 的整个冻结 prose 区间，不能豁免 URL、code、binding 或其它 machine identity。拒绝则保持 representation-only blocked，后续走正式 finding-anchored exact revision。

从 manifest.json 复制 identity 至 bundle_identity，并从 Local terminal handoff 复制 ZIP SHA 至 bundle_sha256。在 decision-template.json 的其它已填 identity/provenance 字段中保留 frozen 值；填写 generation_independent=true（真实独立性声明）、真实 reviewer_session、provider=chatgpt、model=gpt-5.6-sol-high、decision、finding。只输出这一 JSON，不增加 replacement、rating 等字段。Reviewer session 必须与该 locale 全部 Generation session 独立。Local terminal 用 current-check 和 record 记录 immutable evidence；本 Bundle 导出不记录决定，不改变 formal state。
`

// The exact response bytes are the review evidence. Unknown fields (including
// replacement/rating) and trailing data are rejected by the shared strict reader.
type RepresentationReviewDecision struct {
	ReviewID              string `json:"review_id"`
	BundleIdentity        string `json:"bundle_identity"`
	BundleSHA256          string `json:"bundle_sha256"`
	Locale                string `json:"locale"`
	Package               string `json:"package"`
	OldUnitID             string `json:"old_unit_id"`
	CurrentUnitID         string `json:"current_unit_id"`
	PreviousSnapshotID    string `json:"previous_snapshot_id"`
	FormalImpactSHA256    string `json:"formal_impact_sha256"`
	GenerationIndependent bool   `json:"generation_independent"`
	ReviewerSession       string `json:"reviewer_session"`
	Provider              string `json:"provider"`
	Model                 string `json:"model"`
	Decision              string `json:"decision"`
	Finding               string `json:"finding"`
}

type RepresentationReviewReceipt struct {
	Schema      string                          `json:"schema"`
	Decision    RepresentationReviewDecision    `json:"decision"`
	Bundle      GlossaryArchiveReference        `json:"bundle"`
	Evidence    GlossaryArchiveReference        `json:"evidence"`
	Predecessor QualityCheckSnapshotUnit        `json:"predecessor"`
	QCInputs    []QualityCheckFinalizationInput `json:"qc_inputs"`
	Identity    string                          `json:"identity"`
}

func representationReviewBase(locale, id string) string {
	return filepath.ToSlash(filepath.Join("data/representation-reconciliation-reviews", locale, id))
}

func ExportRepresentationReviewBundle(root string, c *Catalog, o RepresentationReviewOptions) ([]byte, *RepresentationReviewBundle, error) {
	return exportRepresentationReviewBundleInWorld(root, c, o, nil)
}

// Only an immutable semantic-authority reader supplies its receipt-bound world.
// Public export/check continue to require today's full current preparation.
func prepareRepresentationReviewMachine(root string, c *Catalog, o RepresentationReconciliationOptions, world *machineReconciliationContext) (*machineReconciliationContext, error) {
	if world == nil {
		return prepareMachineReconciliation(root, c, o)
	}
	want := world.receipt.Options
	want.LanguageReviewIDs = nil
	if !world.historical || !reflect.DeepEqual(want, o) || world.currentGlossarySHA != world.receipt.GlossarySHA256 {
		return nil, fmt.Errorf("semantic review historical machine world mismatch")
	}
	return prepareMachineReconciliationInWorld(root, c, o, world.receipt.GlossarySHA256, world.compatibility)
}

func exportRepresentationReviewBundleInWorld(root string, c *Catalog, o RepresentationReviewOptions, world *machineReconciliationContext) ([]byte, *RepresentationReviewBundle, error) {
	if !reviewIDPattern.MatchString(o.ReviewID) || o.Reconciliation.Apply || len(o.Reconciliation.LanguageReviewIDs) != 0 {
		return nil, nil, fmt.Errorf("explicit review ID and read-only authority-free inputs required")
	}
	ctx, err := prepareRepresentationReviewMachine(root, c, o.Reconciliation, world)
	if err != nil {
		return nil, nil, err
	}
	trust, err := ctx.trustedUnit(root, o.UnitID)
	if err != nil {
		return nil, nil, err
	}
	if !trust.wasUnknown {
		return nil, nil, fmt.Errorf("semantic review requires an unresolved formal impact question")
	}
	current, err := c.Unit(o.UnitID)
	if err != nil {
		return nil, nil, err
	}
	a, ok := current.Adapter.(RepresentationSemanticAdapter)
	if !ok {
		return nil, nil, fmt.Errorf("adapter lacks scoped semantic correspondence proof")
	}
	if machine, ok := current.Adapter.(RepresentationMachineAdapter); ok {
		if _, err := machine.ReconcileRepresentation(trust.evidence.unit.Adapter, trust.raw); err == nil {
			return nil, nil, fmt.Errorf("deterministic machine correspondence needs no language decision")
		}
	}
	q, err := a.ReconciliationQuestion(trust.evidence.unit.Adapter, trust.raw)
	if err != nil {
		return nil, nil, err
	}
	o.Reconciliation = ctx.receipt.Options
	m := &RepresentationReviewBundle{Schema: representationReviewSchema, Options: o, Replacement: trust.replacement, Predecessor: trust.old, PredecessorRating: "A", Question: *q, Files: []TransportBundleFile{}}
	entries := []TransportBundleEntry{}
	add := func(name, path string, b []byte) {
		m.Files = append(m.Files, NewTransportBundleFile(name, path, b))
		entries = append(entries, TransportBundleEntry{Path: name, Data: b})
	}
	paths := []string{ctx.receipt.Snapshot.Path, o.Reconciliation.Impact.Path, o.Reconciliation.HistoricalDocuments.Path, glossaryReviewGlossaryPath(o.Reconciliation.Locale), trust.rawRef.Path, trust.old.CandidatePath, trust.old.ValidationPath}
	base := filepath.ToSlash(filepath.Join("data/retranslation-runs", o.Reconciliation.Locale, trust.old.SelectedBatchID))
	paths = append(paths, base+"/manifest.json", base+"/result.json")
	// Preserve actual imported provider/model/input/output provenance where it
	// exists; legacy accepted batches without it are not given invented fields.
	imports := []string{base + "/generation-imports/initial-manifest.json"}
	if trust.old.Attempt > 1 {
		gm := GenerationBundleManifest{TaskKind: "translation-unit-retry", Attempt: trust.old.Attempt, ExpectedOutputs: []GenerationBundleExpectedOutput{{BundlePath: trust.evidence.record.InputPath}}}
		imports = append(imports, base+"/generation-imports/"+filepath.Base(generationImportReceiptPath(trust.evidence.batchDir, gm)))
	}
	for _, path := range imports {
		if _, err := os.Lstat(filepath.Join(root, path)); err == nil {
			paths = append(paths, path)
		} else if !os.IsNotExist(err) {
			return nil, nil, err
		}
	}
	if origin := trust.evidence.record.GenerationOrigin; origin != nil {
		paths = append(paths, origin.Bundle.Path, origin.Result.Path)
	}
	for _, ref := range ctx.receipt.QCInputs {
		paths = append(paths, ref.Path)
	}
	seen := map[string]bool{}
	for _, path := range paths {
		if seen[path] {
			continue
		}
		seen[path] = true
		b, err := readSnapshotReferencedFile(root, path)
		if ctx.historical && path == glossaryReviewGlossaryPath(o.Reconciliation.Locale) {
			// Preserve the original bundle member/path and reconstruct its exact
			// bytes from the hash-bound glossary archive, never today's glossary.
			b, err = readArchivedGlossary(root, o.Reconciliation.Locale, ctx.currentGlossarySHA)
		}
		if err != nil {
			return nil, nil, err
		}
		add("inputs/"+path, path, b)
	}
	add("predecessor-source.txt", trust.replacement.SourcePath, trust.evidence.unit.Source)
	add("current-source.txt", trust.replacement.SourcePath, current.Source)
	add("old-representation.json", "", q.OldRepresentation)
	add("current-representation.json", "", q.CurrentRepresentation)
	add("changed-representation.json", "", q.ChangedAtoms)
	add("all-representation-changes.json", "", q.RepresentationChanges)
	add("formal-impact-record.json", "", mustJSON(ctx.impact[o.UnitID]))
	add("question.json", "", mustJSON(q))
	add("machine-evidence.json", "", mustJSON(struct {
		Replacement        TranslationUnitReplacement
		Predecessor        QualityCheckSnapshotUnit
		QCInputs           []QualityCheckFinalizationInput
		OriginalRaw        GlossaryArchiveReference
		GlossarySHA256     string
		OldCandidateSHA256 string
	}{trust.replacement, trust.old, ctx.receipt.QCInputs, trust.rawRef, ctx.currentGlossarySHA, sum(trust.evidence.candidate)}))
	add("REVIEWER.md", "", []byte(representationReviewInstructions))
	// Bundle SHA is external framing (a ZIP cannot contain its own SHA). The
	// Reviewer copies it from the verified Local-terminal handoff into evidence.
	d := RepresentationReviewDecision{ReviewID: o.ReviewID, Locale: o.Reconciliation.Locale, Package: o.Reconciliation.Package, OldUnitID: trust.replacement.OldUnitID, CurrentUnitID: o.UnitID, PreviousSnapshotID: o.Reconciliation.PreviousSnapshotID, FormalImpactSHA256: o.Reconciliation.Impact.SHA256, Provider: "chatgpt", Model: FormalChatGPTGenerationModel}
	add("decision-template.json", "", mustJSON(d))
	m.Identity = sum(mustJSON(m))
	b, err := WriteDeterministicTransportBundle(append(mustJSON(m), '\n'), entries)
	return b, m, err
}

func CheckRepresentationReviewBundle(root string, c *Catalog, b []byte) (*RepresentationReviewBundle, error) {
	return checkRepresentationReviewBundleInWorld(root, c, b, nil)
}

func checkRepresentationReviewBundleInWorld(root string, c *Catalog, b []byte, world *machineReconciliationContext) (*RepresentationReviewBundle, error) {
	files, err := ReadTransportBundle(b, 512, 64<<20)
	if err != nil {
		return nil, err
	}
	var m RepresentationReviewBundle
	if err = contentidentity.StrictJSON(files["manifest.json"], &m); err != nil {
		return nil, err
	}
	if err = ValidateTransportBundleInventory(files, m.Files, true); err != nil {
		return nil, err
	}
	want, _, err := exportRepresentationReviewBundleInWorld(root, c, m.Options, world)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(want, b) {
		return nil, fmt.Errorf("representation semantic Reviewer Bundle stale/noncanonical")
	}
	return &m, nil
}

func validateRepresentationDecision(root string, m *RepresentationReviewBundle, b, e []byte) (*RepresentationReviewDecision, error) {
	var d RepresentationReviewDecision
	if err := contentidentity.StrictJSON(e, &d); err != nil {
		return nil, err
	}
	o := m.Options.Reconciliation
	if d.ReviewID != m.Options.ReviewID || d.BundleIdentity != m.Identity || d.BundleSHA256 != sum(b) || d.Locale != o.Locale || d.Package != o.Package || d.OldUnitID != m.Replacement.OldUnitID || d.CurrentUnitID != m.Replacement.CurrentUnitID || d.PreviousSnapshotID != o.PreviousSnapshotID || d.FormalImpactSHA256 != o.Impact.SHA256 {
		return nil, fmt.Errorf("semantic decision exact authority binding mismatch")
	}
	if strings.TrimSpace(d.ReviewerSession) == "" || !d.GenerationIndependent || d.Provider != "chatgpt" || d.Model != FormalChatGPTGenerationModel || strings.TrimSpace(d.Finding) == "" {
		return nil, fmt.Errorf("independent Reviewer provenance and exact reason required")
	}
	if d.Decision != "APPROVE_RECONCILIATION" && d.Decision != "REQUIRE_LANGUAGE_REVISION" {
		return nil, fmt.Errorf("semantic decision must not be a QC rating or replacement")
	}
	if err := CheckUnifiedLanguageRole(root, o.Locale, d.ReviewerSession, "reviewer"); err != nil {
		return nil, err
	}
	// Legacy accepted batches do not always record a session. Keep their real
	// provenance; independence is explicitly attested and checked against the
	// existing locale role authority, never invent a historical session.
	return &d, nil
}

func reviewReceiptIdentity(r RepresentationReviewReceipt) string {
	r.Identity = ""
	return sum(mustJSON(r))
}

func RecordRepresentationReview(root string, c *Catalog, b, e []byte) (*RepresentationReviewReceipt, string, error) {
	m, err := CheckRepresentationReviewBundle(root, c, b)
	if err != nil {
		return nil, "", err
	}
	d, err := validateRepresentationDecision(root, m, b, e)
	if err != nil {
		return nil, "", err
	}
	base := representationReviewBase(d.Locale, d.ReviewID)
	// An existing receipt or incomplete transaction is never overwritten/replayed.
	if _, err = os.Lstat(filepath.Join(root, base)); !os.IsNotExist(err) {
		return nil, "", fmt.Errorf("semantic review receipt/transaction already exists")
	}
	ctx, err := prepareMachineReconciliation(root, c, m.Options.Reconciliation)
	if err != nil {
		return nil, "", err
	}
	r := &RepresentationReviewReceipt{Schema: representationReviewSchema, Decision: *d, Bundle: GlossaryArchiveReference{Path: base + "/bundle.zip", SHA256: sum(b)}, Evidence: GlossaryArchiveReference{Path: base + "/decision.json", SHA256: sum(e)}, Predecessor: m.Predecessor, QCInputs: ctx.receipt.QCInputs}
	r.Identity = reviewReceiptIdentity(*r)
	// Atomic immutable directory publication follows the existing machine receipt
	// pattern. Staging failures remain visible, rather than silently overwriting.
	abs := filepath.Join(root, base)
	if err = os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return nil, "", err
	}
	staging := abs + ".staging"
	if err = os.Mkdir(staging, 0755); err != nil {
		return nil, "", err
	}
	for name, data := range map[string][]byte{"bundle.zip": b, "decision.json": e, "receipt.json": append(mustJSON(r), '\n')} {
		if err = os.WriteFile(filepath.Join(staging, name), data, 0444); err != nil {
			return nil, "", err
		}
	}
	if err = os.Rename(staging, abs); err != nil {
		return nil, "", err
	}
	return r, base + "/receipt.json", nil
}

func (m *machineReconciliationContext) semanticAuthority(root string, c *Catalog, id string) (*RepresentationReviewBundle, *GlossaryArchiveReference, error) {
	var found *RepresentationReviewBundle
	var reference *GlossaryArchiveReference
	for _, reviewID := range m.receipt.Options.LanguageReviewIDs {
		if !reviewIDPattern.MatchString(reviewID) {
			return nil, nil, fmt.Errorf("invalid language authority review ID")
		}
		path := representationReviewBase(m.receipt.Options.Locale, reviewID) + "/receipt.json"
		data, err := readSnapshotReferencedFile(root, path)
		if err != nil {
			return nil, nil, err
		}
		var r RepresentationReviewReceipt
		if err = contentidentity.StrictJSON(data, &r); err != nil {
			return nil, nil, err
		}
		base := representationReviewBase(m.receipt.Options.Locale, reviewID)
		if r.Schema != representationReviewSchema || r.Identity != reviewReceiptIdentity(r) || r.Decision.ReviewID != reviewID || r.Bundle.Path != base+"/bundle.zip" || r.Evidence.Path != base+"/decision.json" {
			return nil, nil, fmt.Errorf("semantic receipt identity mismatch")
		}
		b, err := readSnapshotReferencedFile(root, r.Bundle.Path)
		if err != nil || sum(b) != r.Bundle.SHA256 {
			return nil, nil, fmt.Errorf("semantic bundle archive hash mismatch")
		}
		e, err := readSnapshotReferencedFile(root, r.Evidence.Path)
		if err != nil || sum(e) != r.Evidence.SHA256 {
			return nil, nil, fmt.Errorf("semantic decision evidence hash mismatch")
		}
		var world *machineReconciliationContext
		if m.historical {
			world = m
		}
		bundle, err := checkRepresentationReviewBundleInWorld(root, c, b, world)
		if err != nil {
			return nil, nil, err
		}
		d, err := validateRepresentationDecision(root, bundle, b, e)
		if err != nil {
			return nil, nil, err
		}
		if !reflect.DeepEqual(*d, r.Decision) || !reflect.DeepEqual(r.Predecessor, bundle.Predecessor) || !reflect.DeepEqual(r.QCInputs, m.receipt.QCInputs) {
			return nil, nil, fmt.Errorf("semantic receipt predecessor/QC lineage mismatch")
		}
		want, got := m.receipt.Options, bundle.Options.Reconciliation
		want.LanguageReviewIDs = nil
		if !reflect.DeepEqual(want, got) {
			return nil, nil, fmt.Errorf("semantic authority reconciliation context mismatch")
		}
		if d.CurrentUnitID != id {
			continue
		}
		if d.Decision != "APPROVE_RECONCILIATION" {
			return nil, nil, fmt.Errorf("formal exact language revision required: %s", d.Finding)
		}
		if found != nil {
			return nil, nil, fmt.Errorf("ambiguous semantic review authority")
		}
		found = bundle
		reference = &GlossaryArchiveReference{Path: path, SHA256: sum(data)}
	}
	if found == nil {
		return nil, nil, fmt.Errorf("no exact independent semantic authority receipt")
	}
	return found, reference, nil
}
