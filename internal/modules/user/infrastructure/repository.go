package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/modules/user/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).First(&user, "email = ?", email).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) List(ctx context.Context, query types.PaginationQuery, companyID *uuid.UUID) ([]domain.User, int64, error) {
	var users []domain.User
	var total int64

	dbQuery := r.db.WithContext(ctx).Model(&domain.User{})

	if companyID != nil {
		dbQuery = dbQuery.Where("company_id = ?", companyID)
	}

	if query.Search != "" {
		searchParam := "%" + query.Search + "%"
		dbQuery = dbQuery.Where("name ILIKE ? OR email ILIKE ?", searchParam, searchParam)
	}

	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortBy := "created_at"
	allowedSorts := map[string]bool{
		"name":       true,
		"email":      true,
		"role":       true,
		"created_at": true,
	}
	if allowedSorts[query.SortBy] {
		sortBy = query.SortBy
	}

	orderClause := fmt.Sprintf("%s %s", sortBy, query.SortDir)

	err := dbQuery.Order(orderClause).
		Offset(query.Offset()).
		Limit(query.PerPage).
		Find(&users).Error

	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.User{}, "id = ?", id).Error
}
