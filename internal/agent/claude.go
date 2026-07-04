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

// ClaudeLLM is the PRODUCTION implementation of LLM, backed by the Anthropic
// Messages API with tool-use for structured output. It satisfies the same
// interface as MockLLM, so swapping it in is a one-line change in main; tests
// keep using the mock (no API key needed).
//
// Qualification uses a forced tool call (tool_choice) so the model returns a
// schema-valid Qualification rather than free text — the structured-output
// pattern. The booking decision still happens in the deterministic brain.
type ClaudeLLM struct {
	APIKey string
	Model  string
	HTTP   *http.Client
}

// Default model for the qualifier: a fast, capable Claude tier is the right fit
// for high-volume WhatsApp qualification. Override via NewClaude.
const defaultQualifierModel = "claude-sonnet-4-6"

// NewClaude builds a Claude-backed LLM. Reads ANTHROPIC_API_KEY if apiKey == "".
func NewClaude(apiKey, model string) *ClaudeLLM {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if model == "" {
		model = defaultQualifierModel
	}
	return &ClaudeLLM{APIKey: apiKey, Model: model, HTTP: &http.Client{Timeout: 30 * time.Second}}
}

const qualSystemPrompt = `You are the intake assistant for a network of Istanbul dental clinics, chatting on WhatsApp in the patient's language (Turkish or English). Qualify the lead: identify the treatment segment, urgency, how serious/ready they are, and any stated budget (convert to TRY). Be warm and concise. Call submit_qualification with your structured assessment. Set done=true only when you have enough to book; otherwise set done=false and put your next question in askNext. Never promise a specific appointment time — the scheduling system decides that.`

var qualificationTool = map[string]any{
	"name":        "submit_qualification",
	"description": "Submit the structured qualification of the lead.",
	"input_schema": map[string]any{
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
func (c *ClaudeLLM) Name() string { return "claude:" + c.Model }

// Qualify calls Claude with a forced tool call and decodes the structured result.
func (c *ClaudeLLM) Qualify(ctx context.Context, convo []Turn) (Qualification, error) {
	body := map[string]any{
		"model":       c.Model,
		"max_tokens":  512,
		"system":      qualSystemPrompt,
		"messages":    toAnthropicMessages(convo),
		"tools":       []any{qualificationTool},
		"tool_choice": map[string]any{"type": "tool", "name": "submit_qualification"},
	}
	resp, err := c.call(ctx, body)
	if err != nil {
		return Qualification{}, err
	}
	for _, blk := range resp.Content {
		if blk.Type == "tool_use" && blk.Name == "submit_qualification" {
			var q Qualification
			if err := json.Unmarshal(blk.Input, &q); err != nil {
				return Qualification{}, err
			}
			q.Messages = countRole(convo, "patient")
			return q, nil
		}
	}
	return Qualification{}, fmt.Errorf("claude: no qualification tool call in response")
}

const composeSystemPrompt = `You are a warm WhatsApp intake assistant for an Istanbul dental clinic. Write a short reply in the patient's language. You are given the booking system's AUTHORITATIVE decision. If booked, confirm the EXACT given appointment time and ask them to confirm. If not booked, be reassuring and say you'll find the next available slot. NEVER invent or change an appointment time.`

// Compose asks Claude to phrase the patient reply around the brain's decision.
func (c *ClaudeLLM) Compose(ctx context.Context, convo []Turn, rc ReplyContext) (string, error) {
	fact := "decision: not booked; reason: " + rc.Reason
	if rc.Booked {
		fact = "decision: BOOKED; appointment_time: " + rc.ApptTime.Format(time.RFC1123)
	}
	msgs := toAnthropicMessages(convo)
	msgs = append(msgs, map[string]any{"role": "user", "content": "[system fact — do not alter] " + fact})
	body := map[string]any{
		"model": c.Model, "max_tokens": 256, "system": composeSystemPrompt, "messages": msgs,
	}
	resp, err := c.call(ctx, body)
	if err != nil {
		return "", err
	}
	for _, blk := range resp.Content {
		if blk.Type == "text" {
			return blk.Text, nil
		}
	}
	return "", fmt.Errorf("claude: empty compose response")
}

const voiceToolSystemPrompt = `You are a warm phone-call assistant for an Istanbul dental clinic network. Speak naturally and briefly in the caller's language (Turkish or English) — this is a voice call, not chat. Use the available tools to check availability and book appointments; the tools are the ONLY source of truth for scheduling. NEVER state that an appointment is booked, or invent a time, unless a tool call's result confirms it. If the situation is complex or the caller is upset, use escalate_to_human.`

// DecideTools runs one voice turn with tool-calling: given the conversation and
// the brain's tool definitions, Claude may speak, call tools, or both. This is
// the real (non-mock) implementation of voice.ToolLLM's Decide method — voice
// package adapters wrap this so the voice channel's LLM choice mirrors the text
// agent's (Gemini → Claude → mock, whichever key is configured).
func (c *ClaudeLLM) DecideTools(ctx context.Context, convo []Turn, tools []Tool, clinicID string) (string, []ToolCall, error) {
	anthTools := make([]any, len(tools))
	for i, t := range tools {
		anthTools[i] = map[string]any{"name": t.Name, "description": t.Description, "input_schema": t.InputSchema}
	}
	msgs := toAnthropicMessages(convo)
	msgs = append(msgs, map[string]any{"role": "user",
		"content": fmt.Sprintf("[system fact — do not alter] active clinic id: %s", clinicID)})
	body := map[string]any{
		"model": c.Model, "max_tokens": 512, "system": voiceToolSystemPrompt,
		"messages": msgs, "tools": anthTools,
	}
	resp, err := c.call(ctx, body)
	if err != nil {
		return "", nil, err
	}
	var say string
	var calls []ToolCall
	for _, blk := range resp.Content {
		switch blk.Type {
		case "text":
			say += blk.Text
		case "tool_use":
			var args map[string]any
			if jerr := json.Unmarshal(blk.Input, &args); jerr == nil {
				calls = append(calls, ToolCall{Name: blk.Name, Args: args})
			}
		}
	}
	return say, calls, nil
}

func toAnthropicMessages(convo []Turn) []any {
	out := make([]any, 0, len(convo))
	for _, t := range convo {
		role := "user"
		if t.Role == "agent" {
			role = "assistant"
		}
		out = append(out, map[string]any{"role": role, "content": t.Text})
	}
	if len(out) == 0 {
		out = append(out, map[string]any{"role": "user", "content": "Merhaba"})
	}
	return out
}

type anthropicResp struct {
	Content []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	} `json:"content"`
}

func (c *ClaudeLLM) call(ctx context.Context, body map[string]any) (anthropicResp, error) {
	var out anthropicResp
	if c.APIKey == "" {
		return out, fmt.Errorf("claude: no API key (set ANTHROPIC_API_KEY)")
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(b))
	if err != nil {
		return out, err
	}
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return out, fmt.Errorf("claude: status %d: %s", resp.StatusCode, string(data))
	}
	return out, json.Unmarshal(data, &out)
}

var _ LLM = (*ClaudeLLM)(nil)
