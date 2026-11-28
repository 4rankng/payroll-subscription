package excelkit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func bbdsFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Clean(filepath.Join("..", "..", "..", "tests", "fixtures", name)))
	require.NoError(t, err)
	return data
}

// TestOpenWorkbookConvertsLegacyXLS verifies the OLE2 → excelize conversion
// against a real OnePay BBDS_DETAIL_PO export: sheet names survive, BIFF8
// compressed Latin-1 strings are repaired to UTF-8, and numeric cells keep
// their values.

// Vietnamese headers stored as 8-bit BIFF strings must come back as
// valid UTF-8, not raw Latin-1 bytes.

// Numbers stay numeric through the conversion.

// TestOpenWorkbookReadsStrictOOXLSX pins support for the strict-OOXML
// dialect OnePay ships alongside the .xls export; openpyxl-class readers
// reject it, excelize must not.
