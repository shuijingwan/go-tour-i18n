package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func integratedSurfaceBundleSHA(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func integratedSurfaceReviewerBundleCommand(root string, args []string) error {
	fs := flag.NewFlagSet("site-content "+args[0], flag.ContinueOnError)
	if args[0] == "surface-reviewer-bundle-check" {
		bundle := fs.String("bundle", "", "repository-relative Integrated Surface Reviewer ZIP")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *bundle == "" || fs.NArg() != 0 {
			return fmt.Errorf("--bundle required; no positional arguments")
		}
		b, err := sitecontent.ReadArtifact(root, *bundle)
		if err != nil {
			return err
		}
		start := time.Now()
		m, err := sitecontent.CheckIntegratedSurfaceReviewerBundle(root, b)
		if err != nil {
			return err
		}
		return printJSON(map[string]any{"result": "CURRENT", "bundle": *bundle, "bundle_sha256": integratedSurfaceBundleSHA(b), "bundle_bytes": len(b), "scope_identity_sha256": m.ScopeIdentity, "review_required": m.ReviewRequiredCount, "carried": m.CarriedCount, "wall_seconds": time.Since(start).Seconds()})
	}
	locale := fs.String("locale", "", "locale (optional with --input)")
	input := fs.String("input", "", "repository-relative canonical current Integrated Surface scope JSON")
	output := fs.String("output", "", "repository-relative immutable Reviewer ZIP output")
	var closures repeatedStrings
	fs.Var(&closures, "generation", "prospective package closure (repeatable; same selector as surface-scope)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 || *output == "" || (*input != "" && len(closures) > 0) || (*input == "" && (*locale == "" || len(closures) == 0)) {
		return fmt.Errorf("--output and either --input or --locale with repeatable --generation closure required")
	}
	o := sitecontent.IntegratedSurfaceReviewerOptions{Locale: *locale}
	if *input != "" {
		b, err := sitecontent.ReadArtifact(root, *input)
		if err != nil {
			return err
		}
		o.Scope = b
	}
	for _, name := range closures {
		ref, err := sitecontent.ReferenceFile(root, name)
		if err != nil {
			return err
		}
		o.Closures = append(o.Closures, ref)
	}
	start := time.Now()
	b, m, err := sitecontent.ExportIntegratedSurfaceReviewerBundle(root, o)
	if err != nil {
		return err
	}
	if err := sitecontent.SaveArtifact(root, *output, b); err != nil {
		return err
	}
	return printJSON(map[string]any{"result": "EXPORTED", "bundle": *output, "bundle_sha256": integratedSurfaceBundleSHA(b), "bundle_bytes": len(b), "scope_identity_sha256": m.ScopeIdentity, "scope_sha256": m.Scope.SHA256, "review_required": m.ReviewRequiredCount, "carried": m.CarriedCount, "wall_seconds": time.Since(start).Seconds()})
}
