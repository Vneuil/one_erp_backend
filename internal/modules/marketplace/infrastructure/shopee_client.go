package infrastructure

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/config"
)

// ShopeeClient talks to the Shopee Open Platform v2 API
// (live host partner.shopeemobile.com; sandbox host for this app's
// region-scoped Test Partner ID is openplatform.sandbox.test-stable.shopee.sg
// - confirmed via Shopee's own API Test Tool, see NewShopeeClient).
//
// API version assumed: v2 order/get_order_list + order/get_order_detail,
// the current stable Order API documented at open.shopee.com/developer-guide
// at the time this was written - confirm exact response field names against
// the live docs before relying on them for production order mapping.
type ShopeeClient struct {
	cfg        config.ShopeeConfig
	httpClient *http.Client
	host       string
}

func NewShopeeClient(cfg config.ShopeeConfig) *ShopeeClient {
	host := "https://partner.shopeemobile.com"
	if cfg.Sandbox {
		// Confirmed against this app's own ISV sandbox test account (Local-ID
		// region, Test Partner ID 1243395) via Shopee's built-in API Test
		// Tool: the classic partner.test-stable.shopeemobile.com host that
		// most third-party SDKs document returns "Wrong sign" for this kind
		// of app even with a byte-for-byte correct signature, because the
		// Test Partner ID/Key is only registered against this region-scoped
		// host. Shopee's dashboard only offers this host or the China
		// (.cn) counterpart - no shopeemobile.com option at all.
		host = "https://openplatform.sandbox.test-stable.shopee.sg"
	}
	return &ShopeeClient{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 20 * time.Second},
		host:       host,
	}
}

func (c *ShopeeClient) Enabled() bool { return c.cfg.Enabled() }

// sign implements Shopee Open Platform v2's public-API signature:
// HMAC-SHA256, keyed by partner_key, over partner_id+path+timestamp (plus
// access_token+shop_id appended once the shop is authorized).
func (c *ShopeeClient) sign(path string, timestamp int64, accessToken, shopIDStr string) string {
	base := fmt.Sprintf("%d%s%d", c.cfg.PartnerID, path, timestamp)
	if accessToken != "" {
		base += accessToken
	}
	if shopIDStr != "" {
		base += shopIDStr
	}
	mac := hmac.New(sha256.New, []byte(c.cfg.PartnerKey))
	mac.Write([]byte(base))
	return hex.EncodeToString(mac.Sum(nil))
}

// AuthorizeURL builds the shop-authorization link
// (GET /api/v2/shop/auth_partner), signed with partner_id+path+timestamp.
// Shopee redirects back to redirectURI with ?code=...&shop_id=....
func (c *ShopeeClient) AuthorizeURL(redirectURI string) string {
	const path = "/api/v2/shop/auth_partner"
	ts := time.Now().Unix()
	sign := c.sign(path, ts, "", "")

	v := url.Values{}
	v.Set("partner_id", strconv.FormatInt(c.cfg.PartnerID, 10))
	v.Set("redirect", redirectURI)
	v.Set("timestamp", strconv.FormatInt(ts, 10))
	v.Set("sign", sign)
	return c.host + path + "?" + v.Encode()
}

type ShopeeTokenResponse struct {
	Error        string  `json:"error"`
	Message      string  `json:"message"`
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
	ExpireIn     int64   `json:"expire_in"`
	ShopIDList   []int64 `json:"shop_id_list"`
}

// ExchangeCode exchanges the authorization `code` + `shop_id` for
// access_token/refresh_token via POST /api/v2/auth/token/get.
func (c *ShopeeClient) ExchangeCode(ctx context.Context, code string, shopID int64) (*ShopeeTokenResponse, error) {
	const path = "/api/v2/auth/token/get"
	payload := map[string]any{
		"code":       code,
		"shop_id":    shopID,
		"partner_id": c.cfg.PartnerID,
	}
	return c.tokenRequest(ctx, path, payload, "", "")
}

// RefreshToken exchanges a refresh_token for a new access_token via
// POST /api/v2/auth/access_token/get.
func (c *ShopeeClient) RefreshToken(ctx context.Context, refreshToken string, shopID int64) (*ShopeeTokenResponse, error) {
	const path = "/api/v2/auth/access_token/get"
	payload := map[string]any{
		"refresh_token": refreshToken,
		"shop_id":       shopID,
		"partner_id":    c.cfg.PartnerID,
	}
	return c.tokenRequest(ctx, path, payload, "", "")
}

func (c *ShopeeClient) tokenRequest(ctx context.Context, path string, payload map[string]any, accessToken, shopIDStr string) (*ShopeeTokenResponse, error) {
	ts := time.Now().Unix()
	sign := c.sign(path, ts, accessToken, shopIDStr)

	q := url.Values{}
	q.Set("partner_id", strconv.FormatInt(c.cfg.PartnerID, 10))
	q.Set("timestamp", strconv.FormatInt(ts, 10))
	q.Set("sign", sign)

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+path+"?"+q.Encode(), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Shopee token API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Shopee token API returned status %d: %s", resp.StatusCode, string(body))
	}

	var parsed ShopeeTokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse Shopee token response: %w", err)
	}
	if parsed.Error != "" {
		return nil, fmt.Errorf("Shopee token API error %s: %s", parsed.Error, parsed.Message)
	}
	return &parsed, nil
}

type ShopeeOrder struct {
	OrderSN       string  `json:"order_sn"`
	OrderStatus   string  `json:"order_status"`
	TotalAmount   float64 `json:"total_amount"`
	Currency      string  `json:"currency"`
	CreateTime    int64   `json:"create_time"`
	BuyerUsername string  `json:"buyer_username"`
	// ItemList is populated when get_order_detail is called with
	// response_optional_fields=item_list - used to build real Sales Order
	// lines instead of a single headline amount (see marketplace
	// application's syncShopeeOrder).
	ItemList []ShopeeOrderItem `json:"item_list"`
}

type ShopeeOrderItem struct {
	ItemSKU              string  `json:"item_sku"`
	ItemName             string  `json:"item_name"`
	ModelQuantity        int     `json:"model_quantity_purchased"`
	ModelDiscountedPrice float64 `json:"model_discounted_price"`
}

type shopeeOrderListResponse struct {
	Error    string `json:"error"`
	Message  string `json:"message"`
	Response struct {
		OrderList []struct {
			OrderSN string `json:"order_sn"`
		} `json:"order_list"`
		More bool `json:"more"`
	} `json:"response"`
}

type shopeeOrderDetailResponse struct {
	Error    string `json:"error"`
	Message  string `json:"message"`
	Response struct {
		OrderList []ShopeeOrder `json:"order_list"`
	} `json:"response"`
}

// ListRecentOrders pulls order_sn list via GET /api/v2/order/get_order_list
// for the time window, then fetches full order details via
// GET /api/v2/order/get_order_detail.
func (c *ShopeeClient) ListRecentOrders(ctx context.Context, accessToken string, shopID int64, since time.Duration) ([]ShopeeOrder, error) {
	const listPath = "/api/v2/order/get_order_list"
	// get_order_list rejects any time_from/time_to span over 15 days
	// ("diff in 15days"), unlike TikTok Shop's order search which tolerates
	// the shared syncLookbackWindow's 30 days - clamp independently here
	// rather than shrinking the window for every platform.
	const maxLookback = 15 * 24 * time.Hour
	if since > maxLookback {
		since = maxLookback
	}
	ts := time.Now().Unix()
	shopIDStr := strconv.FormatInt(shopID, 10)
	sign := c.sign(listPath, ts, accessToken, shopIDStr)

	q := url.Values{}
	q.Set("partner_id", strconv.FormatInt(c.cfg.PartnerID, 10))
	q.Set("timestamp", strconv.FormatInt(ts, 10))
	q.Set("sign", sign)
	q.Set("shop_id", shopIDStr)
	q.Set("access_token", accessToken)
	q.Set("time_range_field", "create_time")
	q.Set("time_from", strconv.FormatInt(time.Now().Add(-since).Unix(), 10))
	q.Set("time_to", strconv.FormatInt(time.Now().Unix(), 10))
	q.Set("page_size", "50")
	// order_status is optional on get_order_list; omitting it returns
	// orders in every status. Passing "ALL" is rejected as invalid by this
	// account's API version ("order_status is invalid"), unlike what some
	// older docs/SDKs suggest.

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.host+listPath+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Shopee order list API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Shopee order list API returned status %d: %s", resp.StatusCode, string(body))
	}

	var listParsed shopeeOrderListResponse
	if err := json.Unmarshal(body, &listParsed); err != nil {
		return nil, fmt.Errorf("failed to parse Shopee order list response: %w", err)
	}
	if listParsed.Error != "" {
		return nil, fmt.Errorf("Shopee order list API error %s: %s", listParsed.Error, listParsed.Message)
	}
	if len(listParsed.Response.OrderList) == 0 {
		return nil, nil
	}

	orderSNs := make([]string, 0, len(listParsed.Response.OrderList))
	for _, o := range listParsed.Response.OrderList {
		orderSNs = append(orderSNs, o.OrderSN)
	}

	return c.getOrderDetail(ctx, accessToken, shopID, orderSNs)
}

func (c *ShopeeClient) getOrderDetail(ctx context.Context, accessToken string, shopID int64, orderSNs []string) ([]ShopeeOrder, error) {
	const path = "/api/v2/order/get_order_detail"
	ts := time.Now().Unix()
	shopIDStr := strconv.FormatInt(shopID, 10)
	sign := c.sign(path, ts, accessToken, shopIDStr)

	q := url.Values{}
	q.Set("partner_id", strconv.FormatInt(c.cfg.PartnerID, 10))
	q.Set("timestamp", strconv.FormatInt(ts, 10))
	q.Set("sign", sign)
	q.Set("shop_id", shopIDStr)
	q.Set("access_token", accessToken)

	joined := ""
	for i, sn := range orderSNs {
		if i > 0 {
			joined += ","
		}
		joined += sn
	}
	q.Set("order_sn_list", joined)
	q.Set("response_optional_fields", "item_list")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.host+path+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Shopee order detail API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Shopee order detail API returned status %d: %s", resp.StatusCode, string(body))
	}

	var parsed shopeeOrderDetailResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse Shopee order detail response: %w", err)
	}
	if parsed.Error != "" {
		return nil, fmt.Errorf("Shopee order detail API error %s: %s", parsed.Error, parsed.Message)
	}
	return parsed.Response.OrderList, nil
}

type shopeeShipOrderResponse struct {
	Error    string `json:"error"`
	Message  string `json:"message"`
	Response struct {
		// Populated when Shopee expects the seller to choose a pickup
		// address/time slot instead of shipping "blind" - see ShipOrder's
		// own comment. Kept as raw JSON since its shape only matters when
		// this specific error occurs.
		Info json.RawMessage `json:"info"`
	} `json:"response"`
}

// ShipOrder marks a Shopee order ready to ship via
// POST /api/v2/logistics/ship_order.
//
// CONFIDENCE NOTE: this is Shopee's documented v2 Logistics API endpoint,
// but the request body Shopee accepts depends on which logistics channel is
// bound to the order - a channel that requires seller pickup scheduling
// needs a "pickup": {"address_id": ..., "pickup_time_id": ...} object (both
// values only obtainable from Shopee's own get_address_list/
// get_pickup_time_id APIs, which this client does not yet implement); a
// dropoff channel needs "dropoff": {...}; a "non_integrated" channel needs
// neither. This sends no pickup/dropoff object at all (the non_integrated
// shape), so an order bound to a channel that requires one will fail with
// Shopee's own explicit error (surfaced back to the caller) rather than
// silently doing the wrong thing - implement the address/time-slot lookup
// before relying on this for a pickup-required channel.
func (c *ShopeeClient) ShipOrder(ctx context.Context, accessToken string, shopID int64, orderSN string) error {
	const path = "/api/v2/logistics/ship_order"
	ts := time.Now().Unix()
	shopIDStr := strconv.FormatInt(shopID, 10)
	sign := c.sign(path, ts, accessToken, shopIDStr)

	q := url.Values{}
	q.Set("partner_id", strconv.FormatInt(c.cfg.PartnerID, 10))
	q.Set("timestamp", strconv.FormatInt(ts, 10))
	q.Set("sign", sign)
	q.Set("shop_id", shopIDStr)
	q.Set("access_token", accessToken)

	bodyBytes, err := json.Marshal(map[string]any{"order_sn": orderSN})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+path+"?"+q.Encode(), bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call Shopee ship_order API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("Shopee ship_order API returned status %d: %s", resp.StatusCode, string(body))
	}

	var parsed shopeeShipOrderResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fmt.Errorf("failed to parse Shopee ship_order response: %w", err)
	}
	if parsed.Error != "" {
		return fmt.Errorf("Shopee ship_order API error %s: %s", parsed.Error, parsed.Message)
	}
	return nil
}
