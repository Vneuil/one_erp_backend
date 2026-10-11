package application

import (
	"context"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/finance/domain"
	"github.com/google/uuid"
)

type fakeCashAdvanceRepo struct {
	*fakeLedgerRepo
	advances        map[uuid.UUID]*domain.CashAdvance
	planLines       map[uuid.UUID][]domain.CashAdvancePlanLine
	settlementLines map[uuid.UUID][]domain.CashAdvanceSettlementLine
	crCount         int64
	kbsCount        int64
	bkkCount        int64
}

func newFakeCashAdvanceRepo() *fakeCashAdvanceRepo {
	r := &fakeCashAdvanceRepo{
		fakeLedgerRepo:  newFakeLedgerRepo(),
		advances:        make(map[uuid.UUID]*domain.CashAdvance),
		planLines:       make(map[uuid.UUID][]domain.CashAdvancePlanLine),
		settlementLines: make(map[uuid.UUID][]domain.CashAdvanceSettlementLine),
	}
	// Add required default accounts: Cash (1000), Employee Advance (1400), Expense (5000)
	r.accounts = append(r.accounts,
		domain.Account{ID: uuid.New(), Code: AccountCash, Type: "asset", IsActive: true},
		domain.Account{ID: uuid.New(), Code: AccountEmployeeAdvance, Type: "asset", IsActive: true},
		domain.Account{ID: uuid.New(), Code: AccountExpense, Type: "expense", IsActive: true},
	)
	return r
}

func (f *fakeCashAdvanceRepo) CreateCashAdvance(_ context.Context, ca *domain.CashAdvance) error {
	f.advances[ca.ID] = ca
	return nil
}

func (f *fakeCashAdvanceRepo) UpdateCashAdvance(_ context.Context, ca *domain.CashAdvance) error {
	f.advances[ca.ID] = ca
	return nil
}

func (f *fakeCashAdvanceRepo) GetCashAdvanceByID(_ context.Context, id uuid.UUID) (*domain.CashAdvance, error) {
	ca := f.advances[id]
	if ca == nil {
		return nil, nil
	}
	ca.PlanLines = f.planLines[id]
	ca.SettlementLines = f.settlementLines[id]
	return ca, nil
}

func (f *fakeCashAdvanceRepo) ListCashAdvances(_ context.Context, status, from, to string) ([]domain.CashAdvance, error) {
	var out []domain.CashAdvance
	for _, ca := range f.advances {
		caCopy := *ca
		caCopy.PlanLines = f.planLines[ca.ID]
		caCopy.SettlementLines = f.settlementLines[ca.ID]
		out = append(out, caCopy)
	}
	return out, nil
}

func (f *fakeCashAdvanceRepo) CountCashRequests(_ context.Context, _ string) (int64, error) {
	f.crCount++
	return f.crCount - 1, nil
}

func (f *fakeCashAdvanceRepo) CountCashAdvances(_ context.Context, _ string) (int64, error) {
	f.kbsCount++
	return f.kbsCount - 1, nil
}

func (f *fakeCashAdvanceRepo) CountCashVouchers(_ context.Context, _ string, _ string) (int64, error) {
	f.bkkCount++
	return f.bkkCount - 1, nil
}

func (f *fakeCashAdvanceRepo) ReplaceCashAdvancePlanLines(_ context.Context, advanceID uuid.UUID, lines []domain.CashAdvancePlanLine) error {
	f.planLines[advanceID] = lines
	return nil
}

func (f *fakeCashAdvanceRepo) ReplaceCashAdvanceSettlementLines(_ context.Context, advanceID uuid.UUID, lines []domain.CashAdvanceSettlementLine) error {
	f.settlementLines[advanceID] = lines
	return nil
}

func TestCashAdvanceFullLifecycle(t *testing.T) {
	repo := newFakeCashAdvanceRepo()
	uc := NewFinanceUseCase(repo)
	ctx := context.Background()

	// 1. Create Cash Request
	cr, err := uc.CreateCashRequest(ctx, CashRequestInput{
		EmployeeName: "Budi Santoso",
		Department:   "Operasional Lapangan",
		Purpose:      "Beli perlengkapan logistik dan transport",
		CRAmount:     2_500_000,
		CRDate:       "2026-10-12",
	})
	if err != nil {
		t.Fatalf("CreateCashRequest error: %v", err)
	}
	if cr.CRStatus != "pending" || cr.CRNumber == "" {
		t.Fatalf("CR status or number unexpected: %+v", cr)
	}

	// 2. Reject CR test on separate CR
	crReject, err := uc.CreateCashRequest(ctx, CashRequestInput{
		EmployeeName: "Andi",
		Purpose:      "Operasional yang tidak jelas",
		CRAmount:     500_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	crRejected, err := uc.RejectCashRequest(ctx, crReject.ID, "Anggaran tidak tersedia")
	if err != nil {
		t.Fatal(err)
	}
	if crRejected.CRStatus != "rejected" || crRejected.CRRejectReason != "Anggaran tidak tersedia" {
		t.Fatalf("CR rejected state wrong: %+v", crRejected)
	}

	// 3. Approve first CR
	crApproved, err := uc.ApproveCashRequest(ctx, cr.ID)
	if err != nil {
		t.Fatalf("ApproveCashRequest error: %v", err)
	}
	if crApproved.CRStatus != "approved" {
		t.Fatalf("CR status should be approved, got %s", crApproved.CRStatus)
	}

	// 4. Fill Form Kas Bon
	kbs1, err := uc.FillCashAdvanceForm(ctx, cr.ID, CashAdvanceFormInput{
		KBSDate:    "2026-10-12",
		KBSDueDate: "2026-10-19",
		KBSAmount:  2_500_000,
		Lines: []CashAdvancePlanLineInput{
			{Description: "Bensin dan Tol", EstimatedAmount: 1_000_000, TargetAccountCode: AccountExpense},
			{Description: "Perlengkapan Lapangan", EstimatedAmount: 1_500_000, TargetAccountCode: AccountExpense},
		},
	})
	if err != nil {
		t.Fatalf("FillCashAdvanceForm error: %v", err)
	}
	if kbs1.KBSStatus != "pending" || kbs1.KBSNumber == "" {
		t.Fatalf("KBS status or number unexpected: %+v", kbs1)
	}

	// 4b. Test Reject Form for revision
	kbsRev, err := uc.RejectCashAdvanceForm(ctx, cr.ID, "Estimasi bensin terlalu besar, mohon kurangi")
	if err != nil {
		t.Fatalf("RejectCashAdvanceForm error: %v", err)
	}
	if kbsRev.KBSStatus != "revision" || kbsRev.KBSRejectReason == "" {
		t.Fatalf("KBS revision status wrong: %+v", kbsRev)
	}

	// 4c. Re-fill after revision
	kbs2, err := uc.FillCashAdvanceForm(ctx, cr.ID, CashAdvanceFormInput{
		KBSDate:    "2026-10-12",
		KBSDueDate: "2026-10-19",
		KBSAmount:  2_200_000,
		Lines: []CashAdvancePlanLineInput{
			{Description: "Bensin dan Tol (Revisi)", EstimatedAmount: 700_000, TargetAccountCode: AccountExpense},
			{Description: "Perlengkapan Lapangan", EstimatedAmount: 1_500_000, TargetAccountCode: AccountExpense},
		},
	})
	if err != nil {
		t.Fatalf("Re-fill error: %v", err)
	}
	if kbs2.KBSStatus != "pending" || kbs2.KBSAmount != 2_200_000 {
		t.Fatalf("KBS status should be pending, amount 2.2M, got: %+v", kbs2)
	}

	// 4d. Approve Form Kas Bon
	kbsApproved, err := uc.ApproveCashAdvanceForm(ctx, cr.ID)
	if err != nil {
		t.Fatalf("ApproveCashAdvanceForm error: %v", err)
	}
	if kbsApproved.KBSStatus != "approved" {
		t.Fatalf("KBS status should be approved, got %s", kbsApproved.KBSStatus)
	}

	// 5. Disburse Cash Advance
	disbursed, err := uc.DisburseCashAdvance(ctx, cr.ID, DisburseCashAdvanceInput{
		Date:            "2026-10-12",
		CashAccountCode: AccountCash,
		PaymentMethod:   "Tunai",
	})
	if err != nil {
		t.Fatalf("DisburseCashAdvance error: %v", err)
	}
	if disbursed.DisburseDate == "" || disbursed.DisburseAmount != 2_200_000 {
		t.Fatalf("Disburse error: %+v", disbursed)
	}

	// Check disburse journal entry: Dr. Employee Advance (1400) / Cr. Cash (1000)
	disburseJE := repo.entries[CashAdvanceSourceDisbursePrefix+cr.ID.String()]
	if disburseJE == nil {
		t.Fatal("Disburse journal entry missing")
	}
	if dr, crAmt := entryTotals(disburseJE); dr != 2_200_000 || dr != crAmt {
		t.Fatalf("Disburse JE unbalanced or wrong amount: dr=%v cr=%v", dr, crAmt)
	}

	// 6. Submit Pertanggungjawaban (PJ)
	// Suppose real expenses = 2,000,000 (Sisa 200,000 kembali ke kas)
	pj1, err := uc.SubmitSettlement(ctx, cr.ID, SubmitSettlementInput{
		PJDate:                "2026-10-15",
		SettlementAccountCode: AccountCash,
		Lines: []SettlementLineInput{
			{Description: "Bensin & Tol Riil", AccountCode: AccountExpense, Amount: 650_000, ReceiptNo: "SPBU-001"},
			{Description: "Beli Alat Kerja", AccountCode: AccountExpense, Amount: 1_350_000, ReceiptNo: "INV-9988"},
		},
	})
	if err != nil {
		t.Fatalf("SubmitSettlement error: %v", err)
	}
	if pj1.PJStatus != "pending" || pj1.PJTotalRealized != 2_000_000 || pj1.PJBalanceAmount != 200_000 {
		t.Fatalf("PJ calculation wrong: %+v", pj1)
	}
	if pj1.SettlementAction != "refund_to_company" {
		t.Fatalf("SettlementAction expected refund_to_company, got %s", pj1.SettlementAction)
	}

	// 7. Test Reject PJ for proof revision
	pjRev, err := uc.RejectSettlement(ctx, cr.ID, "Struk bensin belum ada stempel, mohon lampirkan ulang")
	if err != nil {
		t.Fatalf("RejectSettlement error: %v", err)
	}
	if pjRev.PJStatus != "revision" || pjRev.PJRejectReason == "" {
		t.Fatalf("PJ revision status wrong: %+v", pjRev)
	}

	// 7b. Re-submit PJ
	pj2, err := uc.SubmitSettlement(ctx, cr.ID, SubmitSettlementInput{
		PJDate:                "2026-10-16",
		SettlementAccountCode: AccountCash,
		Lines: []SettlementLineInput{
			{Description: "Bensin & Tol Riil (Lengkap)", AccountCode: AccountExpense, Amount: 650_000, ReceiptNo: "SPBU-001-OK"},
			{Description: "Beli Alat Kerja", AccountCode: AccountExpense, Amount: 1_350_000, ReceiptNo: "INV-9988"},
		},
	})
	if err != nil {
		t.Fatalf("Re-submit PJ error: %v", err)
	}
	if pj2.PJStatus != "pending" {
		t.Fatalf("PJ should be pending after resubmit, got %s", pj2.PJStatus)
	}

	// 8. Approve Settlement (Final BKK & Journal Posting)
	settled, err := uc.ApproveSettlement(ctx, cr.ID)
	if err != nil {
		t.Fatalf("ApproveSettlement error: %v", err)
	}
	if settled.PJStatus != "settled" || settled.BKKNumber == "" {
		t.Fatalf("Settled state wrong: %+v", settled)
	}

	// Check final settlement journal:
	// Dr. 5000 Expense: 2,000,000
	// Dr. 1000 Cash (refund returned): 200,000
	// Cr. 1400 Advance: 2,200,000
	settleJE := repo.entries[CashAdvanceSourceSettlementPrefix+cr.ID.String()]
	if settleJE == nil {
		t.Fatal("Settlement journal entry missing")
	}
	if dr, crAmt := entryTotals(settleJE); dr != 2_200_000 || dr != crAmt {
		t.Fatalf("Settlement JE unbalanced: dr=%v cr=%v", dr, crAmt)
	}

	// 9. Verify Reports
	outstanding, err := uc.GetOutstandingCashAdvancesReport(ctx)
	if err != nil {
		t.Fatalf("GetOutstandingCashAdvancesReport error: %v", err)
	}
	// The settled advance and rejected advance should not be outstanding
	for _, o := range outstanding {
		if o.ID == cr.ID {
			t.Errorf("Settled advance should not appear in outstanding report: %+v", o)
		}
		if o.ID == crReject.ID {
			t.Errorf("Rejected advance should not appear in outstanding report: %+v", o)
		}
	}

	diffs, err := uc.GetCashAdvanceDifferencesReport(ctx, "", "")
	if err != nil {
		t.Fatalf("GetCashAdvanceDifferencesReport error: %v", err)
	}
	if len(diffs) == 0 {
		t.Fatal("Expected difference report item for settled advance with 200,000 refund")
	}
	if diffs[0].BalanceAmount != 200_000 || diffs[0].SettlementAction != "refund_to_company" {
		t.Fatalf("Difference report item incorrect: %+v", diffs[0])
	}
}
