package agent

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type fakeRT struct{ body string }

func (f fakeRT) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(f.body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}, nil
}

var testTools = []Tool{
	{Name: "get_availability", Description: "slots", InputSchema: map[string]any{"type": "object"}},
	{Name: "book_appointment", Description: "book", InputSchema: map[string]any{"type": "object"}},
}

// TestClaudeDecideToolsParsesTextAndToolUse confirms a mixed response (some
// spoken text plus a tool call) is split correctly into (say, calls) — the
// contract voice.Agent.Turn relies on to both speak AND act in one turn.
func TestClaudeDecideToolsParsesTextAndToolUse(t *testing.T) {
	body := `{"content":[
		{"type":"text","text":"Hemen bakıyorum."},
		{"type":"tool_use","name":"get_availability","input":{"clinicId":"umraniye","count":3}}
	]}`
	c := &ClaudeLLM{APIKey: "tok", Model: "claude-test", HTTP: &http.Client{Transport: fakeRT{body: body}}}
	say, calls, err := c.DecideTools(context.Background(), []Turn{{Role: "patient", Text: "merhaba"}}, testTools, "umraniye")
	if err != nil {
		t.Fatalf("DecideTools: %v", err)
	}
	if say != "Hemen bakıyorum." {
		t.Fatalf("expected the spoken text to be extracted, got %q", say)
	}
	if len(calls) != 1 || calls[0].Name != "get_availability" {
		t.Fatalf("expected one get_availability call, got %+v", calls)
	}
	if calls[0].Args["clinicId"] != "umraniye" {
		t.Fatalf("tool args not parsed correctly: %+v", calls[0].Args)
	}
}

// TestClaudeDecideToolsNoCallsIsFine confirms a text-only response (no tool
// call) is valid — the model can just speak without acting.
func TestClaudeDecideToolsNoCallsIsFine(t *testing.T) {
	body := `{"content":[{"type":"text","text":"Nasıl yardımcı olabilirim?"}]}`
	c := &ClaudeLLM{APIKey: "tok", HTTP: &http.Client{Transport: fakeRT{body: body}}}
	say, calls, err := c.DecideTools(context.Background(), nil, testTools, "umraniye")
	if err != nil {
		t.Fatalf("DecideTools: %v", err)
	}
	if say == "" || len(calls) != 0 {
		t.Fatalf("expected text-only response, got say=%q calls=%+v", say, calls)
	}
}

// TestGeminiDecideToolsParsesTextAndFunctionCall mirrors the Claude test for
// Gemini's function-calling response shape.
func TestGeminiDecideToolsParsesTextAndFunctionCall(t *testing.T) {
	body := `{"candidates":[{"content":{"parts":[
		{"text":"Hemen bakıyorum."},
		{"functionCall":{"name":"get_availability","args":{"clinicId":"umraniye","count":3}}}
	]}}]}`
	g := &GeminiLLM{APIKey: "tok", Model: "gemini-test", HTTP: &http.Client{Transport: fakeRT{body: body}}}
	say, calls, err := g.DecideTools(context.Background(), []Turn{{Role: "patient", Text: "merhaba"}}, testTools, "umraniye")
	if err != nil {
		t.Fatalf("DecideTools: %v", err)
	}
	if say != "Hemen bakıyorum." {
		t.Fatalf("expected the spoken text to be extracted, got %q", say)
	}
	if len(calls) != 1 || calls[0].Name != "get_availability" {
		t.Fatalf("expected one get_availability call, got %+v", calls)
	}
}

// TestDecideToolsRequiresAPIKey confirms both real LLMs fail fast (no network
// call) without a configured key, matching Qualify/Compose's existing behaviour.
func TestDecideToolsRequiresAPIKey(t *testing.T) {
	c := &ClaudeLLM{}
	if _, _, err := c.DecideTools(context.Background(), nil, testTools, "x"); err == nil {
		t.Fatal("expected an error with no API key")
	}
	g := &GeminiLLM{}
	if _, _, err := g.DecideTools(context.Background(), nil, testTools, "x"); err == nil {
		t.Fatal("expected an error with no API key")
	}
}
