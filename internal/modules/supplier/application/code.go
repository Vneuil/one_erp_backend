package application

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// VendorTypes are the code families Manna uses: VD-PRD-001, VD-PCK-001, VD-ATK-001.
// Other 2-4 letter families are accepted (VD-xxx-xxx) but these are the listed defaults.
var VendorTypes = []VendorTypeInfo{
	{Code: "PRD", Label: "Produk / Bahan Baku"},
	{Code: "PCK", Label: "Kemasan (Packaging)"},
	{Code: "ATK", Label: "Alat Tulis Kantor"},
}

type VendorTypeInfo struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

const vendorCodePrefix = "VD-"

var vendorCodeRe = regexp.MustCompile(`^VD-([A-Z]{2,4})-(\d{3,})$`)
var vendorTypeRe = regexp.MustCompile(`^[A-Z]{2,4}$`)

// normalizeVendorType uppercases and validates a vendor type; empty stays empty.
func normalizeVendorType(t string) (string, bool) {
	t = strings.ToUpper(strings.TrimSpace(t))
	if t == "" {
		return "", true
	}
	return t, vendorTypeRe.MatchString(t)
}

// vendorCodeType extracts the type from a well-formed VD-<type>-<seq> code.
func vendorCodeType(code string) (string, bool) {
	m := vendorCodeRe.FindStringSubmatch(code)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// nextVendorSeq returns the highest sequence among codes of the given type plus one.
func nextVendorSeq(vendorType string, existing []string) int {
	max := 0
	for _, c := range existing {
		m := vendorCodeRe.FindStringSubmatch(strings.ToUpper(c))
		if m == nil || m[1] != vendorType {
			continue
		}
		if n, err := strconv.Atoi(m[2]); err == nil && n > max {
			max = n
		}
	}
	return max + 1
}

func formatVendorCode(vendorType string, seq int) string {
	return fmt.Sprintf("%s%s-%03d", vendorCodePrefix, vendorType, seq)
}
