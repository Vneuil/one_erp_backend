package aiassistant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ChatMessage struct {
	Role    string `json:"role"` // user | assistant
	Content string `json:"content"`
}

var errNoAnswer = fmt.Errorf("penyedia AI tidak mengembalikan jawaban")

func nonEmpty(s string) (string, error) {
	if strings.TrimSpace(s) == "" {
		return "", errNoAnswer
	}
	return s, nil
}

var llmHTTP = &http.Client{Timeout: 60 * time.Second}

func postJSON(ctx context.Context, url string, headers map[string]string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := llmHTTP.Do(req)
	if err != nil {
		return fmt.Errorf("gagal menghubungi penyedia AI: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("penyedia AI mengembalikan status %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}
	return json.Unmarshal(raw, out)
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// Complete sends the conversation to the selected provider and returns the reply text.
func Complete(ctx context.Context, provider, model, apiKey, system string, msgs []ChatMessage) (string, error) {
	switch provider {
	case "openai":
		// OpenAI recommends the Responses API for current models.
		var out struct {
			Output []struct {
				Type    string `json:"type"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"output"`
		}
		if err := postJSON(ctx, "https://api.openai.com/v1/responses", map[string]string{"Authorization": "Bearer " + apiKey},
			map[string]any{"model": model, "instructions": system, "input": msgs}, &out); err != nil {
			return "", err
		}
		var sb strings.Builder
		for _, item := range out.Output {
			if item.Type != "message" {
				continue
			}
			for _, c := range item.Content {
				if c.Type == "output_text" {
					sb.WriteString(c.Text)
				}
			}
		}
		return nonEmpty(sb.String())

	case "deepseek":
		all := append([]ChatMessage{{Role: "system", Content: system}}, msgs...)
		var out struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := postJSON(ctx, "https://api.deepseek.com/chat/completions", map[string]string{"Authorization": "Bearer " + apiKey},
			map[string]any{"model": model, "messages": all, "stream": false}, &out); err != nil {
			return "", err
		}
		if len(out.Choices) == 0 {
			return "", errNoAnswer
		}
		return nonEmpty(out.Choices[0].Message.Content)

	case "claude":
		var out struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := postJSON(ctx, "https://api.anthropic.com/v1/messages",
			map[string]string{"x-api-key": apiKey, "anthropic-version": "2023-06-01"},
			map[string]any{"model": model, "max_tokens": 1024, "system": system, "messages": msgs}, &out); err != nil {
			return "", err
		}
		var sb strings.Builder
		for _, b := range out.Content {
			if b.Type == "text" {
				sb.WriteString(b.Text)
			}
		}
		return nonEmpty(sb.String())

	case "gemini":
		contents := make([]map[string]any, 0, len(msgs))
		for _, m := range msgs {
			role := "user"
			if m.Role == "assistant" {
				role = "model"
			}
			contents = append(contents, map[string]any{"role": role, "parts": []map[string]string{{"text": m.Content}}})
		}
		var out struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		url := "https://generativelanguage.googleapis.com/v1beta/models/" + model + ":generateContent"
		if err := postJSON(ctx, url, map[string]string{"x-goog-api-key": apiKey}, map[string]any{
			"systemInstruction": map[string]any{"parts": []map[string]string{{"text": system}}},
			"contents":          contents,
		}, &out); err != nil {
			return "", err
		}
		if len(out.Candidates) == 0 {
			return "", errNoAnswer
		}
		var sb strings.Builder
		for _, p := range out.Candidates[0].Content.Parts {
			sb.WriteString(p.Text)
		}
		return nonEmpty(sb.String())
	}
	return "", fmt.Errorf("penyedia AI tidak dikenal: %s", provider)
}
