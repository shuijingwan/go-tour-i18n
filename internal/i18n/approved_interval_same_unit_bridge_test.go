package i18n

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type approvedBridgeProofTestAdapter struct{}

func (approvedBridgeProofTestAdapter) ApprovedIntervalBridgeCatalog(c *Catalog) *Catalog { return c }

func (approvedBridgeProofTestAdapter) ProtectedInput(*Glossary) (string, []string, error) {
	return "", nil, nil
}
func (approvedBridgeProofTestAdapter) Restore(s string) ([]byte, error) { return []byte(s), nil }
func (approvedBridgeProofTestAdapter) Validate([]byte, *Glossary) error { return nil }
func (approvedBridgeProofTestAdapter) CompareApprovedIntervalBridge(a, b ApprovedIntervalProjection) error {
	if !reflect.DeepEqual(a, b) {
		return fmt.Errorf("semantic projection changed")
	}
	return nil
}
func (approvedBridgeProofTestAdapter) BridgeApprovedIntervalRevision(ApprovedIntervalProjection, []byte, []byte) (TranslationUnitAdapter, []byte, []byte, error) {
	return nil, nil, nil, fmt.Errorf("proof fixture never constructs a decoder")
}

func TestApprovedIntervalSameUnitBridgeProofBoundary(t *testing.T) {
	u := &TranslationUnit{ID: "synthetic-page", Kind: UnitKindPage, SourceSHA256: sum([]byte("canonical source")), Adapter: approvedBridgeProofTestAdapter{}}
	old := ApprovedIntervalTransport{Schema: ApprovedIntervalTransportSchema, Mode: ApprovedIntervalMode, Record: RetranslationBatchUnit{UnitID: u.ID}, CurrentAuthority: sum([]byte("old validator")), Projection: ApprovedIntervalProjection{UnitID: u.ID, SourceSHA256: u.SourceSHA256, ParentSHA256: sum([]byte("immutable parent"))}, Dependencies: []GlossaryArchiveReference{{approvedIntervalImplementationPaths[0], sum([]byte("old implementation"))}, {"locales/ja-JP/glossary.yaml", sum([]byte("language facts"))}}}
	old.Identity = approvedIntervalIdentity(old)
	current := old
	current.Dependencies = append([]GlossaryArchiveReference(nil), old.Dependencies...)
	current.Dependencies[0].SHA256 = sum([]byte("current implementation"))
	current.CurrentAuthority = sum([]byte("current validator"))
	current.Identity = approvedIntervalIdentity(current)
	if err := sameUnitApprovedProofEquivalent(u, &old, &current); err != nil {
		t.Fatal("implementation-only drift", err)
	}
	for name, edit := range map[string]func(*ApprovedIntervalTransport){
		"semantic-parent":     func(p *ApprovedIntervalTransport) { p.Projection.ParentSHA256 = sum([]byte("changed parent")) },
		"Unit":                func(p *ApprovedIntervalTransport) { p.Record.UnitID = "another" },
		"source":              func(p *ApprovedIntervalTransport) { p.Projection.SourceSHA256 = sum([]byte("changed source")) },
		"Snapshot":            func(p *ApprovedIntervalTransport) { p.Snapshot.Attempt++ },
		"input":               func(p *ApprovedIntervalTransport) { p.Record.InputSHA256 = sum([]byte("changed input")) },
		"semantic-authority":  func(p *ApprovedIntervalTransport) { p.SemanticAuthority.Path = "other.json" },
		"language-dependency": func(p *ApprovedIntervalTransport) { p.Dependencies[1].SHA256 = sum([]byte("changed language")) },
		"dependency-path":     func(p *ApprovedIntervalTransport) { p.Dependencies[0].Path = "unowned.go" },
		"malformed-identity":  func(p *ApprovedIntervalTransport) { p.CurrentAuthority = "invalid" },
	} {
		t.Run(name, func(t *testing.T) {
			bad := current
			bad.Dependencies = append([]GlossaryArchiveReference(nil), current.Dependencies...)
			edit(&bad)
			bad.Identity = approvedIntervalIdentity(bad)
			if err := sameUnitApprovedProofEquivalent(u, &old, &bad); err == nil {
				t.Fatal("resealed semantic/dependency tamper accepted")
			}
		})
	}
	t.Run("unsupported-adapter", func(t *testing.T) {
		unsupported := *u
		unsupported.Adapter = nil
		files := map[string][]byte{approvedIntervalProofMember(u.ID): mustJSON(old)}
		if _, _, _, err := approvedSameUnitBridgeOutput(&Catalog{}, &unsupported, old.Record, nil, nil, files); err == nil || !strings.Contains(err.Error(), "lacks approved-interval") {
			t.Fatal(err)
		}
	})
	t.Run("proof-tamper", func(t *testing.T) {
		bad := old
		bad.Projection.ParentSHA256 = sum([]byte("tampered"))
		if bytes.Equal(mustJSON(old), mustJSON(bad)) {
			t.Fatal("fixture unchanged")
		}
		files := map[string][]byte{approvedIntervalProofMember(u.ID): mustJSON(bad)}
		if _, _, _, err := approvedSameUnitBridgeOutput(&Catalog{}, u, old.Record, nil, nil, files); err == nil || !strings.Contains(err.Error(), "proof identity") {
			t.Fatal(err)
		}
	})
}
