package aiassistant

import "github.com/divinecoid/one-backend/internal/shared/types"

// Settings is the company's chosen LLM provider. One row per company DB.
// APIKey is stored encrypted via foundation/crypto and never returned to clients.
type Settings struct {
	types.BaseEntity
	Provider string `gorm:"type:varchar(20);not null" json:"provider"` // openai | gemini | deepseek | claude
	Model    string `gorm:"type:varchar(100);not null" json:"model"`
	APIKey   string `gorm:"type:text" json:"-"`
}

func (Settings) TableName() string { return "ai_settings" }

type ProviderInfo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	DefaultModel string `json:"defaultModel"`
}

var Providers = []ProviderInfo{
	{ID: "openai", Name: "OpenAI (ChatGPT)", DefaultModel: "gpt-6-luna"},
	{ID: "gemini", Name: "Google Gemini", DefaultModel: "gemini-3.8-flash"},
	{ID: "deepseek", Name: "DeepSeek", DefaultModel: "deepseek-flash"},
	{ID: "claude", Name: "Anthropic Claude", DefaultModel: "claude-sonnet-5-5"},
}

func providerByID(id string) (ProviderInfo, bool) {
	for _, p := range Providers {
		if p.ID == id {
			return p, true
		}
	}
	return ProviderInfo{}, false
}
