package excelkit

// TransferListingCols holds the 0-based column indexes of an OUTBOUND
// transfer listing ("chuyển lô" / bulk-payment request) workbook — the file
// an admin hands to the bank. Indexes are -1 when the listing omits the
// column.
type TransferListingCols struct {
	Order          int
	AccountNumber  int
	AccountName    int
	BankName       int
	Amount         int
	TransactionRef int
}

// maxTransferListingHeaderScan bounds how far into the sheet the header row
// is searched. The listing carries a title row above the column headers.
const maxTransferListingHeaderScan = 20

// LocateTransferListingHeader finds the column-header row of an outbound
// transfer listing. Columns are located by header text, never by position, so
// both outbound shapes parse: the MBank manual template (6 columns, payment
// detail in F) and the OnePay "eMB_BulkPayment" workbook (7 columns, SWIFT
// code in E, payment detail in G).
//
// It returns the header row index, the column map, and whether the row
// carries the columns the KQ converter cannot do without (account, name,
// amount, payment detail).

// mapTransferListingHeaderRow maps one candidate row's cells to listing
// fields by normalized header text. "Ngân hàng thụ hưởng" is matched before
// the beneficiary name so it can never claim that column, and the amount
// match rejects "tổng số tiền" so a summary cell is not read as the header.
