package sitecontent

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestConsumedReplacementCorpusTerminologyReviewContinuity(t *testing.T) {
	root, parent, consumption, _, _ := consumedReplacementFixture(t)
	proposal, input := replacementProposalFixture(t, root, parent, consumption)
	oldCorpus, err := i18n.LoadUnifiedGlossaryCorpus(root)
	if err != nil {
		t.Fatal(err)
	}
	glossary, err := readRegular(root, "locales/zh-CN/glossary.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyConsumedStructuredSourceReplacement(root, proposal, input.PredecessorIdentity, input.PriorPointer.SHA256); err != nil {
		t.Fatal(err)
	}
	if err := CheckUnifiedCorpus(root); err == nil {
		t.Fatal("transitional corpus accepted")
	}
	if _, err := CheckTerminologyAuthority(root); err == nil {
		t.Fatal("transitional terminology accepted")
	}
	raw, err := readRegular(root, i18n.GlossaryCorpusPath)
	if err != nil {
		t.Fatal(err)
	}
	next, err := AdoptCorpusProjection(root, digest(raw), true)
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckUnifiedCorpus(root); err != nil {
		t.Fatal(err)
	}
	summary, err := CheckTerminologyAuthority(root)
	if err != nil || summary.CorpusIdentity != next.Result.Identity {
		t.Fatal("terminology chose another projection", summary, err)
	}
	canonical, err := BuildUnifiedCorpus(root)
	if err != nil || !reflect.DeepEqual(*canonical, next.Result) {
		t.Fatal("canonical projection differs", err)
	}
	// This is the regression trigger: a full re-extraction loses the source
	// replacement namespace, while the shared current projection retains it.
	fresh, err := buildUnifiedCorpus(root, nil, nil)
	if err != nil || fresh.Identity == canonical.Identity {
		t.Fatal("fixture did not distinguish source epochs", err)
	}
	prior := map[string]i18n.LanguageContributor{}
	for _, c := range oldCorpus.Contributors {
		prior[c.ID] = c
	}
	for _, c := range canonical.Contributors {
		if c.Path != "data/site-shell/home.html" && !reflect.DeepEqual(c, prior[c.ID]) {
			t.Fatal("unaffected contributor changed", c.ID)
		}
	}
	if _, err := i18n.RequireUnifiedGlossaryReview(root, "zh-CN"); err == nil {
		t.Fatal("old Review became current")
	}
	if _, err := ReadHistoricalAdoptedCorpusSuccessor(root, consumption); err != nil {
		t.Fatal("historical basis", err)
	}
	bundle, manifest, err := i18n.ExportUnifiedGlossaryBundle(root, "zh-CN", "reviewer")
	if err != nil || manifest.CorpusSHA != canonical.Identity || len(manifest.ExpectedOutputs) != 0 {
		t.Fatal("full Reviewer transport", err)
	}
	if _, err := i18n.CheckUnifiedGlossaryBundle(root, bundle); err != nil {
		t.Fatal(err)
	}
	after, _ := readRegular(root, "locales/zh-CN/glossary.yaml")
	if !bytes.Equal(glossary, after) {
		t.Fatal("machine source transition edited glossary")
	}
	// Each mutation is read afresh; an installed result is never a prior-PASS
	// shortcut. Restore only this synthetic fixture dependency after each case.
	for label, name := range map[string]string{"receipt": CorpusSuccessorPath(next), "archive": "data/glossary-corpus-history/" + next.Predecessor.SHA256 + ".json", "body": input.Body.Path, "corpus": i18n.GlossaryCorpusPath} {
		t.Run(label, func(t *testing.T) {
			before, err := readRegular(root, name)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, name, before)
			writeFixture(t, root, name, append(bytes.Clone(before), 'x'))
			if err := CheckUnifiedCorpus(root); err == nil {
				t.Fatal("corpus accepted drift")
			}
			if _, err := CheckTerminologyAuthority(root); err == nil {
				t.Fatal("terminology accepted drift")
			}
		})
	}
}

func TestFrozenCorpusTerminologyBinding(t *testing.T) {
	root := siteFixture(t, "current-authorization-fresh-fixture")
	authority, snapshot := terminologyTestAuthority(t, terminologyTestFiles())
	manifest, _ := Encode(authority)
	writeFixture(t, root, TerminologyManifestPath, manifest)
	writeFixture(t, root, TerminologySnapshotPath, snapshot)
	corpus, err := buildUnifiedCorpus(root, &authority, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	// This helper checks a frozen epoch's authority binding, not a second
	// current-parser interpretation. The enclosing reader reconstructs its proof.
	for i := range corpus.Contributors {
		c := &corpus.Contributors[i]
		if c.Purpose == i18n.ContributorPurposeTerminologyDiscovery {
			c.Contexts[0].Text = "Synthetic historical projection epoch."
			c.Contexts = append(c.Contexts, i18n.LanguageContext{ID: "zz-frozen-epoch", Text: "A second frozen epoch context.", Protected: []string{}})
			c.SourceSHA = i18n.ContributorIdentity(*c)
		}
	}
	corpus.Identity = i18n.CorpusIdentity(*corpus)
	if err := checkCorpusTerminologyBinding(root, corpus, &authority, snapshot); err != nil {
		t.Fatal(err)
	}
	summary, err := terminologySummary(&authority, snapshot, corpus, true)
	wantContexts := 0
	for _, c := range corpus.Contributors {
		if c.Purpose == i18n.ContributorPurposeTerminologyDiscovery {
			wantContexts += len(c.Contexts)
		}
	}
	if err != nil || summary.Contexts != wantContexts || summary.CorpusIdentity != corpus.Identity {
		t.Fatal("summary used a different projection epoch", summary, err)
	}
	for _, kind := range []string{"parser", "purpose", "missing", "metadata-ref", "snapshot-ref"} {
		t.Run(kind, func(t *testing.T) {
			b, _ := Encode(corpus)
			var changed i18n.GlossarySourceCorpus
			if err := StrictJSON(b, &changed); err != nil {
				t.Fatal(err)
			}
			for i := range changed.Contributors {
				c := &changed.Contributors[i]
				if c.Purpose != i18n.ContributorPurposeTerminologyDiscovery {
					continue
				}
				switch kind {
				case "parser":
					c.Parser = "wrong-parser/v1"
				case "purpose":
					c.Purpose = i18n.ContributorPurposeActiveContent
				case "missing":
					changed.Contributors = append(changed.Contributors[:i], changed.Contributors[i+1:]...)
				}
				if kind != "missing" {
					c.SourceSHA = i18n.ContributorIdentity(*c)
				}
				break
			}
			for i := range changed.Sources {
				if kind == "metadata-ref" && changed.Sources[i].Path == TerminologyManifestPath || kind == "snapshot-ref" && changed.Sources[i].Path == TerminologySnapshotPath {
					changed.Sources[i].SHA256 = strings.Repeat("0", 64)
				}
			}
			changed.Identity = i18n.CorpusIdentity(changed)
			if err := checkCorpusTerminologyBinding(root, &changed, &authority, snapshot); err == nil {
				t.Fatal("wrong terminology binding accepted", kind)
			}
		})
	}
	writeFixture(t, root, TerminologySnapshotPath, append(bytes.Clone(snapshot), 'x'))
	if err := checkCorpusTerminologyBinding(root, corpus, &authority, snapshot); err == nil {
		t.Fatal("same-path snapshot mutation accepted")
	}
}
