package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/qc/domain"
	"github.com/google/uuid"
)

var errNotFound = errors.New("not found")

// mockQCRepo implements minimal QCRepository for testing sieving and oven methods
type mockSievingOvenRepo struct {
	domain.QCRepository
	sievingLogs    map[uuid.UUID]*domain.SievingMonitoringLog
	magnetLogs     map[uuid.UUID]*domain.MagnetTrapLog
	meshBoltLogs   map[uuid.UUID]*domain.MeshBoltMonitoringLog
	filterMeshLogs map[uuid.UUID]*domain.FilterMeshMonitoringLog
	bladeLogs      map[uuid.UUID]*domain.BrontolBladeMonitoringLog
	ovenLogs       map[uuid.UUID]*domain.OvenMonitoringLog
	prefixCounters map[string]int64
}

func newMockSievingOvenRepo() *mockSievingOvenRepo {
	return &mockSievingOvenRepo{
		sievingLogs:    make(map[uuid.UUID]*domain.SievingMonitoringLog),
		magnetLogs:     make(map[uuid.UUID]*domain.MagnetTrapLog),
		meshBoltLogs:   make(map[uuid.UUID]*domain.MeshBoltMonitoringLog),
		filterMeshLogs: make(map[uuid.UUID]*domain.FilterMeshMonitoringLog),
		bladeLogs:      make(map[uuid.UUID]*domain.BrontolBladeMonitoringLog),
		ovenLogs:       make(map[uuid.UUID]*domain.OvenMonitoringLog),
		prefixCounters: make(map[string]int64),
	}
}

func (m *mockSievingOvenRepo) CreateSievingLog(ctx context.Context, log *domain.SievingMonitoringLog) error {
	log.ID = uuid.New()
	m.sievingLogs[log.ID] = log
	return nil
}

func (m *mockSievingOvenRepo) GetSievingLogByID(ctx context.Context, id uuid.UUID) (*domain.SievingMonitoringLog, error) {
	if item, ok := m.sievingLogs[id]; ok {
		return item, nil
	}
	return nil, errNotFound
}

func (m *mockSievingOvenRepo) ListSievingLogs(ctx context.Context, month string, category string, machineNo string, batchNo string) ([]domain.SievingMonitoringLog, error) {
	var res []domain.SievingMonitoringLog
	for _, v := range m.sievingLogs {
		if category != "" && v.Category != category {
			continue
		}
		res = append(res, *v)
	}
	return res, nil
}

func (m *mockSievingOvenRepo) UpdateSievingLog(ctx context.Context, log *domain.SievingMonitoringLog) error {
	m.sievingLogs[log.ID] = log
	return nil
}

func (m *mockSievingOvenRepo) DeleteSievingLog(ctx context.Context, id uuid.UUID) error {
	delete(m.sievingLogs, id)
	return nil
}

func (m *mockSievingOvenRepo) CreateMagnetTrapLog(ctx context.Context, log *domain.MagnetTrapLog) error {
	log.ID = uuid.New()
	m.magnetLogs[log.ID] = log
	return nil
}

func (m *mockSievingOvenRepo) GetMagnetTrapLogByID(ctx context.Context, id uuid.UUID) (*domain.MagnetTrapLog, error) {
	if item, ok := m.magnetLogs[id]; ok {
		return item, nil
	}
	return nil, errNotFound
}

func (m *mockSievingOvenRepo) ListMagnetTrapLogs(ctx context.Context, month string, room string, machineNo string) ([]domain.MagnetTrapLog, error) {
	var res []domain.MagnetTrapLog
	for _, v := range m.magnetLogs {
		res = append(res, *v)
	}
	return res, nil
}

func (m *mockSievingOvenRepo) UpdateMagnetTrapLog(ctx context.Context, log *domain.MagnetTrapLog) error {
	m.magnetLogs[log.ID] = log
	return nil
}

func (m *mockSievingOvenRepo) DeleteMagnetTrapLog(ctx context.Context, id uuid.UUID) error {
	delete(m.magnetLogs, id)
	return nil
}

func (m *mockSievingOvenRepo) CreateMeshBoltLog(ctx context.Context, log *domain.MeshBoltMonitoringLog) error {
	log.ID = uuid.New()
	m.meshBoltLogs[log.ID] = log
	return nil
}

func (m *mockSievingOvenRepo) GetMeshBoltLogByID(ctx context.Context, id uuid.UUID) (*domain.MeshBoltMonitoringLog, error) {
	if item, ok := m.meshBoltLogs[id]; ok {
		return item, nil
	}
	return nil, errNotFound
}

func (m *mockSievingOvenRepo) ListMeshBoltLogs(ctx context.Context, month string, room string, machineNo string, formType string, stageCategory string) ([]domain.MeshBoltMonitoringLog, error) {
	var res []domain.MeshBoltMonitoringLog
	for _, v := range m.meshBoltLogs {
		if formType != "" && !strings.EqualFold(v.FormType, formType) {
			continue
		}
		if stageCategory != "" && !strings.EqualFold(v.StageCategory, stageCategory) {
			continue
		}
		res = append(res, *v)
	}
	return res, nil
}

func (m *mockSievingOvenRepo) UpdateMeshBoltLog(ctx context.Context, log *domain.MeshBoltMonitoringLog) error {
	m.meshBoltLogs[log.ID] = log
	return nil
}

func (m *mockSievingOvenRepo) DeleteMeshBoltLog(ctx context.Context, id uuid.UUID) error {
	delete(m.meshBoltLogs, id)
	return nil
}

func (m *mockSievingOvenRepo) CreateOvenLog(ctx context.Context, log *domain.OvenMonitoringLog) error {
	log.ID = uuid.New()
	m.ovenLogs[log.ID] = log
	return nil
}

func (m *mockSievingOvenRepo) GetOvenLogByID(ctx context.Context, id uuid.UUID) (*domain.OvenMonitoringLog, error) {
	if item, ok := m.ovenLogs[id]; ok {
		return item, nil
	}
	return nil, errNotFound
}

func (m *mockSievingOvenRepo) ListOvenLogs(ctx context.Context, month string, ovenNo string, batchNo string) ([]domain.OvenMonitoringLog, error) {
	var res []domain.OvenMonitoringLog
	for _, v := range m.ovenLogs {
		res = append(res, *v)
	}
	return res, nil
}

func (m *mockSievingOvenRepo) UpdateOvenLog(ctx context.Context, log *domain.OvenMonitoringLog) error {
	m.ovenLogs[log.ID] = log
	return nil
}

func (m *mockSievingOvenRepo) DeleteOvenLog(ctx context.Context, id uuid.UUID) error {
	delete(m.ovenLogs, id)
	return nil
}

func (m *mockSievingOvenRepo) CreateFilterMeshLog(ctx context.Context, log *domain.FilterMeshMonitoringLog) error {
	log.ID = uuid.New()
	m.filterMeshLogs[log.ID] = log
	return nil
}

func (m *mockSievingOvenRepo) GetFilterMeshLogByID(ctx context.Context, id uuid.UUID) (*domain.FilterMeshMonitoringLog, error) {
	if item, ok := m.filterMeshLogs[id]; ok {
		return item, nil
	}
	return nil, errNotFound
}

func (m *mockSievingOvenRepo) ListFilterMeshLogs(ctx context.Context, month string, category string, location string) ([]domain.FilterMeshMonitoringLog, error) {
	var res []domain.FilterMeshMonitoringLog
	for _, v := range m.filterMeshLogs {
		if category != "" && v.Category != category {
			continue
		}
		res = append(res, *v)
	}
	return res, nil
}

func (m *mockSievingOvenRepo) UpdateFilterMeshLog(ctx context.Context, log *domain.FilterMeshMonitoringLog) error {
	m.filterMeshLogs[log.ID] = log
	return nil
}

func (m *mockSievingOvenRepo) DeleteFilterMeshLog(ctx context.Context, id uuid.UUID) error {
	delete(m.filterMeshLogs, id)
	return nil
}

func (m *mockSievingOvenRepo) CreateBrontolBladeLog(ctx context.Context, log *domain.BrontolBladeMonitoringLog) error {
	log.ID = uuid.New()
	m.bladeLogs[log.ID] = log
	return nil
}

func (m *mockSievingOvenRepo) GetBrontolBladeLogByID(ctx context.Context, id uuid.UUID) (*domain.BrontolBladeMonitoringLog, error) {
	if item, ok := m.bladeLogs[id]; ok {
		return item, nil
	}
	return nil, errNotFound
}

func (m *mockSievingOvenRepo) ListBrontolBladeLogs(ctx context.Context, month string, productCode string) ([]domain.BrontolBladeMonitoringLog, error) {
	var res []domain.BrontolBladeMonitoringLog
	for _, v := range m.bladeLogs {
		res = append(res, *v)
	}
	return res, nil
}

func (m *mockSievingOvenRepo) UpdateBrontolBladeLog(ctx context.Context, log *domain.BrontolBladeMonitoringLog) error {
	m.bladeLogs[log.ID] = log
	return nil
}

func (m *mockSievingOvenRepo) DeleteBrontolBladeLog(ctx context.Context, id uuid.UUID) error {
	delete(m.bladeLogs, id)
	return nil
}

func (m *mockSievingOvenRepo) CreateBrontolReport(ctx context.Context, report *domain.BrontolReport) error {
	return nil
}

func (m *mockSievingOvenRepo) CreateRejectReturnReport(ctx context.Context, report *domain.RejectReturnReport) error {
	return nil
}

func (m *mockSievingOvenRepo) CountLogsByPrefix(ctx context.Context, tableName string, prefix string) (int64, error) {
	m.prefixCounters[prefix]++
	return m.prefixCounters[prefix], nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestSievingMNA(t *testing.T) {
	repo := newMockSievingOvenRepo()
	uc := NewQCUseCase(repo)

	ctx := context.Background()
	log, err := uc.CreateSievingLog(ctx, CreateSievingLogDTO{
		MachineNo:      "Mesin Ayak 01",
		CheckDate:      "2026-03-02",
		Category:       "Mentah",
		ProductName:    "Gula Semut Kelapa Organik",
		BatchNo:        "26O10PW001",
		MeshSize:       "Mesh 18",
		InitialPICName: "Operator Ayak 1",
		FinalPICName:   "Operator Ayak 1",
	})
	if err != nil {
		t.Fatalf("expected no error creating sieving log, got %v", err)
	}

	if log.OverallStatus != "passed" {
		t.Errorf("expected overallStatus passed, got %s", log.OverallStatus)
	}
	if log.Category != "Mentah" {
		t.Errorf("expected category Mentah, got %s", log.Category)
	}
}

func TestMagnetTrapCMT_GaussThreshold(t *testing.T) {
	repo := newMockSievingOvenRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	// Test case 1: Gauss Sufficient (>= 700 mT)
	log1, err := uc.CreateMagnetTrapLog(ctx, CreateMagnetTrapLogDTO{
		MachineNo:      "Mesin Ayak 01",
		CheckDate:      "2026-03-02",
		Room:           "Ayak Mentah",
		PICResponsible: "Mandor Ayak",
		PICName:        "Operator 1",
		GaussPoint1:    720.0,
		GaussPoint2:    715.0,
		GaussPoint3:    710.0,
	})
	if err != nil {
		t.Fatalf("error creating magnet trap log 1: %v", err)
	}
	if !log1.IsPowerSufficient {
		t.Errorf("expected power sufficient true, got false")
	}
	if log1.OverallStatus != "cleared" {
		t.Errorf("expected overallStatus cleared, got %s", log1.OverallStatus)
	}

	// Test case 2: Gauss Below Critical Limit (< 700 mT)
	log2, err := uc.CreateMagnetTrapLog(ctx, CreateMagnetTrapLogDTO{
		MachineNo:      "Mesin Ayak 01",
		CheckDate:      "2026-03-02",
		Room:           "Ayak Mentah",
		PICResponsible: "Mandor Ayak",
		PICName:        "Operator 1",
		GaussPoint1:    680.0, // Drop below 700 mT!
		GaussPoint2:    710.0,
		GaussPoint3:    700.0,
	})
	if err != nil {
		t.Fatalf("error creating magnet trap log 2: %v", err)
	}
	if log2.IsPowerSufficient {
		t.Errorf("expected power sufficient false for point < 700, got true")
	}
	if log2.OverallStatus != "warning" {
		t.Errorf("expected overallStatus warning, got %s", log2.OverallStatus)
	}
}

func TestMeshBoltMNM_BoltCountIntegrity(t *testing.T) {
	repo := newMockSievingOvenRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	// Case 1: All 16 bolts intact
	log1, err := uc.CreateMeshBoltLog(ctx, CreateMeshBoltLogDTO{
		MachineNo:        "Mesin Ayak 01",
		CheckDate:        "2026-03-02",
		MeshSize:         "Mesh 18",
		PICResponsible:   "Mandor Ayak",
		InitialPICName:   "Operator 1",
		FinalPICName:     "Operator 1",
		InitialBoltCount: 16,
		FinalBoltCount:   16,
	})
	if err != nil {
		t.Fatalf("error creating mesh bolt log 1: %v", err)
	}
	if !log1.IsCountMatched {
		t.Errorf("expected isCountMatched true, got false")
	}
	if log1.OverallStatus != "passed" {
		t.Errorf("expected status passed, got %s", log1.OverallStatus)
	}

	// Case 2: 1 Bolt Missing at the end (Hazard Alert!)
	log2, err := uc.CreateMeshBoltLog(ctx, CreateMeshBoltLogDTO{
		MachineNo:        "Mesin Ayak 01",
		CheckDate:        "2026-03-02",
		MeshSize:         "Mesh 18",
		PICResponsible:   "Mandor Ayak",
		InitialPICName:   "Operator 1",
		FinalPICName:     "Operator 1",
		InitialBoltCount: 16,
		FinalBoltCount:   15, // Missing 1 bolt!
	})
	if err != nil {
		t.Fatalf("error creating mesh bolt log 2: %v", err)
	}
	if log2.IsCountMatched {
		t.Errorf("expected isCountMatched false, got true")
	}
	if log2.OverallStatus != "hazard_alert" {
		t.Errorf("expected status hazard_alert, got %s", log2.OverallStatus)
	}
}

func TestOvenMNO_MoistureContentTarget(t *testing.T) {
	repo := newMockSievingOvenRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	// Case 1: Target Reached (Final Moisture <= 2.0%)
	log1, err := uc.CreateOvenLog(ctx, CreateOvenLogDTO{
		OvenNo:             "Oven 01",
		CheckDate:          "2026-03-02",
		BatchNo:            "26O10PW001",
		ProductName:        "Gula Semut Kelapa Organik",
		InitialTemperature: 60.0,
		InitialTime:        "08:00",
		InitialMoisture:    2.8,
		StabilizedTime:     "08:45",
		ProcessTemperature: 100.0,
		FinalTemperature:   95.0,
		FinalTime:          "11:30",
		FinalMoisture:      1.8, // <= 2.0%
		PICName:            "Operator Oven 1",
	})
	if err != nil {
		t.Fatalf("error creating oven log 1: %v", err)
	}
	if !log1.IsTargetMoistureReached {
		t.Errorf("expected target moisture reached true, got false")
	}
	if log1.OverallStatus != "passed" {
		t.Errorf("expected overallStatus passed, got %s", log1.OverallStatus)
	}

	// Case 2: Moisture Too High (> 2.0%) -> re_oven
	log2, err := uc.CreateOvenLog(ctx, CreateOvenLogDTO{
		OvenNo:             "Oven 01",
		CheckDate:          "2026-03-02",
		BatchNo:            "26O10PW001",
		ProductName:        "Gula Semut Kelapa Organik",
		InitialTemperature: 60.0,
		InitialTime:        "08:00",
		InitialMoisture:    3.2,
		StabilizedTime:     "08:45",
		ProcessTemperature: 100.0,
		FinalTemperature:   95.0,
		FinalTime:          "10:30",
		FinalMoisture:      2.4, // > 2.0%
		PICName:            "Operator Oven 1",
	})
	if err != nil {
		t.Fatalf("error creating oven log 2: %v", err)
	}
	if log2.IsTargetMoistureReached {
		t.Errorf("expected target moisture reached false for moisture 2.4%%, got true")
	}
	if log2.OverallStatus != "re_oven" {
		t.Errorf("expected overallStatus re_oven, got %s", log2.OverallStatus)
	}
}

func TestFilterMeshMNP_LiquidSugar(t *testing.T) {
	repo := newMockSievingOvenRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	log, err := uc.CreateFilterMeshLog(ctx, CreateFilterMeshLogDTO{
		CheckDate:      "2026-12-01",
		Category:       "gula_cair",
		ProductName:    "Gula Kelapa Cair",
		ProductCode:    "GC-001",
		MeshSize:       "Mesh 80",
		PICResponsible: "Sajiman",
		InitialPICName: "Sajiman",
		FinalPICName:   "Sajiman",
	})
	if err != nil {
		t.Fatalf("error creating filter mesh log: %v", err)
	}
	if log.DocumentCode != "QC-011-MNP" {
		t.Errorf("expected DocumentCode QC-011-MNP, got %s", log.DocumentCode)
	}
	if log.OverallStatus != "passed" {
		t.Errorf("expected passed, got %s", log.OverallStatus)
	}
}

func TestFilterMeshMMV_VCO(t *testing.T) {
	repo := newMockSievingOvenRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	log, err := uc.CreateFilterMeshLog(ctx, CreateFilterMeshLogDTO{
		CheckDate:      "2026-12-01",
		Category:       "vco",
		ProductName:    "Virgin Coconut Oil",
		ProductCode:    "VCO-01",
		MeshSize:       "Mesh 100",
		PICResponsible: "Abel",
		InitialPICName: "Abel",
		FinalPICName:   "Abel",
	})
	if err != nil {
		t.Fatalf("error creating filter mesh log: %v", err)
	}
	if log.DocumentCode != "QC-011-MMV" {
		t.Errorf("expected DocumentCode QC-011-MMV, got %s", log.DocumentCode)
	}
	if log.Location != "Ruang VCO" {
		t.Errorf("expected Location Ruang VCO, got %s", log.Location)
	}
}

func TestBrontolBladeMPB(t *testing.T) {
	repo := newMockSievingOvenRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	// Normal check
	log1, err := uc.CreateBrontolBladeLog(ctx, CreateBrontolBladeLogDTO{
		CheckDate:        "2026-12-01",
		ProductName:      "Gula Semut Kelapa",
		PICResponsible:   "Sajiman",
		InitialPICName:   "Sajiman",
		FinalPICName:     "Sajiman",
		InitialCondition: "Baik",
		FinalCondition:   "Baik",
	})
	if err != nil {
		t.Fatalf("error creating blade log: %v", err)
	}
	if log1.DocumentCode != "QC-011-MPB" {
		t.Errorf("expected QC-011-MPB, got %s", log1.DocumentCode)
	}
	if log1.OverallStatus != "passed" {
		t.Errorf("expected passed, got %s", log1.OverallStatus)
	}

	// Hazard alert check (pisau gompal/retak)
	log2, err := uc.CreateBrontolBladeLog(ctx, CreateBrontolBladeLogDTO{
		CheckDate:        "2026-12-01",
		ProductName:      "Gula Semut Kelapa",
		PICResponsible:   "Sajiman",
		InitialPICName:   "Sajiman",
		FinalPICName:     "Sajiman",
		InitialCondition: "Gompal",
		FinalCondition:   "Gompal",
	})
	if err != nil {
		t.Fatalf("error creating blade log 2: %v", err)
	}
	if log2.OverallStatus != "hazard_alert" {
		t.Errorf("expected hazard_alert for blade gompal, got %s", log2.OverallStatus)
	}
}

func TestMeshBolt_MNB_and_MBB(t *testing.T) {
	repo := newMockSievingOvenRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	// MNB (Tangki Gula Cair)
	mnbLog, err := uc.CreateMeshBoltLog(ctx, CreateMeshBoltLogDTO{
		CheckDate:      "2026-12-01",
		Room:           "Tangki Gula Cair",
		PICResponsible: "Sajiman",
		InitialPICName: "Sajiman",
		FinalPICName:   "Sajiman",
	})
	if err != nil {
		t.Fatalf("error creating MNB log: %v", err)
	}
	if mnbLog.FormType != "MNB" || mnbLog.DocumentCode != "QC-011-MNB" {
		t.Errorf("expected MNB / QC-011-MNB, got %s / %s", mnbLog.FormType, mnbLog.DocumentCode)
	}

	// MBB (Mesin Brontol)
	mbbLog, err := uc.CreateMeshBoltLog(ctx, CreateMeshBoltLogDTO{
		CheckDate:      "2026-12-01",
		Room:           "Mesin Brontol",
		PICResponsible: "Sajiman",
		InitialPICName: "Sajiman",
		FinalPICName:   "Sajiman",
	})
	if err != nil {
		t.Fatalf("error creating MBB log: %v", err)
	}
	if mbbLog.FormType != "MBB" || mbbLog.DocumentCode != "QC-011-MBB" {
		t.Errorf("expected MBB / QC-011-MBB, got %s / %s", mbbLog.FormType, mbbLog.DocumentCode)
	}
}
