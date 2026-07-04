package store

import (
	"testing"

	"disci/brain/internal/domain"
)

// TestOAuthTokenTypeAvoidsCollision guards the real bug this package had: both
// "whatsapp" and "meta_ads" connection types map to Provider="meta", so keying
// storage on (clinicID, provider) alone made connecting BOTH for one clinic
// silently overwrite one token with the other's. Keying on Type fixes it.
func TestOAuthTokenTypeAvoidsCollision(t *testing.T) {
	m := NewMemory()
	m.UpsertOAuthToken(domain.OAuthToken{
		ClinicID: "c1", Provider: "meta", Type: "whatsapp",
		RefreshToken: "wa-token", PhoneNumberID: "1000",
	})
	m.UpsertOAuthToken(domain.OAuthToken{
		ClinicID: "c1", Provider: "meta", Type: "meta_ads",
		RefreshToken: "ads-token",
	})

	wa, ok := m.GetOAuthToken("c1", "whatsapp")
	if !ok || wa.RefreshToken != "wa-token" {
		t.Fatalf("whatsapp token overwritten or missing: %+v (ok=%v)", wa, ok)
	}
	ads, ok := m.GetOAuthToken("c1", "meta_ads")
	if !ok || ads.RefreshToken != "ads-token" {
		t.Fatalf("meta_ads token overwritten or missing: %+v (ok=%v)", ads, ok)
	}
}

// TestOAuthTokenBackwardCompatNoType: a token saved without Type (old rows, or
// Google Ads which only ever has one connection type per provider) still keys
// by Provider, unaffected by the Type-based scheme.
func TestOAuthTokenBackwardCompatNoType(t *testing.T) {
	m := NewMemory()
	m.UpsertOAuthToken(domain.OAuthToken{ClinicID: "c1", Provider: "google", RefreshToken: "g-token"})
	got, ok := m.GetOAuthToken("c1", "google")
	if !ok || got.RefreshToken != "g-token" {
		t.Fatalf("no-Type token should key by provider: %+v (ok=%v)", got, ok)
	}
	all := m.ListOAuthTokens("google")
	if len(all) != 1 {
		t.Fatalf("ListOAuthTokens(google) expected 1, got %d", len(all))
	}
}

// TestResolveClinicByPhoneNumberID is the per-clinic WhatsApp routing resolver:
// once a clinic connects a phone_number_id (via Embedded Signup), inbound
// webhooks for that number must resolve back to the right clinic — and an
// unclaimed number must not resolve to anything (falls back to marketplace
// routing, not a wrong clinic).
func TestResolveClinicByPhoneNumberID(t *testing.T) {
	m := NewMemory()
	m.UpsertOAuthToken(domain.OAuthToken{
		ClinicID: "umraniye", Provider: "meta", Type: "whatsapp", PhoneNumberID: "1000", RefreshToken: "tok",
	})
	m.UpsertOAuthToken(domain.OAuthToken{
		ClinicID: "nisantasi", Provider: "meta", Type: "whatsapp", PhoneNumberID: "2000", RefreshToken: "tok2",
	})

	if cid, ok := m.ResolveClinicByPhoneNumberID("1000"); !ok || cid != "umraniye" {
		t.Fatalf("expected umraniye for phone_number_id 1000, got %q (ok=%v)", cid, ok)
	}
	if cid, ok := m.ResolveClinicByPhoneNumberID("2000"); !ok || cid != "nisantasi" {
		t.Fatalf("expected nisantasi for phone_number_id 2000, got %q (ok=%v)", cid, ok)
	}
	if _, ok := m.ResolveClinicByPhoneNumberID("9999"); ok {
		t.Fatal("unclaimed phone_number_id should not resolve to any clinic")
	}
	if _, ok := m.ResolveClinicByPhoneNumberID(""); ok {
		t.Fatal("empty phone_number_id should never resolve")
	}
}
