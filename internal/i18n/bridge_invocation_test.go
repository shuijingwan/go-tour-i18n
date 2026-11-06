package i18n

import "testing"

func TestBridgeInvocationNilCatalogFailsClosed(t *testing.T) {
	if _, _, err := readSameUnitBridgeForInvocation(t.TempDir(), nil, "zh-CN", GlossaryArchiveReference{Path: "missing.json"}); err == nil {
		t.Fatal("nil Catalog accepted")
	}
}
