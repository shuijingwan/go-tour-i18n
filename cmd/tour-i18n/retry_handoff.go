package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func retryHandoffCommand(root string, c *i18n.Catalog, task string, args []string) error {
	fs := flag.NewFlagSet("generation-bundle "+task, flag.ContinueOnError)
	var paths repeatedStrings
	fs.Var(&paths, "bundle", "canonical single-Unit retry ZIP; repeat for handoff export")
	output := fs.String("output", "", "new aggregate ZIP")
	directory := fs.String("output-dir", "", "new local directory for original single-Unit ZIPs")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || len(paths) == 0 {
		return fmt.Errorf("retry handoff requires --bundle")
	}
	if task == "retry-handoff" {
		if *output == "" || *directory != "" {
			return fmt.Errorf("retry-handoff requires --output and repeated --bundle")
		}
		bundles := [][]byte{}
		for _, p := range paths {
			b, err := readRegularBundleFile(p)
			if err != nil {
				return err
			}
			bundles = append(bundles, b)
		}
		b, h, err := i18n.ExportRetryHandoff(root, c, bundles)
		if err != nil {
			return err
		}
		p, err := writeNewTransportOutput(*output, b)
		if err != nil {
			return err
		}
		fmt.Printf("Retry handoff exported: %s locale=%s batch=%s members=%d\n", p, h.Locale, h.BatchID, len(h.Members))
		return nil
	}
	if len(paths) != 1 || *output != "" {
		return fmt.Errorf("retry handoff check/unpack requires one --bundle")
	}
	b, err := readRegularBundleFile(paths[0])
	if err != nil {
		return err
	}
	if task == "retry-handoff-check" {
		if *directory != "" {
			return fmt.Errorf("check is read-only")
		}
		h, err := i18n.VerifyCurrentRetryHandoff(root, c, b)
		if err != nil {
			return err
		}
		fmt.Printf("Retry handoff: CURRENT members=%d\n", len(h.Members))
		return nil
	}
	if *directory == "" {
		return fmt.Errorf("unpack requires --output-dir")
	}
	h, bundles, err := i18n.ReadRetryHandoff(b)
	if err != nil {
		return err
	}
	// Unpack checks integrity only. Import/check still uses each original ZIP.
	// Atomic directory installation avoids a partially materialized handoff.
	abs, err := filepath.Abs(*directory)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(abs); !os.IsNotExist(err) {
		return fmt.Errorf("output-dir already exists or cannot be inspected")
	}
	stage, err := os.MkdirTemp(filepath.Dir(abs), ".retry-handoff-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	for i, member := range h.Members {
		if _, err := writeNewTransportOutput(filepath.Join(stage, member.BundleSHA256+".zip"), bundles[i]); err != nil {
			return err
		}
	}
	// Linux no-replace directory installation is the existing importer contract.
	if err := i18n.InstallRetryHandoffDirectory(stage, abs); err != nil {
		return err
	}
	fmt.Printf("Retry handoff integrity PASS; original ZIPs: %s members=%d; currentness requires per-Unit current-check\n", abs, len(h.Members))
	return nil
}
