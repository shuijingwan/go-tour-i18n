package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"testing"
)

// The XML fixture has a registered, source-owned two-field representation.
// This provides its adapter boundary only; the production capability builder,
// immutable writer, finding/predecessor proof and current reader run unchanged.
func (a futureCardAdapter) WithCurrentSurfaceTransportContext(root string, c *Catalog, work func(string) error) error {
	observe := func() (string, error) {
		type unit struct {
			Package, Authority, ID, Path, SHA, Route string
			Index                                    int
		}
		var registry []unit
		for _, p := range c.TranslationPackages {
			for _, b := range p.Units {
				raw, err := readSnapshotReferencedFile(root, b.Unit.SourcePath)
				if err != nil || sum(raw) != b.Unit.SourceSHA256 || !bytes.Equal(raw, b.Unit.Source) {
					return "", fmt.Errorf("fixture registered source drift: %s: %v", b.Unit.ID, err)
				}
				registry = append(registry, unit{p.ID, p.AuthoritySHA256, b.Unit.ID, b.Unit.SourcePath, b.Unit.SourceSHA256, b.Route, b.StableIndex})
			}
		}
		if len(registry) == 0 {
			return "", fmt.Errorf("fixture requires registered source authority")
		}
		return sum(mustJSON(struct {
			Registry []unit
			Scope    []string
		}{registry, c.TranslationScope})), nil
	}
	before, err := observe()
	if err != nil {
		return err
	}
	if err := work(before); err != nil {
		return err
	}
	after, err := observe()
	if err != nil {
		return err
	}
	if before != after {
		return fmt.Errorf("fixture registration changed during Surface proof")
	}
	return nil
}

type testCardSurfaceCodec struct {
	Schema string     `json:"schema"`
	Source futureCard `json:"source"`
}

func (a futureCardAdapter) SurfaceTransportProjection(origin TranslationUnitAdapter, raw []byte, regions []string) (SurfaceTransportProjection, TranslationUnitAdapter, error) {
	old, ok := origin.(futureCardAdapter)
	if !ok || !reflect.DeepEqual(old.source, a.source) {
		return SurfaceTransportProjection{}, nil, fmt.Errorf("fixture Surface origin/source mismatch")
	}
	// The known source token mapping restores the frozen passed response; never
	// reverse-protect the candidate or infer regions from translated wording.
	if _, err := old.Restore(string(raw)); err != nil {
		return SurfaceTransportProjection{}, nil, err
	}
	prefix := sum(mustJSON(a.source))
	available := []SurfaceTransportRegion{
		{ID: prefix + "/heading", Kind: "ordinary-prose", SourceCoordinate: "heading", Preview: a.source.Heading},
		{ID: prefix + "/message", Kind: "ordinary-prose", SourceCoordinate: "message", Preview: a.source.Message},
	}
	p := SurfaceTransportProjection{Kind: "ordinary", Available: available}
	if len(regions) == 0 {
		return p, nil, nil
	}
	valid := map[string]bool{}
	for _, r := range available {
		valid[r.ID] = true
	}
	seen := map[string]bool{}
	for _, r := range regions {
		if !valid[r] || seen[r] {
			return p, nil, fmt.Errorf("fixture Surface region outside exact source-owned set")
		}
		seen[r] = true
	}
	p.Editable = append([]string(nil), regions...)
	sort.Strings(p.Editable)
	p.Template = string(raw)
	p.Codec = mustJSON(testCardSurfaceCodec{"test-only/card-surface-codec/v1", a.source})
	// Ordinary source representation already expresses these prose regions;
	// retain the original raw/decoder, just as the production ordinary branch.
	return p, old, nil
}

func (a futureCardAdapter) ReadSurfaceTransportProjection(raw json.RawMessage) (TranslationUnitAdapter, error) {
	var codec testCardSurfaceCodec
	if err := decodeStrictBundleJSON(raw, &codec); err != nil {
		return nil, err
	}
	if codec.Schema != "test-only/card-surface-codec/v1" || !reflect.DeepEqual(codec.Source, a.source) || !bytes.Equal(raw, mustJSON(codec)) {
		return nil, fmt.Errorf("fixture Surface codec/source identity mismatch")
	}
	return futureCardAdapter{codec.Source}, nil
}

func recordTestSurfaceTransport(t *testing.T, root string, c *Catalog, locale, batch, coordinate string) *SurfaceTransportPlan {
	t.Helper()
	if err := languageAuditFixtureRootError(root, repoRoot(t)); err != nil {
		t.Fatal(err)
	}
	discovery, err := PrepareSurfaceRevisionTransport(root, c, locale, batch, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !discovery.CurrentInputs || discovery.RevisionTransportCapable || discovery.Recorded {
		t.Fatal("discovery granted capability", discovery)
	}
	request := discovery.Authority.Request
	for _, b := range discovery.Authority.Bindings {
		q := SurfaceTransportUnitRequest{UnitID: b.UnitID, FindingSHA256: b.FindingSHA256}
		for _, r := range b.Projection.Available {
			if r.SourceCoordinate == coordinate {
				q.Regions = append(q.Regions, r.ID)
			}
		}
		if len(q.Regions) != 1 {
			t.Fatal("explicit fixture source coordinate missing/ambiguous", coordinate)
		}
		request.Units = append(request.Units, q)
	}
	// The request uses exact discovered coordinates and persisted finding SHA.
	// Malformed mappings must still fail before any authority publication.
	wrong := request
	wrong.Units = append([]SurfaceTransportUnitRequest(nil), request.Units...)
	wrong.Units[0].FindingSHA256 = sum([]byte("wrong finding"))
	if _, err := PrepareSurfaceRevisionTransport(root, c, locale, batch, &wrong, false); err == nil {
		t.Fatal("wrong finding gained Surface capability")
	}
	plan, err := PrepareSurfaceRevisionTransport(root, c, locale, batch, &request, true)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.CurrentInputs || !plan.RevisionTransportCapable || !plan.Recorded || plan.Mutation != "known" {
		t.Fatal("capability publication incomplete", plan)
	}
	if _, _, err := readSurfaceTransport(root, c, locale, "surface-fix-001"); err != nil {
		t.Fatal("current capability rejected", err)
	}
	return plan
}
