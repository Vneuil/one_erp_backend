package types

import (
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseEntity contains standard audit fields for all domain models
type BaseEntity struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `gorm:"not null" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"not null" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
}

// BeforeCreate hook to generate UUID if not already provided
func (b *BaseEntity) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// PaginationQuery represents standard query params for pagination
type PaginationQuery struct {
	Page    int    `query:"page"`
	PerPage int    `query:"perPage"`
	SortBy  string `query:"sortBy"`
	SortDir string `query:"sortDir"`
	Search  string `query:"search"`
}

func (p *PaginationQuery) SetDefaults() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PerPage <= 0 {
		p.PerPage = 10
	}
	if p.PerPage > 100 {
		p.PerPage = 100
	}
	if p.SortDir != "asc" && p.SortDir != "desc" {
		p.SortDir = "desc"
	}
}

func (p *PaginationQuery) Offset() int {
	return (p.Page - 1) * p.PerPage
}

// PaginationMeta represents pagination metadata in API responses
type PaginationMeta struct {
	CurrentPage int   `json:"currentPage"`
	PerPage     int   `json:"perPage"`
	TotalItems  int64 `json:"totalItems"`
	TotalPages  int   `json:"totalPages"`
}

func NewPaginationMeta(totalItems int64, page, perPage int) PaginationMeta {
	totalPages := 0
	if perPage > 0 {
		totalPages = int(math.Ceil(float64(totalItems) / float64(perPage)))
	}
	return PaginationMeta{
		CurrentPage: page,
		PerPage:     perPage,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
	}
}
