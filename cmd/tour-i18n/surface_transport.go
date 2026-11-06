package main

import (
	"flag"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func surfaceTransportCommand(root string, c *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("quality-check surface-transport", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	batch := fs.String("batch-id", "", "existing Surface revision batch identity")
	input := fs.String("input", "", "explicit source-region request JSON")
	discover := fs.Bool("discover", false, "list source-owned coordinates; not capability authorization")
	record := fs.Bool("record", false, "record a new immutable capability authority; never overwrite")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *locale == "" || *batch == "" || fs.NArg() != 0 || (*discover == (*input != "")) || (*record && *discover) {
		return fmt.Errorf("usage: quality-check surface-transport --locale <locale> --batch-id <batch-id> (--discover | --input <request.json> [--record])")
	}
	var request *i18n.SurfaceTransportRequest
	if *input != "" {
		b, err := readRegularBundleFile(*input)
		if err != nil {
			return err
		}
		request = &i18n.SurfaceTransportRequest{}
		if err := contentidentity.StrictJSON(b, request); err != nil {
			return err
		}
	}
	p, err := i18n.PrepareSurfaceRevisionTransport(root, c, *locale, *batch, request, *record)
	if err != nil {
		return err
	}
	if !*discover && !p.RevisionTransportCapable {
		return fmt.Errorf("Surface revision transport capability not proved")
	}
	return printJSON(p)
}
