package infrastructure

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/config"
)

// TikTokShopClient talks to the TikTok Shop Partner Center APIs (auth host
// auth.tiktok-shops.com, API host open-api.tiktokglobalshop.com).
//
// API version assumed: 202309 (the order-list endpoint path below,
// /order/202309/orders/search, is the stable version documented as current
// at the time this was written - confirm against
// https://partner.tiktokshop.com/docv2 before going live, since TikTok
// periodically ships newer dated versions of the Order API).
type TikTokShopClient struct {
	cfg        config.TikTokShopConfig
	httpClient *http.Client
}

func NewTikTokShopClient(cfg config.TikTokShopConfig) *TikTokShopClient {
	return &TikTokShopClient{cfg: cfg, httpClient: &http.Client{Timeout: 20 * time.Second}}
}

func (c *TikTokShopClient) Enabled() bool { return c.cfg.Enabled() }

// AuthorizeURL builds the link the seller visits to grant access to their
// shop. TikTok Shop redirects back to the app's registered redirect URI with
// a `code` (auth_code) and `shop_id`/`state` query params.
//
// CORRECTION (2026-09-10): this previously pointed at
// "https://services.tiktokshops.com/open/authorize" with a service_id param
// - that host does not resolve (DNS_PROBE_FINISHED_NXDOMAIN, confirmed live
// in production) and appears to have been a stale/incorrect domain. Switched
// to auth.tiktok-shops.com/oauth/authorize (the same host this client's own
// token exchange below already calls), which is TikTok Shop Partner Center's
// documented authorization endpoint, with the app_key param it expects.
func (c *TikTokShopClient) AuthorizeURL(state string) string {
	v := url.Values{}
	v.Set("app_key", c.cfg.AppKey)
	v.Set("state", state)
	return "https://auth.tiktok-shops.com/oauth/authorize?" + v.Encode()
}

type TikTokTokenResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		AccessToken          string `json:"access_token"`
		AccessTokenExpireIn  int64  `json:"access_token_expire_in"`
		RefreshToken         string `json:"refresh_token"`
		RefreshTokenExpireIn int64  `json:"refresh_token_expire_in"`
		OpenID               string `json:"open_id"`
		SellerName           string `json:"seller_name"`
	} `json:"data"`
}

// ExchangeCode exchanges an auth_code for access_token/refresh_token via
// POST https://auth.tiktok-shops.com/api/v2/token/get (grant_type=authorized_code).
func (c *TikTokShopClient) ExchangeCode(ctx context.Context, authCode string) (*TikTokTokenResponse, error) {
	return c.tokenRequest(ctx, url.Values{
		"app_key":    {c.cfg.AppKey},
		"app_secret": {c.cfg.AppSecret},
		"auth_code":  {authCode},
		"grant_type": {"authorized_code"},
	})
}

// RefreshToken exchanges a refresh_token for a new access_token via the same
// endpoint with grant_type=refresh_token.
func (c *TikTokShopClient) RefreshToken(ctx context.Context, refreshToken string) (*TikTokTokenResponse, error) {
	return c.tokenRequest(ctx, url.Values{
		"app_key":       {c.cfg.AppKey},
		"app_secret":    {c.cfg.AppSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	})
}

func (c *TikTokShopClient) tokenRequest(ctx context.Context, params url.Values) (*TikTokTokenResponse, error) {
	reqURL := "https://auth.tiktok-shops.com/api/v2/token/get?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call TikTok Shop token API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("TikTok Shop token API returned status %d: %s", resp.StatusCode, string(body))
	}

	var parsed TikTokTokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse TikTok Shop token response: %w", err)
	}
	if parsed.Code != 0 {
		return nil, fmt.Errorf("TikTok Shop token API error %d: %s", parsed.Code, parsed.Message)
	}
	return &parsed, nil
}

// sign implements TikTok Shop's request signature: HMAC-SHA256, keyed by
// app_secret, over app_key + sorted(query params except sign/access_token) +
// raw request body, wrapped with app_secret on both ends.
func (c *TikTokShopClient) sign(path string, query url.Values, body []byte) string {
	keys := make([]string, 0, len(query))
	for k := range query {
		if k == "sign" || k == "access_token" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString(path)
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString(query.Get(k))
	}
	base := c.cfg.AppSecret + sb.String() + string(body) + c.cfg.AppSecret

	mac := hmac.New(sha256.New, []byte(c.cfg.AppSecret))
	mac.Write([]byte(base))
	return hex.EncodeToString(mac.Sum(nil))
}

type TikTokOrder struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	CreateTime  int64  `json:"create_time"`
	PaymentInfo struct {
		TotalAmount string `json:"total_amount"`
		Currency    string `json:"currency"`
	} `json:"payment_info"`
	RecipientAddress struct {
		Name string `json:"name"`
	} `json:"recipient_address"`
	BuyerEmail string `json:"buyer_email"`
	// LineItems is populated by TikTok Shop's order search/detail response
	// for each SKU in the order - used to build real Sales Order lines
	// instead of a single headline amount (see marketplace application's
	// syncTikTokOrder).
	LineItems []TikTokLineItem `json:"line_items"`
	// Packages carries the fulfillment package(s) TikTok Shop groups this
	// order's line items into - required to call ShipPackage, since TikTok
	// Shop's Fulfillment API ships a package, not an order directly.
	Packages []struct {
		ID string `json:"id"`
	} `json:"packages"`
}

type TikTokLineItem struct {
	SellerSKU   string `json:"seller_sku"`
	ProductName string `json:"product_name"`
	SalePrice   string `json:"sale_price"`
	// TikTok Shop returns one line-item entry per unit rather than a
	// quantity field on multi-unit lines; callers should treat each entry
	// as quantity 1 unless a future API version adds an explicit count.
}

type tiktokOrderSearchResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Orders        []TikTokOrder `json:"orders"`
		NextPageToken string        `json:"next_page_token"`
	} `json:"data"`
}

// SearchRecentOrders pulls orders created in the last `since` window for the
// connected shop. shopCipher identifies the shop (returned at authorization
// time / from GET /authorization/202309/shops).
func (c *TikTokShopClient) SearchRecentOrders(ctx context.Context, accessToken, shopCipher string, since time.Duration) ([]TikTokOrder, error) {
	const path = "/order/202309/orders/search"
	createTimeGE := time.Now().Add(-since).Unix()

	bodyPayload := map[string]any{
		"create_time_ge": createTimeGE,
	}
	bodyBytes, err := json.Marshal(bodyPayload)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("app_key", c.cfg.AppKey)
	query.Set("shop_cipher", shopCipher)
	query.Set("timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	query.Set("page_size", "50")
	sign := c.sign(path, query, bodyBytes)
	query.Set("sign", sign)

	reqURL := "https://open-api.tiktokglobalshop.com" + path + "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-tts-access-token", accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call TikTok Shop order search API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("TikTok Shop order search API returned status %d: %s", resp.StatusCode, string(body))
	}

	var parsed tiktokOrderSearchResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse TikTok Shop order search response: %w", err)
	}
	if parsed.Code != 0 {
		return nil, fmt.Errorf("TikTok Shop order search API error %d: %s", parsed.Code, parsed.Message)
	}
	return parsed.Data.Orders, nil
}

type tiktokOrderGetResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Orders []TikTokOrder `json:"orders"`
	} `json:"data"`
}

// GetOrder fetches one order's current detail (including its fulfillment
// packages) via GET /order/202309/orders - called before ShipPackage since
// shipping is per-package, not per-order, and the order search result used
// to build the order in the first place doesn't necessarily carry
// up-to-date package info.
func (c *TikTokShopClient) GetOrder(ctx context.Context, accessToken, shopCipher, orderID string) (*TikTokOrder, error) {
	const path = "/order/202309/orders"

	query := url.Values{}
	query.Set("app_key", c.cfg.AppKey)
	query.Set("shop_cipher", shopCipher)
	query.Set("timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	query.Set("ids", orderID)
	sign := c.sign(path, query, nil)
	query.Set("sign", sign)

	reqURL := "https://open-api.tiktokglobalshop.com" + path + "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-tts-access-token", accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call TikTok Shop get order API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("TikTok Shop get order API returned status %d: %s", resp.StatusCode, string(body))
	}

	var parsed tiktokOrderGetResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse TikTok Shop get order response: %w", err)
	}
	if parsed.Code != 0 {
		return nil, fmt.Errorf("TikTok Shop get order API error %d: %s", parsed.Code, parsed.Message)
	}
	if len(parsed.Data.Orders) == 0 {
		return nil, fmt.Errorf("TikTok Shop order %s not found", orderID)
	}
	return &parsed.Data.Orders[0], nil
}

type tiktokShipPackageResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ShipPackage marks a fulfillment package ready for pickup via
// POST /fulfillment/202309/packages/{package_id}/ship.
//
// CONFIDENCE NOTE: the 202309 Fulfillment API's ship endpoint accepts a
// "handover_method" of PICKUP or DROPOFF; PICKUP additionally documents an
// optional "pickup_slot" ({start_time, end_time}, obtained from that
// package's GET .../shipping_document or a slots-listing endpoint this
// client does not implement). This omits pickup_slot - if TikTok Shop
// requires it for this seller's carrier, the API's own error message is
// returned to the caller rather than guessed at.
func (c *TikTokShopClient) ShipPackage(ctx context.Context, accessToken, shopCipher, packageID string) error {
	path := fmt.Sprintf("/fulfillment/202309/packages/%s/ship", packageID)

	bodyBytes, err := json.Marshal(map[string]any{"handover_method": "PICKUP"})
	if err != nil {
		return err
	}

	query := url.Values{}
	query.Set("app_key", c.cfg.AppKey)
	query.Set("shop_cipher", shopCipher)
	query.Set("timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	sign := c.sign(path, query, bodyBytes)
	query.Set("sign", sign)

	reqURL := "https://open-api.tiktokglobalshop.com" + path + "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-tts-access-token", accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call TikTok Shop ship package API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("TikTok Shop ship package API returned status %d: %s", resp.StatusCode, string(body))
	}

	var parsed tiktokShipPackageResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fmt.Errorf("failed to parse TikTok Shop ship package response: %w", err)
	}
	if parsed.Code != 0 {
		return fmt.Errorf("TikTok Shop ship package API error %d: %s", parsed.Code, parsed.Message)
	}
	return nil
}

// TikTokShop is one shop the seller authorized for this app.
type TikTokShop struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Cipher string `json:"cipher"`
	Region string `json:"region"`
}

// signedCall performs a signed Partner Center API call and returns the raw
// response body after checking the HTTP status and TikTok's envelope code.
func (c *TikTokShopClient) signedCall(ctx context.Context, method, path, accessToken, shopCipher string, extra url.Values, payload any) (json.RawMessage, error) {
	var bodyBytes []byte
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		bodyBytes = b
	}
	query := url.Values{}
	for k, v := range extra {
		query[k] = v
	}
	query.Set("app_key", c.cfg.AppKey)
	query.Set("timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	if shopCipher != "" {
		query.Set("shop_cipher", shopCipher)
	}
	query.Set("sign", c.sign(path, query, bodyBytes))

	reqURL := "https://open-api.tiktokglobalshop.com" + path + "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, method, reqURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-tts-access-token", accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call TikTok Shop API %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("TikTok Shop API %s returned status %d: %s", path, resp.StatusCode, string(body))
	}
	var env struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("failed to parse TikTok Shop response from %s: %w", path, err)
	}
	if env.Code != 0 {
		return nil, fmt.Errorf("TikTok Shop API %s error %d: %s", path, env.Code, env.Message)
	}
	return env.Data, nil
}

// GetAuthorizedShops lists the shops granted to this app via
// GET /authorization/202309/shops. The shop `cipher` it returns is required
// as shop_cipher on every order/product call.
func (c *TikTokShopClient) GetAuthorizedShops(ctx context.Context, accessToken string) ([]TikTokShop, error) {
	data, err := c.signedCall(ctx, http.MethodGet, "/authorization/202309/shops", accessToken, "", nil, nil)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Shops []TikTokShop `json:"shops"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse TikTok Shop shops response: %w", err)
	}
	return parsed.Shops, nil
}

// TikTokProduct is one product from the TikTok Shop catalog. ID is TikTok's
// own product id (an 18-19 digit number starting with 17 for Indonesia).
type TikTokProduct struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	SKUs   []struct {
		ID        string `json:"id"`
		SellerSKU string `json:"seller_sku"`
		Price     struct {
			SalePrice string `json:"sale_price"`
			Currency  string `json:"currency"`
		} `json:"price"`
		Inventory []struct {
			Quantity int `json:"quantity"`
		} `json:"inventory"`
	} `json:"skus"`
}

// SearchProducts pages through the shop catalog via
// POST /product/202309/products/search.
func (c *TikTokShopClient) SearchProducts(ctx context.Context, accessToken, shopCipher string) ([]TikTokProduct, error) {
	var all []TikTokProduct
	pageToken := ""
	for page := 0; page < 20; page++ {
		q := url.Values{}
		q.Set("page_size", "50")
		if pageToken != "" {
			q.Set("page_token", pageToken)
		}
		data, err := c.signedCall(ctx, http.MethodPost, "/product/202309/products/search", accessToken, shopCipher, q, map[string]any{})
		if err != nil {
			return nil, err
		}
		var parsed struct {
			Products      []TikTokProduct `json:"products"`
			NextPageToken string          `json:"next_page_token"`
		}
		if err := json.Unmarshal(data, &parsed); err != nil {
			return nil, fmt.Errorf("failed to parse TikTok Shop product search response: %w", err)
		}
		all = append(all, parsed.Products...)
		if parsed.NextPageToken == "" {
			break
		}
		pageToken = parsed.NextPageToken
	}
	return all, nil
}
