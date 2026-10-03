package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/config"
)

// Client abstracts OpenAI-backed audio transcription and text summarization.
type Client interface {
	Transcribe(ctx context.Context, audioBytes []byte, filename string) (transcript string, err error)
	Summarize(ctx context.Context, prompt string) (summary string, err error)
	Enabled() bool
}

// NewClient returns a real OpenAI-backed Client when cfg is fully configured,
// or a no-op fallback otherwise.
func NewClient(cfg config.OpenAIConfig) Client {
	if !cfg.Enabled() {
		return &noopClient{}
	}
	return &openAIClient{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

type openAIClient struct {
	cfg        config.OpenAIConfig
	httpClient *http.Client
}

func (c *openAIClient) Enabled() bool {
	return true
}

func (c *openAIClient) Transcribe(ctx context.Context, audioBytes []byte, filename string) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("model", c.cfg.TranscribeModel); err != nil {
		return "", fmt.Errorf("failed to write model field: %w", err)
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(audioBytes); err != nil {
		return "", fmt.Errorf("failed to write audio bytes: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/audio/transcriptions", &body)
	if err != nil {
		return "", fmt.Errorf("failed to build transcription request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call OpenAI transcription API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read transcription response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI transcription API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("failed to parse transcription response: %w", err)
	}
	return parsed.Text, nil
}

func (c *openAIClient) Summarize(ctx context.Context, prompt string) (string, error) {
	payload := map[string]interface{}{
		"model": c.cfg.ChatModel,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal chat completion request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to build chat completion request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call OpenAI chat completion API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read chat completion response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI chat completion API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("failed to parse chat completion response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("OpenAI chat completion API returned no choices")
	}
	return parsed.Choices[0].Message.Content, nil
}

// noopClient is used when OpenAI credentials are not configured. Callers must
// check Enabled() and fall back to non-AI behavior instead of calling these
// methods.
type noopClient struct{}

func (c *noopClient) Enabled() bool {
	return false
}

func (c *noopClient) Transcribe(ctx context.Context, audioBytes []byte, filename string) (string, error) {
	return "", fmt.Errorf("AI not configured: OPENAI_API_KEY is not set")
}

func (c *noopClient) Summarize(ctx context.Context, prompt string) (string, error) {
	return "", fmt.Errorf("AI not configured: OPENAI_API_KEY is not set")
}
