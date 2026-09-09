package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/tourpolicy"
)

// publicationPolicyCommand is the machine-readable bridge for non-Go
// verifiers. The policy package remains the only locale registry.
func publicationPolicyCommand(args []string) error {
	fs := flag.NewFlagSet("policy publication", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *locale == "" || len(fs.Args()) != 0 {
		return fmt.Errorf("usage: tour-i18n policy publication --locale <locale>")
	}
	policy := tourpolicy.ForLocale(*locale)
	result, err := json.Marshal(struct {
		Locale         string                 `json:"locale"`
		Publication    tourpolicy.Publication `json:"publication"`
		TourAdsEnabled bool                   `json:"tour_ads_enabled"`
	}{*locale, policy, policy.TourAdsEnabled()})
	if err != nil {
		return fmt.Errorf("encode publication policy: %w", err)
	}
	fmt.Println(string(result))
	return nil
}
