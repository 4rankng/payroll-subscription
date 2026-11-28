package excelkit

// mbankListingRows replicates the 6-column outbound chuyenlo listing the
// MBank manual template produces: title row, header row, one data row.
func mbankListingRows() [][]string {
	return [][]string{
		{"", "DANH SÁCH GIAO DỊCH\n(LIST OF TRANSACTIONS)", "", "", "", ""},
		{
			"STT\n(Ord. No.)\n(1)",
			"Số tài khoản\n(Account No.)\n(2)",
			"Tên người thụ hưởng\n(Beneficiary)\n(3)",
			"Ngân hàng thụ hưởng/Chi nhánh\n(Beneficiary Bank)\n(4)",
			"Số tiền\n(Amount)\n(5)",
			"Nội dung chuyển khoản\n(Payment Detail)\n(6)",
		},
		{"1", "0989505854", "PHẠM DOÃN DŨNG", "Quân đội (MB)", "7583325", "VFIC931c9934"},
	}
}

// onepayListingRows replicates the 7-column OnePay eMB_BulkPayment listing,
// which inserts a SWIFT code column at E and shifts amount/payment detail
// right. The locator must resolve it without any positional knowledge.
func onepayListingRows() [][]string {
	return [][]string{
		{
			"STT\n(Ord. No.)",
			"Số tài khoản\n(Account No.)",
			"Tên người thụ hưởng\n(Beneficiary)",
			"Ngân hàng thụ hưởng/Chi nhánh\n(Beneficiary Bank)",
			"Mã SWIFT\n(SWIFT Code)",
			"Số tiền\n(Amount)",
			"Nội dung chuyển khoản\n(Payment Detail)",
		},
		{"1", "80001708644", "DOAN THI HONG", "NGÂN HÀNG TMCP HÀNG HẢI", "MBBEVNVX", "930000", "VFICa8ab43d8"},
	}
}

// The locator must reject a listing that drops the payment-detail column —
// without a transaction code the rows cannot be re-matched to a batch.

// A "Tổng số tiền" summary banner row carries the phrase "số tiền"; reading
// it as the amount header would silently read the grand total as the column.

// A result workbook (the KQ sheet) is not an outbound listing: it must never
// be mistaken for one, or a re-upload could be fed back through the converter.
