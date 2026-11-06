package main

import (
	"flag"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func structuredTimestampRecoveryCommand(root string, args []string) error {
	fs := flag.NewFlagSet("site-content structured-recover-timestamp", flag.ContinueOnError)
	input := fs.String("request", "", "repository-relative timestamp-loss request JSON")
	sha := fs.String("request-sha256", "", "exact reviewed request SHA-256")
	apply := fs.Bool("apply", false, "publish immutable timestamp-loss lineage and Generation evidence")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || *input == "" || *sha == "" {
		return fmt.Errorf("structured-recover-timestamp requires request/request-sha256; no positional or provenance override flags")
	}
	r, err := sitecontent.RecoverStructuredTimestamp(root, sitecontent.Reference{Path: *input, SHA256: *sha}, *apply)
	if err != nil {
		return err
	}
	return printJSON(r)
}
