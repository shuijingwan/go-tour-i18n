package sitecontent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestRepresentationMaterializationReadonlyPlan(t *testing.T) {
	if os.Getenv("GO_TOUR_REPRESENTATION_MACHINE_PROBE") != "1" {
		t.Skip("explicit readonly failure diagnosis")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	immutable := map[string][]byte{}
	paths := []string{"docs/PROTECTED_REPRESENTATION_FORMAL_IMPACT.json", "data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-014/manifest.json", "data/retranslation-runs/zh-CN/chatgpt-zh-CN-047/manifest.json", "data/retranslation-runs/zh-CN/chatgpt-zh-CN-047/result.json"}
	reviewID := os.Getenv("GO_TOUR_REPRESENTATION_LANGUAGE_REVIEW_ID")
	if reviewID != "" {
		paths = append(paths, "data/representation-reconciliation-reviews/zh-CN/"+reviewID+"/receipt.json")
	}
	for _, path := range paths {
		b, err := readRegular(root, path)
		if err != nil {
			t.Fatal(err)
		}
		immutable[path] = b
	}
	defer func() {
		for path, before := range immutable {
			after, err := readRegular(root, path)
			if err != nil || string(before) != string(after) {
				t.Error("formal artifact mutated", path, err)
			}
		}
	}()
	for _, path := range []string{"data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-015", "data/translation-unit-reconciliations/zh-CN/learn-docs-qc014-machine-successors"} {
		if _, err := os.Lstat(filepath.Join(root, path)); !os.IsNotExist(err) {
			t.Fatal("unexpected formal artifact", path, err)
		}
	}
	c, err := i18n.BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err = AttachRegisteredTranslationPackages(root, c); err != nil {
		t.Fatal(err)
	}
	if err = c.UseTranslationPackage("learn-docs-v1"); err != nil {
		t.Fatal(err)
	}
	const historical = "docs/protected-representation-evidence/qc014-historical-representations.json"
	b, err := readRegular(root, historical)
	if err != nil {
		t.Fatal(err)
	}
	o := i18n.RepresentationReconciliationOptions{Locale: "zh-CN", Package: "learn-docs-v1", ReconciliationID: "learn-docs-qc014-machine-successors", PreviousSnapshotID: "learn-docs-v1-qc-014", SnapshotSHA256: "234a37330a348590ab3b46e30d720ad42718dc798cb3002e212b86ce47acb619", Impact: i18n.GlossaryArchiveReference{Path: "docs/PROTECTED_REPRESENTATION_FORMAL_IMPACT.json", SHA256: "8fb332e4ba2b8be55de91c057a3f174ac4bb922536c90e9d4e927e21e6b74e0e"}, HistoricalDocuments: i18n.GlossaryArchiveReference{Path: historical, SHA256: digest(b)}}
	if reviewID != "" {
		o.LanguageReviewIDs = []string{reviewID}
	}
	started := time.Now()
	plan, err := i18n.ReconcileTranslationUnitRepresentations(root, c, o)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ready=%d excluded=%d blocked=%d elapsed=%s", plan.Ready, plan.Excluded, plan.Blocked, time.Since(started))
	for _, r := range plan.Rows {
		if r.Status == "blocked" || r.WasUnknown {
			t.Logf("%s: %s %s", r.Route, r.Status, r.Reason)
		}
	}
	if path := os.Getenv("GO_TOUR_REPRESENTATION_PLAN_OUTPUT"); path != "" {
		b, err := json.MarshalIndent(map[string]any{"report_type": "engineering-read-only-plan", "formal_artifact_mutation": "none", "plan": plan}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, append(b, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	expectedReady, expectedBlocked := 41, 2
	if reviewID != "" {
		expectedReady, expectedBlocked = 42, 1
	}
	if len(plan.Rows) != 44 || plan.Excluded != 1 || plan.Ready != expectedReady || plan.Blocked != expectedBlocked || plan.Applied {
		t.Fatal("unexpected package accounting")
	}
	for _, r := range plan.Rows {
		if r.UnitID == "8eafdd7a23781d41ad01de6a5b46d597b781d0ef9c085dcba81918b0f2550991" {
			t.Fatal("glossary stale entered representation scope")
		}
		if r.UnitID == "8e2b3ec2b00595cb7e8ea8587ea6002107a0e616c18f49fc46562702af10f1c3" && r.Status != "excluded-current-passed" {
			t.Fatal("047 revision not excluded")
		}
	}
	if reviewID != "" {
		jsonFound, dependencyFound := false, false
		for _, u := range plan.Receipt.Units {
			switch u.Replacement.CurrentUnitID {
			case "b49d8e873a0f66f6803077eb534583a155d2bd209a032d7c00fba8fea856da5c":
				jsonFound = u.SemanticAuthority != nil && u.SemanticAuthority.SHA256 == "8b952e7d9c80f5283239f4910fa47f0f31f56f91f5f4ff47743a297da4e5d5c9"
			case "82a1f9b16b20d60a538c1873daf9afb5c446aded1d2bff16e3b5650ed6bafb1e":
				dependencyFound = u.UnknownResolvedMachine
			}
		}
		if !jsonFound || !dependencyFound {
			t.Fatal("existing independent authorities lost", jsonFound, dependencyFound)
		}
		for _, r := range plan.Rows {
			if r.Status == "blocked" && !strings.Contains(r.Reason, "representation-unbindable") {
				t.Fatal("misleading semantic-receipt blocker", r)
			}
		}
	}
}

func TestRepresentationUnknownReadonlyProbe(t *testing.T) {
	if os.Getenv("GO_TOUR_REPRESENTATION_MACHINE_PROBE") != "1" {
		t.Skip("explicit readonly failure diagnosis")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	c, e := i18n.BuildCatalog(root)
	if e != nil {
		t.Fatal(e)
	}
	if e = AttachRegisteredTranslationPackages(root, c); e != nil {
		t.Fatal(e)
	}
	if e = c.UseTranslationPackage("learn-docs-v1"); e != nil {
		t.Fatal(e)
	}
	const frozenPath = "docs/protected-representation-evidence/qc014-historical-representations.json"
	b, e := readRegular(root, frozenPath)
	if e != nil {
		t.Fatal(e)
	}
	frozenSHA := digest(b)
	r, e := resolvePageReplacements(root, c, i18n.GlossaryArchiveReference{Path: frozenPath, SHA256: frozenSHA})
	if e != nil {
		t.Fatal(e)
	}
	b, e = readRegular(root, "data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-014/manifest.json")
	if e != nil {
		t.Fatal(e)
	}
	var s i18n.QualityCheckSnapshotManifest
	if e = json.Unmarshal(b, &s); e != nil {
		t.Fatal(e)
	}
	for _, u := range s.Units {
		if u.SourcePath != "_content/doc/tutorial/json.md" && u.SourcePath != "_content/doc/security/vuln/database.md" {
			continue
		}
		v, e := readRegular(root, u.ValidationPath)
		if e != nil {
			t.Fatal(e)
		}
		var validation i18n.RetranslationValidation
		if e = json.Unmarshal(v, &validation); e != nil {
			t.Fatal(e)
		}
		raw, e := readRegular(root, filepath.ToSlash(filepath.Join("data/retranslation-runs/zh-CN", u.SelectedBatchID, validation.RawResponsePath)))
		if e != nil {
			t.Fatal(e)
		}
		old, _ := r.HistoricalCatalog.Unit(u.UnitID)
		current, _ := c.Unit(r.Units[u.UnitID].CurrentUnitID)
		before := old.Adapter.(PageAdapter)
		after := current.Adapter.(PageAdapter)
		op, _ := protectionPieces(before.Document.Units[0])
		np, _ := protectionPieces(after.Document.Units[0])
		for _, q := range np {
			if q.token == nil {
				continue
			}
			covered := false
			for _, p := range op {
				if p.token != nil && q.start >= p.start && q.end <= p.end {
					covered = true
				}
			}
			if !covered {
				t.Logf("path=%s new_atom=%q role=%s literal_count_in_old_raw=%d", u.SourcePath, q.token.Raw, q.token.Role, strings.Count(string(raw), q.token.Raw))
				for j, p := range op {
					if p.token == nil && q.start >= p.start && q.end <= p.end {
						left, right := 0, len(raw)
						if j > 0 {
							left = strings.Index(string(raw), op[j-1].text) + len(op[j-1].text)
						}
						if j+1 < len(op) {
							right = strings.Index(string(raw), op[j+1].text)
						}
						if left >= 0 && right >= left {
							t.Logf("source_prose=%q target_prose=%q", p.text, string(raw[left:right]))
							if path := os.Getenv("GO_TOUR_VULN_BINDING_EVIDENCE_OUTPUT"); path != "" && u.SourcePath == "_content/doc/security/vuln/database.md" && strings.Count(string(raw[left:right]), "A") == 2 {
								positions := map[string][]int{}
								sourceSites := map[string][][2]int{}
								for _, literal := range []string{"A", "B"} {
									for at := left; at < right; {
										n := strings.Index(string(raw[at:right]), literal)
										if n < 0 {
											break
										}
										at += n
										positions[literal] = append(positions[literal], at)
										at += len(literal)
									}
									for _, site := range np {
										if site.token != nil && site.token.Raw == literal && site.start >= p.start && site.end <= p.end {
											sourceSites[literal] = append(sourceSites[literal], [2]int{site.start, site.end})
										}
									}
								}
								evidence := map[string]any{"schema": "engineering/representation-binding-proof/v1", "formal_artifact_mutation": "none", "disposition": "BLOCKED_NON_UNIQUE", "unknown": true, "old_unit_id": u.UnitID, "current_unit_id": current.ID, "source_path": u.SourcePath, "source_sha256": digest(after.Source), "old_candidate": map[string]string{"path": u.CandidatePath, "sha256": u.CandidateSHA256}, "old_validation": map[string]string{"path": u.ValidationPath, "sha256": u.ValidationSHA256}, "old_raw_sha256": digest(raw), "historical_documents_sha256": frozenSHA, "old_representation_sha256": identity(before.Document), "current_representation_sha256": identity(after.Document), "formal_impact_sha256": "8fb332e4ba2b8be55de91c057a3f174ac4bb922536c90e9d4e927e21e6b74e0e", "predecessor_snapshot": "learn-docs-v1-qc-014", "predecessor_snapshot_sha256": "234a37330a348590ab3b46e30d720ad42718dc798cb3002e212b86ce47acb619", "source_interval": [2]int{p.start, p.end}, "source_prose": p.text, "target_raw_interval": [2]int{left, right}, "target_prose": string(raw[left:right]), "source_sites": sourceSites, "target_sites": positions, "left_anchor": op[j-1].text, "right_anchor": op[j+1].text, "reason": "Each source A/B site has two indistinguishable plain-prose target sites between the same verified protected anchors. Swapping either pair preserves every available machine role, sibling/protected binding and enclosing structure. No unique correspondence; selecting by occurrence, nearest text or Chinese meaning is forbidden."}
								data, err := json.MarshalIndent(evidence, "", "  ")
								if err != nil {
									t.Fatal(err)
								}
								if !filepath.IsAbs(path) {
									path = filepath.Join(root, path)
								}
								if err = os.WriteFile(path, append(data, '\n'), 0644); err != nil {
									t.Fatal(err)
								}
							}
						}
					}
				}
			}
		}
	}
}
