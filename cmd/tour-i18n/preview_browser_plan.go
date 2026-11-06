package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/shuijingwan/go-tour-i18n/internal/site"
)

func previewBrowserPlanCommand(root string, args []string) error {
	f := flag.NewFlagSet("preview-browser-plan", flag.ContinueOnError)
	locale := f.String("locale", "", "activated locale")
	if err := f.Parse(args); err != nil {
		return err
	}
	if *locale == "" || f.NArg() != 0 {
		return fmt.Errorf("preview-browser-plan requires only --locale")
	}
	plan, err := site.LoadPreviewBrowserPlan(root, *locale)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(plan)
}
