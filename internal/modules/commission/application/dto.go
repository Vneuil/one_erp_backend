package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/commission/domain"
	"github.com/google/uuid"
)

type CommissionTierDTO struct {
	MinAmount   float64 `json:"minAmount"`
	MaxAmount   float64 `json:"maxAmount"`
	RatePercent float64 `json:"ratePercent"`
}

type CreateCommissionRuleDTO struct {
	Name        string              `json:"name"`
	Scope       string              `json:"scope"`
	RatePercent float64             `json:"ratePercent"`
	Tiers       []CommissionTierDTO `json:"tiers,omitempty"`
	AppliesTo   string              `json:"appliesTo"`
	CompanyID   *uuid.UUID          `json:"companyId,omitempty"`
}

type CommissionRuleResponseDTO struct {
	ID          uuid.UUID           `json:"id"`
	CompanyID   *uuid.UUID          `json:"companyId,omitempty"`
	Name        string              `json:"name"`
	Scope       string              `json:"scope"`
	RatePercent float64             `json:"ratePercent"`
	Tiers       []CommissionTierDTO `json:"tiers,omitempty"`
	AppliesTo   string              `json:"appliesTo"`
	IsActive    bool                `json:"isActive"`
	CreatedAt   time.Time           `json:"createdAt"`
}

type CreateCommissionRecordDTO struct {
	SalespersonName  string     `json:"salespersonName"`
	SalesOrderID     *uuid.UUID `json:"salesOrderId,omitempty"`
	SalesOrderNumber string     `json:"salesOrderNumber"`
	SalesOrderAmount float64    `json:"salesOrderAmount"`
	CommissionRuleID uuid.UUID  `json:"commissionRuleId"`
	Period           string     `json:"period"`
	CompanyID        *uuid.UUID `json:"companyId,omitempty"`
}

type CommissionRecordResponseDTO struct {
	ID                         uuid.UUID  `json:"id"`
	CompanyID                  *uuid.UUID `json:"companyId,omitempty"`
	SalespersonName            string     `json:"salespersonName"`
	SalesOrderID               *uuid.UUID `json:"salesOrderId,omitempty"`
	SalesOrderNumber           string     `json:"salesOrderNumber"`
	SalesOrderAmount           float64    `json:"salesOrderAmount"`
	CommissionRuleID           *uuid.UUID `json:"commissionRuleId,omitempty"`
	CalculatedCommissionAmount float64    `json:"calculatedCommissionAmount"`
	Status                     string     `json:"status"`
	Period                     string     `json:"period"`
	CreatedAt                  time.Time  `json:"createdAt"`
}

type CommissionSummaryItemDTO struct {
	SalespersonName string  `json:"salespersonName"`
	RecordCount     int     `json:"recordCount"`
	TotalCommission float64 `json:"totalCommission"`
}

type CommissionSummaryResponseDTO struct {
	Period          string                     `json:"period"`
	TotalCommission float64                    `json:"totalCommission"`
	BySalesperson   []CommissionSummaryItemDTO `json:"bySalesperson"`
}

func ToCommissionRuleResponse(r *domain.CommissionRule) *CommissionRuleResponseDTO {
	if r == nil {
		return nil
	}
	return &CommissionRuleResponseDTO{
		ID:          r.ID,
		CompanyID:   r.CompanyID,
		Name:        r.Name,
		Scope:       r.Scope,
		RatePercent: r.RatePercent,
		Tiers:       decodeTiers(r.Tiers),
		AppliesTo:   r.AppliesTo,
		IsActive:    r.IsActive,
		CreatedAt:   r.CreatedAt,
	}
}

func ToCommissionRuleResponseList(rules []domain.CommissionRule) []CommissionRuleResponseDTO {
	result := make([]CommissionRuleResponseDTO, len(rules))
	for i, r := range rules {
		result[i] = *ToCommissionRuleResponse(&r)
	}
	return result
}

func ToCommissionRecordResponse(r *domain.CommissionRecord) *CommissionRecordResponseDTO {
	if r == nil {
		return nil
	}
	return &CommissionRecordResponseDTO{
		ID:                         r.ID,
		CompanyID:                  r.CompanyID,
		SalespersonName:            r.SalespersonName,
		SalesOrderID:               r.SalesOrderID,
		SalesOrderNumber:           r.SalesOrderNumber,
		SalesOrderAmount:           r.SalesOrderAmount,
		CommissionRuleID:           r.CommissionRuleID,
		CalculatedCommissionAmount: r.CalculatedCommissionAmount,
		Status:                     r.Status,
		Period:                     r.Period,
		CreatedAt:                  r.CreatedAt,
	}
}

func ToCommissionRecordResponseList(records []domain.CommissionRecord) []CommissionRecordResponseDTO {
	result := make([]CommissionRecordResponseDTO, len(records))
	for i, r := range records {
		result[i] = *ToCommissionRecordResponse(&r)
	}
	return result
}
