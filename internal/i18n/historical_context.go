package i18n

import "fmt"

// HistoricalContextReader reconstructs exact frozen context bytes through a
// representation-owned lifecycle proof. Consumers must supply the reference
// from immutable evidence; it is never used to create new/current authority.
type HistoricalContextReader func(root, locale string, ref GlossaryArchiveReference) ([]byte, error)

func readHistoricalContextReference(root string, c *Catalog, locale string, ref GlossaryArchiveReference) ([]byte, error) {
	data, err := readCompatibilityFile(root, ref.Path)
	if err != nil {
		return nil, err
	}
	if sum(data) == ref.SHA256 {
		return data, nil
	}
	if c != nil && c.HistoricalContextReader != nil {
		frozen, err := c.HistoricalContextReader(root, locale, ref)
		if err == nil && sum(frozen) == ref.SHA256 {
			return frozen, nil
		}
	}
	return nil, fmt.Errorf("historical context reference changed: %s", ref.Path)
}
