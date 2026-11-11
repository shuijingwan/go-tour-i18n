package i18n

import (
	"strings"
	"testing"
)

func TestCurrentValidationSameUnitBridgeGuards(t *testing.T) {
	r := RetranslationBatchUnit{UnitID: "page", InputPath: "inputs/page.article", SourcePath: "_content/page.md", InputSHA256: sum([]byte("formal"))}
	old := CurrentValidationPredecessor{Schema: "go-learning/current-validation-predecessor-transport/v1", Locale: "zh-CN", UnitID: r.UnitID, SourcePath: r.SourcePath, CurrentInputSHA256: r.InputSHA256, RecoveryClass: "DETERMINISTIC_REBASE_PASS", Template: "template", TemplateSHA256: sum([]byte("template"))}
	for name, edit := range map[string]func(*CurrentValidationPredecessor){
		"schema":         func(p *CurrentValidationPredecessor) { p.Schema = "unknown" },
		"locale":         func(p *CurrentValidationPredecessor) { p.Locale = "ja-JP" },
		"Unit":           func(p *CurrentValidationPredecessor) { p.UnitID = "different" },
		"source":         func(p *CurrentValidationPredecessor) { p.SourcePath = "other.md" },
		"input":          func(p *CurrentValidationPredecessor) { p.CurrentInputSHA256 = sum([]byte("different")) },
		"recovery-class": func(p *CurrentValidationPredecessor) { p.RecoveryClass = "OTHER_FAIL_CLOSED" },
		"empty-template": func(p *CurrentValidationPredecessor) { p.Template = "" },
		"template-SHA":   func(p *CurrentValidationPredecessor) { p.TemplateSHA256 = sum([]byte("different")) },
	} {
		t.Run(name, func(t *testing.T) {
			bad := old
			edit(&bad)
			_, _, _, err := currentValidationSameUnitBridgeOutput(t.TempDir(), nil, nil, "zh-CN", r, nil, nil, map[string][]byte{currentValidationBridgeMember(r): mustJSON(bad)}, nil, nil)
			if err == nil || !strings.Contains(err.Error(), "identity/template mismatch") {
				t.Fatal("malformed cvpred did not fail before construction", err)
			}
		})
	}
	t.Run("ambiguous-members", func(t *testing.T) {
		for _, other := range []string{surfaceBridgeMember(r.UnitID), approvedIntervalProofMember(r.UnitID), commentTransportPath(r)} {
			err := sameUnitBridgeRepresentationUnambiguous(map[string][]byte{currentValidationBridgeMember(r): mustJSON(old), other: []byte("present")}, r)
			if err == nil || !strings.Contains(err.Error(), "representation is ambiguous") {
				t.Fatal(other, err)
			}
		}
	})
	t.Run("ordinary-capability", func(t *testing.T) {
		normal := cvpredOrdinaryTestDecoder{}
		if err := sameUnitPredecessorOrdinaryDecoder(normal, normal); err != nil {
			t.Fatal(err)
		}
		if err := sameUnitPredecessorOrdinaryDecoder(normal, cvpredRestrictedDecoder{}); err == nil {
			t.Fatal("restricted decoder accepted")
		}
		if err := sameUnitPredecessorOrdinaryDecoder(normal, nil); err == nil {
			t.Fatal("missing decoder accepted")
		}
		if err := sameUnitPredecessorOrdinaryDecoder(normal, cvpredUnknownDecoder{}); err == nil {
			t.Fatal("unknown decoder accepted")
		}
	})
}

type cvpredRestrictedDecoder struct{ cvpredOrdinaryTestDecoder }

func (cvpredRestrictedDecoder) UnrestrictedRevisionTransport() bool { return false }

type cvpredUnknownDecoder struct{ cvpredOrdinaryTestDecoder }

type cvpredOrdinaryTestDecoder struct{}

func (cvpredOrdinaryTestDecoder) ProtectedInput(*Glossary) (string, []string, error) {
	return "", nil, nil
}
func (cvpredOrdinaryTestDecoder) Restore(s string) ([]byte, error) { return []byte(s), nil }
func (cvpredOrdinaryTestDecoder) Validate([]byte, *Glossary) error { return nil }
