package i18n

// FindCurrentScopeTranslationAuthorization reuses the checked scope checkpoint
// for planners. It selects no candidate and never creates authorization.
func FindCurrentScopeTranslationAuthorization(root string, c *Catalog, locale string) (*CurrentTranslationAuthorization, error) {
	return findScopeCurrentAuthorization(root, c, locale)
}

// ReadHistoricalTranslationIntegrity consumes an existing flattened immutable
// checkpoint when present, without replaying its current eligibility epoch.
func ReadHistoricalTranslationIntegrity(root string, c *Catalog, locale, snapshot string) (*HistoricalFinalizationIntegrity, error) {
	if proof, err := loadSealedTranslationHistory(root, c, locale, snapshot); err != nil {
		return nil, err
	} else if proof != nil {
		return proof, nil
	}
	return VerifyHistoricalQualityCheckFinalization(root, locale, snapshot, c)
}
