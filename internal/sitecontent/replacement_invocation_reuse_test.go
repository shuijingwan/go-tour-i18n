package sitecontent

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestReplacementReconciliationInvocationReuse(t *testing.T) {
	root, _, c, frozen := replacementPageFixture(t)
	ref := freezeReplacementFixture(t, root, frozen)
	want, err := resolvePageReplacements(root, c, ref)
	if err != nil {
		t.Fatal(err)
	}
	v := newClosureValidation(root)
	v.verifiedBytesSize = 64 << 20 // real source inventory can exhaust byte admission
	v.attachReplacementReuse(c)
	for i := 0; i < 2; i++ {
		got, err := c.ReplacementResolver(root, c, ref)
		if err != nil || !reflect.DeepEqual(want.Units, got.Units) || !reflect.DeepEqual(want.HistoricalCatalog.TranslationPackages, got.HistoricalCatalog.TranslationPackages) {
			t.Fatal("resolution changed", err)
		}
		// Output values are decoded/reconstructed anew, never aliases of memo bytes.
		got.Units["tampered"] = i18n.TranslationUnitReplacement{}
		adapter := got.HistoricalCatalog.Adapted[2].Adapter.(PageAdapter)
		adapter.Source[0] = 'X'
	}
	if len(v.reconciliations) != 1 {
		t.Fatal("expected one exact pure calculation", len(v.reconciliations))
	}
	if _, err := c.ReplacementResolver(root, c, i18n.GlossaryArchiveReference{Path: ref.Path, SHA256: strings.Repeat("0", 64)}); err == nil {
		t.Fatal("memo bypassed expected SHA")
	}
	b, err := readRegular(root, ref.Path)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, ref.Path, append(bytes.Clone(b), ' '))
	if _, err := c.ReplacementResolver(root, c, ref); err == nil {
		t.Fatal("memo bypassed reference drift")
	}
	writeFixture(t, root, ref.Path, b)
	// JSON omitempty collapses this distinction; the typed memo key must not.
	unit := &c.TranslationPackages[0].Units[2]
	adapter := unit.Adapter.(PageAdapter)
	adapter.Document.ReferenceDefinitions = []MarkdownReferenceDefinition{}
	unit.Adapter = adapter
	if _, err := c.ReplacementResolver(root, c, ref); err == nil {
		t.Fatal("memo collapsed nil and empty representation fields")
	}
	adapter.Document.ReferenceDefinitions = nil
	unit.Adapter = adapter
	// The key includes exact current representation and source, not package ID.
	descriptor := &c.TranslationPackages[0]
	a := descriptor.Units[2].Adapter.(PageAdapter)
	a.Document.Route = "/changed"
	descriptor.Units[2].Adapter = a
	if _, err := c.ReplacementResolver(root, c, ref); err == nil {
		t.Fatal("memo bypassed current representation drift")
	}
	if len(newClosureValidation(root).reconciliations) != 0 {
		t.Fatal("memo escaped invocation")
	}
}

func TestPageReplacementCommandInvocation(t *testing.T) {
	root, _, c, frozen := replacementPageFixture(t)
	ref := freezeReplacementFixture(t, root, frozen)
	AttachPageReplacementInvocation(root, c)
	first, err := c.ReplacementResolver(root, c, ref)
	if err != nil {
		t.Fatal(err)
	}
	first.Units["caller-mutation"] = i18n.TranslationUnitReplacement{}
	again, err := c.ReplacementResolver(root, c, ref)
	if err != nil || len(first.Units) == len(again.Units) {
		t.Fatal("projection aliases caller", err)
	}
	b, err := readRegular(root, ref.Path)
	if err != nil {
		t.Fatal(err)
	}
	changed := append(bytes.Clone(b), '\n')
	writeFixture(t, root, ref.Path, changed)
	if _, err := c.ReplacementResolver(root, c, ref); err == nil {
		t.Fatal("warm resolver accepted changed disk bytes")
	}
	// A new invocation starts from the new exact reference, never yesterday's
	// observed-byte graph. JSON whitespace is a valid new frozen representation.
	next := ref
	next.SHA256 = digest(changed)
	AttachPageReplacementInvocation(root, c)
	if _, err := c.ReplacementResolver(root, c, next); err != nil {
		t.Fatal("invocation retained old dependency state", err)
	}
	if _, err := c.ReplacementResolver(t.TempDir(), c, next); err == nil {
		t.Fatal("resolver changed root")
	}
}

func TestReplacementInvocationReuseWithAdoptedPages(t *testing.T) {
	root, _, c, frozen := replacementPageFixture(t)
	ref := freezeReplacementFixture(t, root, frozen)
	want, err := resolvePageReplacements(root, c, ref)
	if err != nil {
		t.Fatal(err)
	}
	for i := range c.Adapted {
		a := c.Adapted[i].Adapter.(PageAdapter)
		c.Adapted[i].Adapter = adoptedPageAdapter{PageAdapter: a, root: root}
	}
	for i := range c.TranslationPackages[0].Units {
		a := c.TranslationPackages[0].Units[i].Adapter.(PageAdapter)
		c.TranslationPackages[0].Units[i].Adapter = adoptedPageAdapter{PageAdapter: a, root: root}
	}
	v := newClosureValidation(root)
	v.metrics = map[string]int{}
	v.attachReplacementReuse(c)
	for i := 0; i < 3; i++ {
		got, err := c.ReplacementResolver(root, c, ref)
		if err != nil || !reflect.DeepEqual(got.Units, want.Units) || !reflect.DeepEqual(got.HistoricalCatalog.TranslationPackages, want.HistoricalCatalog.TranslationPackages) {
			t.Fatal("adopted adapter changed exact projection", err)
		}
		got.Units["caller mutation"] = i18n.TranslationUnitReplacement{}
		a := got.HistoricalCatalog.Adapted[0].Adapter.(PageAdapter)
		a.Source[0] = 'X'
	}
	if v.metrics["replacement-resolution-build:"+ref.Path] != 1 || v.metrics["replacement-resolution-reuse:"+ref.Path] != 2 {
		t.Fatal("owned adopted wrapper disabled invocation reuse", v.metrics)
	}
	// The root remains part of the exact input, even though pure source
	// reconciliation does not use ProtectedInput's current eligibility gate.
	a := c.Adapted[0].Adapter.(adoptedPageAdapter)
	a.root = t.TempDir()
	c.Adapted[0].Adapter = a
	if _, err := c.ReplacementResolver(root, c, ref); err != nil {
		t.Fatal(err)
	}
	if v.metrics["replacement-resolution-build:"+ref.Path] != 2 {
		t.Fatal("private root omitted from key")
	}
	if _, _, err := a.ProtectedInput(&i18n.Glossary{}); err == nil {
		t.Fatal("pure projection cache bypassed current corpus eligibility")
	}
	if _, err := c.ReplacementResolver(root, c, i18n.GlossaryArchiveReference{Path: ref.Path, SHA256: strings.Repeat("0", 64)}); err == nil {
		t.Fatal("wrong SHA reused")
	}
	b, _ := readRegular(root, ref.Path)
	writeFixture(t, root, ref.Path, append(b, ' '))
	if _, err := c.ReplacementResolver(root, c, ref); err == nil {
		t.Fatal("changed representation reused")
	}
	if err := v.verify(); err == nil {
		t.Fatal("concurrent dependency mutation escaped")
	}
}
