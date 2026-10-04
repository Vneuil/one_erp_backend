// Package taxid normalises and validates Indonesian taxpayer identifiers.
package taxid

import (
	"strings"

	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
)

// Digits strips everything but digits, so "01.234.567.8-901.000" and
// "012345678901000" are the same NPWP.
func Digits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// NPWP returns the digits of an NPWP and validates that it has 15 digits (old
// format) or 16 digits (new format). Empty input is allowed and returns "".
func NPWP(s string) (string, error) {
	d := Digits(s)
	if strings.TrimSpace(s) == "" {
		return "", nil
	}
	if len(d) != 15 && len(d) != 16 {
		return "", apperrors.NewBadRequest("NPWP must have 15 or 16 digits")
	}
	return d, nil
}

// NIK returns the digits of a NIK and validates that it has 16 digits.
// Empty input is allowed and returns "".
func NIK(s string) (string, error) {
	d := Digits(s)
	if strings.TrimSpace(s) == "" {
		return "", nil
	}
	if len(d) != 16 {
		return "", apperrors.NewBadRequest("NIK must have 16 digits")
	}
	return d, nil
}

// NPWP16 pads a 15-digit NPWP with a leading 0 to the 16-digit form used by
// Coretax; 16-digit input is returned as is.
func NPWP16(d string) string {
	if len(d) == 15 {
		return "0" + d
	}
	return d
}
