package application

import (
	"testing"
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
