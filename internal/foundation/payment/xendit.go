// Package payment abstracts creating a payable invoice (QRIS / Virtual
// Account) via Xendit. Degrades to a noop that returns an explicit error
// when XENDIT_SECRET_KEY is not configured - never a fake success.
package payment

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/config"
)

type Invoice struct {
	ProviderRef string // Xendit invoice ID
	CheckoutURL string // hosted payment page (QRIS + VA + e-wallets)
	Status      string
}

type Client interface {
	CreateInvoice(ctx context.Context, externalID string, amount float64, description, payerEmail string) (*Invoice, error)
	Enabled() bool
}

func NewClient(cfg config.XenditConfig) Client {
	if !cfg.Enabled() {
		return &noopClient{}
	}
	return &xenditClient{cfg: cfg, httpClient: &http.Client{Timeout: 20 * time.Second}}
}

type xenditClient struct {
	cfg        config.XenditConfig
	httpClient *http.Client
}

func (c *xenditClient) Enabled() bool { return true }

func (c *xenditClient) CreateInvoice(ctx context.Context, externalID string, amount float64, description, payerEmail string) (*Invoice, error) {
	payload := map[string]any{
		"external_id":     externalID,
		"amount":          amount,
		"description":     description,
		"payer_email":     payerEmail,
		"currency":        "IDR",
		"payment_methods": []string{"QRIS", "BCA", "BNI", "BRI", "MANDIRI", "PERMATA"},
	}
	if c.cfg.CallbackBaseURL != "" {
		payload["success_redirect_url"] = c.cfg.CallbackBaseURL + "/checkout/success?externalId=" + externalID
		payload["failure_redirect_url"] = c.cfg.CallbackBaseURL + "/checkout/failed?externalId=" + externalID
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal xendit invoice request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.xendit.co/v2/invoices", bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to build xendit invoice request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.cfg.SecretKey+":")))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Xendit API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Xendit response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Xendit API returned status %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		ID         string `json:"id"`
		InvoiceURL string `json:"invoice_url"`
		Status     string `json:"status"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse Xendit response: %w", err)
	}

	return &Invoice{ProviderRef: parsed.ID, CheckoutURL: parsed.InvoiceURL, Status: parsed.Status}, nil
}

type noopClient struct{}

func (c *noopClient) Enabled() bool { return false }

func (c *noopClient) CreateInvoice(ctx context.Context, externalID string, amount float64, description, payerEmail string) (*Invoice, error) {
	return nil, fmt.Errorf("payment not configured: XENDIT_SECRET_KEY is not set")
}
