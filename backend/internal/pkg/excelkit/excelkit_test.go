package excelkit

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestNormalizeHeader(t *testing.T) {
	cases := []struct{ in, want string }{
		{"  Họ và Tên  ", "họ và tên"}, // trim + lower
		{"họ và tên", "họ và tên"},   // NFD → NFC
		{"Số  TK", "số  tk"},           // internal whitespace preserved
		{"", ""},
	}
	for _, tc := range cases {
		if got := NormalizeHeader(tc.in); got != tc.want {
			t.Errorf("NormalizeHeader(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCollapseHeader(t *testing.T) {
	cases := []struct{ in, want string }{
		{"  Số   tài  khoản ", "số tài khoản"}, // collapsed to single spaces
		{"STT\n(Ord. No.)", "stt (ord. no.)"},  // newline → space
		{"", ""},
	}
	for _, tc := range cases {
		if got := CollapseHeader(tc.in); got != tc.want {
			t.Errorf("CollapseHeader(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// diacritic key (collapsed match)
// ASCII key (unidecode retry)

// collapsed whole-cell key

func TestCellAndNumericCell(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	if err := f.SetCellValue("Sheet1", "B2", "  Nguyễn Văn A  "); err != nil {
		t.Fatal(err)
	}
	if err := f.SetCellValue("Sheet1", "B3", 7.5); err != nil {
		t.Fatal(err)
	}
	if err := f.SetCellValue("Sheet1", "B4", 0); err != nil {
		t.Fatal(err)
	}

	if got := Cell(f, "Sheet1", 2, 2); got != "Nguyễn Văn A" {
		t.Errorf("Cell = %q, want trimmed name", got)
	}
	if got := Cell(f, "Sheet1", 99, 99); got != "" {
		t.Errorf("Cell out of range = %q, want empty", got)
	}

	if v, ok := NumericCell(f, "Sheet1", 2, 3); !ok || v != 7.5 {
		t.Errorf("NumericCell = (%v, %v), want (7.5, true)", v, ok)
	}
	if v, ok := NumericCell(f, "Sheet1", 2, 4); !ok || v != 0 {
		t.Errorf("NumericCell explicit zero = (%v, %v), want (0, true)", v, ok)
	}
	if _, ok := NumericCell(f, "Sheet1", 2, 5); ok {
		t.Error("NumericCell blank = ok, want false")
	}
}

func TestExcelDate(t *testing.T) {
	if d, ok := ExcelDate("45978"); !ok || d.Format("2006-01-02") != "2025-11-17" {
		t.Errorf("ExcelDate(45978) = (%v, %v), want 2025-11-17", d.Format("2006-01-02"), ok)
	}
	// Fractional serial floors to the date.
	if d, ok := ExcelDate("45978.75"); !ok || d.Hour() != 0 {
		t.Errorf("fractional serial not floored: %v", d)
	}
	if _, ok := ExcelDate("abc"); ok {
		t.Error("non-numeric accepted")
	}
	if _, ok := ExcelDate("0"); ok {
		t.Error("serial < 1 accepted")
	}
	if _, ok := ExcelDate("1"); ok {
		t.Error("1900 serial accepted outside 2020-2040 guard")
	}
}

func TestOpenReader(t *testing.T) {
	// A valid workbook opens through the capped reader.
	src := excelize.NewFile()
	if err := src.SetCellValue("Sheet1", "A1", "ok"); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := src.Write(&buf); err != nil {
		t.Fatal(err)
	}

	f, err := OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	defer func() { _ = f.Close() }()
	if got := Cell(f, "Sheet1", 1, 1); got != "ok" {
		t.Errorf("roundtrip cell = %q", got)
	}

	// Garbage is rejected with an error, not a panic.
	if _, err := OpenReader(bytes.NewReader([]byte("not a zip"))); err == nil {
		t.Error("garbage input accepted")
	}

	// Caps are the documented defaults.
	if DefaultUnzipSizeLimit != 64<<20 || DefaultUnzipXMLSizeLimit != 32<<20 {
		t.Errorf("unexpected caps: %d / %d", DefaultUnzipSizeLimit, DefaultUnzipXMLSizeLimit)
	}
}

func TestUnzipLimitEnvOverrides(t *testing.T) {
	t.Setenv("EXCEL_UNZIP_MAX_BYTES", "104857600")
	t.Setenv("EXCEL_XML_MAX_BYTES", "52428800")
	if got := UnzipSizeLimit(); got != 100<<20 {
		t.Errorf("UnzipSizeLimit() = %d, want 100 MiB override", got)
	}
	if got := UnzipXMLSizeLimit(); got != 50<<20 {
		t.Errorf("UnzipXMLSizeLimit() = %d, want 50 MiB override", got)
	}
}

func TestUnzipLimitInvalidEnvFallsBack(t *testing.T) {
	t.Setenv("EXCEL_UNZIP_MAX_BYTES", "not-a-number")
	t.Setenv("EXCEL_XML_MAX_BYTES", "-5")
	if got := UnzipSizeLimit(); got != DefaultUnzipSizeLimit {
		t.Errorf("UnzipSizeLimit() = %d, want default %d", got, DefaultUnzipSizeLimit)
	}
	if got := UnzipXMLSizeLimit(); got != DefaultUnzipXMLSizeLimit {
		t.Errorf("UnzipXMLSizeLimit() = %d, want default %d", got, DefaultUnzipXMLSizeLimit)
	}
}
