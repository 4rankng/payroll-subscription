package main



const flowWeeklyBCC = "WeeklyBCCImport"

// runWeeklyBCCImportTests tests the BCC-<shiftType> sheet format upload.
// Uses docs/WeeklyBCC/BCC LGD.xlsx which has BCC-HC, BCC-OT150, etc. sheets.
func runWeeklyBCCImportTests(client *APIClient, data *TestData, reporter *Reporter) {
	reporter.PrintSection("FLOW: Weekly BCC Import (BCC-<shiftType> sheets)")

	// PARKED (PAYSUB-11 local-dev pass): the BCC LGD fixture is a May-2026
	// rateless weekly file whose import resolution needs donor-era payrate
	// semantics (position/day-type leaf matching) that the consolidated
	// schema + subscription model no longer ship as seed data. The upload
	// pipeline contract (202 acceptance, latest-wins, detail/history reads)
	// is covered by flow_bcc_import, which runs the same endpoints and
	// passes; re-enable this flow when a PAYSUB-era fixture workbook exists.
	reporter.Skip(flowWeeklyBCC, "All weekly BCC tests", "fixture workbook predates the subscription model; coverage via flow_bcc_import")
	return
}
