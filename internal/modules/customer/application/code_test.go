package application

import (
	"context"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/customer/domain"
	"github.com/google/uuid"
)

type fakeCustomerRepo struct {
	domain.CustomerRepository
	byCode map[string]*domain.Customer
}

func (f *fakeCustomerRepo) GetByCode(_ context.Context, code string) (*domain.Customer, error) {
	return f.byCode[code], nil
}
func (f *fakeCustomerRepo) CodesWithPrefix(_ context.Context, prefix string) ([]string, error) {
	var out []string
	for c := range f.byCode {
		if len(c) >= len(prefix) && c[:len(prefix)] == prefix {
			out = append(out, c)
		}
	}
	return out, nil
}
func (f *fakeCustomerRepo) Create(_ context.Context, c *domain.Customer) error {
	c.ID = uuid.New()
	f.byCode[c.Code] = c
	return nil
}

func TestCustomerCodeFollowsTypeAndNeverCollides(t *testing.T) {
	repo := &fakeCustomerRepo{byCode: map[string]*domain.Customer{"B2B-0001": {}, "B2B-0003": {}, "B2C-0001": {}}}
	uc := NewCustomerUseCase(repo)
	ctx := context.Background()

	b2b := CreateCustomerDTO{Name: "PT Acme", CustomerType: "B2B", ContactPerson: "Budi", ContactMobile: "0812"}
	res, err := uc.Create(ctx, b2b)
	if err != nil {
		t.Fatal(err)
	}
	if res.Code != "B2B-0004" {
		t.Fatalf("expected B2B-0004, got %s", res.Code)
	}

	res, err = uc.Create(ctx, CreateCustomerDTO{Name: "Ibu Sari", CustomerType: "B2C"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Code != "B2C-0002" {
		t.Fatalf("expected B2C-0002, got %s", res.Code)
	}

	if _, err := uc.Create(ctx, CreateCustomerDTO{Code: "b2b-0001", Name: "Dup", ContactPerson: "x", ContactPhone: "1"}); err == nil {
		t.Fatal("an explicit duplicate code must still be rejected")
	}
	if _, err := uc.Create(ctx, CreateCustomerDTO{}); err == nil {
		t.Fatal("a name is still required")
	}
}

func TestB2BRequiresContactPersonAndPhone(t *testing.T) {
	uc := NewCustomerUseCase(&fakeCustomerRepo{byCode: map[string]*domain.Customer{}})
	ctx := context.Background()
	if _, err := uc.Create(ctx, CreateCustomerDTO{Name: "PT X", CustomerType: "B2B"}); err == nil {
		t.Fatal("B2B without contact person must fail")
	}
	if _, err := uc.Create(ctx, CreateCustomerDTO{Name: "PT X", CustomerType: "B2B", ContactPerson: "Budi"}); err == nil {
		t.Fatal("B2B without any phone channel must fail")
	}
	if _, err := uc.Create(ctx, CreateCustomerDTO{Name: "PT X", CustomerType: "B2B", ContactPerson: "Budi", ContactWhatsapp: "0812"}); err != nil {
		t.Fatalf("WhatsApp alone satisfies the phone requirement: %v", err)
	}
	if _, err := uc.Create(ctx, CreateCustomerDTO{Name: "Pak Joko", CustomerType: "B2C"}); err != nil {
		t.Fatalf("B2C needs no contact person: %v", err)
	}
}

func TestShippingAddressesGetExactlyOneDefault(t *testing.T) {
	out := toAddressEntities([]AddressDTO{{Address: "Gudang A"}, {Address: "  "}, {Address: "Pabrik B", IsDefault: true}, {Address: "C", IsDefault: true}})
	if len(out) != 3 {
		t.Fatalf("blank address must be dropped, got %d", len(out))
	}
	n := 0
	for _, a := range out {
		if a.IsDefault {
			n++
		}
	}
	if n != 1 || !out[1].IsDefault {
		t.Fatalf("expected the first flagged address to be the single default: %+v", out)
	}
}
