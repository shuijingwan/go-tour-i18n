package main

import (
	"flag"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func parseRepresentationRevisionOptions(args []string) (i18n.RepresentationRevisionOptions, error) {
	fs := flag.NewFlagSet("quality-check representation-revision", flag.ContinueOnError)
	var o i18n.RepresentationRevisionOptions
	addRepresentationReconciliationFlags(fs, &o.Reconciliation)
	fs.StringVar(&o.UnitID, "id", "", "exact unbindable current Unit")
	fs.StringVar(&o.ReopenID, "reopen-id", "", "existing shared replacement authority ID")
	fs.BoolVar(&o.Record, "record", false, "explicitly record immutable machine finding; default inspection only")
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	if err := validateRepresentationOptions(fs, o.Reconciliation); err != nil {
		return o, err
	}
	if o.UnitID == "" || o.ReopenID == "" {
		return o, fmt.Errorf("exact id and reopen-id required")
	}
	return o, nil
}
func representationRevisionCommand(root string, c *i18n.Catalog, args []string) error {
	o, err := parseRepresentationRevisionOptions(args)
	if err != nil {
		return err
	}
	r, path, err := i18n.PrepareRepresentationRevision(root, c, o)
	if err != nil {
		return err
	}
	mutation := "none"
	if o.Record {
		mutation = "known"
	}
	return printJSON(map[string]any{"authority": r, "receipt_path": path, "recorded": o.Record, "mutation": mutation})
}
