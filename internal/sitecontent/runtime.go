package sitecontent

import "io/fs"

func LocaleOrigin(root, locale string) (string, error) {
	profiles, err := liveProfiles(root)
	if err != nil {
		return "", err
	}
	for _, p := range profiles {
		if p.Locale == locale {
			return p.PublicURL, nil
		}
	}
	return "", fs.ErrNotExist
}

// RedirectContracts returns the frozen upstream exact/subtree handler map plus
// redirect-only source aliases. Subtrees have fixed destinations, never suffix
// concatenation. Runtime consumers must still require covered local targets.
func RedirectContracts(root string, g *Global) (map[string]string, error) {
	b, err := readRegular(root, SnapshotPath)
	if err != nil {
		return nil, err
	}
	f, err := snapshotFS(b)
	if err != nil {
		return nil, err
	}
	raw, err := fs.ReadFile(f, "internal/redirect/redirect.go")
	if err != nil {
		return nil, err
	}
	all, err := redirectsFromSource(raw)
	if err != nil {
		return nil, err
	}
	result := map[string]string{}
	for alias, target := range all {
		if alias == "/security" || len(alias) >= 10 && alias[:10] == "/security/" {
			result[alias] = target
		}
	}
	for _, s := range g.Sources {
		if s.Kind == "redirect" {
			result[s.Alias] = s.ResolvedRedirect
		}
	}
	return result, nil
}

// LocalCoverage admits only current evidence, not a caller-provided completion
// boolean. Sparse missing packages stay incomplete.
func LocalCoverage(root, locale string) (*Global, *Locale, error) {
	g, err := LoadCurrent(root)
	if err != nil {
		return nil, nil, err
	}
	l, err := CheckLocale(root, g, locale)
	return g, l, err
}
