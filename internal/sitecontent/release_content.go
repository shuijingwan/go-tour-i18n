package sitecontent

// LocalCoverageFiles is the existing completion contract plus the exact
// reconstructed bytes it validates. A release invocation consumes these bytes
// directly instead of asking CompletedPackageFiles to revalidate all packages
// once for every selected package. No caller-provided completion is accepted.
func LocalCoverageFiles(root, locale string) (*Global, *Locale, map[string]map[string][]byte, error) {
	v := newClosureValidation(root)
	g, l, err := v.localCoverage(root, locale)
	if err != nil {
		return nil, nil, nil, err
	}
	files := map[string]map[string][]byte{}
	for _, c := range l.Packages {
		if c.Package == "tour-v1" {
			continue
		}
		_, payload, err := v.checkedPackageClosure(root, c.Evidence[0])
		if err != nil {
			return nil, nil, nil, err
		}
		files[c.Package] = payload
	}
	if err := v.verify(); err != nil {
		return nil, nil, nil, err
	}
	return g, l, files, nil
}
