package billing

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestParseTransferResultExcel(t *testing.T) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	rows := [][]interface{}{
		{"Người nộp", "", "Ghi chú"},
		{"Họ tên", "CCCD", "Số tiền", "Mã tham chiếu"},
		{"Nguyễn Văn A", "012345678901", "1.234.567", "FT2611191234"},
		{"", "", "", ""},
		{"Trần Thị B", "098765432109", "2 000 000 đ", ""},
	}
	for i, row := range rows {
		if err := f.SetSheetRow(sheet, fmt.Sprintf("A%d", i+1), &row); err != nil {
			t.Fatalf("set row: %v", err)
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := ParseTransferResultExcel(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 rows (blank separator skipped), got %d: %+v", len(got), got)
	}
	if got[0].CCCD != "012345678901" || got[0].Amount != 1234567 || got[0].Reference != "FT2611191234" || got[0].FullName != "Nguyễn Văn A" {
		t.Fatalf("row 0 mismatch: %+v", got[0])
	}
	if got[1].CCCD != "098765432109" || got[1].Amount != 2000000 || got[1].Reference != "" {
		t.Fatalf("row 1 mismatch: %+v", got[1])
	}
	if got[0].RowNumber != 3 || got[1].RowNumber != 5 {
		t.Fatalf("row numbers want [3 5], got [%d %d]", got[0].RowNumber, got[1].RowNumber)
	}
}

func TestParseTransferResultExcelNoCCCDColumn(t *testing.T) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	if err := f.SetSheetRow(sheet, "A1", &[]interface{}{"Name", "Amount"}); err != nil {
		t.Fatalf("set row: %v", err)
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := ParseTransferResultExcel(bytes.NewReader(buf.Bytes())); err == nil {
		t.Fatal("want error when CCCD column missing")
	}
}

func TestParseTransferResultEnglishHeaders(t *testing.T) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	if err := f.SetSheetRow(sheet, "A1", &[]interface{}{"Full Name", "CCCD", "Amount", "Reference"}); err != nil {
		t.Fatalf("set row: %v", err)
	}
	if err := f.SetSheetRow(sheet, "A2", &[]interface{}{"C", "012345678901", "1234567", "REF1"}); err != nil {
		t.Fatalf("set row: %v", err)
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := ParseTransferResultExcel(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 1 || got[0].Amount != 1234567 || got[0].Reference != "REF1" {
		t.Fatalf("mismatch: %+v", got)
	}
}
