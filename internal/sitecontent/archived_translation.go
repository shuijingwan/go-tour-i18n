package sitecontent

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// ProcessImportedPageGeneration adapts an authentic legacy JSON invocation to
// normal shared processing. A 29/1 result is installed and reported intact.
func ProcessImportedPageGeneration(root string, g *Generation, output []byte) (*i18n.RetranslationProcessResult, error) {
	if g.Batch.Task != "initial" {
		return nil, fmt.Errorf("new revision/retry uses the shared retranslation lifecycle")
	}
	resultPath := "data/generation-archives/" + g.Batch.Locale + "/" + digest(output) + ".json"
	if err := SaveArtifact(root, resultPath, output); err != nil {
		return nil, err
	}
	batch := 0
	for _, d := range g.Batch.Documents {
		n := (d.StableIndex-1)/30 + 1
		if batch != 0 && batch != n {
			return nil, fmt.Errorf("mixed fixed Page batch")
		}
		batch = n
	}
	return landArchivedPageGeneration(root, g.Batch.Locale, batch, []i18n.TranslationUnitGenerationOrigin{{Bundle: i18n.GlossaryArchiveReference{Path: g.Bundle.Path, SHA256: g.Bundle.SHA256}, Result: i18n.GlossaryArchiveReference{Path: resultPath, SHA256: digest(output)}}}, false)
}

// LandArchivedPageGeneration is a compatibility adapter for original complete
// JSON transports. No subset input bundle or partial acceptance is created.
// Archives preserve every original member, including failed language outputs.
func LandArchivedPageGeneration(root, locale string, planBatch int, sources []i18n.TranslationUnitGenerationOrigin) (*i18n.RetranslationProcessResult, error) {
	return landArchivedPageGeneration(root, locale, planBatch, sources, true)
}

func landArchivedPageGeneration(root, locale string, planBatch int, sources []i18n.TranslationUnitGenerationOrigin, validOnly bool) (*i18n.RetranslationProcessResult, error) {
	if planBatch < 1 || planBatch > 2 || len(sources) == 0 {
		return nil, fmt.Errorf("invalid original invocation scope")
	}
	catalog, err := i18n.ReadCatalog(root)
	if err != nil {
		return nil, err
	}
	current, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		return nil, err
	}
	if err := i18n.HydrateCatalogSources(catalog, current); err != nil {
		return nil, err
	}
	ids, err := AttachPageTranslationUnits(root, catalog)
	if err != nil {
		return nil, err
	}
	if err := catalog.UseTranslationUnitScope(ids); err != nil {
		return nil, err
	}
	glossary, err := i18n.LoadGlossary(root, locale)
	if err != nil {
		return nil, err
	}
	if err := requireUnifiedReview(root, locale); err != nil {
		return nil, err
	}
	expected := map[string]PageAdapter{}
	selected := []string{}
	for _, v := range catalog.Adapted {
		if (v.StableIndex-1)/30+1 == planBatch {
			expected[v.Unit.ID] = v.Adapter.(PageAdapter)
			selected = append(selected, v.Unit.ID)
		}
	}
	outputs := map[string][]byte{}
	origins := map[string]i18n.TranslationUnitGenerationOrigin{}
	archives := map[string][]byte{}
	for _, origin := range sources {
		bundle, err := readRegular(root, origin.Bundle.Path)
		if err != nil || digest(bundle) != origin.Bundle.SHA256 {
			return nil, fmt.Errorf("original bundle hash/path mismatch")
		}
		result, err := readRegular(root, origin.Result.Path)
		if err != nil || digest(result) != origin.Result.SHA256 {
			return nil, fmt.Errorf("original result hash/path mismatch")
		}
		files, err := i18n.ReadTransportBundle(bundle, 256, 32<<20)
		if err != nil {
			return nil, err
		}
		var m Batch
		if err := StrictJSON(files["manifest.json"], &m); err != nil {
			return nil, err
		}
		if err := i18n.ValidateTransportBundleInventory(files, m.Files, true); err != nil {
			return nil, err
		}
		var out Output
		if err := StrictJSON(result, &out); err != nil {
			return nil, err
		}
		copy := m
		copy.Identity = ""
		if m.Identity != identity(copy) || m.Locale != locale || m.Package != "learn-docs-v1" || m.Schema != WorkflowSchema || out.Schema != WorkflowSchema || out.BatchSHA != m.Identity || len(out.Targets) != len(m.Selected) {
			return nil, fmt.Errorf("original invocation identity/exact-set mismatch")
		}
		for i, t := range out.Targets {
			if t.ID != m.Selected[i] {
				return nil, fmt.Errorf("original invocation order/exact-set mismatch")
			}
		}
		archived := i18n.TranslationUnitGenerationOrigin{Bundle: i18n.GlossaryArchiveReference{Path: "data/generation-archives/" + locale + "/" + digest(bundle) + ".zip", SHA256: digest(bundle)}, Result: i18n.GlossaryArchiveReference{Path: "data/generation-archives/" + locale + "/" + digest(result) + ".json", SHA256: digest(result)}}
		archives[archived.Bundle.Path], archives[archived.Result.Path] = bundle, result
		for _, d := range m.Documents {
			if len(d.Units) != 1 {
				return nil, fmt.Errorf("archived internal slots cannot become TranslationUnits")
			}
			a, ok := expected[d.Units[0].ID]
			if !ok {
				continue
			}
			if d.Path != a.Document.Path || d.Route != a.Document.Route || !bytes.Equal(files["source/"+d.Path], a.Source) || !reflect.DeepEqual(d.Units, a.Document.Units) {
				continue
			}
			for _, t := range out.Targets {
				if t.ID != d.Units[0].ID {
					continue
				}
				if validOnly {
					candidate, err := a.Restore(t.Text)
					if err != nil {
						continue
					}
					if err := a.Validate(candidate, glossary); err != nil {
						continue
					}
				}
				// Verify original glossary identity as well as exact formal input.
				gb, err := readRegular(root, "locales/"+locale+"/glossary.yaml")
				if err != nil {
					return nil, err
				}
				if !bytes.Equal(gb, files["formal/glossary.yaml"]) || digest(gb) != m.GlossarySHA {
					continue
				}
				raw := append(bytes.TrimRight([]byte(t.Text), "\n"), '\n')
				if old, exists := outputs[t.ID]; exists && !bytes.Equal(old, raw) {
					return nil, fmt.Errorf("conflicting valid original candidates: %s", d.Path)
				}
				if _, exists := outputs[t.ID]; !exists {
					outputs[t.ID], origins[t.ID] = raw, archived
				}
			}
		}
	}
	if len(outputs) != len(expected) {
		return nil, fmt.Errorf("可信且输入未变的页面不足：%d/%d；不得重做已有可信页面", len(outputs), len(expected))
	}
	// Only after a complete safe preflight: retain full original transports,
	// export a mechanical shared batch, bind each raw to its original invocation.
	for p, b := range archives {
		if err := SaveArtifact(root, p, b); err != nil {
			return nil, err
		}
	}
	exported, err := i18n.ExportRetranslationBatch(root, catalog, i18n.RetranslationExportOptions{Locale: locale, Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: selected, Limit: 30})
	if err != nil {
		return nil, err
	}
	if err := i18n.LandExistingGenerationBatch(root, catalog, locale, exported.BatchID, origins, outputs); err != nil {
		return nil, err
	}
	return i18n.ProcessRetranslationBatch(root, catalog, i18n.RetranslationProcessOptions{Locale: locale, BatchID: exported.BatchID})
}
