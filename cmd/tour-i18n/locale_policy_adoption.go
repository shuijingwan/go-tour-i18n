package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func localeLanguagePolicyAdoptionCommand(root string, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("locale language-policy-adopt", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	locale := fs.String("locale", "", "existing initialized locale")
	input := fs.String("input", "", "repository-relative complete policy authority JSON")
	apply := fs.Bool("apply", false, "explicit CAS-protected adoption")
	prior := fs.String("prior-sha256", "", "exact metadata SHA from preflight")
	policy := fs.String("input-sha256", "", "exact input policy SHA from preflight")
	result := fs.String("result-sha256", "", "exact expected metadata SHA from preflight")
	seen := map[string]bool{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		if !strings.HasPrefix(arg, "-") {
			continue
		}
		parts := strings.SplitN(strings.TrimLeft(arg, "-"), "=", 2)
		key := parts[0]
		if seen[key] {
			return fmt.Errorf("mutation=none: duplicate flag --%s", key)
		}
		seen[key] = true
		// String flag values may themselves start with '-'; they are not flags.
		if fs.Lookup(key) != nil && key != "apply" && len(parts) == 1 {
			i++
		}
	}
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("mutation=none: %w", err)
	}
	if fs.NArg() != 0 || *locale == "" || *input == "" {
		return fmt.Errorf("mutation=none: --locale and --input required; positional arguments forbidden")
	}
	if !*apply && (seen["prior-sha256"] || seen["input-sha256"] || seen["result-sha256"]) {
		return fmt.Errorf("mutation=none: SHA flags require --apply")
	}
	r, err := i18n.AdoptLocaleLanguagePolicy(root, i18n.LocalePolicyAdoptionOptions{Locale: *locale, Input: *input, Apply: *apply, PriorSHA256: *prior, InputSHA256: *policy, ResultSHA256: *result})
	if err != nil {
		return err
	}
	if err := writeJSON(out, r); err != nil {
		if r.Applied {
			return fmt.Errorf("mutation=known: metadata published but stdout failed: %w", err)
		}
		return fmt.Errorf("mutation=none: stdout failed: %w", err)
	}
	return nil
}
