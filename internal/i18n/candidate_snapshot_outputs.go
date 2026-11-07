package i18n

import (
	"bytes"
	"fmt"
	"reflect"
)

// Reuse the existing process projection/authority boundary, not a second
// comment parser or cache. Only this Snapshot invocation owns these outputs.
func prepareSnapshotRevisionOutputs(root string, c *Catalog, locale string, g *Glossary, latest *latestRetranslationUnits, members ...map[string]*Catalog) (map[string]protectedTranslation, []func() error, error) {
	scopes := map[string]*Catalog{}
	if len(members) > 0 {
		scopes = members[0]
	}
	outputs := map[string]protectedTranslation{}
	checks := []func() error{}
	seen := map[string]bool{}
	// Ordinary successors retain their selected ancestor's response decoder,
	// while their QC authority remains the new Snapshot's independent finding.
	for _, u := range latest.ordered {
		choice := latest.selectedByID[u.ID]
		if choice.machine != nil || choice.manifest.RepresentationBridge != nil || !hasInheritedBridgePredecessor(choice.manifest) {
			continue
		}
		evidenceCatalog := snapshotEvidenceCatalog(scopes, c, u.ID)
		decoder, lineage, err := surfaceTransportOutput(root, evidenceCatalog, locale, choice.manifest)
		ownSurface := lineage != nil
		if err == nil && lineage == nil {
			decoder, lineage, err = inheritedBridgeOutput(root, evidenceCatalog, g, locale, choice.manifest)
		}
		if err != nil {
			return nil, nil, err
		}
		if lineage == nil {
			continue
		}
		if ownSurface {
			outputs[u.ID] = decoder
		}
		base, err := repositoryRelativePath(root, choice.batchDir)
		if err != nil {
			return nil, nil, err
		}
		vb, err := readSnapshotReferencedFile(root, base+"/"+choice.result.ValidationPath)
		if err != nil {
			return nil, nil, err
		}
		v, err := decodeRetranslationValidation(vb, u)
		if err != nil {
			return nil, nil, err
		}
		rawPath := base + "/" + v.RawResponsePath
		raw, err := readSnapshotReferencedFile(root, rawPath)
		if err != nil {
			return nil, nil, err
		}
		identity, err := currentTranslationAuthority(root, u, locale)
		if err != nil {
			return nil, nil, err
		}
		checks = append(checks, func() error {
			if err := lineage.check(root, locale); err != nil {
				return err
			}
			got, err := readSnapshotReferencedFile(root, rawPath)
			if err != nil || !bytes.Equal(raw, got) {
				return fmt.Errorf("Snapshot inherited selected raw drift")
			}
			got, err = readSnapshotReferencedFile(root, base+"/"+choice.result.ValidationPath)
			if err != nil || !bytes.Equal(vb, got) {
				return fmt.Errorf("Snapshot inherited selected validation drift")
			}
			authority, err := currentTranslationAuthority(root, u, locale)
			if err != nil || authority != identity {
				return fmt.Errorf("Snapshot inherited current authority drift")
			}
			return nil
		})
	}
	// Capture the validation-owned response before validation begins. Retried
	// Units may select a retries/ path rather than the original raw response;
	// the reused process context intentionally binds only the original input.
	selectedResponses := map[string][]byte{}
	for _, u := range latest.ordered {
		choice := latest.selectedByID[u.ID]
		if choice.machine != nil || (choice.manifest.RevisionFeedbackSource != CurrentValidationRevisionFeedback && choice.manifest.RepresentationBridge == nil) {
			continue
		}
		base, err := repositoryRelativePath(root, choice.batchDir)
		if err != nil {
			return nil, nil, err
		}
		path := base + "/" + choice.result.ValidationPath
		b, err := readSnapshotReferencedFile(root, path)
		if err != nil {
			return nil, nil, err
		}
		v, err := decodeRetranslationValidation(b, u)
		if err != nil {
			return nil, nil, err
		}
		selectedResponses[path] = bytes.Clone(b)
		path = base + "/" + v.RawResponsePath
		b, err = readSnapshotReferencedFile(root, path)
		if err != nil {
			return nil, nil, err
		}
		selectedResponses[path] = bytes.Clone(b)
	}
	if len(selectedResponses) > 0 {
		checks = append(checks, func() error {
			for path, want := range selectedResponses {
				got, err := readSnapshotReferencedFile(root, path)
				if err != nil || !bytes.Equal(got, want) {
					return fmt.Errorf("Snapshot selected response changed: %s", path)
				}
			}
			return nil
		})
	}
	for _, u := range latest.ordered {
		choice := latest.selectedByID[u.ID]
		if choice.machine != nil || (choice.manifest.RevisionFeedbackSource != CurrentValidationRevisionFeedback && choice.manifest.RepresentationBridge == nil) || seen[choice.batchID] {
			continue
		}
		seen[choice.batchID] = true
		evidenceCatalog := snapshotEvidenceCatalog(scopes, c, u.ID)
		m, err := readRetranslationProcessManifest(choice.batchDir, locale, choice.batchID)
		if err != nil {
			return nil, nil, err
		}
		evidenceCatalog, err = snapshotRevisionEvidenceCatalog(root, c, locale, choice, evidenceCatalog)
		if err != nil {
			return nil, nil, err
		}
		if choice.manifest.RepresentationBridge != nil {
			for _, record := range m.Units {
				selected, ok := latest.selectedByID[record.UnitID]
				if !ok || selected.batchID != choice.batchID {
					continue
				}
				if !reflect.DeepEqual(record, selected.manifest) {
					return nil, nil, fmt.Errorf("Snapshot bridge manifest drift")
				}
				p, err := frozenBridgeOutput(root, evidenceCatalog, g, locale, record)
				if err != nil {
					return nil, nil, err
				}
				outputs[record.UnitID] = p
				ref := *record.RepresentationBridge
				r, err := readFrozenSameUnitBridge(root, locale, ref)
				if err != nil {
					return nil, nil, err
				}
				u, err := c.Unit(record.UnitID)
				if err != nil {
					return nil, nil, err
				}
				identity, err := currentTranslationAuthority(root, u, locale)
				if err != nil {
					return nil, nil, err
				}
				checks = append(checks, func() error {
					fresh, err := readFrozenSameUnitBridge(root, locale, ref)
					if err != nil || !reflect.DeepEqual(r, fresh) {
						return fmt.Errorf("Snapshot frozen bridge drift: %v", err)
					}
					got, err := currentTranslationAuthority(root, u, locale)
					if err != nil || identity != got {
						return fmt.Errorf("Snapshot current bridge validator drift")
					}
					return nil
				})
			}
			continue
		}
		projections, check, err := prepareProcessOutputContext(choice.batchDir, evidenceCatalog, g, m)
		if err != nil {
			return nil, nil, err
		}
		for _, record := range m.Units {
			selected, ok := latest.selectedByID[record.UnitID]
			if !ok || selected.batchID != choice.batchID {
				continue
			}
			if !reflect.DeepEqual(record, selected.manifest) {
				return nil, nil, fmt.Errorf("Snapshot revision manifest changed during selection")
			}
			if p := projections[record.UnitID]; p.Adapter != nil {
				outputs[record.UnitID] = p
			} else if m.TransportRecovery != nil {
				// R includes ordinary siblings. Their decoder is ordinary, while
				// the verified context remains the complete imported approved
				// invocation; do not rebuild that context without its selection.
				u, err := evidenceCatalog.Unit(record.UnitID)
				if err != nil {
					return nil, nil, err
				}
				p, err := prepareTranslationUnitInput(u, g)
				if err != nil {
					return nil, nil, err
				}
				outputs[record.UnitID] = p
			}
		}
		if check != nil {
			checks = append(checks, check)
		}
	}
	return outputs, checks, nil
}

func snapshotPreparedOutput(outputs []map[string]protectedTranslation, id string) *protectedTranslation {
	if len(outputs) > 0 {
		if p, ok := outputs[0][id]; ok {
			return &p
		}
	}
	return nil
}

// The manifest refers to bytes, not cached PASS. Bind the selected current
// evidence and reselect from disk immediately before the write-once commit.
func snapshotSelectionCheck(root string, c *Catalog, locale string, latest *latestRetranslationUnits, s *QualityCheckSnapshotManifest) (func() error, error) {
	refs := map[string][]byte{}
	bind := func(path, sha string) error {
		b, err := readSnapshotReferencedFile(root, path)
		if err != nil || (sha != "" && sum(b) != sha) {
			return fmt.Errorf("Snapshot evidence changed before commit: %s", path)
		}
		refs[path] = bytes.Clone(b)
		return nil
	}
	if err := bind(s.GlossaryPath, s.GlossarySHA256); err != nil {
		return nil, err
	}
	for _, u := range s.Units {
		if err := bind(u.CandidatePath, u.CandidateSHA256); err != nil {
			return nil, err
		}
		if err := bind(u.ValidationPath, u.ValidationSHA256); err != nil {
			return nil, err
		}
		unit, err := c.Unit(u.UnitID)
		if err != nil {
			return nil, err
		}
		b, err := readTranslationUnitSource(root, unit)
		if err != nil {
			return nil, err
		}
		if err := validateSnapshotSourceIdentity(c, unit, b); err != nil {
			return nil, err
		}
		if err := bind(u.SourcePath, sum(b)); err != nil {
			return nil, err
		}
		choice := latest.selectedByID[u.UnitID]
		if choice.machine == nil {
			base, err := repositoryRelativePath(root, choice.batchDir)
			if err != nil {
				return nil, err
			}
			for _, path := range []string{base + "/manifest.json", base + "/result.json", base + "/" + choice.manifest.InputPath} {
				if err := bind(path, ""); err != nil {
					return nil, err
				}
			}
		}
	}
	scope := append([]string(nil), c.TranslationScope...)
	return func() error {
		if !reflect.DeepEqual(scope, c.TranslationScope) {
			return fmt.Errorf("Snapshot package scope changed before commit")
		}
		for path, want := range refs {
			got, err := readSnapshotReferencedFile(root, path)
			if err != nil || !bytes.Equal(got, want) {
				return fmt.Errorf("Snapshot dependency changed before commit: %s", path)
			}
		}
		fresh, err := selectLatestRetranslationUnits(root, c, locale)
		if err != nil {
			return err
		}
		if len(fresh.ordered) != len(latest.ordered) || fresh.pageCount != latest.pageCount || fresh.exampleCount != latest.exampleCount {
			return fmt.Errorf("Snapshot selected exact set changed")
		}
		for i, u := range latest.ordered {
			other := fresh.ordered[i]
			if u.ID != other.ID || u.Kind != other.Kind || u.SourcePath != other.SourcePath || u.SourceSHA256 != other.SourceSHA256 {
				return fmt.Errorf("Snapshot selected source/order changed")
			}
			a, b := latest.selectedByID[u.ID], fresh.selectedByID[u.ID]
			if a.batchID != b.batchID || a.artifactEOF != b.artifactEOF || !reflect.DeepEqual(a.manifest, b.manifest) || !reflect.DeepEqual(a.result, b.result) || (a.machine == nil) != (b.machine == nil) {
				return fmt.Errorf("Snapshot latest candidate changed: %s", u.ID)
			}
			if a.machine != nil && (!reflect.DeepEqual(a.machine.unit, b.machine.unit) || a.machine.ref != b.machine.ref) {
				return fmt.Errorf("Snapshot machine successor changed: %s", u.ID)
			}
		}
		return nil
	}, nil
}
