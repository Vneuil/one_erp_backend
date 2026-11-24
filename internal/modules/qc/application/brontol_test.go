package application

import (
	"context"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/qc/domain"
	"github.com/google/uuid"
)

type mockBrontolRepo struct {
	domain.QCRepository
	brontolReports map[uuid.UUID]*domain.BrontolReport
	rejectReturns  map[uuid.UUID]*domain.RejectReturnReport
	incomingChecks map[uuid.UUID]*domain.RawMaterialIncomingCheck
	sievingLogs    map[uuid.UUID]*domain.SievingMonitoringLog
	ovenLogs       map[uuid.UUID]*domain.OvenMonitoringLog
	fgLogs         map[uuid.UUID]*domain.FinishedGoodsInspectionLog
	prefixCounters map[string]int64
}

func newMockBrontolRepo() *mockBrontolRepo {
	return &mockBrontolRepo{
		brontolReports: make(map[uuid.UUID]*domain.BrontolReport),
		rejectReturns:  make(map[uuid.UUID]*domain.RejectReturnReport),
		incomingChecks: make(map[uuid.UUID]*domain.RawMaterialIncomingCheck),
		sievingLogs:    make(map[uuid.UUID]*domain.SievingMonitoringLog),
		ovenLogs:       make(map[uuid.UUID]*domain.OvenMonitoringLog),
		fgLogs:         make(map[uuid.UUID]*domain.FinishedGoodsInspectionLog),
		prefixCounters: make(map[string]int64),
	}
}

func (m *mockBrontolRepo) CreateBrontolReport(ctx context.Context, report *domain.BrontolReport) error {
	if report.ID == uuid.Nil {
		report.ID = uuid.New()
	}
	m.brontolReports[report.ID] = report
	return nil
}

func (m *mockBrontolRepo) GetBrontolReportByID(ctx context.Context, id uuid.UUID) (*domain.BrontolReport, error) {
	if r, ok := m.brontolReports[id]; ok {
		return r, nil
	}
	return nil, errNotFound
}

func (m *mockBrontolRepo) GetBrontolReportByNo(ctx context.Context, reportNo string) (*domain.BrontolReport, error) {
	for _, r := range m.brontolReports {
		if r.ReportNo == reportNo {
			return r, nil
		}
	}
	return nil, errNotFound
}

func (m *mockBrontolRepo) ListBrontolReports(ctx context.Context, month, stage, action, status, search string) ([]domain.BrontolReport, error) {
	var list []domain.BrontolReport
	for _, r := range m.brontolReports {
		if month != "" && r.ReportMonth != month {
			continue
		}
		if stage != "" && r.SourceStage != stage {
			continue
		}
		if action != "" && r.ActionDisposition != action {
			continue
		}
		if status != "" && r.Status != status {
			continue
		}
		list = append(list, *r)
	}
	return list, nil
}

func (m *mockBrontolRepo) UpdateBrontolReport(ctx context.Context, report *domain.BrontolReport) error {
	m.brontolReports[report.ID] = report
	return nil
}

func (m *mockBrontolRepo) DeleteBrontolReport(ctx context.Context, id uuid.UUID) error {
	delete(m.brontolReports, id)
	return nil
}

func (m *mockBrontolRepo) CreateRejectReturnReport(ctx context.Context, ret *domain.RejectReturnReport) error {
	if ret.ID == uuid.Nil {
		ret.ID = uuid.New()
	}
	m.rejectReturns[ret.ID] = ret
	if r, ok := m.brontolReports[ret.BrontolReportID]; ok {
		r.ReturnReport = ret
	}
	return nil
}

func (m *mockBrontolRepo) GetRejectReturnReportByID(ctx context.Context, id uuid.UUID) (*domain.RejectReturnReport, error) {
	if r, ok := m.rejectReturns[id]; ok {
		return r, nil
	}
	return nil, errNotFound
}

func (m *mockBrontolRepo) GetRejectReturnReportByBrontolID(ctx context.Context, brontolReportID uuid.UUID) (*domain.RejectReturnReport, error) {
	for _, ret := range m.rejectReturns {
		if ret.BrontolReportID == brontolReportID {
			return ret, nil
		}
	}
	return nil, errNotFound
}

func (m *mockBrontolRepo) ListRejectReturnReports(ctx context.Context, month, status, search string) ([]domain.RejectReturnReport, error) {
	var list []domain.RejectReturnReport
	for _, r := range m.rejectReturns {
		if status != "" && r.Status != status {
			continue
		}
		list = append(list, *r)
	}
	return list, nil
}

func (m *mockBrontolRepo) UpdateRejectReturnReport(ctx context.Context, ret *domain.RejectReturnReport) error {
	m.rejectReturns[ret.ID] = ret
	return nil
}

func (m *mockBrontolRepo) DeleteRejectReturnReport(ctx context.Context, id uuid.UUID) error {
	delete(m.rejectReturns, id)
	return nil
}

func (m *mockBrontolRepo) GetBrontolStats(ctx context.Context) (totalCount int64, totalQuantityKg float64, returnCount int64, returnQtyKg float64, reworkCount int64, err error) {
	totalCount = int64(len(m.brontolReports))
	for _, r := range m.brontolReports {
		totalQuantityKg += r.Quantity
		if r.ActionDisposition == domain.ActionReturnToSupplier {
			returnCount++
			returnQtyKg += r.Quantity
		}
		if r.ActionDisposition == domain.ActionRework || r.ActionDisposition == domain.ActionReOven {
			reworkCount++
		}
	}
	return
}

func (m *mockBrontolRepo) CountLogsByPrefix(ctx context.Context, tableName string, prefix string) (int64, error) {
	m.prefixCounters[prefix]++
	return m.prefixCounters[prefix] - 1, nil
}

func (m *mockBrontolRepo) CreateIncomingCheck(ctx context.Context, check *domain.RawMaterialIncomingCheck) error {
	check.ID = uuid.New()
	m.incomingChecks[check.ID] = check
	return nil
}

func (m *mockBrontolRepo) CreateSievingLog(ctx context.Context, log *domain.SievingMonitoringLog) error {
	log.ID = uuid.New()
	m.sievingLogs[log.ID] = log
	return nil
}

func (m *mockBrontolRepo) CreateOvenLog(ctx context.Context, log *domain.OvenMonitoringLog) error {
	log.ID = uuid.New()
	m.ovenLogs[log.ID] = log
	return nil
}

func (m *mockBrontolRepo) CreateFinishedGoodsLog(ctx context.Context, log *domain.FinishedGoodsInspectionLog) error {
	log.ID = uuid.New()
	m.fgLogs[log.ID] = log
	return nil
}

func TestBrontolReportWithReturnCreation(t *testing.T) {
	repo := newMockBrontolRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	// 1. Create Brontol Report with ActionReturnToSupplier
	supID := uuid.New()
	dto := CreateBrontolReportDTO{
		ReportDate:        "2026-11-22",
		SourceStage:       domain.StageIncomingRawMaterial,
		ProductName:       "Gula Kelapa Semut Natural",
		ProductCode:       "GSKN-01",
		BatchNo:           "25N67 PB3",
		MaterialType:      "BM",
		SupplierID:        &supID,
		SupplierName:      "Kelompok Tani Sejahtera",
		Quantity:          25.5,
		Unit:              "KG",
		DefectReason:      "Tercemar kotoran dan serangga (Ditolak Total)",
		ActionDisposition: domain.ActionReturnToSupplier,
		QCInspectorName:   "Ahmad QC",
		OperatorName:      "Budi",
	}

	report, err := uc.CreateBrontolReport(ctx, dto)
	if err != nil {
		t.Fatalf("unexpected error creating brontol report: %v", err)
	}

	if report.ReportNo != "BRT-001-1126" {
		t.Errorf("expected reportNo BRT-001-1126, got %s", report.ReportNo)
	}
	if !report.IsReturn {
		t.Errorf("expected IsReturn to be true")
	}
	if report.ReturnReport == nil {
		t.Fatalf("expected linked ReturnReport to be auto-generated")
	}
	if report.ReturnReport.ReturnNo != "RTR-001-1126" {
		t.Errorf("expected returnNo RTR-001-1126, got %s", report.ReturnReport.ReturnNo)
	}
	if report.ReturnReport.ReturnQuantity != 25.5 {
		t.Errorf("expected return quantity 25.5, got %f", report.ReturnReport.ReturnQuantity)
	}
}

func TestAutoCreateBrontolOnIncomingCheckReject(t *testing.T) {
	repo := newMockBrontolRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	moisture := 3.2
	supID := uuid.New()
	dto := CreateIncomingCheckDTO{
		CheckDate:          "2026-11-22",
		CheckMonth:         "2026-11",
		SupplierID:         &supID,
		SupplierName:       "CV Mitra Alam",
		ProductName:        "Gula Kelapa Organik",
		BatchNo:            "BATCH-IN-001",
		TotalBags:          20,
		GrossWeight:        1000,
		NetWeight:          980,
		AcceptedWeight:     0,
		RejectedWeight:     980, // Entire shipment rejected
		SugarCondition:     "Basah & Menggumpal",
		ForeignMatterClean: false,
		MoistureContent:    &moisture,
		QCName:             "Dewi QC",
	}

	checkRes, err := uc.CreateIncomingCheck(ctx, dto)
	if err != nil {
		t.Fatalf("unexpected error creating incoming check: %v", err)
	}
	if checkRes.OverallStatus != "rejected" {
		t.Fatalf("expected overallStatus rejected, got %s", checkRes.OverallStatus)
	}

	// Verify Brontol and Retur were automatically created
	brontols, err := uc.ListBrontolReports(ctx, "2026-11", domain.StageIncomingRawMaterial, "", "", "")
	if err != nil {
		t.Fatalf("error listing brontols: %v", err)
	}
	if len(brontols) != 1 {
		t.Fatalf("expected 1 brontol report auto-generated, got %d", len(brontols))
	}

	b := brontols[0]
	if b.Quantity != 980 {
		t.Errorf("expected quantity 980, got %f", b.Quantity)
	}
	if b.ActionDisposition != domain.ActionReturnToSupplier {
		t.Errorf("expected action return_to_supplier, got %s", b.ActionDisposition)
	}
	if !b.IsReturn {
		t.Errorf("expected isReturn to be true")
	}

	// Verify linked return
	returns, err := uc.ListRejectReturns(ctx, "2026-11", "", "")
	if err != nil {
		t.Fatalf("error listing returns: %v", err)
	}
	if len(returns) != 1 {
		t.Fatalf("expected 1 reject return report, got %d", len(returns))
	}
	if returns[0].SupplierName != "CV Mitra Alam" {
		t.Errorf("expected supplier CV Mitra Alam, got %s", returns[0].SupplierName)
	}
}

func TestAutoCreateBrontolOnSievingAndOvenFailure(t *testing.T) {
	repo := newMockBrontolRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	// 1. Sieving QC Failure
	initPassed := false
	sievingDTO := CreateSievingLogDTO{
		CheckMonth:      "2026-11",
		CheckDate:       "2026-11-22",
		MachineNo:       "Mesin Ayak 13",
		Category:        "Mentah",
		ProductName:     "Gula Semut Curah",
		BatchNo:         "25N67 PB3",
		MeshSize:        "16",
		InitialFindings: "Mesh sobek 2 cm di sisi kanan",
		InitialAction:   "Ganti mesh ayakan baru",
		InitialPICName:  "Gama",
		InitialQCName:   "Hendra QC",
		InitialQCPassed: &initPassed,
		FinalPICName:    "Gama",
		FinalQCName:     "Hendra QC",
	}

	_, err := uc.CreateSievingLog(ctx, sievingDTO)
	if err != nil {
		t.Fatalf("unexpected error creating sieving log: %v", err)
	}

	// 2. Oven QC Failure (High moisture)
	ovenDTO := CreateOvenLogDTO{
		CheckMonth:          "2026-11",
		CheckDate:           "2026-11-22",
		OvenNo:              "Oven 02",
		ProductName:         "Gula Semut Oven",
		BatchNo:             "25N67 BO2",
		InitialTemperature:  100,
		InitialTime:         "08:00",
		InitialMoisture:     4.5,
		StabilizedTime:      "08:30",
		ProcessTemperature:  98,
		FinalTemperature:    90,
		FinalTime:           "12:00",
		FinalMoisture:       3.1, // > 2.0% (FAILED)
		Issues:              "Suhu sempat drop karena heater korslet",
		ActionTaken:         "Oven ulang selama 2 jam",
		PICName:             "Santoso",
		QCVerifierName:      "Hendra QC",
	}

	_, err = uc.CreateOvenLog(ctx, ovenDTO)
	if err != nil {
		t.Fatalf("unexpected error creating oven log: %v", err)
	}

	// Verify Brontol Reports created
	allBrontol, err := uc.ListBrontolReports(ctx, "2026-11", "", "", "", "")
	if err != nil {
		t.Fatalf("error listing brontols: %v", err)
	}
	if len(allBrontol) != 2 {
		t.Fatalf("expected 2 brontol reports auto-created, got %d", len(allBrontol))
	}

	// Check Stock Card calculation
	cards, err := uc.GetBrontolStockCard(ctx, "2026-11", "")
	if err != nil {
		t.Fatalf("error getting stock cards: %v", err)
	}
	if len(cards) != 2 {
		t.Fatalf("expected 2 stock card rows, got %d", len(cards))
	}
	if cards[0].BalanceQty <= 0 {
		t.Errorf("expected positive balance on row 1, got %f", cards[0].BalanceQty)
	}

	// Check KPI Stats
	stats, err := uc.GetBrontolStats(ctx)
	if err != nil {
		t.Fatalf("error getting stats: %v", err)
	}
	if stats.TotalCount != 2 {
		t.Errorf("expected 2 totalCount, got %d", stats.TotalCount)
	}
	if stats.ReworkCount != 2 {
		t.Errorf("expected 2 reworkCount, got %d", stats.ReworkCount)
	}
}

func TestUpdateRejectReturnStatus(t *testing.T) {
	repo := newMockBrontolRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	dto := CreateBrontolReportDTO{
		ReportDate:        "2026-11-22",
		SourceStage:       domain.StageFinishedGoods,
		ProductName:       "Gula Kelapa Organik 25kg",
		BatchNo:           "BATCH-PJD-REJ",
		Quantity:          50,
		Unit:              "KG",
		DefectReason:      "Kontaminasi serpihan kayu",
		ActionDisposition: domain.ActionReturnToSupplier,
		SupplierName:      "Supplier Utama",
		QCInspectorName:   "Bambang QC",
	}

	report, err := uc.CreateBrontolReport(ctx, dto)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	retID := report.ReturnReport.ID

	// Update shipping info
	courier := "JNE Trucking"
	vehNo := "B 9876 XYZ"
	driver := "Pak Joko"
	updated, err := uc.UpdateRejectReturn(ctx, retID, UpdateRejectReturnDTO{
		CourierName:   courier,
		VehicleNumber: vehNo,
		DriverName:    driver,
		Status:        "shipped",
	})
	if err != nil {
		t.Fatalf("error updating return: %v", err)
	}

	if updated.Status != "shipped" {
		t.Errorf("expected status shipped, got %s", updated.Status)
	}
	if updated.ShippedDate == nil {
		t.Errorf("expected ShippedDate to be populated")
	}
	if updated.DriverName != driver {
		t.Errorf("expected driver %s, got %s", driver, updated.DriverName)
	}
}
