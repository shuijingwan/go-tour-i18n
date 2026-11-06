package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCommentRevisionTransportAuthorityBinding(t *testing.T) {
	root, c, o, ids := currentValidationFixture(t)
	r, _, err := InspectCurrentValidationReopen(root, c, o)
	if err != nil {
		t.Fatal(err)
	}
	u, err := c.Unit(ids[0])
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := os.ReadFile(filepath.Join(root, r.Units[0].Snapshot.CandidatePath))
	if err != nil {
		t.Fatal(err)
	}
	g, err := LoadGlossary(root, o.Locale)
	if err != nil {
		t.Fatal(err)
	}
	p, err := buildCommentRevisionTransport(root, c, g, o.Locale, u, r, candidate)
	if err != nil || p == nil || len(p.Slots) != 3 {
		t.Fatal("authorized projection", err)
	}
	for _, name := range []string{"wrong-slot", "subset", "empty-subset", "superset", "protected", "reason"} {
		t.Run(name, func(t *testing.T) {
			var bad QualityCheckSurfaceReopen
			if err := json.Unmarshal(currentValidationReopenBytes(r), &bad); err != nil {
				t.Fatal(err)
			}
			finding := &bad.CurrentValidation.Findings[0]
			switch name {
			case "wrong-slot":
				finding.CommentSlots[0].SlotID = "not-a-source-coordinate"
			case "subset":
				finding.CommentSlots = finding.CommentSlots[:2]
			case "empty-subset":
				finding.CommentSlots = nil
			case "superset":
				finding.CommentSlots = append(finding.CommentSlots, finding.CommentSlots[0])
			case "protected":
				finding.CommentSlots[0].SemanticKind = "machine-semantic-comment"
			case "reason":
				finding.Reason = "unbound machine finding"
			}
			if _, err := buildCommentRevisionTransport(root, c, g, o.Locale, u, &bad, candidate); err == nil {
				t.Fatal("invalid comment authority accepted")
			}
		})
	}
}
