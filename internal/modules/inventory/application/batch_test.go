package application

import (
	"context"
	"sort"
	"testing"
	"time"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	productDomain "github.com/divinecoid/one-backend/internal/modules/product/domain"
	"github.com/google/uuid"
)

func TestBatchStatus(t *testing.T) {
	cases := []struct {
		expiry, today, want string
		days                int
	}{
		{"", "2026-09-30", "no_expiry", 0},
		{"2026-09-29", "2026-09-30", "expired", -1},
		{"2026-09-30", "2026-09-30", "expiring_soon", 0},
		{"2026-10-30", "2026-09-30", "expiring_soon", 30},
		{"2026-10-31", "2026-09-30", "ok", 31},
	}
	for _, c := range cases {
		got, days := BatchStatus(c.expiry, c.today)
		if got != c.want || (days != nil && *days != c.days) {
			t.Errorf("%s vs %s: %s %v, want %s %d", c.expiry, c.today, got, days, c.want, c.days)
		}
	}
}

type memRepo struct {
	domain.InventoryRepository
	levels  map[[2]uuid.UUID]*domain.StockLevel
	batches []*domain.StockBatch
	moves   []domain.StockMovement
}

func newMem() *memRepo { return &memRepo{levels: map[[2]uuid.UUID]*domain.StockLevel{}} }

func (m *memRepo) WithTransaction(_ context.Context, fn func(domain.InventoryRepository) error) error {
	return fn(m)
}
func (m *memRepo) GetStockLevel(_ context.Context, p, w uuid.UUID) (*domain.StockLevel, error) {
	return m.levels[[2]uuid.UUID{p, w}], nil
}
func (m *memRepo) HasAnyStockLevel(context.Context, uuid.UUID) (bool, error) { return true, nil }
func (m *memRepo) UpsertStockLevel(_ context.Context, s *domain.StockLevel) error {
	m.levels[[2]uuid.UUID{s.ProductID, s.WarehouseID}] = s
	return nil
}
func (m *memRepo) CreateMovement(_ context.Context, mv *domain.StockMovement) error {
	m.moves = append(m.moves, *mv)
	return nil
}
func (m *memRepo) GetWarehouseByID(_ context.Context, id uuid.UUID) (*domain.Warehouse, error) {
	return &domain.Warehouse{Name: "Gudang"}, nil
}
func (m *memRepo) GetBatch(_ context.Context, id uuid.UUID) (*domain.StockBatch, error) {
	for _, b := range m.batches {
		if b.ID == id {
			c := *b
			return &c, nil
		}
	}
	return nil, nil
}
func (m *memRepo) GetBatchByNo(_ context.Context, p, w uuid.UUID, no string) (*domain.StockBatch, error) {
	for _, b := range m.batches {
		if b.ProductID == p && b.WarehouseID == w && b.BatchNo == no {
			c := *b
			return &c, nil
		}
	}
	return nil, nil
}
func (m *memRepo) SaveBatch(_ context.Context, b *domain.StockBatch) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
		c := *b
		m.batches = append(m.batches, &c)
		return nil
	}
	for i := range m.batches {
		if m.batches[i].ID == b.ID {
			c := *b
			m.batches[i] = &c
		}
	}
	return nil
}
func (m *memRepo) ListBatchesFEFO(_ context.Context, p, w uuid.UUID) ([]domain.StockBatch, error) {
	var out []domain.StockBatch
	for _, b := range m.batches {
		if b.ProductID == p && b.WarehouseID == w && b.Quantity > 0 {
			out = append(out, *b)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i].ExpiryDate, out[j].ExpiryDate
		if (a == "") != (b == "") {
			return b == ""
		}
		return a < b
	})
	return out, nil
}

type prodRepo struct {
	productDomain.ProductRepository
}

func (prodRepo) GetByID(context.Context, uuid.UUID) (*productDomain.Product, error) {
	return &productDomain.Product{SKU: "S", Name: "Susu", CostPrice: 5_000}, nil
}

type recLedger struct {
	financeApp.LedgerPoster
	got map[string][]financeApp.LedgerLine
}

func (r *recLedger) PostEntry(_ context.Context, src, _ string, l []financeApp.LedgerLine) error {
	r.got[src] = l
	return nil
}

func inDays(n int) string { return time.Now().In(batchZone).AddDate(0, 0, n).Format("2006-01-02") }

func setup() (*inventoryUseCase, *memRepo, *recLedger, uuid.UUID, uuid.UUID) {
	m := newMem()
	led := &recLedger{got: map[string][]financeApp.LedgerLine{}}
	uc := &inventoryUseCase{repo: m, productRepo: prodRepo{}, ledger: led}
	return uc, m, led, uuid.New(), uuid.New()
}

func qtyOf(m *memRepo, no string) int {
	for _, b := range m.batches {
		if b.BatchNo == no {
			return b.Quantity
		}
	}
	return -1
}

func TestSalesConsumeSoonestExpiringFirstAndSkipExpiredLots(t *testing.T) {
	uc, m, _, p, w := setup()
	ctx := context.Background()
	for _, r := range []ReceiveBatchDTO{
		{ProductID: p, WarehouseID: w, BatchNo: "LATE", ExpiryDate: inDays(90), Quantity: 10},
		{ProductID: p, WarehouseID: w, BatchNo: "SOON", ExpiryDate: inDays(10), Quantity: 5},
		{ProductID: p, WarehouseID: w, BatchNo: "NOEXP", Quantity: 4},
	} {
		if _, err := uc.ReceiveBatch(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	// A lot that has already expired (received earlier, expired since).
	m.batches = append(m.batches, &domain.StockBatch{ID: uuid.New(), ProductID: p, WarehouseID: w, BatchNo: "OLD", ExpiryDate: inDays(-3), Quantity: 6, InitialQty: 6})
	m.levels[[2]uuid.UUID{p, w}].Quantity += 6

	if got, _ := uc.GetAvailability(ctx, p, w); got != 19 { // 25 on hand minus 6 expired
		t.Fatalf("availability must exclude expired stock, got %d", got)
	}
	if _, err := uc.AdjustStock(ctx, AdjustStockDTO{ProductID: p, WarehouseID: w, Quantity: -8}); err != nil {
		t.Fatal(err)
	}
	if qtyOf(m, "SOON") != 0 || qtyOf(m, "LATE") != 7 || qtyOf(m, "NOEXP") != 4 || qtyOf(m, "OLD") != 6 {
		t.Fatalf("FEFO: SOON=%d LATE=%d NOEXP=%d OLD=%d", qtyOf(m, "SOON"), qtyOf(m, "LATE"), qtyOf(m, "NOEXP"), qtyOf(m, "OLD"))
	}
	if last := m.moves[len(m.moves)-1]; last.BatchNo != "" {
		t.Fatalf("a movement spread over two lots names none: %q", last.BatchNo)
	}
	if _, err := uc.AdjustStock(ctx, AdjustStockDTO{ProductID: p, WarehouseID: w, Quantity: -2}); err != nil {
		t.Fatal(err)
	}
	if last := m.moves[len(m.moves)-1]; last.BatchNo != "LATE" || qtyOf(m, "LATE") != 5 {
		t.Fatalf("single-lot movement: %+v LATE=%d", last, qtyOf(m, "LATE"))
	}
}

func TestReceivingRules(t *testing.T) {
	uc, m, _, p, w := setup()
	ctx := context.Background()
	if _, err := uc.ReceiveBatch(ctx, ReceiveBatchDTO{ProductID: p, WarehouseID: w, BatchNo: "A1", ExpiryDate: inDays(60), Quantity: 10}); err != nil {
		t.Fatal(err)
	}
	// Topping up the same lot adds to it; a different expiry for the same lot number is refused.
	if b, err := uc.ReceiveBatch(ctx, ReceiveBatchDTO{ProductID: p, WarehouseID: w, BatchNo: "A1", Quantity: 5}); err != nil || b.Quantity != 15 || b.InitialQty != 15 {
		t.Fatalf("top-up: %+v %v", b, err)
	}
	if _, err := uc.ReceiveBatch(ctx, ReceiveBatchDTO{ProductID: p, WarehouseID: w, BatchNo: "A1", ExpiryDate: inDays(90), Quantity: 1}); err == nil {
		t.Fatal("one lot cannot have two expiry dates")
	}
	for _, bad := range []ReceiveBatchDTO{
		{ProductID: p, WarehouseID: w, BatchNo: "", Quantity: 1},
		{ProductID: p, WarehouseID: w, BatchNo: "B", Quantity: 0},
		{ProductID: p, WarehouseID: w, BatchNo: "B", Quantity: 1, ExpiryDate: "2020-01-01"},
		{ProductID: p, WarehouseID: w, BatchNo: "B", Quantity: 1, ExpiryDate: "31/12/2030"},
	} {
		if _, err := uc.ReceiveBatch(ctx, bad); err == nil {
			t.Errorf("should be refused: %+v", bad)
		}
	}
	if m.levels[[2]uuid.UUID{p, w}].Quantity != 15 {
		t.Fatalf("stock on hand must match what was received: %d", m.levels[[2]uuid.UUID{p, w}].Quantity)
	}
}

func TestWriteOffRemovesTheLotAndBooksTheLoss(t *testing.T) {
	uc, m, led, p, w := setup()
	ctx := context.Background()
	if _, err := uc.ReceiveBatch(ctx, ReceiveBatchDTO{ProductID: p, WarehouseID: w, BatchNo: "X", Quantity: 8}); err != nil {
		t.Fatal(err)
	}
	// Age the lot: it expired yesterday.
	m.batches[0].ExpiryDate = inDays(-1)
	id := m.batches[0].ID
	if _, err := uc.WriteOffBatch(ctx, id, " "); err == nil {
		t.Fatal("a reason is required")
	}
	b, err := uc.WriteOffBatch(ctx, id, "kedaluwarsa")
	if err != nil {
		t.Fatal(err)
	}
	if b.Quantity != 0 || m.levels[[2]uuid.UUID{p, w}].Quantity != 0 {
		t.Fatalf("lot %d, stock %d", b.Quantity, m.levels[[2]uuid.UUID{p, w}].Quantity)
	}
	if last := m.moves[len(m.moves)-1]; last.BatchNo != "X" || last.Quantity != -8 {
		t.Fatalf("movement: %+v", last)
	}
	if len(led.got) != 1 {
		t.Fatalf("expected one write-off entry, got %v", led.got)
	}
	for _, lines := range led.got {
		if lines[0].Debit != 40_000 || lines[1].Credit != 40_000 {
			t.Fatalf("loss should be 8 x 5,000: %+v", lines)
		}
	}
	if _, err := uc.WriteOffBatch(ctx, id, "lagi"); err == nil {
		t.Fatal("an empty lot cannot be written off again")
	}
	// A specific lot cannot be overdrawn.
	if _, err := uc.ReceiveBatch(ctx, ReceiveBatchDTO{ProductID: p, WarehouseID: w, BatchNo: "Y", Quantity: 3}); err != nil {
		t.Fatal(err)
	}
	yid := m.batches[1].ID
	if _, err := uc.AdjustStock(ctx, AdjustStockDTO{ProductID: p, WarehouseID: w, Quantity: -4, BatchID: &yid}); err == nil {
		t.Fatal("taking more than a lot holds must fail")
	}
}
