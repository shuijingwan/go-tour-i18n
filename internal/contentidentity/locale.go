// Package contentidentity defines the unchanged immutable locale completion
// wire contract shared by content scope and downstream freshness proofs.
package contentidentity

const LocaleSchema = "go-learning/locale-content-scope/v1"

type Reference struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type Completion struct {
	Package            string      `json:"package"`
	State              string      `json:"state"`
	PackageIdentity    string      `json:"package_identity_sha256"`
	SourceIdentity     string      `json:"source_authority_sha256"`
	Surfaces           []string    `json:"completed_surfaces"`
	Routes             []string    `json:"local_canonical_routes"`
	RouteFamilies      []string    `json:"local_route_families"`
	EvidenceKind       string      `json:"evidence_kind"`
	Evidence           []Reference `json:"evidence"`
	EvidenceIdentity   string      `json:"evidence_identity_sha256"`
	ContextIdentity    string      `json:"validated_context_sha256"`
	LegacySurfaceState string      `json:"legacy_surface_gate_state"`
}
type Locale struct {
	Schema   string       `json:"schema"`
	Locale   string       `json:"locale"`
	Packages []Completion `json:"packages"`
}
