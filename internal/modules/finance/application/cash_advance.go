package application

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/finance/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

const (
	CashAdvanceSourceDisbursePrefix   = "kbs-disburse:"
	CashAdvanceSourceSettlementPrefix = "kbs-settle:"
)

type CashRequestInput struct {
	EmployeeName string  `json:"employeeName"`
	Department   string  `json:"department"`
	Purpose      string  `json:"purpose"`
	CRAmount     float64 `json:"crAmount"`
	CRDate       string  `json:"crDate"`
}

type CashAdvancePlanLineInput struct {
	Description       string  `json:"description"`
	EstimatedAmount   float64 `json:"estimatedAmount"`
	TargetAccountCode string  `json:"targetAccountCode"`
}

type CashAdvanceFormInput struct {
	KBSDate    string                     `json:"kbsDate"`
	KBSDueDate string                     `json:"kbsDueDate"`
	KBSAmount  float64                    `json:"kbsAmount"`
	Lines      []CashAdvancePlanLineInput `json:"lines"`
}

type DisburseCashAdvanceInput struct {
	Date            string `json:"date"`
	CashAccountCode string `json:"cashAccountCode"`
	PaymentMethod   string `json:"paymentMethod"`
}

type SettlementLineInput struct {
	Description   string  `json:"description"`
	AccountCode   string  `json:"accountCode"`
	Amount        float64 `json:"amount"`
	ReceiptNo     string  `json:"receiptNo"`
	ReceiptDate   string  `json:"receiptDate"`
	AttachmentURL string  `json:"attachmentUrl"`
}

type SubmitSettlementInput struct {
	PJDate                string                `json:"pjDate"`
	SettlementAccountCode string                `json:"settlementAccountCode"`
	Lines                 []SettlementLineInput `json:"lines"`
}

type RejectInput struct {
	Reason string `json:"reason"`
}

type OutstandingCashAdvanceReportItem struct {
	ID                      uuid.UUID `json:"id"`
	CRNumber                string    `json:"crNumber"`
	KBSNumber               string    `json:"kbsNumber"`
	EmployeeName            string    `json:"employeeName"`
	Department              string    `json:"department"`
	Purpose                 string    `json:"purpose"`
	CRDate                  string    `json:"crDate"`
	DisburseDate            string    `json:"disburseDate"`
	KBSDueDate              string    `json:"kbsDueDate"`
	KBSAmount               float64   `json:"kbsAmount"`
	AgingDays               int       `json:"agingDays"`
	Status                  string    `json:"status"`
	DisburseCashAccountCode string    `json:"disburseCashAccountCode"`
}

type CashAdvanceDifferenceReportItem struct {
	ID                    uuid.UUID `json:"id"`
	KBSNumber             string    `json:"kbsNumber"`
	CRNumber              string    `json:"crNumber"`
	EmployeeName          string    `json:"employeeName"`
	Department            string    `json:"department"`
	Purpose               string    `json:"purpose"`
	KBSAmount             float64   `json:"kbsAmount"`
	PJTotalRealized       float64   `json:"pjTotalRealized"`
	BalanceAmount         float64   `json:"balanceAmount"`
	SettlementAction      string    `json:"settlementAction"`
	SettlementAccountCode string    `json:"settlementAccountCode"`
	PJStatus              string    `json:"pjStatus"`
	BKKNumber             string    `json:"bkkNumber"`
	SettledDate           string    `json:"settledDate"`
}

func resolveDocDate(s string) (string, error) {
	today := time.Now().Format("2006-01-02")
	if s == "" {
		return today, nil
	}
	if !validDate(s) {
		return "", apperrors.NewBadRequest("Format tanggal harus YYYY-MM-DD")
	}
	return s, nil
}

func (uc *financeUseCase) CreateCashRequest(ctx context.Context, in CashRequestInput) (*domain.CashAdvance, error) {
	name := strings.TrimSpace(in.EmployeeName)
	if name == "" {
		return nil, apperrors.NewBadRequest("Nama pemohon/karyawan wajib diisi")
	}
	purpose := strings.TrimSpace(in.Purpose)
	if purpose == "" {
		return nil, apperrors.NewBadRequest("Keperluan penggunaan kas wajib diisi")
	}
	if in.CRAmount <= 0 {
		return nil, apperrors.NewBadRequest("Nominal permohonan harus lebih besar dari 0")
	}
	date, err := resolveDocDate(in.CRDate)
	if err != nil {
		return nil, err
	}

	numberPrefix := fmt.Sprintf("CR-%s-", strings.ReplaceAll(date[:7], "-", ""))
	n, err := uc.repo.CountCashRequests(ctx, numberPrefix)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal menomori Cash Request")
	}

	ca := &domain.CashAdvance{
		CRNumber:       fmt.Sprintf("%s%04d", numberPrefix, n+1),
		EmployeeName:   name,
		Department:     strings.TrimSpace(in.Department),
		Purpose:        purpose,
		CRAmount:       round2(in.CRAmount),
		CRDate:         date,
		CRStatus:       "pending",
		KBSStatus:      "draft",
		PJStatus:       "none",
		CreatedByEmail: actor.EmailFrom(ctx),
	}
	ca.ID = uuid.New()

	if err := uc.repo.CreateCashAdvance(ctx, ca); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal menyimpan Cash Request")
	}
	return ca, nil
}

func (uc *financeUseCase) ApproveCashRequest(ctx context.Context, id uuid.UUID) (*domain.CashAdvance, error) {
	ca, err := uc.repo.GetCashAdvanceByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat Cash Request")
	}
	if ca == nil {
		return nil, apperrors.NewNotFound("Cash Request tidak ditemukan")
	}
	if ca.CRStatus != "pending" {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("Cash Request tidak dalam status menunggu approval (status saat ini: %s)", ca.CRStatus))
	}

	ca.CRStatus = "approved"
	ca.ApprovedCRByEmail = actor.EmailFrom(ctx)

	if err := uc.repo.UpdateCashAdvance(ctx, ca); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal menyetujui Cash Request")
	}
	return ca, nil
}

func (uc *financeUseCase) RejectCashRequest(ctx context.Context, id uuid.UUID, reason string) (*domain.CashAdvance, error) {
	ca, err := uc.repo.GetCashAdvanceByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat Cash Request")
	}
	if ca == nil {
		return nil, apperrors.NewNotFound("Cash Request tidak ditemukan")
	}
	if ca.CRStatus != "pending" {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("Cash Request tidak dalam status menunggu approval (status saat ini: %s)", ca.CRStatus))
	}

	ca.CRStatus = "rejected"
	ca.CRRejectReason = strings.TrimSpace(reason)

	if err := uc.repo.UpdateCashAdvance(ctx, ca); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal menolak Cash Request")
	}
	return ca, nil
}

func (uc *financeUseCase) FillCashAdvanceForm(ctx context.Context, id uuid.UUID, in CashAdvanceFormInput) (*domain.CashAdvance, error) {
	ca, err := uc.repo.GetCashAdvanceByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat data Kas Bon")
	}
	if ca == nil {
		return nil, apperrors.NewNotFound("Data Kas Bon tidak ditemukan")
	}
	if ca.CRStatus != "approved" {
		return nil, apperrors.NewBadRequest("Cash Request belum disetujui")
	}
	if ca.KBSStatus != "draft" && ca.KBSStatus != "revision" {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("Form Kas Bon tidak dapat diubah pada status %s", ca.KBSStatus))
	}
	if in.KBSAmount <= 0 {
		return nil, apperrors.NewBadRequest("Nominal Kas Bon harus lebih besar dari 0")
	}
	if len(in.Lines) == 0 {
		return nil, apperrors.NewBadRequest("Minimal satu baris rincian rencana biaya diperlukan")
	}

	date, err := resolveDocDate(in.KBSDate)
	if err != nil {
		return nil, err
	}
	dueDate, err := resolveDocDate(in.KBSDueDate)
	if err != nil {
		return nil, err
	}

	// Generate KBS number if not present yet
	if ca.KBSNumber == "" {
		numberPrefix := fmt.Sprintf("KBS-%s-", strings.ReplaceAll(date[:7], "-", ""))
		n, err := uc.repo.CountCashAdvances(ctx, numberPrefix)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Gagal menomori Form Kas Bon")
		}
		ca.KBSNumber = fmt.Sprintf("%s%04d", numberPrefix, n+1)
	}

	var planLines []domain.CashAdvancePlanLine
	for i, l := range in.Lines {
		desc := strings.TrimSpace(l.Description)
		if desc == "" {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("Baris rencana ke-%d: deskripsi wajib diisi", i+1))
		}
		if l.EstimatedAmount <= 0 {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("Baris rencana ke-%d: nominal estimasi harus positif", i+1))
		}
		planLines = append(planLines, domain.CashAdvancePlanLine{
			BaseEntity:        ca.BaseEntity,
			CashAdvanceID:     ca.ID,
			Description:       desc,
			EstimatedAmount:   round2(l.EstimatedAmount),
			TargetAccountCode: strings.TrimSpace(l.TargetAccountCode),
		})
		planLines[len(planLines)-1].ID = uuid.New()
	}

	ca.KBSDate = date
	ca.KBSDueDate = dueDate
	ca.KBSAmount = round2(in.KBSAmount)
	ca.KBSStatus = "pending"
	ca.KBSRejectReason = ""

	if err := uc.repo.UpdateCashAdvance(ctx, ca); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memperbarui formulir Kas Bon")
	}
	if err := uc.repo.ReplaceCashAdvancePlanLines(ctx, ca.ID, planLines); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal menyimpan rincian rencana Kas Bon")
	}
	ca.PlanLines = planLines
	return ca, nil
}

func (uc *financeUseCase) ApproveCashAdvanceForm(ctx context.Context, id uuid.UUID) (*domain.CashAdvance, error) {
	ca, err := uc.repo.GetCashAdvanceByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat formulir Kas Bon")
	}
	if ca == nil {
		return nil, apperrors.NewNotFound("Formulir Kas Bon tidak ditemukan")
	}
	if ca.KBSStatus != "pending" {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("Formulir Kas Bon tidak dalam status menunggu approval (status saat ini: %s)", ca.KBSStatus))
	}

	ca.KBSStatus = "approved"
	ca.ApprovedKBSByEmail = actor.EmailFrom(ctx)

	if err := uc.repo.UpdateCashAdvance(ctx, ca); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal menyetujui Formulir Kas Bon")
	}
	return ca, nil
}

func (uc *financeUseCase) RejectCashAdvanceForm(ctx context.Context, id uuid.UUID, reason string) (*domain.CashAdvance, error) {
	ca, err := uc.repo.GetCashAdvanceByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat formulir Kas Bon")
	}
	if ca == nil {
		return nil, apperrors.NewNotFound("Formulir Kas Bon tidak ditemukan")
	}
	if ca.KBSStatus != "pending" {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("Formulir Kas Bon tidak dalam status menunggu approval (status saat ini: %s)", ca.KBSStatus))
	}

	ca.KBSStatus = "revision"
	ca.KBSRejectReason = strings.TrimSpace(reason)

	if err := uc.repo.UpdateCashAdvance(ctx, ca); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal menolak Formulir Kas Bon untuk revisi")
	}
	return ca, nil
}

func (uc *financeUseCase) DisburseCashAdvance(ctx context.Context, id uuid.UUID, in DisburseCashAdvanceInput) (*domain.CashAdvance, error) {
	ca, err := uc.repo.GetCashAdvanceByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat data Kas Bon")
	}
	if ca == nil {
		return nil, apperrors.NewNotFound("Data Kas Bon tidak ditemukan")
	}
	if ca.KBSStatus != "approved" {
		return nil, apperrors.NewBadRequest("Formulir Kas Bon belum disetujui")
	}
	if ca.DisburseDate != "" {
		return nil, apperrors.NewBadRequest("Dana Kas Bon sudah dicairkan sebelumnya")
	}

	date, err := resolveDocDate(in.Date)
	if err != nil {
		return nil, err
	}
	cashCode := strings.TrimSpace(in.CashAccountCode)
	if cashCode == "" {
		cashCode = AccountCash
	}

	accounts, err := uc.repo.ListAllAccounts(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat daftar akun")
	}
	byCode := make(map[string]domain.Account, len(accounts))
	for _, a := range accounts {
		byCode[a.Code] = a
	}
	if acc, ok := byCode[cashCode]; ok {
		if !isCashAccount(acc) || !acc.IsActive {
			return nil, apperrors.NewBadRequest("Akun sumber pencairan harus merupakan akun kas/bank yang aktif")
		}
	} else if cashCode != AccountCash {
		return nil, apperrors.NewBadRequest("Akun kas/bank " + cashCode + " tidak ditemukan")
	}

	// Auto post journal entry: Dr. Employee Advance (1400) / Cr. Cash/Bank
	memo := fmt.Sprintf("Pencairan Kas Bon Sementara %s - %s", ca.KBSNumber, ca.EmployeeName)
	sourceDoc := CashAdvanceSourceDisbursePrefix + ca.ID.String()
	lines := []LedgerLine{
		{AccountCode: AccountEmployeeAdvance, Debit: ca.KBSAmount, Description: fmt.Sprintf("Kas Bon: %s", ca.Purpose)},
		{AccountCode: cashCode, Credit: ca.KBSAmount, Description: ca.KBSNumber},
	}
	poster := NewLedgerPoster(uc.repo)
	if err := poster.PostEntryOn(ctx, date, sourceDoc, memo, lines); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memposting jurnal pencairan kas bon")
	}

	je, _ := uc.repo.GetJournalEntryBySourceDoc(ctx, sourceDoc)
	var jID *uuid.UUID
	if je != nil {
		jID = &je.ID
	}

	ca.DisburseDate = date
	ca.DisburseCashAccountCode = cashCode
	ca.DisbursePaymentMethod = strings.TrimSpace(in.PaymentMethod)
	ca.DisburseAmount = ca.KBSAmount
	ca.DisburseJournalID = jID
	ca.DisbursedByEmail = actor.EmailFrom(ctx)

	if err := uc.repo.UpdateCashAdvance(ctx, ca); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memperbarui status pencairan Kas Bon")
	}
	return ca, nil
}

func (uc *financeUseCase) SubmitSettlement(ctx context.Context, id uuid.UUID, in SubmitSettlementInput) (*domain.CashAdvance, error) {
	ca, err := uc.repo.GetCashAdvanceByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat data Kas Bon")
	}
	if ca == nil {
		return nil, apperrors.NewNotFound("Data Kas Bon tidak ditemukan")
	}
	if ca.DisburseDate == "" {
		return nil, apperrors.NewBadRequest("Dana Kas Bon belum dicairkan")
	}
	if ca.PJStatus != "none" && ca.PJStatus != "revision" {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("Pertanggungjawaban tidak dapat disubmit pada status %s", ca.PJStatus))
	}
	if len(in.Lines) == 0 {
		return nil, apperrors.NewBadRequest("Minimal satu baris bukti pengeluaran riil diperlukan")
	}

	date, err := resolveDocDate(in.PJDate)
	if err != nil {
		return nil, err
	}

	accounts, err := uc.repo.ListAllAccounts(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat daftar akun")
	}
	byCode := make(map[string]domain.Account, len(accounts))
	for _, a := range accounts {
		byCode[a.Code] = a
	}

	var totalRealized float64
	var settlementLines []domain.CashAdvanceSettlementLine
	for i, l := range in.Lines {
		code := strings.TrimSpace(l.AccountCode)
		if acc, ok := byCode[code]; !ok || !acc.IsActive {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("Baris bukti ke-%d: akun beban %q tidak ditemukan atau tidak aktif", i+1, code))
		}
		if l.Amount <= 0 {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("Baris bukti ke-%d: nominal pengeluaran harus positif", i+1))
		}
		amt := round2(l.Amount)
		totalRealized += amt
		settlementLines = append(settlementLines, domain.CashAdvanceSettlementLine{
			BaseEntity:    ca.BaseEntity,
			CashAdvanceID: ca.ID,
			Description:   strings.TrimSpace(l.Description),
			AccountCode:   code,
			Amount:        amt,
			ReceiptNo:     strings.TrimSpace(l.ReceiptNo),
			ReceiptDate:   strings.TrimSpace(l.ReceiptDate),
			AttachmentURL: strings.TrimSpace(l.AttachmentURL),
		})
		settlementLines[len(settlementLines)-1].ID = uuid.New()
	}

	totalRealized = round2(totalRealized)
	balance := round2(ca.KBSAmount - totalRealized)

	settlementAction := "exact"
	if balance > 0 {
		settlementAction = "refund_to_company"
	} else if balance < 0 {
		settlementAction = "reimburse_to_employee"
	}

	settleAcc := strings.TrimSpace(in.SettlementAccountCode)
	if settleAcc == "" {
		settleAcc = ca.DisburseCashAccountCode
		if settleAcc == "" {
			settleAcc = AccountCash
		}
	}

	ca.PJDate = date
	ca.PJTotalRealized = totalRealized
	ca.PJBalanceAmount = balance
	ca.SettlementAction = settlementAction
	ca.SettlementAccountCode = settleAcc
	ca.PJStatus = "pending"
	ca.PJRejectReason = ""

	if err := uc.repo.UpdateCashAdvance(ctx, ca); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memperbarui status pertanggungjawaban")
	}
	if err := uc.repo.ReplaceCashAdvanceSettlementLines(ctx, ca.ID, settlementLines); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal menyimpan rincian bukti transaksi")
	}
	ca.SettlementLines = settlementLines
	return ca, nil
}

func (uc *financeUseCase) ApproveSettlement(ctx context.Context, id uuid.UUID) (*domain.CashAdvance, error) {
	ca, err := uc.repo.GetCashAdvanceByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat pertanggungjawaban Kas Bon")
	}
	if ca == nil {
		return nil, apperrors.NewNotFound("Pertanggungjawaban Kas Bon tidak ditemukan")
	}
	if ca.PJStatus != "pending" {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("Pertanggungjawaban tidak dalam status menunggu approval (status saat ini: %s)", ca.PJStatus))
	}
	if len(ca.SettlementLines) == 0 {
		return nil, apperrors.NewBadRequest("Rincian bukti transaksi tidak boleh kosong")
	}

	// Generate BKK Number
	date := ca.PJDate
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	prefix := fmt.Sprintf("BKK-%s-", strings.ReplaceAll(date[:7], "-", ""))
	n, err := uc.repo.CountCashVouchers(ctx, "payment", prefix)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal membuat nomor BKK")
	}
	bkkNumber := fmt.Sprintf("%s%04d", prefix, n+1)
	ca.BKKNumber = bkkNumber

	// Create Settlement Journal Lines:
	// 1. Dr. Expense Accounts per line
	// 2. Cr. Employee Advance (1400) for KBSAmount
	// 3. Balance treatment:
	//    If BalanceAmount > 0 (Refund): Dr. SettlementAccountCode (Kas bertambah)
	//    If BalanceAmount < 0 (Reimbursement): Cr. SettlementAccountCode (Kas berkurang)
	var ledgerLines []LedgerLine
	for _, l := range ca.SettlementLines {
		desc := l.Description
		if l.ReceiptNo != "" {
			desc += fmt.Sprintf(" (Nota: %s)", l.ReceiptNo)
		}
		ledgerLines = append(ledgerLines, LedgerLine{
			AccountCode: l.AccountCode,
			Debit:       l.Amount,
			Description: desc,
		})
	}

	ledgerLines = append(ledgerLines, LedgerLine{
		AccountCode: AccountEmployeeAdvance,
		Credit:      ca.KBSAmount,
		Description: fmt.Sprintf("Penyelesaian Kas Bon %s", ca.KBSNumber),
	})

	settleAcc := ca.SettlementAccountCode
	if settleAcc == "" {
		settleAcc = AccountCash
	}

	if ca.PJBalanceAmount > 0 {
		// Excess returned to company: Debit Cash/Bank
		ledgerLines = append(ledgerLines, LedgerLine{
			AccountCode: settleAcc,
			Debit:       ca.PJBalanceAmount,
			Description: fmt.Sprintf("Pengembalian Sisa Kas Bon %s", ca.KBSNumber),
		})
	} else if ca.PJBalanceAmount < 0 {
		// Deficit reimbursed to employee: Credit Cash/Bank
		deficit := math.Abs(ca.PJBalanceAmount)
		ledgerLines = append(ledgerLines, LedgerLine{
			AccountCode: settleAcc,
			Credit:      deficit,
			Description: fmt.Sprintf("Penggantian Kekurangan Kas Bon %s", ca.KBSNumber),
		})
	}

	memo := fmt.Sprintf("Pertanggungjawaban KBS %s (%s) - %s", ca.KBSNumber, bkkNumber, ca.EmployeeName)
	sourceDoc := CashAdvanceSourceSettlementPrefix + ca.ID.String()
	poster := NewLedgerPoster(uc.repo)
	if err := poster.PostEntryOn(ctx, date, sourceDoc, memo, ledgerLines); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memposting jurnal penyelesaian kas bon")
	}

	je, _ := uc.repo.GetJournalEntryBySourceDoc(ctx, sourceDoc)
	var jID *uuid.UUID
	if je != nil {
		jID = &je.ID
	}

	ca.SettlementJournalID = jID
	ca.PJStatus = "settled"
	ca.ApprovedPJByEmail = actor.EmailFrom(ctx)

	if err := uc.repo.UpdateCashAdvance(ctx, ca); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memperbarui status penyelesaian Kas Bon")
	}
	return ca, nil
}

func (uc *financeUseCase) RejectSettlement(ctx context.Context, id uuid.UUID, reason string) (*domain.CashAdvance, error) {
	ca, err := uc.repo.GetCashAdvanceByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat pertanggungjawaban Kas Bon")
	}
	if ca == nil {
		return nil, apperrors.NewNotFound("Pertanggungjawaban Kas Bon tidak ditemukan")
	}
	if ca.PJStatus != "pending" {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("Pertanggungjawaban tidak dalam status menunggu approval (status saat ini: %s)", ca.PJStatus))
	}

	ca.PJStatus = "revision"
	ca.PJRejectReason = strings.TrimSpace(reason)

	if err := uc.repo.UpdateCashAdvance(ctx, ca); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal menolak pertanggungjawaban untuk perbaikan")
	}
	return ca, nil
}

func (uc *financeUseCase) ListCashAdvances(ctx context.Context, status, from, to string) ([]domain.CashAdvance, error) {
	return uc.repo.ListCashAdvances(ctx, status, from, to)
}

func (uc *financeUseCase) GetCashAdvance(ctx context.Context, id uuid.UUID) (*domain.CashAdvance, error) {
	ca, err := uc.repo.GetCashAdvanceByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat data Kas Bon")
	}
	if ca == nil {
		return nil, apperrors.NewNotFound("Data Kas Bon tidak ditemukan")
	}
	return ca, nil
}

func (uc *financeUseCase) GetOutstandingCashAdvancesReport(ctx context.Context) ([]OutstandingCashAdvanceReportItem, error) {
	all, err := uc.repo.ListCashAdvances(ctx, "", "", "")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat laporan Kas Bon terbuka")
	}

	now := time.Now()
	var out []OutstandingCashAdvanceReportItem
	for _, ca := range all {
		if ca.CRStatus == "rejected" || ca.PJStatus == "settled" {
			continue
		}

		baseDateStr := ca.DisburseDate
		if baseDateStr == "" {
			baseDateStr = ca.CRDate
		}
		agingDays := 0
		if t, err := time.Parse("2006-01-02", baseDateStr); err == nil {
			agingDays = int(now.Sub(t).Hours() / 24)
			if agingDays < 0 {
				agingDays = 0
			}
		}

		statusDesc := "Cash Request Pending"
		if ca.CRStatus == "approved" && ca.KBSStatus == "draft" {
			statusDesc = "Menunggu Form Kas Bon"
		} else if ca.KBSStatus == "pending" {
			statusDesc = "Form Kas Bon Pending Approval"
		} else if ca.KBSStatus == "revision" {
			statusDesc = "Revisi Form Kas Bon"
		} else if ca.KBSStatus == "approved" && ca.DisburseDate == "" {
			statusDesc = "Disetujui - Menunggu Pencairan"
		} else if ca.DisburseDate != "" && (ca.PJStatus == "none" || ca.PJStatus == "") {
			statusDesc = "Dana Dicairkan - Menunggu PJ"
		} else if ca.PJStatus == "pending" {
			statusDesc = "Pertanggungjawaban Pending Approval"
		} else if ca.PJStatus == "revision" {
			statusDesc = "Revisi Bukti Pertanggungjawaban"
		}

		out = append(out, OutstandingCashAdvanceReportItem{
			ID:                      ca.ID,
			CRNumber:                ca.CRNumber,
			KBSNumber:               ca.KBSNumber,
			EmployeeName:            ca.EmployeeName,
			Department:              ca.Department,
			Purpose:                 ca.Purpose,
			CRDate:                  ca.CRDate,
			DisburseDate:            ca.DisburseDate,
			KBSDueDate:              ca.KBSDueDate,
			KBSAmount:               ca.KBSAmount,
			AgingDays:               agingDays,
			Status:                  statusDesc,
			DisburseCashAccountCode: ca.DisburseCashAccountCode,
		})
	}
	return out, nil
}

func (uc *financeUseCase) GetCashAdvanceDifferencesReport(ctx context.Context, from, to string) ([]CashAdvanceDifferenceReportItem, error) {
	all, err := uc.repo.ListCashAdvances(ctx, "", from, to)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat laporan selisih kas bon")
	}

	var out []CashAdvanceDifferenceReportItem
	for _, ca := range all {
		// Include advances that have settlement recorded or completed
		if ca.PJDate == "" && ca.PJTotalRealized == 0 {
			continue
		}
		out = append(out, CashAdvanceDifferenceReportItem{
			ID:                    ca.ID,
			KBSNumber:             ca.KBSNumber,
			CRNumber:              ca.CRNumber,
			EmployeeName:          ca.EmployeeName,
			Department:            ca.Department,
			Purpose:               ca.Purpose,
			KBSAmount:             ca.KBSAmount,
			PJTotalRealized:       ca.PJTotalRealized,
			BalanceAmount:         ca.PJBalanceAmount,
			SettlementAction:      ca.SettlementAction,
			SettlementAccountCode: ca.SettlementAccountCode,
			PJStatus:              ca.PJStatus,
			BKKNumber:             ca.BKKNumber,
			SettledDate:           ca.PJDate,
		})
	}
	return out, nil
}
