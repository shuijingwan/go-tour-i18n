package sitecontent

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

type unknownBridgePage struct{ PageAdapter }

func TestBridgeInvocationApprovedDecoderOwnership(t *testing.T) {
	a, _, _, _ := bridgeGraphInputs(t)
	root := t.TempDir()
	c := &i18n.Catalog{Adapted: []i18n.AdaptedTranslationUnit{{Adapter: a}}, DerivedTransportCodecReader: derivedApprovedCodec}
	AttachPageReplacementInvocation(root, c)
	ref := i18n.GlossaryArchiveReference{Path: "receipt.json", SHA256: "identity"}
	calls := 0
	build := func() (*i18n.SameUnitBridgeResolution, error) {
		calls++
		p := &approvedOrdinaryBridge{
			old:  cloneReplacementValue(reflect.ValueOf(a)).Interface().(PageAdapter),
			text: "template", tokens: []string{"token"}, parentToken: "token", parent: "parent", left: "left", right: "right",
		}
		return &i18n.SameUnitBridgeResolution{Adapters: map[string]i18n.TranslationUnitAdapter{"unit": p}}, nil
	}
	first, err := c.BridgeResolver(root, c, "historical", ref, build)
	if err != nil {
		t.Fatal(err)
	}
	p := first.Adapters["unit"].(*approvedOrdinaryBridge)
	p.old.Source[0] = '!'
	p.old.Document.Slots[0].Source = "consumer mutation"
	p.tokens[0], p.text, p.parentToken, p.parent, p.left, p.right = "changed", "changed", "changed", "changed", "changed", "changed"
	second, err := c.BridgeResolver(root, c, "historical", ref, build)
	if err != nil || calls != 1 {
		t.Fatal("approved decoder not reused", calls, err)
	}
	p = second.Adapters["unit"].(*approvedOrdinaryBridge)
	if !reflect.DeepEqual(p.old, a) || p.text != "template" || p.tokens[0] != "token" || p.parentToken != "token" || p.parent != "parent" || p.left != "left" || p.right != "right" {
		t.Fatal("approved decoder aliases consumer data", p)
	}
	unknownCalls := 0
	for i := 0; i < 2; i++ {
		_, err := c.BridgeResolver(root, c, "current", ref, func() (*i18n.SameUnitBridgeResolution, error) {
			unknownCalls++
			return &i18n.SameUnitBridgeResolution{Adapters: map[string]i18n.TranslationUnitAdapter{"unit": unknownBridgePage{a}}}, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if unknownCalls != 2 {
		t.Fatal("unknown output adapter admitted", unknownCalls)
	}
}

func TestBridgeInvocationReconstructionReuse(t *testing.T) {
	source := []byte("# Heading\n\nOrdinary prose.\n")
	d, err := ParseWorkflowDocument(Source{Path: "_content/doc/bridge.md", Kind: "page", Route: "/doc/bridge", SHA256: digest(source)}, source, 1)
	if err != nil {
		t.Fatal(err)
	}
	c := &i18n.Catalog{Adapted: []i18n.AdaptedTranslationUnit{{Adapter: PageAdapter{*d, source}}}}
	// Registered package Catalogs carry this source-owned reader. A nonnil
	// function must not silently disable exact-data reconstruction reuse.
	c.DerivedTransportCodecReader = derivedApprovedCodec
	root := t.TempDir()
	AttachPageReplacementInvocation(root, c)
	ref := i18n.GlossaryArchiveReference{Path: "receipt.json", SHA256: "identity"}
	calls := 0
	build := func() (*i18n.SameUnitBridgeResolution, error) {
		calls++
		return &i18n.SameUnitBridgeResolution{Receipt: &i18n.QualityCheckSurfaceReopen{Locale: "fixture"}, Adapters: map[string]i18n.TranslationUnitAdapter{"unit": cloneReplacementValue(reflect.ValueOf(PageAdapter{*d, source})).Interface().(PageAdapter)}}, nil
	}
	first, err := c.BridgeResolver(root, c, "current", ref, build)
	if err != nil {
		t.Fatal(err)
	}
	// Mutating the consumer's result must not mutate the private memo.
	first.Receipt.Locale = "changed"
	a := first.Adapters["unit"].(PageAdapter)
	a.Source[0] = '!'
	second, err := c.BridgeResolver(root, c, "current", ref, build)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || second.Receipt.Locale != "fixture" || !bytes.Equal(second.Adapters["unit"].(PageAdapter).Source, source) {
		t.Fatal("reconstruction was not isolated/reused", calls)
	}
	private := second.Adapters["unit"].(PageAdapter)
	private.Document.Slots[0].Source = "changed"
	third, err := c.BridgeResolver(root, c, "current", ref, build)
	if err != nil || reflect.DeepEqual(third.Adapters["unit"], private) {
		t.Fatal("adapter result aliases private mapping", err)
	}
	changed := *c
	changed.TranslationScope = []string{"different"}
	if _, err := c.BridgeResolver(root, &changed, "current", ref, build); err != nil || calls != 2 {
		t.Fatal("typed scope drift reused reconstruction", calls, err)
	}
	AttachPageReplacementInvocation(root, c)
	if _, err := c.BridgeResolver(root, c, "current", ref, build); err != nil || calls != 3 {
		t.Fatal("cross invocation reuse", calls, err)
	}
	if _, err := c.BridgeResolver(t.TempDir(), c, "current", ref, build); err == nil {
		t.Fatal("wrong root accepted")
	}
	if _, err := c.BridgeResolver(root, c, "historical", ref, build); err != nil || calls != 4 {
		t.Fatal("historical reconstruction domain", calls, err)
	}
	if _, err := c.BridgeResolver(root, c, "current", ref, build); err != nil || calls != 4 {
		t.Fatal("historical result contaminated current authority", calls, err)
	}
	foreign := *c
	foreign.Adapted = []i18n.AdaptedTranslationUnit{{Adapter: unknownBridgePage{PageAdapter{*d, source}}}}
	// Even our private construction wrapper must not admit an unknown original
	// representation just because it embeds a known Page adapter.
	wrapped := bridgeRepresentationCatalog(&foreign, successorValidation(root, nil), root)
	for i := 0; i < 2; i++ {
		if _, err := c.BridgeResolver(root, wrapped, "historical", ref, build); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 6 {
		t.Fatal("unknown wrapped adapter admitted", calls)
	}
	withoutCodec := *c
	withoutCodec.DerivedTransportCodecReader = nil
	if _, err := c.BridgeResolver(root, &withoutCodec, "current", ref, build); err != nil || calls != 7 {
		t.Fatal("codec presence omitted from data key", calls, err)
	}
	unknownCodec := *c
	unknownCodec.DerivedTransportCodecReader = func(a i18n.TranslationUnitAdapter, codec, inventory []byte) (any, any, i18n.TranslationUnitAdapter, error) {
		return derivedApprovedCodec(a, codec, inventory)
	}
	for i := 0; i < 2; i++ {
		if _, err := c.BridgeResolver(root, &unknownCodec, "current", ref, build); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 9 {
		t.Fatal("unknown codec reader admitted", calls)
	}
}

// The private adopted wrapper adds a root and a live corpus comparison, not a
// different representation factory. A memo hit must repeat that comparison;
// unsupported domains and failed preparation retain the native builder result.
func TestBridgeInvocationAdoptedAdmission(t *testing.T) {
	root, _, _, _, _ := historicalGlobalSuccessorFixture(t)
	corpus, err := readRegular(root, i18n.GlossaryCorpusPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AdoptCorpusProjection(root, digest(corpus), true); err != nil {
		t.Fatal(err)
	}
	page, _, _, _ := bridgeGraphInputs(t)
	c := &i18n.Catalog{Adapted: []i18n.AdaptedTranslationUnit{{Adapter: adoptedPageAdapter{page, root}}}}
	v := newClosureValidation(root)
	v.metrics = map[string]int{}
	ref := i18n.GlossaryArchiveReference{Path: "receipt.json", SHA256: "identity"}
	calls := 0
	var nativeErr error
	build := func() (*i18n.SameUnitBridgeResolution, error) {
		calls++
		if nativeErr != nil {
			return nil, nativeErr
		}
		return &i18n.SameUnitBridgeResolution{Receipt: &i18n.QualityCheckSurfaceReopen{Locale: "fixture"}, Adapters: map[string]i18n.TranslationUnitAdapter{"unit": page}}, nil
	}
	first, err := v.resolveBridgeReconstruction(root, c, "current", ref, build)
	if err != nil {
		t.Fatal(err)
	}
	second, err := v.resolveBridgeReconstruction(root, c, "current", ref, build)
	if err != nil || calls != 1 || !reflect.DeepEqual(first, second) {
		t.Fatal("adopted exact reconstruction not reused", calls, err)
	}
	second.Receipt.Locale = "consumer mutation"
	third, err := v.resolveBridgeReconstruction(root, c, "current", ref, build)
	if err != nil || !reflect.DeepEqual(first, third) || calls != 1 {
		t.Fatal("private result ownership", err)
	}
	decorated := bridgeRepresentationCatalog(c, successorValidation(root, nil), root)
	if got, err := v.resolveBridgeReconstruction(root, decorated, "current", ref, build); err != nil || calls != 1 || !reflect.DeepEqual(first, got) {
		t.Fatal("readonly decorator bypassed immutable owner", calls, err)
	}
	for _, domain := range []string{"historical", "historical"} {
		if got, err := v.resolveBridgeReconstruction(root, decorated, domain, ref, build); err != nil || !reflect.DeepEqual(first, got) {
			t.Fatal(err)
		}
	}
	if calls != 2 {
		t.Fatal("historical decoder not shared in separate domain", calls)
	}
	foreign := *c
	foreign.Adapted = []i18n.AdaptedTranslationUnit{{Adapter: adoptedPageAdapter{page, t.TempDir()}}}
	for i := 0; i < 2; i++ {
		if _, err := v.resolveBridgeReconstruction(root, &foreign, "current", ref, build); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 4 {
		t.Fatal("foreign root admitted", calls)
	}
	changed := *c
	changed.TranslationScope = []string{"changed"}
	if _, err := v.resolveBridgeReconstruction(root, &changed, "current", ref, build); err != nil || calls != 5 {
		t.Fatal("scope identity omitted", calls, err)
	}
	for _, name := range []string{i18n.GlossaryCorpusPath, GlobalPath, "_content/article.tmpl"} {
		t.Run(name, func(t *testing.T) {
			before, err := readRegular(root, name)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, name, before)
			writeFixture(t, root, name, append(bytes.Clone(before), ' '))
			for _, mode := range []string{"current", "historical"} {
				nativeErr = errors.New("native builder failure")
				beforeCalls := calls
				if _, err := v.resolveBridgeReconstruction(root, decorated, mode, ref, build); err != nativeErr || calls != beforeCalls+1 {
					t.Fatal("fresh drift did not fall back to exact native failure", mode, calls, err)
				}
				nativeErr = nil
				// No invented rejection if the native builder's own scope does
				// not consume the unavailable preparation dependency.
				if _, err := v.resolveBridgeReconstruction(root, decorated, mode, ref, build); err != nil || calls != beforeCalls+2 {
					t.Fatal("native success changed", mode, calls, err)
				}
			}
		})
	}
	beforeCalls := calls
	fresh := newClosureValidation(root)
	if _, err := fresh.resolveBridgeReconstruction(root, c, "current", ref, build); err != nil || calls != beforeCalls+1 {
		t.Fatal("memo escaped invocation", err)
	}
	// The actual nested construction already owns its source observation.
	// Hits must enter that same canonical gate rather than create eager owners.
	err = withReadonlyStructuredValidation(root, func(proof *structuredSuccessorValidation) error {
		proof.sharedSourceObservation, proof.currentSourcePhase = true, true
		defer func() { proof.sharedSourceObservation, proof.currentSourcePhase = false, false }()
		proof.historicalMetrics = map[string]int{}
		wrapped := bridgeRepresentationCatalog(c, proof, root)
		owner := newClosureValidation(root)
		beforeCalls := calls
		for i := 0; i < 3; i++ {
			if _, err := owner.resolveBridgeReconstruction(root, wrapped, "historical", ref, build); err != nil {
				return err
			}
		}
		if calls != beforeCalls+1 || proof.historicalMetrics["current-source-projection-build"] != 1 {
			t.Fatal("nested hit escaped its native source proof owner", calls-beforeCalls, proof.historicalMetrics)
		}
		before, err := readRegular(root, "_content/article.tmpl")
		if err != nil {
			return err
		}
		defer writeFixture(t, root, "_content/article.tmpl", before)
		writeFixture(t, root, "_content/article.tmpl", append(bytes.Clone(before), ' '))
		// The function under test shares capture inside this phase; prove the
		// same outer guard still observes concurrent bytes before escape.
		if err := proof.aggregateVerify(); err == nil {
			t.Fatal("shared owner hid concurrent source drift")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ADOPTED_BRIDGE current_builds=1 current_repeat_builds=0 historical_builds=1 historical_repeat_builds=0 foreign_native=2 drift_fallbacks=12")
}
