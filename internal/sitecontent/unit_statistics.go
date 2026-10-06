package sitecontent

import "sort"

type UnitStatistics struct {
	Contract                string `json:"parser_contract"`
	Diagnostic              bool   `json:"internal_slot_diagnostics"`
	FormalPages             int    `json:"formal_pages"`
	FormalPageBatches       int    `json:"formal_page_generation_and_qc_batches"`
	Documents               int    `json:"documents"`
	Units                   int    `json:"units"`
	Min                     int    `json:"min_source_bytes"`
	Median                  int    `json:"median_source_bytes"`
	P90                     int    `json:"p90_source_bytes"`
	Max                     int    `json:"max_source_bytes"`
	Small                   int    `json:"units_le_20_bytes"`
	Punctuation             int    `json:"punctuation_only_units"`
	LargestDocument         string `json:"largest_document"`
	LargestDocumentUnits    int    `json:"largest_document_units"`
	MinimumReviews          int    `json:"diagnostic_minimum_slot_sets"`
	GenerationBatches       int    `json:"diagnostic_generation_slot_sets"`
	SourceEquivalentReviews int    `json:"diagnostic_source_equivalent_slot_sets"`
}

// Slot counts are internal diagnostics. Formal Reviewer planning uses stable
// Page membership; actual target bytes only enforce transport hard limits.
func SummarizeUnits(docs []Document, raw map[string][]byte) (*UnitStatistics, error) {
	s := &UnitStatistics{Contract: UnitContract, Documents: len(docs), Diagnostic: true}
	for _, d := range docs {
		if d.Kind == "page" {
			s.FormalPages++
		}
	}
	s.FormalPageBatches = (s.FormalPages + 29) / 30
	sizes := []int{}
	targets := map[string]string{}
	for _, d := range docs {
		if len(d.Units) > s.LargestDocumentUnits {
			s.LargestDocument = d.Path
			s.LargestDocumentUnits = len(d.Units)
		}
		for _, u := range d.Units {
			n := len(u.Source)
			sizes = append(sizes, n)
			if n <= 20 {
				s.Small++
			}
			if !hasLanguage(u.Source) {
				s.Punctuation++
			}
			targets[u.ID] = u.Source
		}
	}
	s.Units = len(sizes)
	sort.Ints(sizes)
	if len(sizes) > 0 {
		s.Min = sizes[0]
		s.Median = sizes[(len(sizes)-1)/2]
		s.P90 = sizes[(len(sizes)*9+9)/10-1]
		s.Max = sizes[len(sizes)-1]
	}
	s.MinimumReviews = (s.Units + MaxReviewUnits - 1) / MaxReviewUnits
	gen, err := DiagnosticSlotPlan(docs, raw, selectAll(docs), nil)
	if err != nil {
		return nil, err
	}
	s.GenerationBatches = len(gen.Sets)
	review, err := DiagnosticSlotPlan(docs, raw, selectAll(docs), targets)
	if err != nil {
		return nil, err
	}
	s.SourceEquivalentReviews = len(review.Sets)
	return s, nil
}
