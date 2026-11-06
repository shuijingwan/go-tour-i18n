package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/site"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

// Source/package adapters use the shared i18n TranslationUnit lifecycle.
func siteContentReferencesInOrder(root string, paths []string) ([]sitecontent.Reference, error) {
	result := []sitecontent.Reference{}
	for _, p := range paths {
		r, err := sitecontent.ReferenceFile(root, p)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, nil
}

func siteContentCommand(root string, args []string) error {
	if len(args) > 0 && args[0] == "structured-generation-check" {
		return structuredGenerationCheckCommand(root, args[1:])
	}
	if len(args) > 0 && args[0] == "structured-recover-timestamp" {
		return structuredTimestampRecoveryCommand(root, args[1:])
	}
	if len(args) > 0 && args[0] == "shared-structured-representation-reattest" {
		return sharedRepresentationReattestationCommand(root, args[1:])
	}
	if len(args) > 0 && args[0] == "shared-structured-representation-successor" {
		return sharedStructuredRepresentationCommand(root, args[1:])
	}
	if len(args) > 0 && args[0] == "shell-representation-reattest" {
		return shellRepresentationReattestationCommand(root, args[1:])
	}
	if len(args) > 0 && (args[0] == "shell-representation-successor" || args[0] == "shell-representation-check") {
		return shellRepresentationCommand(root, args)
	}
	if len(args) > 0 && (args[0] == "structured-source-adopt" || args[0] == "structured-source-replace" || args[0] == "structured-source-check" || args[0] == "structured-source-scope") {
		return structuredSourceSuccessorCommand(root, args)
	}
	if len(args) > 0 && args[0] == "adopted-corpus" {
		return adoptedCorpusCommand(root, args[1:])
	}
	if len(args) > 0 && (args[0] == "page-package-adopt" || args[0] == "page-package-adoption-check") {
		return pagePackageAdoptionCommand(root, args)
	}
	if len(args) > 0 && args[0] == "structured-input-reopen" {
		return structuredInputReopenCommand(root, args[1:])
	}
	if len(args) > 0 && (args[0] == "surface-reviewer-bundle" || args[0] == "surface-reviewer-bundle-check") {
		return integratedSurfaceReviewerBundleCommand(root, args)
	}
	if len(args) > 0 && args[0] == "structured-recover" {
		return structuredRecoveryCommand(root, args[1:])
	}
	if len(args) > 0 && (args[0] == "shell-adopt" || args[0] == "shell-adoption-check" || args[0] == "homepage-plan") {
		return shellAdoptionCommand(root, args)
	}
	if len(args) > 0 && args[0] == "corpus-successor" {
		return corpusSuccessorCommand(root, args[1:])
	}
	if len(args) > 0 && (args[0] == "shell-successor" || args[0] == "shell-successor-check") {
		return shellSuccessorCommand(root, args)
	}
	if len(args) > 0 && (args[0] == "source-migrate" || args[0] == "source-check" || args[0] == "source-retirement-check" || args[0] == "source-storage-compatibility") {
		return siteSourceCommand(root, args)
	}
	if len(args) == 0 {
		return fmt.Errorf("usage: site-content <terminology-authority|terminology-check|corpus-check|inventory|inventory-summary|generation-plan|generation-export|generation-check|generation-import|review-plan|review-scope|review-export|review-record|finalize|activation-preflight|activation-apply|coverage|routing-check|reconcile|compatibility-assess>")
	}
	flags := flag.NewFlagSet("site-content "+args[0], flag.ContinueOnError)
	apply := flags.Bool("apply", false, "apply the precise static V2-D authority upgrade")
	locale := flags.String("locale", "", "locale")
	pkg := flags.String("package", "", "package")
	input := flags.String("input", "", "repository-relative input")
	output := flags.String("output", "", "repository-relative immutable output")
	bundle := flags.String("bundle", "", "repository-relative bundle")
	task := flags.String("task", "initial", "initial/revision/glossary-revision/surface-revision; structured supports initial/revision")
	batch := flags.String("batch", "", "batch id")
	planBatch := flags.Int("plan-batch", 1, "1-based deterministic working set; cannot combine with --unit")
	finding := flags.String("finding", "", "review artifact path")
	provider := flags.String("provider", "", "Generation provider")
	model := flags.String("model", "", "actual formal model")
	session := flags.String("session", "", "Generation session")
	at := flags.String("generated-at", "", "RFC3339 UTC")
	prior := flags.String("prior-sha256", "", "expected sparse state SHA")
	priorTerminology := flags.String("prior-terminology-sha256", "", "expected terminology manifest file SHA")
	websiteRoot := flags.String("website-root", "", "clean pinned golang/website checkout")
	goSourceArchive := flags.String("go-source-archive", "", "pinned Go source archive")
	oldSHA := flags.String("old-glossary-sha256", "", "archived old glossary SHA")
	priorEvidence := flags.String("prior-evidence", "", "prior compatibility evidence")
	origin := flags.String("origin", "", "locale HTTPS origin")
	destination := flags.String("destination", "", "navigation destination")
	var units, generations, reviews, compatibility, pageFinalizations repeatedStrings
	flags.Var(&pageFinalizations, "page-finalization", "complete historical predecessor finalization (ordered, repeatable; membership composition only)")
	flags.Var(&units, "unit", "exact unit id (repeatable)")
	flags.Var(&generations, "generation", "generation artifact (repeatable)")
	flags.Var(&reviews, "review", "review artifact (repeatable)")
	flags.Var(&compatibility, "compatibility", "compatibility evidence (repeatable)")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	allowed := map[string]string{"authority-upgrade": "apply", "language-authority": "apply prior-sha256", "terminology-authority": "website-root go-source-archive apply prior-sha256 prior-terminology-sha256", "terminology-check": "", "corpus-check": "", "page-plan": "package", "structured-plan": "locale", "structured-export": "locale output task finding", "structured-check": "bundle", "structured-import": "bundle input provider model session generated-at", "package-closure": "locale package input generation", "surface-scope": "locale generation", "surface-record": "input", "campaign-plan": "locale", "inventory": "package", "inventory-summary": "package", "generation-plan": "locale package task finding", "review-plan": "locale package generation", "generation-export": "locale package task batch finding unit output plan-batch", "generation-check": "bundle", "generation-import": "bundle input provider model session generated-at", "review-scope": "locale package generation unit plan-batch", "review-export": "locale package generation unit output plan-batch", "review-check": "bundle", "review-record": "input", "finalize": "locale package generation review compatibility", "activation-preflight": "input", "activation-apply": "input prior-sha256", "coverage": "locale", "routing-check": "locale origin destination", "compatibility-assess": "locale package old-glossary-sha256 generation prior-evidence", "reconcile": "input bundle"}
	validFlags, known := allowed[args[0]]
	if args[0] == "package-composition" {
		validFlags, known = "locale package page-finalization", true
	}
	if args[0] == "package-closure" {
		validFlags += " page-finalization"
	}
	if args[0] == "generation-land" {
		validFlags, known = "locale package plan-batch generation", true
	}
	if !known {
		return fmt.Errorf("unknown site-content command %s", args[0])
	}
	var invalid error
	flags.Visit(func(f *flag.Flag) {
		if !strings.Contains(" "+validFlags+" ", " "+f.Name+" ") {
			invalid = fmt.Errorf("--%s is not valid for %s", f.Name, args[0])
		}
	})
	if invalid != nil {
		return invalid
	}
	// Legacy Site QC/revision/finalization artifacts remain readable, but new
	// TranslationUnits have exactly one active lifecycle in internal/i18n.
	switch args[0] {
	case "generation-export", "review-plan", "review-scope", "review-export", "review-record", "finalize":
		return fmt.Errorf("正式页面统一使用 retranslation / generation-bundle / quality-check --content-package learn-docs-v1；旧 Site 入口仅保留历史读取兼容")
	}
	if *planBatch < 1 {
		return fmt.Errorf("plan-batch must be positive")
	}
	if len(units) > 0 {
		var explicit bool
		flags.Visit(func(f *flag.Flag) {
			if f.Name == "plan-batch" {
				explicit = true
			}
		})
		if explicit {
			return fmt.Errorf("--unit and --plan-batch are mutually exclusive")
		}
	}
	sort.Strings(units)
	refs := func(paths []string) ([]sitecontent.Reference, error) {
		sort.Strings(paths)
		return siteContentReferencesInOrder(root, paths)
	}
	read := func(p string) ([]byte, error) { return sitecontent.ReadArtifact(root, p) }
	switch args[0] {
	case "generation-land":
		if *pkg != "learn-docs-v1" || len(generations) == 0 || len(generations)%2 != 0 {
			return fmt.Errorf("generation-land requires complete original bundle/result pairs")
		}
		sources := []i18n.TranslationUnitGenerationOrigin{}
		for i := 0; i < len(generations); i += 2 {
			b, err := sitecontent.ReferenceFile(root, generations[i])
			if err != nil {
				return err
			}
			r, err := sitecontent.ReferenceFile(root, generations[i+1])
			if err != nil {
				return err
			}
			sources = append(sources, i18n.TranslationUnitGenerationOrigin{Bundle: i18n.GlossaryArchiveReference{Path: b.Path, SHA256: b.SHA256}, Result: i18n.GlossaryArchiveReference{Path: r.Path, SHA256: r.SHA256}})
		}
		r, err := sitecontent.LandArchivedPageGeneration(root, *locale, *planBatch, sources)
		if err != nil {
			return err
		}
		return writeRetranslationProcessOutput(os.Stdout, r, false)
	case "terminology-authority":
		if *websiteRoot == "" || *goSourceArchive == "" {
			return fmt.Errorf("--website-root and --go-source-archive are required")
		}
		v, err := sitecontent.PrepareTerminologyAuthority(root, *websiteRoot, *goSourceArchive, *prior, *priorTerminology, *apply)
		if err != nil {
			return err
		}
		return printJSON(v)
	case "terminology-check":
		v, err := sitecontent.CheckTerminologyAuthority(root)
		if err != nil {
			return err
		}
		return printJSON(v)
	case "language-authority":
		if *prior != "" {
			v, err := sitecontent.RefreshSharedCorpusAuthority(root, *prior, *apply)
			if err != nil {
				return err
			}
			return printJSON(v)
		}
		v, err := sitecontent.InitializeSharedLanguageAuthority(root, *apply)
		if err != nil {
			return err
		}
		return printJSON(v)
	case "corpus-check":
		return sitecontent.CheckUnifiedCorpus(root)
	case "page-plan":
		g, err := sitecontent.LoadCurrent(root)
		if err != nil {
			return err
		}
		docs, raw, err := sitecontent.WorkflowDocuments(root, g, *pkg)
		if err != nil {
			return err
		}
		ids := []string{}
		for _, d := range docs {
			ids = append(ids, d.Units[0].ID)
		}
		sort.Strings(ids)
		p, err := sitecontent.PlanPageWorkingSets(docs, raw, ids, nil)
		if err != nil {
			return err
		}
		return printJSON(p)
	case "campaign-plan":
		p, err := sitecontent.PlanLocaleLanguage(root, *locale)
		if err != nil {
			return err
		}
		return printJSON(p)
	case "structured-plan":
		p, err := sitecontent.PlanStructuredAssets(root, *locale)
		if err != nil {
			return err
		}
		return printJSON(p)
	case "structured-export":
		var b []byte
		var err error
		if *task == "initial" && *finding == "" {
			b, _, err = sitecontent.ExportStructuredAssets(root, *locale)
		} else if *task == "revision" && *finding != "" {
			ref, e := sitecontent.ReferenceFile(root, *finding)
			if e != nil {
				return e
			}
			b, _, err = sitecontent.ExportStructuredAssetsRevision(root, *locale, ref)
		} else {
			return fmt.Errorf("structured task/finding mismatch")
		}
		if err != nil {
			return err
		}
		return sitecontent.SaveArtifact(root, *output, b)
	case "structured-check":
		b, err := read(*bundle)
		if err != nil {
			return err
		}
		_, err = sitecontent.CheckStructuredAssetsBundle(root, b)
		return err
	case "structured-import":
		b, err := read(*bundle)
		if err != nil {
			return err
		}
		o, err := read(*input)
		if err != nil {
			return err
		}
		_, p, err := sitecontent.ImportStructuredAssets(root, b, o, sitecontent.Provenance{Provider: *provider, Model: *model, Session: *session, GeneratedAt: *at})
		if err != nil {
			return err
		}
		return printJSON(p)
	case "package-composition":
		pf, err := siteContentReferencesInOrder(root, pageFinalizations)
		if err != nil {
			return err
		}
		plan, err := sitecontent.InspectPackageComposition(root, *locale, *pkg, pf)
		if err != nil {
			return err
		}
		return printJSON(plan)
	case "package-closure":
		rs, err := refs(generations)
		if err != nil {
			return err
		}
		if len(pageFinalizations) > 0 {
			if *input != "" {
				return fmt.Errorf("--input and --page-finalization are mutually exclusive")
			}
			pf, err := siteContentReferencesInOrder(root, pageFinalizations)
			if err != nil {
				return err
			}
			_, p, err := sitecontent.FinalizeComposedPackageClosure(root, *locale, *pkg, pf, rs)
			if err != nil {
				return err
			}
			return printJSON(p)
		}
		var page *sitecontent.Reference
		if *input != "" {
			r, err := sitecontent.ReferenceFile(root, *input)
			if err != nil {
				return err
			}
			page = &r
		}
		_, p, err := sitecontent.FinalizePackageClosure(root, *locale, *pkg, page, rs)
		if err != nil {
			return err
		}
		return printJSON(p)
	case "surface-scope":
		rs, err := refs(generations)
		if err != nil {
			return err
		}
		s, err := sitecontent.BuildIntegratedSurfaceScope(root, *locale, rs)
		if err != nil {
			return err
		}
		return printJSON(s)
	case "surface-record":
		b, err := read(*input)
		if err != nil {
			return err
		}
		_, p, err := sitecontent.RecordIntegratedSurface(root, b)
		if err != nil {
			return err
		}
		return printJSON(p)
	case "authority-upgrade":
		p, err := sitecontent.UpgradeFoundation(root, *apply)
		if err != nil {
			return err
		}
		return printJSON(p)
	case "inventory", "inventory-summary":
		g, err := sitecontent.LoadCurrent(root)
		if err != nil {
			return err
		}
		docs, raw, err := sitecontent.PackageDocuments(root, g, *pkg)
		if err != nil {
			return err
		}
		if args[0] == "inventory-summary" {
			stats, err := sitecontent.SummarizeUnits(docs, raw)
			if err != nil {
				return err
			}
			return printJSON(stats)
		}
		return printJSON(docs)
	case "generation-plan", "generation-export":
		var ref *sitecontent.Reference
		if *finding != "" {
			r, err := sitecontent.ReferenceFile(root, *finding)
			if err != nil {
				return err
			}
			ref = &r
		}
		var selected []string
		if len(units) > 0 {
			selected = units
		} else {
			plan, err := sitecontent.GenerationPlan(root, *locale, *pkg, *task, ref)
			if err != nil {
				return err
			}
			if args[0] == "generation-plan" {
				return printJSON(plan)
			}
			if *planBatch > len(plan.Sets) {
				return fmt.Errorf("plan-batch outside current Generation plan")
			}
			selected = plan.Sets[*planBatch-1].Selected
		}
		b, m, err := sitecontent.ExportPackageGeneration(root, *locale, *pkg, *task, *batch, selected, ref)
		if err != nil {
			return err
		}
		if err := sitecontent.SaveArtifact(root, *output, b); err != nil {
			return err
		}
		return printJSON(m)
	case "generation-check":
		b, err := read(*bundle)
		if err != nil {
			return err
		}
		m, err := sitecontent.CheckPackageGenerationBundle(root, b)
		if err != nil {
			return err
		}
		return printJSON(m)
	case "generation-import":
		b, err := read(*bundle)
		if err != nil {
			return err
		}
		m, err := sitecontent.CheckPackageGenerationBundle(root, b)
		if err != nil {
			return err
		}
		if m.Task != "initial" {
			return fmt.Errorf("精确修订与重试统一使用共享 retranslation 生命周期，不写新 Site 记录")
		}
		o, err := read(*input)
		if err != nil {
			return err
		}
		g, name, err := sitecontent.ImportPackageGeneration(root, b, o, sitecontent.Provenance{Provider: *provider, Model: *model, Session: *session, GeneratedAt: *at})
		if err != nil {
			return err
		}
		_ = name
		r, err := sitecontent.ProcessImportedPageGeneration(root, g, o)
		if err != nil {
			return err
		}
		return writeRetranslationProcessOutput(os.Stdout, r, false)
	case "review-plan", "review-scope", "review-export":
		gr, err := refs(generations)
		if err != nil {
			return err
		}
		if len(units) == 0 {
			plan, err := sitecontent.ReviewerPlan(root, *locale, *pkg, gr)
			if err != nil {
				return err
			}
			if args[0] == "review-plan" {
				return printJSON(plan)
			}
			if *planBatch > len(plan.Sets) {
				return fmt.Errorf("plan-batch outside current Reviewer plan")
			}
			units = plan.Sets[*planBatch-1].Selected
		}
		r, err := sitecontent.ReviewScope(root, *locale, *pkg, gr, units)
		if err != nil {
			return err
		}
		if args[0] == "review-scope" {
			return printJSON(r)
		}
		b, err := sitecontent.ExportPackageReview(root, r)
		if err != nil {
			return err
		}
		return sitecontent.SaveArtifact(root, *output, b)
	case "review-check":
		b, err := read(*bundle)
		if err != nil {
			return err
		}
		return sitecontent.CheckPackageReviewBundle(root, b)
	case "review-record":
		b, err := read(*input)
		if err != nil {
			return err
		}
		_, name, err := sitecontent.RecordPackageReview(root, b)
		if err != nil {
			return err
		}
		return printJSON(name)
	case "finalize":
		gr, err := refs(generations)
		if err != nil {
			return err
		}
		rr, err := refs(reviews)
		if err != nil {
			return err
		}
		cr, err := refs(compatibility)
		if err != nil {
			return err
		}
		f, name, err := sitecontent.FinalizePackage(root, *locale, *pkg, gr, rr, cr)
		if err != nil {
			return err
		}
		return printJSON(struct {
			Path     string `json:"path"`
			Identity string `json:"identity"`
		}{name, f.Identity})
	case "activation-preflight", "activation-apply":
		ref, err := sitecontent.ReferenceFile(root, *input)
		if err != nil {
			return err
		}
		if args[0] == "activation-preflight" {
			p, err := sitecontent.ActivationPreflight(root, ref)
			if err != nil {
				return err
			}
			return printJSON(p)
		}
		p, err := sitecontent.ApplyActivation(root, ref, *prior)
		if err != nil {
			return err
		}
		return printJSON(p)
	case "coverage":
		g, l, err := sitecontent.LocalCoverage(root, *locale)
		if err != nil {
			return err
		}
		return printJSON(sitecontent.PackageStates(g, l))
	case "routing-check":
		r, err := site.Load(root, *locale, *origin)
		if err != nil {
			return err
		}
		if *destination != "" {
			d, err := r.Destination(*destination)
			if err != nil {
				return err
			}
			return printJSON(d)
		}
		b, err := r.Sitemap()
		if err != nil {
			return err
		}
		fmt.Print(string(b))
		return nil
	case "compatibility-assess":
		gr, err := refs(generations)
		if err != nil {
			return err
		}
		var p *sitecontent.Reference
		if *priorEvidence != "" {
			r, err := sitecontent.ReferenceFile(root, *priorEvidence)
			if err != nil {
				return err
			}
			p = &r
		}
		e, name, err := sitecontent.AssessPackageCompatibility(root, *locale, *pkg, *oldSHA, gr, p)
		if err != nil {
			return err
		}
		return printJSON(struct {
			Path     string                            `json:"path"`
			Evidence *sitecontent.PackageCompatibility `json:"evidence"`
		}{name, e})
	case "reconcile":
		a, err := read(*input)
		if err != nil {
			return err
		}
		b, err := read(*bundle)
		if err != nil {
			return err
		}
		var old, next []sitecontent.Document
		if err := sitecontent.StrictJSON(a, &old); err != nil {
			return err
		}
		if err := sitecontent.StrictJSON(b, &next); err != nil {
			return err
		}
		r, err := sitecontent.ReconcileDocuments(old, next)
		if err != nil {
			return err
		}
		return printJSON(r)
	default:
		return fmt.Errorf("unknown site-content command %s", args[0])
	}
}
