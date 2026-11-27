package application

import (
	"context"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/qc/domain"
	"github.com/google/uuid"
)

func TestComputePBKStatus(t *testing.T) {
	tests := []struct {
		name      string
		passedQty int
		failedQty int
		expected  string
	}{
		{
			name:      "All passed zero failed",
			passedQty: 100,
			failedQty: 0,
			expected:  "passed",
		},
		{
			name:      "All failed zero passed",
			passedQty: 0,
			failedQty: 50,
			expected:  "failed",
		},
		{
			name:      "Partial passed and failed",
			passedQty: 95,
			failedQty: 5,
			expected:  "partial",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computePBKStatus(tt.passedQty, tt.failedQty)
			if got != tt.expected {
				t.Errorf("computePBKStatus(%d, %d) = %v; want %v", tt.passedQty, tt.failedQty, got, tt.expected)
			}
		})
	}
}

func TestComputeCASStatus(t *testing.T) {
	tests := []struct {
		name              string
		cabinSafe         bool
		undercarriageSafe bool
		cargoAreaSafe     bool
		floorClean        bool
		wallRightClean    bool
		wallLeftClean     bool
		ceilingClean      bool
		doorClean         bool
		expected          string
	}{
		{
			name:              "All safe and all clean",
			cabinSafe:         true,
			undercarriageSafe: true,
			cargoAreaSafe:     true,
			floorClean:        true,
			wallRightClean:    true,
			wallLeftClean:     true,
			ceilingClean:      true,
			doorClean:         true,
			expected:          "cleared",
		},
		{
			name:              "Suspicious goods in cabin",
			cabinSafe:         false,
			undercarriageSafe: true,
			cargoAreaSafe:     true,
			floorClean:        true,
			wallRightClean:    true,
			wallLeftClean:     true,
			ceilingClean:      true,
			doorClean:         true,
			expected:          "rejected",
		},
		{
			name:              "Suspicious goods under carriage",
			cabinSafe:         true,
			undercarriageSafe: false,
			cargoAreaSafe:     true,
			floorClean:        true,
			wallRightClean:    true,
			wallLeftClean:     true,
			ceilingClean:      true,
			doorClean:         true,
			expected:          "rejected",
		},
		{
			name:              "All safe but floor dirty",
			cabinSafe:         true,
			undercarriageSafe: true,
			cargoAreaSafe:     true,
			floorClean:        false,
			wallRightClean:    true,
			wallLeftClean:     true,
			ceilingClean:      true,
			doorClean:         true,
			expected:          "flagged",
		},
		{
			name:              "All safe but door dirty",
			cabinSafe:         true,
			undercarriageSafe: true,
			cargoAreaSafe:     true,
			floorClean:        true,
			wallRightClean:    true,
			wallLeftClean:     true,
			ceilingClean:      true,
			doorClean:         false,
			expected:          "flagged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeCASStatus(
				tt.cabinSafe, tt.undercarriageSafe, tt.cargoAreaSafe,
				tt.floorClean, tt.wallRightClean, tt.wallLeftClean, tt.ceilingClean, tt.doorClean,
			)
			if got != tt.expected {
				t.Errorf("computeCASStatus() = %v; want %v", got, tt.expected)
			}
		})
	}
}

func TestComputePBDStatus(t *testing.T) {
	moistureLow := 1.8
	moistureHigh := 3.5

	tests := []struct {
		name           string
		acceptedWeight float64
		rejectedWeight float64
		moisture       *float64
		foreignClean   bool
		expectedStatus string
	}{
		{
			name:           "Standard passed with low moisture",
			acceptedWeight: 1000,
			rejectedWeight: 0,
			moisture:       &moistureLow,
			foreignClean:   true,
			expectedStatus: "passed",
		},
		{
			name:           "Conditional due to high moisture",
			acceptedWeight: 1000,
			rejectedWeight: 0,
			moisture:       &moistureHigh,
			foreignClean:   true,
			expectedStatus: "conditional",
		},
		{
			name:           "Conditional due to partial rejection",
			acceptedWeight: 900,
			rejectedWeight: 100,
			moisture:       &moistureLow,
			foreignClean:   true,
			expectedStatus: "conditional",
		},
		{
			name:           "Rejected due to foreign matter contamination",
			acceptedWeight: 1000,
			rejectedWeight: 0,
			moisture:       &moistureLow,
			foreignClean:   false,
			expectedStatus: "rejected",
		},
		{
			name:           "Rejected total rejection zero accepted",
			acceptedWeight: 0,
			rejectedWeight: 1000,
			moisture:       &moistureLow,
			foreignClean:   true,
			expectedStatus: "rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, _ := computePBDStatus(tt.acceptedWeight, tt.rejectedWeight, tt.moisture, tt.foreignClean)
			if status != tt.expectedStatus {
				t.Errorf("computePBDStatus() status = %v; want %v", status, tt.expectedStatus)
			}
		})
	}
}

type mockPBDRepo struct {
	domain.QCRepository
	checks map[uuid.UUID]*domain.RawMaterialIncomingCheck
}

func newMockPBDRepo() *mockPBDRepo {
	return &mockPBDRepo{checks: make(map[uuid.UUID]*domain.RawMaterialIncomingCheck)}
}

func (m *mockPBDRepo) CreateIncomingCheck(_ context.Context, check *domain.RawMaterialIncomingCheck) error {
	if check.ID == uuid.Nil {
		check.ID = uuid.New()
	}
	m.checks[check.ID] = check
	return nil
}

func (m *mockPBDRepo) GetIncomingCheckByID(_ context.Context, id uuid.UUID) (*domain.RawMaterialIncomingCheck, error) {
	return m.checks[id], nil
}

func (m *mockPBDRepo) ListIncomingChecks(_ context.Context, month string, date string, lbmNo string) ([]domain.RawMaterialIncomingCheck, error) {
	var out []domain.RawMaterialIncomingCheck
	for _, c := range m.checks {
		if month != "" && c.CheckMonth != month {
			continue
		}
		if date != "" && c.CheckDate != date {
			continue
		}
		if lbmNo != "" && c.LBMNo != lbmNo {
			continue
		}
		out = append(out, *c)
	}
	return out, nil
}

func (m *mockPBDRepo) UpdateIncomingCheck(_ context.Context, check *domain.RawMaterialIncomingCheck) error {
	m.checks[check.ID] = check
	return nil
}

func (m *mockPBDRepo) CreateBrontolReport(_ context.Context, _ *domain.BrontolReport) error {
	return nil
}

func (m *mockPBDRepo) CreateRejectReturnReport(_ context.Context, _ *domain.RejectReturnReport) error {
	return nil
}

func TestIncomingCheckPBDRev05Lifecycle(t *testing.T) {
	repo := newMockPBDRepo()
	uc := NewQCUseCase(repo)
	ctx := context.Background()

	bTrue := true
	bFalse := false
	moisture := 1.75
	brix := 78.5
	ph := 6.4
	passRate := 98.5

	created, err := uc.CreateIncomingCheck(ctx, CreateIncomingCheckDTO{
		FairtradeStatus:    "FAIRTRADE",
		Certifications:     "ORGANIC EU, ORGANIC USDA",
		CheckDate:          "2026-11-26",
		SupplierName:       "Pak Wiryo",
		ProductName:        "Gula Semut Kelapa Organik",
		ItemCode:           "26O48PW01",
		ItemType:           "Gula Semut",
		SampleBags:         13,
		NetWeight:          1000,
		AcceptedWeight:     1000,
		CheckPackaging:     &bTrue,
		CheckShape:         &bTrue,
		CheckTaste:         &bTrue,
		CheckAroma:         &bTrue,
		ColorScore:         1,
		MoistureContent:    &moisture,
		BrixValue:          &brix,
		CheckSolubility:    &bTrue,
		PhValue:            &ph,
		CheckPackageLabel:  &bTrue,
		PassRatePercent:    &passRate,
		InspectionDecision: "LOLOS",
		ForeignMatterClean: true,
		QCName:             "Adin",
		QAName:             "Mario",
	})
	if err != nil {
		t.Fatalf("CreateIncomingCheck error: %v", err)
	}
	if created.DocumentCode != "QC-002-PBD" || created.RevisionCode != "05" {
		t.Errorf("unexpected doc code/rev: %s / %s", created.DocumentCode, created.RevisionCode)
	}
	if created.FairtradeStatus != "FAIRTRADE" || created.Certifications != "ORGANIC EU, ORGANIC USDA" {
		t.Errorf("unexpected certification metadata: %s / %s", created.FairtradeStatus, created.Certifications)
	}
	if created.ItemCode != "26O48PW01" || created.BatchNo != "26O48PW01" || created.ItemType != "Gula Semut" {
		t.Errorf("unexpected item identity: %+v", created)
	}
	if created.ColorScore != 1 || created.PhValue == nil || *created.PhValue != 6.4 {
		t.Errorf("unexpected color/pH: %d / %v", created.ColorScore, created.PhValue)
	}
	if created.PassRatePercent != 98.5 || created.InspectionDecision != "LOLOS" {
		t.Errorf("unexpected passRate/decision: %v / %s", created.PassRatePercent, created.InspectionDecision)
	}

	listed, err := uc.ListIncomingChecks(ctx, "2026-11", "", "")
	if err != nil || len(listed) != 1 {
		t.Fatalf("ListIncomingChecks by month expected 1, got %d (err=%v)", len(listed), err)
	}

	passRateLow := 60.0
	updated, err := uc.UpdateIncomingCheck(ctx, created.ID, UpdateIncomingCheckDTO{
		CheckTaste:         &bFalse,
		ColorScore:         3,
		PassRatePercent:    &passRateLow,
		InspectionDecision: "DITOLAK",
		ForeignMatterClean: true,
		AcceptedWeight:     600,
		RejectedWeight:     400,
	})
	if err != nil {
		t.Fatalf("UpdateIncomingCheck error: %v", err)
	}
	if updated.InspectionDecision != "DITOLAK" || updated.OverallStatus != "rejected" {
		t.Errorf("expected DITOLAK / rejected, got %s / %s", updated.InspectionDecision, updated.OverallStatus)
	}
	if updated.ColorScore != 3 || updated.ColorAppearance != "3 - Dark Brown" {
		t.Errorf("expected color score 3 (Dark Brown), got %d / %s", updated.ColorScore, updated.ColorAppearance)
	}
}
