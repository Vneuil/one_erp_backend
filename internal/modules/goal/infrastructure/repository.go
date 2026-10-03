package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/goal/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type goalsRepository struct {
	db *gorm.DB
}

func NewGoalsRepository(db *gorm.DB) domain.GoalsRepository {
	return &goalsRepository{db: db}
}

func (r *goalsRepository) CreateGoal(ctx context.Context, g *domain.Goal) error {
	tenantctx.SetTenantID(ctx, &g.TenantID)
	return r.db.WithContext(ctx).Create(g).Error
}

func (r *goalsRepository) GetGoalByID(ctx context.Context, id uuid.UUID) (*domain.Goal, error) {
	var g domain.Goal
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&g).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &g, nil
}

func (r *goalsRepository) ListGoals(ctx context.Context, query types.PaginationQuery) ([]domain.Goal, int64, error) {
	var goals []domain.Goal
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Goal{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("title ILIKE ? OR owner_name ILIKE ? OR category ILIKE ?", pattern, pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&goals).Error
	return goals, total, err
}

func (r *goalsRepository) ListAllGoals(ctx context.Context) ([]domain.Goal, error) {
	var goals []domain.Goal
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Order("created_at desc").Find(&goals).Error
	return goals, err
}

func (r *goalsRepository) UpdateGoal(ctx context.Context, g *domain.Goal) error {
	return r.db.WithContext(ctx).Save(g).Error
}

func (r *goalsRepository) CountGoals(ctx context.Context) (int64, error) {
	var total int64
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Goal{})).Count(&total).Error
	return total, err
}

func (r *goalsRepository) CreateCheckIn(ctx context.Context, ci *domain.GoalCheckIn) error {
	return r.db.WithContext(ctx).Create(ci).Error
}

func (r *goalsRepository) ListCheckInsByGoal(ctx context.Context, goalID uuid.UUID) ([]domain.GoalCheckIn, error) {
	var checkIns []domain.GoalCheckIn
	err := r.db.WithContext(ctx).Where("goal_id = ?", goalID).Order("checked_in_date desc").Find(&checkIns).Error
	return checkIns, err
}
