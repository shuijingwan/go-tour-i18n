package i18n

import (
	"fmt"
	"path"
	"sort"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

const (
	TransportRecoverySchema             = "go-learning/installed-derived-transport-recovery/v1"
	TransportRecoveryCapability         = "renew-approved-derived-transport-invocation"
	DerivedTransportConstructionSchema  = "go-learning/derived-transport-construction/v1"
	TransportRecoveryExecutionSchema    = "go-learning/transport-recovery-generation-execution/v1"
	TransportRecoveryTaskKind           = "translation-unit-transport-stale-recovery"
	transportRecoveryReason             = "installed_derived_transport_stale"
	transportRecoveryConstructionMember = "formal/transport-recovery/construction.json"
)

// Recovery references are physical byte references. Construction identities are
// separate: re-encoding a JSON artifact cannot grant a recovery capability.
type TransportRecoveryReference struct {
	Path     string `json:"path"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
	Identity string `json:"identity"`
}

type RecoveryFileReference struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type RecoveryDependency struct {
	Role string                `json:"role"`
	Ref  RecoveryFileReference `json:"ref"`
}

type RecoveryPackage struct {
	ID                  string               `json:"id"`
	AuthoritySHA256     string               `json:"authority_sha256"`
	ScopeIdentitySHA256 string               `json:"scope_identity_sha256"`
	Units               []RecoverySourceUnit `json:"units"`
	References          []RecoveryDependency `json:"references"`
}

type RecoverySourceUnit struct {
	UnitID                 string                `json:"unit_id"`
	Kind                   UnitKind              `json:"kind"`
	Source                 RecoveryFileReference `json:"source"`
	InputSHA256            string                `json:"input_sha256"`
	RepresentationIdentity string                `json:"representation_identity"`
}

type RecoveryInvocation struct {
	BatchID             string                `json:"batch_id"`
	Manifest            RecoveryFileReference `json:"manifest"`
	Result              RecoveryFileReference `json:"result"`
	TaskKind            string                `json:"task_kind"`
	ImportReceipt       RecoveryFileReference `json:"import_receipt"`
	Archive             RecoveryFileReference `json:"archive"`
	InputIdentitySHA256 string                `json:"input_identity_sha256"`
	Provider            string                `json:"provider"`
	Model               string                `json:"model"`
	FinalUnits          []RecoveryFinalUnit   `json:"final_units"`
}

type RecoveryFinalUnit struct {
	UnitID     string                `json:"unit_id"`
	Attempt    int                   `json:"attempt"`
	Raw        RecoveryFileReference `json:"raw"`
	Candidate  RecoveryFileReference `json:"candidate"`
	Validation RecoveryFileReference `json:"validation"`
	History    []RecoveryDependency  `json:"history"`
}

type RecoveryLanguageAuthority struct {
	Reopen         RecoveryFileReference    `json:"reopen"`
	ReopenIdentity string                   `json:"reopen_identity"`
	Snapshot       RecoveryFileReference    `json:"snapshot"`
	Scope          []RetranslationBatchUnit `json:"scope"`
	References     []RecoveryDependency     `json:"references"`
}

type RecoveryDependencyDelta struct {
	Role         string                `json:"role"`
	Old          RecoveryFileReference `json:"old"`
	Current      RecoveryFileReference `json:"current"`
	Declarations []string              `json:"declarations"`
}

type RecoveryDirectDrift struct {
	FrozenConstruction          *RecoveryFileReference    `json:"frozen_construction,omitempty"`
	CurrentConstruction         *RecoveryFileReference    `json:"current_construction,omitempty"`
	UnitID                      string                    `json:"unit_id"`
	Classification              string                    `json:"classification"`
	OldProof                    RecoveryFileReference     `json:"old_proof"`
	OldProofIdentity            string                    `json:"old_proof_identity"`
	CurrentProof                RecoveryFileReference     `json:"current_proof"`
	CurrentProofIdentity        string                    `json:"current_proof_identity"`
	OldConstructionIdentity     string                    `json:"derived_old_construction_identity"`
	CurrentConstructionIdentity string                    `json:"current_construction_identity"`
	Dependencies                []RecoveryDependencyDelta `json:"dependency_deltas"`
}

type RecoveryClosure struct {
	ContextRoot      string                `json:"context_root"`
	UnitIDs          []string              `json:"unit_ids"`
	NonDirectUnitIDs []string              `json:"non_direct_unit_ids"`
	Edges            []RecoveryContextEdge `json:"edges"`
}

type RecoveryContextEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Role string `json:"role"`
}

type RecoveryAssessment struct {
	OriginalBaseline           *CurrentTranslationAuthorization `json:"original_baseline_assessment"`
	HistoricalIntegrity        bool                             `json:"historical_integrity"`
	LanguageAuthorityIdentity  string                           `json:"language_authority_identity"`
	CurrentCandidateValidation []RecoveryValidationAssessment   `json:"current_candidate_validation"`
	References                 []RecoveryDependency             `json:"references"`
}

type RecoveryValidationAssessment struct {
	UnitID            string `json:"unit_id"`
	CandidateSHA256   string `json:"candidate_sha256"`
	ValidatorIdentity string `json:"validator_identity"`
	Status            string `json:"status"`
}

type RecoveryAllowedAction struct {
	Operation                     string `json:"operation"`
	Provider                      string `json:"provider"`
	Model                         string `json:"model"`
	NewLanguageGenerationRequired bool   `json:"new_language_generation_required"`
	OldRawReplayAllowed           bool   `json:"old_raw_replay_allowed"`
}

type InstalledDerivedTransportRecovery struct {
	Schema            string                      `json:"schema"`
	Capability        string                      `json:"capability"`
	Locale            string                      `json:"locale"`
	ContentPackage    RecoveryPackage             `json:"content_package"`
	Predecessor       RecoveryInvocation          `json:"predecessor_batch_invocation"`
	LanguageAuthority RecoveryLanguageAuthority   `json:"original_language_authority"`
	DirectDrift       []RecoveryDirectDrift       `json:"direct_drift_units"`
	RecoveryUnits     RecoveryClosure             `json:"recovery_units"`
	CurrentAssessment RecoveryAssessment          `json:"current_assessment"`
	CurrentEpoch      []RecoveryDependency        `json:"current_epoch"`
	Construction      RecoveryFileReference       `json:"construction"`
	AllowedAction     RecoveryAllowedAction       `json:"allowed_action"`
	Supersedes        *TransportRecoveryReference `json:"supersedes,omitempty"`
	Members           []RecoveryFileReference     `json:"members"`
	Identity          string                      `json:"identity"`
}

func transportRecoveryIdentity(a InstalledDerivedTransportRecovery) string {
	a.Identity = ""
	return sum(mustJSON(a))
}

func transportRecoveryDirectory(locale, identity string) string {
	return "data/translation-transport-recoveries/" + locale + "/" + identity
}

func validRecoveryFileReference(r RecoveryFileReference) bool {
	return validateGenerationInstallPath(r.Path) == nil && validSHA256(r.SHA256) && r.Size >= 0
}

func recoveryMemberReference(name string, b []byte) RecoveryFileReference {
	return RecoveryFileReference{Path: name, SHA256: sum(b), Size: int64(len(b))}
}

func validateRecoveryAuthority(a *InstalledDerivedTransportRecovery) error {
	if a.Schema != TransportRecoverySchema || a.Capability != TransportRecoveryCapability || ValidateLocaleName(a.Locale) != nil || a.Identity != transportRecoveryIdentity(*a) {
		return fmt.Errorf("recovery authority schema/capability/identity mismatch")
	}
	if err := validateRecoveryAuthorityInventory(a); err != nil {
		return err
	}
	if a.ContentPackage.ID == "" || a.ContentPackage.ID == tourNativeTranslationPackage || !validSHA256(a.ContentPackage.AuthoritySHA256) || !validSHA256(a.ContentPackage.ScopeIdentitySHA256) || validateBatchID(a.Predecessor.BatchID) != nil {
		return fmt.Errorf("recovery authority requires complete registered package and predecessor")
	}
	if a.AllowedAction.Operation != "retranslation-export-with-transport-recovery" || !a.AllowedAction.NewLanguageGenerationRequired || a.AllowedAction.OldRawReplayAllowed || a.AllowedAction.Provider != a.Predecessor.Provider || a.AllowedAction.Model != a.Predecessor.Model {
		return fmt.Errorf("recovery authority successor action mismatch")
	}
	if err := requireFormalGenerationIdentity(a.Predecessor.Provider, a.Predecessor.Model, a.Predecessor.BatchID); err != nil {
		return err
	}
	if len(a.DirectDrift) == 0 || len(a.RecoveryUnits.UnitIDs) == 0 || len(a.RecoveryUnits.UnitIDs) > MaxRetranslationExportLimit || len(a.RecoveryUnits.UnitIDs) != len(a.LanguageAuthority.Scope) || len(a.Predecessor.FinalUnits) != len(a.RecoveryUnits.UnitIDs) {
		return fmt.Errorf("recovery authority incomplete D/R scope")
	}
	seen := map[string]bool{}
	kind := a.LanguageAuthority.Scope[0].UnitKind
	if kind != UnitKindPage {
		return fmt.Errorf("approved recovery requires a homogeneous Page batch")
	}
	for i, id := range a.RecoveryUnits.UnitIDs {
		r := a.LanguageAuthority.Scope[i]
		if seen[id] || id != r.UnitID || id != a.Predecessor.FinalUnits[i].UnitID || r.UnitKind != kind || r.RevisionFeedbackSource != CurrentValidationRevisionFeedback || r.RepresentationBridge != nil {
			return fmt.Errorf("recovery authority nonexclusive ordered revision scope")
		}
		seen[id] = true
	}
	last := -1
	for _, d := range a.DirectDrift {
		i := -1
		for j, id := range a.RecoveryUnits.UnitIDs {
			if id == d.UnitID {
				i = j
			}
		}
		if i <= last || (d.Classification != "construction-dependency-drift" && d.Classification != "construction-contract-drift") || len(d.Dependencies) == 0 || d.OldConstructionIdentity == d.CurrentConstructionIdentity {
			return fmt.Errorf("recovery authority invalid direct drift")
		}
		last = i
	}
	for _, r := range a.Members {
		if !validRecoveryFileReference(r) || r.Path == "authority.json" || path.Clean(r.Path) != r.Path {
			return fmt.Errorf("recovery authority invalid member")
		}
	}
	return nil
}

func decodeRecoveryAuthority(b []byte) (*InstalledDerivedTransportRecovery, error) {
	var a InstalledDerivedTransportRecovery
	if err := contentidentity.StrictJSON(b, &a); err != nil {
		return nil, err
	}
	if err := validateRecoveryAuthority(&a); err != nil {
		return nil, err
	}
	return &a, nil
}

func sortedRecoveryDependencies(in []RecoveryDependency) ([]RecoveryDependency, error) {
	out := append([]RecoveryDependency(nil), in...)
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Role != b.Role {
			return a.Role < b.Role
		}
		if a.Ref.Path != b.Ref.Path {
			return a.Ref.Path < b.Ref.Path
		}
		return a.Ref.SHA256 < b.Ref.SHA256
	})
	for i, d := range out {
		if d.Role == "" || !validRecoveryFileReference(d.Ref) {
			return nil, fmt.Errorf("invalid recovery dependency")
		}
		if i > 0 && out[i-1].Role == d.Role && out[i-1].Ref.Path == d.Ref.Path {
			return nil, fmt.Errorf("duplicate recovery dependency")
		}
	}
	return out, nil
}
