package i18n

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestHistoricalPromotionRecoveryActivationContext(t *testing.T) {
	root, c, _ := historicalRecoveryFixture(t)
	before, err := readCompatibilityFile(root, "locales/zh-CN/content-scope.json")
	if err != nil {
		t.Fatal(err)
	}
	e, path, err := RecoverHistoricalPromotion(root, c, "zh-CN", "qc-002", true)
	if err != nil {
		t.Fatal(err)
	}
	var state contentidentity.Locale
	if err := json.Unmarshal(before, &state); err != nil {
		t.Fatal(err)
	}
	state.Packages = append(state.Packages, contentidentity.Completion{Package: "synthetic-prospective-package", State: "complete"})
	b, _ := json.MarshalIndent(state, "", "  ")
	writeSnapshotSource(t, root, e.HistoricalClosure.Path, append(b, '\n'))
	if _, err := verifyTargetSuccessor(root, path, c); err == nil {
		t.Fatal("changed closure accepted without exact historical context reader")
	}
	c.HistoricalContextReader = func(_ string, locale string, ref GlossaryArchiveReference) ([]byte, error) {
		if locale != "zh-CN" || ref != *e.HistoricalClosure {
			return nil, fmt.Errorf("not the frozen closure reference")
		}
		return before, nil // Representation proof is exercised in sitecontent.
	}
	if _, err := verifyTargetSuccessor(root, path, c); err != nil {
		t.Fatal("immutable successor could not use proven frozen closure bytes", err)
	}
	c.HistoricalContextReader = func(string, string, GlossaryArchiveReference) ([]byte, error) {
		return []byte("wrong historical bytes"), nil
	}
	if _, err := verifyTargetSuccessor(root, path, c); err == nil {
		t.Fatal("reader supplied bytes without the immutable reference SHA")
	}
}
