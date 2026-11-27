package application

import (
	"context"
	"strings"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/qc/domain"
	"github.com/google/uuid"
)

type mockLiquidSugarRepo struct {
	domain.QCRepository
	mss        map[uuid.UUID]*domain.SterilizationTempLog
	pvk        map[uuid.UUID]*domain.LiquidPackagingVisualCheck
	vmd        map[uuid.UUID]*domain.MetalDetectorVerificationLog
	cpp        map[uuid.UUID]*domain.MetalDetectorCleaningLog
	mnp        map[uuid.UUID]*domain.FilterMeshMonitoringLog
	mnb        map[uuid.UUID]*domain.MeshBoltMonitoringLog
	mnc        map[uuid.UUID]*domain.LiquidSugarProcessingLog
	pjc        map[uuid.UUID]*domain.LiquidSugarFinishedGoodsLog
	vhc        map[uuid.UUID]*domain.LiquidSugarWashVerification
	brontol    []*domain.BrontolReport
}

func newMockLiquidSugarRepo() *mockLiquidSugarRepo {
	return &mockLiquidSugarRepo{
		mss: make(map[uuid.UUID]*domain.SterilizationTempLog),
		pvk: make(map[uuid.UUID]*domain.LiquidPackagingVisualCheck),
		vmd: make(map[uuid.UUID]*domain.MetalDetectorVerificationLog),
		cpp: make(map[uuid.UUID]*domain.MetalDetectorCleaningLog),
		mnp: make(map[uuid.UUID]*domain.FilterMeshMonitoringLog),
		mnb: make(map[uuid.UUID]*domain.MeshBoltMonitoringLog),
		mnc: make(map[uuid.UUID]*domain.LiquidSugarProcessingLog),
		pjc: make(map[uuid.UUID]*domain.LiquidSugarFinishedGoodsLog),
		vhc: make(map[uuid.UUID]*domain.LiquidSugarWashVerification),
	}
}

func (m *mockLiquidSugarRepo) CreateSterilizationTempLog(_ context.Context, log *domain.SterilizationTempLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	m.mss[log.ID] = log
	return nil
}
func (m *mockLiquidSugarRepo) GetSterilizationTempLogByID(_ context.Context, id uuid.UUID) (*domain.SterilizationTempLog, error) {
	if it, ok := m.mss[id]; ok {
		return it, nil
	}
	return nil, errNotFound
}
func (m *mockLiquidSugarRepo) ListSterilizationTempLogs(_ context.Context, month, stage, batchNo string) ([]domain.SterilizationTempLog, error) {
	var out []domain.SterilizationTempLog
	for _, it := range m.mss {
		if month != "" && it.CheckMonth != month {
			continue
		}
		if stage != "" && it.Stage != stage {
			continue
		}
		if batchNo != "" && it.BatchNo != batchNo {
			continue
		}
		out = append(out, *it)
	}
	return out, nil
}
func (m *mockLiquidSugarRepo) UpdateSterilizationTempLog(_ context.Context, log *domain.SterilizationTempLog) error {
	m.mss[log.ID] = log
	return nil
}
func (m *mockLiquidSugarRepo) DeleteSterilizationTempLog(_ context.Context, id uuid.UUID) error {
	delete(m.mss, id)
	return nil
}

func (m *mockLiquidSugarRepo) CreateLiquidPackagingVisualCheck(_ context.Context, log *domain.LiquidPackagingVisualCheck) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	m.pvk[log.ID] = log
	return nil
}
func (m *mockLiquidSugarRepo) GetLiquidPackagingVisualCheckByID(_ context.Context, id uuid.UUID) (*domain.LiquidPackagingVisualCheck, error) {
	if it, ok := m.pvk[id]; ok {
		return it, nil
	}
	return nil, errNotFound
}
func (m *mockLiquidSugarRepo) ListLiquidPackagingVisualChecks(_ context.Context, month, batchNo string) ([]domain.LiquidPackagingVisualCheck, error) {
	var out []domain.LiquidPackagingVisualCheck
	for _, it := range m.pvk {
		if month != "" && it.CheckMonth != month {
			continue
		}
		if batchNo != "" && it.BatchNo != batchNo {
			continue
		}
		out = append(out, *it)
	}
	return out, nil
}
func (m *mockLiquidSugarRepo) UpdateLiquidPackagingVisualCheck(_ context.Context, log *domain.LiquidPackagingVisualCheck) error {
	m.pvk[log.ID] = log
	return nil
}
func (m *mockLiquidSugarRepo) DeleteLiquidPackagingVisualCheck(_ context.Context, id uuid.UUID) error {
	delete(m.pvk, id)
	return nil
}

func (m *mockLiquidSugarRepo) CreateMetalVerificationLog(_ context.Context, log *domain.MetalDetectorVerificationLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	m.vmd[log.ID] = log
	return nil
}
func (m *mockLiquidSugarRepo) ListMetalVerificationLogs(_ context.Context, month, productType, location string) ([]domain.MetalDetectorVerificationLog, error) {
	var out []domain.MetalDetectorVerificationLog
	for _, it := range m.vmd {
		if month != "" && it.CheckMonth != month {
			continue
		}
		if productType != "" && it.ProductType != productType {
			continue
		}
		if location != "" && it.Location != location {
			continue
		}
		out = append(out, *it)
	}
	return out, nil
}

func (m *mockLiquidSugarRepo) CreateMetalCleaningLog(_ context.Context, log *domain.MetalDetectorCleaningLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	m.cpp[log.ID] = log
	return nil
}
func (m *mockLiquidSugarRepo) ListMetalCleaningLogs(_ context.Context, month, machineNo, location string) ([]domain.MetalDetectorCleaningLog, error) {
	var out []domain.MetalDetectorCleaningLog
	for _, it := range m.cpp {
		if month != "" && it.CheckMonth != month {
			continue
		}
		if machineNo != "" && it.MachineNo != machineNo {
			continue
		}
		if location != "" && it.Location != location {
			continue
		}
		out = append(out, *it)
	}
	return out, nil
}

func (m *mockLiquidSugarRepo) CreateFilterMeshLog(_ context.Context, log *domain.FilterMeshMonitoringLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	m.mnp[log.ID] = log
	return nil
}
func (m *mockLiquidSugarRepo) ListFilterMeshLogs(_ context.Context, month, category, location string) ([]domain.FilterMeshMonitoringLog, error) {
	var out []domain.FilterMeshMonitoringLog
	for _, it := range m.mnp {
		if month != "" && it.CheckMonth != month {
			continue
		}
		if category != "" && it.Category != category {
			continue
		}
		if location != "" && it.Location != location {
			continue
		}
		out = append(out, *it)
	}
	return out, nil
}

func (m *mockLiquidSugarRepo) CreateMeshBoltLog(_ context.Context, log *domain.MeshBoltMonitoringLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	m.mnb[log.ID] = log
	return nil
}
func (m *mockLiquidSugarRepo) ListMeshBoltLogs(_ context.Context, month, room, machineNo, formType, stageCategory string) ([]domain.MeshBoltMonitoringLog, error) {
	var out []domain.MeshBoltMonitoringLog
	for _, it := range m.mnb {
		if month != "" && it.CheckMonth != month {
			continue
		}
		if formType != "" && !strings.EqualFold(it.FormType, formType) {
			continue
		}
		if stageCategory != "" && !strings.EqualFold(it.StageCategory, stageCategory) {
			continue
		}
		out = append(out, *it)
	}
	return out, nil
}

func (m *mockLiquidSugarRepo) CreateLiquidSugarProcessingLog(_ context.Context, log *domain.LiquidSugarProcessingLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	m.mnc[log.ID] = log
	return nil
}
func (m *mockLiquidSugarRepo) GetLiquidSugarProcessingLogByID(_ context.Context, id uuid.UUID) (*domain.LiquidSugarProcessingLog, error) {
	if it, ok := m.mnc[id]; ok {
		return it, nil
	}
	return nil, errNotFound
}
func (m *mockLiquidSugarRepo) ListLiquidSugarProcessingLogs(_ context.Context, month, batchNo, productCode string) ([]domain.LiquidSugarProcessingLog, error) {
	var out []domain.LiquidSugarProcessingLog
	for _, it := range m.mnc {
		if month != "" && it.CheckMonth != month {
			continue
		}
		if batchNo != "" && it.BatchNo != batchNo {
			continue
		}
		if productCode != "" && !strings.Contains(it.ProductCode, productCode) {
			continue
		}
		out = append(out, *it)
	}
	return out, nil
}
func (m *mockLiquidSugarRepo) UpdateLiquidSugarProcessingLog(_ context.Context, log *domain.LiquidSugarProcessingLog) error {
	m.mnc[log.ID] = log
	return nil
}
func (m *mockLiquidSugarRepo) DeleteLiquidSugarProcessingLog(_ context.Context, id uuid.UUID) error {
	delete(m.mnc, id)
	return nil
}

func (m *mockLiquidSugarRepo) CreateLiquidSugarFinishedGoodsLog(_ context.Context, log *domain.LiquidSugarFinishedGoodsLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	m.pjc[log.ID] = log
	return nil
}
func (m *mockLiquidSugarRepo) GetLiquidSugarFinishedGoodsLogByID(_ context.Context, id uuid.UUID) (*domain.LiquidSugarFinishedGoodsLog, error) {
	if it, ok := m.pjc[id]; ok {
		return it, nil
	}
	return nil, errNotFound
}
func (m *mockLiquidSugarRepo) ListLiquidSugarFinishedGoodsLogs(_ context.Context, month, batchNo, decision string) ([]domain.LiquidSugarFinishedGoodsLog, error) {
	var out []domain.LiquidSugarFinishedGoodsLog
	for _, it := range m.pjc {
		if month != "" && it.CheckMonth != month {
			continue
		}
		if batchNo != "" && it.BatchNo != batchNo {
			continue
		}
		if decision != "" && it.InspectionDecision != decision {
			continue
		}
		out = append(out, *it)
	}
	return out, nil
}
func (m *mockLiquidSugarRepo) UpdateLiquidSugarFinishedGoodsLog(_ context.Context, log *domain.LiquidSugarFinishedGoodsLog) error {
	m.pjc[log.ID] = log
	return nil
}
func (m *mockLiquidSugarRepo) DeleteLiquidSugarFinishedGoodsLog(_ context.Context, id uuid.UUID) error {
	delete(m.pjc, id)
	return nil
}

func (m *mockLiquidSugarRepo) CreateLiquidSugarWashVerification(_ context.Context, log *domain.LiquidSugarWashVerification) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	m.vhc[log.ID] = log
	return nil
}
func (m *mockLiquidSugarRepo) GetLiquidSugarWashVerificationByID(_ context.Context, id uuid.UUID) (*domain.LiquidSugarWashVerification, error) {
	if it, ok := m.vhc[id]; ok {
		return it, nil
	}
	return nil, errNotFound
}
func (m *mockLiquidSugarRepo) ListLiquidSugarWashVerifications(_ context.Context, month, batchNo string) ([]domain.LiquidSugarWashVerification, error) {
	var out []domain.LiquidSugarWashVerification
	for _, it := range m.vhc {
		if month != "" && it.CheckMonth != month {
			continue
		}
		if batchNo != "" && it.BatchNo != batchNo {
			continue
		}
		out = append(out, *it)
	}
	return out, nil
}
func (m *mockLiquidSugarRepo) UpdateLiquidSugarWashVerification(_ context.Context, log *domain.LiquidSugarWashVerification) error {
	m.vhc[log.ID] = log
	return nil
}
func (m *mockLiquidSugarRepo) DeleteLiquidSugarWashVerification(_ context.Context, id uuid.UUID) error {
	delete(m.vhc, id)
	return nil
}

func (m *mockLiquidSugarRepo) CreateBrontolReport(_ context.Context, report *domain.BrontolReport) error {
	if report.ID == uuid.Nil {
		report.ID = uuid.New()
	}
	m.brontol = append(m.brontol, report)
	return nil
}

func TestSterilizationTempMSS(t *testing.T) {
	repo := newMockLiquidSugarRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	passLog, err := uc.CreateSterilizationTempLog(ctx, CreateSterilizationTempLogDTO{
		Stage:         "tahap_1",
		CheckDate:     "2026-11-27",
		BatchNo:       "GC-001",
		PackagingType: "Jerigen 25kg",
		InitialTime:   "07:00",
		InitialTemp:   32,
		HeatingTime:   "07:15",
		HeatingTemp:   72.5,
		FinalTime:     "08:05",
		FinalTemp:     74.0,
		QCName:        "Andin",
	})
	if err != nil {
		t.Fatalf("CreateSterilizationTempLog pass error: %v", err)
	}
	if !passLog.IsStandardMet || passLog.DurationMinutes != 50 || passLog.OverallStatus != "passed" {
		t.Errorf("expected passed MSS with 50 min, got met=%v dur=%d status=%s", passLog.IsStandardMet, passLog.DurationMinutes, passLog.OverallStatus)
	}

	failLog, err := uc.CreateSterilizationTempLog(ctx, CreateSterilizationTempLogDTO{
		Stage:         "tahap_2",
		CheckDate:     "2026-11-27",
		BatchNo:       "GC-001",
		PackagingType: "Drum 200L",
		InitialTime:   "08:10",
		InitialTemp:   30,
		HeatingTime:   "08:20",
		HeatingTemp:   65.0,
		FinalTime:     "08:50",
		FinalTemp:     66.0,
	})
	if err != nil {
		t.Fatalf("CreateSterilizationTempLog fail error: %v", err)
	}
	if failLog.IsStandardMet || failLog.DurationMinutes != 30 || failLog.OverallStatus != "action_required" {
		t.Errorf("expected action_required MSS with 30 min, got met=%v dur=%d status=%s", failLog.IsStandardMet, failLog.DurationMinutes, failLog.OverallStatus)
	}
}

func TestLiquidPackagingVisualPVK(t *testing.T) {
	repo := newMockLiquidSugarRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	clean, err := uc.CreateLiquidPackagingVisualCheck(ctx, CreateLiquidPackagingVisualCheckDTO{
		CheckDate:          "2026-11-27",
		BatchNo:            "GC-001",
		PackagingType:      "Jerigen 25 Kg",
		Quantity:           40,
		BuyerName:          "Buyer Export A",
		SterilizationInfo:  "Steril 72C 45m",
		PackagingCondition: "B",
		PICName:            "Sajiman",
		QCName:             "Andin",
	})
	if err != nil {
		t.Fatalf("CreateLiquidPackagingVisualCheck clean error: %v", err)
	}
	if clean.OverallStatus != "passed" || clean.PackagingCondition != "B" {
		t.Errorf("expected passed / B, got %s / %s", clean.OverallStatus, clean.PackagingCondition)
	}

	dirty, err := uc.CreateLiquidPackagingVisualCheck(ctx, CreateLiquidPackagingVisualCheckDTO{
		CheckDate:          "2026-11-27",
		BatchNo:            "GC-001",
		PackagingType:      "Jerigen 25 Kg",
		Quantity:           2,
		PackagingCondition: "K",
		Findings:           "Debu di tutup jerigen",
		ActionTaken:        "Cuci ulang",
		PICName:            "Sajiman",
	})
	if err != nil {
		t.Fatalf("CreateLiquidPackagingVisualCheck dirty error: %v", err)
	}
	if dirty.OverallStatus != "action_required" || dirty.PackagingCondition != "K" {
		t.Errorf("expected action_required / K, got %s / %s", dirty.OverallStatus, dirty.PackagingCondition)
	}
}

func TestMetalDetectorVMD_GulaCairDefaults(t *testing.T) {
	repo := newMockLiquidSugarRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	vmd, err := uc.CreateMetalVerificationLog(ctx, CreateMetalVerificationLogDTO{
		CheckDate:   "2026-11-27",
		ProductType: "gula_cair",
		BatchNo:     "GC-001",
		CheckTime:   "08:00",
		PICName:     "Sajiman",
	})
	if err != nil {
		t.Fatalf("CreateMetalVerificationLog gula_cair error: %v", err)
	}
	if vmd.RevisionCode != "01" || vmd.Location != "Gula Cair" || vmd.PackageCategory != "Gula Cair" {
		t.Errorf("unexpected gula_cair metadata: rev=%s loc=%s pkg=%s", vmd.RevisionCode, vmd.Location, vmd.PackageCategory)
	}
	if vmd.FeTestPieceSize != "Fe >= 1.5mm" || vmd.NonFeTestPieceSize != "Non-Fe >= 1.5mm" || vmd.SSTestPieceSize != "SS >= 2.5mm" {
		t.Errorf("unexpected gula_cair test pieces: fe=%s nonfe=%s ss=%s", vmd.FeTestPieceSize, vmd.NonFeTestPieceSize, vmd.SSTestPieceSize)
	}
}

func TestLiquidSugarProcessingMNC_CCP(t *testing.T) {
	repo := newMockLiquidSugarRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	okLog, err := uc.CreateLiquidSugarProcessingLog(ctx, CreateLiquidSugarProcessingLogDTO{
		CheckDate:        "2026-11-27",
		BatchNo:          "GC-001",
		ProductCode:      "GC-ORG-01",
		InitialTime:      "09:00",
		InitialTemp:      32,
		InitialMoisture:  28.5,
		InitialBrix:      70.0,
		InitialPH:        6.1,
		HeatingTime:      "09:20",
		HeatingTemp:      92.5,
		FinalTime:        "09:35",
		FinalProductTemp: 88.0,
		FinalMoisture:    24.0,
		FinalBrix:        75.5,
		FinalPH:          6.0,
		PICName:          "Sajiman",
	})
	if err != nil {
		t.Fatalf("CreateLiquidSugarProcessingLog ok error: %v", err)
	}
	if !okLog.IsCcpPassed || okLog.HeatingDurationMin != 15 || okLog.OverallStatus != "passed" {
		t.Errorf("expected passed CCP (15 min), got passed=%v dur=%d status=%s", okLog.IsCcpPassed, okLog.HeatingDurationMin, okLog.OverallStatus)
	}

	alertLog, err := uc.CreateLiquidSugarProcessingLog(ctx, CreateLiquidSugarProcessingLogDTO{
		CheckDate:   "2026-11-27",
		BatchNo:     "GC-001",
		ProductCode: "GC-ORG-01",
		InitialTime: "10:00",
		HeatingTime: "10:15",
		HeatingTemp: 87.0,
		FinalTime:   "10:20",
		PICName:     "Sajiman",
	})
	if err != nil {
		t.Fatalf("CreateLiquidSugarProcessingLog alert error: %v", err)
	}
	if alertLog.IsCcpPassed || alertLog.OverallStatus != "ccp_alert" {
		t.Errorf("expected ccp_alert, got passed=%v status=%s", alertLog.IsCcpPassed, alertLog.OverallStatus)
	}
}

func TestLiquidSugarFinishedGoodsPJC_RejectTriggersBrontol(t *testing.T) {
	repo := newMockLiquidSugarRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	bFalse := false
	passLog, err := uc.CreateLiquidSugarFinishedGoodsLog(ctx, CreateLiquidSugarFinishedGoodsLogDTO{
		CheckDate:          "2026-11-27",
		ProductName:        "Gula Cair Kelapa Organik",
		BatchNo:            "GC-001",
		ColorScore:         2,
		MoistureContent:    24.0,
		PHValue:            6.0,
		BrixValue:          75.5,
		InspectionDecision: "LOLOS",
		QCPersonnelName:    "Andin",
	})
	if err != nil {
		t.Fatalf("CreateLiquidSugarFinishedGoodsLog pass error: %v", err)
	}
	if passLog.InspectionDecision != "LOLOS" || passLog.ColorDescription != "Normal" || len(repo.brontol) != 0 {
		t.Errorf("unexpected pass PJC state: decision=%s color=%s brontol=%d", passLog.InspectionDecision, passLog.ColorDescription, len(repo.brontol))
	}

	rejectLog, err := uc.CreateLiquidSugarFinishedGoodsLog(ctx, CreateLiquidSugarFinishedGoodsLogDTO{
		CheckDate:              "2026-11-27",
		ProductName:            "Gula Cair Kelapa Organik",
		BatchNo:                "GC-002",
		CheckFreeForeignMatter: &bFalse,
		MoistureContent:        27.0,
		PHValue:                5.2,
		BrixValue:              69.0,
		InspectionDecision:     "DITOLAK",
		QCPersonnelName:        "Andin",
	})
	if err != nil {
		t.Fatalf("CreateLiquidSugarFinishedGoodsLog reject error: %v", err)
	}
	if rejectLog.InspectionDecision != "DITOLAK" || len(repo.brontol) != 1 {
		t.Errorf("expected DITOLAK & 1 brontol report, got decision=%s brontol=%d", rejectLog.InspectionDecision, len(repo.brontol))
	}
}

func TestLiquidSugarWashVHC(t *testing.T) {
	repo := newMockLiquidSugarRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	passVhc, err := uc.CreateLiquidSugarWashVerification(ctx, CreateLiquidSugarWashVerificationDTO{
		CheckDate:            "2026-11-27",
		BatchNo:              "GC-001",
		WaterTempActual:      78.5,
		CirculationStartTime: "15:00",
		CirculationEndTime:   "15:15",
		RinseColor:           "Bening",
		RinseAroma:           "Normal",
		RinseDirt:            "Bebas kotoran",
		QCVerifierName:       "Andin",
	})
	if err != nil {
		t.Fatalf("CreateLiquidSugarWashVerification pass error: %v", err)
	}
	if !passVhc.IsTempStandardMet || !passVhc.IsCirculationStandardMet || !passVhc.IsRinseStandardMet || passVhc.OverallStatus != "passed" {
		t.Errorf("expected passed VHC, got %+v", passVhc)
	}

	reWashVhc, err := uc.CreateLiquidSugarWashVerification(ctx, CreateLiquidSugarWashVerificationDTO{
		CheckDate:            "2026-11-27",
		BatchNo:              "GC-001",
		WaterTempActual:      70.0,
		CirculationStartTime: "15:00",
		CirculationEndTime:   "15:05",
		RinseColor:           "Keruh",
		RinseAroma:           "Aroma gula",
		RinseDirt:            "Bebas kotoran",
		QCVerifierName:       "Andin",
	})
	if err != nil {
		t.Fatalf("CreateLiquidSugarWashVerification rewash error: %v", err)
	}
	if reWashVhc.IsTempStandardMet || reWashVhc.IsCirculationStandardMet || reWashVhc.IsRinseStandardMet || reWashVhc.OverallStatus != "re_wash" {
		t.Errorf("expected re_wash VHC, got %+v", reWashVhc)
	}
}

func TestLiquidSugarFlowStatus(t *testing.T) {
	repo := newMockLiquidSugarRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	const batch = "GC-2611-001"
	const month = "2026-11"

	// Stage 1: 4 forms (MSS-1, PVK, VMD, CPP)
	_, err := uc.CreateSterilizationTempLog(ctx, CreateSterilizationTempLogDTO{
		Stage: "tahap_1", CheckDate: "2026-11-27", BatchNo: batch, PackagingType: "Jerigen 25kg",
		InitialTime: "07:00", HeatingTime: "07:10", HeatingTemp: 72, FinalTime: "08:00",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = uc.CreateLiquidPackagingVisualCheck(ctx, CreateLiquidPackagingVisualCheckDTO{
		CheckDate: "2026-11-27", BatchNo: batch, PackagingType: "Jerigen 25kg", Quantity: 20, PICName: "Sajiman",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = uc.CreateMetalVerificationLog(ctx, CreateMetalVerificationLogDTO{
		CheckDate: "2026-11-27", ProductType: "gula_cair", BatchNo: batch, CheckTime: "08:00", PICName: "Sajiman",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = uc.CreateMetalCleaningLog(ctx, CreateMetalCleaningLogDTO{
		CheckDate: "2026-11-27", Location: "R. Gula Cair", EquipmentName: "Metal Detector",
		BatchNo: batch, StartTime: "08:00", EndTime: "09:00", OperatorName: "Sajiman",
	})
	if err != nil {
		t.Fatal(err)
	}

	st1, err := uc.GetLiquidSugarFlowStatus(ctx, month, batch)
	if err != nil {
		t.Fatal(err)
	}
	if !st1.Stage1Done || st1.Stage2Done || st1.CurrentStage != 2 || st1.CompletedStagesCount != 1 {
		t.Errorf("expected stage 1 done and current=2, got %+v", st1)
	}

	// Stage 2: MSS-2 + QC-011 (MNP & MNB)
	_, err = uc.CreateSterilizationTempLog(ctx, CreateSterilizationTempLogDTO{
		Stage: "tahap_2", CheckDate: "2026-11-27", BatchNo: batch, PackagingType: "Jerigen 25kg",
		InitialTime: "08:30", HeatingTime: "08:40", HeatingTemp: 73, FinalTime: "09:30",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = uc.CreateFilterMeshLog(ctx, CreateFilterMeshLogDTO{
		Category: "gula_cair", CheckDate: "2026-11-27", ProductName: "Gula Cair", BatchNo: batch,
		MeshSize: "Mesh 150", PICResponsible: "Sajiman", InitialPICName: "Sajiman", FinalPICName: "Sajiman",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = uc.CreateMeshBoltLog(ctx, CreateMeshBoltLogDTO{
		FormType: "MNB", StageCategory: "gula_cair", CheckDate: "2026-11-27", Room: "Tangki Gula Cair",
		BatchNo: batch, PICResponsible: "Sajiman", InitialPICName: "Sajiman", FinalPICName: "Sajiman",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Stage 3: MNC
	_, err = uc.CreateLiquidSugarProcessingLog(ctx, CreateLiquidSugarProcessingLogDTO{
		CheckDate: "2026-11-27", BatchNo: batch, ProductCode: "GC-01",
		InitialTime: "09:30", HeatingTime: "09:45", HeatingTemp: 92, FinalTime: "10:00", PICName: "Sajiman",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Stage 4: PJC
	_, err = uc.CreateLiquidSugarFinishedGoodsLog(ctx, CreateLiquidSugarFinishedGoodsLogDTO{
		CheckDate: "2026-11-27", ProductName: "Gula Cair Organik", BatchNo: batch,
		MoistureContent: 23.5, PHValue: 6.1, BrixValue: 76.0, QCPersonnelName: "Andin",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Stage 5: VHC
	_, err = uc.CreateLiquidSugarWashVerification(ctx, CreateLiquidSugarWashVerificationDTO{
		CheckDate: "2026-11-27", BatchNo: batch, WaterTempActual: 78,
		CirculationStartTime: "14:00", CirculationEndTime: "14:15", QCVerifierName: "Andin",
	})
	if err != nil {
		t.Fatal(err)
	}

	stAll, err := uc.GetLiquidSugarFlowStatus(ctx, month, batch)
	if err != nil {
		t.Fatal(err)
	}
	if !stAll.Stage1Done || !stAll.Stage2Done || !stAll.Stage3Done || !stAll.Stage4Done || !stAll.Stage5Done || stAll.CompletedStagesCount != 5 {
		t.Errorf("expected all 5 stages done, got %+v", stAll)
	}
	if len(stAll.BatchNumbers) != 1 || stAll.BatchNumbers[0] != batch {
		t.Errorf("expected batchNumbers [%s], got %v", batch, stAll.BatchNumbers)
	}
}
