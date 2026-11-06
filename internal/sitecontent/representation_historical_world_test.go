package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestRepresentationSemanticHistoricalWorld(t *testing.T) {
	root, c, o := semanticReviewFixture(t)
	b, m, err := i18n.ExportRepresentationReviewBundle(root, c, o)
	if err != nil {
		t.Fatal(err)
	}
	d := i18n.RepresentationReviewDecision{ReviewID: o.ReviewID, BundleIdentity: m.Identity, BundleSHA256: digest(b), Locale: o.Reconciliation.Locale, Package: o.Reconciliation.Package, OldUnitID: m.Replacement.OldUnitID, CurrentUnitID: o.UnitID, PreviousSnapshotID: o.Reconciliation.PreviousSnapshotID, FormalImpactSHA256: o.Reconciliation.Impact.SHA256, ReviewerSession: "synthetic-independent-semantic-reviewer", Provider: "chatgpt", Model: i18n.FormalChatGPTGenerationModel, Decision: "APPROVE_RECONCILIATION", Finding: "Synthetic correspondence fixture only.", GenerationIndependent: true}
	decision, _ := json.Marshal(d)
	r, _, err := i18n.RecordRepresentationReview(root, c, b, decision)
	if err != nil {
		t.Fatal(err)
	}
	options := o.Reconciliation
	options.LanguageReviewIDs, options.Apply = []string{o.ReviewID}, true
	plan, err := i18n.ReconcileTranslationUnitRepresentations(root, c, options)
	if err != nil || !plan.Applied {
		t.Fatal("semantic machine fixture", plan, err)
	}
	oldID := m.Replacement.OldUnitID
	before, refsBefore, err := i18n.NewRepresentationSuccessorVerifier(root, c, "zh-CN").VerifyHistorical(oldID, o.UnitID)
	if err != nil {
		t.Fatal(err)
	}
	glossary, err := readRegular(root, "locales/zh-CN/glossary.yaml")
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "locales/zh-CN/glossary.yaml", append(bytes.Clone(glossary), []byte("\n# later current glossary world\n")...))
	after, refsAfter, err := i18n.NewRepresentationSuccessorVerifier(root, c, "zh-CN").VerifyHistorical(oldID, o.UnitID)
	left, _ := json.Marshal(struct {
		Mapping any
		Refs    any
	}{before, refsBefore})
	right, _ := json.Marshal(struct {
		Mapping any
		Refs    any
	}{after, refsAfter})
	if err != nil || !bytes.Equal(left, right) {
		t.Fatal("historical semantic bundle lost world or changed machine result", err)
	}
	if _, err := i18n.CheckRepresentationReviewBundle(root, c, b); err == nil {
		t.Fatal("ordinary current bundle check inherited historical world")
	}
	if _, _, err := i18n.ExportRepresentationReviewBundle(root, c, o); err == nil {
		t.Fatal("ordinary current export inherited historical world")
	}
	machine := "data/translation-unit-reconciliations/zh-CN/" + options.ReconciliationID + "/receipt.json"
	archive := "data/glossary-history/zh-CN/" + plan.Receipt.GlossarySHA256 + ".yaml"
	for name, path := range map[string]string{"semantic-receipt": "data/representation-reconciliation-reviews/zh-CN/" + o.ReviewID + "/receipt.json", "semantic-decision": r.Evidence.Path, "machine-receipt": machine, "archived-glossary": archive, "source-drift": m.Replacement.SourcePath} {
		t.Run(name, func(t *testing.T) {
			data, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			abs := filepath.Join(root, path)
			info, _ := os.Stat(abs)
			if err := os.Chmod(abs, 0644); err != nil {
				t.Fatal(err)
			}
			defer func() { writeFixture(t, root, path, data); os.Chmod(abs, info.Mode()) }()
			writeFixture(t, root, path, append(bytes.Clone(data), ' '))
			if name == "source-drift" {
				// This reader receives an explicit Catalog source projection;
				// a source change must enter that authority, not remain hidden
				// behind the earlier frozen in-memory source snapshot.
				current, err := c.Unit(o.UnitID)
				if err != nil {
					t.Fatal(err)
				}
				before := bytes.Clone(current.Source)
				current.Source[0] ^= 1
				defer copy(current.Source, before)
			}
			if _, _, err := i18n.NewRepresentationSuccessorVerifier(root, c, "zh-CN").VerifyHistorical(oldID, o.UnitID); err == nil {
				t.Fatal("fresh historical invocation accepted tamper")
			}
		})
	}
}
