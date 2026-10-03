package application

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/goal/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

var validGoalTypes = map[string]bool{
	"individual": true,
	"team":       true,
}

var validGoalStatuses = map[string]bool{
	"active":    true,
	"completed": true,
	"missed":    true,
	"cancelled": true,
}

type GoalsUseCase interface {
	CreateGoal(ctx context.Context, dto CreateGoalDTO) (*GoalResponseDTO, error)
	GetGoalByID(ctx context.Context, id uuid.UUID) (*GoalResponseDTO, error)
	ListGoals(ctx context.Context, query types.PaginationQuery) ([]GoalResponseDTO, types.PaginationMeta, error)
	UpdateGoal(ctx context.Context, id uuid.UUID, dto UpdateGoalDTO) (*GoalResponseDTO, error)
	CheckIn(ctx context.Context, id uuid.UUID, dto CheckInDTO) (*GoalResponseDTO, error)
	CompleteGoal(ctx context.Context, id uuid.UUID) (*GoalResponseDTO, error)
	GetSummary(ctx context.Context) (*domain.GoalSummary, error)

	SeedInitialData(ctx context.Context) error
}

type goalsUseCase struct {
	repo domain.GoalsRepository
}

func NewGoalsUseCase(repo domain.GoalsRepository) GoalsUseCase {
	return &goalsUseCase{repo: repo}
}

func computeProgress(target, current float64) float64 {
	if target <= 0 {
		return 0
	}
	progress := current / target * 100
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	return progress
}

func (uc *goalsUseCase) validateDates(periodStart, periodEnd string) error {
	start, err := time.Parse("2006-01-02", periodStart)
	if err != nil {
		return apperrors.NewBadRequest("Period start must be in YYYY-MM-DD format")
	}
	end, err := time.Parse("2006-01-02", periodEnd)
	if err != nil {
		return apperrors.NewBadRequest("Period end must be in YYYY-MM-DD format")
	}
	if !end.After(start) {
		return apperrors.NewBadRequest("Period end must be after period start")
	}
	return nil
}

func (uc *goalsUseCase) CreateGoal(ctx context.Context, dto CreateGoalDTO) (*GoalResponseDTO, error) {
	if dto.Title == "" {
		return nil, apperrors.NewBadRequest("Goal title is required")
	}
	if !validGoalTypes[dto.GoalType] {
		return nil, apperrors.NewBadRequest("Goal type must be one of individual, team")
	}
	if dto.OwnerName == "" {
		return nil, apperrors.NewBadRequest("Owner name is required")
	}
	if dto.Category == "" {
		return nil, apperrors.NewBadRequest("Category is required")
	}
	if dto.TargetValue <= 0 {
		return nil, apperrors.NewBadRequest("Target value must be positive")
	}
	if err := uc.validateDates(dto.PeriodStart, dto.PeriodEnd); err != nil {
		return nil, err
	}

	g := &domain.Goal{
		CompanyID:       dto.CompanyID,
		Title:           dto.Title,
		Description:     dto.Description,
		GoalType:        dto.GoalType,
		OwnerName:       dto.OwnerName,
		Category:        dto.Category,
		TargetValue:     dto.TargetValue,
		CurrentValue:    0,
		Unit:            dto.Unit,
		PeriodStart:     dto.PeriodStart,
		PeriodEnd:       dto.PeriodEnd,
		Status:          "active",
		ProgressPercent: 0,
	}

	if err := uc.repo.CreateGoal(ctx, g); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create goal")
	}
	return ToGoalResponse(g), nil
}

func (uc *goalsUseCase) GetGoalByID(ctx context.Context, id uuid.UUID) (*GoalResponseDTO, error) {
	g, err := uc.repo.GetGoalByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get goal")
	}
	if g == nil {
		return nil, apperrors.NewNotFound("Goal not found")
	}
	return ToGoalResponse(g), nil
}

func (uc *goalsUseCase) ListGoals(ctx context.Context, query types.PaginationQuery) ([]GoalResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	goals, total, err := uc.repo.ListGoals(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list goals")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToGoalResponseList(goals), meta, nil
}

func (uc *goalsUseCase) UpdateGoal(ctx context.Context, id uuid.UUID, dto UpdateGoalDTO) (*GoalResponseDTO, error) {
	g, err := uc.repo.GetGoalByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get goal")
	}
	if g == nil {
		return nil, apperrors.NewNotFound("Goal not found")
	}
	if dto.Title != "" {
		g.Title = dto.Title
	}
	if dto.Description != "" {
		g.Description = dto.Description
	}
	if dto.GoalType != "" {
		if !validGoalTypes[dto.GoalType] {
			return nil, apperrors.NewBadRequest("Goal type must be one of individual, team")
		}
		g.GoalType = dto.GoalType
	}
	if dto.OwnerName != "" {
		g.OwnerName = dto.OwnerName
	}
	if dto.Category != "" {
		g.Category = dto.Category
	}
	if dto.TargetValue != nil {
		if *dto.TargetValue <= 0 {
			return nil, apperrors.NewBadRequest("Target value must be positive")
		}
		g.TargetValue = *dto.TargetValue
	}
	if dto.Unit != "" {
		g.Unit = dto.Unit
	}
	if dto.PeriodStart != "" {
		g.PeriodStart = dto.PeriodStart
	}
	if dto.PeriodEnd != "" {
		g.PeriodEnd = dto.PeriodEnd
	}
	if err := uc.validateDates(g.PeriodStart, g.PeriodEnd); err != nil {
		return nil, err
	}
	g.ProgressPercent = computeProgress(g.TargetValue, g.CurrentValue)

	if err := uc.repo.UpdateGoal(ctx, g); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update goal")
	}
	return ToGoalResponse(g), nil
}

func (uc *goalsUseCase) CheckIn(ctx context.Context, id uuid.UUID, dto CheckInDTO) (*GoalResponseDTO, error) {
	g, err := uc.repo.GetGoalByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get goal")
	}
	if g == nil {
		return nil, apperrors.NewNotFound("Goal not found")
	}
	if dto.ValueRecorded < 0 {
		return nil, apperrors.NewBadRequest("Value recorded cannot be negative")
	}
	checkedInDate := dto.CheckedInDate
	if checkedInDate == "" {
		checkedInDate = time.Now().Format("2006-01-02")
	} else if _, err := time.Parse("2006-01-02", checkedInDate); err != nil {
		return nil, apperrors.NewBadRequest("Checked in date must be in YYYY-MM-DD format")
	}

	ci := &domain.GoalCheckIn{
		GoalID:        g.ID,
		ValueRecorded: dto.ValueRecorded,
		Note:          dto.Note,
		CheckedInDate: checkedInDate,
	}
	if err := uc.repo.CreateCheckIn(ctx, ci); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to record check-in")
	}

	g.CurrentValue = dto.ValueRecorded
	g.ProgressPercent = computeProgress(g.TargetValue, g.CurrentValue)
	if g.ProgressPercent >= 100 && g.Status == "active" {
		g.Status = "completed"
	}

	if err := uc.repo.UpdateGoal(ctx, g); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update goal after check-in")
	}
	return ToGoalResponse(g), nil
}

func (uc *goalsUseCase) CompleteGoal(ctx context.Context, id uuid.UUID) (*GoalResponseDTO, error) {
	g, err := uc.repo.GetGoalByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get goal")
	}
	if g == nil {
		return nil, apperrors.NewNotFound("Goal not found")
	}
	g.Status = "completed"
	g.ProgressPercent = 100
	if err := uc.repo.UpdateGoal(ctx, g); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to complete goal")
	}
	return ToGoalResponse(g), nil
}

func (uc *goalsUseCase) GetSummary(ctx context.Context) (*domain.GoalSummary, error) {
	goals, err := uc.repo.ListAllGoals(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list goals")
	}
	summary := &domain.GoalSummary{}
	var totalProgress float64
	for _, g := range goals {
		summary.TotalGoals++
		totalProgress += g.ProgressPercent
		switch g.Status {
		case "active":
			summary.ActiveGoals++
		case "completed":
			summary.CompletedGoals++
		case "missed":
			summary.MissedGoals++
		case "cancelled":
			summary.CancelledGoals++
		}
	}
	if summary.TotalGoals > 0 {
		summary.AverageProgress = totalProgress / float64(summary.TotalGoals)
	}
	return summary, nil
}

// SeedInitialData populates a few sample goals on first boot
func (uc *goalsUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.CountGoals(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	now := time.Now()
	seeds := []struct {
		dto      CreateGoalDTO
		checkIns []CheckInDTO
		complete bool
	}{
		{
			dto: CreateGoalDTO{
				Title:       "Peningkatan Penjualan Regional Q3",
				Description: "Target penjualan tim regional untuk kuartal berjalan",
				GoalType:    "team",
				OwnerName:   "Tim Sales Jakarta",
				Category:    "Sales Revenue",
				TargetValue: 1500000000,
				Unit:        "Rp",
				PeriodStart: now.AddDate(0, -2, 0).Format("2006-01-02"),
				PeriodEnd:   now.AddDate(0, 1, 0).Format("2006-01-02"),
			},
			checkIns: []CheckInDTO{
				{ValueRecorded: 600000000, Note: "Check-in bulan pertama", CheckedInDate: now.AddDate(0, -1, 0).Format("2006-01-02")},
				{ValueRecorded: 1050000000, Note: "Check-in bulan kedua", CheckedInDate: now.AddDate(0, 0, -5).Format("2006-01-02")},
			},
		},
		{
			dto: CreateGoalDTO{
				Title:       "Produktivitas Produksi Karton Box",
				Description: "Target output unit per bulan di lini produksi utama",
				GoalType:    "team",
				OwnerName:   "Tim Produksi",
				Category:    "Productivity",
				TargetValue: 50000,
				Unit:        "units",
				PeriodStart: now.AddDate(0, -1, 0).Format("2006-01-02"),
				PeriodEnd:   now.AddDate(0, 2, 0).Format("2006-01-02"),
			},
			checkIns: []CheckInDTO{
				{ValueRecorded: 18000, Note: "Progress awal", CheckedInDate: now.AddDate(0, 0, -10).Format("2006-01-02")},
			},
		},
		{
			dto: CreateGoalDTO{
				Title:       "Skor Kepuasan Pelanggan",
				Description: "Target skor CSAT rata-rata dari survei pelanggan",
				GoalType:    "individual",
				OwnerName:   "Andi Wijaya",
				Category:    "Customer Satisfaction",
				TargetValue: 90,
				Unit:        "points",
				PeriodStart: now.AddDate(0, -3, 0).Format("2006-01-02"),
				PeriodEnd:   now.AddDate(0, -1, 0).Format("2006-01-02"),
			},
			checkIns: []CheckInDTO{
				{ValueRecorded: 92, Note: "Target tercapai", CheckedInDate: now.AddDate(0, -1, -2).Format("2006-01-02")},
			},
			complete: true,
		},
		{
			dto: CreateGoalDTO{
				Title:       "Efisiensi Biaya Operasional",
				Description: "Target penurunan biaya operasional gudang",
				GoalType:    "team",
				OwnerName:   "Tim Warehouse",
				Category:    "Cost Efficiency",
				TargetValue: 100,
				Unit:        "%",
				PeriodStart: now.AddDate(0, -1, 0).Format("2006-01-02"),
				PeriodEnd:   now.AddDate(0, 3, 0).Format("2006-01-02"),
			},
		},
		{
			dto: CreateGoalDTO{
				Title:       "Rekrutmen Talenta Baru",
				Description: "Target jumlah karyawan baru yang direkrut tahun ini",
				GoalType:    "team",
				OwnerName:   "Tim HR",
				Category:    "Recruitment",
				TargetValue: 25,
				Unit:        "units",
				PeriodStart: now.AddDate(0, -6, 0).Format("2006-01-02"),
				PeriodEnd:   now.AddDate(0, 6, 0).Format("2006-01-02"),
			},
			checkIns: []CheckInDTO{
				{ValueRecorded: 10, Note: "Progress semester pertama", CheckedInDate: now.AddDate(0, 0, -15).Format("2006-01-02")},
			},
		},
	}

	for _, seed := range seeds {
		res, err := uc.CreateGoal(ctx, seed.dto)
		if err != nil {
			continue
		}
		for _, ci := range seed.checkIns {
			if _, err := uc.CheckIn(ctx, res.ID, ci); err != nil {
				continue
			}
		}
		if seed.complete {
			_, _ = uc.CompleteGoal(ctx, res.ID)
		}
	}
	return nil
}
