package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// GeminiLLM is a Google Gemini implementation of LLM, using the Generative
// Language API with function-calling for structured qualification output. Same
// interface as MockLLM/ClaudeLLM, so it drops in with one env var.
//
// Qualify forces a function call (mode=ANY) so the model returns a schema-valid
// Qualification; the booking decision still happens in the deterministic brain.
type GeminiLLM struct {
	APIKey string
	Model  string
	HTTP   *http.Client
}

// Default Gemini model for the qualifier. Override via NewGemini / GEMINI_MODEL;
// set whatever model your key has access to.
const defaultGeminiModel = "gemini-2.0-flash"

// NewGemini builds a Gemini-backed LLM. Reads GEMINI_API_KEY if apiKey == "".
func NewGemini(apiKey, model string) *GeminiLLM {
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if model == "" {
		model = defaultGeminiModel
	}
	return &GeminiLLM{APIKey: apiKey, Model: model, HTTP: &http.Client{Timeout: 30 * time.Second}}
}

var geminiQualFn = map[string]any{
	"name":        "submit_qualification",
	"description": "Submit the structured qualification of the dental lead.",
	"parameters": map[string]any{
		"type": "object",
		"properties": map[string]any{
			"segment":   map[string]any{"type": "string", "enum": []string{"aesthetic", "implant", "ortho", "general"}},
			"treatment": map[string]any{"type": "string"},
			"urgency":   map[string]any{"type": "number", "description": "Urgency as a decimal between 0.0 and 1.0 (e.g. 0.85 for acute pain)."},
			"intent":    map[string]any{"type": "number", "description": "Readiness/seriousness as a decimal between 0.0 and 1.0."},
			"budgetTry": map[string]any{"type": "number", "description": "Stated budget converted to Turkish Lira (TRY); 0 if unknown."},
			"locale":    map[string]any{"type": "string", "enum": []string{"tr", "en"}},
			"done":      map[string]any{"type": "boolean"},
			"askNext":   map[string]any{"type": "string"},
		},
		"required": []string{"segment", "urgency", "intent", "locale", "done"},
	},
}

// Name identifies this provider in diagnostics.
func (g *GeminiLLM) Name() string { return "gemini:" + g.Model }

// Qualify runs structured qualification via a forced function call.
func (g *GeminiLLM) Qualify(ctx context.Context, convo []Turn) (Qualification, error) {
	body := map[string]any{
		"systemInstruction": map[string]any{"parts": []any{map[string]any{"text": qualSystemPrompt}}},
		"contents":          geminiContents(convo),
		"tools":             []any{map[string]any{"functionDeclarations": []any{geminiQualFn}}},
		"toolConfig": map[string]any{"functionCallingConfig": map[string]any{
			"mode": "ANY", "allowedFunctionNames": []string{"submit_qualification"}}},
	}
	resp, err := g.call(ctx, body)
	if err != nil {
		return Qualification{}, err
	}
	for _, c := range resp.Candidates {
		for _, p := range c.Content.Parts {
			if p.FunctionCall != nil && p.FunctionCall.Name == "submit_qualification" {
				var q Qualification
				if err := json.Unmarshal(p.FunctionCall.Args, &q); err != nil {
					return Qualification{}, err
				}
				q.Messages = countRole(convo, "patient")
				return q, nil
			}
		}
	}
	return Qualification{}, fmt.Errorf("gemini: no qualification function call in response")
}

// Compose phrases the patient reply around the brain's authoritative decision.
func (g *GeminiLLM) Compose(ctx context.Context, convo []Turn, rc ReplyContext) (string, error) {
	fact := "decision: not booked; reason: " + rc.Reason
	switch {
	case rc.AlreadyBooked:
		fact = "session already booked; appointment_time: " + rc.ApptTime.Format(time.RFC1123) + "; this is a follow-up — confirm/acknowledge, do NOT create a new appointment"
	case rc.Booked:
		fact = "decision: BOOKED; appointment_time: " + rc.ApptTime.Format(time.RFC1123)
	}
	contents := geminiContents(convo)
	contents = append(contents, map[string]any{"role": "user",
		"parts": []any{map[string]any{"text": "[system fact — do not alter] " + fact}}})
	body := map[string]any{
		"systemInstruction": map[string]any{"parts": []any{map[string]any{"text": composeSystemPrompt}}},
		"contents":          contents,
	}
	resp, err := g.call(ctx, body)
	if err != nil {
		return "", err
	}
	for _, c := range resp.Candidates {
		for _, p := range c.Content.Parts {
			if p.Text != "" {
				return p.Text, nil
			}
		}
	}
	return "", fmt.Errorf("gemini: empty compose response")
}

// DecideTools runs one voice turn with tool-calling: given the conversation and
// the brain's tool definitions, Gemini may speak, call tools, or both (mode
// "AUTO" — unlike Qualify's forced call). Same real (non-mock) contract as
// ClaudeLLM.DecideTools; voice package adapters wrap whichever is configured.
func (g *GeminiLLM) DecideTools(ctx context.Context, convo []Turn, tools []Tool, clinicID string) (string, []ToolCall, error) {
	fnDecls := make([]any, len(tools))
	for i, t := range tools {
		fnDecls[i] = map[string]any{"name": t.Name, "description": t.Description, "parameters": t.InputSchema}
	}
	contents := geminiContents(convo)
	contents = append(contents, map[string]any{"role": "user",
		"parts": []any{map[string]any{"text": fmt.Sprintf("[system fact — do not alter] active clinic id: %s", clinicID)}}})
	body := map[string]any{
		"systemInstruction": map[string]any{"parts": []any{map[string]any{"text": voiceToolSystemPrompt}}},
		"contents":          contents,
		"tools":             []any{map[string]any{"functionDeclarations": fnDecls}},
		"toolConfig":        map[string]any{"functionCallingConfig": map[string]any{"mode": "AUTO"}},
	}
	resp, err := g.call(ctx, body)
	if err != nil {
		return "", nil, err
	}
	var say string
	var calls []ToolCall
	for _, c := range resp.Candidates {
		for _, p := range c.Content.Parts {
			if p.Text != "" {
				say += p.Text
			}
			if p.FunctionCall != nil {
				var args map[string]any
				if jerr := json.Unmarshal(p.FunctionCall.Args, &args); jerr == nil {
					calls = append(calls, ToolCall{Name: p.FunctionCall.Name, Args: args})
				}
			}
		}
	}
	return say, calls, nil
}

func geminiContents(convo []Turn) []any {
	out := make([]any, 0, len(convo))
	for _, t := range convo {
		role := "user"
		if t.Role == "agent" {
			role = "model"
		}
		out = append(out, map[string]any{"role": role, "parts": []any{map[string]any{"text": t.Text}}})
	}
	if len(out) == 0 {
		out = append(out, map[string]any{"role": "user", "parts": []any{map[string]any{"text": "Merhaba"}}})
	}
	return out
}

type geminiResp struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text         string `json:"text"`
				FunctionCall *struct {
					Name string          `json:"name"`
					Args json.RawMessage `json:"args"`
				} `json:"functionCall"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (g *GeminiLLM) call(ctx context.Context, body map[string]any) (geminiResp, error) {
	var out geminiResp
	if g.APIKey == "" {
		return out, fmt.Errorf("gemini: no API key (set GEMINI_API_KEY)")
	}
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.Model, g.APIKey)
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return out, err
	}
	req.Header.Set("content-type", "application/json")
	resp, err := g.HTTP.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return out, fmt.Errorf("gemini: status %d: %s", resp.StatusCode, string(data))
	}
	return out, json.Unmarshal(data, &out)
}

var _ LLM = (*GeminiLLM)(nil)
