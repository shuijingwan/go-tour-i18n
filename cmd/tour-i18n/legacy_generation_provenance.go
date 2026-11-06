package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func parseLegacyGenerationProvenanceOptions(args []string) (i18n.LegacyGenerationProvenanceOptions, bool, string, error) {
	var o i18n.LegacyGenerationProvenanceOptions
	fs := flag.NewFlagSet("generation-bundle legacy-provenance", flag.ContinueOnError)
	fs.StringVar(&o.Locale, "locale", "", "target locale")
	fs.StringVar(&o.ContentPackage, "content-package", "", "complete package identity")
	fs.StringVar(&o.BatchID, "batch-id", "", "exact historical batch")
	fs.StringVar(&o.UnitID, "unit-id", "", "single historical Page Unit")
	fs.StringVar(&o.CurrentUnitID, "current-unit-id", "", "exact current Page Unit")
	fs.IntVar(&o.Attempt, "attempt", 0, "exact original attempt")
	fs.StringVar(&o.TaskKind, "task-kind", "", "original imported task")
	fs.StringVar(&o.ImportReceipt.Path, "import-receipt", "", "repository-relative trusted import receipt")
	fs.StringVar(&o.ImportReceipt.SHA256, "import-receipt-sha256", "", "trusted receipt SHA")
	fs.StringVar(&o.OriginalBundleSHA256, "original-bundle-sha256", "", "original unavailable archive SHA")
	fs.StringVar(&o.PreviousSnapshotID, "previous-snapshot-id", "", "explicit current predecessor Snapshot")
	fs.StringVar(&o.SnapshotSHA256, "previous-snapshot-sha256", "", "exact Snapshot SHA")
	record := fs.Bool("record", false, "explicit write-once record; default readonly")
	expected := fs.String("expected-identity", "", "exact preflight identity, record only")
	// flag.FlagSet normally accepts repeated scalar flags. This authority never
	// silently discards an earlier selector, including --name=value spellings.
	seen := map[string]bool{}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			name := strings.SplitN(strings.TrimLeft(arg, "-"), "=", 2)[0]
			if seen[name] {
				return o, false, "", fmt.Errorf("duplicate legacy provenance flag: %s", name)
			}
			seen[name] = true
		}
	}
	if err := fs.Parse(args); err != nil {
		return o, false, "", err
	}
	for _, name := range []string{"locale", "content-package", "batch-id", "unit-id", "current-unit-id", "attempt", "task-kind", "import-receipt", "import-receipt-sha256", "original-bundle-sha256", "previous-snapshot-id", "previous-snapshot-sha256"} {
		if !seen[name] || fs.Lookup(name).Value.String() == "" {
			return o, false, "", fmt.Errorf("explicit --%s required", name)
		}
	}
	if fs.NArg() != 0 || (*record && *expected == "") || (!*record && seen["expected-identity"]) {
		return o, false, "", fmt.Errorf("record requires expected-identity; inspection forbids it; no positional arguments")
	}
	return o, *record, *expected, nil
}
func legacyGenerationProvenanceCommand(root string, c *i18n.Catalog, args []string) error {
	o, record, expected, err := parseLegacyGenerationProvenanceOptions(args)
	if err != nil {
		return err
	}
	p, err := i18n.PrepareLegacyGenerationProvenance(root, c, o, record, expected)
	if p != nil {
		if e := printJSON(p); e != nil {
			return e
		}
	}
	return err
}
