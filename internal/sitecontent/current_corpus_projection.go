package sitecontent

import (
	"fmt"
	"os"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Current source governance owns projection epochs. A registered successor is
// rebuilt from its exact archived basis and current authorities; unchanged
// contributors retain their frozen projection. Only bootstrap uses today's
// extractors for the whole corpus. All readers use this same selection rule.
func currentCorpusProjection(root string, authority *TerminologyDiscoveryAuthority, snapshot []byte, contexts ...*structuredSuccessorValidation) (*i18n.GlossarySourceCorpus, error) {
	if err := replacementPublicationGuard(root); err != nil {
		return nil, err
	}
	if _, err := os.Lstat(pathJoin(root, i18n.GlossaryCorpusPath)); os.IsNotExist(err) {
		entries, scanErr := os.ReadDir(pathJoin(root, CorpusSuccessorRoot))
		if scanErr != nil && !os.IsNotExist(scanErr) {
			return nil, scanErr
		}
		if len(entries) != 0 {
			return nil, fmt.Errorf("current corpus missing with successor evidence")
		}
		return buildUnifiedCorpus(root, authority, snapshot)
	} else if err != nil {
		return nil, err
	}
	installed, err := checkedAppliedCorpusSuccessor(root, contexts...)
	if err != nil {
		return nil, err
	}
	if installed == nil {
		return buildUnifiedCorpus(root, authority, snapshot)
	}
	if err := checkCorpusTerminologyBinding(root, &installed.Result, authority, snapshot); err != nil {
		return nil, err
	}
	return &installed.Result, nil
}

// The frozen projection is not permission to ignore discovery authority drift.
// Exact metadata/snapshot hashes and the complete declared contributor set,
// purposes and parsers must still agree with today's registered authority.
func checkCorpusTerminologyBinding(root string, corpus *i18n.GlossarySourceCorpus, authority *TerminologyDiscoveryAuthority, snapshot []byte) error {
	if err := i18n.ValidateGlossaryCorpus(corpus); err != nil {
		return err
	}
	expected := map[string]TerminologySource{}
	if authority != nil {
		if err := ValidateTerminologyAuthority(*authority); err != nil {
			return err
		}
		if digest(snapshot) != authority.SnapshotSHA256 {
			return fmt.Errorf("terminology snapshot identity mismatch")
		}
		if _, err := readTerminologySnapshot(*authority, snapshot); err != nil {
			return err
		}
		manifest, err := Encode(authority)
		if err != nil {
			return err
		}
		for _, ref := range []Reference{{TerminologyManifestPath, digest(manifest)}, {TerminologySnapshotPath, digest(snapshot)}} {
			found := false
			for _, source := range corpus.Sources {
				found = found || source.Path == ref.Path && source.SHA256 == ref.SHA256
			}
			if !found {
				return fmt.Errorf("corpus does not bind current terminology authority: %s", ref.Path)
			}
			if _, err := readRegularReference(root, ref); err != nil {
				return err
			}
		}
		for _, source := range authority.Sources {
			expected[source.ID] = source
		}
	}
	for _, c := range corpus.Contributors {
		if i18n.EffectiveContributorPurpose(c) != i18n.ContributorPurposeTerminologyDiscovery {
			if _, exists := expected[c.ID]; exists {
				return fmt.Errorf("terminology contributor has wrong purpose: %s", c.ID)
			}
			continue
		}
		source, ok := expected[c.ID]
		if !ok || c.Parser != source.Parser || c.Path != source.SnapshotPath || c.Package != terminologyPackage || c.Structured || c.Route != "" {
			return fmt.Errorf("corpus terminology contributor authority mismatch: %s", c.ID)
		}
		delete(expected, c.ID)
	}
	if len(expected) != 0 {
		return fmt.Errorf("corpus terminology contributor exact-set mismatch")
	}
	return nil
}
