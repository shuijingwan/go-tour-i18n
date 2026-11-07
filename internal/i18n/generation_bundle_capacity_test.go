package i18n

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type generationCapacityAdapter struct {
	recoveryFixtureAdapter
	refs []GlossaryArchiveReference
}

func (a *generationCapacityAdapter) GenerationContextReferences(string, string) ([]GlossaryArchiveReference, error) {
	return a.refs, nil
}

// The reference closure, rather than the output count, exceeds the old reader
// limit. All source, locale and context evidence is synthetic and local.
func TestGenerationBundleAbove512FilesIsCurrent(t *testing.T) {
	root := t.TempDir()
	copyBundleAuthority(t, root, translationUnitGenerationAuthorityPaths)
	writeRetranslationTestGlossaryForLocale(t, root, "zh-CN")
	recordSyntheticUnifiedReview(t, root, "synthetic-capacity-review", "zh-CN")
	a := &generationCapacityAdapter{}
	if err := os.MkdirAll(filepath.Join(root, "context"), 0755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 600; i++ {
		path := fmt.Sprintf("context/source-%03d.txt", i)
		data := []byte(fmt.Sprintf("Frozen source authority %d\n", i))
		if err := os.WriteFile(filepath.Join(root, path), data, 0644); err != nil {
			t.Fatal(err)
		}
		a.refs = append(a.refs, GlossaryArchiveReference{Path: path, SHA256: sum(data)})
	}
	source := []byte("Frozen English source")
	c := &Catalog{}
	if err := c.RegisterTranslationPackage(TranslationPackageDescriptor{ID: "capacity-fixture", Units: []AdaptedTranslationUnit{{
		Unit:    TranslationUnit{ID: "capacity-home", Kind: UnitKindPage, Source: source, SourceSHA256: sum(source), SourcePath: "_content/fixture.md"},
		Adapter: a, Route: "/fixture/", StableIndex: 1,
	}}}); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage("capacity-fixture"); err != nil {
		t.Fatal(err)
	}
	batch, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	bundle, m, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: "zh-CN", BatchID: batch.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(bundle), int64(len(bundle)))
	if err != nil {
		t.Fatal(err)
	}
	if len(z.File) <= 512 || len(z.File) != len(m.Files)+1 || len(m.ExpectedOutputs) != 1 {
		t.Fatalf("regression must exceed old limit: members=%d inventory=%d outputs=%d", len(z.File), len(m.Files), len(m.ExpectedOutputs))
	}
	current, err := VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle)
	if err != nil || current.InputIdentitySHA256 != m.InputIdentitySHA256 {
		t.Fatalf("canonical current-check rejected exporter output: %v", err)
	}
	t.Logf("GT512_REGRESSION=PASS members=%d manifest_files=%d", len(z.File), len(m.Files))
	// More capacity must not turn a changed reference into current evidence.
	if err := os.WriteFile(filepath.Join(root, a.refs[0].Path), []byte("changed source authority"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle); err == nil {
		t.Fatal("current-check accepted a changed context dependency")
	}
}

func TestGenerationBundleDecoderCapacityRemainsBounded(t *testing.T) {
	t.Run("member-count", func(t *testing.T) {
		entries := make([]TransportBundleEntry, TranslationUnitGenerationBundleMaxFiles)
		for i := range entries {
			entries[i].Path = fmt.Sprintf("context/%05d.txt", i)
		}
		bundle, err := WriteDeterministicTransportBundle([]byte("{}"), entries)
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = decodeGenerationBundle(bundle)
		if err == nil || !strings.Contains(err.Error(), "file count") {
			t.Fatalf("central decoder did not reject excessive member count: %v", err)
		}
	})
	t.Run("total-bytes", func(t *testing.T) {
		// An oversized declared STORE member must fail before payload allocation.
		var buffer bytes.Buffer
		w := zip.NewWriter(&buffer)
		h := &zip.FileHeader{Name: "manifest.json", Method: zip.Store, UncompressedSize64: uint64(TranslationUnitGenerationBundleMaxBytes + 1)}
		h.SetMode(0644)
		if _, err := w.CreateRaw(h); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		_, _, err := decodeGenerationBundle(buffer.Bytes())
		if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("exceeds %d bytes", TranslationUnitGenerationBundleMaxBytes)) {
			t.Fatalf("central decoder did not reject excessive bytes: %v", err)
		}
	})
}
