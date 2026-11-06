package main

import (
	"flag"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func structuredInputReopenCommand(root string, args []string) error {
	fs := flag.NewFlagSet("site-content structured-input-reopen", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	asset := fs.String("asset", "", "exact registered structured asset ID")
	apply := fs.Bool("apply", false, "保存 immutable machine reopen receipt；默认只读 preflight")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || *locale == "" || *asset == "" {
		return fmt.Errorf("structured-input-reopen requires --locale and --asset, without positional arguments")
	}
	r, err := sitecontent.ReopenStructuredInput(root, *locale, *asset, *apply)
	if err != nil {
		if printErr := printJSON(struct {
			Ready  bool   `json:"ready"`
			Reason string `json:"reason"`
		}{false, err.Error()}); printErr != nil {
			return printErr
		}
		return err
	}
	return printJSON(r)
}
