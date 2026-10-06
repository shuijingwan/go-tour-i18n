package ui

import "fmt"

// ValidateLocalizedBytes applies the existing catalog and placeholder/markup
// contract without installing a target file or falling back to English.
func ValidateLocalizedBytes(locale string, english, target []byte) error {
	source, err := parseCatalog(english)
	if err != nil {
		return err
	}
	candidate, err := parseCatalog(target)
	if err != nil {
		return err
	}
	if source.Locale != "en" || candidate.Locale != locale {
		return fmt.Errorf("catalog locale mismatch")
	}
	return validateCoverage(source, candidate)
}
