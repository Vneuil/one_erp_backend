package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/marketplace/domain"
)

// ConnectionResponseDTO is the connection shape returned to clients. Token
// values are intentionally never included.
type ConnectionResponseDTO struct {
	Platform        domain.Platform `json:"platform"`
	ShopID          string          `json:"shopId"`
	ShopName        string          `json:"shopName"`
	Status          string          `json:"status"`
	ConnectedAt     time.Time       `json:"connectedAt"`
	AccessExpiresAt time.Time       `json:"accessExpiresAt"`
	LastSyncAt      *time.Time      `json:"lastSyncAt,omitempty"`
	LastSyncError   string          `json:"lastSyncError,omitempty"`
}

func ToConnectionResponse(c *domain.MarketplaceConnection) ConnectionResponseDTO {
	return ConnectionResponseDTO{
		Platform:        c.Platform,
		ShopID:          c.ShopID,
		ShopName:        c.ShopName,
		Status:          c.Status,
		ConnectedAt:     c.ConnectedAt,
		AccessExpiresAt: c.AccessExpiresAt,
		LastSyncAt:      c.LastSyncAt,
		LastSyncError:   c.LastSyncError,
	}
}

func ToConnectionResponseList(cs []domain.MarketplaceConnection) []ConnectionResponseDTO {
	out := make([]ConnectionResponseDTO, 0, len(cs))
	for i := range cs {
		out = append(out, ToConnectionResponse(&cs[i]))
	}
	return out
}

type SyncResultDTO struct {
	Platform      domain.Platform `json:"platform"`
	OrdersFetched int             `json:"ordersFetched"`
	OrdersCreated int             `json:"ordersCreated"`
	OrdersSkipped int             `json:"ordersSkipped"`

	ProductsFetched int `json:"productsFetched"`
	ProductsCreated int `json:"productsCreated"`
	ProductsUpdated int `json:"productsUpdated"`
}
