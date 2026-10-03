package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/loyalty/domain"
	"github.com/google/uuid"
)

type LoyaltyConfigResponseDTO struct {
	RupiahPerPoint float64 `json:"rupiahPerPoint"`
}

type UpdateLoyaltyConfigDTO struct {
	RupiahPerPoint float64 `json:"rupiahPerPoint"`
}

type LoyaltyMemberResponseDTO struct {
	ID            uuid.UUID `json:"id"`
	MemberCode    string    `json:"memberCode"`
	PhoneNumber   string    `json:"phoneNumber"`
	Name          string    `json:"name"`
	PointsBalance int       `json:"pointsBalance"`
	CreatedAt     time.Time `json:"createdAt"`
}

func ToLoyaltyMemberResponse(m *domain.LoyaltyMember) *LoyaltyMemberResponseDTO {
	if m == nil {
		return nil
	}
	return &LoyaltyMemberResponseDTO{
		ID:            m.ID,
		MemberCode:    m.MemberCode,
		PhoneNumber:   m.PhoneNumber,
		Name:          m.Name,
		PointsBalance: m.PointsBalance,
		CreatedAt:     m.CreatedAt,
	}
}

func ToLoyaltyMemberResponseList(members []domain.LoyaltyMember) []LoyaltyMemberResponseDTO {
	out := make([]LoyaltyMemberResponseDTO, 0, len(members))
	for i := range members {
		out = append(out, *ToLoyaltyMemberResponse(&members[i]))
	}
	return out
}

type EnrollMemberDTO struct {
	PhoneNumber string `json:"phoneNumber"`
	Name        string `json:"name"`
}

// EarnPointsDTO is used internally (e.g. by the POS module) to award points
// for a completed sale. If the phone number has no existing member, one is
// enrolled automatically.
type EarnPointsDTO struct {
	PhoneNumber      string
	Name             string
	SpendAmount      float64
	SalesOrderID     *uuid.UUID
	SalesOrderNumber string
}

type EarnPointsResultDTO struct {
	Member        LoyaltyMemberResponseDTO `json:"member"`
	PointsEarned  int                      `json:"pointsEarned"`
	NewlyEnrolled bool                     `json:"newlyEnrolled"`
}

type LoyaltyTransactionResponseDTO struct {
	ID               uuid.UUID `json:"id"`
	Type             string    `json:"type"`
	Points           int       `json:"points"`
	BalanceAfter     int       `json:"balanceAfter"`
	Description      string    `json:"description,omitempty"`
	SalesOrderNumber string    `json:"salesOrderNumber,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
}

func ToLoyaltyTransactionResponseList(txs []domain.LoyaltyTransaction) []LoyaltyTransactionResponseDTO {
	out := make([]LoyaltyTransactionResponseDTO, 0, len(txs))
	for _, t := range txs {
		out = append(out, LoyaltyTransactionResponseDTO{
			ID:               t.ID,
			Type:             t.Type,
			Points:           t.Points,
			BalanceAfter:     t.BalanceAfter,
			Description:      t.Description,
			SalesOrderNumber: t.SalesOrderNumber,
			CreatedAt:        t.CreatedAt,
		})
	}
	return out
}

type MemberCardDTO struct {
	Member       LoyaltyMemberResponseDTO        `json:"member"`
	Transactions []LoyaltyTransactionResponseDTO `json:"transactions"`
}
