package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Older shared historical readers carry a Unit record, not its containing
// batch ID. Recover that locator through the exact predecessor's root link and
// immutable child binding. Never scan latest receipts or infer a package from
// Unit membership; supersessions follow their unique explicit next links.
func recoveryResponseOutput(root string, c *Catalog, g *Glossary, locale string, selected []RetranslationBatchUnit, r RetranslationBatchUnit) (protectedTranslation, error) {
	if ValidateLocaleName(locale) != nil || validateBatchID(r.PreviousBatchID) != nil {
		return protectedTranslation{}, fmt.Errorf("recovery response predecessor locator missing")
	}
	ip := "data/retranslation-runs/" + locale + "/" + r.PreviousBatchID + "/generation-imports/initial-manifest.json"
	b, err := readCompatibilityFile(root, ip)
	if err != nil {
		return protectedTranslation{}, err
	}
	var receipt GenerationResultBundleManifest
	if err = contentidentity.StrictJSON(b, &receipt); err != nil {
		return protectedTranslation{}, err
	}
	if receipt.Locale != locale || receipt.BatchID != r.PreviousBatchID || receipt.Attempt != 1 || !validSHA256(receipt.GenerationBundleSHA256) {
		return protectedTranslation{}, fmt.Errorf("recovery response predecessor receipt mismatch")
	}
	linkPath := recoveryRootDirectory(locale) + "/roots/" + receipt.GenerationBundleSHA256 + ".json"
	seen := map[string]bool{}
	var authority *InstalledDerivedTransportRecovery
	for {
		link, err := readRecoveryLink(root, linkPath)
		if err != nil {
			return protectedTranslation{}, err
		}
		if link.Locale != locale || link.RootInvocationSHA256 != receipt.GenerationBundleSHA256 || seen[link.Next.Identity] {
			return protectedTranslation{}, fmt.Errorf("recovery response chain mismatch/cycle")
		}
		seen[link.Next.Identity] = true
		authority, _, err = readRecoveryExactAuthority(root, link.Next)
		if err != nil {
			return protectedTranslation{}, err
		}
		if authority.Predecessor.BatchID != r.PreviousBatchID {
			return protectedTranslation{}, fmt.Errorf("recovery response root predecessor mismatch")
		}
		next := recoveryRootDirectory(locale) + "/supersessions/" + authority.Identity + ".json"
		if _, err = readCompatibilityFile(root, next); os.IsNotExist(err) {
			break
		} else if err != nil {
			return protectedTranslation{}, err
		}
		linkPath = next
	}
	child, err := findTransportRecoveryChild(root, authority)
	if err != nil {
		return protectedTranslation{}, err
	}
	if child == nil || !reflect.DeepEqual(child.Units, selected) {
		return protectedTranslation{}, fmt.Errorf("recovery response exact child/complete scope mismatch")
	}
	c, err = recoveryPackageCatalog(c, authority.ContentPackage.ID)
	if err != nil {
		return protectedTranslation{}, err
	}
	s, _, err := importedApprovedIntervalSelection(root, c, child)
	if err != nil || s == nil {
		return protectedTranslation{}, fmt.Errorf("recovery response requires frozen approved selection: %v", err)
	}
	return prepareInvocationRetranslationOutput(root, c, g, locale, selected, r, child.BatchID)
}

func recoveryChildAllPassed(root string, m *RetranslationBatchManifest) bool {
	dir := filepath.Join(root, "data/retranslation-runs", m.Locale, m.BatchID)
	result, err := readPromotionResult(dir, m.Locale, m.BatchID, m.UnitCount)
	if err != nil {
		return false
	}
	byID := map[string]RetranslationUnitResult{}
	for _, r := range result.Units {
		byID[r.UnitID] = r
	}
	for _, r := range m.Units {
		x, ok := byID[r.UnitID]
		if !ok || x.Status != "passed" {
			return false
		}
		v, err := readPromotionValidation(dir, m.BatchID, m.Locale, r, x)
		if err != nil || v.Status != "passed" {
			return false
		}
	}
	return true
}
