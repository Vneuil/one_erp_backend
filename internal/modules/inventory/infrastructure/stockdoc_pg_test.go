package infrastructure

import (
	"context"
	"os"
	"testing"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	financeDomain "github.com/divinecoid/one-backend/internal/modules/finance/domain"
	financeInfra "github.com/divinecoid/one-backend/internal/modules/finance/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/inventory/application"
	"github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	productDomain "github.com/divinecoid/one-backend/internal/modules/product/domain"
	productInfra "github.com/divinecoid/one-backend/internal/modules/product/infrastructure"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Runs against a real Postgres when INVENTORY_TEST_DSN is set (point it at an EMPTY scratch database).
func TestStockDocumentsAgainstPostgres(t *testing.T) {
	dsn := os.Getenv("INVENTORY_TEST_DSN")
	if dsn == "" {
		t.Skip("INVENTORY_TEST_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.Warehouse{}, &domain.StockLevel{}, &domain.StockMovement{}, &domain.StockBatch{}, &domain.StockOpname{},
		&domain.StockOpnameLine{}, &domain.StockDocument{}, &domain.StockDocumentLine{}, &productDomain.Product{},
		&financeDomain.Account{}, &financeDomain.JournalEntry{}, &financeDomain.JournalLine{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	repo := NewInventoryRepository(db)
	finRepo := financeInfra.NewFinanceRepository(db)
	if err := financeApp.NewFinanceUseCase(finRepo).SeedInitialData(ctx); err != nil {
		t.Fatal(err)
	}
	uc := application.NewInventoryUseCase(repo, productInfra.NewProductRepository(db), application.WithLedger(financeApp.NewLedgerPoster(finRepo)))

	wh := domain.Warehouse{Code: "W1", Name: "Utama", IsActive: true}
	wh.ID = uuid.New()
	pa := productDomain.Product{SKU: "SKU-A", Name: "Tepung", Category: "x", Unit: "kg", CostPrice: 5000}
	pb := productDomain.Product{SKU: "SKU-B", Name: "Gula", Category: "x", Unit: "kg", CostPrice: 2000}
	pa.ID, pb.ID = uuid.New(), uuid.New()
	for _, v := range []any{&wh, &pa, &pb} {
		if err := db.Create(v).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []uuid.UUID{pa.ID, pb.ID} {
		if _, err := uc.AdjustStock(ctx, application.AdjustStockDTO{ProductID: p, WarehouseID: wh.ID, Quantity: 100}); err != nil {
			t.Fatal(err)
		}
	}

	issue, err := uc.CreateStockDocument(ctx, application.CreateStockDocumentDTO{Type: domain.DocMaterialIssue, WarehouseID: wh.ID, Reference: "PO-1",
		Lines: []application.StockDocumentLineInput{{ProductID: pa.ID, Quantity: 30}, {ProductID: pb.ID, Quantity: 10}}})
	if err != nil || !issue.Posted || issue.TotalValue != 170000 || len(issue.Lines) != 2 || issue.Lines[0].ProductSKU == "" {
		t.Fatalf("issue: %+v err=%v", issue, err)
	}
	got, err := uc.GetStockDocument(ctx, issue.ID)
	if err != nil || got.Number != issue.Number || len(got.Lines) != 2 || got.WarehouseName != "Utama" {
		t.Fatalf("get: %+v err=%v", got, err)
	}
	if _, err := uc.CreateStockDocument(ctx, application.CreateStockDocumentDTO{Type: domain.DocScrap, Reason: "rusak", WarehouseID: wh.ID,
		Lines: []application.StockDocumentLineInput{{ProductID: pa.ID, Quantity: 5}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.CreateStockDocument(ctx, application.CreateStockDocumentDTO{Type: domain.DocScrap, Reason: "rusak", WarehouseID: wh.ID,
		Lines: []application.StockDocumentLineInput{{ProductID: pa.ID, Quantity: 500}}}); err == nil {
		t.Fatal("scrapping more than is on hand must fail")
	}
	list, err := uc.ListStockDocuments(ctx, "", "", "")
	if err != nil || len(list) != 2 {
		t.Fatalf("list: %d err=%v", len(list), err)
	}
	if only, _ := uc.ListStockDocuments(ctx, domain.DocScrap, "", ""); len(only) != 1 || only[0].Type != domain.DocScrap {
		t.Fatalf("filter by type: %+v", only)
	}
	if lv, _ := repo.GetStockLevel(ctx, pa.ID, wh.ID); lv.Quantity != 65 {
		t.Fatalf("stock A after issue and scrap: %d", lv.Quantity)
	}

	// The journal balances and shows the issue and the scrap.
	tb, err := financeApp.NewFinanceUseCase(finRepo).TrialBalance(ctx, "")
	if err != nil || tb.TotalDebit != tb.TotalCredit {
		t.Fatalf("trial balance: %+v err=%v", tb, err)
	}
	byCode := map[string]float64{}
	for _, l := range tb.Lines {
		byCode[l.AccountCode] = l.Debit - l.Credit
	}
	if byCode[financeApp.AccountWIP] != 170000 || byCode[financeApp.AccountScrapLoss] != 25000 || byCode[financeApp.AccountInventory] != -195000 {
		t.Fatalf("balances by account: %v", byCode)
	}

	// Opname: import counts by SKU, finalize and check the variance journal.
	op, err := uc.CreateOpname(ctx, application.CreateOpnameDTO{WarehouseID: wh.ID})
	if err != nil || len(op.Lines) != 2 {
		t.Fatalf("opname: %+v err=%v", op, err)
	}
	res, err := uc.ImportOpnameCounts(ctx, op.ID, []application.OpnameCountRow{{SKU: "sku-a", CountedQty: 60}, {SKU: "NOPE", CountedQty: 1}})
	if err != nil || res.Applied != 1 || len(res.Errors) != 1 || res.Opname.Status != "in_progress" {
		t.Fatalf("import: %+v err=%v", res, err)
	}
	if _, err := uc.FinalizeOpname(ctx, op.ID); err != nil {
		t.Fatal(err)
	}
	tb, _ = financeApp.NewFinanceUseCase(finRepo).TrialBalance(ctx, "")
	byCode = map[string]float64{}
	for _, l := range tb.Lines {
		byCode[l.AccountCode] = l.Debit - l.Credit
	}
	// Product A: 65 on hand, counted 60 => 5 short x 5,000 = 25,000 more adjustment loss.
	if byCode[financeApp.AccountInventoryAdjustment] != 25000 || tb.TotalDebit != tb.TotalCredit {
		t.Fatalf("after opname: %v", byCode)
	}
}
