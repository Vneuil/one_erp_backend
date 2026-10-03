package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/crypto"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/marketplace/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// encryptConnectionSecrets encrypts the sensitive credential fields on conn
// in place before they are persisted. See internal/foundation/crypto for
// the encrypt-if-key-configured, else-plaintext fallback behavior.
func encryptConnectionSecrets(conn *domain.MarketplaceConnection) error {
	for _, f := range []*string{&conn.AccessToken, &conn.RefreshToken, &conn.ApiSellerKey, &conn.SignatureKey} {
		enc, err := crypto.Encrypt(*f)
		if err != nil {
			return err
		}
		*f = enc
	}
	return nil
}

// decryptConnectionSecrets reverses encryptConnectionSecrets after a read.
func decryptConnectionSecrets(conn *domain.MarketplaceConnection) error {
	for _, f := range []*string{&conn.AccessToken, &conn.RefreshToken, &conn.ApiSellerKey, &conn.SignatureKey} {
		dec, err := crypto.Decrypt(*f)
		if err != nil {
			return err
		}
		*f = dec
	}
	return nil
}

type marketplaceRepository struct {
	db *gorm.DB
}

func NewMarketplaceRepository(db *gorm.DB) domain.MarketplaceRepository {
	return &marketplaceRepository{db: db}
}

func (r *marketplaceRepository) UpsertConnection(ctx context.Context, conn *domain.MarketplaceConnection) error {
	if err := encryptConnectionSecrets(conn); err != nil {
		return err
	}

	var existing domain.MarketplaceConnection
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Where("platform = ?", conn.Platform).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			tenantctx.SetTenantID(ctx, &conn.TenantID)
			return r.db.WithContext(ctx).Create(conn).Error
		}
		return err
	}

	conn.ID = existing.ID
	return tenantctx.Scope(ctx, r.db.WithContext(ctx)).Model(&domain.MarketplaceConnection{}).Where("id = ?", existing.ID).Updates(map[string]any{
		"shop_id":               conn.ShopID,
		"shop_name":             conn.ShopName,
		"shop_cipher":           conn.ShopCipher,
		"access_token":          conn.AccessToken,
		"refresh_token":         conn.RefreshToken,
		"access_expires_at":     conn.AccessExpiresAt,
		"refresh_expires_at":    conn.RefreshExpiresAt,
		"status":                conn.Status,
		"connected_at":          conn.ConnectedAt,
		"last_sync_at":          conn.LastSyncAt,
		"last_sync_error":       conn.LastSyncError,
		"business_partner_code": conn.BusinessPartnerCode,
		"mta_username":          conn.MtaUsername,
		"api_seller_key":        conn.ApiSellerKey,
		"signature_key":         conn.SignatureKey,
	}).Error
}

func (r *marketplaceRepository) GetConnection(ctx context.Context, platform domain.Platform) (*domain.MarketplaceConnection, error) {
	var conn domain.MarketplaceConnection
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Where("platform = ?", platform).First(&conn).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if err := decryptConnectionSecrets(&conn); err != nil {
		return nil, err
	}
	return &conn, nil
}

func (r *marketplaceRepository) ListConnections(ctx context.Context) ([]domain.MarketplaceConnection, error) {
	var conns []domain.MarketplaceConnection
	if err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Order("created_at desc").Find(&conns).Error; err != nil {
		return nil, err
	}
	for i := range conns {
		if err := decryptConnectionSecrets(&conns[i]); err != nil {
			return nil, err
		}
	}
	return conns, nil
}

func (r *marketplaceRepository) DeleteConnection(ctx context.Context, platform domain.Platform) error {
	return tenantctx.Scope(ctx, r.db.WithContext(ctx)).Model(&domain.MarketplaceConnection{}).
		Where("platform = ?", platform).
		Updates(map[string]any{"status": "disconnected", "access_token": "", "refresh_token": "", "api_seller_key": "", "signature_key": ""}).Error
}

func (r *marketplaceRepository) IsOrderSynced(ctx context.Context, platform domain.Platform, marketplaceOrderID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.MarketplaceOrderSync{}).
		Where("platform = ? AND marketplace_order_id = ?", platform, marketplaceOrderID).
		Count(&count).Error
	return count > 0, err
}

func (r *marketplaceRepository) RecordOrderSync(ctx context.Context, rec *domain.MarketplaceOrderSync) error {
	if rec.SyncedAt.IsZero() {
		rec.SyncedAt = time.Now()
	}
	tenantctx.SetTenantID(ctx, &rec.TenantID)
	return r.db.WithContext(ctx).Create(rec).Error
}

func (r *marketplaceRepository) GetOrderSyncBySalesOrderID(ctx context.Context, salesOrderID uuid.UUID) (*domain.MarketplaceOrderSync, error) {
	var rec domain.MarketplaceOrderSync
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Where("sales_order_id = ?", salesOrderID).First(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}
