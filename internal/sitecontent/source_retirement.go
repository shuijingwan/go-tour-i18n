package sitecontent

import (
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/sourcestorage"
)

type SourceZIPRetirement struct {
	Status            string      `json:"status"`
	Blockers          []Reference `json:"physical_zip_evidence_references"`
	Consumer          string      `json:"consumer"`
	CoveredReferences int         `json:"covered_corpus_references"`
	CoveredEntries    int         `json:"covered_source_entries"`
}

// Retirement requires strict current proof for every unchanged current/archive
// corpus reference, plus the original registered projection and corpus checks.
func CheckSourceZIPRetirement(root string) (*SourceZIPRetirement, error) {
	if _, err := CheckFrozenSources(root); err != nil {
		return nil, err
	}
	// Retirement eligibility precedes deletion. Verify the full source/global
	// projection and membership, without applying downstream representation
	// authority whose frozen envelope may already require the ZIP to be absent.
	g, err := loadSourceCurrent(root)
	if err != nil {
		return nil, err
	}
	if _, err := applyPackageMembership(root, g); err != nil {
		return nil, err
	}
	paths, err := CorpusAuthorityPaths(root)
	if err != nil {
		return nil, err
	}
	r := &SourceZIPRetirement{Status: "READY", Blockers: []Reference{}, Consumer: "strict storage compatibility; unchanged corpus path/SHA and semantic identities"}
	covered := map[string]bool{}
	for _, p := range paths {
		b, err := readRegular(root, p)
		if err != nil {
			return nil, err
		}
		c, err := i18n.LoadGlossaryCorpusAt(root, p)
		if err != nil {
			return nil, fmt.Errorf("corpus verification %s: %w", p, err)
		}
		for _, ref := range c.Sources {
			if ref.Path == SnapshotPath {
				required, err := i18n.CorpusStorageRequirements(c)
				if err != nil {
					return nil, err
				}
				if err := sourcestorage.HasCurrentProof(root, contentidentity.Reference{Path: p, SHA256: digest(b)}, c.Identity, contentidentity.Reference{Path: ref.Path, SHA256: ref.SHA256}, required); err != nil {
					r.Blockers = append(r.Blockers, Reference{Path: p, SHA256: digest(b)})
					r.Status = "ZIP_RETIREMENT_BLOCKED"
					continue
				}
				r.CoveredReferences++
				for _, name := range required {
					covered[name] = true
				}
			}
		}
	}
	r.CoveredEntries = len(covered)
	if r.CoveredReferences == 0 {
		return nil, fmt.Errorf("no exact corpus references covered")
	}
	if err := CheckUnifiedCorpus(root); err != nil {
		return nil, err
	}
	// No deletion here: the maintainer-authorized removal additionally requires
	// the isolated ZIP-absent regression and the same checks after retirement.
	return r, nil
}
