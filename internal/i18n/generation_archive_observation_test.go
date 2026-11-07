package i18n

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Bounded engineering probe: the known five archives, with a saturated raw
// pool, and two fresh-read construction passes in one transformation lifetime.
// It performs no currentness shortcut and writes no repository artifacts.
func TestGenerationArchiveObservationExistingCapacity(t *testing.T) {
	root, census := os.Getenv("GO_TOUR_ARCHIVE_OBSERVATION_ROOT"), os.Getenv("GO_TOUR_ARCHIVE_OBSERVATION_CENSUS")
	if root == "" || census == "" {
		t.Skip("explicit disposable archive observation inputs required")
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || root != resolved || !bytes.HasPrefix([]byte(root), []byte("/tmp/")) {
		t.Fatal("canonical disposable repository required", err)
	}
	var archives []struct {
		Path    string `json:"path"`
		SHA     string `json:"sha256"`
		Batch   string `json:"batch"`
		Task    string `json:"task"`
		Attempt int    `json:"attempt"`
	}
	b, err := os.ReadFile(census)
	if err != nil || json.Unmarshal(b, &archives) != nil || len(archives) == 0 {
		t.Fatal("explicit archive census required", err)
	}
	done := contentidentity.BeginMemoization()
	defer done()
	// Use the real existing raw budget, not a test setter or production flag.
	for _, value := range []byte{0x38, 0xc1} {
		contentidentity.SHA256Hex(bytes.Repeat([]byte{value}, 32<<20))
	}
	for pass := 1; pass <= 2; pass++ {
		start := time.Now()
		for i := 0; i < 222; i++ {
			a := archives[i%len(archives)]
			fresh, err := readSnapshotReferencedFile(root, a.Path)
			if err != nil || sum(fresh) != a.SHA {
				t.Fatal("fresh archive SHA mismatch", a.Path, err)
			}
			m, files, err := readGenerationBundle(fresh)
			if err != nil || m.BatchID != a.Batch || m.TaskKind != a.Task || m.Attempt != a.Attempt {
				t.Fatal("archive identity/contract changed", a.Path, err)
			}
			if err := ValidateTransportBundleInventory(files, m.Files, true); err != nil {
				t.Fatal(err)
			}
			if _, err := contentidentity.Transform(generationArchiveIndexOperation, generationArchiveObservationKey(fresh), func() ([]byte, error) {
				return nil, fmt.Errorf("saturated raw pool still starved decoded archive index")
			}); err != nil {
				t.Fatal(err)
			}
		}
		t.Logf("ARCHIVE_OBSERVATION pass=%d fresh_reads=222 elapsed_seconds=%.6f", pass, time.Since(start).Seconds())
	}
}

func observedArchiveFixture(t *testing.T, payload []byte) []byte {
	return observedArchiveFixtureWithMethod(t, payload, zip.Store)
}

func observedArchiveFixtureWithMethod(t *testing.T, payload []byte, method uint16) []byte {
	t.Helper()
	m := GenerationBundleManifest{SchemaVersion: 1, Kind: "go-tour-i18n/generation-bundle", Locale: "xx", BatchID: "synthetic-archive", ExpectedOutputs: []GenerationBundleExpectedOutput{{BundlePath: "outputs/unit.article", InstallPath: "raw-responses/unit.article", UnitID: "unit"}}}
	m.Files = []TransportBundleFile{NewTransportBundleFile("formal/input", "", payload)}
	b, err := writeDeterministicTransportBundle(mustJSON(m), []TransportBundleEntry{{Path: "formal/input", Data: payload}}, method)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestGenerationArchiveObservedDecode(t *testing.T) {
	for _, method := range []uint16{zip.Store, zip.Deflate} {
		t.Run(fmt.Sprint(method), func(t *testing.T) { testGenerationArchiveObservedDecode(t, method) })
	}
}

func testGenerationArchiveObservedDecode(t *testing.T, method uint16) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	archive := observedArchiveFixtureWithMethod(t, bytes.Repeat([]byte("exact source-owned protected input\n"), 8192), method)
	want, files, err := decodeGenerationBundle(archive)
	if err != nil {
		t.Fatal(err)
	}
	done := contentidentity.BeginMemoization()
	for i := 0; i < 3; i++ {
		got, copy, err := readGenerationBundle(bytes.Clone(archive))
		if err != nil || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(copy, files) {
			t.Fatal("observed decode changed canonical result", err)
		}
		got.Files[0].SHA256 = "caller mutation"
		copy["formal/input"][0] ^= 1
		delete(copy, "manifest.json")
	}
	if _, err := contentidentity.Transform(generationArchiveIndexOperation, generationArchiveObservationKey(archive), func() ([]byte, error) {
		return nil, fmt.Errorf("same invocation failed to reuse the immutable index")
	}); err != nil {
		t.Fatal(err)
	}
	// Fresh second-build bytes cannot reuse a prior structural verdict for a
	// different archive. CRC/inventory validation still rejects the mutation.
	tampered := bytes.Clone(archive)
	z, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range z.File {
		if f.Name == "formal/input" {
			at, err := f.DataOffset()
			if err != nil {
				t.Fatal(err)
			}
			tampered[at+int64(f.CompressedSize64)/2] ^= 1
		}
	}
	for i := 0; i < 2; i++ {
		if _, _, err := readGenerationBundle(tampered); err == nil {
			t.Fatal("changed bytes or failed parse reused an old index")
		}
	}
	failedRecomputed := false
	_, _ = contentidentity.Transform(generationArchiveIndexOperation, generationArchiveObservationKey(tampered), func() ([]byte, error) {
		failedRecomputed = true
		return nil, fmt.Errorf("failed structural proof is not reusable")
	})
	if !failedRecomputed {
		t.Fatal("failed structural decode was retained")
	}
	other := observedArchiveFixture(t, []byte("different exact valid archive"))
	if _, otherFiles, err := readGenerationBundle(other); err != nil || bytes.Equal(otherFiles["formal/input"], files["formal/input"]) {
		t.Fatal("different valid identity reused old payload", err)
	}
	done()
	if contentidentity.MemoizationActive() {
		t.Fatal("archive owner escaped invocation")
	}
	finish := contentidentity.BeginMemoization()
	newOwner := false
	if _, err := contentidentity.Transform(generationArchiveIndexOperation, generationArchiveObservationKey(archive), func() ([]byte, error) {
		newOwner = true
		return indexGenerationArchive(archive)
	}); err != nil || newOwner {
		t.Fatal("fresh invocation did not reuse the exact pure L2 index", err)
	}
	finish()
	for _, root := range []string{t.TempDir(), t.TempDir()} {
		path := filepath.Join(root, "archive.zip")
		if err := os.WriteFile(path, archive, 0600); err != nil {
			t.Fatal(err)
		}
		finish := contentidentity.BeginMemoization()
		b, _ := os.ReadFile(path)
		if _, _, err := readGenerationBundle(b); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, tampered, 0600); err != nil {
			t.Fatal(err)
		}
		b, _ = os.ReadFile(path)
		if _, _, err := readGenerationBundle(b); err == nil {
			t.Fatal("fresh filesystem bytes replaced by previous archive")
		}
		finish()
	}
}

func TestGenerationArchiveObservedDecodeRetainsStructuralRejection(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	for _, kind := range []string{"wrong-kind", "bad-inventory", "unsafe-output"} {
		t.Run(kind, func(t *testing.T) {
			b := observedArchiveFixture(t, []byte("fixture"))
			m, files, _ := decodeGenerationBundle(b)
			switch kind {
			case "wrong-kind":
				m.Kind = "unrelated"
			case "bad-inventory":
				m.Files[0].SHA256 = sum([]byte("wrong"))
			case "unsafe-output":
				m.ExpectedOutputs[0].BundlePath = "../escape"
			}
			raw, _ := json.Marshal(m)
			b, _ = WriteDeterministicTransportBundle(raw, []TransportBundleEntry{{Path: "formal/input", Data: files["formal/input"]}})
			if _, _, err := readGenerationBundle(b); err == nil {
				t.Fatal("structural contract weakened", kind)
			}
		})
	}
}
