package sitecontent

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Opt-in diagnosis of archived transport. A stale batch is never passed off as
// current; compare each exact input with current authority before reuse.
func TestFormalPageBatchReadOnlyDiagnosis(t *testing.T) {
	dir := os.Getenv("GO_TOUR_PAGE_DIAG_DIR")
	if dir == "" {
		t.Skip("set GO_TOUR_PAGE_DIAG_DIR to existing staging")
	}
	read := func(name, sha string) []byte {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || digest(b) != sha {
			t.Fatalf("input identity: %s: %v", name, err)
		}
		return b
	}
	bundle := read("zh-CN-learn-docs-page-batch-1-generation.zip", "b2ba5475f865e287125709cef2b33d46f7c787c8c93cf7ba68871977cf388179")
	output := read("zh-CN-learn-docs-page-batch-1-generation-result.json", "8c00a1ea6b432356191f9604886d94d9eae051eb5b2d952fc5df43e091fcb99d")
	fs, err := i18n.ReadTransportBundle(bundle, 256, 32<<20)
	if err != nil {
		t.Fatal(err)
	}
	var m Batch
	if err := StrictJSON(fs["manifest.json"], &m); err != nil {
		t.Fatal(err)
	}
	if err := i18n.ValidateTransportBundleInventory(fs, m.Files, true); err != nil {
		t.Fatal(err)
	}
	var o Output
	if err := StrictJSON(output, &o); err != nil {
		t.Fatal(err)
	}
	if m.Identity != "93bc9951db44b1ca73ec5d2e24fc44c29693dc168bbc2b96b45e3f8f47fa752c" || o.BatchSHA != m.Identity || len(o.Targets) != 30 {
		t.Fatal("archived identity/scope mismatch")
	}
	root := repositoryRoot(t)
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	docs, raw, err := WorkflowDocuments(root, g, m.Package)
	if err != nil {
		t.Fatal(err)
	}
	current := map[string]Document{}
	for _, d := range docs {
		current[d.Path] = d
	}
	targets := map[string]string{}
	for i, target := range o.Targets {
		if target.ID != m.Selected[i] {
			t.Fatal("archived exact-set/order mismatch")
		}
		targets[target.ID] = target.Text
	}
	type row struct {
		Path         string `json:"path"`
		OldID        string `json:"old_id"`
		NewID        string `json:"new_id"`
		Affected     bool   `json:"input_changed"`
		ArchiveError string `json:"archived_validation_error,omitempty"`
		ChangedSlots []struct {
			Old Unit `json:"old"`
			New Unit `json:"new"`
		} `json:"changed_slots,omitempty"`
	}
	rows := []row{}
	pass := 0
	wantFailures := map[string]bool{
		"_content/doc/modules/gomod-ref.md":          true,
		"_content/doc/tutorial/getting-started.html": true,
		"_content/doc/tutorial/web-service-gin.md":   true,
		"_content/doc/tutorial/fuzz.md":              true,
	}
	glossary, err := i18n.LoadGlossary(root, m.Locale)
	if err != nil {
		t.Fatal(err)
	}
	recoverable := []Document{}
	recoveredTargets := []Target{}
	for _, d := range m.Documents {
		next := current[d.Path]
		if !reflect.DeepEqual(raw[d.Path], fs["source/"+d.Path]) {
			t.Fatal("source bytes changed", d.Path)
		}
		old, now := d.Units[0], next.Units[0]
		r := row{Path: d.Path, OldID: old.ID, NewID: now.ID, Affected: !reflect.DeepEqual(old, now)}
		for _, oldSlot := range d.Slots {
			for _, newSlot := range next.Slots {
				if oldSlot.Start == newSlot.Start && oldSlot.End == newSlot.End && !reflect.DeepEqual(oldSlot, newSlot) {
					r.ChangedSlots = append(r.ChangedSlots, struct {
						Old Unit `json:"old"`
						New Unit `json:"new"`
					}{oldSlot, newSlot})
				}
			}
		}
		_, err := ReconstructWorkflow(d, raw[d.Path], map[string]string{old.ID: targets[old.ID]})
		if err != nil {
			r.ArchiveError = err.Error()
		} else {
			pass++
		}
		if (err != nil) != wantFailures[d.Path] {
			t.Errorf("unexpected archived validation: %s: %v", d.Path, err)
		}
		if !r.Affected {
			if glossaryErr := i18n.ValidateParsedGlossary(now.Source, targets[old.ID], glossary); glossaryErr != nil {
				t.Fatal(d.Path, glossaryErr)
			}
			_, currentErr := ReconstructWorkflow(next, raw[d.Path], map[string]string{now.ID: targets[old.ID]})
			if fmt.Sprint(currentErr) != fmt.Sprint(err) {
				t.Fatal("unchanged Page validation differs", d.Path, currentErr, err)
			}
			if currentErr == nil {
				recoverable = append(recoverable, next)
				recoveredTargets = append(recoveredTargets, Target{ID: now.ID, Text: targets[old.ID]})
			}
		}
		t.Logf("ARCHIVED path=%s old=%s new=%s affected=%t error=%v", r.Path, r.OldID, r.NewID, r.Affected, err)
		rows = append(rows, r)
	}
	if pass != 26 || len(recoverable) != 22 {
		t.Fatalf("diagnosis scope: pass=%d recoverable=%d", pass, len(recoverable))
	}
	if err := validateTargets(root, m.Locale, recoverable, recoveredTargets); err != nil {
		t.Fatal("exact-input recovery validation", err)
	}
	t.Logf("CURRENT exact-input target validation: %d/22 PASS; no import/new evidence", len(recoverable))
	t.Logf("ARCHIVED diagnosis %d/30 PASS; not a current-bundle validation", pass)
	if file := os.Getenv("GO_TOUR_PAGE_DIAG_REPORT"); file != "" {
		b, err := Encode(rows)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
