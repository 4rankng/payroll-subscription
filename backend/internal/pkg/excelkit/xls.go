package excelkit

// The legacy BIFF reader surfaces every cell as one of a handful of record
// types. Numbers and RK-packed numbers are written back as numeric cells so
// excelize's General rendering matches the .xlsx path (parseMoney accepts
// both "345000" and "345,000.00"); everything else is stored as text.
const (
	recordTypeNumber = "*record.Number"
	recordTypeRK     = "*record.Rk"
)

// openXLS converts a legacy BIFF (.xls, OLE2 compound document) workbook into
// an in-memory xlsx workbook, so every downstream excelize-based parser keeps
// working unchanged. Cell values are copied as values (no formulas, styles or
// merges): BIFF text cells come back as strings, numeric cells as floats.
//
// BIFF8 stores plain-Latin text as 8-bit "compressed" strings — the low
// bytes of UTF-16 code units, i.e. Latin-1. The reader returns those bytes
// unconverted, which are invalid UTF-8 (e.g. "Phí" surfaces as "Ph\xed"),
// so every string passes through repairLatin1 before being written. Formulas
// and rich-text records are skipped by the reader itself; OnePay statements
// are literal data dumps, so nothing is lost for the layouts we parse.

// repairLatin1 re-encodes raw Latin-1 bytes (BIFF8 compressed strings) as
// UTF-8. Valid UTF-8 (BIFF8 UTF-16 strings already decoded by the reader)
// passes through untouched.
