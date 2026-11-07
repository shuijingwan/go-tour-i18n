package i18n

import (
	"fmt"
	"reflect"
)

// The source provider has already verified package-membership-successor/v1.
// Consume that explicit ordered predecessor projection, never a path union or
// count-derived scope. These catalogs decode evidence, not future scheduling.
func snapshotMembershipCatalogs(root string, c *Catalog) (map[string]*Catalog, func() error, error) {
	out := map[string]*Catalog{}
	if c == nil || len(c.TranslationScope) == 0 {
		return out, nil, nil
	}
	pkg, authority, _, ordered, err := translationAuthorizationScope(c)
	if err != nil {
		return nil, nil, err
	}
	var membership *TranslationPackageMembership
	for _, d := range c.TranslationPackages {
		if d.ID == pkg && d.AuthoritySHA256 == authority {
			membership = d.Membership
		}
	}
	if membership == nil {
		return out, nil, nil
	}
	if len(membership.Predecessors) < 2 || len(membership.References) == 0 {
		return nil, nil, fmt.Errorf("Snapshot membership predecessor authority missing")
	}
	declared := *membership
	declared.Predecessors = append([]TranslationPackagePredecessor(nil), membership.Predecessors...)
	declared.References = append([]GlossaryArchiveReference(nil), membership.References...)
	check := func() error {
		if !reflect.DeepEqual(*membership, declared) {
			return fmt.Errorf("Snapshot membership declaration changed")
		}
		for _, r := range declared.References {
			b, err := readSnapshotReferencedFile(root, r.Path)
			if err != nil || !validSHA256(r.SHA256) || sum(b) != r.SHA256 {
				return fmt.Errorf("Snapshot membership dependency changed: %s", r.Path)
			}
		}
		return nil
	}
	if err := check(); err != nil {
		return nil, nil, err
	}
	next := 0
	seen := map[string]bool{}
	for _, p := range declared.Predecessors {
		key := p.ID + ":" + p.AuthoritySHA256
		if seen[key] {
			return nil, nil, fmt.Errorf("duplicate Snapshot membership predecessor")
		}
		seen[key] = true
		h, err := HistoricalTranslationPackageCatalog(c, p.ID, p.AuthoritySHA256)
		if err != nil {
			return nil, nil, err
		}
		_, _, _, members, err := translationAuthorizationScope(h)
		if err != nil {
			return nil, nil, err
		}
		for _, u := range members {
			if next >= len(ordered) || out[u.ID] != nil {
				return nil, nil, fmt.Errorf("Snapshot membership overlap/extra Unit")
			}
			current := ordered[next]
			if current.ID != u.ID || current.Kind != u.Kind || current.SourcePath != u.SourcePath || current.SourceSHA256 != u.SourceSHA256 {
				return nil, nil, fmt.Errorf("Snapshot membership predecessor ordered source mismatch")
			}
			out[u.ID] = h
			next++
		}
	}
	if next != len(ordered) {
		return nil, nil, fmt.Errorf("Snapshot membership predecessor incomplete exact set")
	}
	return out, check, nil
}

func snapshotEvidenceCatalog(scopes map[string]*Catalog, current *Catalog, id string) *Catalog {
	if c := scopes[id]; c != nil {
		return c
	}
	return current
}

// Current-validation revisions retain the complete package of their frozen
// Snapshot. A membership child is authority for historical candidate evidence,
// not a substitute scope for that revision's decoder or invocation validation.
func snapshotRevisionEvidenceCatalog(root string, current *Catalog, locale string, choice selectedRetranslationUnit, fallback *Catalog) (*Catalog, error) {
	if choice.machine != nil || choice.manifest.RevisionFeedbackSource != CurrentValidationRevisionFeedback {
		return fallback, nil
	}
	inventory := *current
	inventory.TranslationScope = nil
	frozen, _, err := selectSnapshotTranslationPackage(root, &inventory, locale, choice.manifest.PreviousSnapshotID)
	return frozen, err
}

// Reuse sealed shared current authorization only for its exact accepted
// machine candidate. New revision candidates receive no rating or A carry.
func snapshotMembershipAssessments(root, locale string, members map[string]*Catalog, latest *latestRetranslationUnits, required ...map[*Catalog]string) (map[*Catalog]*CurrentTranslationAuthorization, func() error, error) {
	proofs := map[*Catalog]*CurrentTranslationAuthorization{}
	for _, c := range members {
		if _, done := proofs[c]; done {
			continue
		}
		_, _, scope, _, err := translationAuthorizationScope(c)
		if err != nil {
			return nil, nil, err
		}
		previous := ""
		if len(required) > 0 {
			previous = required[0][c]
		}
		for id, choice := range latest.selectedByID {
			r := choice.manifest
			if members[id] != c || choice.machine != nil || r.RevisionFeedbackSource != "surface_review" {
				continue
			}
			inventory := *c
			inventory.TranslationScope = nil
			h, registered, err := SelectFinalizedTranslationPackage(root, &inventory, locale, r.PreviousSnapshotID)
			if err != nil {
				return nil, nil, err
			}
			_, _, selected, _, err := translationAuthorizationScope(h)
			if err != nil {
				return nil, nil, err
			}
			if !registered || selected != scope {
				continue
			}
			if previous != "" && previous != r.PreviousSnapshotID {
				return nil, nil, fmt.Errorf("ambiguous membership Surface finalized predecessor")
			}
			previous = r.PreviousSnapshotID
		}
		proofs[c] = nil
		if previous != "" {
			a, err := RequireCurrentTranslationAuthorization(root, c, locale, previous)
			if err != nil {
				return nil, nil, err
			}
			proofs[c] = a
		} else {
			// A sibling without a revision selects only an exact complete accepted
			// candidate set through the existing Snapshot re-attestation reader.
			memberLatest := *latest
			memberLatest.ordered, memberLatest.pageCount, memberLatest.exampleCount, err = localeWorkflowUnitList(c)
			if err != nil {
				return nil, nil, err
			}
			a, err := selectedSnapshotReattestation(root, c, locale, &memberLatest)
			if err != nil {
				return nil, nil, err
			}
			if a != nil && !a.Ready() {
				return nil, nil, fmt.Errorf("Snapshot membership predecessor not currently compatible")
			}
			proofs[c] = a
		}
	}
	return proofs, func() error {
		for c, a := range proofs {
			if a == nil {
				continue
			}
			fresh, err := RequireCurrentTranslationAuthorization(root, c, locale, a.Historical.Finalization.SnapshotID)
			if err != nil || !reflect.DeepEqual(fresh, a) {
				return fmt.Errorf("Snapshot membership current authorization changed: %v", err)
			}
		}
		return nil
	}, nil
}

func validateSnapshotMembershipCandidate(root string, current *Catalog, locale string, g *Glossary, u *TranslationUnit, choice selectedRetranslationUnit, output *protectedTranslation, members map[string]*Catalog, proofs map[*Catalog]*CurrentTranslationAuthorization, compatibility *PreparedGlossaryCompatibility) (*validatedRetranslationCandidate, error) {
	c := snapshotEvidenceCatalog(members, current, u.ID)
	if c == current {
		return validateSelectedRetranslationCandidateWithOutput(root, c, locale, g, u, choice, output, compatibility)
	}
	if proofs[c] != nil {
		a := proofs[c]
		if !a.Ready() {
			return nil, fmt.Errorf("Snapshot membership current authorization not Ready")
		}
		var old QualityCheckSnapshotUnit
		if choice.machine != nil {
			old = choice.machine.snapshot(1)
		}
		matched := false
		reviewerGlossary := ""
		for _, accepted := range a.Units {
			old.Index = accepted.Snapshot.Index
			match := choice.machine != nil && qualityCheckSnapshotIdentityMatches(old, accepted.Snapshot)
			if choice.machine == nil {
				match = choiceMatchesSnapshot(locale, choice, accepted.Snapshot) && accepted.Snapshot.UnitID == u.ID
			}
			if match {
				old = accepted.Snapshot
				reviewerGlossary = accepted.ReviewerGlossarySHA
				matched = true
				break
			}
		}
		if !matched && choice.machine != nil {
			return nil, fmt.Errorf("Snapshot machine candidate differs from membership current authorization")
		}
		if matched {
			if !selectedRetranslationIdentityMatches(u, choice) || (choice.machine == nil && choice.result.Status != "passed") {
				return nil, fmt.Errorf("Snapshot accepted selection source/status changed")
			}
			candidate, err := readSnapshotReferencedFile(root, old.CandidatePath)
			if err != nil || sum(candidate) != old.CandidateSHA256 {
				return nil, fmt.Errorf("Snapshot membership accepted candidate changed")
			}
			machines := map[string]*machineSelection{}
			if choice.machine != nil {
				machines[choice.machine.ref.SHA256+u.ID] = choice.machine
			}
			if err := verifyAcceptedFrozenInputWithMachines(root, locale, u, old, candidate, reviewerGlossary, machines, c); err != nil {
				return nil, err
			}
			if err := ValidateTranslationUnitCandidate(root, current, u.ID, locale, candidate); err != nil {
				return nil, err
			}
			vb, err := readSnapshotReferencedFile(root, old.ValidationPath)
			if err != nil || sum(vb) != old.ValidationSHA256 {
				return nil, fmt.Errorf("Snapshot membership accepted validation changed")
			}
			return &validatedRetranslationCandidate{candidatePath: choice.result.CandidatePath, validationPath: choice.result.ValidationPath, candidate: candidate, validationData: vb, attempt: old.Attempt}, nil
		}
	}
	evidenceCatalog, err := snapshotRevisionEvidenceCatalog(root, current, locale, choice, c)
	if err != nil {
		return nil, err
	}
	v, err := validateSelectedRetranslationCandidateWithOutput(root, evidenceCatalog, locale, g, u, choice, output, compatibility)
	if err != nil {
		return nil, err
	}
	if err := ValidateTranslationUnitCandidate(root, current, u.ID, locale, v.candidate); err != nil {
		return nil, err
	}
	return v, nil
}
