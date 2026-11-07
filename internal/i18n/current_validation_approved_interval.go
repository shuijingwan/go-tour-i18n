package i18n

// InspectApprovedIntervalPredecessors is a readonly engineering probe. The
// in-memory request has exactly the same strict contract as the export file.
// Ordinary inspection owns the baseline; only explicitly selected Units receive
// the approved-interval overlay, whose authority is independently reconstructed.
func InspectApprovedIntervalPredecessors(root string, c *Catalog, m *RetranslationBatchManifest, request []byte) ([]CurrentValidationPredecessor, error) {
	rows, _, err := InspectApprovedIntervalReadiness(root, c, m, request)
	return rows, err
}

// InspectApprovedIntervalReadiness exposes the same freshly verified derived
// proofs for engineering evidence; it publishes no transport or formal state.
func InspectApprovedIntervalReadiness(root string, c *Catalog, m *RetranslationBatchManifest, request []byte) ([]CurrentValidationPredecessor, []ApprovedIntervalTransport, error) {
	selection, err := parseApprovedIntervalSelectionBytes(root, request, c, m)
	if err != nil {
		return nil, nil, err
	}
	rows, err := inspectCurrentValidationPredecessors(root, c, m.Locale, m.Units)
	if err != nil {
		return nil, nil, err
	}
	_, reopen, err := surfaceRevisionCatalog(root, c, m.Locale, m.Units[0].RevisionAuthorizationID)
	if err != nil {
		return nil, nil, err
	}
	g, err := LoadGlossary(root, m.Locale)
	if err != nil {
		return nil, nil, err
	}
	var proofs []ApprovedIntervalTransport
	for i, r := range m.Units {
		if selection.entry(r.UnitID) == nil {
			continue
		}
		row := &rows[i]
		proof, err := buildApprovedIntervalTransport(root, c, g, m.Locale, r, reopen, row.Snapshot, row.evidence, selection)
		if err != nil {
			row.RecoveryClass, row.Blocker = "APPROVED_INTERVAL_FAIL_CLOSED", err.Error()
		} else {
			row.RecoveryClass, row.Blocker = "APPROVED_INTERVAL_PASS", ""
			row.Template, row.TemplateSHA256 = proof.Projection.Text, sum([]byte(proof.Projection.Text))
			row.output = proof.output
			proofs = append(proofs, *proof)
		}
	}
	return rows, proofs, nil
}
