package application

import (
	"context"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/qc/domain"
	"github.com/google/uuid"
)

type mockPackingMetalPJDRepo struct {
	domain.QCRepository
	needleLogs     map[uuid.UUID]*domain.NeedleMonitoringLog
	scissorLogs    map[uuid.UUID]*domain.ScissorMonitoringLog
	metalCleanLogs map[uuid.UUID]*domain.MetalDetectorCleaningLog
	metalVerifLogs map[uuid.UUID]*domain.MetalDetectorVerificationLog
	fgLogs         map[uuid.UUID]*domain.FinishedGoodsInspectionLog
	prefixCounters map[string]int64
}

func newMockPackingMetalPJDRepo() *mockPackingMetalPJDRepo {
	return &mockPackingMetalPJDRepo{
		needleLogs:     make(map[uuid.UUID]*domain.NeedleMonitoringLog),
		scissorLogs:    make(map[uuid.UUID]*domain.ScissorMonitoringLog),
		metalCleanLogs: make(map[uuid.UUID]*domain.MetalDetectorCleaningLog),
		metalVerifLogs: make(map[uuid.UUID]*domain.MetalDetectorVerificationLog),
		fgLogs:         make(map[uuid.UUID]*domain.FinishedGoodsInspectionLog),
		prefixCounters: make(map[string]int64),
	}
}

func (m *mockPackingMetalPJDRepo) CountLogsByPrefix(ctx context.Context, tableName string, prefix string) (int64, error) {
	c := m.prefixCounters[prefix]
	m.prefixCounters[prefix] = c + 1
	return c, nil
}

// Needle
func (m *mockPackingMetalPJDRepo) CreateNeedleLog(ctx context.Context, log *domain.NeedleMonitoringLog) error {
	log.ID = uuid.New()
	m.needleLogs[log.ID] = log
	return nil
}
func (m *mockPackingMetalPJDRepo) GetNeedleLogByID(ctx context.Context, id uuid.UUID) (*domain.NeedleMonitoringLog, error) {
	if it, ok := m.needleLogs[id]; ok {
		return it, nil
	}
	return nil, errNotFound
}
func (m *mockPackingMetalPJDRepo) ListNeedleLogs(ctx context.Context, month string, machineNo string) ([]domain.NeedleMonitoringLog, error) {
	var res []domain.NeedleMonitoringLog
	for _, it := range m.needleLogs {
		res = append(res, *it)
	}
	return res, nil
}
func (m *mockPackingMetalPJDRepo) UpdateNeedleLog(ctx context.Context, log *domain.NeedleMonitoringLog) error {
	m.needleLogs[log.ID] = log
	return nil
}
func (m *mockPackingMetalPJDRepo) DeleteNeedleLog(ctx context.Context, id uuid.UUID) error {
	delete(m.needleLogs, id)
	return nil
}

// Scissor
func (m *mockPackingMetalPJDRepo) CreateScissorLog(ctx context.Context, log *domain.ScissorMonitoringLog) error {
	log.ID = uuid.New()
	m.scissorLogs[log.ID] = log
	return nil
}
func (m *mockPackingMetalPJDRepo) GetScissorLogByID(ctx context.Context, id uuid.UUID) (*domain.ScissorMonitoringLog, error) {
	if it, ok := m.scissorLogs[id]; ok {
		return it, nil
	}
	return nil, errNotFound
}
func (m *mockPackingMetalPJDRepo) ListScissorLogs(ctx context.Context, month string, location string) ([]domain.ScissorMonitoringLog, error) {
	var res []domain.ScissorMonitoringLog
	for _, it := range m.scissorLogs {
		res = append(res, *it)
	}
	return res, nil
}
func (m *mockPackingMetalPJDRepo) UpdateScissorLog(ctx context.Context, log *domain.ScissorMonitoringLog) error {
	m.scissorLogs[log.ID] = log
	return nil
}
func (m *mockPackingMetalPJDRepo) DeleteScissorLog(ctx context.Context, id uuid.UUID) error {
	delete(m.scissorLogs, id)
	return nil
}

// Metal Cleaning
func (m *mockPackingMetalPJDRepo) CreateMetalCleaningLog(ctx context.Context, log *domain.MetalDetectorCleaningLog) error {
	log.ID = uuid.New()
	m.metalCleanLogs[log.ID] = log
	return nil
}
func (m *mockPackingMetalPJDRepo) GetMetalCleaningLogByID(ctx context.Context, id uuid.UUID) (*domain.MetalDetectorCleaningLog, error) {
	if it, ok := m.metalCleanLogs[id]; ok {
		return it, nil
	}
	return nil, errNotFound
}
func (m *mockPackingMetalPJDRepo) ListMetalCleaningLogs(ctx context.Context, month string, machineNo string, location string) ([]domain.MetalDetectorCleaningLog, error) {
	var res []domain.MetalDetectorCleaningLog
	for _, it := range m.metalCleanLogs {
		res = append(res, *it)
	}
	return res, nil
}
func (m *mockPackingMetalPJDRepo) UpdateMetalCleaningLog(ctx context.Context, log *domain.MetalDetectorCleaningLog) error {
	m.metalCleanLogs[log.ID] = log
	return nil
}
func (m *mockPackingMetalPJDRepo) DeleteMetalCleaningLog(ctx context.Context, id uuid.UUID) error {
	delete(m.metalCleanLogs, id)
	return nil
}

// Metal Verification
func (m *mockPackingMetalPJDRepo) CreateMetalVerificationLog(ctx context.Context, log *domain.MetalDetectorVerificationLog) error {
	log.ID = uuid.New()
	m.metalVerifLogs[log.ID] = log
	return nil
}
func (m *mockPackingMetalPJDRepo) GetMetalVerificationLogByID(ctx context.Context, id uuid.UUID) (*domain.MetalDetectorVerificationLog, error) {
	if it, ok := m.metalVerifLogs[id]; ok {
		return it, nil
	}
	return nil, errNotFound
}
func (m *mockPackingMetalPJDRepo) ListMetalVerificationLogs(ctx context.Context, month string, productType string, location string) ([]domain.MetalDetectorVerificationLog, error) {
	var res []domain.MetalDetectorVerificationLog
	for _, it := range m.metalVerifLogs {
		res = append(res, *it)
	}
	return res, nil
}
func (m *mockPackingMetalPJDRepo) UpdateMetalVerificationLog(ctx context.Context, log *domain.MetalDetectorVerificationLog) error {
	m.metalVerifLogs[log.ID] = log
	return nil
}
func (m *mockPackingMetalPJDRepo) DeleteMetalVerificationLog(ctx context.Context, id uuid.UUID) error {
	delete(m.metalVerifLogs, id)
	return nil
}

// Finished Goods (PJD)
func (m *mockPackingMetalPJDRepo) CreateFinishedGoodsLog(ctx context.Context, log *domain.FinishedGoodsInspectionLog) error {
	log.ID = uuid.New()
	m.fgLogs[log.ID] = log
	return nil
}
func (m *mockPackingMetalPJDRepo) GetFinishedGoodsLogByID(ctx context.Context, id uuid.UUID) (*domain.FinishedGoodsInspectionLog, error) {
	if it, ok := m.fgLogs[id]; ok {
		return it, nil
	}
	return nil, errNotFound
}
func (m *mockPackingMetalPJDRepo) ListFinishedGoodsLogs(ctx context.Context, month string, productName string, batchNo string, decision string) ([]domain.FinishedGoodsInspectionLog, error) {
	var res []domain.FinishedGoodsInspectionLog
	for _, it := range m.fgLogs {
		res = append(res, *it)
	}
	return res, nil
}
func (m *mockPackingMetalPJDRepo) UpdateFinishedGoodsLog(ctx context.Context, log *domain.FinishedGoodsInspectionLog) error {
	m.fgLogs[log.ID] = log
	return nil
}
func (m *mockPackingMetalPJDRepo) DeleteFinishedGoodsLog(ctx context.Context, id uuid.UUID) error {
	delete(m.fgLogs, id)
	return nil
}

// ─── Test Cases ─────────────────────────────────────────────────────────────

func TestNeedleMonitoring_BrokenHazardAlert(t *testing.T) {
	repo := newMockPackingMetalPJDRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	// 1. Normal passed needle check
	dtoNormal := CreateNeedleLogDTO{
		CheckDate:         "2026-03-01",
		SewingMachineNo:   "Mesin Jahit 01",
		PICResponsible:    "Roji",
		InitialCondition:  "Baik",
		FinalCondition:    "Baik",
		NeedleStockInRoom: 10,
		PICName:           "Roji",
	}
	logNormal, err := uc.CreateNeedleLog(ctx, dtoNormal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logNormal.OverallStatus != "passed" {
		t.Errorf("expected status 'passed', got '%s'", logNormal.OverallStatus)
	}

	// 2. Broken needle -> hazard_alert
	dtoBroken := CreateNeedleLogDTO{
		CheckDate:         "2026-03-02",
		SewingMachineNo:   "Mesin Jahit 02",
		PICResponsible:    "Roji",
		InitialCondition:  "Baik",
		FinalCondition:    "Patah",
		NeedleStockInRoom: 9,
		PICName:           "Roji",
	}
	logBroken, err := uc.CreateNeedleLog(ctx, dtoBroken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logBroken.OverallStatus != "hazard_alert" {
		t.Errorf("expected status 'hazard_alert', got '%s'", logBroken.OverallStatus)
	}
}

func TestScissorMonitoring_CountMismatchHazardAlert(t *testing.T) {
	repo := newMockPackingMetalPJDRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	// Missing scissor -> hazard_alert
	dtoMismatch := CreateScissorLogDTO{
		CheckDate:        "2026-03-01",
		Location:         "Ruang Packing",
		PICResponsible:   "Roji",
		InitialCount:     2,
		FinalCount:       1, // 1 gunting hilang!
		InitialCondition: "Baik",
		FinalCondition:   "Baik",
		PICName:          "Roji",
	}
	log, err := uc.CreateScissorLog(ctx, dtoMismatch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if log.OverallStatus != "hazard_alert" || log.IsCountMatched {
		t.Errorf("expected status 'hazard_alert' and isCountMatched=false, got status='%s', isMatched=%v", log.OverallStatus, log.IsCountMatched)
	}
}

func TestMetalDetectorVerification_AlarmFailure(t *testing.T) {
	repo := newMockPackingMetalPJDRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	alarmFailed := false
	convStopped := true

	dto := CreateMetalVerificationLogDTO{
		CheckDate:            "2026-03-01",
		CheckTime:            "08:00",
		ProductType:          "gula_semut",
		PackageCategory:      "<= 12.5 kg",
		MonitoringRound:      1,
		FeAlarmTriggered:     &alarmFailed, // Gagal bunyi!
		FeConveyorStopped:    &convStopped,
		NonFeAlarmTriggered:  &convStopped,
		NonFeConveyorStopped: &convStopped,
		SSAlarmTriggered:     &convStopped,
		SSConveyorStopped:    &convStopped,
		PICName:              "Budi",
	}
	log, err := uc.CreateMetalVerificationLog(ctx, dto)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if log.OverallStatus != "failed" || log.IsAllPiecesPassed {
		t.Errorf("expected overallStatus 'failed' and isAllPiecesPassed=false, got status='%s', allPassed=%v", log.OverallStatus, log.IsAllPiecesPassed)
	}
}

func TestFinishedGoodsInspection_MoistureAndDecision(t *testing.T) {
	repo := newMockPackingMetalPJDRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	// 1. Lolos jika kadar air <= 2.0% dan bebas benda asing
	dtoPass := CreateFinishedGoodsDTO{
		CheckDate:       "2026-03-01",
		ProductName:     "Organic Coconut Sugar",
		BatchNo:         "BATCH-2026-001",
		NetWeight:       25.0,
		GrossWeight:     25.3,
		MoistureContent: 1.8, // <= 2.0%
		ColorScore:      2,
		QCPersonnelName: "Siti QC",
	}
	logPass, err := uc.CreateFinishedGoodsLog(ctx, dtoPass)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logPass.InspectionDecision != "lolos" || !logPass.IsMoistureStandard {
		t.Errorf("expected decision 'lolos', got '%s'", logPass.InspectionDecision)
	}

	// 2. Ditolak jika kadar air > 2.0%
	dtoFail := CreateFinishedGoodsDTO{
		CheckDate:       "2026-03-01",
		ProductName:     "Organic Coconut Sugar",
		BatchNo:         "BATCH-2026-002",
		NetWeight:       25.0,
		GrossWeight:     25.3,
		MoistureContent: 2.5, // > 2.0%
		ColorScore:      2,
		QCPersonnelName: "Siti QC",
	}
	logFail, err := uc.CreateFinishedGoodsLog(ctx, dtoFail)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logFail.InspectionDecision != "ditolak" || logFail.IsMoistureStandard {
		t.Errorf("expected decision 'ditolak', got '%s'", logFail.InspectionDecision)
	}
}
