package excelkit

// MBankStatementCols holds the 0-based column indexes of the fields the
// bank-result importers read from an MBank "Kết quả giao dịch" bulk-transfer
// statement export. Indexes are -1 when the export omits the column.
type MBankStatementCols struct {
	AccountNumber   int
	AccountName     int
	BankName        int
	Amount          int
	TransactionCode int
	Status          int
	BankRef         int
}

// maxStatementHeaderScan bounds how far into the sheet the header row is
// searched. The statement carries ~11 banner rows above the column headers.
const maxStatementHeaderScan = 40

// LocateMBankStatementHeader finds the column-header row of the MBank
// "Kết quả giao dịch" bulk-transfer statement ("THÔNG TIN CHI TIẾT GIAO DỊCH
// LÔ / Detailed Statement of Bulk Transfer"). Columns are located by header
// text, never by position, so renamed sheets or extra banner rows still
// parse. It returns the header row index, the column map, and whether the
// row carries the required columns (account, amount, transaction code,
// transaction status). Files that do not match keep their owning island's
// legacy positional parsing.

// mapStatementHeaderRow maps one candidate row's cells to statement fields
// by normalized header text. More specific phrases are matched first so
// "NGÂN HÀNG THỤ HƯỞNG" can never claim the beneficiary-name column.

// normalizeStatementCell prepares a header cell for keyword matching: NFC
// form (Vietnamese exports may arrive NFD), lower-cased, whitespace
// collapsed.
