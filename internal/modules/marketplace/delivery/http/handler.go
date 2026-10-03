package http

import (
	"context"
	"net/url"
	"strconv"

	"github.com/divinecoid/one-backend/internal/foundation/config"
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	approvalApp "github.com/divinecoid/one-backend/internal/modules/approval/application"
	approvalInfra "github.com/divinecoid/one-backend/internal/modules/approval/infrastructure"
	currencyApp "github.com/divinecoid/one-backend/internal/modules/currency/application"
	currencyInfra "github.com/divinecoid/one-backend/internal/modules/currency/infrastructure"
	inventoryApp "github.com/divinecoid/one-backend/internal/modules/inventory/application"
	inventoryInfra "github.com/divinecoid/one-backend/internal/modules/inventory/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/marketplace/application"
	"github.com/divinecoid/one-backend/internal/modules/marketplace/domain"
	"github.com/divinecoid/one-backend/internal/modules/marketplace/infrastructure"
	productInfra "github.com/divinecoid/one-backend/internal/modules/product/infrastructure"
	salesApp "github.com/divinecoid/one-backend/internal/modules/sales/application"
	salesInfra "github.com/divinecoid/one-backend/internal/modules/sales/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	tiktokCfg   config.TikTokShopConfig
	shopeeCfg   config.ShopeeConfig
	blibliCfg   config.BlibliConfig
	lazadaCfg   config.LazadaConfig
	frontendURL string
	jwtSecret   string
	manager     *tenantMgr.Manager
}

func NewHandler(tiktokCfg config.TikTokShopConfig, shopeeCfg config.ShopeeConfig, blibliCfg config.BlibliConfig, lazadaCfg config.LazadaConfig, frontendURL, jwtSecret string, manager *tenantMgr.Manager) *Handler {
	return &Handler{tiktokCfg: tiktokCfg, shopeeCfg: shopeeCfg, blibliCfg: blibliCfg, lazadaCfg: lazadaCfg, frontendURL: frontendURL, jwtSecret: jwtSecret, manager: manager}
}

// useCaseForDB wires the marketplace use case to a real sales.SalesUseCase
// (same construction as sales/delivery/http/handler.go's resolve) so
// synced marketplace orders go through the same stock-checked CreateOrder
// path as manually created Sales Orders, instead of writing bare header
// rows directly via salesRepo.
func (h *Handler) useCaseForDB(tenantDB *gorm.DB) application.MarketplaceUseCase {
	repo := infrastructure.NewMarketplaceRepository(tenantDB)
	salesRepo := salesInfra.NewSalesRepository(tenantDB)

	currencyRepo := currencyInfra.NewCurrencyRepository(tenantDB)
	currencyUC := currencyApp.NewCurrencyUseCase(currencyRepo)
	inventoryRepo := inventoryInfra.NewInventoryRepository(tenantDB)
	productRepo := productInfra.NewProductRepository(tenantDB)
	inventoryUC := inventoryApp.NewInventoryUseCase(inventoryRepo, productRepo)
	approvalRepo := approvalInfra.NewApprovalRepository(tenantDB)
	approvalUC := approvalApp.NewApprovalUseCase(approvalRepo)
	salesUC := salesApp.NewSalesUseCase(salesRepo, currencyUC, inventoryUC, approvalUC)

	return application.NewMarketplaceUseCase(repo, salesRepo, salesUC, productRepo, inventoryUC, h.tiktokCfg, h.shopeeCfg, h.blibliCfg, h.lazadaCfg)
}

// resolve builds a use-case bound to the caller's own tenant database, the
// same pattern used by the sales module's handler. Used by every route that
// runs behind Protected+TenantContext (i.e. everything except the OAuth
// provider callbacks - see resolveFromToken).
func (h *Handler) resolve(c *fiber.Ctx) (application.MarketplaceUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	return h.useCaseForDB(tenantDB), nil
}

// ctx returns the request context with the active Tenant ID (see
// modules/workspace) attached, so tenant-aware repository queries can read
// it via tenantctx.FromContext without every usecase method needing an
// extra parameter. A nil tenant ID (the common case) means "no filter".
// Not used by the OAuth callback routes (TikTokCallback/ShopeeCallback and
// resolveFromToken), which resolve identity from a token round-tripped
// through the OAuth state param rather than middleware.CurrentUser, and so
// keep using c.UserContext() directly.
func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

// resolveFromToken resolves the tenant database for the TikTok Shop /
// Shopee OAuth redirect callbacks. These are plain browser GET redirects
// initiated by the marketplace's own servers, not our frontend's XHR client,
// so they carry no Authorization header and can't run behind the normal
// Protected+TenantContext middleware. Instead, the same user's JWT (already
// validated once when they clicked "Connect") is round-tripped through the
// OAuth `state` param (TikTok) or appended to our own redirect URI (Shopee)
// and re-validated here exactly like Protected would.
func (h *Handler) resolveFromToken(c *fiber.Ctx, token string) (application.MarketplaceUseCase, error) {
	if token == "" {
		return nil, apperrors.NewUnauthorized("Missing session token in OAuth callback")
	}
	claims, err := utils.ParseJWT(token, h.jwtSecret)
	if err != nil {
		return nil, apperrors.NewUnauthorized("Invalid or expired session token in OAuth callback")
	}
	if claims.CompanyID == nil {
		return nil, apperrors.NewBadRequest("No active company for this session")
	}
	tenantDB, err := h.manager.GetDB(c.UserContext(), *claims.CompanyID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to resolve tenant database")
	}
	return h.useCaseForDB(tenantDB), nil
}

func (h *Handler) ListConnections(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	conns, err := uc.ListConnections(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Marketplace connections retrieved successfully", conns)
}

// bearerToken extracts the raw JWT from this request's own Authorization
// header (stripping the "Bearer " prefix), so it can be threaded through to
// the OAuth callback.
func bearerToken(c *fiber.Ctx) string {
	auth := c.Get("Authorization")
	const prefix = "Bearer "
	if len(auth) > len(prefix) && auth[:len(prefix)] == prefix {
		return auth[len(prefix):]
	}
	return ""
}

func (h *Handler) TikTokConnect(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	token := bearerToken(c)
	if token == "" {
		return apperrors.NewUnauthorized("Missing session token")
	}
	authURL, err := uc.TikTokConnectURL(h.ctx(c), token)
	if err != nil {
		return err
	}
	return response.OK(c, "TikTok Shop authorize URL generated", fiber.Map{"authorizeUrl": authURL})
}

func (h *Handler) TikTokCallback(c *fiber.Ctx) error {
	uc, err := h.resolveFromToken(c, c.Query("state"))
	if err != nil {
		return h.redirectWithError(c, "tiktok_shop", err)
	}
	code := c.Query("code")
	if _, err := uc.TikTokHandleCallback(c.UserContext(), code); err != nil {
		return h.redirectWithError(c, "tiktok_shop", err)
	}
	return c.Redirect(h.frontendURL+"/settings/integration?marketplace=tiktok_shop&status=connected", fiber.StatusFound)
}

func (h *Handler) ShopeeConnect(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	token := bearerToken(c)
	if token == "" {
		return apperrors.NewUnauthorized("Missing session token")
	}
	// Shopee's shop/auth_partner endpoint has no state param, so the session
	// token rides along on our own redirect URI's query string instead; the
	// provider preserves it and appends its own code/shop_id params.
	redirectURI := h.shopeeCfg.RedirectURI + "?token=" + url.QueryEscape(token)
	authURL, err := uc.ShopeeConnectURLWithRedirect(h.ctx(c), redirectURI)
	if err != nil {
		return err
	}
	return response.OK(c, "Shopee authorize URL generated", fiber.Map{"authorizeUrl": authURL})
}

func (h *Handler) ShopeeCallback(c *fiber.Ctx) error {
	uc, err := h.resolveFromToken(c, c.Query("token"))
	if err != nil {
		return h.redirectWithError(c, "shopee", err)
	}
	code := c.Query("code")
	shopID, _ := strconv.ParseInt(c.Query("shop_id"), 10, 64)
	if _, err := uc.ShopeeHandleCallback(c.UserContext(), code, shopID); err != nil {
		return h.redirectWithError(c, "shopee", err)
	}
	return c.Redirect(h.frontendURL+"/settings/integration?marketplace=shopee&status=connected", fiber.StatusFound)
}

// blibliConnectRequest is the JSON body for POST /marketplace/blibli/connect
// - the 4 values a seller copies from their own Blibli Seller Center
// account (User Profile > Seller API Manager). There is no redirect/OAuth
// flow for Blibli, so this is a plain authenticated POST, not a callback.
type blibliConnectRequest struct {
	BusinessPartnerCode string `json:"businessPartnerCode"`
	MtaUsername         string `json:"mtaUsername"`
	ApiSellerKey        string `json:"apiSellerKey"`
	SignatureKey        string `json:"signatureKey"`
}

func (h *Handler) BlibliConnect(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var body blibliConnectRequest
	if err := c.BodyParser(&body); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	conn, err := uc.ConnectBlibli(h.ctx(c), body.BusinessPartnerCode, body.MtaUsername, body.ApiSellerKey, body.SignatureKey)
	if err != nil {
		return err
	}
	return response.OK(c, "Blibli connected successfully", conn)
}

func (h *Handler) LazadaConnect(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	token := bearerToken(c)
	if token == "" {
		return apperrors.NewUnauthorized("Missing session token")
	}
	authURL, err := uc.LazadaConnectURL(h.ctx(c), token)
	if err != nil {
		return err
	}
	return response.OK(c, "Lazada authorize URL generated", fiber.Map{"authorizeUrl": authURL})
}

func (h *Handler) LazadaCallback(c *fiber.Ctx) error {
	uc, err := h.resolveFromToken(c, c.Query("state"))
	if err != nil {
		return h.redirectWithError(c, "lazada", err)
	}
	code := c.Query("code")
	if _, err := uc.LazadaHandleCallback(c.UserContext(), code); err != nil {
		return h.redirectWithError(c, "lazada", err)
	}
	return c.Redirect(h.frontendURL+"/settings/integration?marketplace=lazada&status=connected", fiber.StatusFound)
}

func (h *Handler) Sync(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	platform := domain.Platform(c.Params("platform"))
	if !platform.Valid() {
		return apperrors.NewBadRequest("Unsupported marketplace platform")
	}
	result, err := uc.Sync(h.ctx(c), platform)
	if err != nil {
		return err
	}
	return response.OK(c, "Marketplace sync completed", result)
}

func (h *Handler) MarkOrderReadyToShip(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	salesOrderID, err := uuid.Parse(c.Params("salesOrderId"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid sales order ID")
	}
	delivery, err := uc.MarkOrderReadyToShip(h.ctx(c), salesOrderID)
	if err != nil {
		return err
	}
	return response.OK(c, "Order marked ready to ship", delivery)
}

func (h *Handler) Disconnect(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	platform := domain.Platform(c.Params("platform"))
	if !platform.Valid() {
		return apperrors.NewBadRequest("Unsupported marketplace platform")
	}
	if err := uc.Disconnect(h.ctx(c), platform); err != nil {
		return err
	}
	return response.OK(c, "Marketplace disconnected successfully", nil)
}

func (h *Handler) redirectWithError(c *fiber.Ctx, platform string, err error) error {
	msg := "connection_failed"
	if appErr, ok := err.(*apperrors.AppError); ok {
		msg = appErr.Message
	}
	return c.Redirect(h.frontendURL+"/settings/integration?marketplace="+platform+"&status=error&message="+url.QueryEscape(msg), fiber.StatusFound)
}
