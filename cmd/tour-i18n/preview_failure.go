package main

import (
	"flag"
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/site"
)

func previewFailureCommand(root string, c *i18n.Catalog, args []string) error {
	f := flag.NewFlagSet("quality-check preview-failure", flag.ContinueOnError)
	locale := f.String("locale", "", "target locale")
	snapshot := f.String("previous-snapshot-id", "", "current finalized predecessor")
	unit := f.String("unit-id", "", "exact affected TU")
	route := f.String("route", "", "exact preview route")
	base := f.String("preview-url", "", "running loopback preview origin")
	id := f.String("failure-id", "", "immutable machine failure receipt id")
	if err := f.Parse(args); err != nil {
		return err
	}
	if *locale == "" || *snapshot == "" || *unit == "" || *route == "" || *base == "" || *id == "" || f.NArg() != 0 {
		return fmt.Errorf("quality-check preview-failure requires --locale --previous-snapshot-id --unit-id --route --preview-url --failure-id")
	}
	path, err := site.CapturePreviewTUFailure(root, c, *locale, *snapshot, *unit, *route, *base, *id)
	if err != nil {
		return err
	}
	fmt.Printf("Machine preview TU failure recorded (revision authorization only): %s\n", path)
	return nil
}
