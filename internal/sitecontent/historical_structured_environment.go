package sitecontent

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// These are the complete parser/protector data graphs, not their acceptance
// comparison. Frozen coordinates, target exact-set/protected restoration and
// the source/target structure comparison stay in the canonical consumer.
// Filesystem, QC/Surface evidence and current authority are observed outside.
type historicalStructuredGraph struct {
	Source Document
	Target Document
}

type historicalStructuredInput struct {
	Document    Document
	Raw, Target []byte
}

func deriveHistoricalStructuredGraph(in historicalStructuredInput) (*historicalStructuredGraph, error) {
	source := documentSource(in.Document)
	shape, err := parseDocument(source, in.Raw, false)
	if err != nil {
		return nil, err
	}
	source.SHA256 = digest(in.Target)
	parsed, err := parseDocument(source, in.Target, false)
	if err != nil {
		// Preserve the canonical target-parser error boundary.
		return nil, fmt.Errorf("frozen structured protected structure/parser round-trip changed")
	}
	return &historicalStructuredGraph{Source: *shape, Target: *parsed}, nil
}

// One owner for the QC/Surface historical reconstruction data component.
// Exact input bytes plus the executable identity bind the deterministic graph.
// Unknown/lossy codec or unavailable cache always uses the original parser.
// built measures full two-document derivation, never validation/PASS hits.
func sharedHistoricalStructuredGraph(d Document, raw, target []byte) (*historicalStructuredGraph, bool, error) {
	in := historicalStructuredInput{Document: d, Raw: raw, Target: target}
	canonical := func() (*historicalStructuredGraph, bool, error) {
		g, err := deriveHistoricalStructuredGraph(in)
		return g, true, err
	}
	key, err := json.Marshal(in)
	if err != nil {
		return canonical()
	}
	var copyInput historicalStructuredInput
	if json.Unmarshal(key, &copyInput) != nil || !reflect.DeepEqual(in, copyInput) {
		return canonical()
	}
	built := false
	b, err := contentidentity.Transform("sitecontent/historical-structured-graph/v1", key, func() ([]byte, error) {
		built = true
		g, err := deriveHistoricalStructuredGraph(in)
		if err != nil {
			return nil, err
		}
		b, err := json.Marshal(g)
		if err != nil {
			return nil, err
		}
		var copied historicalStructuredGraph
		if json.Unmarshal(b, &copied) != nil || !reflect.DeepEqual(g, &copied) {
			return nil, fmt.Errorf("historical structured graph codec is not lossless")
		}
		return b, nil
	})
	if err != nil {
		return canonical()
	}
	var g historicalStructuredGraph
	if json.Unmarshal(b, &g) != nil || g.Source.Path != d.Path || g.Target.Path != d.Path || g.Source.SourceSHA != digest(raw) || g.Target.SourceSHA != digest(target) || g.Source.Contract != d.Contract || g.Target.Contract != d.Contract {
		return canonical()
	}
	return &g, built, nil
}
