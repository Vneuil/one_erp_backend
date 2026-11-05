package application

import (
	"testing"
)

func TestTC_PRD_002_FixedFormHeaderPR09(t *testing.T) {
	header := GetRequisitionFormHeader("Organik EU")
	if header.DocumentNumber != "PR-09" {
		t.Fatalf("expected document number PR-09, got %s", header.DocumentNumber)
	}
	if header.CertificationCode != "Organik EU" {
		t.Fatalf("expected certification code 'Organik EU', got %s", header.CertificationCode)
	}
}

func TestTC_PRD_003_QCChecklistEvaluation(t *testing.T) {
	// Passing case: moisture 1.5% (<2.0%), metal trap clean
	resPass := EvaluateQCChecklist(1.5, true)
	if !resPass.Passed || !resPass.MoistureOK || !resPass.MetalTrapOK {
		t.Fatalf("expected pass for 1.5%% moisture and clean metal trap, got %+v", resPass)
	}

	// Failing case: moisture 2.5% (>= 2.0%)
	resFailMoisture := EvaluateQCChecklist(2.5, true)
	if resFailMoisture.Passed || resFailMoisture.MoistureOK {
		t.Fatalf("expected fail for 2.5%% moisture, got %+v", resFailMoisture)
	}

	// Failing case: metal trap not clean
	resFailTrap := EvaluateQCChecklist(1.8, false)
	if resFailTrap.Passed || resFailTrap.MetalTrapOK {
		t.Fatalf("expected fail for dirty metal trap, got %+v", resFailTrap)
	}
}

func TestTC_PRD_004_ManualFillingSettleConfirmation(t *testing.T) {
	// Fluctuation: live reading is 11.2, but settle button NOT pressed
	recorded, ok := ConfirmSettleWeight(11.2, false)
	if ok || recorded != 0 {
		t.Fatalf("scale fluctuation must not record weight without settle confirmation")
	}

	// Settled: reading is 10.00 kg and settle confirmed
	recorded, ok = ConfirmSettleWeight(10.00, true)
	if !ok || recorded != 10.00 {
		t.Fatalf("expected 10.00 kg recorded on settle, got %f, ok=%v", recorded, ok)
	}
}

func TestTC_PRD_005_BulkTraceabilitySerialsGeneration(t *testing.T) {
	// 850 barcodes: Year 2026, Type 'N', Week 48, Process "PW"
	serials, err := GenerateTraceabilitySerials(2026, "Natural", 48, "PW", 850)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(serials) != 850 {
		t.Fatalf("expected 850 serials, got %d", len(serials))
	}

	// First barcode: 26N48PW001
	if serials[0] != "26N48PW001" {
		t.Fatalf("expected first serial 26N48PW001, got %s", serials[0])
	}

	// Last barcode: 26N48PW850
	if serials[849] != "26N48PW850" {
		t.Fatalf("expected 850th serial 26N48PW850, got %s", serials[849])
	}

	// Check uniqueness across all 850
	seen := make(map[string]bool)
	for _, s := range serials {
		if seen[s] {
			t.Fatalf("duplicate serial found: %s", s)
		}
		seen[s] = true
	}
}

