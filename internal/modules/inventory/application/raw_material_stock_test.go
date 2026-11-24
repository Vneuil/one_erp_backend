package application

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	"github.com/google/uuid"
)

type sbbMemRepo struct {
	*memRepo
	entries map[uuid.UUID]*domain.RawMaterialStockEntry
	logs    []domain.RawMaterialStockLog
}

func newSBBMemRepo() *sbbMemRepo {
	return &sbbMemRepo{
		memRepo: newMem(),
		entries: map[uuid.UUID]*domain.RawMaterialStockEntry{},
	}
}

func (r *sbbMemRepo) WithTransaction(_ context.Context, fn func(domain.InventoryRepository) error) error {
	return fn(r)
}

func (r *sbbMemRepo) ListMovementsByReference(_ context.Context, ref string) ([]domain.StockMovement, error) {
	var out []domain.StockMovement
	for _, m := range r.moves {
		if m.Reference == ref {
			out = append(out, m)
		}
	}
	return out, nil
}

func (r *sbbMemRepo) CreateRawMaterialStockEntry(_ context.Context, e *domain.RawMaterialStockEntry) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	cp := *e
	r.entries[cp.ID] = &cp
	return nil
}

func (r *sbbMemRepo) GetRawMaterialStockEntry(_ context.Context, id uuid.UUID) (*domain.RawMaterialStockEntry, error) {
	e, ok := r.entries[id]
	if !ok {
		return nil, nil
	}
	cp := *e
	cp.Logs = nil
	for _, l := range r.logs {
		if l.EntryID == id {
			cp.Logs = append(cp.Logs, l)
		}
	}
	return &cp, nil
}

func (r *sbbMemRepo) UpdateRawMaterialStockEntry(_ context.Context, e *domain.RawMaterialStockEntry) error {
	cp := *e
	r.entries[cp.ID] = &cp
	return nil
}

func (r *sbbMemRepo) DeleteRawMaterialStockEntry(_ context.Context, id uuid.UUID) error {
	delete(r.entries, id)
	return nil
}

func (r *sbbMemRepo) ListRawMaterialStockEntries(_ context.Context, f domain.RawMaterialStockFilter) ([]domain.RawMaterialStockEntry, error) {
	var out []domain.RawMaterialStockEntry
	for _, e := range r.entries {
		if f.Status != "" && f.Status != "all" && e.Status != f.Status {
			continue
		}
		if f.WarehouseID != nil && e.WarehouseID != *f.WarehouseID {
			continue
		}
		if f.GoodsReceiptID != nil && (e.GoodsReceiptID == nil || *e.GoodsReceiptID != *f.GoodsReceiptID) {
			continue
		}
		out = append(out, *e)
	}
	return out, nil
}

func (r *sbbMemRepo) ListAvailableRawMaterialLots(_ context.Context, productID, warehouseID *uuid.UUID) ([]domain.RawMaterialStockEntry, error) {
	var out []domain.RawMaterialStockEntry
	for _, e := range r.entries {
		if e.Status != domain.RawStockStatusAvailable || e.QtyRemaining <= 0 {
			continue
		}
		if productID != nil && e.ProductID != *productID {
			continue
		}
		if warehouseID != nil && e.WarehouseID != *warehouseID {
			continue
		}
		out = append(out, *e)
	}
	sort.Slice(out, func(i, j int) bool {
		ei, ej := out[i].ExpiryDate, out[j].ExpiryDate
		if ei == "" && ej != "" {
			return false
		}
		if ei != "" && ej == "" {
			return true
		}
		if ei != ej {
			return ei < ej
		}
		return out[i].EntryDate < out[j].EntryDate
	})
	return out, nil
}

func (r *sbbMemRepo) CountRawMaterialStockEntriesByPrefix(_ context.Context, prefix string) (int64, error) {
	var n int64
	for _, e := range r.entries {
		if strings.HasPrefix(e.EntryNo, prefix) {
			n++
		}
	}
	return n, nil
}

func (r *sbbMemRepo) CreateRawMaterialStockLog(_ context.Context, l *domain.RawMaterialStockLog) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	r.logs = append(r.logs, *l)
	return nil
}

func (r *sbbMemRepo) ListRawMaterialStockLogs(_ context.Context, entryID uuid.UUID) ([]domain.RawMaterialStockLog, error) {
	var out []domain.RawMaterialStockLog
	for _, l := range r.logs {
		if l.EntryID == entryID {
			out = append(out, l)
		}
	}
	return out, nil
}

type fakeInboundReader struct {
	data map[uuid.UUID]*LBMInboundRaw
}

func (f *fakeInboundReader) ReadLBMContext(_ context.Context, id uuid.UUID) (*LBMInboundRaw, error) {
	return f.data[id], nil
}

func TestRawMaterialStock_QCGateAndAntiDoubleCounting(t *testing.T) {
	ctx := context.Background()
	repo := newSBBMemRepo()
	prodID := uuid.New()
	whID := uuid.New()
	grID := uuid.New()
	moisture := 1.8

	// Simulate LBM already confirmed & booked 500 kg into StockLevel and StockBatch
	repo.levels[[2]uuid.UUID{prodID, whID}] = &domain.StockLevel{ProductID: prodID, WarehouseID: whID, Quantity: 500}
	repo.batches = append(repo.batches, &domain.StockBatch{
		ID: uuid.New(), ProductID: prodID, WarehouseID: whID, BatchNo: "LOT-2611-01", InitialQty: 500, Quantity: 500,
	})
	repo.moves = append(repo.moves, domain.StockMovement{
		ProductID: prodID, WarehouseID: whID, Type: domain.MovementIn, Quantity: 500, Balance: 500, Reference: "GR-202611-0001", BatchNo: "LOT-2611-01",
	})

	inbound := &fakeInboundReader{
		data: map[uuid.UUID]*LBMInboundRaw{
			grID: {
				GoodsReceiptID: grID,
				LBMNo:          "GR-202611-0001",
				PONo:           "PO-202611-0001",
				ReceivedDate:   "2026-11-18",
				WarehouseID:    &whID,
				SupplierName:   "Koperasi Gula Organik Kulon Progo",
				CAS:            &QCCASSummaryDTO{ID: uuid.New(), OverallStatus: "cleared", DriverName: "Budi"},
				PBD: &QCPBDSummaryDTO{
					ID: uuid.New(), OverallStatus: "passed", AcceptedWeight: 490, RejectedWeight: 10,
					MoistureContent: &moisture, MeshSize: "Mesh 18", RecommendedAction: "Bypass Oven",
				},
				PBK: &QCPBKSummaryDTO{ID: uuid.New(), OverallStatus: "passed"},
			},
		},
	}

	uc := &inventoryUseCase{
		repo: repo,
		productRepo: productRepoMap{byID: products{
			prodID: {SKU: "BB-GS-01", Name: "Gula Kelapa Semut Organik", Unit: "KG", OrganicType: "Organik", Certifications: "EU, USDA, Halal, Kosher"},
		}},
		inboundReader: inbound,
	}

	// 1. Verify GetLBMInboundContext pulls all 3 QC statuses and recommends available
	lbmCtx, err := uc.GetLBMInboundContext(ctx, grID)
	if err != nil {
		t.Fatalf("GetLBMInboundContext failed: %v", err)
	}
	if !lbmCtx.QCGateReady || lbmCtx.RecommendedStatus != domain.RawStockStatusAvailable {
		t.Fatalf("expected QCGateReady=true and RecommendedStatus=available, got ready=%v rec=%s", lbmCtx.QCGateReady, lbmCtx.RecommendedStatus)
	}

	// 2. Create & immediately post SBB from LBM (490 accepted, 10 rejected)
	created, err := uc.CreateRawMaterialStockEntry(ctx, CreateRawMaterialStockDTO{
		EntryDate:       "2026-11-18",
		SourceType:      domain.RawStockSourceLBM,
		GoodsReceiptID:  &grID,
		ProductID:       prodID,
		WarehouseID:     whID,
		BatchNo:         "LOT-2611-01",
		QtyLBM:          500,
		QtyReceived:     490,
		QtyRejected:     10,
		TotalPackages:   20,
		StorageZone:     "Zona Organik",
		RackLocation:    "RAK-A1",
		PalletNo:        "PLT-001",
		ExpiryDate:      "2027-11-18",
		PostImmediately: true,
	})
	if err != nil {
		t.Fatalf("CreateRawMaterialStockEntry failed: %v", err)
	}
	if created.EntryNo != "SBB-001-1126" {
		t.Errorf("want EntryNo SBB-001-1126, got %s", created.EntryNo)
	}
	if created.Status != domain.RawStockStatusAvailable {
		t.Errorf("want status available, got %s", created.Status)
	}
	if created.StockAddedBySBB {
		t.Errorf("expected StockAddedBySBB=false because LBM already booked stock")
	}

	// Anti Double-Counting assertion:
	// StockLevel was 500 (from LBM). Posting SBB with 490 accepted + 10 rejected MUST result in 490 (NOT 990!).
	lvl := repo.levels[[2]uuid.UUID{prodID, whID}]
	if lvl.Quantity != 490 {
		t.Fatalf("anti double-counting failed: expected stock level 490, got %d", lvl.Quantity)
	}
	// Batch expiry must be enriched from SBB
	if repo.batches[0].ExpiryDate != "2027-11-18" || repo.batches[0].Quantity != 490 {
		t.Fatalf("expected batch expiry 2027-11-18 and qty 490, got expiry=%s qty=%d", repo.batches[0].ExpiryDate, repo.batches[0].Quantity)
	}
	if len(created.Logs) != 1 || created.Logs[0].EventType != domain.RawStockEventPutaway {
		t.Errorf("expected 1 putaway log, got %+v", created.Logs)
	}

	// 3. Duplicate guard: trying to create another SBB for the same LBM + Product + Lot must be blocked
	if _, err := uc.CreateRawMaterialStockEntry(ctx, CreateRawMaterialStockDTO{
		EntryDate:      "2026-11-18",
		SourceType:     domain.RawStockSourceLBM,
		GoodsReceiptID: &grID,
		ProductID:      prodID,
		WarehouseID:    whID,
		BatchNo:        "LOT-2611-01",
		QtyReceived:    490,
	}); err == nil {
		t.Fatalf("expected conflict error when creating duplicate SBB for same LBM and BatchNo")
	}
}

func TestRawMaterialStock_SmartQCGateBlocksMissingOrRejected(t *testing.T) {
	ctx := context.Background()
	repo := newSBBMemRepo()
	prodID := uuid.New()
	whID := uuid.New()
	grMissingID := uuid.New()
	grRejectedID := uuid.New()

	inbound := &fakeInboundReader{
		data: map[uuid.UUID]*LBMInboundRaw{
			grMissingID: {
				GoodsReceiptID: grMissingID, LBMNo: "GR-202611-0002", WarehouseID: &whID,
				// CAS & PBD missing
			},
			grRejectedID: {
				GoodsReceiptID: grRejectedID, LBMNo: "GR-202611-0003", WarehouseID: &whID,
				CAS: &QCCASSummaryDTO{ID: uuid.New(), OverallStatus: "cleared"},
				PBD: &QCPBDSummaryDTO{ID: uuid.New(), OverallStatus: "rejected"},
			},
		},
	}
	uc := &inventoryUseCase{repo: repo, productRepo: productRepoMap{byID: products{}}, inboundReader: inbound}

	// Missing CAS/PBD -> trying to post directly as available MUST fail
	_, err := uc.CreateRawMaterialStockEntry(ctx, CreateRawMaterialStockDTO{
		EntryDate: "2026-11-18", SourceType: domain.RawStockSourceLBM, GoodsReceiptID: &grMissingID,
		ProductID: prodID, WarehouseID: whID, BatchNo: "LOT-MISS", QtyReceived: 100,
		PostImmediately: true, TargetStatus: domain.RawStockStatusAvailable,
	})
	if err == nil {
		t.Fatalf("expected Smart QC Gate error when posting available with missing CAS/PBD")
	}

	// Rejected PBD -> trying to post as available MUST fail, posting as quarantine succeeds
	draft, err := uc.CreateRawMaterialStockEntry(ctx, CreateRawMaterialStockDTO{
		EntryDate: "2026-11-18", SourceType: domain.RawStockSourceLBM, GoodsReceiptID: &grRejectedID,
		ProductID: prodID, WarehouseID: whID, BatchNo: "LOT-REJ", QtyReceived: 100,
	})
	if err != nil {
		t.Fatalf("create draft failed: %v", err)
	}
	if _, err := uc.PostRawMaterialStockEntry(ctx, draft.ID, PostRawMaterialStockDTO{TargetStatus: domain.RawStockStatusAvailable}); err == nil {
		t.Fatalf("expected Smart QC Gate error when PBD is rejected")
	}
	qRes, err := uc.PostRawMaterialStockEntry(ctx, draft.ID, PostRawMaterialStockDTO{TargetStatus: domain.RawStockStatusQuarantine})
	if err != nil {
		t.Fatalf("posting rejected PBD to quarantine should succeed: %v", err)
	}
	if qRes.Status != domain.RawStockStatusQuarantine {
		t.Fatalf("want quarantine, got %s", qRes.Status)
	}

	// Release quarantine to available
	rel, err := uc.ReleaseQuarantine(ctx, qRes.ID, ReleaseQuarantineDTO{
		Decision: domain.RawStockStatusAvailable, ApprovedBy: "QA Manager", Notes: "Re-oven & re-test lolos",
	})
	if err != nil {
		t.Fatalf("ReleaseQuarantine failed: %v", err)
	}
	if rel.Status != domain.RawStockStatusAvailable {
		t.Fatalf("want available after release, got %s", rel.Status)
	}
}

func TestRawMaterialStock_OpeningBalanceSyncVoidAndFEFO(t *testing.T) {
	ctx := context.Background()
	repo := newSBBMemRepo()
	prodID := uuid.New()
	whID := uuid.New()
	uc := &inventoryUseCase{repo: repo, productRepo: productRepoMap{byID: products{
		prodID: {SKU: "BB-01", Name: "Gula Semut", Unit: "KG"},
	}}}

	// Opening balance adds stock directly (StockAddedBySBB=true)
	lotLate, err := uc.CreateRawMaterialStockEntry(ctx, CreateRawMaterialStockDTO{
		EntryDate: "2026-11-18", SourceType: domain.RawStockSourceOpeningBalance,
		ProductID: prodID, WarehouseID: whID, BatchNo: "OB-LATE", QtyReceived: 200, TotalPackages: 8,
		ExpiryDate: "2027-12-31", PostImmediately: true,
	})
	if err != nil {
		t.Fatalf("create lotLate failed: %v", err)
	}
	lotSoon, err := uc.CreateRawMaterialStockEntry(ctx, CreateRawMaterialStockDTO{
		EntryDate: "2026-11-18", SourceType: domain.RawStockSourceOpeningBalance,
		ProductID: prodID, WarehouseID: whID, BatchNo: "OB-SOON", QtyReceived: 150, TotalPackages: 6,
		ExpiryDate: "2027-02-01", PostImmediately: true,
	})
	if err != nil {
		t.Fatalf("create lotSoon failed: %v", err)
	}
	if repo.levels[[2]uuid.UUID{prodID, whID}].Quantity != 350 {
		t.Fatalf("expected stock level 350, got %d", repo.levels[[2]uuid.UUID{prodID, whID}].Quantity)
	}

	// FEFO check: OB-SOON must come before OB-LATE
	avail, err := uc.ListAvailableRawMaterialLots(ctx, &prodID, &whID)
	if err != nil || len(avail) != 2 {
		t.Fatalf("expected 2 available lots, got %d (err=%v)", len(avail), err)
	}
	if avail[0].BatchNo != "OB-SOON" || avail[1].BatchNo != "OB-LATE" {
		t.Fatalf("FEFO order wrong: got %s then %s", avail[0].BatchNo, avail[1].BatchNo)
	}

	// Voiding OB-LATE reverses its 200 kg because StockAddedBySBB=true
	voided, err := uc.VoidRawMaterialStockEntry(ctx, lotLate.ID, "Salah input saldo awal")
	if err != nil {
		t.Fatalf("Void failed: %v", err)
	}
	if voided.Status != domain.RawStockStatusVoided || voided.QtyRemaining != 0 {
		t.Fatalf("expected voided with 0 remaining, got status=%s rem=%v", voided.Status, voided.QtyRemaining)
	}
	if repo.levels[[2]uuid.UUID{prodID, whID}].Quantity != 150 {
		t.Fatalf("expected stock level reversed to 150, got %d", repo.levels[[2]uuid.UUID{prodID, whID}].Quantity)
	}

	// Summary check
	sum, err := uc.GetRawMaterialStockSummary(ctx, &whID)
	if err != nil {
		t.Fatalf("GetRawMaterialStockSummary failed: %v", err)
	}
	if sum.AvailableCount != 1 || sum.TotalAvailableQty != 150 || sum.TotalAvailablePackages != 6 {
		t.Fatalf("unexpected summary: %+v", sum)
	}

	// Prevent void when lot already has issued qty
	rawSoon := repo.entries[lotSoon.ID]
	rawSoon.QtyIssued = 50
	if _, err := uc.VoidRawMaterialStockEntry(ctx, lotSoon.ID, "Test block"); err == nil {
		t.Fatalf("expected error when voiding a lot that has already been issued")
	}
}
