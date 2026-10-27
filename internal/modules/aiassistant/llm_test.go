package aiassistant

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type fakeTransport struct {
	replies []string // one JSON body per call, in order
	bodies  []string
}

func (f *fakeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	b, _ := io.ReadAll(r.Body)
	f.bodies = append(f.bodies, string(b))
	body := f.replies[len(f.bodies)-1]
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
}

func runWith(t *testing.T, provider string, replies ...string) (string, *fakeTransport, []string) {
	t.Helper()
	ft := &fakeTransport{replies: replies}
	old := llmHTTP.Transport
	llmHTTP.Transport = ft
	defer func() { llmHTTP.Transport = old }()
	var called []string
	exec := func(_ context.Context, name string, args map[string]any) (string, error) {
		called = append(called, name)
		return `{"total":42}`, nil
	}
	out, err := Complete(context.Background(), provider, "m", "k", "sys", []ChatMessage{{Role: "user", Content: "stok?"}}, tools, exec)
	if err != nil {
		t.Fatalf("%s: %v", provider, err)
	}
	return out, ft, called
}

func TestToolLoops(t *testing.T) {
	cases := map[string][]string{
		"claude": {
			`{"content":[{"type":"thinking","thinking":"x"},{"type":"tool_use","id":"t1","name":"inventory_valuation","input":{}}]}`,
			`{"content":[{"type":"text","text":"Totalnya 42"}]}`,
		},
		"openai": {
			`{"output":[{"type":"function_call","call_id":"c1","name":"inventory_valuation","arguments":"{}"}]}`,
			`{"output":[{"type":"message","content":[{"type":"output_text","text":"Totalnya 42"}]}]}`,
		},
		"deepseek": {
			`{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"c1","function":{"name":"inventory_valuation","arguments":"{}"}}]}}]}`,
			`{"choices":[{"message":{"role":"assistant","content":"Totalnya 42"}}]}`,
		},
		"gemini": {
			`{"candidates":[{"content":{"parts":[{"functionCall":{"name":"inventory_valuation","args":{}}}]}}]}`,
			`{"candidates":[{"content":{"parts":[{"text":"Totalnya 42"}]}}]}`,
		},
	}
	for provider, replies := range cases {
		out, ft, called := runWith(t, provider, replies...)
		if out != "Totalnya 42" {
			t.Errorf("%s: got %q", provider, out)
		}
		if len(called) != 1 || called[0] != "inventory_valuation" {
			t.Errorf("%s: tool calls %v", provider, called)
		}
		if len(ft.bodies) != 2 || !strings.Contains(ft.bodies[1], `{\"total\":42}`) {
			t.Errorf("%s: second request lacks tool result: %v", provider, ft.bodies)
		}
		if !strings.Contains(ft.bodies[0], "inventory_valuation") {
			t.Errorf("%s: tools not sent", provider)
		}
	}
}
