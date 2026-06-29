package agent

import (
	"context"
	"fmt"
	"time"

	"disci/brain/internal/domain"
	"disci/brain/internal/engine"
)

// This is the AGENT's action surface. BrainMeta is an agent: it perceives
// (WhatsApp / voice / form), reasons (LLM), and ACTS through these TOOLS — and
// every tool runs deterministically against the brain. Any LLM (text agent, or a
// realtime VOICE model via function-calling) calls the same tools, so the
// "LLM proposes, brain disposes" guarantee holds on every channel.

// Tool is a function the LLM may call (shape suits Gemini functionDeclarations /
// OpenAI tools / Anthropic tools).
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// ToolCall is the model's request to run a tool with JSON-decoded args.
type ToolCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// ToolResult is what we hand back to the model after running a tool.
type ToolResult struct {
	Name   string         `json:"name"`
	Output map[string]any `json:"output"`
}

// BrainTools executes the agent's tools against the engine.
type BrainTools struct {
	Eng *engine.Engine
	Now func() time.Time
}

func NewBrainTools(eng *engine.Engine) *BrainTools { return &BrainTools{Eng: eng, Now: time.Now} }

func (t *BrainTools) now() time.Time {
	if t.Now != nil {
		return t.Now()
	}
	return time.Now()
}

// Definitions describes the tools for the LLM. Pass these as the model's
// function declarations / tool list.
func (t *BrainTools) Definitions() []Tool {
	num := func() map[string]any { return map[string]any{"type": "number"} }
	str := func() map[string]any { return map[string]any{"type": "string"} }
	return []Tool{
		{Name: "get_availability", Description: "Bir kliniğin en yakın uygun randevu saatlerini getirir.",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{
				"clinicId": str(), "count": num()}, "required": []string{"clinicId"}}},
		{Name: "book_appointment", Description: "Hastayı niteleyip beynin verdiği uygun slota randevu oluşturur.",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{
				"phone": str(), "clinicId": str(), "armId": str(), "segment": str(),
				"urgency": num(), "budgetTry": num()},
				"required": []string{"clinicId", "segment"}}},
		{Name: "escalate_to_human", Description: "Karmaşık/yüksek-değerli durumu klinik personeline aktarır.",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{"reason": str()}}},
	}
}

// Execute runs one tool call and returns its result (errors surface in Output).
func (t *BrainTools) Execute(ctx context.Context, c ToolCall) ToolResult {
	switch c.Name {
	case "get_availability":
		n := int(getFloat(c.Args, "count", 3))
		if n <= 0 || n > 10 {
			n = 3
		}
		slots := t.Eng.NextSlots(getStr(c.Args, "clinicId"), n, t.now())
		opts := make([]string, len(slots))
		for i, s := range slots {
			opts[i] = s.Format(time.RFC3339)
		}
		return ToolResult{Name: c.Name, Output: map[string]any{"slots": opts}}

	case "book_appointment":
		seg := domain.Segment(getStr(c.Args, "segment"))
		urgency := getFloat(c.Args, "urgency", 0.3)
		budget := getFloat(c.Args, "budgetTry", 0)
		intent := 0.55 + urgency*0.2
		if budget > 0 {
			intent += 0.15
		}
		if intent > 0.95 {
			intent = 0.95
		}
		lead := domain.Lead{
			ID:       "agent-" + getStr(c.Args, "phone"),
			Phone:    getStr(c.Args, "phone"),
			ClinicID: getStr(c.Args, "clinicId"),
			ArmID:    getStr(c.Args, "armId"),
			Segment:  seg,
			Features: domain.LeadFeatures{
				HourOfDay: 14, MessagesExchanged: 4, FirstResponseSecs: 20,
				UrgencyScore: urgency, StatedBudgetTRY: budget, IntentScore: intent,
			},
		}
		dec := t.Eng.HandleLead(lead, t.now())
		out := map[string]any{"booked": dec.Booked, "reason": dec.Reason}
		if dec.Booked {
			out["apptTime"] = dec.ApptTime.Format(time.RFC3339)
		}
		return ToolResult{Name: c.Name, Output: out}

	case "escalate_to_human":
		return ToolResult{Name: c.Name, Output: map[string]any{"escalated": true, "reason": getStr(c.Args, "reason")}}

	default:
		return ToolResult{Name: c.Name, Output: map[string]any{"error": fmt.Sprintf("unknown tool %q", c.Name)}}
	}
}

func getStr(m map[string]any, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}
func getFloat(m map[string]any, k string, def float64) float64 {
	switch v := m[k].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	}
	return def
}
