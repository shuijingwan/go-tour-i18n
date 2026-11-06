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
func TestInlineLanguageProjection(t *testing.T) {
	output := os.Getenv("GO_TOUR_INLINE_PROJECTION")
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

func TestInlineLanguageImpact(t *testing.T) {
	before := os.Getenv("GO_TOUR_INLINE_BEFORE")
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
	b, err = os.ReadFile(filepath.Join(root, "data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-001/manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &snapshot); err != nil {
		t.Fatal(err)
	}
	var qc i18n.QualityCheckResults
	b, err = os.ReadFile(filepath.Join(root, "data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-001/quality-check-results.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &qc); err != nil {
		t.Fatal(err)
	}
	glossary, err := i18n.LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	if len(old) != 49 || len(next) != 49 || len(snapshot.Units) != 49 || len(qc.Results) != 49 {
		t.Fatal("exact 49 Page scope")
	}
	var report bytes.Buffer
	report.WriteString("# Learn/Docs inline language parser 影响分析\n\n历史 predecessor：`learn-docs-v1-qc-001`。本报告是诊断，不是 Generation、Snapshot 或 QC evidence。\n\n")
	changed, carried, revision := []int{}, []int{}, []int{}
	for i, d := range next {
		o := old[i]
		s := snapshot.Units[i]
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
		if drift {
			changed = append(changed, d.StableIndex)
			fmt.Fprintf(&report, "正式 Unit source identity（index %d）：old=`%s`；new=`%s`。\n\n", d.StableIndex, o.Units[0].SourceSHA, d.Units[0].SourceSHA)
			fmt.Fprintf(&report, "## %d — %s\n\n- old Unit ID: `%s`\n- new Unit ID: `%s`\n- old input SHA: `%s`\n- new input SHA: `%s`\n- source SHA（bytes 不变）: `%s`\n- old structure identity: `%s`\n- new structure identity: `%s`\n- internal slots: %d → %d；protected raw/roles 与 formal input 变化；原评级 %s。\n- 原因：直接自然语言 run 包含完整 inline protected element，此前被整体排除；现在作为语言槽位和 inline tokens。旧 candidate 不兼容，需 Generation/validation 和独立 QC。\n\n", d.StableIndex, d.Path, o.Units[0].ID, d.Units[0].ID, record.InputSHA256, digest(newInput), d.SourceSHA, o.StructureSHA, d.StructureSHA, len(o.Slots), len(d.Slots), qc.Results[i].Rating)
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
			if qc.Results[i].Rating == "A" {
				carried = append(carried, d.StableIndex)
			} else {
				revision = append(revision, d.StableIndex)
			}
		}
	}
	fmt.Fprintf(&report, "## 精确 scope\n\nchanged=%d；unchanged=%d。\n\n- identity_changed / Generation + validation + QC: `%v`\n- unchanged A carry-forward: `%v`\n- unchanged B/C 可按原 finding revision: `%v`\n\nunchanged 的 path/source bytes/internal slots/protected raw/roles/formal input/Unit ID/source identity/structure identity 全部逐字段一致；candidate 字节、restore 和当前 validator 均通过。\n", len(changed), 49-len(changed), changed, carried, revision)
	t.Log(report.String())
	if output := os.Getenv("GO_TOUR_INLINE_REPORT"); output != "" {
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

func stringsTrimLF(s string) string { return strings.TrimRight(s, "\n") }
