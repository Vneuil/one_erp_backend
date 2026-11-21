package application

import (
	"context"
	"strings"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/qc/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

type mockPlkCcoRepo struct {
	domain.QCRepository
	plkLogs        map[uuid.UUID]*domain.PackagingLabelCheckLog
	ccoLogs        map[uuid.UUID]*domain.ContainerChecklistLog
	prefixCounters map[string]int64
}

func newMockPlkCcoRepo() *mockPlkCcoRepo {
	return &mockPlkCcoRepo{
		plkLogs:        make(map[uuid.UUID]*domain.PackagingLabelCheckLog),
		ccoLogs:        make(map[uuid.UUID]*domain.ContainerChecklistLog),
		prefixCounters: make(map[string]int64),
	}
}

func (m *mockPlkCcoRepo) CountLogsByPrefix(_ context.Context, _ string, prefix string) (int64, error) {
	c := m.prefixCounters[prefix]
	m.prefixCounters[prefix] = c + 1
	return c, nil
}

// PLK
func (m *mockPlkCcoRepo) CreatePackagingLabelCheck(_ context.Context, log *domain.PackagingLabelCheckLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	m.plkLogs[log.ID] = log
	return nil
}

func (m *mockPlkCcoRepo) GetPackagingLabelCheckByID(_ context.Context, id uuid.UUID) (*domain.PackagingLabelCheckLog, error) {
	if it, ok := m.plkLogs[id]; ok {
		return it, nil
	}
	return nil, apperrors.NewNotFound("PLK tidak ditemukan")
}

func (m *mockPlkCcoRepo) ListPackagingLabelChecks(_ context.Context, month string, batchNo string, decision string) ([]domain.PackagingLabelCheckLog, error) {
	var res []domain.PackagingLabelCheckLog
	for _, it := range m.plkLogs {
		if month != "" && it.CheckMonth != month {
			continue
		}
		if decision != "" && it.OverallDecision != decision {
			continue
		}
		res = append(res, *it)
	}
	return res, nil
}

func (m *mockPlkCcoRepo) UpdatePackagingLabelCheck(_ context.Context, log *domain.PackagingLabelCheckLog) error {
	m.plkLogs[log.ID] = log
	return nil
}

func (m *mockPlkCcoRepo) DeletePackagingLabelCheck(_ context.Context, id uuid.UUID) error {
	delete(m.plkLogs, id)
	return nil
}

// CCO
func (m *mockPlkCcoRepo) CreateContainerChecklist(_ context.Context, log *domain.ContainerChecklistLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	m.ccoLogs[log.ID] = log
	return nil
}

func (m *mockPlkCcoRepo) GetContainerChecklistByID(_ context.Context, id uuid.UUID) (*domain.ContainerChecklistLog, error) {
	if it, ok := m.ccoLogs[id]; ok {
		return it, nil
	}
	return nil, apperrors.NewNotFound("CCO tidak ditemukan")
}

func (m *mockPlkCcoRepo) ListContainerChecklists(_ context.Context, month string, containerNo string, destination string, decision string) ([]domain.ContainerChecklistLog, error) {
	var res []domain.ContainerChecklistLog
	for _, it := range m.ccoLogs {
		if month != "" && it.CheckMonth != month {
			continue
		}
		if decision != "" && it.OverallDecision != decision {
			continue
		}
		res = append(res, *it)
	}
	return res, nil
}

func (m *mockPlkCcoRepo) UpdateContainerChecklist(_ context.Context, log *domain.ContainerChecklistLog) error {
	m.ccoLogs[log.ID] = log
	return nil
}

func (m *mockPlkCcoRepo) DeleteContainerChecklist(_ context.Context, id uuid.UUID) error {
	delete(m.ccoLogs, id)
	return nil
}

func TestPLKAndCCOFlow(t *testing.T) {
	repo := newMockPlkCcoRepo()
	uc := &qcUseCase{repo: repo}
	ctx := context.Background()

	// 1. Create PLK
	plkInput := CreatePackagingLabelCheckDTO{
		CheckDate:       "2026-11-21",
		QCInspectorName: "Rudi Hartono (QC)",
		FairtradeStatus: "FAIRTRADE",
		Certification:   "ORGANIC EU",
		BatchNo:         "BATCH-202611-001",
		Items: []CreatePackagingLabelCheckItemDTO{
			{
				ItemNo:             1,
				ItemName:           "Organic Coconut Sugar 25kg",
				ItemCode:           "GSO-001",
				BatchNo:            "BATCH-202611-001",
				BuyerName:          "Eco Organic BV",
				PackagingType:      "Sak Kertas Kraft 25kg",
				StickerNo:          "STK-001 s/d 040",
				IsBatchMatch:       true,
				IsExpiryMatch:      true,
				IsCertLogoMatch:    true,
				IsPrintQualityGood: true,
				IsPositionAligned:  true,
				InspectedQty:       40,
				PassedQty:          40,
				RejectedQty:        0,
				ItemDecision:       "lolos",
			},
		},
	}

	plk, err := uc.CreatePackagingLabelCheck(ctx, plkInput)
	if err != nil {
		t.Fatalf("CreatePackagingLabelCheck failed: %v", err)
	}
	if !strings.HasPrefix(plk.DocumentNo, "PLK-") {
		t.Errorf("expected DocumentNo prefix PLK-, got %s", plk.DocumentNo)
	}
	if plk.OverallDecision != "lolos" {
		t.Errorf("expected OverallDecision lolos, got %s", plk.OverallDecision)
	}
	if len(plk.Items) != 1 {
		t.Fatalf("expected 1 item in PLK, got %d", len(plk.Items))
	}

	// 2. Create CCO (All Passed)
	ccoInput := CreateContainerChecklistDTO{
		CheckDate:             "2026-11-21",
		IDCardType:            "SIM",
		IDCardNumber:          "920193819203",
		DriverName:            "Agus Santoso",
		TransporterCompany:    "PT Samudera Logistik",
		VehicleLicensePlate:   "B 9182 UE",
		ContainerName:         "Maersk Line",
		ContainerNumber:       "MSKU9182391",
		ContainerSize:         "20ft",
		SealNumber:            "ML-ID-918201",
		CabinAreaStatus:       "aman",
		UnderVehicleStatus:    "aman",
		InsideContainerStatus: "aman",
		FloorCleanliness:      "bersih",
		RightWallCleanliness:  "bersih",
		LeftWallCleanliness:   "bersih",
		RoofCleanliness:       "bersih",
		DoorCleanliness:       "bersih",
		FloorHoleStatus:       "rapat",
		RightWallHoleStatus:   "rapat",
		LeftWallHoleStatus:    "rapat",
		RoofHoleStatus:        "rapat",
		DoorHoleStatus:        "rapat",
		ShipmentDestination:   "Rotterdam Port, Netherlands",
		InspectorQC:           "Rudi Hartono (QC)",
		VerifierQA:            "Steven (QA Lead)",
	}

	cco, err := uc.CreateContainerChecklist(ctx, ccoInput)
	if err != nil {
		t.Fatalf("CreateContainerChecklist failed: %v", err)
	}
	if !strings.HasPrefix(cco.DocumentNo, "CCO-") {
		t.Errorf("expected DocumentNo prefix CCO-, got %s", cco.DocumentNo)
	}
	if cco.OverallDecision != "lolos" {
		t.Errorf("expected OverallDecision lolos, got %s", cco.OverallDecision)
	}

	// 3. Create CCO with Leak -> Expected Ditolak
	ccoLeakInput := ccoInput
	ccoLeakInput.ContainerNumber = "MSKU9999999"
	ccoLeakInput.RoofHoleStatus = "bocor" // Celah bocor cahaya pada atap kontainer!
	ccoLeakInput.OverallDecision = ""

	ccoLeak, err := uc.CreateContainerChecklist(ctx, ccoLeakInput)
	if err != nil {
		t.Fatalf("CreateContainerChecklist with leak failed: %v", err)
	}
	if ccoLeak.OverallDecision != "ditolak" {
		t.Errorf("expected OverallDecision ditolak due to roof leak, got %s", ccoLeak.OverallDecision)
	}
}
