package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func parseUniqueRecoveryFlags(fs *flag.FlagSet, args []string) (map[string]bool, error) {
	seen := map[string]bool{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			return nil, fmt.Errorf("unexpected recovery argument %q", arg)
		}
		name, value, equals := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
		f := fs.Lookup(name)
		if f == nil {
			return nil, fmt.Errorf("unknown recovery flag --%s", name)
		}
		if seen[name] {
			return nil, fmt.Errorf("duplicate recovery flag --%s", name)
		}
		seen[name] = true
		isBool := false
		if v, ok := f.Value.(interface{ IsBoolFlag() bool }); ok {
			isBool = v.IsBoolFlag()
		}
		if isBool {
			if equals {
				return nil, fmt.Errorf("recovery boolean flag must be bare --%s", name)
			}
			continue
		}
		if !equals {
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("empty recovery flag --%s", name)
			}
			value = args[i]
		}
		if value == "" || strings.HasPrefix(value, "--") {
			return nil, fmt.Errorf("empty recovery flag --%s", name)
		}
	}
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() != 0 {
		return nil, fmt.Errorf("unexpected recovery arguments")
	}
	return seen, nil
}

func transportRecoveryCommand(root string, c *i18n.Catalog, args []string, hydrationErrors ...error) error {
	fs := flag.NewFlagSet("retranslation transport-recovery", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	pkg := fs.String("content-package", "", "complete registered package")
	predecessor := fs.String("predecessor-batch", "", "exact installed predecessor")
	supersedes := fs.String("supersedes", "", "explicit prior authority")
	supersedesSHA := fs.String("supersedes-sha256", "", "exact prior authority SHA")
	record := fs.Bool("record", false, "record immutable recovery authority")
	check := fs.Bool("check", false, "check exact recorded authority")
	expect := fs.String("expect-identity", "", "fresh preflight identity")
	resume := fs.Bool("resume", false, "resume same sealed record transaction")
	authority := fs.String("authority", "", "exact authority")
	authoritySHA := fs.String("authority-sha256", "", "exact authority SHA")
	seen, err := parseUniqueRecoveryFlags(fs, args)
	if err != nil {
		return err
	}
	if *locale == "" || *pkg == "" || (*record && *check) || (*resume && !*record) {
		return fmt.Errorf("exclusive recovery locale/package/preflight/record/check flags required")
	}
	if *check {
		if *authority == "" || *authoritySHA == "" || seen["predecessor-batch"] || seen["supersedes"] || seen["supersedes-sha256"] || seen["expect-identity"] || seen["resume"] {
			return fmt.Errorf("check requires only exact authority, locale and package")
		}
		if len(hydrationErrors) > 0 {
			return printRecoveryHydrationBlock(*locale, *pkg, "", *authority, *authoritySHA, true, hydrationErrors[0])
		}
		ref, err := i18n.ReadTransportRecoveryReference(root, *authority, *authoritySHA)
		if err != nil {
			return err
		}
		r, checkErr := i18n.CheckTransportRecovery(root, c, *locale, *pkg, ref)
		if r != nil {
			if err = printJSON(r); err != nil {
				return err
			}
		}
		return checkErr
	}
	if *predecessor == "" || seen["authority"] || seen["authority-sha256"] || (*record != seen["expect-identity"]) || seen["supersedes"] != seen["supersedes-sha256"] {
		return fmt.Errorf("recovery preflight/record flag contract mismatch")
	}
	if len(hydrationErrors) > 0 {
		return printRecoveryHydrationBlock(*locale, *pkg, *predecessor, "", "", false, hydrationErrors[0])
	}
	o := i18n.TransportRecoveryOptions{Locale: *locale, ContentPackage: *pkg, PredecessorBatch: *predecessor}
	if seen["supersedes"] {
		ref, err := i18n.ReadTransportRecoveryReference(root, *supersedes, *supersedesSHA)
		if err != nil {
			return err
		}
		o.Supersedes = &ref
	}
	if *record {
		r, recordErr := i18n.RecordTransportRecovery(root, c, o, *expect, *resume)
		if r != nil {
			if err = printJSON(r); err != nil {
				return err
			}
		}
		return recordErr
	}
	p, preflightErr := i18n.PreflightTransportRecovery(root, c, o)
	if p != nil {
		if err = printJSON(p); err != nil {
			return err
		}
	}
	return preflightErr
}

func transportRecoveryHydrationError(root string, args []string, err error) error {
	if len(args) < 2 || args[0] != "retranslation" || args[1] != "transport-recovery" {
		return err
	}
	return transportRecoveryCommand(root, nil, args[2:], err)
}

func printRecoveryHydrationBlock(locale, pkg, batch, path, sha string, check bool, cause error) error {
	blockers := []i18n.TransportRecoveryBlocker{{Code: "CATALOG_HYDRATION_FAILED", Detail: cause.Error()}}
	if check {
		r := i18n.TransportRecoveryCheck{Schema: "go-learning/transport-recovery-plan/v1", Status: "BLOCKED", Authority: i18n.TransportRecoveryReference{Path: path, SHA256: sha}, Mutation: "none", Blockers: blockers}
		if err := printJSON(r); err != nil {
			return err
		}
	} else {
		r := i18n.TransportRecoveryPlan{Schema: "go-learning/transport-recovery-plan/v1", Status: "BLOCKED", Locale: locale, ContentPackage: pkg, PredecessorBatch: batch, DirectDriftUnits: []string{}, RecoveryUnits: []string{}, Mutation: "none", Blockers: blockers}
		if err := printJSON(r); err != nil {
			return err
		}
	}
	return cause
}

func transportRecoveryExportCommand(root string, c *i18n.Catalog, pkg string, args []string) error {
	fs := flag.NewFlagSet("retranslation export --transport-recovery", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	provider := fs.String("generator", "", "same selected provider")
	authority := fs.String("transport-recovery", "", "exact recovery authority")
	sha := fs.String("transport-recovery-sha256", "", "exact authority SHA")
	resume := fs.Bool("resume", false, "resume same sealed export")
	if _, err := parseUniqueRecoveryFlags(fs, args); err != nil {
		return err
	}
	if *locale == "" || pkg == "" || *provider == "" || *authority == "" || *sha == "" {
		return fmt.Errorf("recovery export requires explicit locale/package/generator/authority/SHA")
	}
	ref, err := i18n.ReadTransportRecoveryReference(root, *authority, *sha)
	if err != nil {
		return err
	}
	r, err := i18n.ExportTransportRecoveryBatch(root, c, *locale, pkg, i18n.RetranslationGenerator(*provider), ref, *resume)
	if err != nil {
		return err
	}
	printRetranslationExportSummary(r)
	fmt.Printf("transport_recovery=%s identity=%s predecessor_batch=%s units=%d\n", ref.Path, ref.Identity, r.PredecessorBatchID, r.UnitCount)
	return nil
}

func hasRecoveryExportFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--transport-recovery" || strings.HasPrefix(arg, "--transport-recovery=") {
			return true
		}
	}
	return false
}
