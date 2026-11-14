package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func qualityCheckCorrectCommand(root string, catalog *i18n.Catalog, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("quality-check correct", flag.ContinueOnError)
	values := map[string]*repeatedStrings{}
	for _, name := range []string{"locale", "snapshot-id", "reviewer-session", "correction-file", "correction-sha256"} {
		values[name] = &repeatedStrings{}
		fs.Var(values[name], name, name)
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected quality-check correct arguments: %s", strings.Join(fs.Args(), " "))
	}
	for name, value := range values {
		if len(*value) != 1 || (*value)[0] == "" {
			return fmt.Errorf("exactly one non-empty --%s is required", name)
		}
	}
	get := func(name string) string { return (*values[name])[0] }
	evidence, err := os.ReadFile(get("correction-file"))
	if err != nil {
		return err
	}
	result, err := i18n.RecordQualityCheckCorrection(root, catalog, i18n.QualityCheckCorrectionOptions{
		Locale: get("locale"), SnapshotID: get("snapshot-id"), ReviewerSession: get("reviewer-session"), EvidenceSHA256: get("correction-sha256"), Evidence: evidence,
	})
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
