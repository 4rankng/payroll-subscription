package excelkit

// statementHeaderRow replicates the column-header row of the MBank
// "Kết quả giao dịch" bulk-transfer statement (merged cells appear as the
// leading cell followed by empty continuations, exactly like excelize
// GetRows output).
func statementHeaderRow() []string {
	row := make([]string, 25)
	row[0] = " STT"
	row[2] = " SỐ TÀI KHOẢN\n (Account No)"
	row[5] = " TÊN ĐƠN VỊ THỤ HƯỞNG\n (Beneficiary Name)"
	row[6] = " NGÂN HÀNG THỤ HƯỞNG\n (Beneficiary Bank)"
	row[7] = " SỐ TIỀN GIAO DỊCH\n (Amount)"
	row[9] = " CHI TIẾT THANH TOÁN\n (Payment Details)"
	row[12] = " PHÍ (CHƯA GỒM VAT)\n (Charge not VAT)"
	row[15] = " HÌNH THỨC CHUYỂN KHOẢN\n (Payment Method)"
	row[16] = " TRẠNG THÁI GIAO DỊCH\n (Transaction status)"
	row[17] = " TRẠNG THÁI HẠCH TOÁN\n (Account deduction status)"
	row[18] = " GHI CHÚ\n (Note)"
	row[23] = " BÚT TOÁN\n (FT number)"
	return row
}

// statementBannerRows replicates the banner block above the header, including
// the applicant-info rows that reuse phrases like "Số Tài khoản" and
// "Tổng số tiền" — the locator must not mistake them for the header.
func statementBannerRows() [][]string {
	rows := make([][]string, 11)
	for i := range rows {
		rows[i] = make([]string, 25)
	}
	rows[1][4] = "THÔNG TIN CHI TIẾT GIAO DỊCH LÔ"
	rows[1][14] = "NGÂN HÀNG TMCP QUÂN ĐỘI"
	rows[6][0] = "Đơn vị/Người chuyển\n(Applicant)"
	rows[6][5] = "CTY TNHH MTV GP PHAN MEM TING TING"
	rows[6][11] = "Số tham chiếu (Ref No.)"
	rows[6][15] = "202609102419883866"
	rows[7][0] = "Số Tài khoản(A/C No.)"
	rows[7][5] = "271866699"
	rows[7][6] = "Loại tiền \n(Currency)"
	rows[7][7] = "VND"
	rows[7][10] = "Tổng số tiền \n(Total amount of transaction)"
	rows[7][15] = "399206690.0"
	rows[9][11] = "Số giao dịch trong lô \n(The number of transaction)"
	rows[9][15] = "166"
	rows[10][0] = " BẢNG CHI TIẾT GIAO DỊCH TRONG LÔ"
	return rows
}

// Legacy "Kết quả chuyển tiền theo bảng kê" export: short headers, data
// from row 5, transaction code in column F, status in column H.

// If the accounting-status column is ever renamed to include "bút toán",
// it must not claim the FT-reference column slot.
