package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

type Config struct {
	App        AppConfig
	Database   DatabaseConfig
	JWT        JWTConfig
	R2         R2Config
	Storage    StorageConfig
	OpenAI     OpenAIConfig
	WhatsApp   WhatsAppConfig
	Email      EmailConfig
	Xendit     XenditConfig
	Frontend   FrontendConfig
	Platform   PlatformConfig
	TikTokShop TikTokShopConfig
	Shopee     ShopeeConfig
	Blibli     BlibliConfig
	Lazada     LazadaConfig
	Internal   InternalConfig
}

// InternalConfig guards service-to-service endpoints (see
// internal/modules/provisioning) called by the standalone abc-backend
// service to provision companies/tenants here. No envDefault, same
// fail-closed philosophy as JWT.Secret: an unset token refuses the route
// entirely rather than accepting unauthenticated calls.
type InternalConfig struct {
	ProvisioningToken string `env:"INTERNAL_PROVISIONING_TOKEN"`
}

type AppConfig struct {
	Name string `env:"APP_NAME" envDefault:"one-backend"`
	Env  string `env:"APP_ENV" envDefault:"development"`
	Port int    `env:"APP_PORT" envDefault:"3000"`
	// SeedDemoData makes new tenants (and the shared chat/drive/docflow data)
	// start with sample employees, customers, projects, orders and so on. It is
	// off by default: a real company must start empty. Reference data the system
	// needs to work, such as the chart of accounts, is always seeded.
	SeedDemoData bool `env:"SEED_DEMO_DATA" envDefault:"false"`
	// AutoOnboardEmployees creates an HR employee record for a company member the
	// first time they use the company (matched by email), so leave, attendance and
	// reimbursement work without manual setup. On by default.
	AutoOnboardEmployees bool `env:"AUTO_ONBOARD_EMPLOYEES" envDefault:"true"`
}

func (a *AppConfig) IsDevelopment() bool {
	return a.Env == "development" || a.Env == "local" || a.Env == ""
}

type DatabaseConfig struct {
	Host                   string `env:"DB_HOST" envDefault:"localhost"`
	Port                   int    `env:"DB_PORT" envDefault:"5432"`
	User                   string `env:"DB_USER" envDefault:"postgres"`
	Password               string `env:"DB_PASSWORD" envDefault:""`
	DBName                 string `env:"DB_NAME" envDefault:"one_erp"`
	SSLMode                string `env:"DB_SSLMODE" envDefault:"disable"`
	MaxOpenConns           int    `env:"DB_MAX_OPEN_CONNS" envDefault:"25"`
	MaxIdleConns           int    `env:"DB_MAX_IDLE_CONNS" envDefault:"10"`
	ConnMaxLifetimeMinutes int    `env:"DB_CONN_MAX_LIFETIME_MINUTES" envDefault:"30"`
}

func (d *DatabaseConfig) DSN() string {
	return d.DSNForDatabase(d.DBName)
}

// DSNForDatabase builds a DSN for an arbitrary database name on the same
// Postgres host/credentials as the central database. Used to connect to
// per-tenant databases and to the "postgres" maintenance database.
func (d *DatabaseConfig) DSNForDatabase(dbName string) string {
	if d.Password == "" {
		return fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s",
			d.Host, d.Port, d.User, dbName, d.SSLMode)
	}
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, dbName, d.SSLMode)
}

type JWTConfig struct {
	// Secret has no envDefault on purpose: a committed default would let
	// every deployment that forgets to set JWT_SECRET silently sign and
	// accept tokens with a well-known key. Load() enforces that this is set
	// (falling back to a fixed dev-only value in APP_ENV=development only).
	Secret    string        `env:"JWT_SECRET"`
	ExpiresIn time.Duration `env:"JWT_EXPIRES_IN" envDefault:"24h"`
}

// StorageConfig configures the fallback used when R2 is not set up: files are
// kept on local disk under LocalDir (suitable for development and single-node
// installs; use R2 for anything that must survive the machine).
type StorageConfig struct {
	LocalDir string `env:"LOCAL_STORAGE_DIR"`
}

// R2Config holds Cloudflare R2 (S3-compatible) object storage credentials.
type R2Config struct {
	AccountID       string `env:"R2_ACCOUNT_ID"`
	AccessKeyID     string `env:"R2_ACCESS_KEY_ID"`
	SecretAccessKey string `env:"R2_SECRET_ACCESS_KEY"`
	BucketName      string `env:"R2_BUCKET_NAME"`
	PublicBaseURL   string `env:"R2_PUBLIC_BASE_URL"`
}

func (r *R2Config) Enabled() bool {
	return r.AccountID != "" && r.AccessKeyID != "" && r.SecretAccessKey != "" && r.BucketName != ""
}

func (r *R2Config) Endpoint() string {
	return fmt.Sprintf("https://%s.r2.cloudflarestorage.com", r.AccountID)
}

// OpenAIConfig holds OpenAI API credentials for transcription and AI summaries.
type OpenAIConfig struct {
	APIKey          string `env:"OPENAI_API_KEY"`
	ChatModel       string `env:"OPENAI_CHAT_MODEL" envDefault:"gpt-4o-mini"`
	TranscribeModel string `env:"OPENAI_TRANSCRIBE_MODEL" envDefault:"gpt-4o-transcribe"`
}

func (o *OpenAIConfig) Enabled() bool {
	return o.APIKey != ""
}

// WhatsAppConfig holds Meta WhatsApp Cloud API credentials for sending
// onboarding invite links to new clients.
type WhatsAppConfig struct {
	PhoneNumberID string `env:"WHATSAPP_PHONE_NUMBER_ID"`
	AccessToken   string `env:"WHATSAPP_ACCESS_TOKEN"`
	APIVersion    string `env:"WHATSAPP_API_VERSION" envDefault:"v20.0"`

	// WebhookVerifyToken is compared against Meta's `hub.verify_token` query
	// param during the webhook verification handshake
	// (GET /omnichannel/webhooks/whatsapp). Invent any random string and set
	// the same value in the Meta App Dashboard's webhook config.
	WebhookVerifyToken string `env:"WHATSAPP_WEBHOOK_VERIFY_TOKEN"`

	// DefaultCompanyIDRaw is the company whose tenant database inbound
	// WhatsApp webhook messages are written into, as a raw string (parsed
	// via DefaultCompanyID() to avoid depending on the env library's
	// TextUnmarshaler support for uuid.UUID). Phase 1 of the Omnichannel
	// Inbox feature assumes a single platform-level WhatsApp Business
	// number, so there is no per-message way to know which company it
	// belongs to - see internal/modules/omnichannel for the full
	// explanation and the follow-up needed for true multi-tenant WhatsApp.
	DefaultCompanyIDRaw string `env:"WHATSAPP_DEFAULT_COMPANY_ID"`

	// AppSecret is Meta's App Secret, used to verify the
	// X-Hub-Signature-256 header on inbound webhook deliveries. Not yet
	// wired up to any verification logic - see
	// internal/modules/omnichannel/delivery/http/handler.go's WhatsAppWebhook
	// for the documented gap.
	AppSecret string `env:"WHATSAPP_APP_SECRET"`
}

func (w *WhatsAppConfig) Enabled() bool {
	return w.PhoneNumberID != "" && w.AccessToken != ""
}

// DefaultCompanyID parses DefaultCompanyIDRaw into a *uuid.UUID, or returns
// nil if unset/invalid.
func (w *WhatsAppConfig) DefaultCompanyID() *uuid.UUID {
	if w.DefaultCompanyIDRaw == "" {
		return nil
	}
	id, err := uuid.Parse(w.DefaultCompanyIDRaw)
	if err != nil {
		return nil
	}
	return &id
}

// EmailConfig holds SMTP credentials for sending onboarding invite emails.
type EmailConfig struct {
	SMTPHost  string `env:"SMTP_HOST"`
	SMTPPort  int    `env:"SMTP_PORT" envDefault:"587"`
	SMTPUser  string `env:"SMTP_USER"`
	SMTPPass  string `env:"SMTP_PASSWORD"`
	FromEmail string `env:"SMTP_FROM_EMAIL"`
	FromName  string `env:"SMTP_FROM_NAME" envDefault:"ONE ERP"`
}

func (e *EmailConfig) Enabled() bool {
	return e.SMTPHost != "" && e.SMTPUser != "" && e.SMTPPass != "" && e.FromEmail != ""
}

// XenditConfig holds Xendit payment gateway credentials for QRIS/Virtual
// Account subscription payments.
type XenditConfig struct {
	SecretKey       string `env:"XENDIT_SECRET_KEY"`
	WebhookToken    string `env:"XENDIT_WEBHOOK_TOKEN"`
	CallbackBaseURL string `env:"XENDIT_CALLBACK_BASE_URL"`
}

func (x *XenditConfig) Enabled() bool {
	return x.SecretKey != ""
}

// FrontendConfig holds the base URL of the frontend app, used to build
// onboarding invite links sent to clients.
type FrontendConfig struct {
	BaseURL string `env:"FRONTEND_BASE_URL" envDefault:"http://localhost:3001"`
}

// PlatformConfig holds the current list price for NEW subscriptions. Per
// business policy, a price change here only ever affects companies created
// after the change - each Subscription locks in whatever this value was at
// its own creation time (stored on Subscription.PricePerMonth) and keeps
// that rate for life, including renewals. Changing this env var and
// restarting the backend is the only way to raise/lower the list price;
// there is no admin UI for it yet.
type PlatformConfig struct {
	CurrentPricePerMonth float64 `env:"PLATFORM_PRICE_PER_MONTH" envDefault:"1000000"`
}

// TikTokShopConfig holds TikTok Shop Partner Center app credentials used to
// authorize a seller's shop (OAuth) and to sign Order API requests.
// Register the redirect URI (AppBaseURL + "/api/v1/marketplace/tiktok/callback")
// in the Partner Center app settings.
type TikTokShopConfig struct {
	AppKey      string `env:"TIKTOK_SHOP_APP_KEY"`
	AppSecret   string `env:"TIKTOK_SHOP_APP_SECRET"`
	RedirectURI string `env:"TIKTOK_SHOP_REDIRECT_URI"`
}

func (t *TikTokShopConfig) Enabled() bool {
	return t.AppKey != "" && t.AppSecret != ""
}

// ShopeeConfig holds Shopee Open Platform partner credentials used to
// authorize a seller's shop (OAuth) and to sign API requests.
// Register the redirect URI (AppBaseURL + "/api/v1/marketplace/shopee/callback")
// in the Shopee Open Platform app settings.
type ShopeeConfig struct {
	PartnerID   int64  `env:"SHOPEE_PARTNER_ID"`
	PartnerKey  string `env:"SHOPEE_PARTNER_KEY"`
	RedirectURI string `env:"SHOPEE_REDIRECT_URI"`
	// Sandbox selects Shopee's UAT host (partner.test-stable.shopeemobile.com)
	// for the Test Partner ID/Key issued while the app is still in
	// "Developing" status in Shopee Open Platform. Flip to false once the
	// app is approved and live Partner ID/Key are issued.
	Sandbox bool `env:"SHOPEE_SANDBOX" envDefault:"true"`
}

func (s *ShopeeConfig) Enabled() bool {
	return s.PartnerID != 0 && s.PartnerKey != ""
}

// BlibliConfig holds the ISV-level (ONE ERP's own) Blibli Partner API
// credentials, issued once by Blibli when registering as a partner. Unlike
// TikTok Shop/Shopee, Blibli has no per-seller OAuth redirect flow: these
// ApiClientId/ApiClientKey values authenticate every request via HTTP Basic
// Auth, while each tenant additionally supplies their own Business Partner
// Code / MTA Username / API Seller Key / Signature Key (typed into a form in
// Settings > Integrations, stored on MarketplaceConnection) that identify
// their own Blibli Seller Center account.
type BlibliConfig struct {
	APIClientID     string `env:"BLIBLI_API_CLIENT_ID"`
	APIClientSecret string `env:"BLIBLI_API_CLIENT_SECRET"`
	// Sandbox selects Blibli's UAT host (api-uata.gdn-app.com) instead of the
	// production host (api.gdn-app.com).
	Sandbox bool `env:"BLIBLI_SANDBOX" envDefault:"true"`
}

func (b *BlibliConfig) Enabled() bool {
	return b.APIClientID != "" && b.APIClientSecret != ""
}

// LazadaConfig holds Lazada Open Platform (open.lazada.com) app credentials
// used to authorize a seller's shop (OAuth) and to sign Order API requests.
// Register the redirect URI (AppBaseURL + "/api/v1/marketplace/lazada/callback")
// in the app's Callback URL setting on open.lazada.com.
type LazadaConfig struct {
	AppKey      string `env:"LAZADA_APP_KEY"`
	AppSecret   string `env:"LAZADA_APP_SECRET"`
	RedirectURI string `env:"LAZADA_REDIRECT_URI"`
}

func (l *LazadaConfig) Enabled() bool {
	return l.AppKey != "" && l.AppSecret != ""
}

// Load loads environment variables and parses into Config struct
func Load() (*Config, error) {
	// Search for .env from current directory up to root containing go.mod
	if root := findProjectRoot(); root != "" {
		_ = godotenv.Load(filepath.Join(root, ".env"))
	} else {
		_ = godotenv.Load()
	}

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}

	if cfg.JWT.Secret == "" {
		if cfg.App.IsDevelopment() {
			cfg.JWT.Secret = "dev-only-insecure-jwt-secret"
		} else {
			return nil, fmt.Errorf("JWT_SECRET must be set (APP_ENV=%q is not development)", cfg.App.Env)
		}
	}

	return cfg, nil
}

func findProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
