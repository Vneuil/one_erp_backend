package aiassistant

import (
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// MCPToken lets an external AI app call this company's MCP endpoint. Only the
// SHA-256 hash of the secret is stored; the plaintext is shown once at creation.
type MCPToken struct {
	types.BaseEntity
	Name       string     `gorm:"type:varchar(100);not null" json:"name"`
	TokenHash  string     `gorm:"type:varchar(64);not null;uniqueIndex" json:"-"`
	Hint       string     `gorm:"type:varchar(12)" json:"hint"` // last chars of the secret, for recognition
	CreatedBy  uuid.UUID  `gorm:"type:uuid" json:"createdBy"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
}

func (MCPToken) TableName() string { return "ai_mcp_tokens" }
