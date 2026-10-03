package application

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/config"
	inventoryApp "github.com/divinecoid/one-backend/internal/modules/inventory/application"
	"github.com/divinecoid/one-backend/internal/modules/marketplace/domain"
	"github.com/divinecoid/one-backend/internal/modules/marketplace/infrastructure"
	productDomain "github.com/divinecoid/one-backend/internal/modules/product/domain"
	salesApp "github.com/divinecoid/one-backend/internal/modules/sales/application"
	salesDomain "github.com/divinecoid/one-backend/internal/modules/sales/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// syncLookbackWindow is how far back each sync call looks for new orders.
// Re-running sync is safe (see domain.MarketplaceOrderSync dedupe), so this
// window only needs to comfortably exceed how often /sync is expected to be
// called (manual "Sync Now" clicks).
const syncLookbackWindow = 30 * 24 * time.Hour

type MarketplaceUseCase interface {
	ListConnections(ctx context.Context) ([]ConnectionResponseDTO, error)
	TikTokConnectURL(ctx context.Context, state string) (string, error)
	TikTokHandleCallback(ctx context.Context, authCode string) (*ConnectionResponseDTO, error)
	ShopeeConnectURLWithRedirect(ctx context.Context, redirectURI string) (string, error)
	ShopeeHandleCallback(ctx context.Context, code string, shopID int64) (*ConnectionResponseDTO, error)
	ConnectBlibli(ctx context.Context, businessPartnerCode, mtaUsername, apiSellerKey, signatureKey string) (*ConnectionResponseDTO, error)
	LazadaConnectURL(ctx context.Context, state string) (string, error)
	LazadaHandleCallback(ctx context.Context, code string) (*ConnectionResponseDTO, error)
	Disconnect(ctx context.Context, platform domain.Platform) error
	Sync(ctx context.Context, platform domain.Platform) (*SyncResultDTO, error)
	// MarkOrderReadyToShip marks a marketplace-synced Sales Order ready for
	// pickup via that marketplace's own fulfillment API (see each client's
	// Ship*/RTS method), then records the resulting internal Delivery.
	MarkOrderReadyToShip(ctx context.Context, salesOrderID uuid.UUID) (*salesApp.DeliveryResponseDTO, error)
}

type marketplaceUseCase struct {
	repo        domain.MarketplaceRepository
	salesRepo   salesDomain.SalesRepository
	salesUC     salesApp.SalesUseCase
	productRepo productDomain.ProductRepository
	inventoryUC inventoryApp.InventoryUseCase
	tiktok      *infrastructure.TikTokShopClient
	shopee      *infrastructure.ShopeeClient
	shopeeCfg   config.ShopeeConfig
	blibli      *infrastructure.BlibliClient
	lazada      *infrastructure.LazadaClient
	lazadaCfg   config.LazadaConfig
}

// NewMarketplaceUseCase wires marketplace sync to real Sales Order creation
// (via salesUC) so that synced orders go through the same stock-check-and-
// deduct path as manually created orders, instead of writing bare header
// rows with salesRepo.CreateOrder directly. productRepo/inventoryUC are used
// to resolve marketplace line items (SKU/name) to real Products and to pick
// a default warehouse when the marketplace order carries none.
func NewMarketplaceUseCase(
	repo domain.MarketplaceRepository,
	salesRepo salesDomain.SalesRepository,
	salesUC salesApp.SalesUseCase,
	productRepo productDomain.ProductRepository,
	inventoryUC inventoryApp.InventoryUseCase,
	tiktokCfg config.TikTokShopConfig,
	shopeeCfg config.ShopeeConfig,
	blibliCfg config.BlibliConfig,
	lazadaCfg config.LazadaConfig,
) MarketplaceUseCase {
	return &marketplaceUseCase{
		repo:        repo,
		salesRepo:   salesRepo,
		salesUC:     salesUC,
		productRepo: productRepo,
		inventoryUC: inventoryUC,
		tiktok:      infrastructure.NewTikTokShopClient(tiktokCfg),
		shopee:      infrastructure.NewShopeeClient(shopeeCfg),
		shopeeCfg:   shopeeCfg,
		blibli:      infrastructure.NewBlibliClient(blibliCfg),
		lazada:      infrastructure.NewLazadaClient(lazadaCfg),
		lazadaCfg:   lazadaCfg,
	}
}

// resolveDefaultWarehouseID returns the tenant's first warehouse, used when
// a marketplace order carries no warehouse context of its own. Returns nil
// (not an error) when the tenant has no warehouses yet, so sync still
// creates the order header - just without stock deduction, matching
// CreateOrder's existing "only deduct when WarehouseID+Lines both present"
// gate.
func (uc *marketplaceUseCase) resolveDefaultWarehouseID(ctx context.Context) *uuid.UUID {
	if uc.inventoryUC == nil {
		return nil
	}
	warehouses, _, err := uc.inventoryUC.ListWarehouses(ctx, types.PaginationQuery{Page: 1, PerPage: 1})
	if err != nil || len(warehouses) == 0 {
		return nil
	}
	id := warehouses[0].ID
	return &id
}

// resolveProduct matches a marketplace line item to a real Product: first
// by exact SKU, then falling back to a name-based search. Returns nil (not
// an error) when nothing matches, so the caller can skip just that line
// instead of failing the whole order sync.
func (uc *marketplaceUseCase) resolveProduct(ctx context.Context, sku, name string) *productDomain.Product {
	if uc.productRepo == nil {
		return nil
	}
	if sku != "" {
		if p, err := uc.productRepo.GetBySKU(ctx, sku); err == nil && p != nil {
			return p
		}
	}
	if name != "" {
		if products, _, err := uc.productRepo.List(ctx, types.PaginationQuery{Page: 1, PerPage: 1, Search: name}); err == nil && len(products) > 0 {
			return &products[0]
		}
	}
	return nil
}

func (uc *marketplaceUseCase) LazadaConnectURL(ctx context.Context, state string) (string, error) {
	if !uc.lazada.Enabled() {
		return "", apperrors.NewServiceUnavailable("Lazada integration is not configured (LAZADA_APP_KEY/LAZADA_APP_SECRET missing). Please contact support.")
	}
	return uc.lazada.AuthorizeURL(uc.lazadaCfg.RedirectURI, state), nil
}

func (uc *marketplaceUseCase) LazadaHandleCallback(ctx context.Context, code string) (*ConnectionResponseDTO, error) {
	if !uc.lazada.Enabled() {
		return nil, apperrors.NewServiceUnavailable("Lazada integration is not configured.")
	}
	if code == "" {
		return nil, apperrors.NewBadRequest("Missing authorization code")
	}

	tokenResp, err := uc.lazada.ExchangeCode(ctx, code)
	if err != nil {
		return nil, apperrors.NewServiceUnavailable("Failed to exchange Lazada authorization code: " + err.Error())
	}

	now := time.Now()
	shopName := tokenResp.Account
	if shopName == "" {
		shopName = tokenResp.AccountID
	}
	conn := &domain.MarketplaceConnection{
		Platform:         domain.PlatformLazada,
		ShopID:           tokenResp.AccountID,
		ShopName:         shopName,
		AccessToken:      tokenResp.AccessToken,
		RefreshToken:     tokenResp.RefreshToken,
		AccessExpiresAt:  now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		RefreshExpiresAt: now.Add(365 * 24 * time.Hour), // Lazada refresh_token validity is not returned by the token response; re-authorize if refresh fails
		Status:           "connected",
		ConnectedAt:      now,
	}
	if err := uc.repo.UpsertConnection(ctx, conn); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to store Lazada connection")
	}

	resp := ToConnectionResponse(conn)
	return &resp, nil
}

// ConnectBlibli stores a tenant's Blibli Seller Center credentials after
// verifying them with a live orderDetail call, mirroring
// TikTokHandleCallback/ShopeeHandleCallback's structure - the connection is
// only ever marked "connected" once the credentials are proven to work,
// never optimistically.
func (uc *marketplaceUseCase) ConnectBlibli(ctx context.Context, businessPartnerCode, mtaUsername, apiSellerKey, signatureKey string) (*ConnectionResponseDTO, error) {
	if !uc.blibli.Enabled() {
		return nil, apperrors.NewServiceUnavailable("Blibli integration is not configured (BLIBLI_API_CLIENT_ID/BLIBLI_API_CLIENT_SECRET missing). Please contact support.")
	}
	if businessPartnerCode == "" || mtaUsername == "" || apiSellerKey == "" {
		return nil, apperrors.NewBadRequest("Business Partner Code, MTA Username and API Seller Key are required")
	}

	creds := infrastructure.BlibliCredentials{
		BusinessPartnerCode: businessPartnerCode,
		MtaUsername:         mtaUsername,
		ApiSellerKey:        apiSellerKey,
		SignatureKey:        signatureKey,
	}
	if err := uc.blibli.TestConnection(ctx, creds); err != nil {
		return nil, apperrors.NewBadRequest("Could not verify Blibli credentials: " + err.Error())
	}

	now := time.Now()
	conn := &domain.MarketplaceConnection{
		Platform:            domain.PlatformBlibli,
		ShopName:            businessPartnerCode,
		BusinessPartnerCode: businessPartnerCode,
		MtaUsername:         mtaUsername,
		ApiSellerKey:        apiSellerKey,
		SignatureKey:        signatureKey,
		Status:              "connected",
		ConnectedAt:         now,
		// Blibli has no OAuth token lifecycle - AccessExpiresAt/RefreshExpiresAt
		// are left zero-valued and never consulted for this platform (see
		// ensureFreshToken/Sync, both of which special-case PlatformBlibli).
	}
	if err := uc.repo.UpsertConnection(ctx, conn); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to store Blibli connection")
	}

	resp := ToConnectionResponse(conn)
	return &resp, nil
}

func (uc *marketplaceUseCase) ListConnections(ctx context.Context) ([]ConnectionResponseDTO, error) {
	conns, err := uc.repo.ListConnections(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list marketplace connections")
	}
	return ToConnectionResponseList(conns), nil
}

func (uc *marketplaceUseCase) TikTokConnectURL(ctx context.Context, state string) (string, error) {
	if !uc.tiktok.Enabled() {
		return "", apperrors.NewServiceUnavailable("TikTok Shop integration is not configured (TIKTOK_SHOP_APP_KEY/TIKTOK_SHOP_APP_SECRET missing). Please contact support.")
	}
	return uc.tiktok.AuthorizeURL(state), nil
}

func (uc *marketplaceUseCase) TikTokHandleCallback(ctx context.Context, authCode string) (*ConnectionResponseDTO, error) {
	if !uc.tiktok.Enabled() {
		return nil, apperrors.NewServiceUnavailable("TikTok Shop integration is not configured.")
	}
	if authCode == "" {
		return nil, apperrors.NewBadRequest("Missing authorization code")
	}

	tokenResp, err := uc.tiktok.ExchangeCode(ctx, authCode)
	if err != nil {
		return nil, apperrors.NewServiceUnavailable("Failed to exchange TikTok Shop authorization code: " + err.Error())
	}

	// shop_cipher is required on every order/product API call, so resolve
	// the authorized shop now rather than leaving it empty.
	var shopID, shopCipher, shopName string
	shopName = tokenResp.Data.SellerName
	shops, err := uc.tiktok.GetAuthorizedShops(ctx, tokenResp.Data.AccessToken)
	if err != nil {
		return nil, apperrors.NewServiceUnavailable("Failed to fetch authorized TikTok Shop shops: " + err.Error())
	}
	if len(shops) == 0 {
		return nil, apperrors.NewBadRequest("No TikTok Shop shop was authorized for this app")
	}
	shopID, shopCipher = shops[0].ID, shops[0].Cipher
	if shops[0].Name != "" {
		shopName = shops[0].Name
	}

	now := time.Now()
	conn := &domain.MarketplaceConnection{
		Platform:         domain.PlatformTikTokShop,
		ShopID:           shopID,
		ShopCipher:       shopCipher,
		ShopName:         shopName,
		AccessToken:      tokenResp.Data.AccessToken,
		RefreshToken:     tokenResp.Data.RefreshToken,
		AccessExpiresAt:  now.Add(time.Duration(tokenResp.Data.AccessTokenExpireIn) * time.Second),
		RefreshExpiresAt: now.Add(time.Duration(tokenResp.Data.RefreshTokenExpireIn) * time.Second),
		Status:           "connected",
		ConnectedAt:      now,
	}
	if err := uc.repo.UpsertConnection(ctx, conn); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to store TikTok Shop connection")
	}

	resp := ToConnectionResponse(conn)
	return &resp, nil
}

func (uc *marketplaceUseCase) ShopeeConnectURLWithRedirect(ctx context.Context, redirectURI string) (string, error) {
	if !uc.shopee.Enabled() {
		return "", apperrors.NewServiceUnavailable("Shopee integration is not configured (SHOPEE_PARTNER_ID/SHOPEE_PARTNER_KEY missing). Please contact support.")
	}
	return uc.shopee.AuthorizeURL(redirectURI), nil
}

func (uc *marketplaceUseCase) ShopeeHandleCallback(ctx context.Context, code string, shopID int64) (*ConnectionResponseDTO, error) {
	if !uc.shopee.Enabled() {
		return nil, apperrors.NewServiceUnavailable("Shopee integration is not configured.")
	}
	if code == "" || shopID == 0 {
		return nil, apperrors.NewBadRequest("Missing authorization code or shop_id")
	}

	tokenResp, err := uc.shopee.ExchangeCode(ctx, code, shopID)
	if err != nil {
		return nil, apperrors.NewServiceUnavailable("Failed to exchange Shopee authorization code: " + err.Error())
	}

	now := time.Now()
	conn := &domain.MarketplaceConnection{
		Platform:         domain.PlatformShopee,
		ShopID:           strconv.FormatInt(shopID, 10),
		AccessToken:      tokenResp.AccessToken,
		RefreshToken:     tokenResp.RefreshToken,
		AccessExpiresAt:  now.Add(time.Duration(tokenResp.ExpireIn) * time.Second),
		RefreshExpiresAt: now.Add(365 * 24 * time.Hour), // Shopee refresh_token is valid ~1 year of continued use; re-authorize if refresh fails
		Status:           "connected",
		ConnectedAt:      now,
	}
	if err := uc.repo.UpsertConnection(ctx, conn); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to store Shopee connection")
	}

	resp := ToConnectionResponse(conn)
	return &resp, nil
}

func (uc *marketplaceUseCase) Disconnect(ctx context.Context, platform domain.Platform) error {
	if !platform.Valid() {
		return apperrors.NewBadRequest("Unsupported marketplace platform")
	}
	if err := uc.repo.DeleteConnection(ctx, platform); err != nil {
		return apperrors.NewInternal(err, "Failed to disconnect marketplace connection")
	}
	return nil
}

func (uc *marketplaceUseCase) Sync(ctx context.Context, platform domain.Platform) (*SyncResultDTO, error) {
	if !platform.Valid() {
		return nil, apperrors.NewBadRequest("Unsupported marketplace platform")
	}

	conn, err := uc.repo.GetConnection(ctx, platform)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load marketplace connection")
	}
	if conn == nil || conn.Status != "connected" {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("%s is not connected. Connect it first from Settings > Integrations.", platform.ChannelLabel()))
	}

	var accessToken string
	if platform != domain.PlatformBlibli {
		var refreshErr error
		accessToken, refreshErr = uc.ensureFreshToken(ctx, conn)
		if refreshErr != nil {
			return nil, apperrors.NewServiceUnavailable("Failed to refresh marketplace access token: " + refreshErr.Error())
		}
	}

	result := &SyncResultDTO{Platform: platform}

	switch platform {
	case domain.PlatformTikTokShop:
		if err := uc.syncTikTokProducts(ctx, conn, accessToken, result); err != nil {
			uc.recordSyncError(ctx, conn, err)
			return nil, apperrors.NewServiceUnavailable("Failed to sync TikTok Shop products: " + err.Error())
		}
		orders, err := uc.tiktok.SearchRecentOrders(ctx, accessToken, conn.ShopCipher, syncLookbackWindow)
		if err != nil {
			uc.recordSyncError(ctx, conn, err)
			return nil, apperrors.NewServiceUnavailable("Failed to fetch TikTok Shop orders: " + err.Error())
		}
		result.OrdersFetched = len(orders)
		for _, o := range orders {
			created, err := uc.syncTikTokOrder(ctx, conn, o)
			if err != nil {
				return nil, apperrors.NewInternal(err, "Failed to sync TikTok Shop order "+o.ID)
			}
			if created {
				result.OrdersCreated++
			} else {
				result.OrdersSkipped++
			}
		}

	case domain.PlatformShopee:
		shopID, _ := strconv.ParseInt(conn.ShopID, 10, 64)
		orders, err := uc.shopee.ListRecentOrders(ctx, accessToken, shopID, syncLookbackWindow)
		if err != nil {
			uc.recordSyncError(ctx, conn, err)
			return nil, apperrors.NewServiceUnavailable("Failed to fetch Shopee orders: " + err.Error())
		}
		result.OrdersFetched = len(orders)
		for _, o := range orders {
			created, err := uc.syncShopeeOrder(ctx, conn, o)
			if err != nil {
				return nil, apperrors.NewInternal(err, "Failed to sync Shopee order "+o.OrderSN)
			}
			if created {
				result.OrdersCreated++
			} else {
				result.OrdersSkipped++
			}
		}

	case domain.PlatformBlibli:
		creds := infrastructure.BlibliCredentials{
			BusinessPartnerCode: conn.BusinessPartnerCode,
			MtaUsername:         conn.MtaUsername,
			ApiSellerKey:        conn.ApiSellerKey,
			SignatureKey:        conn.SignatureKey,
		}
		orders, err := uc.blibli.ListRecentOrders(ctx, creds, syncLookbackWindow)
		if err != nil {
			uc.recordSyncError(ctx, conn, err)
			return nil, apperrors.NewServiceUnavailable("Failed to fetch Blibli orders: " + err.Error())
		}
		result.OrdersFetched = len(orders)
		for _, o := range orders {
			created, err := uc.syncBlibliOrder(ctx, conn, o)
			if err != nil {
				return nil, apperrors.NewInternal(err, "Failed to sync Blibli order "+o.OrderNo)
			}
			if created {
				result.OrdersCreated++
			} else {
				result.OrdersSkipped++
			}
		}

	case domain.PlatformLazada:
		orders, err := uc.lazada.ListRecentOrders(ctx, accessToken, syncLookbackWindow)
		if err != nil {
			uc.recordSyncError(ctx, conn, err)
			return nil, apperrors.NewServiceUnavailable("Failed to fetch Lazada orders: " + err.Error())
		}
		result.OrdersFetched = len(orders)
		for _, o := range orders {
			created, err := uc.syncLazadaOrder(ctx, conn, o)
			if err != nil {
				return nil, apperrors.NewInternal(err, fmt.Sprintf("Failed to sync Lazada order %d", o.OrderID))
			}
			if created {
				result.OrdersCreated++
			} else {
				result.OrdersSkipped++
			}
		}
	}

	now := time.Now()
	conn.LastSyncAt = &now
	conn.LastSyncError = ""
	_ = uc.repo.UpsertConnection(ctx, conn)

	return result, nil
}

// MarkOrderReadyToShip routes a "ready to ship"/pickup request for a
// marketplace-synced Sales Order to that marketplace's own fulfillment API,
// then records the shipment as an internal Delivery covering the order's
// full quantity (marketplace fulfillment ships the whole package at once,
// unlike a warehouse delivery which CreateDelivery lets split across many
// partial DOs). See each platform client's Ship*/RTS method for
// platform-specific confidence notes - Shopee and TikTok Shop call
// documented, dated API versions; Lazada and Blibli are best-effort and
// have not been verified against a live seller account.
func (uc *marketplaceUseCase) MarkOrderReadyToShip(ctx context.Context, salesOrderID uuid.UUID) (*salesApp.DeliveryResponseDTO, error) {
	sync, err := uc.repo.GetOrderSyncBySalesOrderID(ctx, salesOrderID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to look up marketplace order")
	}
	if sync == nil {
		return nil, apperrors.NewBadRequest("This sales order was not created from a marketplace sync")
	}

	conn, err := uc.repo.GetConnection(ctx, sync.Platform)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load marketplace connection")
	}
	if conn == nil || conn.Status != "connected" {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("%s is not connected. Connect it first from Settings > Integrations.", sync.Platform.ChannelLabel()))
	}

	var accessToken string
	if sync.Platform != domain.PlatformBlibli {
		accessToken, err = uc.ensureFreshToken(ctx, conn)
		if err != nil {
			return nil, apperrors.NewServiceUnavailable("Failed to refresh marketplace access token: " + err.Error())
		}
	}

	switch sync.Platform {
	case domain.PlatformTikTokShop:
		order, err := uc.tiktok.GetOrder(ctx, accessToken, conn.ShopCipher, sync.MarketplaceOrderID)
		if err != nil {
			return nil, apperrors.NewServiceUnavailable("Failed to fetch TikTok Shop order: " + err.Error())
		}
		if len(order.Packages) == 0 {
			return nil, apperrors.NewBadRequest("TikTok Shop order has no fulfillment package to ship yet")
		}
		for _, pkg := range order.Packages {
			if err := uc.tiktok.ShipPackage(ctx, accessToken, conn.ShopCipher, pkg.ID); err != nil {
				return nil, apperrors.NewServiceUnavailable("Failed to mark TikTok Shop package ready to ship: " + err.Error())
			}
		}

	case domain.PlatformShopee:
		shopID, _ := strconv.ParseInt(conn.ShopID, 10, 64)
		if err := uc.shopee.ShipOrder(ctx, accessToken, shopID, sync.MarketplaceOrderID); err != nil {
			return nil, apperrors.NewServiceUnavailable("Failed to mark Shopee order ready to ship: " + err.Error())
		}

	case domain.PlatformLazada:
		orderID, convErr := strconv.ParseInt(sync.MarketplaceOrderID, 10, 64)
		if convErr != nil {
			return nil, apperrors.NewInternal(convErr, "Invalid Lazada order ID")
		}
		items, err := uc.lazada.GetOrderItems(ctx, accessToken, orderID)
		if err != nil {
			return nil, apperrors.NewServiceUnavailable("Failed to fetch Lazada order items: " + err.Error())
		}
		ids := make([]int64, 0, len(items))
		for _, it := range items {
			if it.OrderItemID != 0 {
				ids = append(ids, it.OrderItemID)
			}
		}
		if len(ids) == 0 {
			return nil, apperrors.NewBadRequest("Lazada order has no item IDs to mark ready to ship")
		}
		if err := uc.lazada.SetReadyToShip(ctx, accessToken, ids); err != nil {
			return nil, apperrors.NewServiceUnavailable("Failed to mark Lazada order ready to ship: " + err.Error())
		}

	case domain.PlatformBlibli:
		creds := infrastructure.BlibliCredentials{
			BusinessPartnerCode: conn.BusinessPartnerCode,
			MtaUsername:         conn.MtaUsername,
			ApiSellerKey:        conn.ApiSellerKey,
			SignatureKey:        conn.SignatureKey,
		}
		// Blibli's order rows are per line item (OrderNo+OrderItemNo), and
		// the sync record only stores the order-level MarketplaceOrderID -
		// look the order back up to mark every item under it ready.
		orders, err := uc.blibli.ListRecentOrders(ctx, creds, syncLookbackWindow)
		if err != nil {
			return nil, apperrors.NewServiceUnavailable("Failed to fetch Blibli order: " + err.Error())
		}
		found := false
		for _, o := range orders {
			if o.OrderNo != sync.MarketplaceOrderID {
				continue
			}
			found = true
			if err := uc.blibli.MarkReadyToShip(ctx, creds, o.OrderNo, o.OrderItemNo); err != nil {
				return nil, apperrors.NewServiceUnavailable("Failed to mark Blibli order ready to ship: " + err.Error())
			}
		}
		if !found {
			return nil, apperrors.NewNotFound("Blibli order not found")
		}

	default:
		return nil, apperrors.NewBadRequest("Unsupported marketplace platform")
	}

	order, err := uc.salesRepo.GetOrderByID(ctx, salesOrderID)
	if err != nil || order == nil {
		return nil, apperrors.NewInternal(err, "Marketplace order was marked ready to ship, but failed to load it to record the delivery")
	}
	lines := make([]salesApp.DeliveryLineDTO, 0, len(order.Lines))
	for _, l := range order.Lines {
		lines = append(lines, salesApp.DeliveryLineDTO{ProductID: l.ProductID, Quantity: l.Quantity})
	}
	return uc.salesUC.CreateDelivery(ctx, salesApp.CreateDeliveryDTO{
		SalesOrderID: salesOrderID,
		CustomerName: order.CustomerName,
		Carrier:      sync.Platform.ChannelLabel() + " Pickup",
		Lines:        lines,
	})
}

func (uc *marketplaceUseCase) recordSyncError(ctx context.Context, conn *domain.MarketplaceConnection, syncErr error) {
	conn.LastSyncError = syncErr.Error()
	now := time.Now()
	conn.LastSyncAt = &now
	_ = uc.repo.UpsertConnection(ctx, conn)
}

// ensureFreshToken refreshes the access token if it is within 5 minutes of
// expiry (or already expired), and persists the rotated tokens.
func (uc *marketplaceUseCase) ensureFreshToken(ctx context.Context, conn *domain.MarketplaceConnection) (string, error) {
	if time.Until(conn.AccessExpiresAt) > 5*time.Minute {
		return conn.AccessToken, nil
	}

	switch conn.Platform {
	case domain.PlatformTikTokShop:
		resp, err := uc.tiktok.RefreshToken(ctx, conn.RefreshToken)
		if err != nil {
			return "", err
		}
		now := time.Now()
		conn.AccessToken = resp.Data.AccessToken
		conn.RefreshToken = resp.Data.RefreshToken
		conn.AccessExpiresAt = now.Add(time.Duration(resp.Data.AccessTokenExpireIn) * time.Second)
		conn.RefreshExpiresAt = now.Add(time.Duration(resp.Data.RefreshTokenExpireIn) * time.Second)
		if err := uc.repo.UpsertConnection(ctx, conn); err != nil {
			return "", err
		}
		return conn.AccessToken, nil

	case domain.PlatformShopee:
		shopID, _ := strconv.ParseInt(conn.ShopID, 10, 64)
		resp, err := uc.shopee.RefreshToken(ctx, conn.RefreshToken, shopID)
		if err != nil {
			return "", err
		}
		now := time.Now()
		conn.AccessToken = resp.AccessToken
		conn.RefreshToken = resp.RefreshToken
		conn.AccessExpiresAt = now.Add(time.Duration(resp.ExpireIn) * time.Second)
		if err := uc.repo.UpsertConnection(ctx, conn); err != nil {
			return "", err
		}
		return conn.AccessToken, nil

	case domain.PlatformLazada:
		resp, err := uc.lazada.RefreshToken(ctx, conn.RefreshToken)
		if err != nil {
			return "", err
		}
		now := time.Now()
		conn.AccessToken = resp.AccessToken
		conn.RefreshToken = resp.RefreshToken
		conn.AccessExpiresAt = now.Add(time.Duration(resp.ExpiresIn) * time.Second)
		if err := uc.repo.UpsertConnection(ctx, conn); err != nil {
			return "", err
		}
		return conn.AccessToken, nil
	}
	return conn.AccessToken, nil
}

// syncTikTokProducts imports the TikTok Shop catalog into Products, matching
// first on TikTok product id, then on seller SKU, and creating a product
// when neither exists. Each SKU of a TikTok product becomes one Product.
func (uc *marketplaceUseCase) syncTikTokProducts(ctx context.Context, conn *domain.MarketplaceConnection, accessToken string, result *SyncResultDTO) error {
	if uc.productRepo == nil {
		return nil
	}
	products, err := uc.tiktok.SearchProducts(ctx, accessToken, conn.ShopCipher)
	if err != nil {
		return err
	}
	result.ProductsFetched = len(products)
	platform := string(domain.PlatformTikTokShop)
	for _, tp := range products {
		for i, sku := range tp.SKUs {
			code := sku.SellerSKU
			if code == "" {
				code = fmt.Sprintf("TT-%s-%d", tp.ID, i+1)
			}
			price, _ := strconv.ParseFloat(sku.Price.SalePrice, 64)
			stock := 0
			for _, inv := range sku.Inventory {
				stock += inv.Quantity
			}
			existing, _ := uc.productRepo.GetBySKU(ctx, code)
			if existing != nil && existing.ID != uuid.Nil {
				existing.MarketplacePlatform = platform
				existing.MarketplaceProductID = tp.ID
				if price > 0 {
					existing.SellingPrice = price
				}
				if err := uc.productRepo.Update(ctx, existing); err != nil {
					return err
				}
				result.ProductsUpdated++
				continue
			}
			p := &productDomain.Product{
				SKU:                  code,
				Name:                 tp.Title,
				Category:             "TikTok Shop",
				Unit:                 "pcs",
				Stock:                stock,
				SellingPrice:         price,
				Status:               "in_stock",
				MarketplacePlatform:  platform,
				MarketplaceProductID: tp.ID,
			}
			if stock <= 0 {
				p.Status = "out_of_stock"
			}
			if err := uc.productRepo.Create(ctx, p); err != nil {
				return err
			}
			result.ProductsCreated++
		}
	}
	return nil
}

func (uc *marketplaceUseCase) syncTikTokOrder(ctx context.Context, conn *domain.MarketplaceConnection, o infrastructure.TikTokOrder) (bool, error) {
	already, err := uc.repo.IsOrderSynced(ctx, domain.PlatformTikTokShop, o.ID)
	if err != nil {
		return false, err
	}
	if already {
		return false, nil
	}

	amount, _ := strconv.ParseFloat(o.PaymentInfo.TotalAmount, 64)
	customerName := o.RecipientAddress.Name
	if customerName == "" {
		customerName = "TikTok Shop Buyer"
	}
	var lines []salesApp.SalesOrderLineDTO
	for _, li := range o.LineItems {
		p := uc.resolveProduct(ctx, li.SellerSKU, li.ProductName)
		if p == nil {
			continue // no matching product - skip this line rather than fail the whole sync
		}
		price, _ := strconv.ParseFloat(li.SalePrice, 64)
		lines = append(lines, salesApp.SalesOrderLineDTO{ProductID: p.ID, Quantity: 1, UnitPrice: price})
	}

	orderID, err := uc.createSyncedSalesOrder(ctx, salesApp.CreateSalesOrderDTO{
		OrderNumber:         fmt.Sprintf("TT-%s", o.ID),
		CustomerName:        customerName,
		TotalAmount:         amount,
		Status:              mapMarketplaceStatus(o.Status),
		PaymentStatus:       "Paid",
		Channel:             domain.PlatformTikTokShop.ChannelLabel(),
		WarehouseID:         uc.resolveDefaultWarehouseID(ctx),
		Lines:               lines,
		MarketplacePlatform: string(domain.PlatformTikTokShop),
		MarketplaceOrderID:  o.ID,
	})
	if err != nil {
		return false, err
	}
	if err := uc.repo.RecordOrderSync(ctx, &domain.MarketplaceOrderSync{
		Platform:           domain.PlatformTikTokShop,
		MarketplaceOrderID: o.ID,
		SalesOrderID:       orderID,
	}); err != nil {
		return false, err
	}
	return true, nil
}

// createSyncedSalesOrder creates the Sales Order through the real SalesUseCase
// (stock-checked, real lines) when available, falling back to the bare
// salesRepo.CreateOrder path only if this marketplaceUseCase was constructed
// without a salesUC (e.g. an older caller/test). orderDate is set inside
// SalesUseCase.CreateOrder itself (today's date), matching how manually
// created orders are dated.
func (uc *marketplaceUseCase) createSyncedSalesOrder(ctx context.Context, dto salesApp.CreateSalesOrderDTO) (uuid.UUID, error) {
	if uc.salesUC != nil {
		resp, err := uc.salesUC.CreateOrder(ctx, dto)
		if err != nil {
			return uuid.Nil, err
		}
		return resp.ID, nil
	}

	order := &salesDomain.SalesOrder{
		OrderNumber:         dto.OrderNumber,
		CustomerName:        dto.CustomerName,
		TotalAmount:         dto.TotalAmount,
		Status:              dto.Status,
		PaymentStatus:       dto.PaymentStatus,
		Channel:             dto.Channel,
		OrderDate:           time.Now().Format("2006-01-02"),
		MarketplacePlatform: dto.MarketplacePlatform,
		MarketplaceOrderID:  dto.MarketplaceOrderID,
	}
	if err := uc.salesRepo.CreateOrder(ctx, order); err != nil {
		return uuid.Nil, err
	}
	return order.ID, nil
}

func (uc *marketplaceUseCase) syncShopeeOrder(ctx context.Context, conn *domain.MarketplaceConnection, o infrastructure.ShopeeOrder) (bool, error) {
	already, err := uc.repo.IsOrderSynced(ctx, domain.PlatformShopee, o.OrderSN)
	if err != nil {
		return false, err
	}
	if already {
		return false, nil
	}

	customerName := o.BuyerUsername
	if customerName == "" {
		customerName = "Shopee Buyer"
	}
	var lines []salesApp.SalesOrderLineDTO
	for _, li := range o.ItemList {
		p := uc.resolveProduct(ctx, li.ItemSKU, li.ItemName)
		if p == nil {
			continue
		}
		qty := float64(li.ModelQuantity)
		if qty <= 0 {
			qty = 1
		}
		lines = append(lines, salesApp.SalesOrderLineDTO{ProductID: p.ID, Quantity: qty, UnitPrice: li.ModelDiscountedPrice})
	}

	orderID, err := uc.createSyncedSalesOrder(ctx, salesApp.CreateSalesOrderDTO{
		OrderNumber:         fmt.Sprintf("SHP-%s", o.OrderSN),
		CustomerName:        customerName,
		TotalAmount:         o.TotalAmount,
		Status:              mapMarketplaceStatus(o.OrderStatus),
		PaymentStatus:       "Paid",
		Channel:             domain.PlatformShopee.ChannelLabel(),
		WarehouseID:         uc.resolveDefaultWarehouseID(ctx),
		Lines:               lines,
		MarketplacePlatform: string(domain.PlatformShopee),
		MarketplaceOrderID:  o.OrderSN,
	})
	if err != nil {
		return false, err
	}
	if err := uc.repo.RecordOrderSync(ctx, &domain.MarketplaceOrderSync{
		Platform:           domain.PlatformShopee,
		MarketplaceOrderID: o.OrderSN,
		SalesOrderID:       orderID,
	}); err != nil {
		return false, err
	}
	return true, nil
}

func (uc *marketplaceUseCase) syncBlibliOrder(ctx context.Context, conn *domain.MarketplaceConnection, o infrastructure.BlibliOrder) (bool, error) {
	orderKey := o.OrderNo
	if orderKey == "" {
		orderKey = o.OrderItemNo
	}
	already, err := uc.repo.IsOrderSynced(ctx, domain.PlatformBlibli, orderKey)
	if err != nil {
		return false, err
	}
	if already {
		return false, nil
	}

	customerName := o.CustomerName
	if customerName == "" {
		customerName = "Blibli Buyer"
	}
	var lines []salesApp.SalesOrderLineDTO
	if p := uc.resolveProduct(ctx, o.ItemSku, o.ItemName); p != nil {
		qty := float64(o.ItemQty)
		if qty <= 0 {
			qty = 1
		}
		lines = append(lines, salesApp.SalesOrderLineDTO{ProductID: p.ID, Quantity: qty, UnitPrice: o.ItemPrice})
	}

	orderID, err := uc.createSyncedSalesOrder(ctx, salesApp.CreateSalesOrderDTO{
		OrderNumber:         fmt.Sprintf("BLI-%s", orderKey),
		CustomerName:        customerName,
		TotalAmount:         o.ItemPrice,
		Status:              mapMarketplaceStatus(o.Status),
		PaymentStatus:       "Paid",
		Channel:             domain.PlatformBlibli.ChannelLabel(),
		WarehouseID:         uc.resolveDefaultWarehouseID(ctx),
		Lines:               lines,
		MarketplacePlatform: string(domain.PlatformBlibli),
		MarketplaceOrderID:  orderKey,
	})
	if err != nil {
		return false, err
	}
	if err := uc.repo.RecordOrderSync(ctx, &domain.MarketplaceOrderSync{
		Platform:           domain.PlatformBlibli,
		MarketplaceOrderID: orderKey,
		SalesOrderID:       orderID,
	}); err != nil {
		return false, err
	}
	return true, nil
}

func (uc *marketplaceUseCase) syncLazadaOrder(ctx context.Context, conn *domain.MarketplaceConnection, o infrastructure.LazadaOrder) (bool, error) {
	orderKey := strconv.FormatInt(o.OrderID, 10)
	already, err := uc.repo.IsOrderSynced(ctx, domain.PlatformLazada, orderKey)
	if err != nil {
		return false, err
	}
	if already {
		return false, nil
	}

	amount, _ := strconv.ParseFloat(o.Price, 64)
	var lines []salesApp.SalesOrderLineDTO
	for _, li := range o.Items {
		p := uc.resolveProduct(ctx, li.SKU, li.Name)
		if p == nil {
			continue
		}
		price, _ := strconv.ParseFloat(li.ItemPrice, 64)
		lines = append(lines, salesApp.SalesOrderLineDTO{ProductID: p.ID, Quantity: 1, UnitPrice: price})
	}

	orderID, err := uc.createSyncedSalesOrder(ctx, salesApp.CreateSalesOrderDTO{
		OrderNumber:         fmt.Sprintf("LZD-%s", orderKey),
		CustomerName:        o.CustomerName(),
		TotalAmount:         amount,
		Status:              mapMarketplaceStatus(strings.ToUpper(o.Status())),
		PaymentStatus:       "Paid",
		Channel:             domain.PlatformLazada.ChannelLabel(),
		WarehouseID:         uc.resolveDefaultWarehouseID(ctx),
		Lines:               lines,
		MarketplacePlatform: string(domain.PlatformLazada),
		MarketplaceOrderID:  orderKey,
	})
	if err != nil {
		return false, err
	}
	if err := uc.repo.RecordOrderSync(ctx, &domain.MarketplaceOrderSync{
		Platform:           domain.PlatformLazada,
		MarketplaceOrderID: orderKey,
		SalesOrderID:       orderID,
	}); err != nil {
		return false, err
	}
	return true, nil
}

// mapMarketplaceStatus normalizes marketplace-specific order status strings
// to this codebase's Sales Order status vocabulary (Confirmed/Processing/
// Delivered/Cancelled), defaulting to "Confirmed" for anything unrecognized
// rather than guessing.
func mapMarketplaceStatus(raw string) string {
	switch raw {
	case "COMPLETED", "COMPLETE":
		return "Delivered"
	case "CANCELLED", "CANCEL", "IN_CANCEL":
		return "Cancelled"
	case "SHIPPED", "TO_SHIP", "PROCESSING", "AWAITING_SHIPMENT", "READY_TO_SHIP":
		return "Processing"
	default:
		return "Confirmed"
	}
}
