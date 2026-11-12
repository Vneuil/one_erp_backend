package application

import "testing"

func TestNextVendorSeqPerType(t *testing.T) {
	existing := []string{"VD-PRD-001", "VD-PRD-004", "VD-PCK-007", "SUP-099"}
	if got := nextVendorSeq("PRD", existing); got != 5 {
		t.Fatalf("PRD next = %d, want 5", got)
	}
	if got := nextVendorSeq("PCK", existing); got != 8 {
		t.Fatalf("PCK next = %d, want 8", got)
	}
	if got := nextVendorSeq("ATK", existing); got != 1 {
		t.Fatalf("ATK next = %d, want 1", got)
	}
	if formatVendorCode("ATK", 2) != "VD-ATK-002" {
		t.Fatal("bad format")
	}
}

func TestVendorCodeValidation(t *testing.T) {
	if _, ok := vendorCodeType("VD-PRD-001"); !ok {
		t.Fatal("VD-PRD-001 must be valid")
	}
	for _, bad := range []string{"PCK-001", "V-PCK-001", "VD-PRD-1", "vd-prd-001"} {
		if _, ok := vendorCodeType(bad); ok {
			t.Fatalf("%s must be invalid", bad)
		}
	}
	if _, ok := normalizeVendorType("p1"); ok {
		t.Fatal("digits are not a valid type")
	}
}
