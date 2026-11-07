package main

import (
	"flag"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func historicalConstructionBytesCommand(root string, c *i18n.Catalog, args []string, hydrationErrors ...error) error {
	fs := flag.NewFlagSet("generation-bundle historical-construction-bytes", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	pkg := fs.String("content-package", "", "complete registered package")
	batch := fs.String("batch-id", "", "exact installed initial batch")
	unit := fs.String("unit-id", "", "approved Unit")
	logical := fs.String("logical-dependency-path", "", "exact declared implementation path")
	source := fs.String("source", "", "absolute external historical whole-file copy")
	kind := fs.String("source-kind", "", "codex-artifact, editor-history, tmp-copy or backup")
	evidence := fs.String("acquisition-evidence", "", "absolute external acquisition artifact")
	evidenceSHA := fs.String("acquisition-evidence-sha256", "", "exact acquisition evidence hash")
	record := fs.Bool("record", false, "publish supplemental historical byte evidence")
	check := fs.Bool("check", false, "verify exact recorded historical byte evidence")
	expect := fs.String("expect-identity", "", "expected typed authority identity")
	cas := fs.String("expect-cas", "", "expected ephemeral preflight CAS token")
	resume := fs.Bool("resume", false, "resume the same complete sealed transaction")
	authority := fs.String("authority", "", "repository-relative authority")
	authoritySHA := fs.String("authority-sha256", "", "exact authority bytes hash")
	seen, err := parseUniqueRecoveryFlags(fs, args)
	if err != nil {
		return err
	}
	if *locale == "" || *pkg == "" || (*record && *check) || (*resume && !*record) {
		return fmt.Errorf("explicit locale/package and exclusive record/check modes required")
	}
	if *check {
		if *authority == "" || *authoritySHA == "" {
			return fmt.Errorf("check needs exact authority path/SHA")
		}
		for _, flag := range []string{"batch-id", "unit-id", "logical-dependency-path", "source", "source-kind", "acquisition-evidence", "acquisition-evidence-sha256", "expect-identity", "expect-cas", "resume"} {
			if seen[flag] {
				return fmt.Errorf("check rejects --%s", flag)
			}
		}
		if len(hydrationErrors) > 0 {
			return printHistoricalConstructionHydrationBlock(*locale, *pkg, "", "", "check", hydrationErrors[0])
		}
		p, e := i18n.CheckHistoricalConstructionBytes(root, c, *locale, *pkg, *authority, *authoritySHA)
		if p != nil {
			if err = printJSON(p); err != nil {
				return err
			}
		}
		return e
	}
	if *batch == "" || *unit == "" || *logical == "" || *source == "" || *kind == "" || *evidence == "" || *evidenceSHA == "" || seen["authority"] || seen["authority-sha256"] || seen["expect-identity"] != *record || seen["expect-cas"] != *record {
		return fmt.Errorf("historical byte preflight/record input and CAS flag contract mismatch")
	}
	mode := "preflight"
	if *record {
		mode = "record"
	}
	if len(hydrationErrors) > 0 {
		return printHistoricalConstructionHydrationBlock(*locale, *pkg, *batch, *unit, mode, hydrationErrors[0])
	}
	o := i18n.HistoricalBytesOptions{Locale: *locale, ContentPackage: *pkg, BatchID: *batch, UnitID: *unit, LogicalPath: *logical, Source: *source, SourceKind: *kind, Evidence: *evidence, EvidenceSHA256: *evidenceSHA}
	var p *i18n.HistoricalBytesPlan
	if *record {
		p, err = i18n.RecordHistoricalConstructionBytes(root, c, o, *expect, *cas, *resume)
	} else {
		p, err = i18n.PreflightHistoricalConstructionBytes(root, c, o)
	}
	if p != nil {
		if e := printJSON(p); e != nil {
			return e
		}
	}
	return err
}

func historicalConstructionHydrationError(root string, args []string, err error) error {
	if len(args) >= 2 && args[0] == "generation-bundle" && args[1] == "historical-construction-bytes" {
		return historicalConstructionBytesCommand(root, nil, args[2:], err)
	}
	return transportRecoveryHydrationError(root, args, err)
}

func printHistoricalConstructionHydrationBlock(locale, pkg, batch, unit, mode string, cause error) error {
	p := i18n.HistoricalBytesPlan{Schema: "go-learning/historical-construction-bytes-plan/v1", Mode: mode, Status: "BLOCKED", Locale: locale, ContentPackage: pkg, SourceVerification: "unknown", Blockers: []i18n.HistoricalBytesBlocker{{Code: "UNVERIFIED_HISTORICAL_GRAPH", Detail: cause.Error()}}, Mutation: "none"}
	if batch != "" {
		p.BatchID = &batch
	}
	if unit != "" {
		p.UnitID = &unit
	}
	if err := printJSON(p); err != nil {
		return err
	}
	return cause
}
