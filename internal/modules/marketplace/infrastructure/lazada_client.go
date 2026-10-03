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

// LazadaClient talks to the Lazada Open Platform (open.lazada.com) Order API
// and OAuth token endpoints.
//
// CONFIDENCE NOTE (verify against a real Lazada Open Platform sandbox/app
// before relying on this in production - unlike BlibliClient, this was NOT
// verified against an official reference client, only against Lazada Open
// Platform's long-stable, publicly documented conventions):
//   - Auth host: auth.lazada.com (authorize page + token create/refresh).
//   - Data/REST host: this client uses the generic api.lazada.com/rest
//     gateway. Some integrations instead call a country-specific host
//     (e.g. api.lazada.co.id/rest) for order data - if orders/get returns
//     an "IncompleteSignature"/host-mismatch error in practice, switch
//     dataHost to the country-specific domain for the seller's market.
//   - Signing: HMAC-SHA256 (hex, uppercase) over
//     apiPath + concatenated "key"+"value" pairs of every non-file
//     parameter (sign excluded), sorted by key ascending - keyed by
//     AppSecret. This is Lazada's well-documented "Signing Requests"
//     algorithm and has been stable for years, but was not tested against
//     a live account while writing this.
type LazadaClient struct {
	cfg        config.LazadaConfig
	httpClient *http.Client
	authHost   string
	dataHost   string
}

func NewLazadaClient(cfg config.LazadaConfig) *LazadaClient {
	return &LazadaClient{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 20 * time.Second},
		authHost:   "https://auth.lazada.com",
		dataHost:   "https://api.lazada.com",
	}
}

func (c *LazadaClient) Enabled() bool { return c.cfg.Enabled() }

// sign implements Lazada Open Platform's request signature: HMAC-SHA256,
// keyed by AppSecret, over apiPath followed by every non-"sign" parameter's
// key+value concatenated in ascending key order, hex-encoded uppercase.
func (c *LazadaClient) sign(apiPath string, params url.Values) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "sign" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString(apiPath)
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString(params.Get(k))
	}

	mac := hmac.New(sha256.New, []byte(c.cfg.AppSecret))
	mac.Write([]byte(b.String()))
	return strings.ToUpper(hex.EncodeToString(mac.Sum(nil)))
}

// AuthorizeURL builds the link the seller visits to grant access to their
// Lazada shop. Lazada redirects back to redirectURI with a `code` query
// param (and echoes `state` back unchanged).
func (c *LazadaClient) AuthorizeURL(redirectURI, state string) string {
	v := url.Values{}
	v.Set("response_type", "code")
	v.Set("force_auth", "true")
	v.Set("redirect_uri", redirectURI)
	v.Set("client_id", c.cfg.AppKey)
	v.Set("state", state)
	return c.authHost + "/oauth/authorize?" + v.Encode()
}

type LazadaTokenResponse struct {
	Code         string `json:"code"`
	Message      string `json:"message"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Country      string `json:"country"`
	AccountID    string `json:"account_id"` // seller account identifier, used as ShopID
	Account      string `json:"account"`    // seller-facing account/shop name
}

func (t *LazadaTokenResponse) ok() bool {
	return t.Code == "" || t.Code == "0"
}

// ExchangeCode exchanges an authorization `code` for access_token/refresh_token
// via POST auth.lazada.com/rest/auth/token/create.
func (c *LazadaClient) ExchangeCode(ctx context.Context, code string) (*LazadaTokenResponse, error) {
	return c.tokenRequest(ctx, "/auth/token/create", url.Values{"code": {code}})
}

// RefreshToken exchanges a refresh_token for a new access_token via
// POST auth.lazada.com/rest/auth/token/refresh.
func (c *LazadaClient) RefreshToken(ctx context.Context, refreshToken string) (*LazadaTokenResponse, error) {
	return c.tokenRequest(ctx, "/auth/token/refresh", url.Values{"refresh_token": {refreshToken}})
}

func (c *LazadaClient) tokenRequest(ctx context.Context, apiPath string, extra url.Values) (*LazadaTokenResponse, error) {
	q := url.Values{}
	q.Set("app_key", c.cfg.AppKey)
	q.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	q.Set("sign_method", "sha256")
	for k, vs := range extra {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	q.Set("sign", c.sign(apiPath, q))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.authHost+"/rest"+apiPath+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Lazada token API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Lazada token API returned status %d: %s", resp.StatusCode, string(body))
	}

	var parsed LazadaTokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse Lazada token response: %w", err)
	}
	if !parsed.ok() {
		return nil, fmt.Errorf("Lazada token API error %s: %s", parsed.Code, parsed.Message)
	}
	return &parsed, nil
}

// LazadaOrder is the subset of Lazada's orders/get response this
// integration maps into a Sales Order.
type LazadaOrder struct {
	OrderID       int64    `json:"order_id"`
	OrderNumber   string   `json:"order_number"`
	Statuses      []string `json:"statuses"`
	Price         string   `json:"price"`
	CreatedAt     string   `json:"created_at"`
	CustomerFirst string   `json:"customer_first_name"`
	CustomerLast  string   `json:"customer_last_name"`
	// Items is populated separately (see GetOrderItems) since Lazada's
	// orders/get response carries no line items - used to build real Sales
	// Order lines instead of a single headline amount (see marketplace
	// application's syncLazadaOrder).
	Items []LazadaOrderItem `json:"-"`
}

type LazadaOrderItem struct {
	OrderItemID int64  `json:"order_item_id"`
	SKU         string `json:"sku"`
	Name        string `json:"name"`
	ItemPrice   string `json:"item_price"`
	ShopSKU     string `json:"shop_sku"`
}

type lazadaOrderItemsResponse struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Data    []LazadaOrderItem `json:"data"`
}

// GetOrderItems fetches line items for one order via
// GET api.lazada.com/rest/order/items/get. Called per-order after
// ListRecentOrders since Lazada's order list response carries no items.
func (c *LazadaClient) GetOrderItems(ctx context.Context, accessToken string, orderID int64) ([]LazadaOrderItem, error) {
	const apiPath = "/order/items/get"

	q := url.Values{}
	q.Set("app_key", c.cfg.AppKey)
	q.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	q.Set("sign_method", "sha256")
	q.Set("access_token", accessToken)
	q.Set("order_id", strconv.FormatInt(orderID, 10))
	q.Set("sign", c.sign(apiPath, q))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.dataHost+"/rest"+apiPath+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Lazada order/items/get API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Lazada order/items/get API returned status %d: %s", resp.StatusCode, string(body))
	}

	var parsed lazadaOrderItemsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse Lazada order/items/get response: %w", err)
	}
	if parsed.Code != "" && parsed.Code != "0" {
		return nil, fmt.Errorf("Lazada order/items/get API error %s: %s", parsed.Code, parsed.Message)
	}
	return parsed.Data, nil
}

// Status returns the order's current (first) status string, or "" if none.
func (o LazadaOrder) Status() string {
	if len(o.Statuses) > 0 {
		return o.Statuses[0]
	}
	return ""
}

// CustomerName joins the buyer's first/last name, falling back to a generic
// label when Lazada doesn't return one.
func (o LazadaOrder) CustomerName() string {
	name := strings.TrimSpace(o.CustomerFirst + " " + o.CustomerLast)
	if name == "" {
		return "Lazada Buyer"
	}
	return name
}

type lazadaOrderListResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Orders     []LazadaOrder `json:"orders"`
		CountTotal int           `json:"countTotal"`
	} `json:"data"`
}

// ListRecentOrders fetches orders created within `since` via
// GET api.lazada.com/rest/orders/get.
func (c *LazadaClient) ListRecentOrders(ctx context.Context, accessToken string, since time.Duration) ([]LazadaOrder, error) {
	const apiPath = "/orders/get"

	q := url.Values{}
	q.Set("app_key", c.cfg.AppKey)
	q.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	q.Set("sign_method", "sha256")
	q.Set("access_token", accessToken)
	q.Set("created_after", time.Now().Add(-since).UTC().Format(time.RFC3339))
	q.Set("sort_by", "created_at")
	q.Set("sort_direction", "DESC")
	q.Set("offset", "0")
	q.Set("limit", "100")
	q.Set("sign", c.sign(apiPath, q))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.dataHost+"/rest"+apiPath+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Lazada orders/get API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Lazada orders/get API returned status %d: %s", resp.StatusCode, string(body))
	}

	var parsed lazadaOrderListResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse Lazada orders/get response: %w", err)
	}
	if parsed.Code != "" && parsed.Code != "0" {
		return nil, fmt.Errorf("Lazada orders/get API error %s: %s", parsed.Code, parsed.Message)
	}

	orders := parsed.Data.Orders
	for i := range orders {
		items, err := c.GetOrderItems(ctx, accessToken, orders[i].OrderID)
		if err != nil {
			// Best-effort: a per-order item fetch failure shouldn't fail the
			// whole sync - the order still syncs with no lines, same as
			// before this line-item support was added.
			continue
		}
		orders[i].Items = items
	}
	return orders, nil
}

type lazadaRTSResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SetReadyToShip marks an order's items ready to ship via
// GET api.lazada.com/rest/order/rts.
//
// CONFIDENCE NOTE (same caveat as this whole client - not verified against
// a live account): Lazada's RTS endpoint takes order_item_ids as a
// JSON-array-shaped string (e.g. "[123,456]"), and optionally
// delivery_type ("dropship" for seller-arranged/pickup fulfillment vs
// Lazada's own integrated logistics, which needs no delivery_type at all).
// Since "siap pickup" means the seller arranges pickup rather than using an
// integrated carrier, this always sends delivery_type=dropship - if this
// seller's shipment provider is actually one of Lazada's own integrated
// couriers, omit delivery_type instead (or add a shipment_provider param;
// its exact accepted values could not be confirmed without live docs).
func (c *LazadaClient) SetReadyToShip(ctx context.Context, accessToken string, orderItemIDs []int64) error {
	const apiPath = "/order/rts"

	idsJSON, err := json.Marshal(orderItemIDs)
	if err != nil {
		return err
	}

	q := url.Values{}
	q.Set("app_key", c.cfg.AppKey)
	q.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	q.Set("sign_method", "sha256")
	q.Set("access_token", accessToken)
	q.Set("order_item_ids", string(idsJSON))
	q.Set("delivery_type", "dropship")
	q.Set("sign", c.sign(apiPath, q))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.dataHost+"/rest"+apiPath+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call Lazada order/rts API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("Lazada order/rts API returned status %d: %s", resp.StatusCode, string(body))
	}

	var parsed lazadaRTSResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fmt.Errorf("failed to parse Lazada order/rts response: %w", err)
	}
	if parsed.Code != "" && parsed.Code != "0" {
		return fmt.Errorf("Lazada order/rts API error %s: %s", parsed.Code, parsed.Message)
	}
	return nil
}
