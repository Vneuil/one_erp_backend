package infrastructure

import (
	"context"
	"errors"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/warehouse/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type warehouseRepository struct {
	db *gorm.DB
}

func NewWarehouseRepository(db *gorm.DB) domain.WarehouseRepository {
	return &warehouseRepository{db: db}
}

// Pick waves

func (r *warehouseRepository) CreatePickWave(ctx context.Context, w *domain.PickWave) error {
	tenantctx.SetTenantID(ctx, &w.TenantID)
	return r.db.WithContext(ctx).Create(w).Error
}

func (r *warehouseRepository) GetPickWaveByID(ctx context.Context, id uuid.UUID) (*domain.PickWave, error) {
	var w domain.PickWave
	err := r.db.WithContext(ctx).Preload("Lines").Where("id = ?", id).First(&w).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &w, nil
}

func (r *warehouseRepository) ListPickWaves(ctx context.Context, query types.PaginationQuery) ([]domain.PickWave, int64, error) {
	var waves []domain.PickWave
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.PickWave{}))
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Preload("Lines").Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&waves).Error
	return waves, total, err
}

func (r *warehouseRepository) UpdatePickWave(ctx context.Context, w *domain.PickWave) error {
	return r.db.WithContext(ctx).Save(w).Error
}

func (r *warehouseRepository) UpdatePickWaveLine(ctx context.Context, l *domain.PickWaveLine) error {
	return r.db.WithContext(ctx).Save(l).Error
}

// Packing sessions

func (r *warehouseRepository) CreatePackingSession(ctx context.Context, s *domain.PackingSession) error {
	tenantctx.SetTenantID(ctx, &s.TenantID)
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *warehouseRepository) GetPackingSessionByID(ctx context.Context, id uuid.UUID) (*domain.PackingSession, error) {
	var s domain.PackingSession
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *warehouseRepository) ListPackingSessions(ctx context.Context, query types.PaginationQuery) ([]domain.PackingSession, int64, error) {
	var sessions []domain.PackingSession
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.PackingSession{}))
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&sessions).Error
	return sessions, total, err
}

func (r *warehouseRepository) UpdatePackingSession(ctx context.Context, s *domain.PackingSession) error {
	return r.db.WithContext(ctx).Save(s).Error
}

// Shipments

func (r *warehouseRepository) CreateShipment(ctx context.Context, s *domain.Shipment) error {
	tenantctx.SetTenantID(ctx, &s.TenantID)
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *warehouseRepository) GetShipmentByID(ctx context.Context, id uuid.UUID) (*domain.Shipment, error) {
	var s domain.Shipment
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *warehouseRepository) ListShipments(ctx context.Context, query types.PaginationQuery) ([]domain.Shipment, int64, error) {
	var shipments []domain.Shipment
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Shipment{}))
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&shipments).Error
	return shipments, total, err
}

func (r *warehouseRepository) UpdateShipment(ctx context.Context, s *domain.Shipment) error {
	return r.db.WithContext(ctx).Save(s).Error
}

func (r *warehouseRepository) WithTransaction(ctx context.Context, fn func(txRepo domain.WarehouseRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&warehouseRepository{db: tx})
	})
}
