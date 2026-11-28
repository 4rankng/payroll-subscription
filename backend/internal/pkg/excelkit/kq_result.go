package excelkit

// KQ result workbook layout. This is the "Kết quả chuyển tiền" sheet the
// bank-result importer reads back through TransactionCodeStrategy: sheet
// `data`, rows 1-3 title block, row 4 column headers, data from row 5, nine
// columns A-I.
//
// The layout lives here — not in the wallet or payroll service — because two
// independent producers write it (the wallet batch KQ download and the
// "Tạo KQ CK" chuyenlo-file converter) and one importer parses it.
const (
	KQSheetName    = "data"
	kqTitleRow     = 1
	kqRefRow       = 2
	kqDateRow      = 3
	KQHeaderRow    = 4
	KQFirstDataRow = 5
	kqLastCol      = "I"
)

// KQHeaders is the row-4 header set of the KQ result workbook.
var KQHeaders = []string{
	"STT",
	"Số tài khoản",
	"Tên người thụ hưởng",
	"Ngân hàng thụ hưởng",
	"Số tiền",
	"Nội dung",
	"Phí",
	"Trạng thái",
	"FT / Ghi chú",
}

// KQRow is one transfer line of a KQ result workbook. Column G (Phí) and
// column I (FT / Ghi chú) are carried verbatim; the importer ignores G and
// reads I as the bank reference on success rows or the error message on
// failed rows.
type KQRow struct {
	Order          int
	AccountNumber  string
	AccountName    string
	BankName       string
	Amount         int64
	TransactionRef string
	Fee            int64
	Status         string
	Reference      string
}

// KQTitle carries the three title-block rows printed above the headers.
type KQTitle struct {
	Title    string
	Ref      string
	DateLine string
}

// WriteKQWorkbook renders rows into the KQ result workbook layout and returns
// the .xlsx bytes.

// KQDateLine renders the Vietnamese date stamp used by the title block.
