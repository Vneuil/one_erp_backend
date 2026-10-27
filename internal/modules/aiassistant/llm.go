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

// ToolExec runs one tool call and returns its result as text for the model.
type ToolExec func(ctx context.Context, name string, args map[string]any) (string, error)

const maxToolRounds = 6

var errNoAnswer = fmt.Errorf("penyedia AI tidak mengembalikan jawaban")

func nonEmpty(s string) (string, error) {
	if strings.TrimSpace(s) == "" {
		return "", errNoAnswer
	}
	return s, nil
}

var llmHTTP = &http.Client{Timeout: 90 * time.Second}

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

func parseArgs(raw string) map[string]any {
	args := map[string]any{}
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &args)
	}
	return args
}

// runTool never fails the conversation: errors go back to the model as text.
func runTool(ctx context.Context, exec ToolExec, name string, args map[string]any) string {
	out, err := exec(ctx, name, args)
	if err != nil {
		return "Error: " + err.Error()
	}
	return truncate(out, 12000)
}

func hasProps(t tool) bool {
	p, _ := t.InputSchema["properties"].(map[string]any)
	return len(p) > 0
}

// Complete sends the conversation to the selected provider, letting the model
// call the given tools (read-only ERP reports) until it produces a final answer.
func Complete(ctx context.Context, provider, model, apiKey, system string, msgs []ChatMessage, defs []tool, exec ToolExec) (string, error) {
	switch provider {
	case "openai":
		return completeOpenAI(ctx, model, apiKey, system, msgs, defs, exec)
	case "deepseek":
		return completeDeepSeek(ctx, model, apiKey, system, msgs, defs, exec)
	case "claude":
		return completeClaude(ctx, model, apiKey, system, msgs, defs, exec)
	case "gemini":
		return completeGemini(ctx, model, apiKey, system, msgs, defs, exec)
	}
	return "", fmt.Errorf("penyedia AI tidak dikenal: %s", provider)
}

func completeOpenAI(ctx context.Context, model, key, system string, msgs []ChatMessage, defs []tool, exec ToolExec) (string, error) {
	// OpenAI recommends the Responses API for current models.
	toolDefs := make([]map[string]any, 0, len(defs))
	for _, t := range defs {
		toolDefs = append(toolDefs, map[string]any{"type": "function", "name": t.Name, "description": t.Description, "parameters": t.InputSchema})
	}
	input := make([]any, 0, len(msgs))
	for _, m := range msgs {
		input = append(input, m)
	}
	for round := 0; round < maxToolRounds; round++ {
		var out struct {
			Output []json.RawMessage `json:"output"`
		}
		if err := postJSON(ctx, "https://api.openai.com/v1/responses", map[string]string{"Authorization": "Bearer " + key},
			map[string]any{"model": model, "instructions": system, "input": input, "tools": toolDefs}, &out); err != nil {
			return "", err
		}
		var text strings.Builder
		type call struct{ id, name, args string }
		var calls []call
		for _, raw := range out.Output {
			var item struct {
				Type      string `json:"type"`
				CallID    string `json:"call_id"`
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
				Content   []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			}
			if json.Unmarshal(raw, &item) != nil {
				continue
			}
			input = append(input, raw) // keep reasoning/call items for the next round
			switch item.Type {
			case "function_call":
				calls = append(calls, call{item.CallID, item.Name, item.Arguments})
			case "message":
				for _, c := range item.Content {
					if c.Type == "output_text" {
						text.WriteString(c.Text)
					}
				}
			}
		}
		if len(calls) == 0 {
			return nonEmpty(text.String())
		}
		for _, c := range calls {
			input = append(input, map[string]any{"type": "function_call_output", "call_id": c.id, "output": runTool(ctx, exec, c.name, parseArgs(c.args))})
		}
	}
	return "", fmt.Errorf("terlalu banyak pemanggilan tool")
}

func completeDeepSeek(ctx context.Context, model, key, system string, msgs []ChatMessage, defs []tool, exec ToolExec) (string, error) {
	toolDefs := make([]map[string]any, 0, len(defs))
	for _, t := range defs {
		toolDefs = append(toolDefs, map[string]any{"type": "function", "function": map[string]any{"name": t.Name, "description": t.Description, "parameters": t.InputSchema}})
	}
	messages := []any{ChatMessage{Role: "system", Content: system}}
	for _, m := range msgs {
		messages = append(messages, m)
	}
	for round := 0; round < maxToolRounds; round++ {
		var out struct {
			Choices []struct {
				Message json.RawMessage `json:"message"`
			} `json:"choices"`
		}
		if err := postJSON(ctx, "https://api.deepseek.com/chat/completions", map[string]string{"Authorization": "Bearer " + key},
			map[string]any{"model": model, "messages": messages, "tools": toolDefs, "stream": false}, &out); err != nil {
			return "", err
		}
		if len(out.Choices) == 0 {
			return "", errNoAnswer
		}
		var msg struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		}
		if err := json.Unmarshal(out.Choices[0].Message, &msg); err != nil {
			return "", err
		}
		if len(msg.ToolCalls) == 0 {
			return nonEmpty(msg.Content)
		}
		messages = append(messages, out.Choices[0].Message)
		for _, c := range msg.ToolCalls {
			messages = append(messages, map[string]any{"role": "tool", "tool_call_id": c.ID,
				"content": runTool(ctx, exec, c.Function.Name, parseArgs(c.Function.Arguments))})
		}
	}
	return "", fmt.Errorf("terlalu banyak pemanggilan tool")
}

func completeClaude(ctx context.Context, model, key, system string, msgs []ChatMessage, defs []tool, exec ToolExec) (string, error) {
	toolDefs := make([]map[string]any, 0, len(defs))
	for _, t := range defs {
		toolDefs = append(toolDefs, map[string]any{"name": t.Name, "description": t.Description, "input_schema": t.InputSchema})
	}
	messages := make([]any, 0, len(msgs))
	for _, m := range msgs {
		messages = append(messages, m)
	}
	for round := 0; round < maxToolRounds; round++ {
		var out struct {
			Content []json.RawMessage `json:"content"`
		}
		if err := postJSON(ctx, "https://api.anthropic.com/v1/messages",
			map[string]string{"x-api-key": key, "anthropic-version": "2023-06-01"},
			map[string]any{"model": model, "max_tokens": 2048, "system": system, "messages": messages, "tools": toolDefs}, &out); err != nil {
			return "", err
		}
		var text strings.Builder
		var results []map[string]any
		for _, raw := range out.Content {
			var b struct {
				Type  string          `json:"type"`
				Text  string          `json:"text"`
				ID    string          `json:"id"`
				Name  string          `json:"name"`
				Input json.RawMessage `json:"input"`
			}
			if json.Unmarshal(raw, &b) != nil {
				continue
			}
			switch b.Type {
			case "text":
				text.WriteString(b.Text)
			case "tool_use":
				results = append(results, map[string]any{"type": "tool_result", "tool_use_id": b.ID,
					"content": runTool(ctx, exec, b.Name, parseArgs(string(b.Input)))})
			}
		}
		if len(results) == 0 {
			return nonEmpty(text.String())
		}
		// Echo the full assistant turn (including any thinking blocks), then the results.
		messages = append(messages, map[string]any{"role": "assistant", "content": out.Content}, map[string]any{"role": "user", "content": results})
	}
	return "", fmt.Errorf("terlalu banyak pemanggilan tool")
}

func completeGemini(ctx context.Context, model, key, system string, msgs []ChatMessage, defs []tool, exec ToolExec) (string, error) {
	decls := make([]map[string]any, 0, len(defs))
	for _, t := range defs {
		d := map[string]any{"name": t.Name, "description": t.Description}
		if hasProps(t) { // Gemini rejects an empty OBJECT schema
			d["parameters"] = t.InputSchema
		}
		decls = append(decls, d)
	}
	contents := make([]any, 0, len(msgs))
	for _, m := range msgs {
		role := "user"
		if m.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, map[string]any{"role": role, "parts": []map[string]string{{"text": m.Content}}})
	}
	url := "https://generativelanguage.googleapis.com/v1beta/models/" + model + ":generateContent"
	for round := 0; round < maxToolRounds; round++ {
		var out struct {
			Candidates []struct {
				Content struct {
					Parts []json.RawMessage `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if err := postJSON(ctx, url, map[string]string{"x-goog-api-key": key}, map[string]any{
			"systemInstruction": map[string]any{"parts": []map[string]string{{"text": system}}},
			"contents":          contents,
			"tools":             []map[string]any{{"functionDeclarations": decls}},
		}, &out); err != nil {
			return "", err
		}
		if len(out.Candidates) == 0 {
			return "", errNoAnswer
		}
		parts := out.Candidates[0].Content.Parts
		var text strings.Builder
		var responses []map[string]any
		for _, raw := range parts {
			var p struct {
				Text         string `json:"text"`
				FunctionCall *struct {
					Name string         `json:"name"`
					Args map[string]any `json:"args"`
				} `json:"functionCall"`
			}
			if json.Unmarshal(raw, &p) != nil {
				continue
			}
			if p.FunctionCall != nil {
				responses = append(responses, map[string]any{"functionResponse": map[string]any{
					"name": p.FunctionCall.Name, "response": map[string]any{"result": runTool(ctx, exec, p.FunctionCall.Name, p.FunctionCall.Args)}}})
			} else {
				text.WriteString(p.Text)
			}
		}
		if len(responses) == 0 {
			return nonEmpty(text.String())
		}
		// Echo the model turn as-is (keeps thought signatures), then the tool results.
		contents = append(contents, map[string]any{"role": "model", "parts": parts}, map[string]any{"role": "user", "parts": responses})
	}
	return "", fmt.Errorf("terlalu banyak pemanggilan tool")
}
