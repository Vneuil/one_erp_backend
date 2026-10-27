package aiassistant

import (
	"errors"
	"strings"

	"github.com/divinecoid/one-backend/internal/foundation/crypto"
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Handler struct{}

type settingsView struct {
	Provider  string         `json:"provider"`
	Model     string         `json:"model"`
	HasAPIKey bool           `json:"hasApiKey"`
	KeyHint   string         `json:"keyHint"`
	Providers []ProviderInfo `json:"providers"`
}

func (h *Handler) db(c *fiber.Ctx) (*gorm.DB, error) {
	db := middleware.TenantDB(c)
	if db == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
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
	system := "Anda adalah Asisten AI untuk sistem ERP perusahaan. Jawab dalam bahasa pengguna, ringkas dan jelas. " +
		"Gunakan hanya data ERP yang diberikan pada bagian KONTEKS; jika data tidak ada, katakan tidak tahu dan jangan mengarang angka."
	if ctxText := strings.TrimSpace(in.Context); ctxText != "" {
		system += "\n\nKONTEKS DATA ERP:\n" + truncate(ctxText, 8000)
	}
	reply, err := Complete(c.UserContext(), s.Provider, s.Model, key, system, in.Messages)
	if err != nil {
		return apperrors.NewBadRequest(err.Error())
	}
	return response.OK(c, "OK", fiber.Map{"reply": reply})
}
