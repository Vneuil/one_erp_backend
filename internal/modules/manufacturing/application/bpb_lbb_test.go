package application

import (
	"context"
	"strings"
	"testing"

	inventoryDomain "github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	"github.com/divinecoid/one-backend/internal/modules/manufacturing/domain"
	"github.com/google/uuid"
)

type bpbMockMfgRepo struct {
	domain.ManufacturingRepository
	inv         *bpbMockInvRepo
	reqs        map[uuid.UUID]*domain.MaterialRequisition
	lbbs        map[uuid.UUID]*domain.RawMaterialRequestReport
	lbbEntries  map[uuid.UUID][]domain.RawMaterialRequestBagEntry
	orders      map[uuid.UUID]*domain.ProductionOrder
	genealogies []domain.BatchGenealogy
}

func newBpbMockMfgRepo(inv *bpbMockInvRepo) *bpbMockMfgRepo {
	return &bpbMockMfgRepo{
		inv:        inv,
		reqs:       make(map[uuid.UUID]*domain.MaterialRequisition),
		lbbs:       make(map[uuid.UUID]*domain.RawMaterialRequestReport),
		lbbEntries: make(map[uuid.UUID][]domain.RawMaterialRequestBagEntry),
		orders:     make(map[uuid.UUID]*domain.ProductionOrder),
	}
}

func (m *bpbMockMfgRepo) CountRequisitionsByPrefix(_ context.Context, prefix string) (int64, error) {
	var n int64
	for _, r := range m.reqs {
		if strings.HasPrefix(r.RequestNo, prefix) {
			n++
		}
	}
	return n, nil
}

func (m *bpbMockMfgRepo) CreateRequisition(_ context.Context, req *domain.MaterialRequisition) error {
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}
	for i := range req.Lines {
		if req.Lines[i].ID == uuid.Nil {
			req.Lines[i].ID = uuid.New()
		}
		req.Lines[i].RequisitionID = req.ID
	}
	m.reqs[req.ID] = req
	return nil
}

func (m *bpbMockMfgRepo) GetRequisitionByID(_ context.Context, id uuid.UUID) (*domain.MaterialRequisition, error) {
	req := m.reqs[id]
	if req != nil {
		// Attach lbbs
		var linkedLBBs []domain.RawMaterialRequestReport
		for _, l := range m.lbbs {
			if l.RequisitionID != nil && *l.RequisitionID == id {
				linkedLBBs = append(linkedLBBs, *l)
			}
		}
		req.LBBReports = linkedLBBs
	}
	return req, nil
}

func (m *bpbMockMfgRepo) ListRequisitions(_ context.Context, reqType, status string) ([]domain.MaterialRequisition, error) {
	var out []domain.MaterialRequisition
	for _, r := range m.reqs {
		if reqType != "" && r.RequisitionType != reqType {
			continue
		}
		if status != "" && r.Status != status {
			continue
		}
		out = append(out, *r)
	}
	return out, nil
}

func (m *bpbMockMfgRepo) UpdateRequisition(_ context.Context, req *domain.MaterialRequisition) error {
	m.reqs[req.ID] = req
	return nil
}

func (m *bpbMockMfgRepo) UpdateRequisitionLine(_ context.Context, line *domain.MaterialRequisitionLine) error {
	for _, r := range m.reqs {
		for i := range r.Lines {
			if r.Lines[i].ID == line.ID {
				r.Lines[i] = *line
				return nil
			}
		}
	}
	return nil
}

func (m *bpbMockMfgRepo) UpdateRequisitionStatus(_ context.Context, id uuid.UUID, status string, issuedBy string) error {
	if req, ok := m.reqs[id]; ok {
		req.Status = status
		if issuedBy != "" {
			req.IssuedBy = issuedBy
		}
	}
	return nil
}

func (m *bpbMockMfgRepo) CountLBBReportsByPrefix(_ context.Context, prefix string) (int64, error) {
	var n int64
	for _, l := range m.lbbs {
		if strings.HasPrefix(l.ReportNo, prefix) {
			n++
		}
	}
	return n, nil
}

func (m *bpbMockMfgRepo) CreateLBBReport(_ context.Context, rep *domain.RawMaterialRequestReport) error {
	if rep.ID == uuid.Nil {
		rep.ID = uuid.New()
	}
	m.lbbs[rep.ID] = rep
	return nil
}

func (m *bpbMockMfgRepo) GetLBBReportByID(_ context.Context, id uuid.UUID) (*domain.RawMaterialRequestReport, error) {
	rep := m.lbbs[id]
	if rep != nil {
		rep.BagEntries = m.lbbEntries[id]
	}
	return rep, nil
}

func (m *bpbMockMfgRepo) ListLBBReports(_ context.Context, reqID *uuid.UUID, variantType, status string) ([]domain.RawMaterialRequestReport, error) {
	var out []domain.RawMaterialRequestReport
	for _, l := range m.lbbs {
		if reqID != nil && (l.RequisitionID == nil || *l.RequisitionID != *reqID) {
			continue
		}
		if variantType != "" && l.VariantType != variantType {
			continue
		}
		if status != "" && l.Status != status {
			continue
		}
		item := *l
		item.BagEntries = m.lbbEntries[l.ID]
		out = append(out, item)
	}
	return out, nil
}

func (m *bpbMockMfgRepo) UpdateLBBReport(_ context.Context, rep *domain.RawMaterialRequestReport) error {
	m.lbbs[rep.ID] = rep
	return nil
}

func (m *bpbMockMfgRepo) ReplaceLBBBagEntries(_ context.Context, reportID uuid.UUID, entries []domain.RawMaterialRequestBagEntry) error {
	m.lbbEntries[reportID] = entries
	return nil
}

func (m *bpbMockMfgRepo) DeleteLBBReport(_ context.Context, id uuid.UUID) error {
	delete(m.lbbs, id)
	delete(m.lbbEntries, id)
	return nil
}

func (m *bpbMockMfgRepo) GetBOMByID(_ context.Context, _ uuid.UUID) (*domain.BillOfMaterial, error) {
	return nil, nil
}

func (m *bpbMockMfgRepo) GetOrderByID(_ context.Context, id uuid.UUID) (*domain.ProductionOrder, error) {
	return m.orders[id], nil
}

func (m *bpbMockMfgRepo) UpdateOrder(_ context.Context, order *domain.ProductionOrder) error {
	m.orders[order.ID] = order
	return nil
}

func (m *bpbMockMfgRepo) CreateGenealogy(_ context.Context, g *domain.BatchGenealogy) error {
	m.genealogies = append(m.genealogies, *g)
	return nil
}

func (m *bpbMockMfgRepo) WithTransactionAndInventory(_ context.Context, fn func(domain.ManufacturingRepository, inventoryDomain.InventoryRepository) error) error {
	return fn(m, m.inv)
}

type bpbMockInvRepo struct {
	inventoryDomain.InventoryRepository
	sbbEntries map[uuid.UUID]*inventoryDomain.RawMaterialStockEntry
	stockLogs  []inventoryDomain.RawMaterialStockLog
	stockLvls  map[[2]uuid.UUID]*inventoryDomain.StockLevel
	batches    map[string]*inventoryDomain.StockBatch
	movements  []inventoryDomain.StockMovement
}

func newBpbMockInvRepo() *bpbMockInvRepo {
	return &bpbMockInvRepo{
		sbbEntries: make(map[uuid.UUID]*inventoryDomain.RawMaterialStockEntry),
		stockLvls:  make(map[[2]uuid.UUID]*inventoryDomain.StockLevel),
		batches:    make(map[string]*inventoryDomain.StockBatch),
	}
}

func (m *bpbMockInvRepo) GetWarehouseByID(_ context.Context, id uuid.UUID) (*inventoryDomain.Warehouse, error) {
	return &inventoryDomain.Warehouse{Name: "Gudang Utama"}, nil
}

func (m *bpbMockInvRepo) GetRawMaterialStockEntry(_ context.Context, id uuid.UUID) (*inventoryDomain.RawMaterialStockEntry, error) {
	return m.sbbEntries[id], nil
}

func (m *bpbMockInvRepo) UpdateRawMaterialStockEntry(_ context.Context, e *inventoryDomain.RawMaterialStockEntry) error {
	m.sbbEntries[e.ID] = e
	return nil
}

func (m *bpbMockInvRepo) CreateRawMaterialStockLog(_ context.Context, l *inventoryDomain.RawMaterialStockLog) error {
	m.stockLogs = append(m.stockLogs, *l)
	return nil
}

func (m *bpbMockInvRepo) GetStockLevel(_ context.Context, productID, warehouseID uuid.UUID) (*inventoryDomain.StockLevel, error) {
	return m.stockLvls[[2]uuid.UUID{productID, warehouseID}], nil
}

func (m *bpbMockInvRepo) UpsertStockLevel(_ context.Context, s *inventoryDomain.StockLevel) error {
	m.stockLvls[[2]uuid.UUID{s.ProductID, s.WarehouseID}] = s
	return nil
}

func (m *bpbMockInvRepo) GetBatchByNo(_ context.Context, _, _ uuid.UUID, batchNo string) (*inventoryDomain.StockBatch, error) {
	return m.batches[batchNo], nil
}

func (m *bpbMockInvRepo) SaveBatch(_ context.Context, b *inventoryDomain.StockBatch) error {
	m.batches[b.BatchNo] = b
	return nil
}

func (m *bpbMockInvRepo) CreateMovement(_ context.Context, mv *inventoryDomain.StockMovement) error {
	m.movements = append(m.movements, *mv)
	return nil
}

func TestBPB_LBB_Workflow_EndToEnd(t *testing.T) {
	ctx := context.Background()
	invRepo := newBpbMockInvRepo()
	mfgRepo := newBpbMockMfgRepo(invRepo)

	uc := &manufacturingUseCase{
		repo:          mfgRepo,
		inventoryRepo: invRepo,
	}

	prodID := uuid.New()
	whID := uuid.New()

	// 1. Setup available RawMaterialStockEntry (SBB GDG-002-SBB)
	sbbID := uuid.New()
	moisture := 1.8
	sbb := &inventoryDomain.RawMaterialStockEntry{
		EntryNo:           "SBB-202611-0001",
		EntryDate:         "2026-11-18",
		Status:            inventoryDomain.RawStockStatusAvailable,
		ProductID:         prodID,
		ProductName:       "Gula Semut Kelapa Organik",
		BatchNo:           "18O12KP",
		SupplierName:      "Kelompok Tani Sejahtera",
		WarehouseID:       whID,
		RackLocation:      "A-01",
		PalletNo:          "P-05",
		MoistureContent:   &moisture,
		QtyReceived:       1000.0,
		QtyIssued:         0,
		QtyReserved:       0,
		QtyRemaining:      1000.0,
		TotalPackages:     40,
		RemainingPackages: 40,
		PackageUnit:       "Sak",
		UoM:               "KG",
	}
	sbb.ID = sbbID
	invRepo.sbbEntries[sbbID] = sbb

	// Main warehouse stock level and batch
	invRepo.stockLvls[[2]uuid.UUID{prodID, whID}] = &inventoryDomain.StockLevel{
		ProductID:   prodID,
		WarehouseID: whID,
		Quantity:    1000,
	}
	invRepo.batches["18O12KP"] = &inventoryDomain.StockBatch{
		ProductID:   prodID,
		WarehouseID: whID,
		BatchNo:     "18O12KP",
		Quantity:    1000,
	}

	// 2. Produksi Create BPB (Bon Permintaan Bahan Baku GDG-011-BPB)
	// Try over-reserving (1200 kg > 1000 kg available) -> should fail
	_, err := uc.CreateRequisition(ctx, CreateMaterialRequisitionDTO{
		RequisitionType:  domain.RequisitionTypeRawMaterial,
		DocumentCode:     "GDG-011-BPB",
		MaterialCategory: "GSKN",
		FairtradeStatus:  "Fairtrade",
		RequestDate:      "2026-11-18",
		Department:       "Produksi Ayak",
		RequestedBy:      "Mandor Produksi",
		Lines: []MaterialRequisitionLineDTO{
			{
				RawMaterialStockID:    &sbbID,
				QuantityRequested:     1200.0,
				PackageCountRequested: 48,
			},
		},
	})
	if err == nil {
		t.Fatalf("expected error when requesting more than available SBB stock")
	}

	// Valid requisition (500 kg, 20 sak)
	bpb, err := uc.CreateRequisition(ctx, CreateMaterialRequisitionDTO{
		RequisitionType:  domain.RequisitionTypeRawMaterial,
		DocumentCode:     "GDG-011-BPB",
		MaterialCategory: "GSKN",
		FairtradeStatus:  "Fairtrade",
		RequestDate:      "2026-11-18",
		Department:       "Produksi Ayak",
		RequestedBy:      "Mandor Produksi",
		Lines: []MaterialRequisitionLineDTO{
			{
				RawMaterialStockID:    &sbbID,
				QuantityRequested:     500.0,
				PackageCountRequested: 20,
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create BPB: %v", err)
	}

	if !strings.HasPrefix(bpb.RequestNo, "BPB-") {
		t.Fatalf("expected BPB- prefix, got %s", bpb.RequestNo)
	}
	if bpb.Status != "submitted" {
		t.Fatalf("expected status submitted, got %s", bpb.Status)
	}

	// Verify SBB reservation
	updatedSBB := invRepo.sbbEntries[sbbID]
	if updatedSBB.QtyReserved != 500.0 {
		t.Fatalf("expected QtyReserved = 500.0, got %f", updatedSBB.QtyReserved)
	}

	// 3. Operator / Gudang Isi LBB (Laporan Permintaan Bahan Baku GDG-013-LBB)
	// 20 bags weighed with 0.03 kg plastic tare per bag
	weight25 := 25.0
	bagEntries := make([]CreateLBBBagEntryDTO, 0, 20)
	for i := 1; i <= 20; i++ {
		w := weight25
		bagEntries = append(bagEntries, CreateLBBBagEntryDTO{
			RowNumber:   i,
			LotCode:     "18O12KP",
			WeightValue: &w,
		})
	}

	lineID := bpb.Lines[0].ID
	lbb, err := uc.CreateLBBReport(ctx, CreateLBBReportDTO{
		RequisitionID:     &bpb.ID,
		RequisitionLineID: &lineID,
		ReportDate:        "2026-11-18",
		VariantType:       "GSKN",
		FairtradeStatus:   "Fairtrade",
		ProductID:         prodID,
		MaterialCode:      "18O12KP",
		ProductName:       "Gula Semut Kelapa Organik",
		SupplierName:      "Kelompok Tani Sejahtera",
		PlasticTarePerBag: 0.03, // 0.03 kg * 20 sak = 0.6 kg tare
		OperatorRole:      "Operator Ayak",
		PreparedBy:        "Budi (Operator Ayak)",
		BagEntries:        bagEntries,
	})
	if err != nil {
		t.Fatalf("failed to create LBB: %v", err)
	}

	if lbb.TotalBags != 20 {
		t.Fatalf("expected 20 bags, got %d", lbb.TotalBags)
	}
	if lbb.TotalGrossWeight != 500.0 { // 20 * 25
		t.Fatalf("expected 500.0 gross weight, got %f", lbb.TotalGrossWeight)
	}
	expectedPlasticTare := 20 * 0.03 // 0.6
	if lbb.TotalPlasticTare != expectedPlasticTare {
		t.Fatalf("expected plastic tare %f, got %f", expectedPlasticTare, lbb.TotalPlasticTare)
	}
	expectedNetWeight := 500.0 - 0.6 // 499.4
	if lbb.TotalNetWeight != expectedNetWeight {
		t.Fatalf("expected net weight %f, got %f", expectedNetWeight, lbb.TotalNetWeight)
	}

	// Verify sync back to BPB line and status change to lbb_recorded
	bpbReloaded, _ := uc.GetRequisitionByID(ctx, bpb.ID)
	if bpbReloaded.Status != "lbb_recorded" {
		t.Fatalf("expected status lbb_recorded, got %s", bpbReloaded.Status)
	}
	if bpbReloaded.Lines[0].QuantityIssued != expectedNetWeight {
		t.Fatalf("expected BPB QuantityIssued synced to %f, got %f", expectedNetWeight, bpbReloaded.Lines[0].QuantityIssued)
	}

	// 4. ACC Gudang (ApproveRequisitionByWarehouse)
	approvedBPB, err := uc.ApproveRequisitionByWarehouse(ctx, bpb.ID, ApproveRequisitionByWarehouseDTO{
		WarehouseApprovedBy: "Abel (Manager Gudang)",
		Notes:               "Bahan baku siap diambil",
	})
	if err != nil {
		t.Fatalf("failed to approve requisition by warehouse: %v", err)
	}

	if approvedBPB.Status != "warehouse_approved" {
		t.Fatalf("expected status warehouse_approved, got %s", approvedBPB.Status)
	}
	if !approvedBPB.StockDeducted {
		t.Fatalf("expected stockDeducted = true")
	}

	// Verify SBB balances after issue
	sbbAfterIssue := invRepo.sbbEntries[sbbID]
	if sbbAfterIssue.QtyReserved != 0 {
		t.Fatalf("expected QtyReserved reset to 0, got %f", sbbAfterIssue.QtyReserved)
	}
	if sbbAfterIssue.QtyIssued != expectedNetWeight {
		t.Fatalf("expected QtyIssued %f, got %f", expectedNetWeight, sbbAfterIssue.QtyIssued)
	}
	expectedRemaining := 1000.0 - expectedNetWeight
	if sbbAfterIssue.QtyRemaining != expectedRemaining {
		t.Fatalf("expected QtyRemaining %f, got %f", expectedRemaining, sbbAfterIssue.QtyRemaining)
	}
	if sbbAfterIssue.RemainingPackages != 20 { // 40 - 20
		t.Fatalf("expected remaining packages 20, got %d", sbbAfterIssue.RemainingPackages)
	}

	// Verify warehouse stock level deducted
	stkLvl := invRepo.stockLvls[[2]uuid.UUID{prodID, whID}]
	if stkLvl.Quantity > 501 || stkLvl.Quantity < 500 {
		t.Fatalf("expected stock level ~501 (1000 - 499), got %d", stkLvl.Quantity)
	}

	// Verify warehouse movements logged
	if len(invRepo.movements) == 0 {
		t.Fatalf("expected stock movement logged")
	}

	// 5. Masuk ke Produksi (EnterProductionFromRequisition)
	orderID := uuid.New()
	mfgRepo.orders[orderID] = &domain.ProductionOrder{
		OrderNumber: "PO-202611-001",
		Status:      domain.OrderPlanned,
	}
	mfgRepo.orders[orderID].ID = orderID
	approvedBPB.ProductionOrderID = &orderID
	_ = mfgRepo.UpdateRequisition(ctx, approvedBPB)

	inProdBPB, err := uc.EnterProductionFromRequisition(ctx, bpb.ID, EnterProductionDTO{
		EnteredProductionBy: "Operator Produksi Ayak",
		Notes:               "Bahan diterima di lini ayak",
	})
	if err != nil {
		t.Fatalf("failed to enter production: %v", err)
	}

	if inProdBPB.Status != "in_production" {
		t.Fatalf("expected status in_production, got %s", inProdBPB.Status)
	}

	// Verify production order released
	if mfgRepo.orders[orderID].Status != domain.OrderReleased {
		t.Fatalf("expected production order status released, got %s", mfgRepo.orders[orderID].Status)
	}

	// Verify batch genealogy recorded
	if len(mfgRepo.genealogies) == 0 {
		t.Fatalf("expected genealogy record created")
	}
	g := mfgRepo.genealogies[0]
	if g.ParentBatchNo != "18O12KP" || g.ChildBatchNo != "PO-202611-001" {
		t.Fatalf("unexpected genealogy: %+v", g)
	}
}
