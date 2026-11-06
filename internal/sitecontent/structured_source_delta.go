package sitecontent

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

type StructuredSourceDelta struct {
	Locale            string    `json:"locale"`
	Authority         Reference `json:"source_authority"`
	Closure           Reference `json:"accepted_closure"`
	Generation        Reference `json:"accepted_generation"`
	AcceptedTargetSHA string    `json:"accepted_target_sha256"`
	Carry             []Target  `json:"carried_slots"`
	Pending           []string  `json:"pending_generation_slot_ids"`
	Identity          string    `json:"identity_sha256"`
}

func sourceDeltaFromBaseline(root, locale string, a StructuredSourceSuccessor, b *structuredBaseline) (*StructuredSourceDelta, error) {
	owner, ok := b.Owners[a.Previous.Path]
	if !ok || b.Closure.Locale != locale || b.Closure.Package != a.Package {
		return nil, fmt.Errorf("source successor baseline locale/package/asset mismatch")
	}
	g := b.Generations[owner.Path]
	if g == nil {
		return nil, fmt.Errorf("source successor accepted generation missing")
	}
	var asset *StructuredAsset
	var target *StructuredTarget
	for i := range g.Plan.Assets {
		if g.Plan.Assets[i].Document != nil && g.Plan.Assets[i].Document.Path == a.Previous.Path {
			asset = &g.Plan.Assets[i]
		}
	}
	for i := range g.Output.Outputs {
		if g.Output.Outputs[i].Path == "locales/"+locale+"/structured/"+a.Previous.Path {
			target = &g.Output.Outputs[i]
		}
	}
	if asset == nil || target == nil || asset.Package != a.Package {
		return nil, fmt.Errorf("source successor accepted representation does not match exact predecessor slots")
	}
	values, err := structuredTargetValues(*asset, *target, &i18n.Glossary{Mandatory: map[string]string{}})
	if err != nil {
		return nil, err
	}
	oldRaw := b.Inputs[owner.Path]["source/"+a.Previous.Path]
	if !sourceBaselineInAncestry(root, a, *asset.Document, oldRaw, 0) {
		return nil, fmt.Errorf("source successor historical source bytes mismatch")
	}
	restored, err := reconstructFrozenStructuredRepresentation(*asset.Document, oldRaw, values)
	if err != nil || !bytes.Equal(restored, b.Files[a.Previous.Path]) {
		return nil, fmt.Errorf("source successor accepted target restore mismatch")
	}
	raw, _ := Encode(a)
	d := &StructuredSourceDelta{Locale: locale, Authority: Reference{StructuredSourceSuccessorPath(&a), digest(raw)}, Closure: b.ClosureRef, Generation: owner, AcceptedTargetSHA: digest(restored), Carry: []Target{}, Pending: []string{}}
	carried := map[string]bool{}
	for _, id := range a.CarryIDs {
		text, ok := values[id]
		if !ok {
			if a.Replacement != nil {
				continue
			} // source-only slots may never have an accepted target
			return nil, fmt.Errorf("missing accepted target slot")
		}
		carried[id] = true
		d.Carry = append(d.Carry, Target{ID: id, Text: text})
	}
	for _, u := range a.Current.Units {
		if !carried[u.ID] {
			d.Pending = append(d.Pending, u.ID)
		}
	}
	d.Identity = identity(*d)
	return d, nil
}

// Source-only scope inspection does not require or create a new language
// review. Actual Generation still requires the current full corpus Review.
func PlanStructuredSourceSuccessor(root, locale string, ref Reference) (*StructuredSourceDelta, error) {
	if err := i18n.ValidateLocaleName(locale); err != nil {
		return nil, err
	}
	a, err := CheckStructuredSourceSuccessor(root, ref)
	if err != nil {
		return nil, err
	}
	return planStructuredSourceDelta(root, locale, *a)
}

func InspectStructuredSourceSuccessor(root, locale string, proposal Reference) (*StructuredSourceDelta, error) {
	if err := i18n.ValidateLocaleName(locale); err != nil {
		return nil, err
	}
	a, err := BuildStructuredSourceSuccessor(root, proposal)
	if err != nil {
		return nil, err
	}
	return planStructuredSourceDelta(root, locale, *a)
}

func planStructuredSourceDelta(root, locale string, a StructuredSourceSuccessor) (*StructuredSourceDelta, error) {
	g, err := LoadCurrent(root)
	if err != nil {
		return nil, err
	}
	l, err := readLocaleAuthority(root, g, locale)
	if err != nil {
		return nil, err
	}
	v := newClosureValidation(root)
	for _, c := range l.Packages {
		if c.Package == a.Package {
			b, err := v.historicalStructuredCompletion(root, locale, c)
			if err != nil {
				return nil, err
			}
			d, err := sourceDeltaFromBaseline(root, locale, a, b)
			if err != nil {
				return nil, err
			}
			if err := v.verify(); err != nil {
				return nil, err
			}
			return d, nil
		}
	}
	return nil, fmt.Errorf("source successor requires an accepted structured baseline")
}

func (v *closureValidation) sourceExpansionAssessment(root string, p StructuredPlan, a StructuredAsset, b *structuredBaseline, x *StructuredAssessment) (bool, error) {
	g, err := v.loadCurrent(root)
	if err != nil {
		return false, err
	}
	for _, ref := range g.StructuredSuccessors {
		s, err := CheckStructuredSourceSuccessor(root, ref)
		if err != nil {
			return false, err
		}
		if a.Document == nil || s.Package != a.Package || s.Current.Path != a.Document.Path {
			continue
		}
		if !reflect.DeepEqual(s.Current, *a.Document) {
			return false, fmt.Errorf("source successor current representation drift")
		}
		d, err := sourceDeltaFromBaseline(root, p.Locale, *s, b)
		if err != nil {
			return false, err
		}
		glossary, err := i18n.LoadGlossary(root, p.Locale)
		if err != nil {
			return false, err
		}
		values := map[string]string{}
		for _, slot := range d.Carry {
			values[slot.ID] = slot.Text
		}
		for _, u := range s.Current.Units {
			if text, ok := values[u.ID]; ok {
				if err := i18n.ValidateParsedGlossary(u.Source, text, glossary); err != nil {
					return false, fmt.Errorf("accepted source slot needs separate glossary revision: %w", err)
				}
			}
		}
		x.Classification, x.Reason, x.SourceSuccessor = "affected", "source-owned content expansion", &ref
		if s.Replacement != nil {
			x.Reason = "source-owned content replacement"
		}
		x.Identity = ""
		x.Identity = identity(*x)
		return true, nil
	}
	return false, nil
}

func sourceDeltaCarry(root string, p StructuredPlan, a StructuredAsset, b *structuredBaseline) ([]Target, error) {
	if a.Assessment == nil || a.Assessment.SourceSuccessor == nil {
		return nil, nil
	}
	s, err := CheckStructuredSourceSuccessor(root, *a.Assessment.SourceSuccessor)
	if err != nil {
		return nil, err
	}
	d, err := sourceDeltaFromBaseline(root, p.Locale, *s, b)
	if err != nil {
		return nil, err
	}
	return d.Carry, nil
}

func sourceBaselineInAncestry(root string, a StructuredSourceSuccessor, d Document, raw []byte, depth int) bool {
	if depth > 32 || digest(raw) != d.SourceSHA {
		return false
	}
	if reflect.DeepEqual(d, a.Old) {
		return true
	}
	if a.Replacement == nil {
		return false
	}
	parent, err := CheckStructuredSourceSuccessor(root, a.Replacement.Predecessor)
	return err == nil && sourceBaselineInAncestry(root, *parent, d, raw, depth+1)
}
