package application

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/loyalty/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
)

const defaultRupiahPerPoint = 1000

type LoyaltyUseCase interface {
	GetConfig(ctx context.Context) (*LoyaltyConfigResponseDTO, error)
	UpdateConfig(ctx context.Context, dto UpdateLoyaltyConfigDTO) (*LoyaltyConfigResponseDTO, error)

	LookupMemberByPhone(ctx context.Context, phone string) (*MemberCardDTO, error)
	EnrollMember(ctx context.Context, dto EnrollMemberDTO) (*LoyaltyMemberResponseDTO, error)
	ListMembers(ctx context.Context, query types.PaginationQuery) ([]LoyaltyMemberResponseDTO, types.PaginationMeta, error)

	// EarnPoints awards points for a completed sale, auto-enrolling the
	// phone number as a new member if it isn't one yet.
	EarnPoints(ctx context.Context, dto EarnPointsDTO) (*EarnPointsResultDTO, error)
}

type loyaltyUseCase struct {
	repo domain.LoyaltyRepository
}

func NewLoyaltyUseCase(repo domain.LoyaltyRepository) LoyaltyUseCase {
	return &loyaltyUseCase{repo: repo}
}

func normalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	if strings.HasPrefix(phone, "+") {
		phone = phone[1:]
	}
	if strings.HasPrefix(phone, "0") {
		phone = "62" + phone[1:]
	}
	return phone
}

func (uc *loyaltyUseCase) GetConfig(ctx context.Context) (*LoyaltyConfigResponseDTO, error) {
	cfg, err := uc.repo.GetConfig(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load loyalty config")
	}
	if cfg == nil {
		return &LoyaltyConfigResponseDTO{RupiahPerPoint: defaultRupiahPerPoint}, nil
	}
	return &LoyaltyConfigResponseDTO{RupiahPerPoint: cfg.RupiahPerPoint}, nil
}

func (uc *loyaltyUseCase) UpdateConfig(ctx context.Context, dto UpdateLoyaltyConfigDTO) (*LoyaltyConfigResponseDTO, error) {
	if dto.RupiahPerPoint <= 0 {
		return nil, apperrors.NewBadRequest("rupiahPerPoint must be greater than 0")
	}

	cfg, err := uc.repo.GetConfig(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load loyalty config")
	}
	if cfg == nil {
		cfg = &domain.LoyaltyConfig{}
	}
	cfg.RupiahPerPoint = dto.RupiahPerPoint

	if err := uc.repo.UpsertConfig(ctx, cfg); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save loyalty config")
	}

	return &LoyaltyConfigResponseDTO{RupiahPerPoint: cfg.RupiahPerPoint}, nil
}

func (uc *loyaltyUseCase) LookupMemberByPhone(ctx context.Context, phone string) (*MemberCardDTO, error) {
	phone = normalizePhone(phone)
	if phone == "" {
		return nil, apperrors.NewBadRequest("Phone number is required")
	}

	member, err := uc.repo.GetMemberByPhone(ctx, phone)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to look up member")
	}
	if member == nil {
		return nil, apperrors.NewNotFound("No membership card found for this phone number")
	}

	txs, err := uc.repo.ListTransactionsByMember(ctx, member.ID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load point history")
	}

	return &MemberCardDTO{
		Member:       *ToLoyaltyMemberResponse(member),
		Transactions: ToLoyaltyTransactionResponseList(txs),
	}, nil
}

func (uc *loyaltyUseCase) generateMemberCode(ctx context.Context) (string, error) {
	year := time.Now().Format("06")
	prefix := fmt.Sprintf("MBR%s", year)
	count, err := uc.repo.CountMembersWithCodePrefix(ctx, prefix)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%05d", prefix, count+1), nil
}

func (uc *loyaltyUseCase) EnrollMember(ctx context.Context, dto EnrollMemberDTO) (*LoyaltyMemberResponseDTO, error) {
	phone := normalizePhone(dto.PhoneNumber)
	if phone == "" {
		return nil, apperrors.NewBadRequest("Phone number is required")
	}

	existing, err := uc.repo.GetMemberByPhone(ctx, phone)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to check existing member")
	}
	if existing != nil {
		return nil, apperrors.NewBadRequest("A membership card already exists for this phone number")
	}

	code, err := uc.generateMemberCode(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to generate member code")
	}

	name := dto.Name
	if name == "" {
		name = "Member"
	}

	member := &domain.LoyaltyMember{
		MemberCode:  code,
		PhoneNumber: phone,
		Name:        name,
	}
	if err := uc.repo.CreateMember(ctx, member); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to enroll member")
	}

	return ToLoyaltyMemberResponse(member), nil
}

func (uc *loyaltyUseCase) ListMembers(ctx context.Context, query types.PaginationQuery) ([]LoyaltyMemberResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	members, total, err := uc.repo.ListMembers(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list members")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToLoyaltyMemberResponseList(members), meta, nil
}

func (uc *loyaltyUseCase) EarnPoints(ctx context.Context, dto EarnPointsDTO) (*EarnPointsResultDTO, error) {
	phone := normalizePhone(dto.PhoneNumber)
	if phone == "" {
		return nil, apperrors.NewBadRequest("Phone number is required")
	}
	if dto.SpendAmount <= 0 {
		return nil, apperrors.NewBadRequest("Spend amount must be greater than 0")
	}

	cfg, err := uc.repo.GetConfig(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load loyalty config")
	}
	rupiahPerPoint := float64(defaultRupiahPerPoint)
	if cfg != nil && cfg.RupiahPerPoint > 0 {
		rupiahPerPoint = cfg.RupiahPerPoint
	}

	member, err := uc.repo.GetMemberByPhone(ctx, phone)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to look up member")
	}

	newlyEnrolled := false
	if member == nil {
		code, err := uc.generateMemberCode(ctx)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to generate member code")
		}
		name := dto.Name
		if name == "" {
			name = "Member"
		}
		member = &domain.LoyaltyMember{
			MemberCode:  code,
			PhoneNumber: phone,
			Name:        name,
		}
		if err := uc.repo.CreateMember(ctx, member); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to enroll member")
		}
		newlyEnrolled = true
	}

	pointsEarned := int(math.Floor(dto.SpendAmount / rupiahPerPoint))
	newBalance := member.PointsBalance + pointsEarned

	if pointsEarned > 0 {
		if err := uc.repo.UpdateMemberBalance(ctx, member.ID, newBalance); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to update points balance")
		}
		member.PointsBalance = newBalance

		txn := &domain.LoyaltyTransaction{
			MemberID:         member.ID,
			Type:             "earn",
			Points:           pointsEarned,
			BalanceAfter:     newBalance,
			Description:      "Poin dari transaksi penjualan",
			SalesOrderID:     dto.SalesOrderID,
			SalesOrderNumber: dto.SalesOrderNumber,
		}
		if err := uc.repo.CreateTransaction(ctx, txn); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to record point transaction")
		}
	}

	return &EarnPointsResultDTO{
		Member:        *ToLoyaltyMemberResponse(member),
		PointsEarned:  pointsEarned,
		NewlyEnrolled: newlyEnrolled,
	}, nil
}
