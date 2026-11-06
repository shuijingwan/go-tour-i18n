package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCommentProcessContextRejectsDependencyMutation(t *testing.T) {
	root, c, o, e := importedCommentProcessFixture(t)
	dir := filepath.Join(root, "data/retranslation-runs", o.Locale, e.BatchID)
	m, err := readRetranslationProcessManifest(dir, o.Locale, e.BatchID)
	if err != nil {
		t.Fatal(err)
	}
	g, err := LoadGlossary(root, o.Locale)
	if err != nil {
		t.Fatal(err)
	}
	p, check, err := prepareProcessOutputContext(dir, c, g, m)
	if err != nil || len(p) != 9 {
		t.Fatal(len(p), err)
	}
	paths := []string{m.Units[0].SourcePath, "data/retranslation-runs/" + o.Locale + "/" + e.BatchID + "/" + m.Units[0].InputPath,
		"data/retranslation-runs/" + o.Locale + "/" + e.BatchID + "/raw-responses/" + filepath.Base(m.Units[0].InputPath),
		"data/retranslation-runs/" + o.Locale + "/" + e.BatchID + "/manifest.json", glossaryReviewGlossaryPath(o.Locale),
		"internal/i18n/generation_revision_inputs.go", "data/quality-check-reopens/" + o.Locale + "/" + o.ReopenID + ".json",
		"data/quality-check-snapshots/" + o.Locale + "/" + o.PreviousSnapshotID + "/manifest.json"}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			full := filepath.Join(root, path)
			b, err := os.ReadFile(full)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(full, append(append([]byte(nil), b...), '\n'), 0600); err != nil {
				t.Fatal(err)
			}
			defer os.WriteFile(full, b, 0600)
			if err := check(); err == nil {
				t.Fatal("changed authority accepted")
			}
		})
	}
	if err := check(); err != nil {
		t.Fatal("unchanged exact authority rejected", err)
	}
}

type mutatingProcessAdapter struct {
	validationReopenAdapter
	mutate func()
}

func (a mutatingProcessAdapter) Validate(b []byte, g *Glossary) error {
	if err := a.validationReopenAdapter.Validate(b, g); err != nil {
		return err
	}
	a.mutate()
	return nil
}
func TestCommentProcessContextChecksBeforeCommit(t *testing.T) {
	root, c, o, e := importedCommentProcessFixture(t)
	binding := c.Adapted[0]
	once := false
	binding.Adapter = mutatingProcessAdapter{binding.Adapter.(validationReopenAdapter), func() {
		if once {
			return
		}
		once = true
		p := filepath.Join(root, binding.Unit.SourcePath)
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(p, append(b, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}}
	c.Adapted[0] = binding
	c.TranslationPackages[0].Units[0] = binding
	if _, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: o.Locale, BatchID: e.BatchID}); err == nil {
		t.Fatal("mid-invocation source change committed")
	}
	dir := filepath.Join(root, "data/retranslation-runs", o.Locale, e.BatchID)
	for _, path := range []string{"result.json", "candidates", "validation"} {
		if _, err := os.Lstat(filepath.Join(dir, path)); !os.IsNotExist(err) {
			t.Fatal("partial process publication", path, err)
		}
	}
}
