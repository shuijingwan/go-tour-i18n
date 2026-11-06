package main

import (
	"flag"
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func corpusSuccessorCommand(root string, args []string) error {
	f := flag.NewFlagSet("site-content corpus-successor", flag.ContinueOnError)
	input := f.String("input", "", "exact pending shell affected-scope artifact")
	prior := f.String("prior-sha256", "", "exact predecessor corpus file SHA for CAS")
	apply := f.Bool("apply", false, "archive/install explicit corpus successor; no language or review operation")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 || *input == "" {
		return fmt.Errorf("exact --input required; no positional arguments")
	}
	if *apply {
		p, err := sitecontent.ApplyCorpusSuccessor(root, *input, *prior)
		if err != nil {
			return err
		}
		return printJSON(p)
	}
	p, err := sitecontent.BuildCorpusSuccessor(root, *input)
	if err != nil {
		return err
	}
	if *prior != "" && *prior != p.Predecessor.SHA256 {
		return fmt.Errorf("predecessor CAS differs")
	}
	return printJSON(p)
}
