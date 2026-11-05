package application

import (
	"fmt"
	"strings"
)

// GenerateTraceabilitySerials generates sequential final batch traceability codes (TC-PRD-005)
// Format: [YY][N/O][BB][SS][NNN]
// Examples:
// - Year: 2026 -> 26
// - OrganicType: Natural -> 'N', Organic -> 'O'
// - Week: 48 -> "48"
// - Step/Process: "PW" (Packing Warehouse)
// - Serial: 001 to 850
func GenerateTraceabilitySerials(year int, organicType string, week int, processCode string, count int) ([]string, error) {
	if count <= 0 {
		return nil, fmt.Errorf("count must be greater than 0")
	}
	yy := year % 100
	typeCode := "N"
	if strings.EqualFold(organicType, "Organic") || strings.EqualFold(organicType, "Organik") || strings.EqualFold(organicType, "O") || strings.EqualFold(organicType, "EU") {
		typeCode = "O"
	} else if strings.EqualFold(organicType, "Natural") || strings.EqualFold(organicType, "N") {
		typeCode = "N"
	}

	proc := strings.ToUpper(strings.TrimSpace(processCode))
	if proc == "" {
		proc = "PW"
	}

	serials := make([]string, count)
	for i := 1; i <= count; i++ {
		serials[i-1] = fmt.Sprintf("%02d%s%02d%s%03d", yy, typeCode, week, proc, i)
	}
	return serials, nil
}

// RequisitionFormHeader contains the fixed document header per auditor requirements (TC-PRD-002)
type RequisitionFormHeader struct {
	DocumentNumber    string `json:"documentNumber"`    // Fixed: PR-09
	CertificationCode string `json:"certificationCode"` // e.g. "Organik EU"
	StandardTitle     string `json:"standardTitle"`     // Form Pengambilan Bahan Baku
}

// GetRequisitionFormHeader returns the standardized form header with static PR-09 code and organic standard.
func GetRequisitionFormHeader(organicType string) RequisitionFormHeader {
	cert := "Organik EU"
	if strings.EqualFold(organicType, "Natural") {
		cert = "Non-Organik / Natural"
	}
	return RequisitionFormHeader{
		DocumentNumber:    "PR-09",
		CertificationCode: cert,
		StandardTitle:     "Form Pengambilan Bahan Baku",
	}
}

// QCChecklistResult holds the result of inspecting the 4 manufacturing lines (TC-PRD-003)
type QCChecklistResult struct {
	Passed      bool     `json:"passed"`
	MoistureOK  bool     `json:"moistureOk"`
	MetalTrapOK bool     `json:"metalTrapOk"`
	Remarks     []string `json:"remarks"`
}

// EvaluateQCChecklist evaluates quality parameters:
// - Moisture (Kadar Air) must be < 2.0%
// - Metal Trap must be Clean (True)
func EvaluateQCChecklist(moisture float64, metalTrapClean bool) QCChecklistResult {
	res := QCChecklistResult{
		Passed:      true,
		MoistureOK:  moisture < 2.0,
		MetalTrapOK: metalTrapClean,
		Remarks:     make([]string, 0),
	}

	if !res.MoistureOK {
		res.Passed = false
		res.Remarks = append(res.Remarks, fmt.Sprintf("Kadar air %.2f%% melebihi batas toleransi maksimal 2.0%%", moisture))
	}
	if !res.MetalTrapOK {
		res.Passed = false
		res.Remarks = append(res.Remarks, "Metal trap kotor atau terdeteksi serpihan logam (harus Bersih/Clean)")
	}

	return res
}

// ConfirmSettleWeight simulates the manual filling scale settle button (TC-PRD-004).
// Weight is only captured when confirmed/settled by the operator.
func ConfirmSettleWeight(liveReading float64, isSettled bool) (float64, bool) {
	if !isSettled {
		return 0, false
	}
	return liveReading, true
}

