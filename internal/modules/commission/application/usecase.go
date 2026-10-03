package application

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/divinecoid/one-backend/internal/modules/commission/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

var validScopes = map[string]bool{
	"flat":   true,
	"tiered": true,
}

var validRecordStatuses = map[string]bool{
	"pending":  true,
	"approved": true,
	"paid":     true,
}

func encodeTiers(tiers []CommissionTierDTO) string {
	if len(tiers) == 0 {
		return ""
	}
	b, err := json.Marshal(tiers)
	if err != nil {
		return ""
	}
	return string(b)
}

func decodeTiers(raw string) []CommissionTierDTO {
	if raw == "" {
		return nil
	}
	var tiers []CommissionTierDTO
	if err := json.Unmarshal([]byte(raw), &tiers); err != nil {
		return nil
	}
	return tiers
}

type CommissionUseCase interface {
	CreateRule(ctx context.Context, dto CreateCommissionRuleDTO) (*CommissionRuleResponseDTO, error)
	GetRuleByID(ctx context.Context, id uuid.UUID) (*CommissionRuleResponseDTO, error)
	ListRules(ctx context.Context, query types.PaginationQuery) ([]CommissionRuleResponseDTO, types.PaginationMeta, error)

	CalculateAndCreateRecord(ctx context.Context, dto CreateCommissionRecordDTO) (*CommissionRecordResponseDTO, error)
	GetRecordByID(ctx context.Context, id uuid.UUID) (*CommissionRecordResponseDTO, error)
	ListRecords(ctx context.Context, query types.PaginationQuery) ([]CommissionRecordResponseDTO, types.PaginationMeta, error)
	ApproveRecord(ctx context.Context, id uuid.UUID) (*CommissionRecordResponseDTO, error)
	MarkRecordPaid(ctx context.Context, id uuid.UUID) (*CommissionRecordResponseDTO, error)

	GetSummary(ctx context.Context, period string) (*CommissionSummaryResponseDTO, error)

	SeedInitialData(ctx context.Context) error
}

type commissionUseCase struct {
	repo domain.CommissionRepository
}

func NewCommissionUseCase(repo domain.CommissionRepository) CommissionUseCase {
	return &commissionUseCase{repo: repo}
}

func (uc *commissionUseCase) validateTiers(tiers []CommissionTierDTO) error {
	for _, t := range tiers {
		if t.RatePercent < 0 || t.RatePercent > 100 {
			return apperrors.NewBadRequest("Tier rate percent must be between 0 and 100")
		}
		if t.MaxAmount != 0 && t.MaxAmount < t.MinAmount {
			return apperrors.NewBadRequest("Tier max amount must not be less than min amount")
		}
	}
	return nil
}

func (uc *commissionUseCase) CreateRule(ctx context.Context, dto CreateCommissionRuleDTO) (*CommissionRuleResponseDTO, error) {
	if dto.Name == "" {
		return nil, apperrors.NewBadRequest("Rule name is required")
	}
	if !validScopes[dto.Scope] {
		return nil, apperrors.NewBadRequest("Scope must be one of flat, tiered")
	}
	if dto.AppliesTo == "" {
		dto.AppliesTo = "all_products"
	}

	if dto.Scope == "flat" {
		if dto.RatePercent < 0 || dto.RatePercent > 100 {
			return nil, apperrors.NewBadRequest("Rate percent must be between 0 and 100")
		}
	} else {
		if len(dto.Tiers) == 0 {
			return nil, apperrors.NewBadRequest("Tiered rules require at least one tier")
		}
		if err := uc.validateTiers(dto.Tiers); err != nil {
			return nil, err
		}
	}

	rule := &domain.CommissionRule{
		CompanyID:   dto.CompanyID,
		Name:        dto.Name,
		Scope:       dto.Scope,
		RatePercent: dto.RatePercent,
		Tiers:       encodeTiers(dto.Tiers),
		AppliesTo:   dto.AppliesTo,
		IsActive:    true,
	}

	if err := uc.repo.CreateRule(ctx, rule); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create commission rule")
	}
	return ToCommissionRuleResponse(rule), nil
}

func (uc *commissionUseCase) GetRuleByID(ctx context.Context, id uuid.UUID) (*CommissionRuleResponseDTO, error) {
	rule, err := uc.repo.GetRuleByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get commission rule")
	}
	if rule == nil {
		return nil, apperrors.NewNotFound("Commission rule not found")
	}
	return ToCommissionRuleResponse(rule), nil
}

func (uc *commissionUseCase) ListRules(ctx context.Context, query types.PaginationQuery) ([]CommissionRuleResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	rules, total, err := uc.repo.ListRules(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list commission rules")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToCommissionRuleResponseList(rules), meta, nil
}

func calculateCommission(rule *domain.CommissionRule, amount float64) (float64, error) {
	if rule.Scope == "flat" {
		return amount * rule.RatePercent / 100, nil
	}
	tiers := decodeTiers(rule.Tiers)
	for _, t := range tiers {
		if amount >= t.MinAmount && (t.MaxAmount == 0 || amount <= t.MaxAmount) {
			return amount * t.RatePercent / 100, nil
		}
	}
	return 0, apperrors.NewBadRequest("No matching commission tier found for the given amount")
}

func (uc *commissionUseCase) CalculateAndCreateRecord(ctx context.Context, dto CreateCommissionRecordDTO) (*CommissionRecordResponseDTO, error) {
	if dto.SalespersonName == "" {
		return nil, apperrors.NewBadRequest("Salesperson name is required")
	}
	if dto.SalesOrderAmount < 0 {
		return nil, apperrors.NewBadRequest("Sales order amount cannot be negative")
	}
	if dto.Period == "" {
		return nil, apperrors.NewBadRequest("Period is required (format YYYY-MM)")
	}

	rule, err := uc.repo.GetRuleByID(ctx, dto.CommissionRuleID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get commission rule")
	}
	if rule == nil {
		return nil, apperrors.NewNotFound("Commission rule not found")
	}
	if !rule.IsActive {
		return nil, apperrors.NewBadRequest("Commission rule is not active")
	}

	amount, err := calculateCommission(rule, dto.SalesOrderAmount)
	if err != nil {
		return nil, err
	}

	rec := &domain.CommissionRecord{
		CompanyID:                  dto.CompanyID,
		SalespersonName:            dto.SalespersonName,
		SalesOrderID:               dto.SalesOrderID,
		SalesOrderNumber:           dto.SalesOrderNumber,
		SalesOrderAmount:           dto.SalesOrderAmount,
		CommissionRuleID:           &rule.ID,
		CalculatedCommissionAmount: amount,
		Status:                     "pending",
		Period:                     dto.Period,
	}

	if err := uc.repo.CreateRecord(ctx, rec); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create commission record")
	}
	return ToCommissionRecordResponse(rec), nil
}

func (uc *commissionUseCase) GetRecordByID(ctx context.Context, id uuid.UUID) (*CommissionRecordResponseDTO, error) {
	rec, err := uc.repo.GetRecordByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get commission record")
	}
	if rec == nil {
		return nil, apperrors.NewNotFound("Commission record not found")
	}
	return ToCommissionRecordResponse(rec), nil
}

func (uc *commissionUseCase) ListRecords(ctx context.Context, query types.PaginationQuery) ([]CommissionRecordResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	records, total, err := uc.repo.ListRecords(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list commission records")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToCommissionRecordResponseList(records), meta, nil
}

func (uc *commissionUseCase) transitionStatus(ctx context.Context, id uuid.UUID, from, to string) (*CommissionRecordResponseDTO, error) {
	rec, err := uc.repo.GetRecordByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get commission record")
	}
	if rec == nil {
		return nil, apperrors.NewNotFound("Commission record not found")
	}
	if rec.Status != from {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("Commission record must be %s to transition to %s", from, to))
	}
	rec.Status = to
	if err := uc.repo.UpdateRecord(ctx, rec); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update commission record")
	}
	return ToCommissionRecordResponse(rec), nil
}

func (uc *commissionUseCase) ApproveRecord(ctx context.Context, id uuid.UUID) (*CommissionRecordResponseDTO, error) {
	return uc.transitionStatus(ctx, id, "pending", "approved")
}

func (uc *commissionUseCase) MarkRecordPaid(ctx context.Context, id uuid.UUID) (*CommissionRecordResponseDTO, error) {
	return uc.transitionStatus(ctx, id, "approved", "paid")
}

func (uc *commissionUseCase) GetSummary(ctx context.Context, period string) (*CommissionSummaryResponseDTO, error) {
	if period == "" {
		return nil, apperrors.NewBadRequest("Period is required (format YYYY-MM)")
	}
	records, err := uc.repo.ListRecordsByPeriod(ctx, period)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list commission records")
	}

	totals := map[string]*CommissionSummaryItemDTO{}
	var order []string
	var totalCommission float64
	for _, r := range records {
		item, ok := totals[r.SalespersonName]
		if !ok {
			item = &CommissionSummaryItemDTO{SalespersonName: r.SalespersonName}
			totals[r.SalespersonName] = item
			order = append(order, r.SalespersonName)
		}
		item.RecordCount++
		item.TotalCommission += r.CalculatedCommissionAmount
		totalCommission += r.CalculatedCommissionAmount
	}

	bySalesperson := make([]CommissionSummaryItemDTO, 0, len(order))
	for _, name := range order {
		bySalesperson = append(bySalesperson, *totals[name])
	}

	return &CommissionSummaryResponseDTO{
		Period:          period,
		TotalCommission: totalCommission,
		BySalesperson:   bySalesperson,
	}, nil
}

// SeedInitialData populates a default commission rule and a few sample records on first boot
func (uc *commissionUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.CountRules(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	rule, err := uc.CreateRule(ctx, CreateCommissionRuleDTO{
		Name:        "Standard Flat Commission",
		Scope:       "flat",
		RatePercent: 5,
		AppliesTo:   "all_products",
	})
	if err != nil {
		return err
	}

	period := "2026-09"
	samples := []CreateCommissionRecordDTO{
		{
			SalespersonName:  "Rizky Ramadhan",
			SalesOrderNumber: "SO-2026-0814",
			SalesOrderAmount: 450000000,
			CommissionRuleID: rule.ID,
			Period:           period,
		},
		{
			SalespersonName:  "Rizky Ramadhan",
			SalesOrderNumber: "SO-2026-0820",
			SalesOrderAmount: 185000000,
			CommissionRuleID: rule.ID,
			Period:           period,
		},
		{
			SalespersonName:  "Budi Santoso",
			SalesOrderNumber: "POS-260901",
			SalesOrderAmount: 35000000,
			CommissionRuleID: rule.ID,
			Period:           period,
		},
	}
	for _, s := range samples {
		if _, err := uc.CalculateAndCreateRecord(ctx, s); err != nil {
			continue
		}
	}
	return nil
}
