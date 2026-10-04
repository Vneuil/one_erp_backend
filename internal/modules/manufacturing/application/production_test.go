package application

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	inventoryDomain "github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	"github.com/divinecoid/one-backend/internal/modules/manufacturing/domain"
	productDomain "github.com/divinecoid/one-backend/internal/modules/product/domain"
	"github.com/google/uuid"
)

type memMfg struct {
	domain.ManufacturingRepository
	inv     *memInv
	boms    map[uuid.UUID]*domain.BillOfMaterial
	orders  []*domain.ProductionOrder
	logs    []domain.ProductionStepLog
	batches []domain.ProductionBatch
}

func (m *memMfg) CreateBOM(_ context.Context, b *domain.BillOfMaterial) error {
	b.ID = uuid.New()
	for i := range b.Processes {
		b.Processes[i].ID = uuid.New()
	}
	m.boms[b.ID] = b
	return nil
}
func (m *memMfg) GetBOMByID(_ context.Context, id uuid.UUID) (*domain.BillOfMaterial, error) {
	return m.boms[id], nil
}
func (m *memMfg) CreateOrder(_ context.Context, o *domain.ProductionOrder) error {
	o.ID = uuid.New()
	o.CreatedAt = time.Now()
	for i := range o.Steps {
		o.Steps[i].ID, o.Steps[i].OrderID = uuid.New(), o.ID
	}
	m.orders = append(m.orders, o)
	return nil
}
func (m *memMfg) GetOrderByID(_ context.Context, id uuid.UUID) (*domain.ProductionOrder, error) {
	for _, o := range m.orders {
		if o.ID == id {
			return o, nil
		}
	}
	return nil, nil
}
func (m *memMfg) UpdateOrder(context.Context, *domain.ProductionOrder) error { return nil }
func (m *memMfg) UpdateStep(context.Context, *domain.ProductionStep) error   { return nil }
func (m *memMfg) CreateStepLog(_ context.Context, l *domain.ProductionStepLog) error {
	m.logs = append(m.logs, *l)
	return nil
}
func (m *memMfg) CreateBatch(_ context.Context, b *domain.ProductionBatch) error {
	b.ID = uuid.New()
	m.batches = append(m.batches, *b)
	return nil
}
func (m *memMfg) ListAllOrders(context.Context) ([]domain.ProductionOrder, error) {
	out := make([]domain.ProductionOrder, len(m.orders))
	for i, o := range m.orders {
		out[i] = *o
	}
	return out, nil
}
func (m *memMfg) ListStepLogsAll() ([]domain.ProductionStepLog, error) { return m.logs, nil }
func (m *memMfg) CountOrdersByPrefix(_ context.Context, prefix string) (int64, error) {
	var n int64
	for _, o := range m.orders {
		if strings.HasPrefix(o.OrderNumber, prefix) {
			n++
		}
	}
	return n, nil
}
func (m *memMfg) WithTransaction(_ context.Context, fn func(domain.ManufacturingRepository) error) error {
	return fn(m)
}
func (m *memMfg) WithTransactionAndInventory(_ context.Context, fn func(domain.ManufacturingRepository, inventoryDomain.InventoryRepository) error) error {
	return fn(m, m.inv)
}

type memInv struct {
	inventoryDomain.InventoryRepository
	levels map[[2]uuid.UUID]*inventoryDomain.StockLevel
	moves  []inventoryDomain.StockMovement
	issued map[uuid.UUID]map[uuid.UUID]int
}

func (m *memInv) GetStockLevel(_ context.Context, p, w uuid.UUID) (*inventoryDomain.StockLevel, error) {
	return m.levels[[2]uuid.UUID{p, w}], nil
}
func (m *memInv) UpsertStockLevel(_ context.Context, s *inventoryDomain.StockLevel) error {
	m.levels[[2]uuid.UUID{s.ProductID, s.WarehouseID}] = s
	return nil
}
func (m *memInv) CreateMovement(_ context.Context, mv *inventoryDomain.StockMovement) error {
	m.moves = append(m.moves, *mv)
	return nil
}
func (m *memInv) GetWarehouseByID(context.Context, uuid.UUID) (*inventoryDomain.Warehouse, error) {
	return &inventoryDomain.Warehouse{Name: "Pabrik"}, nil
}
func (m *memInv) SumIssuedByOrder(_ context.Context, id uuid.UUID) (map[uuid.UUID]int, error) {
	return m.issued[id], nil
}

type memProducts struct {
	productDomain.ProductRepository
	byID map[uuid.UUID]*productDomain.Product
}

func (m memProducts) GetByID(_ context.Context, id uuid.UUID) (*productDomain.Product, error) {
	return m.byID[id], nil
}

type memLedger struct {
	financeApp.LedgerPoster
	entries map[string][]financeApp.LedgerLine
}

func (l *memLedger) PostEntryOn(_ context.Context, _, src, _ string, lines []financeApp.LedgerLine) error {
	l.entries[src] = lines
	return nil
}

type fixture struct {
	uc           *manufacturingUseCase
	repo         *memMfg
	inv          *memInv
	ledger       *memLedger
	wh           uuid.UUID
	fg, rm1, rm2 uuid.UUID
	bom          *domain.BillOfMaterial
}

// newFixture has a finished good made of 2 x RM1 (cost 1,000) and 1 x RM2 (cost 500),
// i.e. 2,500 of material per unit, routed through Potong -> Jahit -> QC.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	fg, rm1, rm2, wh := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	inv := &memInv{levels: map[[2]uuid.UUID]*inventoryDomain.StockLevel{}, issued: map[uuid.UUID]map[uuid.UUID]int{}}
	for _, p := range []uuid.UUID{rm1, rm2} {
		inv.levels[[2]uuid.UUID{p, wh}] = &inventoryDomain.StockLevel{ProductID: p, WarehouseID: wh, Quantity: 1000}
	}
	repo := &memMfg{inv: inv, boms: map[uuid.UUID]*domain.BillOfMaterial{}}
	led := &memLedger{entries: map[string][]financeApp.LedgerLine{}}
	uc := &manufacturingUseCase{repo: repo, inventoryRepo: inv, ledger: led, productRepo: memProducts{byID: map[uuid.UUID]*productDomain.Product{
		fg: {SKU: "FG", Name: "Kemeja"}, rm1: {SKU: "R1", Name: "Kain", CostPrice: 1000}, rm2: {SKU: "R2", Name: "Kancing", CostPrice: 500}}}}
	f := &fixture{uc: uc, repo: repo, inv: inv, ledger: led, wh: wh, fg: fg, rm1: rm1, rm2: rm2}
	b, err := uc.CreateBOM(context.Background(), CreateBOMDTO{ProductID: fg, Name: "Kemeja", Lines: []BOMLineDTO{{ComponentProductID: rm1, QuantityRequired: 2}, {ComponentProductID: rm2, QuantityRequired: 1}},
		Processes: []BOMProcessDTO{{Name: "Potong", StandardMinutes: 5}, {Name: "Jahit", StandardMinutes: 20}, {Name: "QC", StandardMinutes: 2}}})
	if err != nil {
		t.Fatal(err)
	}
	f.bom = repo.boms[b.ID]
	return f
}

func (f *fixture) newOrder(t *testing.T, qty int, mode string) *domain.ProductionOrder {
	t.Helper()
	dto, err := f.uc.CreateOrder(context.Background(), CreateProductionOrderDTO{BOMID: f.bom.ID, WarehouseID: f.wh, QuantityToProduce: qty, PlannedDate: "2026-03-10", MaterialMode: mode})
	if err != nil {
		t.Fatal(err)
	}
	o, _ := f.repo.GetOrderByID(context.Background(), dto.ID)
	return o
}

func (f *fixture) release(o *domain.ProductionOrder) { o.Status = domain.OrderReleased }

func (f *fixture) log(t *testing.T, o *domain.ProductionOrder, step, qty int) error {
	t.Helper()
	_, err := f.uc.LogStep(context.Background(), o.ID, o.Steps[step].ID, LogStepDTO{Quantity: qty})
	return err
}

func TestBOMProcessesAndOrderNumberingSnapshotRouting(t *testing.T) {
	f := newFixture(t)
	if len(f.bom.Processes) != 3 || f.bom.Processes[1].Sequence != 2 || f.bom.Processes[1].Name != "Jahit" {
		t.Fatalf("processes: %+v", f.bom.Processes)
	}
	o1, o2 := f.newOrder(t, 10, ""), f.newOrder(t, 5, domain.MaterialIssued)
	prefix := "PRD-" + time.Now().Format("200601") + "-"
	if o1.OrderNumber != prefix+"0001" || o2.OrderNumber != prefix+"0002" || o1.MaterialMode != domain.MaterialBackflush || o2.MaterialMode != domain.MaterialIssued {
		t.Fatalf("orders: %s/%s modes %s/%s", o1.OrderNumber, o2.OrderNumber, o1.MaterialMode, o2.MaterialMode)
	}
	if len(o1.Steps) != 3 || o1.Steps[2].Name != "QC" || o1.Steps[1].StandardMinutes != 20 || o1.Steps[0].Status != domain.StepPending {
		t.Fatalf("steps: %+v", o1.Steps)
	}
	f.bom.Processes = f.bom.Processes[:1] // later routing edits must not touch a running order
	if len(o1.Steps) != 3 {
		t.Fatal("an order keeps the routing it was created with")
	}
	ctx := context.Background()
	for name, dto := range map[string]CreateBOMDTO{
		"blank step name": {ProductID: f.fg, Name: "x", Lines: []BOMLineDTO{{ComponentProductID: f.rm1, QuantityRequired: 1}}, Processes: []BOMProcessDTO{{Name: " "}}},
		"negative time":   {ProductID: f.fg, Name: "x", Lines: []BOMLineDTO{{ComponentProductID: f.rm1, QuantityRequired: 1}}, Processes: []BOMProcessDTO{{Name: "A", StandardMinutes: -1}}},
	} {
		if _, err := f.uc.CreateBOM(ctx, dto); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
	if _, err := f.uc.CreateOrder(ctx, CreateProductionOrderDTO{BOMID: f.bom.ID, WarehouseID: f.wh, QuantityToProduce: 1, MaterialMode: "weird"}); err == nil {
		t.Fatal("unknown material mode must be rejected")
	}
	if _, err := f.uc.CreateOrder(ctx, CreateProductionOrderDTO{BOMID: f.bom.ID, WarehouseID: f.wh, QuantityToProduce: 1, PlannedDate: "10/03/2026"}); err == nil {
		t.Fatal("a malformed planned date must be rejected")
	}
}

func TestLogStepEnforcesTheRouting(t *testing.T) {
	f := newFixture(t)
	o := f.newOrder(t, 10, "")
	if err := f.log(t, o, 0, 5); err == nil {
		t.Fatal("a planned (unreleased) order cannot take process output")
	}
	f.release(o)
	if err := f.log(t, o, 1, 1); err == nil || !strings.Contains(err.Error(), "limited by what step 1 (Potong)") {
		t.Fatalf("step 2 cannot run ahead of step 1: %v", err)
	}
	if err := f.log(t, o, 0, 11); err == nil {
		t.Fatal("step 1 is limited by the order quantity")
	}
	if err := f.log(t, o, 0, 6); err != nil {
		t.Fatal(err)
	}
	if o.Status != domain.OrderInProgress || o.Steps[0].Status != domain.StepInProgress || o.Steps[0].StartedDate == "" {
		t.Fatalf("after the first log: %+v status %s", o.Steps[0], o.Status)
	}
	if err := f.log(t, o, 1, 6); err != nil {
		t.Fatal(err)
	}
	if err := f.log(t, o, 1, 1); err == nil {
		t.Fatal("step 2 has now used everything step 1 passed")
	}
	if err := f.log(t, o, 0, 4); err != nil || o.Steps[0].Status != domain.StepDone || o.Steps[0].CompletedDate == "" {
		t.Fatalf("finishing step 1: %v %+v", err, o.Steps[0])
	}
	if len(f.repo.logs) != 3 {
		t.Fatalf("every accepted log is recorded: %d", len(f.repo.logs))
	}
	for name, dto := range map[string]LogStepDTO{"zero": {Quantity: 0}, "future": {Quantity: 1, Date: time.Now().AddDate(0, 0, 2).Format("2006-01-02")}, "bad date": {Quantity: 1, Date: "x"}} {
		if _, err := f.uc.LogStep(context.Background(), o.ID, o.Steps[2].ID, dto); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
	o.Status = domain.OrderPaused
	if err := f.log(t, o, 0, 1); err == nil {
		t.Fatal("a paused order takes no output")
	}
	dto := f.uc.toOrderResponse(context.Background(), o)
	if dto.Steps[1].WaitingQuantity != 4 || dto.Steps[2].WaitingQuantity != 6 || dto.OrderNumber != o.OrderNumber {
		t.Fatalf("waiting quantities: %+v", dto.Steps)
	}
}

func TestCompleteBatchOnlyForUnitsThatPassedEveryStep(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	o := f.newOrder(t, 10, "")
	f.release(o)
	if _, err := f.uc.CompleteBatch(ctx, o.ID, CompleteBatchDTO{QuantityCompleted: 1}); err == nil || !strings.Contains(err.Error(), "Only 0 units have passed") {
		t.Fatalf("nothing has passed the routing yet: %v", err)
	}
	for step, qty := range []int{10, 8, 6} {
		if err := f.log(t, o, step, qty); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.uc.CompleteBatch(ctx, o.ID, CompleteBatchDTO{QuantityCompleted: 7}); err == nil {
		t.Fatal("only 6 units have passed the last step")
	}
	if _, err := f.uc.CompleteBatch(ctx, o.ID, CompleteBatchDTO{QuantityCompleted: 4}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.CompleteBatch(ctx, o.ID, CompleteBatchDTO{QuantityCompleted: 3}); err == nil {
		t.Fatal("2 units are left that have passed the routing")
	}
	// Backflush: materials left stock at completion (4 units: 8 of RM1, 4 of RM2) and finished goods came in.
	if got := f.inv.levels[[2]uuid.UUID{f.rm1, f.wh}].Quantity; got != 992 {
		t.Fatalf("RM1: %d", got)
	}
	if got := f.inv.levels[[2]uuid.UUID{f.fg, f.wh}].Quantity; got != 4 {
		t.Fatalf("finished goods: %d", got)
	}
	if len(f.ledger.entries) != 0 {
		t.Fatalf("a backflush order books nothing (materials and goods are both inventory): %v", f.ledger.entries)
	}
}

func TestIssuedModeUsesIssuedMaterialsAndBooksOutOfWIP(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	o := f.newOrder(t, 10, domain.MaterialIssued)
	f.release(o)
	for step := range o.Steps {
		if err := f.log(t, o, step, 10); err != nil {
			t.Fatal(err)
		}
	}
	f.inv.issued[o.ID] = map[uuid.UUID]int{f.rm1: 10, f.rm2: 5} // enough for 5 units only
	if _, err := f.uc.CompleteBatch(ctx, o.ID, CompleteBatchDTO{QuantityCompleted: 6}); err == nil || !strings.Contains(err.Error(), "do not cover 6 completed units") {
		t.Fatalf("issued materials cover 5 units, not 6: %v", err)
	}
	if _, err := f.uc.CompleteBatch(ctx, o.ID, CompleteBatchDTO{QuantityCompleted: 5}); err != nil {
		t.Fatal(err)
	}
	if got := f.inv.levels[[2]uuid.UUID{f.rm1, f.wh}].Quantity; got != 1000 {
		t.Fatalf("issued mode consumes no materials at completion, RM1 is %d", got)
	}
	if got := f.inv.levels[[2]uuid.UUID{f.fg, f.wh}].Quantity; got != 5 {
		t.Fatalf("finished goods: %d", got)
	}
	var entry []financeApp.LedgerLine
	for _, e := range f.ledger.entries {
		entry = e
	}
	// 5 units x (2 x 1,000 + 1 x 500) = 12,500 out of WIP into inventory.
	if len(f.ledger.entries) != 1 || entry[0].AccountCode != financeApp.AccountInventory || entry[0].Debit != 12_500 || entry[1].AccountCode != financeApp.AccountWIP || entry[1].Credit != 12_500 {
		t.Fatalf("journal: %v", f.ledger.entries)
	}
	// The second batch must account for what is already completed: 8 units need 16 / 8, only 10 / 5 issued.
	if _, err := f.uc.CompleteBatch(ctx, o.ID, CompleteBatchDTO{QuantityCompleted: 3}); err == nil {
		t.Fatal("cumulative requirement must be checked")
	}
}

func TestReportsDailyOrderSummaryProcessAndWIP(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	a := f.newOrder(t, 10, "")                   // backflush, planned 2026-03-10
	b := f.newOrder(t, 4, domain.MaterialIssued) // issued
	f.release(a)
	f.release(b)
	logOn := func(o *domain.ProductionOrder, step, qty int, date string) {
		if _, err := f.uc.LogStep(ctx, o.ID, o.Steps[step].ID, LogStepDTO{Quantity: qty, Date: date}); err != nil {
			t.Fatal(err)
		}
	}
	logOn(a, 0, 10, "2026-03-09")
	logOn(a, 1, 6, "2026-03-10")
	logOn(b, 0, 4, "2026-03-10")
	logOn(b, 1, 4, "2026-03-11")
	logOn(b, 2, 4, "2026-03-11")
	f.inv.issued[b.ID] = map[uuid.UUID]int{f.rm1: 8, f.rm2: 4} // material for 4 units
	if _, err := f.uc.CompleteBatch(ctx, b.ID, CompleteBatchDTO{QuantityCompleted: 1, CompletionDate: "2026-03-12"}); err != nil {
		t.Fatal(err)
	}
	// Pretend the rest of b finished too, two days after its planned date, for the summary below.
	b.ActualCompletionDate, b.Status, b.QuantityCompleted = "2026-03-12", domain.OrderCompleted, 4

	orders, boms, err := f.uc.loadReportOrders(ctx)
	if err != nil || len(orders) != 2 || len(boms) != 1 {
		t.Fatalf("load: %d orders err=%v", len(orders), err)
	}
	logs, _ := f.repo.ListStepLogsAll()
	daily := buildDailyReport("2026-03-01", "2026-03-31", orders, logs, f.repo.batches)
	if len(daily.Days) != 4 || daily.PlannedUnits != 14 || daily.StepUnits != 28 || daily.UnitsDone != 1 {
		t.Fatalf("daily: %+v", daily)
	}
	var d10 DailyReportDay
	for _, d := range daily.Days {
		if d.Date == "2026-03-10" {
			d10 = d
		}
	}
	if len(d10.PlannedOrders) != 2 || d10.StepUnits != 10 || d10.PlannedUnits != 14 {
		t.Fatalf("10 March: %+v", d10)
	}

	sum := buildOrderSummary("2026-03-01", "2026-03-31", orders)
	if len(sum.Orders) != 2 || sum.Planned != 14 || sum.ByStatus[domain.OrderCompleted] != 1 || sum.ByStatus[domain.OrderInProgress] != 1 || len(sum.ByProduct) != 1 {
		t.Fatalf("summary: %+v", sum)
	}
	if sum.OnTimePct != 0 || sum.CompletedSet != 1 {
		t.Fatalf("the completed order finished 2 days after its planned date: %+v", sum)
	}
	for _, r := range sum.Orders {
		if r.Number == b.OrderNumber && r.LateDays != 2 {
			t.Fatalf("late days: %+v", r)
		}
	}

	proc := buildProcessSummary("2026-03-01", "2026-03-31", orders, logs)
	byName := map[string]ProcessSummaryRow{}
	for _, r := range proc.Rows {
		byName[r.Process] = r
	}
	// Potong: 10 + 4 units over 2 days; Jahit: 6 + 4 units (std 20 min) = 200 minutes; order a still has 4 waiting at Jahit.
	if byName["Potong"].Units != 14 || byName["Potong"].ActiveDays != 2 || byName["Potong"].AvgPerDay != 7 || byName["Jahit"].StandardMinutes != 200 || byName["Jahit"].Waiting != 4 || byName["QC"].Units != 4 {
		t.Fatalf("processes: %+v", byName)
	}

	cost := map[uuid.UUID]float64{f.bom.ID: 2500}
	a.Status = domain.OrderInProgress
	b.Status, b.QuantityCompleted = domain.OrderInProgress, 1
	orders, _, _ = f.uc.loadReportOrders(ctx)
	wip := buildWIPReport(orders, wipInputs{costPerUnit: cost, issued: map[uuid.UUID]float64{b.ID: 10_000}})
	rows := map[string]WIPOrderRow{}
	for _, r := range wip.Orders {
		rows[r.Number] = r
	}
	// a (backflush): 10 units past step 1, none finished => 10 x 2,500. b (issued): 10,000 issued - 1 finished unit x 2,500.
	if rows[a.OrderNumber].UnitsInWIP != 10 || rows[a.OrderNumber].WIPValue != 25_000 || rows[b.OrderNumber].UnitsInWIP != 3 || rows[b.OrderNumber].WIPValue != 7_500 || wip.TotalValue != 32_500 || wip.TotalUnits != 13 {
		t.Fatalf("wip: %+v", wip)
	}
	if len(rows[a.OrderNumber].Steps) != 3 || rows[a.OrderNumber].Steps[1].Waiting != 4 {
		t.Fatalf("wip steps: %+v", rows[a.OrderNumber].Steps)
	}
	a.Status = domain.OrderCompleted
	orders, _, _ = f.uc.loadReportOrders(ctx)
	if wip := buildWIPReport(orders, wipInputs{costPerUnit: cost}); len(wip.Orders) != 1 {
		t.Fatalf("finished orders are not WIP: %+v", wip.Orders)
	}
}

func TestReportRange(t *testing.T) {
	if f, to, err := reportRange("", ""); err != nil || f != reportToday()[:8]+"01" || to != reportToday() {
		t.Fatalf("default: %s %s %v", f, to, err)
	}
	for _, bad := range [][2]string{{"2026-04-01", "2026-03-01"}, {"x", ""}} {
		if _, _, err := reportRange(bad[0], bad[1]); err == nil {
			t.Fatalf("%v must fail", bad)
		}
	}
	_ = fmt.Sprint
}
