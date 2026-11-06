package main

import (
	"flag"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func qualityCheckAuthorizationCommand(root string, c *i18n.Catalog, args []string) error {
	f := flag.NewFlagSet("quality-check authorization", flag.ContinueOnError)
	locale := f.String("locale", "", "target locale")
	snapshot := f.String("snapshot-id", "", "finalized Snapshot")
	if err := f.Parse(args); err != nil {
		return err
	}
	if *locale == "" || *snapshot == "" || f.NArg() != 0 {
		return fmt.Errorf("--locale and --snapshot-id are required")
	}
	scoped, registered, err := i18n.SelectFinalizedTranslationPackage(root, c, *locale, *snapshot)
	if err != nil {
		return err
	}
	if registered {
		// Preserve the sealed checkpoint reader and its fail-closed validation.
		// Only an absent checkpoint falls back to a fresh readonly assessment.
		a, err := i18n.FindCurrentTranslationAuthorization(root, scoped, *locale, *snapshot)
		if err != nil {
			return err
		}
		if a == nil {
			a, err = i18n.AssessCurrentTranslation(root, scoped, *locale, *snapshot)
		}
		if a != nil {
			if e := printJSON(a); e != nil {
				return e
			}
		}
		if err != nil {
			return err
		}
		if !a.Ready() {
			return fmt.Errorf("current authorization blocked: affected=%d stale=%d unknown=%d", a.Affected, a.Stale, a.Unknown)
		}
		return nil
	}
	d, err := i18n.DiagnoseQualityCheckAuthorization(root, scoped, *locale, *snapshot)
	if err != nil {
		return err
	}
	if err := printJSON(d); err != nil {
		return err
	}
	if !d.CanonicalAuthorized {
		return fmt.Errorf("current canonical authorization blocked: %s", d.Reason)
	}
	return nil
}

func translationReattestationCommand(root string, c *i18n.Catalog, args []string) error {
	f := flag.NewFlagSet("quality-check reattest", flag.ContinueOnError)
	locale := f.String("locale", "", "locale")
	snapshot := f.String("snapshot-id", "", "historical A-only finalization Snapshot")
	apply := f.Bool("apply", false, "save immutable current authorization only if every Unit is eligible")
	canonical := f.Bool("canonical", false, "bind installed canonical targets to exact accepted A candidates")
	if err := f.Parse(args); err != nil {
		return err
	}
	if *locale == "" || *snapshot == "" || f.NArg() != 0 {
		return fmt.Errorf("--locale and --snapshot-id are required")
	}
	if *canonical && !*apply {
		return fmt.Errorf("--canonical requires --apply")
	}
	var a *i18n.CurrentTranslationAuthorization
	var err error
	if *apply {
		a, _, err = i18n.EstablishCurrentTranslationAuthorization(root, c, *locale, *snapshot, *canonical)
	} else {
		a, err = i18n.AssessCurrentTranslation(root, c, *locale, *snapshot)
	}
	if a != nil {
		if e := printJSON(a); e != nil {
			return e
		}
	}
	if err != nil {
		return err
	}
	if !a.Ready() {
		return fmt.Errorf("current re-attestation blocked: affected=%d stale=%d unknown=%d", a.Affected, a.Stale, a.Unknown)
	}
	return nil
}
