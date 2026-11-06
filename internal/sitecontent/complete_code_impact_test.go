package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Explicit read-only projection of the complete frozen package. The output is
// diagnostic, never a candidate, Snapshot, receipt or review.
func TestCompleteCodeProjection(t *testing.T) {
	output := os.Getenv("GO_TOUR_TEACHING_PROJECTION")
	if output == "" {
		t.Skip("explicit frozen package projection")
	}
	root := repositoryRoot(t)
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	docs, _, err := WorkflowDocuments(root, g, "learn-docs-v1")
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 49 {
		t.Fatal(len(docs))
	}
	b, err := json.MarshalIndent(docs, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.Write(append(b, '\n')); err != nil {
		t.Fatal(err)
	}
}

func TestCompleteCodeImpact(t *testing.T) {
	before := os.Getenv("GO_TOUR_TEACHING_BEFORE")
	if before == "" {
		t.Skip("explicit before/current comparison")
	}
	root := repositoryRoot(t)
	b, err := os.ReadFile(before)
	if err != nil {
		t.Fatal(err)
	}
	var old []Document
	if err := json.Unmarshal(b, &old); err != nil {
		t.Fatal(err)
	}
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	next, raw, err := WorkflowDocuments(root, g, "learn-docs-v1")
	if err != nil {
		t.Fatal(err)
	}
	var snapshot i18n.QualityCheckSnapshotManifest
	b, err = os.ReadFile(filepath.Join(root, "data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-002/manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &snapshot); err != nil {
		t.Fatal(err)
	}
	var qc i18n.QualityCheckResults
	b, err = os.ReadFile(filepath.Join(root, "data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-002/quality-check-results.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &qc); err != nil {
		t.Fatal(err)
	}
	// qc-002 stores seven new judgments; the other exact identities retain
	// their immutable qc-001 lineage. This is read-only diagnostic selection.
	var predecessor i18n.QualityCheckResults
	b, err = os.ReadFile(filepath.Join(root, "data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-001/quality-check-results.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &predecessor); err != nil {
		t.Fatal(err)
	}
	ratings := map[string]string{}
	for _, r := range predecessor.Results {
		ratings[r.UnitID] = r.Rating
	}
	for _, r := range qc.Results {
		ratings[r.UnitID] = r.Rating
	}
	glossary, err := i18n.LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	if len(old) != 49 || len(next) != 49 || len(snapshot.Units) != 49 {
		t.Fatal("exact 49 Page scope")
	}
	var report bytes.Buffer
	var inventory bytes.Buffer
	inventory.WriteString("\n## 全部49 Page 的逐字段对照\n\n全部 source bytes unchanged。teaching span 是 source coordinate 内部教学槽位；before / after 完整 Document（含 slots、protected raw/role/pair、正式 input）分别保存在 `/tmp/go-tour-teaching-before.json` 和 `/tmp/go-tour-teaching-after.json`。unchanged 使用完整 Document 的 deep equality 判定；不是仅比较 Unit ID。\n\n| index | path | 状态 | slots old/new | teaching spans old/new | protected old/new |\n| --- | --- | --- | --- | --- | --- |\n")
	report.WriteString("# Learn/Docs complete-code teaching-comment parser 影响分析\n\n历史 predecessor：`learn-docs-v1-qc-002`。本报告是诊断，不是 Generation、Snapshot 或 QC evidence。\n\n")
	changed, carried, revision := []int{}, []int{}, []int{}
	for i, d := range next {
		o := old[i]
		s := snapshot.Units[i]
		rating := ratings[s.UnitID]
		if rating == "" {
			t.Fatal("missing historical QC lineage", s.UnitID)
		}
		if o.Path != d.Path || o.SourceSHA != d.SourceSHA || digest(raw[d.Path]) != o.SourceSHA || o.Units[0].ID != s.UnitID {
			t.Fatal("baseline/source mismatch", d.Path)
		}
		var m i18n.RetranslationBatchManifest
		b, err := os.ReadFile(filepath.Join(root, "data/retranslation-runs/zh-CN", s.SelectedBatchID, "manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatal(err)
		}
		var record i18n.RetranslationBatchUnit
		for _, r := range m.Units {
			if r.UnitID == s.UnitID {
				record = r
			}
		}
		input, err := os.ReadFile(filepath.Join(root, "data/retranslation-runs/zh-CN", s.SelectedBatchID, record.InputPath))
		if err != nil {
			t.Fatal(err)
		}
		if digest(input) != record.InputSHA256 || !bytes.Equal(input, append([]byte(stringsTrimLF(o.Units[0].Source)), '\n')) {
			t.Fatal("old formal input mismatch", d.Path)
		}
		newInput := append([]byte(stringsTrimLF(d.Units[0].Source)), '\n')
		drift := !reflect.DeepEqual(o, d)
		teachingCount := func(doc Document) int {
			n := 0
			for _, slot := range doc.Slots {
				if strings.Contains(slot.Locator, "#teaching-comment:") {
					n++
				}
			}
			return n
		}
		state := "unchanged"
		if drift {
			state = "identity_changed"
		}
		fmt.Fprintf(&inventory, "| %d | %s | %s | %d/%d | %d/%d | %d/%d |\n", d.StableIndex, d.Path, state, len(o.Slots), len(d.Slots), teachingCount(o), teachingCount(d), len(o.Units[0].Protected), len(d.Units[0].Protected))
		if drift {
			changed = append(changed, d.StableIndex)
			fmt.Fprintf(&report, "正式 Unit source identity（index %d）：old=`%s`；new=`%s`。\n\n", d.StableIndex, o.Units[0].SourceSHA, d.Units[0].SourceSHA)
			fmt.Fprintf(&report, "## %d — %s\n\n- old Unit ID: `%s`\n- new Unit ID: `%s`\n- old input SHA: `%s`\n- new input SHA: `%s`\n- source SHA（bytes 不变）: `%s`\n- old structure identity: `%s`\n- new structure identity: `%s`\n- internal slots: %d → %d；protected raw/roles 与 formal input 变化；原评级 %s。\n- 原因：共享 analyzer 现在接受完整 Go 文件，完整 teaching body 保留为一个 slot，identifier 为 comment 内 atom；其余代码仍为 boundary。旧 candidate 不兼容，需 Generation/validation 和独立 QC。\n\n", d.StableIndex, d.Path, o.Units[0].ID, d.Units[0].ID, record.InputSHA256, digest(newInput), d.SourceSHA, o.StructureSHA, d.StructureSHA, len(o.Slots), len(d.Slots), rating)
			if o.Units[0].ID == d.Units[0].ID || bytes.Equal(input, newInput) {
				t.Fatal("unexpected diagnostic-only drift")
			}
		} else {
			candidate, err := os.ReadFile(filepath.Join(root, s.CandidatePath))
			if err != nil || digest(candidate) != s.CandidateSHA256 {
				t.Fatal("candidate hash", err)
			}
			vb, err := os.ReadFile(filepath.Join(root, s.ValidationPath))
			if err != nil || digest(vb) != s.ValidationSHA256 {
				t.Fatal("validation hash", err)
			}
			var validation i18n.RetranslationValidation
			if err := json.Unmarshal(vb, &validation); err != nil {
				t.Fatal(err)
			}
			output, err := os.ReadFile(filepath.Join(root, "data/retranslation-runs/zh-CN", s.SelectedBatchID, validation.RawResponsePath))
			if err != nil {
				t.Fatal(err)
			}
			a := PageAdapter{d, raw[d.Path]}
			got, err := a.Restore(string(output))
			if err != nil || !bytes.Equal(append(bytes.TrimRight(got, "\n"), '\n'), candidate) {
				t.Fatal("unchanged candidate restoration", d.Path, err)
			}
			if err := a.Validate(candidate, glossary); err != nil {
				t.Fatal(err)
			}
			if rating == "A" {
				carried = append(carried, d.StableIndex)
			} else {
				revision = append(revision, d.StableIndex)
			}
		}
	}
	fmt.Fprintf(&report, "## 精确 scope\n\nchanged=%d；unchanged=%d。\n\n- identity_changed / Generation + validation + QC: `%v`\n- unchanged A carry-forward: `%v`\n- unchanged B/C 可按原 finding revision: `%v`\n\nunchanged 的 path/source bytes/internal slots/protected raw/roles/formal input/Unit ID/source identity/structure identity 全部逐字段一致；candidate 字节、restore 和当前 validator 均通过。\n", len(changed), 49-len(changed), changed, carried, revision)
	report.Write(inventory.Bytes())
	t.Log(report.String())
	if output := os.Getenv("GO_TOUR_TEACHING_REPORT"); output != "" {
		f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if _, err := f.Write(report.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
}
