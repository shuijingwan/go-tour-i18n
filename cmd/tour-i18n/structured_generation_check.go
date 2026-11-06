package main

import (
	"flag"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

type structuredGenerationCheckResult struct {
	Schema                    string                 `json:"schema"`
	Status                    string                 `json:"status"`
	Generation                sitecontent.Reference  `json:"generation"`
	Locale                    string                 `json:"locale"`
	PlanIdentity              string                 `json:"plan_identity_sha256"`
	OutputCount               int                    `json:"output_count"`
	ValidatedFileCount        int                    `json:"validated_file_count"`
	TimestampRecovery         *sitecontent.Reference `json:"timestamp_recovery,omitempty"`
	OriginalGeneratedAtStatus string                 `json:"original_generated_at_status,omitempty"`
	Mutation                  string                 `json:"mutation"`
}

func structuredGenerationCheckCommand(root string, args []string) error {
	fs := flag.NewFlagSet("site-content structured-generation-check", flag.ContinueOnError)
	input := fs.String("input", "", "repository-relative published StructuredGeneration")
	sha := fs.String("input-sha256", "", "exact StructuredGeneration file SHA-256")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || *input == "" || *sha == "" {
		return fmt.Errorf("structured-generation-check requires input/input-sha256; no positional arguments")
	}
	// Reuse repository path/type/symlink validation; never infer the trusted SHA
	// from the filename or accept a different observed file as the requested ref.
	ref, err := sitecontent.ReferenceFile(root, *input)
	if err != nil {
		return err
	}
	if ref.SHA256 != *sha {
		return fmt.Errorf("StructuredGeneration input SHA-256 mismatch")
	}
	r, files, err := sitecontent.CheckStructuredGeneration(root, ref)
	if err != nil {
		return err
	}
	result := structuredGenerationCheckResult{Schema: "go-learning/structured-generation-check/v1", Status: "PASS", Generation: ref, Locale: r.Plan.Locale, PlanIdentity: r.Plan.Identity, OutputCount: len(r.Output.Outputs), ValidatedFileCount: len(files), TimestampRecovery: r.TimestampRecovery, Mutation: "none"}
	// This status follows a successfully validated authority, never an empty
	// generated_at string considered in isolation.
	if r.TimestampRecovery != nil {
		result.OriginalGeneratedAtStatus = "unavailable"
	}
	return printJSON(result)
}
