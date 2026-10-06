package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func contentScopeCommand(root string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: content-scope <init|inventory-check|bootstrap-tour|repair-uncommitted-v2-a|check|status> [--source-root PATH|--locale LOCALE|--all]")
	}
	f := flag.NewFlagSet("content-scope "+args[0], flag.ContinueOnError)
	source := f.String("source-root", "", "read-only frozen upstream checkout")
	locale := f.String("locale", "", "one locale")
	all := f.Bool("all", false, "all existing live locales")
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	switch args[0] {
	case "repair-uncommitted-v2-a":
		if !*all || *locale != "" || *source != "" {
			return fmt.Errorf("repair-uncommitted-v2-a requires only --all")
		}
		n, err := sitecontent.RepairUncommittedV2A(root)
		if err != nil {
			return err
		}
		fmt.Printf("content-scope repair-uncommitted-v2-a: PASS locales=%d snapshot=unchanged\n", n)
		return nil
	case "init":
		if *source == "" || *locale != "" || *all {
			return fmt.Errorf("init requires only --source-root")
		}
		for _, p := range []string{sitecontent.GlobalPath, sitecontent.SnapshotPath} {
			if _, err := os.Lstat(filepath.Join(root, p)); !os.IsNotExist(err) {
				return fmt.Errorf("init requires absent authority: %s", p)
			}
		}
		snapshot, err := sitecontent.FrozenSnapshot(*source)
		if err != nil {
			return err
		}
		g, err := sitecontent.BuildGlobal(root, snapshot)
		if err != nil {
			return err
		}
		b, err := sitecontent.Encode(g)
		if err != nil {
			return err
		}
		if err := sitecontent.WriteNew(root, sitecontent.SnapshotPath, snapshot); err != nil {
			return err
		}
		if err := sitecontent.WriteNew(root, sitecontent.GlobalPath, b); err != nil {
			return err
		}
		fmt.Printf("content-scope init: PASS identity=%s\n", g.Identity)
		return nil
	case "inventory-check":
		if *source == "" || *locale != "" || *all {
			return fmt.Errorf("inventory-check requires only --source-root")
		}
		if _, err := sitecontent.LoadCurrent(root); err != nil {
			return err
		}
		snapshot, err := sitecontent.FrozenSnapshot(*source)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(filepath.Join(root, sitecontent.SnapshotPath))
		if err != nil {
			return err
		}
		if !bytes.Equal(b, snapshot) {
			return fmt.Errorf("frozen inventory STALE")
		}
		fmt.Println("content-scope inventory-check: CURRENT")
		return nil
	case "bootstrap-tour":
		if !*all || *locale != "" || *source != "" {
			return fmt.Errorf("bootstrap-tour requires only --all")
		}
		g, err := sitecontent.LoadCurrent(root)
		if err != nil {
			return err
		}
		n, err := sitecontent.BootstrapTour(root, g)
		if err != nil {
			return err
		}
		fmt.Printf("content-scope bootstrap-tour: PASS tour-v1_complete=%d site-v2-shell_complete=0 learn-docs-v1_complete=0\n", n)
		return nil
	case "check", "status":
		if *source != "" || (*all == (*locale != "")) {
			return fmt.Errorf("check/status requires exactly one of --locale or --all")
		}
		g, err := sitecontent.LoadCurrent(root)
		if err != nil {
			return err
		}
		locales := []string{*locale}
		if *all {
			locales, err = sitecontent.LiveLocales(root)
			if err != nil {
				return err
			}
		}
		for _, l := range locales {
			state, err := sitecontent.CheckLocale(root, g, l)
			if err != nil {
				return err
			}
			fmt.Printf("locale=%s current=CURRENT", l)
			for _, p := range sitecontent.PackageStates(g, state) {
				fmt.Printf(" %s=%s", p.Package, p.State)
				if p.LegacySurfaceState != "" {
					fmt.Printf(" legacy_surface_gate=%s", p.LegacySurfaceState)
				}
			}
			fmt.Println()
		}
		fmt.Printf("content-scope %s: CURRENT locales=%d identity=%s\n", args[0], len(locales), g.Identity)
		return nil
	default:
		return fmt.Errorf("unknown content-scope command %q", args[0])
	}
}
