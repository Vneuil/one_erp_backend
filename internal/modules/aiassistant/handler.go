package aiassistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/divinecoid/one-backend/internal/foundation/crypto"
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/report/application"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Handler struct {
	manager *tenantMgr.Manager
	central *gorm.DB
}

type settingsView struct {
	Provider  string         `json:"provider"`
	Model     string         `json:"model"`
	HasAPIKey bool           `json:"hasApiKey"`
	KeyHint   string         `json:"keyHint"`
	Providers []ProviderInfo `json:"providers"`
}

// migrated remembers tenant DBs whose tables were verified, since existing
// companies are only backfilled when an admin runs the schema backfill.
var migrated sync.Map

func ensureSchema(db *gorm.DB) error {
	if _, ok := migrated.Load(db); ok {
		return nil
	}
	if err := db.AutoMigrate(&Settings{}, &MCPToken{}); err != nil {
		return err
	}
	migrated.Store(db, true)
	return nil
}

func (h *Handler) db(c *fiber.Ctx) (*gorm.DB, error) {
	db := middleware.TenantDB(c)
	if db == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	if err := ensureSchema(db); err != nil {
		return nil, apperrors.NewInternal(err)
	}
	return db, nil
}

func load(db *gorm.DB) (*Settings, error) {
	var s Settings
	err := db.First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &s, err
}

func view(s *Settings) settingsView {
	v := settingsView{Providers: Providers}
	if s == nil {
		return v
	}
	v.Provider, v.Model = s.Provider, s.Model
	if key, err := crypto.Decrypt(s.APIKey); err == nil && key != "" {
		v.HasAPIKey = true
		if len(key) > 4 {
			v.KeyHint = "••••" + key[len(key)-4:]
		} else {
			v.KeyHint = "••••"
		}
	}
	return v
}

func (h *Handler) Get(c *fiber.Ctx) error {
	db, err := h.db(c)
	if err != nil {
		return err
	}
	s, err := load(db)
	if err != nil {
		return err
	}
	return response.OK(c, "AI settings retrieved successfully", view(s))
}

func (h *Handler) Save(c *fiber.Ctx) error {
	db, err := h.db(c)
	if err != nil {
		return err
	}
	var in struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
		APIKey   string `json:"apiKey"`
	}
	if err := c.BodyParser(&in); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	p, ok := providerByID(in.Provider)
	if !ok {
		return apperrors.NewBadRequest("Penyedia AI tidak valid")
	}
	model := strings.TrimSpace(in.Model)
	if model == "" {
		model = p.DefaultModel
	}
	s, err := load(db)
	if err != nil {
		return err
	}
	if s == nil {
		s = &Settings{}
	}
	s.Provider, s.Model = p.ID, model
	if key := strings.TrimSpace(in.APIKey); key != "" {
		enc, err := crypto.Encrypt(key)
		if err != nil {
			return apperrors.NewInternal(err)
		}
		s.APIKey = enc
	}
	if err := db.Save(s).Error; err != nil {
		return err
	}
	return response.OK(c, "AI settings saved successfully", view(s))
}

func (h *Handler) Chat(c *fiber.Ctx) error {
	db, err := h.db(c)
	if err != nil {
		return err
	}
	var in struct {
		Messages []ChatMessage `json:"messages"`
		Context  string        `json:"context"`
	}
	if err := c.BodyParser(&in); err != nil || len(in.Messages) == 0 {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if len(in.Messages) > 30 {
		in.Messages = in.Messages[len(in.Messages)-30:]
	}
	s, err := load(db)
	if err != nil {
		return err
	}
	if s == nil || s.APIKey == "" {
		return apperrors.NewBadRequest("Asisten AI belum dikonfigurasi. Admin dapat mengaturnya di Pengaturan > Setting AI.")
	}
	key, err := crypto.Decrypt(s.APIKey)
	if err != nil {
		return apperrors.NewInternal(err)
	}
	system := "Anda adalah Asisten AI untuk sistem ERP perusahaan (Manna ERP). Jawab dalam bahasa pengguna, ringkas dan jelas. " +
		"Untuk pertanyaan tentang data perusahaan (stok, penjualan, piutang, hutang, pembelian, ringkasan eksekutif), WAJIB panggil tool yang tersedia dan jawab hanya dari hasilnya; jangan mengarang angka. " +
		"Jika pengguna bertanya tool apa yang tersedia, sebutkan nama dan fungsi tool tersebut. Tanggal hari ini: " + today() + " (WIB). " +
		"Format jawaban: gunakan **tebal** untuk angka penting dan daftar berawalan \"- \" bila perlu; jangan memakai tabel atau heading."
	if ctxText := strings.TrimSpace(in.Context); ctxText != "" {
		system += "\n\nKONTEKS TAMBAHAN:\n" + truncate(ctxText, 8000)
	}
	exec := func(ctx context.Context, name string, args map[string]any) (string, error) {
		for _, t := range tools {
			if t.Name == name {
				out, err := t.run(ctx, application.NewReportQuery(db), args)
				if err != nil {
					return "", err
				}
				body, err := json.Marshal(out)
				return string(body), err
			}
		}
		return "", fmt.Errorf("tool tidak dikenal: %s", name)
	}
	reply, err := Complete(c.UserContext(), s.Provider, s.Model, key, system, in.Messages, tools, exec)
	if err != nil {
		return apperrors.NewBadRequest(err.Error())
	}
	return response.OK(c, "OK", fiber.Map{"reply": reply})
}
