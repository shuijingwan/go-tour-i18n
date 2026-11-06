package sitecontent

import (
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func closureStructuralFixture(t *testing.T) (*closureValidation, Reference) {
	t.Helper()
	root := t.TempDir()
	base, current, catalog := historicalRegistrationFixture(t)
	for _, g := range []*Global{base, current} {
		g.Packages[0].ID = "learn-docs-v1"
		g.Sources[0].Package = "learn-docs-v1"
	}
	current.HistoricalPackages[0] = base.Packages[0]
	catalog.TranslationPackages[0].ID = "learn-docs-v1"
	if err := registerStructuredHistoricalPackages(base, current, catalog); err != nil {
		t.Fatal(err)
	}
	b, _ := Encode(current)
	writeFixture(t, root, GlobalPath, b)
	l := Locale{Schema: LocaleSchema, Locale: "zh-CN", Packages: []Completion{}}
	b, _ = Encode(l)
	writeFixture(t, root, LocalePath(l.Locale), b)
	f := i18n.QualityCheckFinalization{SchemaVersion: 1, EvidenceType: i18n.QualityCheckFinalizationEvidenceType, Locale: l.Locale, SnapshotID: "synthetic-qc-001"}
	b, _ = Encode(f)
	name := "data/quality-check-snapshots/zh-CN/synthetic-qc-001/finalization.json"
	writeFixture(t, root, name, b)
	page := Reference{name, digest(b)}
	c := PackageClosure{Schema: ClosureSchema, Locale: l.Locale, Package: "learn-docs-v1", PackageSHA: base.Packages[0].Identity, PageFinalization: &page, Structured: []Reference{}, Files: []Reference{}}
	c.Identity = identity(c)
	b, _ = Encode(c)
	name = workflowPath(c.Locale, c.Package, "closure", c.Identity)
	writeFixture(t, root, name, b)
	v := newClosureValidation(root)
	v.global, v.catalog = current, catalog // immutable context supplied by this isolated unit fixture
	return v, Reference{name, digest(b)}
}

func TestClosureStructuralPreflightRejectsBeforeExpensiveGraph(t *testing.T) {
	for _, label := range []string{"wrong-package", "unregistered-history", "wrong-history-SHA", "wrong-finalization-locale", "wrong-reference-SHA"} {
		t.Run(label, func(t *testing.T) {
			v, ref := closureStructuralFixture(t)
			pkg := "learn-docs-v1"
			switch label {
			case "wrong-package":
				pkg = "unregistered-package"
			case "unregistered-history":
				v.catalog.HistoricalTranslationPackages = nil
			case "wrong-history-SHA":
				v.catalog.HistoricalTranslationPackages[0].AuthoritySHA256 = digest([]byte("wrong"))
			case "wrong-reference-SHA":
				ref.SHA256 = strings.Repeat("0", 64)
			case "wrong-finalization-locale":
				var c PackageClosure
				_ = readReference(v.root, ref, &c)
				var f i18n.QualityCheckFinalization
				_ = readReference(v.root, *c.PageFinalization, &f)
				f.Locale = "de-DE"
				b, _ := Encode(f)
				writeFixture(t, v.root, c.PageFinalization.Path, b)
			}
			if _, _, err := v.buildPackageClosure(v.root, "zh-CN", pkg, nil, nil, &ref); err == nil {
				t.Fatal("impossible closure accepted")
			}
			for key, count := range v.counts {
				if strings.HasPrefix(key, "closure:") || strings.HasPrefix(key, "generation:") || key == "current-corpus" || key == "current-review" {
					if count > 0 {
						t.Fatal("expensive validation ran before structural rejection", key, count)
					}
				}
			}
		})
	}
}

func TestClosureStructuralPreflightDoesNotGrantValidation(t *testing.T) {
	v, ref := closureStructuralFixture(t)
	if err := v.preflightClosureInputs("zh-CN", "learn-docs-v1", nil, nil, nil, &ref); err != nil {
		t.Fatal(err)
	}
	if _, _, err := v.buildPackageClosure(v.root, "zh-CN", "learn-docs-v1", nil, nil, &ref); err == nil || v.counts["current-corpus"] != 1 {
		t.Fatal("structural header replaced mandatory complete current corpus validation", err, v.counts)
	}
	// Successful structural inspection never seals mutable bytes. Same-path
	// changes and wrong SHA references must be rejected on reuse and at return.
	b, _ := readRegular(v.root, ref.Path)
	writeFixture(t, v.root, ref.Path, append(b, ' '))
	if err := v.preflightClosureInputs("zh-CN", "learn-docs-v1", nil, nil, nil, &ref); err == nil {
		t.Fatal("same-path mutation accepted")
	}
	if err := v.verify(); err == nil {
		t.Fatal("concurrent mutation escaped final fresh check")
	}
}
