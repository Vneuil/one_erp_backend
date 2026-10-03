package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/banking/domain"
	"github.com/google/uuid"
)

// Bank accounts

type CreateBankAccountDTO struct {
	BankName          string     `json:"bankName"`
	AccountNumber     string     `json:"accountNumber"`
	AccountHolder     string     `json:"accountHolder"`
	Currency          string     `json:"currency"`
	Branch            string     `json:"branch"`
	LinkedGLAccountID *uuid.UUID `json:"linkedGlAccountId,omitempty"`
	Status            string     `json:"status"`
	InitialBalance    float64    `json:"initialBalance"`
}

type BankAccountResponseDTO struct {
	ID                uuid.UUID  `json:"id"`
	BankName          string     `json:"bankName"`
	AccountNumber     string     `json:"accountNumber"`
	AccountHolder     string     `json:"accountHolder"`
	Currency          string     `json:"currency"`
	CurrentBalance    float64    `json:"currentBalance"`
	Branch            string     `json:"branch"`
	LinkedGLAccountID *uuid.UUID `json:"linkedGlAccountId,omitempty"`
	Status            string     `json:"status"`
	UnreconciledCount int64      `json:"unreconciledCount"`
	CreatedAt         time.Time  `json:"createdAt"`
}

func ToBankAccountResponse(a *domain.BankAccount, unreconciledCount int64) *BankAccountResponseDTO {
	if a == nil {
		return nil
	}
	return &BankAccountResponseDTO{
		ID:                a.ID,
		BankName:          a.BankName,
		AccountNumber:     a.AccountNumber,
		AccountHolder:     a.AccountHolder,
		Currency:          a.Currency,
		CurrentBalance:    a.CurrentBalance,
		Branch:            a.Branch,
		LinkedGLAccountID: a.LinkedGLAccountID,
		Status:            a.Status,
		UnreconciledCount: unreconciledCount,
		CreatedAt:         a.CreatedAt,
	}
}

// Statement lines

type CreateStatementLineDTO struct {
	TransactionDate string  `json:"transactionDate"`
	Description     string  `json:"description"`
	Amount          float64 `json:"amount"`
}

type ImportStatementLinesDTO struct {
	Lines []CreateStatementLineDTO `json:"lines"`
}

type ManualMatchDTO struct {
	JournalLineID uuid.UUID `json:"journalLineId"`
}

type StatementLineResponseDTO struct {
	ID                   uuid.UUID  `json:"id"`
	BankAccountID        uuid.UUID  `json:"bankAccountId"`
	TransactionDate      string     `json:"transactionDate"`
	Description          string     `json:"description"`
	Amount               float64    `json:"amount"`
	IsReconciled         bool       `json:"isReconciled"`
	ReconciledAt         *string    `json:"reconciledAt,omitempty"`
	MatchedJournalLineID *uuid.UUID `json:"matchedJournalLineId,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
}

func ToStatementLineResponse(l *domain.BankStatementLine) *StatementLineResponseDTO {
	if l == nil {
		return nil
	}
	return &StatementLineResponseDTO{
		ID:                   l.ID,
		BankAccountID:        l.BankAccountID,
		TransactionDate:      l.TransactionDate,
		Description:          l.Description,
		Amount:               l.Amount,
		IsReconciled:         l.IsReconciled,
		ReconciledAt:         l.ReconciledAt,
		MatchedJournalLineID: l.MatchedJournalLineID,
		CreatedAt:            l.CreatedAt,
	}
}

func ToStatementLineResponseList(lines []domain.BankStatementLine) []StatementLineResponseDTO {
	result := make([]StatementLineResponseDTO, len(lines))
	for i, l := range lines {
		result[i] = *ToStatementLineResponse(&l)
	}
	return result
}

// Reconciliation

type ReconcileSummaryDTO struct {
	MatchedCount      int `json:"matchedCount"`
	UnreconciledCount int `json:"unreconciledCount"`
}
