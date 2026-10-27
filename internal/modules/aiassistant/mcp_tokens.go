package aiassistant

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const tokenPrefix = "mcp_"

func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

// Token format: mcp_<companyUUID>.<random>. The company ID lets the MCP
// endpoint find the right tenant database before it can look the hash up.
func newTokenSecret(companyID uuid.UUID) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return tokenPrefix + companyID.String() + "." + base64.RawURLEncoding.EncodeToString(b), nil
}

func (h *Handler) ListTokens(c *fiber.Ctx) error {
	db, err := h.db(c)
	if err != nil {
		return err
	}
	var items []MCPToken
	if err := db.Order("created_at desc").Find(&items).Error; err != nil {
		return err
	}
	return response.OK(c, "MCP tokens retrieved successfully", items)
}

func (h *Handler) CreateToken(c *fiber.Ctx) error {
	db, err := h.db(c)
	if err != nil {
		return err
	}
	claims := middleware.CurrentUser(c)
	if claims == nil || claims.CompanyID == nil {
		return apperrors.NewBadRequest("No active company.")
	}
	var in struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&in); err != nil || strings.TrimSpace(in.Name) == "" {
		return apperrors.NewBadRequest("Nama token wajib diisi")
	}
	secret, err := newTokenSecret(*claims.CompanyID)
	if err != nil {
		return apperrors.NewInternal(err)
	}
	t := MCPToken{
		Name:      strings.TrimSpace(in.Name),
		TokenHash: hashToken(secret),
		Hint:      secret[len(secret)-4:],
		CreatedBy: claims.UserID,
	}
	if err := db.Create(&t).Error; err != nil {
		return err
	}
	return response.Created(c, "MCP token created. Copy it now; it will not be shown again.", fiber.Map{
		"token": t, "secret": secret,
	})
}

func (h *Handler) RevokeToken(c *fiber.Ctx) error {
	db, err := h.db(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid id")
	}
	res := db.Delete(&MCPToken{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperrors.NewNotFound("Token tidak ditemukan")
	}
	return response.OK(c, "MCP token revoked", nil)
}

// authenticate resolves a bearer MCP token to the company's tenant DB.
func (h *Handler) authenticate(c *fiber.Ctx) (*MCPToken, error) {
	raw := strings.TrimSpace(strings.TrimPrefix(c.Get("Authorization"), "Bearer "))
	if !strings.HasPrefix(raw, tokenPrefix) {
		return nil, apperrors.NewUnauthorized("MCP token required")
	}
	idPart, _, ok := strings.Cut(strings.TrimPrefix(raw, tokenPrefix), ".")
	companyID, err := uuid.Parse(idPart)
	if !ok || err != nil {
		return nil, apperrors.NewUnauthorized("Invalid MCP token")
	}
	if h.central != nil {
		if active, err := licenseActive(c, h.central, companyID); err == nil && !active {
			return nil, apperrors.NewForbidden("Company license is not active")
		}
	}
	db, err := h.manager.GetDB(c.UserContext(), companyID)
	if err != nil {
		return nil, apperrors.NewUnauthorized("Invalid MCP token")
	}
	if err := ensureSchema(db); err != nil {
		return nil, apperrors.NewInternal(err)
	}
	var t MCPToken
	if err := db.Where("token_hash = ?", hashToken(raw)).First(&t).Error; err != nil {
		return nil, apperrors.NewUnauthorized("Invalid MCP token")
	}
	now := time.Now()
	db.Model(&MCPToken{}).Where("id = ?", t.ID).Update("last_used_at", now)
	c.Locals(mcpDBKey, db)
	return &t, nil
}
