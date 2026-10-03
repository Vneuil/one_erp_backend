package application

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/reimbursement/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/sod"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type ReimbursementUseCase interface {
	CreateClaim(ctx context.Context, dto CreateClaimDTO) (*ClaimResponseDTO, error)
	ListClaims(ctx context.Context, query types.PaginationQuery) ([]ClaimResponseDTO, types.PaginationMeta, error)
	ApproveClaim(ctx context.Context, id uuid.UUID) (*ClaimResponseDTO, error)
	RejectClaim(ctx context.Context, id uuid.UUID) (*ClaimResponseDTO, error)
	MarkPaid(ctx context.Context, id uuid.UUID) (*ClaimResponseDTO, error)

	SeedInitialData(ctx context.Context) error
}

type reimbursementUseCase struct {
	repo   domain.ReimbursementRepository
	ledger financeApp.LedgerPoster // optional; nil disables auto-posting to the general ledger
}

// Option customises optional collaborators of the reimbursement use case.
type Option func(*reimbursementUseCase)

// WithLedger enables automatic journal posting when a claim is approved and paid.
func WithLedger(l financeApp.LedgerPoster) Option {
	return func(uc *reimbursementUseCase) { uc.ledger = l }
}

func NewReimbursementUseCase(repo domain.ReimbursementRepository, opts ...Option) ReimbursementUseCase {
	uc := &reimbursementUseCase{repo: repo}
	for _, o := range opts {
		o(uc)
	}
	return uc
}

// postLedger records an automatic journal entry. A ledger failure must not
// roll back the claim, so it is logged for later reconciliation.
func (uc *reimbursementUseCase) postLedger(ctx context.Context, sourceDoc, memo string, lines []financeApp.LedgerLine) {
	if uc.ledger == nil {
		return
	}
	if err := uc.ledger.PostEntry(ctx, sourceDoc, memo, lines); err != nil {
		slog.Warn("reimbursement: failed to post ledger entry", "sourceDoc", sourceDoc, "error", err)
	}
}

// generateClaimNo builds a sequential, timestamp-seeded claim number of the
// form "RMB-<year>-<0001>", derived from the current total claim count (+1)
// scoped to the caller's tenant DB. Numbers restart per tenant/company.
func (uc *reimbursementUseCase) generateClaimNo(ctx context.Context) (string, error) {
	count, err := uc.repo.CountClaims(ctx)
	if err != nil {
		return "", err
	}
	year := time.Now().Year()
	return fmt.Sprintf("RMB-%d-%04d", year, count+1), nil
}

func (uc *reimbursementUseCase) CreateClaim(ctx context.Context, dto CreateClaimDTO) (*ClaimResponseDTO, error) {
	name := strings.TrimSpace(dto.EmployeeName)
	if name == "" {
		return nil, apperrors.NewBadRequest("Employee Name is required")
	}
	if dto.Amount <= 0 {
		return nil, apperrors.NewBadRequest("Amount must be greater than zero")
	}

	category := dto.Category
	if category == "" {
		category = "Lainnya"
	}
	date := dto.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	claimNo, err := uc.generateClaimNo(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to generate claim number")
	}

	claim := &domain.ReimbursementClaim{
		ClaimNo:         claimNo,
		EmployeeName:    name,
		RequesterEmail:  actor.EmailFrom(ctx),
		Department:      dto.Department,
		Category:        category,
		Amount:          dto.Amount,
		Description:     dto.Description,
		Date:            date,
		ReceiptAttached: dto.ReceiptAttached,
		Status:          "pending_approval",
	}

	if err := uc.repo.CreateClaim(ctx, claim); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create reimbursement claim")
	}

	return ToClaimResponse(claim), nil
}

func (uc *reimbursementUseCase) ListClaims(ctx context.Context, query types.PaginationQuery) ([]ClaimResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	claims, total, err := uc.repo.ListClaims(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list reimbursement claims")
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToClaimResponseList(claims), meta, nil
}

func (uc *reimbursementUseCase) ApproveClaim(ctx context.Context, id uuid.UUID) (*ClaimResponseDTO, error) {
	claim, err := uc.getClaim(ctx, id)
	if err != nil {
		return nil, err
	}

	if claim.Status != "pending_approval" {
		return nil, apperrors.NewBadRequest("Only pending claims can be approved")
	}
	if err := sod.ForbidSelfApproval(ctx, claim.RequesterEmail); err != nil {
		return nil, err
	}

	claim.Status = "approved"
	if err := uc.repo.UpdateClaim(ctx, claim); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to approve claim")
	}

	e := ClaimApprovalLedgerEntry(claim)
	uc.postLedger(ctx, e.SourceDoc, e.Memo, e.Lines)
	return ToClaimResponse(claim), nil
}

func (uc *reimbursementUseCase) RejectClaim(ctx context.Context, id uuid.UUID) (*ClaimResponseDTO, error) {
	claim, err := uc.getClaim(ctx, id)
	if err != nil {
		return nil, err
	}

	if claim.Status != "pending_approval" {
		return nil, apperrors.NewBadRequest("Only pending claims can be rejected")
	}

	claim.Status = "rejected"
	if err := uc.repo.UpdateClaim(ctx, claim); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to reject claim")
	}
	return ToClaimResponse(claim), nil
}

func (uc *reimbursementUseCase) MarkPaid(ctx context.Context, id uuid.UUID) (*ClaimResponseDTO, error) {
	claim, err := uc.getClaim(ctx, id)
	if err != nil {
		return nil, err
	}

	if claim.Status != "approved" {
		return nil, apperrors.NewBadRequest("Only approved claims can be marked as paid")
	}
	if err := sod.ForbidSelfApproval(ctx, claim.RequesterEmail); err != nil {
		return nil, err
	}

	claim.Status = "paid"
	if err := uc.repo.UpdateClaim(ctx, claim); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to mark claim as paid")
	}

	e := ClaimPaymentLedgerEntry(claim)
	uc.postLedger(ctx, e.SourceDoc, e.Memo, e.Lines)
	return ToClaimResponse(claim), nil
}

func (uc *reimbursementUseCase) getClaim(ctx context.Context, id uuid.UUID) (*domain.ReimbursementClaim, error) {
	claim, err := uc.repo.GetClaimByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get claim")
	}
	if claim == nil {
		return nil, apperrors.NewNotFound("Reimbursement claim not found")
	}
	return claim, nil
}

func (uc *reimbursementUseCase) SeedInitialData(ctx context.Context) error {
	// No default seed data required for the reimbursement module.
	return nil
}
