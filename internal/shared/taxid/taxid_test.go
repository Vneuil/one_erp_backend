package taxid

import "testing"

func TestNPWPAndNIK(t *testing.T) {
	if d, err := NPWP("01.234.567.8-901.000"); err != nil || d != "012345678901000" {
		t.Fatalf("npwp: %q %v", d, err)
	}
	if d, err := NPWP("0123456789012345"); err != nil || len(d) != 16 {
		t.Fatalf("16-digit npwp: %q %v", d, err)
	}
	if _, err := NPWP("12345"); err == nil {
		t.Fatal("short npwp must fail")
	}
	if d, err := NPWP("  "); err != nil || d != "" {
		t.Fatal("blank npwp is allowed")
	}
	if _, err := NIK("123"); err == nil {
		t.Fatal("short nik must fail")
	}
	if d, err := NIK("3201 0112 3456 7890"); err != nil || d != "3201011234567890" {
		t.Fatalf("nik: %q %v", d, err)
	}
	if NPWP16("012345678901000") != "0012345678901000" || NPWP16("0123456789012345") != "0123456789012345" {
		t.Fatal("npwp16 padding wrong")
	}
}
