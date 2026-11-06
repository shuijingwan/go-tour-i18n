package i18n

import (
	"fmt"
	"reflect"
	"testing"
)

type syntheticContextAdapter struct {
	TranslationUnitAdapter
	extraError bool
}

func (syntheticContextAdapter) GenerationContextReferences(string, string) ([]GlossaryArchiveReference, error) {
	return []GlossaryArchiveReference{{Path: "source-authority.json", SHA256: sum([]byte("source"))}}, nil
}
func (a syntheticContextAdapter) RepresentationContextReferences(string) ([]GlossaryArchiveReference, error) {
	if a.extraError {
		return nil, fmt.Errorf("stale representation authority")
	}
	return []GlossaryArchiveReference{{Path: "representation-authority.json", SHA256: sum([]byte("representation"))}}, nil
}
func TestRepresentationGenerationContextIsAdditive(t *testing.T) {
	a := syntheticContextAdapter{}
	refs, err := translationGenerationContexts("root", "zh-CN", a)
	if err != nil {
		t.Fatal(err)
	}
	base, _ := a.GenerationContextReferences("root", "zh-CN")
	extra, _ := a.RepresentationContextReferences("root")
	if !reflect.DeepEqual(refs, append(base, extra...)) {
		t.Fatal("independent source/representation bindings lost", refs)
	}
	if _, err := translationGenerationContexts("root", "zh-CN", syntheticContextAdapter{extraError: true}); err == nil {
		t.Fatal("invalid representation binding accepted")
	}
}
