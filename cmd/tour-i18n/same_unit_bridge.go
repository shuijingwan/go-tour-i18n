package main

import (
	"flag"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func parseSameUnitBridgeOptions(args []string) (i18n.SameUnitBridgeOptions, error) {
	fs := flag.NewFlagSet("quality-check representation-bridge", flag.ContinueOnError)
	var o i18n.SameUnitBridgeOptions
	var packages, ids repeatedStrings
	fs.Var(&packages, "content-package", "required complete registered package")
	fs.Var(&ids, "unit-id", "exact QC B/C/D Page identity (repeatable)")
	fs.StringVar(&o.Locale, "locale", "", "target locale")
	fs.StringVar(&o.BridgeID, "bridge-id", "", "immutable shared representation authority ID")
	fs.StringVar(&o.PreviousSnapshotID, "previous-snapshot-id", "", "reviewed B/C/D predecessor Snapshot")
	fs.StringVar(&o.SnapshotSHA256, "previous-snapshot-sha256", "", "exact predecessor SHA256")
	fs.StringVar(&o.ReviewerEvidence.Path, "reviewer-evidence", "", "formal independent Reviewer evidence")
	fs.StringVar(&o.ReviewerEvidence.SHA256, "reviewer-evidence-sha256", "", "exact evidence SHA256")
	fs.BoolVar(&o.Record, "record", false, "write immutable receipt only after full fresh proof; default readonly")
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	if len(packages) != 1 || packages[0] == "" || fs.NArg() != 0 || len(ids) == 0 || o.Locale == "" || o.BridgeID == "" || o.PreviousSnapshotID == "" || o.SnapshotSHA256 == "" || o.ReviewerEvidence.Path == "" || o.ReviewerEvidence.SHA256 == "" {
		return o, fmt.Errorf("one explicit content-package, locale/bridge/Snapshot/evidence path-SHA and exact unit-id set required")
	}
	o.ContentPackage, o.UnitIDs = packages[0], ids
	return o, nil
}
func sameUnitBridgeCommand(root string, c *i18n.Catalog, args []string) error {
	o, err := parseSameUnitBridgeOptions(args)
	if err != nil {
		return err
	}
	plan, err := i18n.PrepareSameUnitRepresentationBridge(root, sitecontent.BridgeRepresentationCatalog(root, c), o)
	if err != nil {
		return err
	}
	return printJSON(plan)
}
