package application

import (
	"context"
	"strings"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/manufacturing/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

type lhpLstMockRepo struct {
	domain.ManufacturingRepository
	lhps map[uuid.UUID]*domain.ProductionOutputReport
	lsts map[uuid.UUID]*domain.HandoverReport
}

func newLhpLstMockRepo() *lhpLstMockRepo {
	return &lhpLstMockRepo{
		lhps: make(map[uuid.UUID]*domain.ProductionOutputReport),
		lsts: make(map[uuid.UUID]*domain.HandoverReport),
	}
}

func (m *lhpLstMockRepo) CountLHPByPrefix(_ context.Context, prefix string) (int64, error) {
	var count int64
	for _, r := range m.lhps {
		if strings.HasPrefix(r.ReportNo, prefix) {
			count++
		}
	}
	return count, nil
}

func (m *lhpLstMockRepo) CreateLHP(_ context.Context, r *domain.ProductionOutputReport) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	for i := range r.Items {
		if r.Items[i].ID == uuid.Nil {
			r.Items[i].ID = uuid.New()
		}
	}
	m.lhps[r.ID] = r
	return nil
}

func (m *lhpLstMockRepo) GetLHPByID(_ context.Context, id uuid.UUID) (*domain.ProductionOutputReport, error) {
	r, ok := m.lhps[id]
	if !ok {
		return nil, apperrors.NewNotFound("LHP tidak ditemukan")
	}
	return r, nil
}

func (m *lhpLstMockRepo) ListLHP(_ context.Context, orderID *uuid.UUID, batchNo, status string) ([]domain.ProductionOutputReport, error) {
	var res []domain.ProductionOutputReport
	for _, r := range m.lhps {
		if orderID != nil && (r.ProductionOrderID == nil || *r.ProductionOrderID != *orderID) {
			continue
		}
		if status != "" && r.Status != status {
			continue
		}
		res = append(res, *r)
	}
	return res, nil
}

func (m *lhpLstMockRepo) UpdateLHP(_ context.Context, r *domain.ProductionOutputReport) error {
	m.lhps[r.ID] = r
	return nil
}

func (m *lhpLstMockRepo) DeleteLHP(_ context.Context, id uuid.UUID) error {
	delete(m.lhps, id)
	return nil
}

func (m *lhpLstMockRepo) CountLSTByPrefix(_ context.Context, prefix string) (int64, error) {
	var count int64
	for _, r := range m.lsts {
		if strings.HasPrefix(r.ReportNo, prefix) {
			count++
		}
	}
	return count, nil
}

func (m *lhpLstMockRepo) CreateLST(_ context.Context, r *domain.HandoverReport) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	for i := range r.Items {
		if r.Items[i].ID == uuid.Nil {
			r.Items[i].ID = uuid.New()
		}
	}
	m.lsts[r.ID] = r
	return nil
}

func (m *lhpLstMockRepo) GetLSTByID(_ context.Context, id uuid.UUID) (*domain.HandoverReport, error) {
	r, ok := m.lsts[id]
	if !ok {
		return nil, apperrors.NewNotFound("LHP tidak ditemukan")
	}
	return r, nil
}

func (m *lhpLstMockRepo) ListLST(_ context.Context, lhpID *uuid.UUID, batchNo, status string) ([]domain.HandoverReport, error) {
	var res []domain.HandoverReport
	for _, r := range m.lsts {
		if lhpID != nil && (r.LHPID == nil || *r.LHPID != *lhpID) {
			continue
		}
		if status != "" && r.Status != status {
			continue
		}
		res = append(res, *r)
	}
	return res, nil
}

func (m *lhpLstMockRepo) UpdateLST(_ context.Context, r *domain.HandoverReport) error {
	m.lsts[r.ID] = r
	return nil
}

func (m *lhpLstMockRepo) DeleteLST(_ context.Context, id uuid.UUID) error {
	delete(m.lsts, id)
	return nil
}

func TestLHPAndLSTFlow(t *testing.T) {
	repo := newLhpLstMockRepo()
	uc := &manufacturingUseCase{repo: repo}
	ctx := context.Background()

	// 1. Create LHP
	lhpInput := CreateLHPInput{
		FairtradeStatus:  "NON FAIRTRADE",
		Certifications:   "ORGANIC EU",
		PackingDate:      "2026-11-21",
		PackingStartTime: "08:00",
		PackingEndTime:   "16:30",
		ProductionPIC:    "Budi - Produksi",
		AdminPIC:         "Sari Dwi A.",
		Items: []CreateLHPItemInput{
			{
				ItemNo:                 1,
				RawMaterialCode:        "GSO-001",
				ProductName:            "Organic Coconut Sugar 25kg",
				BatchNo:                "BATCH-202611-001",
				UnitPcs:                40,
				PrimaryPackagingType:   "Plastik PE",
				SecondaryPackagingType: "Sak Kertas Kraft",
				WeightPerPackKg:        25.0,
				TotalWeightKg:          1000.0,
				StickerNo:              "STK-001 s/d 040",
				RemainingStockKg:       12.5,
				BuyerName:              "Eco Organic BV",
			},
		},
	}

	lhp, err := uc.CreateLHP(ctx, lhpInput)
	if err != nil {
		t.Fatalf("CreateLHP failed: %v", err)
	}
	if !strings.HasPrefix(lhp.ReportNo, "PRO-002-LHP-") {
		t.Errorf("expected reportNo prefix PRO-002-LHP-, got %s", lhp.ReportNo)
	}
	if lhp.TotalWeightKg != 1000.0 {
		t.Errorf("expected TotalWeightKg 1000.0, got %f", lhp.TotalWeightKg)
	}
	if lhp.TotalQuantityPcs != 40 {
		t.Errorf("expected TotalQuantityPcs 40, got %d", lhp.TotalQuantityPcs)
	}

	// 2. Generate LST from LHP
	lst, err := uc.GenerateLSTFromLHP(ctx, lhp.ID, "Budi - Produksi")
	if err != nil {
		t.Fatalf("GenerateLSTFromLHP failed: %v", err)
	}
	if !strings.HasPrefix(lst.ReportNo, "GDG-009-LST-") {
		t.Errorf("expected reportNo prefix GDG-009-LST-, got %s", lst.ReportNo)
	}
	if len(lst.Items) != 1 {
		t.Fatalf("expected 1 LST item, got %d", len(lst.Items))
	}
	if lst.Items[0].BatchNo != "BATCH-202611-001" {
		t.Errorf("expected batch BATCH-202611-001, got %s", lst.Items[0].BatchNo)
	}

	// 3. QC Verification
	verifiedLST, err := uc.VerifyLSTByQC(ctx, lst.ID, "QC Team")
	if err != nil {
		t.Fatalf("VerifyLSTByQC failed: %v", err)
	}
	if verifiedLST.Status != "verified_qc" {
		t.Errorf("expected status verified_qc, got %s", verifiedLST.Status)
	}

	// 4. Warehouse Acceptance
	acceptedLST, err := uc.AcceptLSTByWarehouse(ctx, lst.ID, "Gudang Head")
	if err != nil {
		t.Fatalf("AcceptLSTByWarehouse failed: %v", err)
	}
	if acceptedLST.Status != "accepted_warehouse" {
		t.Errorf("expected status accepted_warehouse, got %s", acceptedLST.Status)
	}
}
