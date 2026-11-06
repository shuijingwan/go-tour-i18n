package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestSurfaceResultPackPreimportFive(t *testing.T) {
	root, c, batch := surfaceCapabilityFixtureSize(t, 5)
	d, err := i18n.PrepareSurfaceRevisionTransport(root, c, "zh-CN", batch, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	r := surfaceCapabilityRequest(t, d, true)
	p, err := i18n.PrepareSurfaceRevisionTransport(root, c, "zh-CN", batch, &r, true)
	if err != nil {
		t.Fatal(err)
	}
	bundle, bm, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: batch})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, b := range p.Authority.Bindings {
		writeFixture(t, dir, b.UnitID+".article", bridgeEOF([]byte(strings.Replace(b.Projection.Template, "function", "函数", 1))))
	}
	seal := filepath.Join(root, "data/retranslation-runs/zh-CN", batch, "generation-imports/initial-manifest.json")
	if _, err := os.Stat(seal); !os.IsNotExist(err) {
		t.Fatal("fixture already imported", err)
	}
	out := os.Getenv("GO_TOUR_SURFACE_PACK_PROFILE")
	if out != "" {
		if err := os.MkdirAll(out, 0700); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(out, "result-pack.cpu.pprof"))
		if err != nil {
			t.Fatal(err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		defer pprof.StopCPUProfile()
	}
	done := contentidentity.BeginMemoization()
	start := time.Now()
	result, rm, packErr := i18n.PackGenerationResultBundle(root, c, bundle, "chatgpt", i18n.FormalChatGPTGenerationModel, dir)
	seconds := time.Since(start).Seconds()
	done()
	t.Logf("RESULT_PACK wall_seconds=%.3f units=%d err=%v", seconds, len(rm.Outputs), packErr)
	if out != "" {
		b, _ := json.MarshalIndent(map[string]any{"wall_seconds": seconds, "outputs": len(rm.Outputs), "error": func() string {
			if packErr != nil {
				return packErr.Error()
			}
			return ""
		}()}, "", "  ")
		if err := os.WriteFile(filepath.Join(out, "metrics.json"), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if packErr != nil {
		t.Fatal("pre-import result-pack must not require an imported archive", packErr)
	}
	if len(rm.Outputs) != 5 || bm.Attempt != 1 || len(result) == 0 {
		t.Fatal("incomplete result", rm)
	}
	if _, err := os.Stat(seal); !os.IsNotExist(err) {
		t.Fatal("pack installed import receipt", err)
	}
	if out != "" {
		return
	} // Profiling measures one command, not the rejection matrix.
	first := p.Authority.Bindings[0]
	outputPath := filepath.Join(dir, first.UnitID+".article")
	before, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	var codec struct{ Parts []struct{ Token string } }
	if err := json.Unmarshal(first.Projection.Codec, &codec); err != nil || len(codec.Parts) == 0 {
		t.Fatal("projection token fixture", err)
	}
	token := codec.Parts[0].Token
	for _, name := range []string{"missing-token", "duplicate-token", "changed-token", "missing-output", "extra-output", "stale-review", "tampered-authority", "tampered-codec"} {
		t.Run(name, func(t *testing.T) {
			input := bundle
			switch name {
			case "missing-token", "duplicate-token", "changed-token":
				edit := ""
				if name == "duplicate-token" {
					edit = token + token
				}
				if name == "changed-token" {
					edit = "⟪GTI18N_UNKNOWN⟫"
				}
				writeFixture(t, dir, filepath.Base(outputPath), bytes.Replace(before, []byte(token), []byte(edit), 1))
				defer writeFixture(t, dir, filepath.Base(outputPath), before)
			case "missing-output":
				if err := os.Remove(outputPath); err != nil {
					t.Fatal(err)
				}
				defer writeFixture(t, dir, filepath.Base(outputPath), before)
			case "extra-output":
				writeFixture(t, dir, "other.article", before)
				defer os.Remove(filepath.Join(dir, "other.article"))
			case "stale-review":
				path := p.Authority.CurrentReview.Path
				b, err := readRegular(root, path)
				if err != nil {
					t.Fatal(err)
				}
				writeFixture(t, root, path, append(bytes.Clone(b), ' '))
				defer writeFixture(t, root, path, b)
			case "tampered-authority", "tampered-codec":
				path := "formal/surface-transport-authority/" + first.UnitID + ".json"
				if name == "tampered-codec" {
					path = "formal/surface-transport/" + first.UnitID + ".json"
				}
				input = rewriteSurfaceGenerationBundle(t, bundle, func(fs map[string][]byte) { fs[path] = append(fs[path], ' ') })
			}
			if _, _, err := i18n.PackGenerationResultBundle(root, c, input, "chatgpt", i18n.FormalChatGPTGenerationModel, dir); err == nil {
				t.Fatal("invalid pre-import transport accepted", name)
			}
			if _, err := os.Stat(seal); !os.IsNotExist(err) {
				t.Fatal("negative pack mutated lifecycle", err)
			}
		})
	}
	// The correction must not change the canonical Generation bytes or Result
	// framing. Both result-pack and import independently verify the same Bundle.
	again, _, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: batch})
	if err != nil || !bytes.Equal(again, bundle) {
		t.Fatal("Generation Bundle bytes changed", err)
	}
	second, _, err := i18n.PackGenerationResultBundle(root, c, bundle, "chatgpt", i18n.FormalChatGPTGenerationModel, dir)
	if err != nil || !bytes.Equal(result, second) {
		t.Fatal("non-deterministic Result transport", err)
	}
	if _, err := i18n.ImportGenerationResultBundle(root, c, bundle, result); err != nil {
		t.Fatal("isolated import --result", err)
	}
	sealed, err := os.ReadFile(seal)
	if err != nil {
		t.Fatal(err)
	}
	base := "data/retranslation-runs/zh-CN/" + batch
	archive := "data/generation-archives/zh-CN/" + digest(bundle) + ".zip"
	for _, name := range []string{"missing-seal", "tampered-seal", "wrong-provider", "wrong-model", "wrong-attempt", "wrong-archive-sha", "archive-tamper", "current-bundle-substitution", "raw-output-sha", "receipt-output-sha"} {
		t.Run("postimport-"+name, func(t *testing.T) {
			path := base + "/generation-imports/initial-manifest.json"
			original := sealed
			changed := bytes.Clone(sealed)
			var receipt i18n.GenerationResultBundleManifest
			if err := json.Unmarshal(sealed, &receipt); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "missing-seal":
				if err := os.Remove(seal); err != nil {
					t.Fatal(err)
				}
				defer writeFixture(t, root, path, original)
			case "tampered-seal":
				changed = []byte("invalid")
			case "wrong-provider":
				receipt.Provider = "codex"
			case "wrong-model":
				receipt.Model = "unapproved-model"
			case "wrong-attempt":
				receipt.Attempt++
			case "wrong-archive-sha":
				receipt.GenerationBundleSHA256 = strings.Repeat("0", 64)
			case "receipt-output-sha":
				receipt.Outputs[0].SHA256 = strings.Repeat("0", 64)
			case "archive-tamper", "current-bundle-substitution":
				path = archive
				original = bundle
				changed = append(bytes.Clone(bundle), 'x')
				if name == "current-bundle-substitution" {
					changed = rewriteSurfaceGenerationBundle(t, bundle, func(fs map[string][]byte) { fs["generation-context.json"] = append(fs["generation-context.json"], ' ') })
				}
			case "raw-output-sha":
				path = base + "/raw-responses/" + first.UnitID + ".article"
				original = before
				changed = bytes.Replace(before, []byte("函数"), []byte("方法"), 1)
			}
			if name != "missing-seal" {
				if strings.HasPrefix(name, "wrong-") || name == "receipt-output-sha" {
					changed, _ = json.Marshal(receipt)
				}
				writeFixture(t, root, path, changed)
				defer writeFixture(t, root, path, original)
			}
			if _, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: batch}); err == nil {
				t.Fatal("historical decoder accepted missing/tampered imported provenance", name)
			}
			if _, err := os.Stat(filepath.Join(root, base, "result.json")); !os.IsNotExist(err) {
				t.Fatal("failed historical guard published process evidence", err)
			}
		})
	}
	processed, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: batch})
	if err != nil || processed.ValidationPassed != 5 {
		t.Fatal("isolated sealed process", processed, err)
	}
}

func rewriteSurfaceGenerationBundle(t *testing.T, bundle []byte, change func(map[string][]byte)) []byte {
	t.Helper()
	fs, err := i18n.ReadTransportBundle(bundle, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	var m i18n.GenerationBundleManifest
	if err := json.Unmarshal(fs["manifest.json"], &m); err != nil {
		t.Fatal(err)
	}
	change(fs)
	entries := []i18n.TransportBundleEntry{}
	var identity strings.Builder
	for i, f := range m.Files {
		m.Files[i] = i18n.NewTransportBundleFile(f.BundlePath, f.RepositoryPath, fs[f.BundlePath])
		identity.WriteString(f.BundlePath + "\x00" + m.Files[i].SHA256 + "\x00")
		entries = append(entries, i18n.TransportBundleEntry{Path: f.BundlePath, Data: fs[f.BundlePath]})
	}
	m.InputIdentitySHA256 = digest([]byte(identity.String()))
	b, _ := json.MarshalIndent(m, "", "  ")
	result, err := i18n.WriteDeterministicTransportBundle(append(b, '\n'), entries)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
