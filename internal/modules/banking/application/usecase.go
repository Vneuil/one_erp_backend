package application

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/banking/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type BankingUseCase interface {
	// Bank accounts
	CreateBankAccount(ctx context.Context, dto CreateBankAccountDTO) (*BankAccountResponseDTO, error)
	GetBankAccountByID(ctx context.Context, id uuid.UUID) (*BankAccountResponseDTO, error)
	ListBankAccounts(ctx context.Context, query types.PaginationQuery) ([]BankAccountResponseDTO, types.PaginationMeta, error)

	// Statement lines
	ImportStatementLines(ctx context.Context, bankAccountID uuid.UUID, dto ImportStatementLinesDTO) ([]StatementLineResponseDTO, error)
	ListStatementLines(ctx context.Context, bankAccountID uuid.UUID, query types.PaginationQuery) ([]StatementLineResponseDTO, types.PaginationMeta, error)

	// Reconciliation
	AutoReconcile(ctx context.Context, bankAccountID uuid.UUID) (*ReconcileSummaryDTO, error)
	ManualMatch(ctx context.Context, statementLineID uuid.UUID, dto ManualMatchDTO) (*StatementLineResponseDTO, error)

	SeedInitialData(ctx context.Context) error
}

type bankingUseCase struct {
	repo domain.BankingRepository
}

func NewBankingUseCase(repo domain.BankingRepository) BankingUseCase {
	return &bankingUseCase{repo: repo}
}

// Bank accounts

func (uc *bankingUseCase) CreateBankAccount(ctx context.Context, dto CreateBankAccountDTO) (*BankAccountResponseDTO, error) {
	if dto.BankName == "" || dto.AccountNumber == "" {
		return nil, apperrors.NewBadRequest("Bank name and account number are required")
	}
	currency := dto.Currency
	if currency == "" {
		currency = "IDR"
	}
	status := dto.Status
	if status == "" {
		status = "active"
	}
	a := &domain.BankAccount{
		BankName:          dto.BankName,
		AccountNumber:     dto.AccountNumber,
		AccountHolder:     dto.AccountHolder,
		Currency:          currency,
		Branch:            dto.Branch,
		LinkedGLAccountID: dto.LinkedGLAccountID,
		Status:            status,
		CurrentBalance:    dto.InitialBalance,
	}
	if err := uc.repo.CreateBankAccount(ctx, a); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create bank account")
	}
	return ToBankAccountResponse(a, 0), nil
}

func (uc *bankingUseCase) GetBankAccountByID(ctx context.Context, id uuid.UUID) (*BankAccountResponseDTO, error) {
	a, err := uc.repo.GetBankAccountByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get bank account")
	}
	if a == nil {
		return nil, apperrors.NewNotFound("Bank account not found")
	}
	count, err := uc.repo.CountUnreconciledStatementLines(ctx, a.ID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to count unreconciled statement lines")
	}
	return ToBankAccountResponse(a, count), nil
}

func (uc *bankingUseCase) ListBankAccounts(ctx context.Context, query types.PaginationQuery) ([]BankAccountResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	accounts, total, err := uc.repo.ListBankAccounts(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list bank accounts")
	}
	result := make([]BankAccountResponseDTO, len(accounts))
	for i, a := range accounts {
		count, err := uc.repo.CountUnreconciledStatementLines(ctx, a.ID)
		if err != nil {
			return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to count unreconciled statement lines")
		}
		result[i] = *ToBankAccountResponse(&a, count)
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return result, meta, nil
}

// Statement lines

func (uc *bankingUseCase) ImportStatementLines(ctx context.Context, bankAccountID uuid.UUID, dto ImportStatementLinesDTO) ([]StatementLineResponseDTO, error) {
	a, err := uc.repo.GetBankAccountByID(ctx, bankAccountID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get bank account")
	}
	if a == nil {
		return nil, apperrors.NewNotFound("Bank account not found")
	}
	if len(dto.Lines) == 0 {
		return nil, apperrors.NewBadRequest("At least one statement line is required")
	}

	lines := make([]domain.BankStatementLine, len(dto.Lines))
	for i, l := range dto.Lines {
		if l.Amount == 0 {
			return nil, apperrors.NewBadRequest("Statement line amount cannot be zero")
		}
		date := l.TransactionDate
		if date == "" {
			date = time.Now().Format("2006-01-02")
		}
		lines[i] = domain.BankStatementLine{
			BankAccountID:   bankAccountID,
			TransactionDate: date,
			Description:     l.Description,
			Amount:          l.Amount,
			IsReconciled:    false,
		}
	}

	if err := uc.repo.CreateBankStatementLines(ctx, lines); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to import statement lines")
	}

	var delta float64
	for _, l := range lines {
		delta += l.Amount
	}
	a.CurrentBalance += delta
	if err := uc.repo.UpdateBankAccount(ctx, a); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update bank account balance")
	}

	return ToStatementLineResponseList(lines), nil
}

func (uc *bankingUseCase) ListStatementLines(ctx context.Context, bankAccountID uuid.UUID, query types.PaginationQuery) ([]StatementLineResponseDTO, types.PaginationMeta, error) {
	a, err := uc.repo.GetBankAccountByID(ctx, bankAccountID)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to get bank account")
	}
	if a == nil {
		return nil, types.PaginationMeta{}, apperrors.NewNotFound("Bank account not found")
	}
	query.SetDefaults()
	lines, total, err := uc.repo.ListBankStatementLines(ctx, bankAccountID, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list statement lines")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToStatementLineResponseList(lines), meta, nil
}

// Reconciliation

// AutoReconcile matches unreconciled statement lines against posted journal lines on the
// bank account's linked GL account: same amount (debit for outflow/negative, credit for
// inflow/positive), transaction date within +/-3 days, and not already matched on either side.
func (uc *bankingUseCase) AutoReconcile(ctx context.Context, bankAccountID uuid.UUID) (*ReconcileSummaryDTO, error) {
	a, err := uc.repo.GetBankAccountByID(ctx, bankAccountID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get bank account")
	}
	if a == nil {
		return nil, apperrors.NewNotFound("Bank account not found")
	}
	if a.LinkedGLAccountID == nil {
		return nil, apperrors.NewBadRequest("Bank account has no linked GL account to reconcile against")
	}

	unreconciled, err := uc.repo.ListUnreconciledStatementLines(ctx, bankAccountID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list unreconciled statement lines")
	}

	candidates, err := uc.repo.FindCandidateJournalLines(ctx, *a.LinkedGLAccountID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to find candidate journal lines")
	}

	usedJournalLines := make(map[uuid.UUID]bool)
	matched := 0

	for i := range unreconciled {
		line := &unreconciled[i]
		best := findBestCandidate(line, candidates, usedJournalLines)
		if best == nil {
			continue
		}
		usedJournalLines[best.ID] = true
		now := time.Now().Format("2006-01-02")
		line.IsReconciled = true
		line.ReconciledAt = &now
		matchedID := best.ID
		line.MatchedJournalLineID = &matchedID
		if err := uc.repo.UpdateBankStatementLine(ctx, line); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to update statement line")
		}
		matched++
	}

	remaining, err := uc.repo.CountUnreconciledStatementLines(ctx, bankAccountID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to count remaining unreconciled statement lines")
	}

	return &ReconcileSummaryDTO{MatchedCount: matched, UnreconciledCount: int(remaining)}, nil
}

func findBestCandidate(line *domain.BankStatementLine, candidates []domain.JournalLineCandidate, used map[uuid.UUID]bool) *domain.JournalLineCandidate {
	lineDate, err := time.Parse("2006-01-02", line.TransactionDate)
	if err != nil {
		return nil
	}
	var expectedAmount float64
	if line.Amount >= 0 {
		expectedAmount = line.Amount // credit/inflow -> debit on cash/bank GL account
	} else {
		expectedAmount = -line.Amount // debit/outflow -> credit on cash/bank GL account
	}

	for i := range candidates {
		c := &candidates[i]
		if used[c.ID] {
			continue
		}
		var candidateAmount float64
		if line.Amount >= 0 {
			candidateAmount = c.Debit
		} else {
			candidateAmount = c.Credit
		}
		if candidateAmount != expectedAmount {
			continue
		}
		cDate, err := time.Parse("2006-01-02", c.Date)
		if err != nil {
			continue
		}
		diff := lineDate.Sub(cDate).Hours() / 24
		if diff < -3 || diff > 3 {
			continue
		}
		return c
	}
	return nil
}

func (uc *bankingUseCase) ManualMatch(ctx context.Context, statementLineID uuid.UUID, dto ManualMatchDTO) (*StatementLineResponseDTO, error) {
	if dto.JournalLineID == uuid.Nil {
		return nil, apperrors.NewBadRequest("journalLineId is required")
	}
	line, err := uc.repo.GetBankStatementLineByID(ctx, statementLineID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get statement line")
	}
	if line == nil {
		return nil, apperrors.NewNotFound("Statement line not found")
	}
	if line.IsReconciled {
		return nil, apperrors.NewConflict("Statement line is already reconciled")
	}
	jl, err := uc.repo.GetJournalLineByID(ctx, dto.JournalLineID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get journal line")
	}
	if jl == nil {
		return nil, apperrors.NewNotFound("Journal line not found")
	}

	now := time.Now().Format("2006-01-02")
	line.IsReconciled = true
	line.ReconciledAt = &now
	matchedID := jl.ID
	line.MatchedJournalLineID = &matchedID
	if err := uc.repo.UpdateBankStatementLine(ctx, line); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update statement line")
	}
	return ToStatementLineResponse(line), nil
}

// SeedInitialData populates a couple of sample bank accounts with unreconciled statement lines
func (uc *bankingUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.CountBankAccounts(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	accounts := []CreateBankAccountDTO{
		{BankName: "Bank Central Asia (BCA)", AccountNumber: "8800123456", AccountHolder: "PT Sentosa Mandiri Solusindo", Currency: "IDR", Branch: "KCU Cikarang Industrial"},
		{BankName: "Bank Mandiri (Persero)", AccountNumber: "1420011223344", AccountHolder: "PT Sentosa Mandiri Solusindo", Currency: "IDR", Branch: "KC Jakarta Sudirman"},
	}

	now := time.Now()
	for i, dto := range accounts {
		created, err := uc.CreateBankAccount(ctx, dto)
		if err != nil {
			continue
		}
		lines := ImportStatementLinesDTO{
			Lines: []CreateStatementLineDTO{
				{TransactionDate: now.AddDate(0, 0, -1).Format("2006-01-02"), Description: "BIAYA ADM REKENING KORAN", Amount: -25000},
				{TransactionDate: now.AddDate(0, 0, -1).Format("2006-01-02"), Description: "PENDAPATAN BUNGA JASA GIRO", Amount: 420000},
			},
		}
		if i == 0 {
			lines.Lines = append(lines.Lines, CreateStatementLineDTO{
				TransactionDate: now.Format("2006-01-02"), Description: "TRSF E-BANKING CR DARI PT GRAHA KONSTRUKSI", Amount: 450000000,
			})
		}
		_, _ = uc.ImportStatementLines(ctx, created.ID, lines)
	}
	return nil
}
