package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// CompanyRepository defines the storage interface for Company domain
type CompanyRepository interface {
	Create(ctx context.Context, company *Company) error
	GetByID(ctx context.Context, id uuid.UUID) (*Company, error)
	GetByCode(ctx context.Context, code string) (*Company, error)
	List(ctx context.Context, query types.PaginationQuery) ([]Company, int64, error)
	Update(ctx context.Context, company *Company) error
	Delete(ctx context.Context, id uuid.UUID) error
}
