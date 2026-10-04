package infrastructure

import (
	"context"
	"os"
	"testing"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	financeDomain "github.com/divinecoid/one-backend/internal/modules/finance/domain"
	financeInfra "github.com/divinecoid/one-backend/internal/modules/finance/infrastructure"
	invApp "github.com/divinecoid/one-backend/internal/modules/inventory/application"
	invDomain "github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	invInfra "github.com/divinecoid/one-backend/internal/modules/inventory/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/manufacturing/application"
	"github.com/divinecoid/one-backend/internal/modules/manufacturing/domain"
	productDomain "github.com/divinecoid/one-backend/internal/modules/product/domain"
	productInfra "github.com/divinecoid/one-backend/internal/modules/product/infrastructure"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Runs against a real Postgres when MFG_TEST_DSN is set (point it at an EMPTY scratch database).
func TestProductionRoutingAgainstPostgres(t *testing.T) {
	dsn := os.Getenv("MFG_TEST_DSN")
	if dsn == "" {
		t.Skip("MFG_TEST_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.BillOfMaterial{}, &domain.BOMLine{}, &domain.BOMProcess{}, &domain.ProductionOrder{}, &domain.ProductionBatch{},
		&domain.ProductionStep{}, &domain.ProductionStepLog{}, &invDomain.Warehouse{}, &invDomain.StockLevel{}, &invDomain.StockMovement{}, &invDomain.StockBatch{},
		&invDomain.StockDocument{}, &invDomain.StockDocumentLine{}, &productDomain.Product{},
		&financeDomain.Account{}, &financeDomain.JournalEntry{}, &financeDomain.JournalLine{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	finRepo := financeInfra.NewFinanceRepository(db)
	if err := financeApp.NewFinanceUseCase(finRepo).SeedInitialData(ctx); err != nil {
		t.Fatal(err)
	}
	ledger := financeApp.NewLedgerPoster(finRepo)
	productRepo := productInfra.NewProductRepository(db)
	inventory := invApp.NewInventoryUseCase(invInfra.NewInventoryRepository(db), productRepo, invApp.WithLedger(ledger))
	uc := application.NewManufacturingUseCase(NewManufacturingRepository(db), invInfra.NewInventoryRepository(db), productRepo,
		application.WithLedger(ledger), application.WithWIPReader(LedgerWIPReader{Repo: finRepo}))

	wh := invDomain.Warehouse{Code: "W", Name: "Pabrik", IsActive: true}
	wh.ID = uuid.New()
	fg := productDomain.Product{SKU: "FG", Name: "Kemeja", Category: "x", Unit: "pcs"}
	rm := productDomain.Product{SKU: "RM", Name: "Kain", Category: "x", Unit: "m", CostPrice: 1000}
	fg.ID, rm.ID = uuid.New(), uuid.New()
	for _, v := range []any{&wh, &fg, &rm} {
		if err := db.Create(v).Error; err != nil {
			t.Fatal(err)
		}
	}
	if _, err := inventory.AdjustStock(ctx, invApp.AdjustStockDTO{ProductID: rm.ID, WarehouseID: wh.ID, Quantity: 500}); err != nil {
		t.Fatal(err)
	}

	bom, err := uc.CreateBOM(ctx, application.CreateBOMDTO{ProductID: fg.ID, Name: "Kemeja", Lines: []application.BOMLineDTO{{ComponentProductID: rm.ID, QuantityRequired: 2}},
		Processes: []application.BOMProcessDTO{{Name: "Potong", StandardMinutes: 5}, {Name: "Jahit", StandardMinutes: 20}}})
	if err != nil || len(bom.Processes) != 2 || bom.Processes[0].Name != "Potong" || bom.Processes[1].Sequence != 2 {
		t.Fatalf("bom: %+v err=%v", bom, err)
	}
	got, err := uc.GetBOMByID(ctx, bom.ID)
	if err != nil || len(got.Processes) != 2 || got.Processes[1].Name != "Jahit" {
		t.Fatalf("reloaded bom processes (ordered): %+v err=%v", got, err)
	}

	order, err := uc.CreateOrder(ctx, application.CreateProductionOrderDTO{BOMID: bom.ID, WarehouseID: wh.ID, QuantityToProduce: 10, PlannedDate: "2026-03-10", MaterialMode: "issued"})
	if err != nil || len(order.Steps) != 2 || order.MaterialMode != "issued" || order.OrderNumber == "" {
		t.Fatalf("order: %+v err=%v", order, err)
	}
	if _, err := uc.ReleaseOrder(ctx, order.ID); err != nil {
		t.Fatal(err)
	}

	// Issue materials for 6 units (12 m) against the order, then run the routing.
	issue, err := inventory.CreateStockDocument(ctx, invApp.CreateStockDocumentDTO{Type: invDomain.DocMaterialIssue, WarehouseID: wh.ID, ProductionOrderID: &order.ID,
		Lines: []invApp.StockDocumentLineInput{{ProductID: rm.ID, Quantity: 12}}})
	if err != nil || issue.ProductionOrderID == nil || issue.TotalValue != 12000 {
		t.Fatalf("issue: %+v err=%v", issue, err)
	}
	o, err := uc.LogStep(ctx, order.ID, order.Steps[0].ID, application.LogStepDTO{Quantity: 10, Date: "2026-03-09"})
	if err != nil || o.Status != "in_progress" || o.Steps[0].Status != "done" || o.Steps[1].WaitingQuantity != 10 {
		t.Fatalf("log step 1: %+v err=%v", o, err)
	}
	if _, err := uc.LogStep(ctx, order.ID, order.Steps[1].ID, application.LogStepDTO{Quantity: 11}); err == nil {
		t.Fatal("step 2 cannot exceed step 1")
	}
	if _, err := uc.LogStep(ctx, order.ID, order.Steps[1].ID, application.LogStepDTO{Quantity: 8, Date: "2026-03-10"}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.CompleteBatch(ctx, order.ID, application.CompleteBatchDTO{QuantityCompleted: 9}); err == nil {
		t.Fatal("only 8 units passed the whole routing")
	}
	if _, err := uc.CompleteBatch(ctx, order.ID, application.CompleteBatchDTO{QuantityCompleted: 7}); err == nil {
		t.Fatal("12 m issued covers 6 units, not 7")
	}
	done, err := uc.CompleteBatch(ctx, order.ID, application.CompleteBatchDTO{QuantityCompleted: 6, CompletionDate: "2026-03-11"})
	if err != nil || done.QuantityCompleted != 6 || done.Status != "in_progress" {
		t.Fatalf("complete: %+v err=%v", done, err)
	}

	// Stock: the 12 m left at issue time and nothing more; 6 finished goods arrived.
	var rmLevel, fgLevel invDomain.StockLevel
	db.Where("product_id = ?", rm.ID).First(&rmLevel)
	db.Where("product_id = ?", fg.ID).First(&fgLevel)
	if rmLevel.Quantity != 488 || fgLevel.Quantity != 6 {
		t.Fatalf("stock: rm=%d fg=%d", rmLevel.Quantity, fgLevel.Quantity)
	}
	// Ledger: 12,000 into WIP at issue, 12,000 (6 units x 2,000) out at completion => WIP nets to zero.
	bal, err := LedgerWIPReader{Repo: finRepo}.WIPBalance(ctx)
	if err != nil || bal != 0 {
		t.Fatalf("WIP balance should be 0 after the issued materials were all used, got %v err=%v", bal, err)
	}

	// Reports.
	daily, err := uc.DailyReport(ctx, "2026-03-01", "2026-03-31")
	if err != nil || len(daily.Days) != 3 || daily.StepUnits != 18 || daily.UnitsDone != 6 || daily.PlannedUnits != 10 {
		t.Fatalf("daily: %+v err=%v", daily, err)
	}
	sum, err := uc.OrderSummary(ctx, "2026-03-01", "2026-03-31")
	if err != nil || len(sum.Orders) != 1 || sum.Planned != 10 || sum.Completed != 6 || sum.CompletePct != 60 {
		t.Fatalf("summary: %+v err=%v", sum, err)
	}
	proc, err := uc.ProcessSummary(ctx, "2026-03-01", "2026-03-31")
	if err != nil || len(proc.Rows) != 2 || proc.Rows[0].Process != "Potong" || proc.Rows[0].Units != 10 {
		t.Fatalf("process: %+v err=%v", proc, err)
	}
	wip, err := uc.WIPReport(ctx)
	if err != nil || len(wip.Orders) != 1 || wip.Orders[0].UnitsInWIP != 4 || wip.Orders[0].IssuedValue != 12000 || wip.Orders[0].WIPValue != 0 || wip.LedgerBalance == nil || *wip.LedgerBalance != 0 {
		t.Fatalf("wip: %+v err=%v", wip, err)
	}
	// Re-reading the order keeps the steps in sequence.
	reloaded, _ := uc.GetOrderByID(ctx, order.ID)
	if len(reloaded.Steps) != 2 || reloaded.Steps[0].Name != "Potong" || reloaded.Steps[1].QuantityDone != 8 {
		t.Fatalf("reloaded: %+v", reloaded)
	}
}
