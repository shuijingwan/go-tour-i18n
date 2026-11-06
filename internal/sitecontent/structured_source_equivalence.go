package sitecontent

import (
	"bytes"
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"reflect"
)

// An explicit active-content purpose in v2 does not alter the parser's exact
// source contexts. All parser/path/route/structure and language fields remain
// exact; derived contributor hashes are independently verified by the corpus.
func sameStructuredContributorInput(old, current []byte, id, currentSHA string) bool {
	group := func(raw []byte) ([]i18n.LanguageContributor, error) {
		var c i18n.GlossarySourceCorpus
		if err := StrictJSON(raw, &c); err != nil {
			return nil, err
		}
		if err := i18n.ValidateGlossaryCorpus(&c); err != nil {
			return nil, err
		}
		groups, err := structuredContributorGroups(c.Contributors)
		if err != nil {
			return nil, err
		}
		out := groups[id]
		if bytes.Equal(raw, current) && identity(out) != currentSHA {
			return nil, fmt.Errorf("current structured source identity mismatch")
		}
		for i := range out {
			out[i].SourceSHA = ""
			out[i].Purpose = i18n.EffectiveContributorPurpose(out[i])
		}
		return out, nil
	}
	a, err := group(old)
	if err != nil || len(a) == 0 {
		return false
	}
	b, err := group(current)
	return err == nil && reflect.DeepEqual(a, b)
}
