package application

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/divinecoid/one-backend/internal/modules/customer/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
)

const (
	CustomerTypeB2B = "B2B" // perusahaan
	CustomerTypeB2C = "B2C" // perorangan
)

var customerCodeRe = regexp.MustCompile(`^(B2B|B2C)-(\d{4,})$`)

func customerCodeType(code string) (string, bool) {
	m := customerCodeRe.FindStringSubmatch(strings.ToUpper(strings.TrimSpace(code)))
	if m == nil {
		return "", false
	}
	return m[1], true
}

// normalizeCustomerType resolves the type from the explicit value, else from an
// explicit code's prefix, else defaults to B2B.
func normalizeCustomerType(t, code string) (string, error) {
	t = strings.ToUpper(strings.TrimSpace(t))
	if t == "" {
		if m := customerCodeRe.FindStringSubmatch(code); m != nil {
			return m[1], nil
		}
		return CustomerTypeB2B, nil
	}
	if t != CustomerTypeB2B && t != CustomerTypeB2C {
		return "", apperrors.NewBadRequest("Customer type must be B2B (perusahaan) or B2C (perorangan)")
	}
	return t, nil
}

// validateB2BContact: a company customer needs a contact person and at least one phone channel.
func validateB2BContact(ctype, person, phone, mobile, wa string) error {
	if ctype != CustomerTypeB2B {
		return nil
	}
	if strings.TrimSpace(person) == "" {
		return apperrors.NewBadRequest("B2B customers require a Contact Person name")
	}
	if strings.TrimSpace(phone) == "" && strings.TrimSpace(mobile) == "" && strings.TrimSpace(wa) == "" {
		return apperrors.NewBadRequest("B2B customers require at least one of phone, mobile or WhatsApp")
	}
	return nil
}

func nextCustomerSeq(ctype string, existing []string) int {
	max := 0
	for _, c := range existing {
		m := customerCodeRe.FindStringSubmatch(strings.ToUpper(c))
		if m == nil || m[1] != ctype {
			continue
		}
		if n, err := strconv.Atoi(m[2]); err == nil && n > max {
			max = n
		}
	}
	return max + 1
}

func formatCustomerCode(ctype string, seq int) string { return fmt.Sprintf("%s-%04d", ctype, seq) }

func normalizeStatus(s string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "active":
		return "Active", nil
	case "inactive":
		return "Inactive", nil
	}
	return "", apperrors.NewBadRequest("Status must be Active or Inactive")
}

// defaultCity falls back to Semarang when no city is given.
func defaultCity(c string) string {
	if c = strings.TrimSpace(c); c != "" {
		return c
	}
	return "Semarang"
}

func toAddressEntities(in []AddressDTO) []domain.CustomerAddress {
	out := make([]domain.CustomerAddress, 0, len(in))
	hasDefault := false
	for _, a := range in {
		if strings.TrimSpace(a.Address) == "" {
			continue
		}
		e := domain.CustomerAddress{
			Label: strings.TrimSpace(a.Label), RecipientName: a.RecipientName, Phone: a.Phone,
			Address: strings.TrimSpace(a.Address), City: a.City, Province: a.Province, PostalCode: a.PostalCode,
			IsDefault: a.IsDefault && !hasDefault,
		}
		hasDefault = hasDefault || e.IsDefault
		out = append(out, e)
	}
	if len(out) > 0 && !hasDefault {
		out[0].IsDefault = true
	}
	return out
}
