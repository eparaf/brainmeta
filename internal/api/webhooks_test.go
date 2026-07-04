package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"disci/brain/internal/agent"
	"disci/brain/internal/domain"
	"disci/brain/internal/meta"
	"disci/brain/internal/store"
)

type fakeLeadFetcher struct {
	lead  meta.Lead
	err   error
	calls int
}

func (f *fakeLeadFetcher) FetchLead(ctx context.Context, leadgenID string) (meta.Lead, error) {
	f.calls++
	return f.lead, f.err
}

func signBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func metaLeadsBody(leadgenID string) []byte {
	payload := map[string]any{
		"entry": []map[string]any{{
			"changes": []map[string]any{{
				"field": "leadgen",
				"value": map[string]any{"leadgen_id": leadgenID},
			}},
		}},
	}
	b, _ := json.Marshal(payload)
	return b
}

// TestHandleMetaLeadsSignatureEnforced: a bad HMAC signature must be rejected,
// same guarantee as the WhatsApp inbound webhook — otherwise anyone could POST
// fake "leadgen" events and inject spam leads into the agent.
func TestHandleMetaLeadsSignatureEnforced(t *testing.T) {
	s, _ := newTestServer()
	s.appSecret = "shh"
	body := metaLeadsBody("lg1")
	r := httptest.NewRequest("POST", "/webhooks/meta-leads", bytes.NewReader(body))
	r.Header.Set("X-Hub-Signature-256", "sha256=deadbeef")
	w := httptest.NewRecorder()
	s.handleMetaLeads(w, r)
	if w.Code != 401 {
		t.Fatalf("bad signature should be 401, got %d", w.Code)
	}
}

// TestHandleMetaLeadsValidSignaturePasses: a correctly-signed payload proceeds.
func TestHandleMetaLeadsValidSignaturePasses(t *testing.T) {
	s, _ := newTestServer()
	s.appSecret = "shh"
	s.agent = agent.New(agent.MockLLM{}, s.eng)
	fake := &fakeLeadFetcher{lead: meta.Lead{ID: "lg1", Name: "Ayşe", Phone: "+905551230000"}}
	s.SetLeadAds(fake)

	body := metaLeadsBody("lg1")
	r := httptest.NewRequest("POST", "/webhooks/meta-leads", bytes.NewReader(body))
	r.Header.Set("X-Hub-Signature-256", signBody("shh", body))
	w := httptest.NewRecorder()
	s.handleMetaLeads(w, r)
	if w.Code != 200 {
		t.Fatalf("correctly-signed payload should be 200, got %d", w.Code)
	}
	if fake.calls != 1 {
		t.Fatalf("expected FetchLead to be called once, got %d", fake.calls)
	}
}

// TestHandleMetaLeadsFetchesAndRoutesToAgent: the leadgen_id from the webhook is
// used to fetch the real lead (name/phone), which then flows through the SAME
// agent path a website form uses — proving the leadgen id alone (the historical
// gap) is no longer where processing stops.
func TestHandleMetaLeadsFetchesAndRoutesToAgent(t *testing.T) {
	s, _ := newTestServer()
	s.agent = agent.New(agent.MockLLM{}, s.eng)
	fake := &fakeLeadFetcher{lead: meta.Lead{ID: "lg1", Name: "Ayşe", Phone: "+905551230000"}}
	s.SetLeadAds(fake)

	body := metaLeadsBody("lg1")
	r := httptest.NewRequest("POST", "/webhooks/meta-leads", bytes.NewReader(body)) // no appSecret set → sig check is a no-op
	w := httptest.NewRecorder()
	s.handleMetaLeads(w, r)
	if w.Code != 200 {
		t.Fatalf("want 200 ack, got %d", w.Code)
	}
	if fake.calls != 1 {
		t.Fatalf("expected FetchLead to be called once, got %d", fake.calls)
	}
	if _, ok := s.sessions.Get("+905551230000"); !ok {
		t.Fatal("lead should have been routed through the agent (no session found)")
	}
}

// TestHandleMetaLeadsSkipsWhenNoPhone: a lead with no phone in field_data must
// not crash or spuriously create an agent session.
func TestHandleMetaLeadsSkipsWhenNoPhone(t *testing.T) {
	s, _ := newTestServer()
	s.agent = agent.New(agent.MockLLM{}, s.eng)
	fake := &fakeLeadFetcher{lead: meta.Lead{ID: "lg1", Name: "No Phone"}}
	s.SetLeadAds(fake)

	body := metaLeadsBody("lg1")
	r := httptest.NewRequest("POST", "/webhooks/meta-leads", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.handleMetaLeads(w, r)
	if w.Code != 200 {
		t.Fatalf("want 200 ack even when lead has no phone, got %d", w.Code)
	}
}

// TestHandleMetaLeadsNoLeadAdsConfiguredJustAcks: without a Graph API token
// configured (SetLeadAds never called), the webhook must still ack cleanly.
func TestHandleMetaLeadsNoLeadAdsConfiguredJustAcks(t *testing.T) {
	s, _ := newTestServer()
	body := metaLeadsBody("lg1")
	r := httptest.NewRequest("POST", "/webhooks/meta-leads", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.handleMetaLeads(w, r)
	if w.Code != 200 {
		t.Fatalf("want 200 ack, got %d", w.Code)
	}
}

// TestHandleWAInboundResolvesClinicByPhoneNumberID is the end-to-end proof of
// the per-clinic WhatsApp routing resolver: a clinic that completed Embedded
// Signup (an OAuthToken row with its phone_number_id) gets inbound messages
// on THAT number routed to it — the lead the brain ultimately saves carries the
// resolved ClinicID, not the previous unconditional "" (brain routes) behaviour.
func TestHandleWAInboundResolvesClinicByPhoneNumberID(t *testing.T) {
	s, st := newTestServer()
	s.eng.RegisterClinic(domain.Clinic{
		ID: "umraniye", Segment: domain.SegmentImplant, CloseRate: 0.42,
		DailyCapacity: 10, GuaranteedApptsPerMonth: 80, MonthlyAdBudget: 220_000,
	})
	s.agent = agent.New(agent.MockLLM{}, s.eng)
	st.UpsertOAuthToken(domain.OAuthToken{
		ClinicID: "umraniye", Provider: "meta", Type: "whatsapp",
		PhoneNumberID: "555000111", RefreshToken: "wa-tok",
	})

	payload := map[string]any{
		"entry": []map[string]any{{
			"changes": []map[string]any{{
				"value": map[string]any{
					"metadata": map[string]any{"phone_number_id": "555000111"},
					"messages": []map[string]any{{
						"from": "905551110000", "type": "text",
						"text": map[string]any{"body": "implant yaptırmak istiyorum, bütçem 60000 TL acil ağrım var"},
					}},
				},
			}},
		}},
	}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	s.handleWAInbound(w, httptest.NewRequest("POST", "/webhooks/whatsapp", bytes.NewReader(body)))
	if w.Code != 200 {
		t.Fatalf("want 200 ack, got %d", w.Code)
	}

	leads := st.ListLeads(store.LeadFilter{})
	if len(leads) != 1 {
		t.Fatalf("expected exactly one lead saved, got %d", len(leads))
	}
	if leads[0].ClinicID != "umraniye" {
		t.Fatalf("lead should be routed to the phone_number_id's clinic 'umraniye', got %q", leads[0].ClinicID)
	}
}

// TestHandleWAInboundUnresolvedPhoneFallsBackToMarketplace: a message on a
// number no clinic has claimed must NOT crash and must still process (brain
// routes across the marketplace, unchanged prior behaviour).
func TestHandleWAInboundUnresolvedPhoneFallsBackToMarketplace(t *testing.T) {
	s, _ := newTestServer()
	s.agent = agent.New(agent.MockLLM{}, s.eng)
	payload := map[string]any{
		"entry": []map[string]any{{
			"changes": []map[string]any{{
				"value": map[string]any{
					"metadata": map[string]any{"phone_number_id": "unclaimed-999"},
					"messages": []map[string]any{{
						"from": "905551110000", "type": "text",
						"text": map[string]any{"body": "merhaba"},
					}},
				},
			}},
		}},
	}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	s.handleWAInbound(w, httptest.NewRequest("POST", "/webhooks/whatsapp", bytes.NewReader(body)))
	if w.Code != 200 {
		t.Fatalf("want 200 ack, got %d", w.Code)
	}
}
