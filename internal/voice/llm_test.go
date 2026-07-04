package voice

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"disci/brain/internal/agent"
)

type fakeRT struct{ body string }

func (f fakeRT) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(f.body))}, nil
}

// TestClaudeToolLLMDrivesFullVoiceTurn is the end-to-end proof that wiring a
// REAL LLM (via the adapter) into voice.Agent works exactly like MockLLM does:
// a fake Claude response calling book_appointment flows through Agent.Turn,
// converts voice.Turn <-> agent.Turn correctly, and the brain (not the model)
// produces the authoritative booking. This is the gap #16 closes — previously
// voice ALWAYS used MockLLM regardless of configured API keys.
func TestClaudeToolLLMDrivesFullVoiceTurn(t *testing.T) {
	a, sp := newAgent() // from voice_test.go: real BrainTools + engine, umraniye registered
	body := `{"content":[
		{"type":"text","text":"Hemen ayarlıyorum."},
		{"type":"tool_use","name":"book_appointment","input":{"phone":"+90555","clinicId":"umraniye","segment":"implant","urgency":0.9,"budgetTry":60000}}
	]}`
	claude := &agent.ClaudeLLM{APIKey: "tok", HTTP: &http.Client{Transport: fakeRT{body: body}}}
	a.LLM = ClaudeToolLLM{ClaudeLLM: claude}

	sess := &Session{Phone: "+90555", ClinicID: "umraniye"}
	reply, err := a.Turn(context.Background(), sess, "implant istiyorum acil ağrım var")
	if err != nil {
		t.Fatalf("Turn: %v", err)
	}
	if !sess.Booked {
		t.Fatalf("expected the real tool-calling adapter to drive a booking, reply=%q", reply)
	}
	if len(sp.Said) != 1 {
		t.Fatal("speaker should have spoken once")
	}
}

// TestGeminiToolLLMDrivesFullVoiceTurn mirrors the above for the Gemini adapter.
func TestGeminiToolLLMDrivesFullVoiceTurn(t *testing.T) {
	a, _ := newAgent()
	body := `{"candidates":[{"content":{"parts":[
		{"text":"Hemen ayarlıyorum."},
		{"functionCall":{"name":"book_appointment","args":{"phone":"+90555","clinicId":"umraniye","segment":"implant","urgency":0.9,"budgetTry":60000}}}
	]}}]}`
	gem := &agent.GeminiLLM{APIKey: "tok", HTTP: &http.Client{Transport: fakeRT{body: body}}}
	a.LLM = GeminiToolLLM{GeminiLLM: gem}

	sess := &Session{Phone: "+90555", ClinicID: "umraniye"}
	if _, err := a.Turn(context.Background(), sess, "implant istiyorum acil ağrım var"); err != nil {
		t.Fatalf("Turn: %v", err)
	}
	if !sess.Booked {
		t.Fatal("expected the Gemini tool-calling adapter to drive a booking")
	}
}
