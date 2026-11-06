package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func structuredSourceSuccessorCommand(root string, args []string) error {
	f := flag.NewFlagSet(args[0], flag.ContinueOnError)
	input := f.String("input", "", "repository-relative proposal or authority")
	sha := f.String("input-sha256", "", "exact input SHA-256")
	prior := f.String("prior-identity", "", "exact predecessor identity: append scope or replacement receipt; apply only")
	priorPointer := f.String("prior-pointer-sha256", "", "exact source pointer SHA for consumed replacement or unconsumed correction")
	locale := f.String("locale", "", "accepted locale baseline; scope only")
	apply := f.Bool("apply", false, "publish source authority only")
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	if f.NArg() != 0 || *input == "" || *sha == "" {
		return fmt.Errorf("--input and --input-sha256 required")
	}
	ref, err := sitecontent.ReferenceFile(root, *input)
	if err != nil {
		return err
	}
	if ref.SHA256 != *sha {
		return fmt.Errorf("input SHA mismatch")
	}
	if args[0] == "structured-source-adopt" {
		b, err := sitecontent.ReadArtifact(root, *input)
		if err != nil {
			return err
		}
		var h struct {
			Schema string `json:"schema"`
		}
		if err := json.Unmarshal(b, &h); err != nil {
			return err
		}
		if h.Schema == sitecontent.StructuredSourceReplacementSchema {
			return fmt.Errorf("use structured-source-replace for consumed replacement")
		}
	}
	if args[0] == "structured-source-replace" {
		if *locale != "" {
			return fmt.Errorf("source replacement is locale independent")
		}
		if !*apply {
			if *prior != "" || *priorPointer != "" {
				return fmt.Errorf("CAS flags require --apply")
			}
			p, err := sitecontent.InspectConsumedStructuredSourceReplacement(root, ref)
			if err != nil {
				return err
			}
			return printJSON(p)
		}
		if *prior == "" || *priorPointer == "" {
			return fmt.Errorf("replacement apply requires exact --prior-identity and --prior-pointer-sha256")
		}
		p, err := sitecontent.ApplyConsumedStructuredSourceReplacement(root, ref, *prior, *priorPointer)
		if err != nil {
			return err
		}
		return printJSON(p)
	}
	if args[0] == "structured-source-scope" {
		if *locale == "" || *apply || *prior != "" || *priorPointer != "" {
			return fmt.Errorf("scope requires only exact authority and --locale")
		}
		p, err := sitecontent.PlanStructuredSourceSuccessor(root, *locale, ref)
		if err != nil {
			return err
		}
		return printJSON(p)
	}
	if *locale != "" && args[0] == "structured-source-adopt" && !*apply && *prior == "" && *priorPointer == "" {
		p, err := sitecontent.InspectStructuredSourceSuccessor(root, *locale, ref)
		if err != nil {
			return err
		}
		return printJSON(p)
	}
	if *locale != "" {
		return fmt.Errorf("source authority is locale independent")
	}
	if args[0] == "structured-source-check" {
		if *apply || *prior != "" || *priorPointer != "" {
			return fmt.Errorf("check is readonly")
		}
		a, err := sitecontent.CheckStructuredSourceSuccessor(root, ref)
		if err != nil {
			return err
		}
		g, err := sitecontent.LoadCurrent(root)
		if err != nil {
			return err
		}
		found := false
		for _, r := range g.StructuredSuccessors {
			found = found || r == ref
		}
		if !found {
			return fmt.Errorf("source authority is not current")
		}
		return printJSON(struct {
			State     string                                 `json:"state"`
			Authority *sitecontent.StructuredSourceSuccessor `json:"authority"`
			Mutation  string                                 `json:"mutation"`
		}{"CURRENT", a, "none"})
	}
	var a *sitecontent.StructuredSourceSuccessor
	if *apply {
		if *priorPointer != "" {
			a, err = sitecontent.CorrectUnconsumedStructuredSource(root, ref, *prior, *priorPointer)
		} else {
			a, err = sitecontent.AdoptStructuredSourceSuccessor(root, ref, *prior)
		}
	} else {
		if *prior != "" || *priorPointer != "" {
			return fmt.Errorf("CAS applies only with --apply")
		}
		a, err = sitecontent.BuildStructuredSourceSuccessor(root, ref)
		if err == nil && a.Replacement != nil {
			return fmt.Errorf("use structured-source-replace for consumed replacement")
		}
	}
	if err != nil {
		return err
	}
	return printJSON(struct {
		Authority *sitecontent.StructuredSourceSuccessor `json:"authority"`
		Path      string                                 `json:"receipt_path"`
		Applied   bool                                   `json:"applied"`
	}{a, sitecontent.StructuredSourceSuccessorPath(a), *apply})
}

func adoptedCorpusCommand(root string, args []string) error {
	f := flag.NewFlagSet("adopted-corpus", flag.ContinueOnError)
	prior := f.String("prior-sha256", "", "exact predecessor corpus SHA-256")
	apply := f.Bool("apply", false, "publish source projection through existing corpus successor CAS")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	p, err := sitecontent.AdoptCorpusProjection(root, *prior, *apply)
	if err != nil {
		return err
	}
	return printJSON(p)
}
