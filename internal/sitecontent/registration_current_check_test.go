package sitecontent

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Reuse the complete registered/adopted fixture already constructed by the
// registration test. Only this TempDir gets a new batch/transport; production
// batch062 and its Bundle are never exported or copied into mutable setup.
func testRegistrationCurrentCheckProcesses(t *testing.T, root string, c *i18n.Catalog) {
	t.Helper()
	if err := c.UseTranslationPackage("learn-docs-v1"); err != nil {
		t.Fatal(err)
	}
	u := c.TranslationScope[0]
	e, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, BatchID: "chatgpt-zh-CN-901", UnitKind: i18n.UnitKindPage, UnitIDs: []string{u}})
	if err != nil {
		t.Fatal("fixture pending batch", err)
	}
	bundle, expected, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID})
	if err != nil {
		t.Fatal("fixture transport", err)
	}
	writeFixture(t, root, "engineering/current.zip", bundle)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	for phase := 0; phase < 2; phase++ {
		cmd := exec.Command(os.Args[0], "-test.run=^TestRegistrationInvocationAuthorityReuse$", "-test.count=1", "-test.v")
		cmd.Env = append(os.Environ(), "GO_TOUR_CURRENT_SOURCE_ROOT="+root)
		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(string(b), err)
		}
		found := false
		for _, line := range strings.Split(string(b), "\n") {
			at := strings.Index(line, "CURRENT_SOURCE=")
			if at < 0 {
				continue
			}
			var row struct {
				Identity                                string
				Builds, ManifestCaptures, CaptureReuses int
			}
			if err := json.Unmarshal([]byte(line[at+len("CURRENT_SOURCE="):]), &row); err != nil {
				t.Fatal(err)
			}
			builds := 1
			if phase == 1 {
				builds = 0
			}
			if row.Identity != expected.InputIdentitySHA256 || row.Builds != builds || row.ManifestCaptures != 1 || row.CaptureReuses == 0 {
				t.Fatal("exact current-check/owner cold-warm identity changed", phase, row)
			}
			t.Logf("CURRENT_CHECK_SOURCE phase=%d builds=%d manifest_captures=%d capture_reuses=%d identity=%s", phase, row.Builds, row.ManifestCaptures, row.CaptureReuses, row.Identity)
			found = true
		}
		if !found {
			t.Fatal(string(b))
		}
	}
}

func registrationCurrentCheckChild(t *testing.T, root string) {
	t.Helper()
	finish := contentidentity.BeginMemoization()
	defer finish()
	c, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	var owned *structuredSuccessorValidation
	// Same owner + core call as AttachRegisteredTranslationPackagesReadonly;
	// metrics are test-only and do not alter the phase/guard policy.
	err = withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		owned = v
		v.historicalMetrics = map[string]int{}
		return AttachRegisteredTranslationPackages(root, c, v)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage("learn-docs-v1"); err != nil {
		t.Fatal(err)
	}
	bundle, err := readRegular(root, "engineering/current.zip")
	if err != nil {
		t.Fatal(err)
	}
	current, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle)
	if err != nil {
		t.Fatal("exact public current-check", err)
	}
	b, _ := json.Marshal(struct {
		Identity                                string
		Builds, ManifestCaptures, CaptureReuses int
	}{current.InputIdentitySHA256, owned.historicalMetrics["global-environment-build"], owned.historicalMetrics["manifest-file-observation"], owned.historicalMetrics["source-path-capture-reuse"]})
	t.Logf("CURRENT_SOURCE=%s", b)
}
