package i18n

import (
	"bytes"
	"fmt"
)

// validateRetranslationCandidateArtifact is workflow eligibility, separate from
// translation/representation authority. Match Git's default blank-at-eol bytes,
// not Unicode whitespace; reject rather than rewrite semantic whitespace.
func validateRetranslationCandidateArtifact(candidate []byte) error {
	for i, line := range bytes.Split(candidate, []byte{'\n'}) {
		if len(line) > 0 {
			switch line[len(line)-1] {
			case ' ', '\t', '\r':
				return fmt.Errorf("candidate has trailing whitespace at line %d", i+1)
			}
		}
	}
	return nil
}
