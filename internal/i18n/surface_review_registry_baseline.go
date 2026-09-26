package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
)

const (
	localeSurfaceReviewRegistryBaselineSchemaVersion = 1
	localeSurfaceReviewRegistryBaselineKind          = "go-tour-i18n/locale-surface-review-registry-baseline"
)

// LocaleSurfaceReviewRegistryBaseline is independent compatibility evidence for
// an unchanged schema v2 A gate. It does not rewrite or reinterpret that gate's
// original language-quality decision.
type LocaleSurfaceReviewRegistryBaseline struct {
	SchemaVersion                   int                            `json:"schema_version"`
	Kind                            string                         `json:"kind"`
	Locale                          string                         `json:"locale"`
	ReviewID                        string                         `json:"review_id"`
	GatePath                        string                         `json:"gate_path"`
	GateSHA256                      string                         `json:"gate_sha256"`
	LanguagesConfigSHA256           string                         `json:"languages_config_sha256"`
	LanguagesReviewProjectionSHA256 string                         `json:"languages_review_projection_sha256"`
	LanguageRegistryBaseline        LanguageRegistryReviewBaseline `json:"language_registry_baseline"`
	ProductionIdentitySHA256        string                         `json:"production_identity_sha256"`
}

func LocaleSurfaceReviewRegistryBaselinePath(root, locale, reviewID string) (string, error) {
	if _, err := LocaleSurfaceReviewAGatePath(root, locale, reviewID); err != nil {
		return "", err
	}
	return filepath.Join(root, "data", "locale-surface-reviews", locale, reviewID+".registry-baseline.json"), nil
}

// RecordLocaleSurfaceReviewRegistryBaseline records new, immutable
// compatibility evidence for a currently exact schema v2 gate. Schema v1
// remains exact-only, while schema v3 carries this projection in the gate.
func RecordLocaleSurfaceReviewRegistryBaseline(root, locale, reviewID string, catalog *Catalog) (*LocaleSurfaceReviewRegistryBaseline, string, error) {
	gatePath, err := LocaleSurfaceReviewAGatePath(root, locale, reviewID)
	if err != nil {
		return nil, "", err
	}
	gateData, err := os.ReadFile(gatePath)
	if os.IsNotExist(err) {
		return nil, "", fmt.Errorf("Locale Surface Review A gate missing for %s review_id=%s", locale, reviewID)
	}
	if err != nil {
		return nil, "", fmt.Errorf("read Locale Surface Review A gate %s: %w", reviewID, err)
	}
	var gate LocaleSurfaceReviewAGate
	if err := decodeSingleJSON(gateData, &gate); err != nil {
		return nil, "", fmt.Errorf("language review evidence/gate stale: malformed Locale Surface Review A gate %s", filepath.Base(gatePath))
	}
	if err := validateLocaleSurfaceReviewAGate(gate, locale); err != nil {
		return nil, "", err
	}
	if gate.ReviewID != reviewID {
		return nil, "", fmt.Errorf("language review evidence/gate stale: review identity mismatch for %s", reviewID)
	}
	if gate.SchemaVersion != localeSurfaceReviewASchemaVersionV2 {
		return nil, "", fmt.Errorf("registry baseline requires a current schema v2 gate; review_id=%s has schema %d", reviewID, gate.SchemaVersion)
	}
	current, err := currentLocaleSurfaceReviewAInputs(root, locale, catalog, localeSurfaceReviewASchemaVersionV2)
	if err != nil {
		return nil, "", err
	}
	if !reflect.DeepEqual(gate.Inputs, current) {
		return nil, "", fmt.Errorf("registry baseline requires the unchanged exact schema v2 gate for %s review_id=%s", locale, reviewID)
	}
	if err := validateCurrentLanguageRegistryCompatibility(root); err != nil {
		return nil, "", fmt.Errorf("cannot record registry baseline: %w", err)
	}
	projection, err := currentLanguageReviewProjectionSHA256(root, locale)
	if err != nil {
		return nil, "", fmt.Errorf("cannot record registry baseline: %w", err)
	}
	registryBaseline, err := currentLanguageRegistryReviewBaseline(root)
	if err != nil {
		return nil, "", fmt.Errorf("cannot record registry baseline: %w", err)
	}
	if !validLanguageRegistryReviewBaseline(registryBaseline) {
		return nil, "", fmt.Errorf("cannot record registry baseline: incomplete registry review baseline")
	}
	identityData, err := os.ReadFile(filepath.Join(root, "production", "identity.json"))
	if err != nil {
		return nil, "", fmt.Errorf("read registry baseline production identity: %w", err)
	}
	baselinePath, err := LocaleSurfaceReviewRegistryBaselinePath(root, locale, reviewID)
	if err != nil {
		return nil, "", err
	}
	relativeGatePath := filepath.ToSlash(filepath.Join("data", "locale-surface-reviews", locale, reviewID+".a-gate.json"))
	baseline := &LocaleSurfaceReviewRegistryBaseline{
		SchemaVersion:                   localeSurfaceReviewRegistryBaselineSchemaVersion,
		Kind:                            localeSurfaceReviewRegistryBaselineKind,
		Locale:                          locale,
		ReviewID:                        reviewID,
		GatePath:                        relativeGatePath,
		GateSHA256:                      hashBytes(gateData),
		LanguagesConfigSHA256:           gate.Inputs.LanguagesConfigSHA256,
		LanguagesReviewProjectionSHA256: projection,
		LanguageRegistryBaseline:        registryBaseline,
		ProductionIdentitySHA256:        hashBytes(identityData),
	}
	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return nil, "", err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(baselinePath), 0755); err != nil {
		return nil, "", err
	}
	file, err := os.OpenFile(baselinePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if os.IsExist(err) {
		return nil, "", fmt.Errorf("Locale Surface Review registry baseline already exists: %s", filepath.ToSlash(baselinePath))
	}
	if err != nil {
		return nil, "", err
	}
	if _, err = file.Write(data); err == nil {
		err = file.Close()
	} else {
		_ = file.Close()
	}
	if err != nil {
		return nil, "", err
	}
	return baseline, baselinePath, nil
}

func currentLocaleSurfaceReviewRegistryBaseline(root, locale string, gate LocaleSurfaceReviewAGate, gateData []byte, current LocaleSurfaceReviewAInputs) (bool, error) {
	path, err := LocaleSurfaceReviewRegistryBaselinePath(root, locale, gate.ReviewID)
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read Locale Surface Review registry baseline: %w", err)
	}
	var baseline LocaleSurfaceReviewRegistryBaseline
	if err := decodeSingleJSON(data, &baseline); err != nil {
		return false, fmt.Errorf("language review evidence/gate stale: malformed registry baseline %s: %w", filepath.Base(path), err)
	}
	expectedGatePath := filepath.ToSlash(filepath.Join("data", "locale-surface-reviews", locale, gate.ReviewID+".a-gate.json"))
	if baseline.SchemaVersion != localeSurfaceReviewRegistryBaselineSchemaVersion ||
		baseline.Kind != localeSurfaceReviewRegistryBaselineKind ||
		baseline.Locale != locale ||
		baseline.ReviewID != gate.ReviewID ||
		baseline.GatePath != expectedGatePath ||
		baseline.GateSHA256 != hashBytes(gateData) ||
		baseline.LanguagesConfigSHA256 == "" ||
		baseline.LanguagesConfigSHA256 != gate.Inputs.LanguagesConfigSHA256 ||
		baseline.LanguagesReviewProjectionSHA256 == "" ||
		!validLanguageRegistryReviewBaseline(baseline.LanguageRegistryBaseline) ||
		baseline.ProductionIdentitySHA256 == "" {
		return false, fmt.Errorf("language review evidence/gate stale: invalid registry baseline %s", filepath.Base(path))
	}
	if err := validateCurrentLanguageRegistryCompatibility(root); err != nil {
		return false, fmt.Errorf("language review evidence/gate stale: language registry compatibility: %w", err)
	}
	projection, err := currentLanguageReviewProjectionSHA256(root, locale)
	if err != nil {
		return false, fmt.Errorf("language review evidence/gate stale: language review projection: %w", err)
	}
	if projection != baseline.LanguagesReviewProjectionSHA256 {
		return false, nil
	}
	currentRegistryBaseline, err := currentLanguageRegistryReviewBaseline(root)
	if err != nil {
		return false, fmt.Errorf("language review evidence/gate stale: language registry baseline: %w", err)
	}
	if !languageRegistryReviewBaselineCompatible(baseline.LanguageRegistryBaseline, currentRegistryBaseline) {
		return false, nil
	}
	recorded := gate.Inputs
	recorded.LanguagesConfigSHA256 = current.LanguagesConfigSHA256
	return reflect.DeepEqual(recorded, current), nil
}

func decodeSingleJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	return decodeSingleJSONFromDecoder(decoder, target)
}

func decodeSingleJSONValue(data []byte, target any) error {
	return decodeSingleJSONFromDecoder(json.NewDecoder(bytes.NewReader(data)), target)
}

func decodeSingleJSONFromDecoder(decoder *json.Decoder, target any) error {
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}
