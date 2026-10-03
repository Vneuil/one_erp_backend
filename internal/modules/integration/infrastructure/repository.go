package infrastructure

import (
	"context"
	"errors"

	"github.com/divinecoid/one-backend/internal/modules/integration/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type apiKeyRepository struct {
	db *gorm.DB
}

func NewAPIKeyRepository(db *gorm.DB) domain.APIKeyRepository {
	return &apiKeyRepository{db: db}
}

func (r *apiKeyRepository) Create(ctx context.Context, k *domain.APIKey) error {
	return r.db.WithContext(ctx).Create(k).Error
}

func (r *apiKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.APIKey, error) {
	var k domain.APIKey
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&k).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &k, nil
}

func (r *apiKeyRepository) ListAll(ctx context.Context) ([]domain.APIKey, error) {
	var keys []domain.APIKey
	err := r.db.WithContext(ctx).Order("created_at desc").Find(&keys).Error
	return keys, err
}

func (r *apiKeyRepository) Update(ctx context.Context, k *domain.APIKey) error {
	return r.db.WithContext(ctx).Save(k).Error
}

func (r *apiKeyRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.APIKey{}).Count(&total).Error
	return total, err
}

type webhookRepository struct {
	db *gorm.DB
}

func NewWebhookRepository(db *gorm.DB) domain.WebhookRepository {
	return &webhookRepository{db: db}
}

func (r *webhookRepository) Create(ctx context.Context, w *domain.WebhookSubscription) error {
	return r.db.WithContext(ctx).Create(w).Error
}

func (r *webhookRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.WebhookSubscription, error) {
	var w domain.WebhookSubscription
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&w).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &w, nil
}

func (r *webhookRepository) ListAll(ctx context.Context) ([]domain.WebhookSubscription, error) {
	var webhooks []domain.WebhookSubscription
	err := r.db.WithContext(ctx).Order("created_at desc").Find(&webhooks).Error
	return webhooks, err
}

func (r *webhookRepository) Update(ctx context.Context, w *domain.WebhookSubscription) error {
	return r.db.WithContext(ctx).Save(w).Error
}

func (r *webhookRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.WebhookSubscription{}).Count(&total).Error
	return total, err
}

func (r *webhookRepository) CreateDelivery(ctx context.Context, d *domain.WebhookDelivery) error {
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *webhookRepository) ListDeliveriesBySubscription(ctx context.Context, subscriptionID uuid.UUID) ([]domain.WebhookDelivery, error) {
	var deliveries []domain.WebhookDelivery
	err := r.db.WithContext(ctx).Where("subscription_id = ?", subscriptionID).Order("attempted_at desc").Find(&deliveries).Error
	return deliveries, err
}
