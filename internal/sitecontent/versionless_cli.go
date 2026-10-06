package sitecontent

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// RefreshSharedCorpusAuthority is an explicit compare-and-swap of static source
// projection only. It never performs source sync or touches locale evidence.
func RefreshSharedCorpusAuthority(root, priorSHA string, apply bool) (any, error) {
	if !validSHA(priorSHA) {
		return nil, fmt.Errorf("exact prior corpus file SHA required")
	}
	g, err := LoadCurrent(root)
	if err != nil {
		return nil, err
	}
	if _, err := LoadPageRegistry(root, g); err != nil {
		return nil, err // append-only registry must reconcile independently first
	}
	old, err := readRegular(root, i18n.GlossaryCorpusPath)
	if err != nil || digest(old) != priorSHA {
		return nil, fmt.Errorf("static corpus authority changed")
	}
	c, err := BuildUnifiedCorpus(root)
	if err != nil {
		return nil, err
	}
	b, err := Encode(c)
	if err != nil {
		return nil, err
	}
	if apply && digest(b) != priorSHA {
		dir := filepath.Join(root, "data")
		lock, err := os.OpenFile(filepath.Join(dir, ".glossary-corpus.lock"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return nil, err
		}
		lock.Close()
		defer os.Remove(lock.Name())
		f, err := os.CreateTemp(dir, ".glossary-corpus-*")
		if err != nil {
			return nil, err
		}
		defer os.Remove(f.Name())
		if err := f.Chmod(0644); err != nil {
			f.Close()
			return nil, err
		}
		if _, err := f.Write(b); err != nil {
			f.Close()
			return nil, err
		}
		if err := f.Sync(); err != nil {
			f.Close()
			return nil, err
		}
		if err := f.Close(); err != nil {
			return nil, err
		}
		old, err := readRegular(root, i18n.GlossaryCorpusPath)
		if err != nil || digest(old) != priorSHA {
			return nil, fmt.Errorf("concurrent corpus mutation")
		}
		if err := os.Rename(f.Name(), filepath.Join(root, i18n.GlossaryCorpusPath)); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// InitializeSharedLanguageAuthority creates static source authorities only.
// It does not initialize a locale or manufacture any language evidence.
func InitializeSharedLanguageAuthority(root string, apply bool) (any, error) {
	g, err := LoadCurrent(root)
	if err != nil {
		return nil, err
	}
	pages, err := FreezePageRegistry(g)
	if err != nil {
		return nil, err
	}
	corpus, err := BuildUnifiedCorpus(root)
	if err != nil {
		return nil, err
	}
	if apply {
		if err := saveImmutable(root, PageRegistryPath, pages); err != nil {
			return nil, err
		}
		if err := saveImmutable(root, "data/glossary-source-corpus.json", corpus); err != nil {
			return nil, err
		}
	}
	return struct {
		Pages  *PageRegistry `json:"page_registry"`
		Corpus any           `json:"glossary_corpus"`
	}{pages, corpus}, nil
}
