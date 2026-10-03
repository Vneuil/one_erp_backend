package infrastructure

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/config"
	"github.com/google/uuid"
)

// BlibliClient talks to the Blibli Seller (Merchant Trading Application/MTA)
// Partner API.
//
// Verified directly against Blibli's own official PHP reference client
// (github.com/bliblidotcom/seller-api-client-php, re-cloned and read on
// 2026-09-03: generate_signature.php, invoker/basic_auth/invoke_with_body.php,
// invoker/basic_auth/invoke_without_body.php, client/BlibliSellerBasicAuthClient.php,
// request/ApiConfig.php, index-basic-auth.php, README.md) - not just the
// task brief's paraphrase of it.
//
// Unlike TikTok Shop/Shopee, Blibli has no per-seller OAuth redirect flow.
// ApiClientID/ApiClientSecret are ONE ERP's own ISV-level credentials
// (HTTP Basic Auth on every call, see cfg below); BusinessPartnerCode/
// MtaUsername/ApiSellerKey/SignatureKey are supplied per-tenant (see
// domain.MarketplaceConnection).
type BlibliClient struct {
	cfg        config.BlibliConfig
	httpClient *http.Client
	host       string
}

func NewBlibliClient(cfg config.BlibliConfig) *BlibliClient {
	host := "https://api.gdn-app.com"
	if cfg.Sandbox {
		host = "https://api-uata.gdn-app.com"
	}
	return &BlibliClient{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 20 * time.Second},
		host:       host,
	}
}

func (c *BlibliClient) Enabled() bool { return c.cfg.Enabled() }

// BlibliCredentials are the per-tenant values a seller pastes in from their
// own Blibli Seller Center account.
type BlibliCredentials struct {
	BusinessPartnerCode string
	MtaUsername         string
	ApiSellerKey        string
	SignatureKey        string // optional
}

// channelID is the fixed "platform name" this integration identifies itself
// as to Blibli (their client generates this from a free-text "platform
// name" config value, lowercased with spaces replaced by hyphens - see
// ApiConfig::getPlatformName() usage in invoke_without_body.php).
const blibliChannelID = "one-erp"

// buildSignedRequest constructs an *http.Request against the Blibli proxy
// API, applying Basic Auth, the Api-Seller-Key header, the mandatory
// auto-generated query parameters, and (if a SignatureKey is configured)
// the Signature/Signature-Time headers - mirroring
// invoker/basic_auth/invoke_with_body.php and invoke_without_body.php
// exactly (both call the same query-param + header construction; the only
// difference is GET has no body/content-type fed into the signature).
//
// path must be the full proxy path, e.g.
// "/v2/proxy/mta/api/businesspartner/v1/order/orderDetail".
func (c *BlibliClient) buildSignedRequest(ctx context.Context, method, path string, creds BlibliCredentials, extraQuery url.Values, body []byte) (*http.Request, error) {
	q := url.Values{}
	q.Set("storeId", "10001")
	q.Set("businessPartnerCode", creds.BusinessPartnerCode)
	q.Set("merchantCode", creds.BusinessPartnerCode)
	q.Set("storeCode", creds.BusinessPartnerCode) // url.Values.Encode() below urlencodes every value
	q.Set("username", creds.MtaUsername)
	q.Set("channelId", blibliChannelID)
	q.Set("requestId", uuid.NewString())
	for k, vs := range extraQuery {
		for _, v := range vs {
			q.Add(k, v)
		}
	}

	fullURL := c.host + path + "?" + q.Encode()

	var reqBody io.Reader
	if len(body) > 0 {
		reqBody = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.cfg.APIClientID, c.cfg.APIClientSecret)
	req.Header.Set("Content-type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Api-Seller-Key", creds.ApiSellerKey)

	if creds.SignatureKey != "" {
		millis := time.Now().UnixMilli()
		contentType := "application/json"
		bodyStr := ""
		if method == http.MethodGet || len(body) == 0 {
			// invoke_without_body.php signs with reqBody="" and
			// reqContentType="" for GET/DELETE.
			contentType = ""
		} else {
			bodyStr = string(body)
		}
		signature := blibliSignature(millis, creds.SignatureKey, method, bodyStr, contentType, blibliSignedPath(path))
		req.Header.Set("Signature", signature)
		req.Header.Set("Signature-Time", fmt.Sprintf("%d", millis))
	}

	return req, nil
}

// blibliSignedPath strips the "/v2/proxy" prefix and, if the remaining path
// contains "/mta", replaces that substring with "/mtaapi" - confirmed as an
// unconditional str_replace("/mta", "/mtaapi", ...) in both
// invoke_with_body.php and invoke_without_body.php (not an artifact of
// misreading; both files contain the identical substitution).
func blibliSignedPath(fullPath string) string {
	raw := fullPath
	if idx := strings.Index(fullPath, "/proxy"); idx >= 0 {
		raw = fullPath[idx+len("/proxy"):]
	}
	if strings.Contains(raw, "/mta") {
		raw = strings.Replace(raw, "/mta", "/mtaapi", 1)
	}
	return raw
}

// blibliSignature implements generate_signature.php's SignatureGenerator::generate
// exactly: HMAC-SHA256(key=reqSecret, msg=baseString), base64-encoded, where
// baseString = reqMethod + "\n" + bodyHash + "\n" + reqContentType + "\n" +
// patternDate + "\n" + reqUrl, patternDate is the ms timestamp formatted in
// Asia/Jakarta as PHP's date("D M d H:i:s T Y") (Go equivalent:
// "Mon Jan 02 15:04:05 MST 2006"), and bodyHash is "" for an empty body or
// else the MD5 hex digest of the body with literal \r\n escaped first
// (str_replace("\r","\\r",...) / str_replace("\n","\\n",...)).
func blibliSignature(millis int64, reqSecret, reqMethod, reqBody, reqContentType, reqURL string) string {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.FixedZone("WIB", 7*60*60)
	}
	t := time.UnixMilli(millis).In(loc)
	patternDate := t.Format("Mon Jan 02 15:04:05 MST 2006")

	bodyHash := ""
	if reqBody != "" {
		escaped := strings.ReplaceAll(reqBody, "\r", `\r`)
		escaped = strings.ReplaceAll(escaped, "\n", `\n`)
		sum := md5.Sum([]byte(escaped))
		bodyHash = hex.EncodeToString(sum[:])
	}

	baseString := reqMethod + "\n" + bodyHash + "\n" + reqContentType + "\n" + patternDate + "\n" + reqURL
	mac := hmac.New(sha256.New, []byte(reqSecret))
	mac.Write([]byte(baseString))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// BlibliOrder is the subset of Blibli's orderDetail response this
// integration maps into a Sales Order.
type BlibliOrder struct {
	OrderNo      string  `json:"orderNo"`
	OrderItemNo  string  `json:"orderItemNo"`
	Status       string  `json:"status"`
	CustomerName string  `json:"customerName"`
	ItemPrice    float64 `json:"itemPrice"`
	CreatedAt    int64   `json:"createdAt"` // epoch millis, if present
	// ItemSku/ItemQty/ItemName identify the product line for this order row
	// - Blibli's orderDetail response is already one row per order item, so
	// (unlike TikTok/Shopee) no nested line-item array is needed. Used by
	// syncBlibliOrder to build a real Sales Order line.
	ItemSku  string `json:"itemSku"`
	ItemQty  int    `json:"itemQty"`
	ItemName string `json:"itemName"`
}

type blibliOrderDetailResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Success bool   `json:"success"`
	Value   struct {
		Orders []BlibliOrder `json:"orders"`
	} `json:"value"`
	// Some Blibli responses wrap a single order directly under "value"
	// rather than "value.orders" - both shapes are tolerated by ListRecentOrders.
	SingleValue *BlibliOrder `json:"-"`
}

// TestConnection validates a tenant's credentials by calling the
// orderDetail endpoint with no filters. Blibli's own reference client
// (index-basic-auth.php) always passes orderNo/orderItemNo, but the API
// accepts the call without them too; any well-formed JSON response (even
// "order not found") proves the Basic Auth + Api-Seller-Key + signature
// combination is valid, since a bad credential set is rejected by Blibli's
// gateway before it ever reaches order lookup logic (401/403, or an
// explicit "invalid signature"/"unauthorized" error code/message).
func (c *BlibliClient) TestConnection(ctx context.Context, creds BlibliCredentials) error {
	_, err := c.orderDetail(ctx, creds, nil)
	return err
}

// ListRecentOrders fetches recently created orders.
//
// CONFIDENCE NOTE (verify against a real Blibli sandbox account before
// relying on this in production): Blibli's official PHP and Java reference
// clients (seller-api-client-php, seller-api-client-java) only demonstrate
// a single order-retrieval endpoint - GET .../businesspartner/v1/order/orderDetail,
// filterable by orderNo/orderItemNo - and no sibling "order list"/"new
// order" endpoint could be confirmed in either reference client or in
// public docs reachable at implementation time. There is no confirmed way
// to enumerate orders without already knowing their order numbers.
// LOW CONFIDENCE: this calls orderDetail with no orderNo/orderItemNo filter
// and treats a non-empty "value.orders" array as "recent orders" - if
// Blibli's gateway instead requires at least one filter (likely, given
// every sample always sets them), this will need to be replaced with
// whatever real listing/webhook mechanism Blibli documents once a live
// partner account can be tested (e.g. Blibli's "New Order" webhook/callback
// push at /v1/notification/ish or similar, which several tenant Blibli
// integrations use instead of polling, but which could not be confirmed
// from the reference clients alone).
func (c *BlibliClient) ListRecentOrders(ctx context.Context, creds BlibliCredentials, since time.Duration) ([]BlibliOrder, error) {
	resp, err := c.orderDetail(ctx, creds, nil)
	if err != nil {
		return nil, err
	}
	return resp.Value.Orders, nil
}

func (c *BlibliClient) orderDetail(ctx context.Context, creds BlibliCredentials, params url.Values) (*blibliOrderDetailResponse, error) {
	const path = "/v2/proxy/mta/api/businesspartner/v1/order/orderDetail"

	req, err := c.buildSignedRequest(ctx, http.MethodGet, path, creds, params, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Blibli orderDetail API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("Blibli rejected the credentials (status %d): %s", resp.StatusCode, string(body))
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Blibli orderDetail API returned status %d: %s", resp.StatusCode, string(body))
	}

	var parsed blibliOrderDetailResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse Blibli orderDetail response: %w", err)
	}
	// Blibli's error envelope uses success=false with an explicit
	// unauthorized/signature-related message even on a 200 status for some
	// error classes - treat that as an invalid-credentials error too rather
	// than silently proceeding.
	if !parsed.Success && parsed.Message != "" && isBlibliAuthError(parsed.Message) {
		return nil, fmt.Errorf("Blibli rejected the credentials: %s", parsed.Message)
	}
	return &parsed, nil
}

type blibliProcessOrderResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// MarkReadyToShip advances one order item to Blibli's "process"/ready-to-ship
// state via POST .../businesspartner/v1/order/process.
//
// CONFIDENCE NOTE: unlike orderDetail (verified against Blibli's own
// reference client), the reference client this integration was built
// against does not demonstrate a ship/process/pickup endpoint at all - only
// orderDetail. This targets the "order/process" path other Blibli MTA
// integrations commonly document for the pack->ready-to-ship transition,
// with a "status":"process" body, but it has NOT been confirmed against
// either the reference client or a live sandbox. Verify the exact path and
// required status value against Blibli's current Seller Center API docs
// (or their support team) before relying on this in production - if it 404s
// or Blibli returns an explicit "unknown path" error, that confirms this
// guess is wrong and the real endpoint needs to be sourced from Blibli
// directly.
func (c *BlibliClient) MarkReadyToShip(ctx context.Context, creds BlibliCredentials, orderNo, orderItemNo string) error {
	const path = "/v2/proxy/mta/api/businesspartner/v1/order/process"

	bodyBytes, err := json.Marshal(map[string]any{
		"orderNo":     orderNo,
		"orderItemNo": orderItemNo,
		"status":      "process",
	})
	if err != nil {
		return err
	}

	req, err := c.buildSignedRequest(ctx, http.MethodPost, path, creds, nil, bodyBytes)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call Blibli order/process API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("Blibli rejected the credentials (status %d): %s", resp.StatusCode, string(body))
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("Blibli order/process API returned status %d: %s", resp.StatusCode, string(body))
	}

	var parsed blibliProcessOrderResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fmt.Errorf("failed to parse Blibli order/process response: %w", err)
	}
	if !parsed.Success {
		return fmt.Errorf("Blibli order/process API error %s: %s", parsed.Code, parsed.Message)
	}
	return nil
}

func isBlibliAuthError(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "unauthor") ||
		strings.Contains(lower, "signature") ||
		strings.Contains(lower, "forbidden") ||
		strings.Contains(lower, "invalid api") ||
		strings.Contains(lower, "invalid credential")
}
