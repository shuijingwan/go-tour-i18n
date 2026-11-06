package sitecontent

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Only the representation calculation is shared. The resolver still reads and
// hashes its frozen reference, checks the package/registry and rebuilds the
// historical Catalog each time. Its callers still verify every current source,
// receipt, candidate, validation, Review and historical event.
func reconcileReplacementRepresentations(old []Document, next []PageAdapter) (*Reconciliation, error) {
	// Page parsing has no filesystem or glossary inputs. Full Documents bind all
	// coordinates, dependencies and representation modes; Source binds exact raw
	// bytes. The algorithm/options are fixed within this invocation's executable.
	type inputs struct {
		Old  []Document
		Next []PageAdapter
	}
	input := inputs{old, next}
	key, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	compute := func() ([]byte, error) {
		docs := make([]Document, 0, len(next))
		for _, a := range next {
			parsed, err := currentReplacementPage(a)
			if err != nil || !reflect.DeepEqual(*parsed, a.Document) {
				return nil, fmt.Errorf("current replacement parser representation stale")
			}
			docs = append(docs, a.Document)
		}
		r, err := ReconcileHistoricalPages(old, docs)
		if err != nil {
			return nil, err
		}
		return json.Marshal(r)
	}
	// JSON omitempty can collapse nil and empty slices even though the parser's
	// DeepEqual check distinguishes them. Only round-trip-exact inputs may use
	// this key: any two admitted inputs with equal keys must be DeepEqual.
	var decoded inputs
	if err := json.Unmarshal(key, &decoded); err != nil {
		return nil, err
	}
	var b []byte
	if reflect.DeepEqual(input, decoded) {
		b, err = contentidentity.Transform("sitecontent/replacement-representations/v1", key, compute)
	} else {
		b, err = compute()
	}
	if err != nil {
		return nil, err
	}
	var out Reconciliation
	if err := json.Unmarshal(b, &out); err != nil {
		// A damaged derived representation must not turn storage into a new
		// authority/availability gate. The canonical pure construction decides.
		b, err = compute()
		if err != nil {
			return nil, err
		}
		out = Reconciliation{}
		if err := json.Unmarshal(b, &out); err != nil {
			return nil, err
		}
	}
	return &out, nil
}
