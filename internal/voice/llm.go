package voice

import (
	"context"

	"disci/brain/internal/agent"
)

// ClaudeToolLLM and GeminiToolLLM adapt agent.ClaudeLLM/agent.GeminiLLM's real
// (non-mock) tool-calling implementation to voice.ToolLLM. They exist only to
// convert voice.Turn <-> agent.Turn — the two types are structurally identical
// (Role, Text) but kept distinct so the voice and agent packages don't depend on
// each other's turn semantics; the actual API call logic lives once, in
// agent.ClaudeLLM.DecideTools / agent.GeminiLLM.DecideTools.
type ClaudeToolLLM struct{ *agent.ClaudeLLM }

func (c ClaudeToolLLM) Decide(ctx context.Context, convo []Turn, tools []agent.Tool, clinicID string) (string, []agent.ToolCall, error) {
	return c.ClaudeLLM.DecideTools(ctx, toAgentTurns(convo), tools, clinicID)
}

type GeminiToolLLM struct{ *agent.GeminiLLM }

func (g GeminiToolLLM) Decide(ctx context.Context, convo []Turn, tools []agent.Tool, clinicID string) (string, []agent.ToolCall, error) {
	return g.GeminiLLM.DecideTools(ctx, toAgentTurns(convo), tools, clinicID)
}

func toAgentTurns(convo []Turn) []agent.Turn {
	out := make([]agent.Turn, len(convo))
	for i, t := range convo {
		out[i] = agent.Turn{Role: t.Role, Text: t.Text}
	}
	return out
}

var (
	_ ToolLLM = ClaudeToolLLM{}
	_ ToolLLM = GeminiToolLLM{}
)
