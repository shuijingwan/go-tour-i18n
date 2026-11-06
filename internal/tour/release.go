package tour

import "fmt"

// CheckReleaseCanonicalOrigin binds the frozen Tour renderer to the sealed
// production target. A registry disagreement is a machine failure; it cannot
// authorize a fallback target or change an immutable language Surface context.
func CheckReleaseCanonicalOrigin(locale, origin string) error {
	current, err := productionOriginForLocale(locale)
	if err != nil {
		return err
	}
	if current != origin {
		return fmt.Errorf("sealed production canonical identity mismatch")
	}
	return nil
}
