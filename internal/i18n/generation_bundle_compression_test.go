package i18n

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
)

func generationCompressionEntries(m GenerationBundleManifest, files map[string][]byte) []TransportBundleEntry {
	entries := make([]TransportBundleEntry, 0, len(m.Files))
	for _, f := range m.Files {
		entries = append(entries, TransportBundleEntry{Path: f.BundlePath, Data: files[f.BundlePath]})
	}
	return entries
}

func TestGenerationBundleDeflateCanonicalTransport(t *testing.T) {
	root, catalog, batch := generationBundleFixture(t, "zh-CN", RetranslationGeneratorChatGPT)
	opts := GenerationBundleOptions{Locale: "zh-CN", BatchID: batch}
	first, m, err := ExportTranslationUnitGenerationBundle(root, catalog, opts)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := ExportTranslationUnitGenerationBundle(root, catalog, opts)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatal("independent canonical reconstructions differ", err)
	}
	t.Logf("DEFLATE_BUILD_1_SHA256=%s DEFLATE_BUILD_2_SHA256=%s", sum(first), sum(second))
	parsed, files, err := readGenerationBundle(first)
	if err != nil || !reflect.DeepEqual(parsed, m) {
		t.Fatal("Deflate parser/inventory/hash proof", err)
	}
	z, err := zip.NewReader(bytes.NewReader(first), int64(len(first)))
	if err != nil {
		t.Fatal(err)
	}
	for i, f := range z.File {
		if f.Method != zip.Deflate || f.Mode().Perm() != 0644 || !f.Modified.Equal(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)) {
			t.Fatal("non-canonical method or metadata", f.Name)
		}
		if (i == 0 && f.Name != "manifest.json") || (i > 1 && z.File[i-1].Name >= f.Name) {
			t.Fatal("non-canonical order", f.Name)
		}
		// Level is not an observable ZIP field. Compare the actual compressed
		// stream against independently configured level 5, with fixed writes.
		var expected bytes.Buffer
		compressor, err := flate.NewWriter(&expected, 5)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = compressor.Write(files[f.Name]); err != nil {
			t.Fatal(err)
		}
		if err = compressor.Close(); err != nil {
			t.Fatal(err)
		}
		raw, err := f.OpenRaw()
		if err != nil {
			t.Fatal(err)
		}
		actual, err := io.ReadAll(raw)
		if err != nil || !bytes.Equal(actual, expected.Bytes()) {
			t.Fatal("canonical compressor differs from explicit level 5", f.Name, err)
		}
	}
	if _, err := VerifyCurrentTranslationUnitGenerationBundle(root, catalog, first); err != nil {
		t.Fatal("new Deflate is not current", err)
	}
	legacy, err := WriteDeterministicTransportBundle(files["manifest.json"], generationCompressionEntries(m, files))
	if err != nil {
		t.Fatal(err)
	}
	lm, lf, err := readGenerationBundle(legacy)
	if err != nil || !reflect.DeepEqual(lm, m) || !reflect.DeepEqual(lf, files) {
		t.Fatal("legacy Store lost exact logical graph", err)
	}
	if _, err = VerifyCurrentTranslationUnitGenerationBundle(root, catalog, legacy); err == nil || !strings.Contains(err.Error(), "stale or non-canonical") {
		t.Fatal("readable Store incorrectly remains canonical CURRENT", err)
	}
	legacyZIP, err := zip.NewReader(bytes.NewReader(legacy), int64(len(legacy)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range legacyZIP.File {
		if f.Method != zip.Store {
			t.Fatal("shared default writer changed another family")
		}
	}
	if len(first) >= len(legacy)/2 {
		t.Fatalf("representative source/authority Bundle did not compress: deflate=%d store=%d", len(first), len(legacy))
	}
	t.Logf("COMPRESSION_SIZE_REGRESSION=PASS deflate=%d store=%d", len(first), len(legacy))
}

func TestGenerationDeflateLogicalByteLimit(t *testing.T) {
	// Tiny physical ZIP, much larger decoded payload: budget is logical bytes.
	payload := bytes.Repeat([]byte("frozen source evidence\n"), 8192)
	bundle := observedArchiveFixtureWithMethod(t, payload, zip.Deflate)
	if len(bundle) >= 8192 {
		t.Fatal("fixture no longer exercises compression expansion")
	}
	if _, err := ReadTransportBundle(bundle, TranslationUnitGenerationBundleMaxFiles, 8192); err == nil || !strings.Contains(err.Error(), "exceeds 8192 bytes") {
		t.Fatal("compressed size substituted for logical budget", err)
	}
	// Check the canonical 128 MiB limit without allocating an oversized payload.
	for _, size := range []uint64{uint64(TranslationUnitGenerationBundleMaxBytes + 1), ^uint64(0)} {
		var buffer bytes.Buffer
		w := zip.NewWriter(&buffer)
		h := &zip.FileHeader{Name: "manifest.json", Method: zip.Deflate, UncompressedSize64: size}
		h.SetMode(0644)
		if _, err := w.CreateRaw(h); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		if _, _, err := decodeGenerationBundle(buffer.Bytes()); err == nil || !strings.Contains(err.Error(), fmt.Sprintf("exceeds %d bytes", TranslationUnitGenerationBundleMaxBytes)) {
			t.Fatal("central decoder lost logical bound or unsigned size safety", size, err)
		}
	}
}

func TestGenerationDeflateCRCAndInventory(t *testing.T) {
	bundle := observedArchiveFixtureWithMethod(t, []byte("exact logical payload"), zip.Deflate)
	z, err := zip.NewReader(bytes.NewReader(bundle), int64(len(bundle)))
	if err != nil {
		t.Fatal(err)
	}
	// Keep a valid compressed stream and correct sizes; corrupt only CRC.
	var bad bytes.Buffer
	w := zip.NewWriter(&bad)
	for _, f := range z.File {
		h := f.FileHeader
		h.Flags &^= 8 // explicit sizes/CRC, no data descriptor
		h.CRC32 ^= 1
		dst, err := w.CreateRaw(&h)
		if err != nil {
			t.Fatal(err)
		}
		src, err := f.OpenRaw()
		if err != nil {
			t.Fatal(err)
		}
		if _, err = io.Copy(dst, src); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readGenerationBundle(bad.Bytes()); err == nil {
		t.Fatal("corrupted Deflate CRC accepted")
	}
	m, files, err := readGenerationBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	files["formal/input"] = []byte("different valid compressed bytes")
	tampered, err := writeTranslationUnitGenerationBundle(files["manifest.json"], generationCompressionEntries(m, files))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := readGenerationBundle(tampered); err == nil || !strings.Contains(err.Error(), "identity mismatch") {
		t.Fatal("valid CRC bypassed logical inventory/hash validation", err)
	}
}
