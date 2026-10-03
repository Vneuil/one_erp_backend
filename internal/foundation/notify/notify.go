// Package notify abstracts sending onboarding invite links to prospective
// clients via email and WhatsApp. Like foundation/storage and foundation/ai,
// each channel degrades to a noop that returns an explicit error when its
// credentials are not configured - never a silent fallback.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/config"
)

// Notifier sends an onboarding invite link over a channel.
type Notifier interface {
	SendEmail(ctx context.Context, toEmail, toName, subject, body string) error
	SendWhatsApp(ctx context.Context, toPhoneE164, message string) error
	// SendWhatsAppWithCredentials sends via an arbitrary phone_number_id +
	// access token pair instead of the platform-level WhatsAppConfig - used
	// by omnichannel's per-tenant ChannelConnection send path (see
	// internal/modules/omnichannel/application/usecase.go SendMessage) so the
	// Graph API call itself is implemented exactly once.
	SendWhatsAppWithCredentials(ctx context.Context, phoneNumberID, accessToken, apiVersion, toPhoneE164, message string) error
	EmailEnabled() bool
	WhatsAppEnabled() bool
}

type notifier struct {
	email      config.EmailConfig
	whatsapp   config.WhatsAppConfig
	httpClient *http.Client
}

func New(emailCfg config.EmailConfig, waCfg config.WhatsAppConfig) Notifier {
	return &notifier{
		email:      emailCfg,
		whatsapp:   waCfg,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (n *notifier) EmailEnabled() bool    { return n.email.Enabled() }
func (n *notifier) WhatsAppEnabled() bool { return n.whatsapp.Enabled() }

func (n *notifier) SendEmail(ctx context.Context, toEmail, toName, subject, body string) error {
	if !n.email.Enabled() {
		return fmt.Errorf("email not configured: SMTP_HOST/SMTP_USER/SMTP_PASSWORD/SMTP_FROM_EMAIL are not set")
	}

	addr := fmt.Sprintf("%s:%d", n.email.SMTPHost, n.email.SMTPPort)
	auth := smtp.PlainAuth("", n.email.SMTPUser, n.email.SMTPPass, n.email.SMTPHost)

	msg := fmt.Sprintf(
		"From: %s <%s>\r\nTo: %s <%s>\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		n.email.FromName, n.email.FromEmail, toName, toEmail, subject, body,
	)

	if err := smtp.SendMail(addr, auth, n.email.FromEmail, []string{toEmail}, []byte(msg)); err != nil {
		return fmt.Errorf("failed to send invite email: %w", err)
	}
	return nil
}

func (n *notifier) SendWhatsApp(ctx context.Context, toPhoneE164, message string) error {
	if !n.whatsapp.Enabled() {
		return fmt.Errorf("whatsapp not configured: WHATSAPP_PHONE_NUMBER_ID/WHATSAPP_ACCESS_TOKEN are not set")
	}
	return n.SendWhatsAppWithCredentials(ctx, n.whatsapp.PhoneNumberID, n.whatsapp.AccessToken, n.whatsapp.APIVersion, toPhoneE164, message)
}

// SendWhatsAppWithCredentials implements the actual WhatsApp Cloud API
// "send message" call (the platform-level SendWhatsApp above and
// omnichannel's per-tenant ChannelConnection path both call into this one
// implementation - see the Notifier interface doc).
func (n *notifier) SendWhatsAppWithCredentials(ctx context.Context, phoneNumberID, accessToken, apiVersion, toPhoneE164, message string) error {
	if phoneNumberID == "" || accessToken == "" {
		return fmt.Errorf("whatsapp not configured: phone number id / access token are not set")
	}
	if apiVersion == "" {
		apiVersion = "v20.0"
	}

	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/messages", apiVersion, phoneNumberID)
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"to":                toPhoneE164,
		"type":              "text",
		"text":              map[string]string{"body": message},
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal whatsapp payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to build whatsapp request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call WhatsApp Cloud API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("WhatsApp Cloud API returned status %d", resp.StatusCode)
	}
	return nil
}
