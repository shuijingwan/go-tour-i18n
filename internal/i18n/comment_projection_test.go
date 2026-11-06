package i18n

import (
	"strings"
	"testing"
)

func TestCommentBlockDispositionTourProjection(t *testing.T) {
	c, err := BuildSourceCatalog(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	matched := 0
	for _, p := range c.Pages {
		examples := []Example{}
		for _, e := range c.Examples {
			for _, id := range e.ReferencedBy {
				if id == p.ID {
					examples = append(examples, e)
				}
			}
		}
		for _, b := range TourPageCommentBlocks(p.Source) {
			if !b.Unsupported {
				continue
			}
			proof, ok := TourCommentFreeProjection(b.Code, examples)
			expected := p.ID == "basics/5" || p.ID == "moretypes/17"
			if ok != expected {
				t.Fatalf("%s/%s projection=%v proof=%s", p.ID, b.Coordinate, ok, proof)
			}
			if ok {
				matched++
				for _, bad := range []string{b.Code + "\n// The teaching comment.\n", b.Code + "\nunknown()", strings.ReplaceAll(b.Code, "int", "string")} {
					if bad == b.Code {
						continue
					}
					if _, ok := TourCommentFreeProjection(bad, examples); ok {
						t.Fatal("unbound/comment-bearing projection accepted", bad)
					}
				}
				if _, ok := TourCommentFreeProjection(b.Code, nil); ok {
					t.Fatal("projection without source authority")
				}
			}
		}
	}
	if matched != 4 {
		t.Fatal("real Tour exact set", matched)
	}
}
