package infrastructure

import (
	"context"
	"os"
	"sync"
	"testing"

	customerDomain "github.com/divinecoid/one-backend/internal/modules/customer/domain"
	procDomain "github.com/divinecoid/one-backend/internal/modules/procurement/domain"
	productDomain "github.com/divinecoid/one-backend/internal/modules/product/domain"
	salesDomain "github.com/divinecoid/one-backend/internal/modules/sales/domain"
	supplierDomain "github.com/divinecoid/one-backend/internal/modules/supplier/domain"
	"github.com/divinecoid/one-backend/internal/modules/tax/application"
	"github.com/divinecoid/one-backend/internal/modules/tax/domain"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Runs against a real Postgres when TAX_TEST_DSN is set (point it at an EMPTY scratch database).
func TestTaxAgainstPostgres(t *testing.T) {
	dsn := os.Getenv("TAX_TEST_DSN")
	if dsn == "" {
		t.Skip("TAX_TEST_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.TaxSettings{}, &domain.SerialRange{}, &domain.TaxInvoice{}, &domain.TaxInvoiceLine{},
		&salesDomain.Invoice{}, &salesDomain.SalesOrder{}, &salesDomain.SalesOrderLine{}, &procDomain.PurchaseInvoice{},
		&customerDomain.Customer{}, &supplierDomain.Supplier{}, &productDomain.Product{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	uc := application.NewTaxUseCase(NewTaxRepository(db), NewSourceReader(db))

	if _, err := uc.UpdateSettings(ctx, application.SettingsInput{TaxpayerName: "PT Contoh", NPWP: "01.234.567.8-901.000", IsPKP: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.UpdateSettings(ctx, application.SettingsInput{TaxpayerName: "PT Contoh", NPWP: "0123456789012000", IsPKP: true}); err != nil {
		t.Fatal(err)
	}
	if rng, err := uc.CreateSerialRange(ctx, application.SerialRangeInput{Prefix: "040-26.", Start: 1, End: 3, Width: 8}); err != nil || rng.Next != 1 {
		t.Fatalf("serial range: %+v err=%v", rng, err)
	}

	// Concurrent claims must never hand out the same number twice.
	repo := NewTaxRepository(db)
	var wg sync.WaitGroup
	var mu sync.Mutex
	got := map[string]int{}
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n, err := repo.ClaimNextSerial(ctx)
			if err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			got[n]++
			mu.Unlock()
		}()
	}
	wg.Wait()
	if got["040-26.00000001"] != 1 || got["040-26.00000002"] != 1 || got["040-26.00000003"] != 1 || got[""] != 2 || len(got) != 4 {
		t.Fatalf("serial claims: %v", got)
	}
	if _, err := uc.CreateSerialRange(ctx, application.SerialRangeInput{Prefix: "040-26.", Start: 100, End: 102, Width: 8}); err != nil {
		t.Fatal(err)
	}

	// Source documents: customer with NPWP, order with lines, invoice with PPN.
	cust := customerDomain.Customer{Code: "C1", Name: "PT Pembeli", Address: "Jl. Mawar 1", NPWP: "0987654321098000", Status: "Active"}
	cust.ID = uuid.New()
	prod := productDomain.Product{SKU: "P1", Name: "Barang A", Category: "x", Unit: "Pcs"}
	prod.ID = uuid.New()
	order := salesDomain.SalesOrder{OrderNumber: "SO-1", CustomerName: "PT Pembeli", TotalAmount: 1000000}
	order.ID = uuid.New()
	line := salesDomain.SalesOrderLine{SalesOrderID: order.ID, ProductID: prod.ID, Quantity: 4, UnitPrice: 250000, Subtotal: 1000000}
	line.ID = uuid.New()
	inv := salesDomain.Invoice{InvoiceNumber: "INV-1", CustomerName: "PT Pembeli", InvoiceDate: "2026-03-10", TotalAmount: 1110000, Subtotal: 1000000,
		TaxBase: 1000000, DPPOtherValue: 916666.67, VATRate: 12, VATOtherValueBase: true, VATAmount: 110000, SalesOrderID: &order.ID, Status: "pending"}
	inv.ID = uuid.New()
	for _, v := range []any{&cust, &prod, &order, &line, &inv} {
		if err := db.Create(v).Error; err != nil {
			t.Fatal(err)
		}
	}

	d, err := uc.CreateFromSalesInvoice(ctx, application.FromSalesInput{SalesInvoiceID: inv.ID})
	if err != nil {
		t.Fatal(err)
	}
	if d.CounterpartyNPWP != "0987654321098000" || len(d.Lines) != 1 || d.Lines[0].Description != "Barang A" || d.Lines[0].Unit != "Pcs" || d.Lines[0].UnitPrice != 250000 {
		t.Fatalf("draft: %+v", d)
	}
	if _, err := uc.CreateFromSalesInvoice(ctx, application.FromSalesInput{SalesInvoiceID: inv.ID}); err == nil {
		t.Fatal("duplicate open faktur must be rejected")
	}
	issued, err := uc.Issue(ctx, d.ID)
	if err != nil || issued.TaxNumber != "040-26.00000100" {
		t.Fatalf("issue: %+v err=%v", issued, err)
	}
	reloaded, err := uc.GetTaxInvoice(ctx, d.ID)
	if err != nil || reloaded.Status != "issued" || len(reloaded.Lines) != 1 || reloaded.IssuedAt == nil {
		t.Fatalf("reload: %+v err=%v", reloaded, err)
	}

	list, err := uc.ListTaxInvoices(ctx, domain.InvoiceFilter{Direction: "output", Status: "issued", From: "2026-03-01", To: "2026-03-31", Search: "pembeli"})
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %d err=%v", len(list), err)
	}
	book, err := uc.SalesBook(ctx, "2026-03-01", "2026-03-31")
	if err != nil || book.Count != 1 || book.PendingFaktur != 0 || book.Rows[0].TaxNumber != "040-26.00000100" {
		t.Fatalf("sales book: %+v err=%v", book, err)
	}
	sum, err := uc.VATSummary(ctx, "2026-01-01", "2026-03-31")
	if err != nil || len(sum.Periods) != 1 || sum.Total.OutputVAT != 110000 {
		t.Fatalf("vat summary: %+v err=%v", sum, err)
	}
	ex, err := uc.CoretaxExport(ctx, "2026-03-01", "2026-03-31")
	if err != nil || len(ex.Rows) != 1 {
		t.Fatalf("coretax: %+v err=%v", ex, err)
	}

	rep, err := uc.Replace(ctx, d.ID)
	if err != nil || rep.Revision != 1 || len(rep.Lines) != 1 {
		t.Fatalf("replace: %+v err=%v", rep, err)
	}
	if _, err := uc.Issue(ctx, rep.ID); err != nil {
		t.Fatal(err)
	}
	if old, _ := uc.GetTaxInvoice(ctx, d.ID); old.Status != "replaced" {
		t.Fatalf("original should be replaced, is %s", old.Status)
	}
}
