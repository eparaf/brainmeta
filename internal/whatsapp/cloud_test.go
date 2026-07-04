package whatsapp

import "testing"

// TestParseInboundExtractsPhoneNumberID confirms the RECEIVING business
// number's phone_number_id (value.metadata.phone_number_id in the real Cloud
// API payload shape) is parsed — this is what lets the API layer resolve which
// clinic an inbound message belongs to (see api.handleWAInbound).
func TestParseInboundExtractsPhoneNumberID(t *testing.T) {
	body := []byte(`{
		"entry": [{
			"changes": [{
				"value": {
					"metadata": {"phone_number_id": "123456789"},
					"contacts": [{"profile": {"name": "Ayşe"}, "wa_id": "905551234567"}],
					"messages": [{"from": "905551234567", "type": "text", "text": {"body": "Merhaba"}}]
				}
			}]
		}]
	}`)
	ins, err := ParseInbound(body)
	if err != nil {
		t.Fatalf("ParseInbound: %v", err)
	}
	if len(ins) != 1 {
		t.Fatalf("expected 1 inbound message, got %d", len(ins))
	}
	if ins[0].PhoneNumberID != "123456789" {
		t.Fatalf("expected phone_number_id 123456789, got %q", ins[0].PhoneNumberID)
	}
	if ins[0].From != "905551234567" || ins[0].Name != "Ayşe" || ins[0].Text != "Merhaba" {
		t.Fatalf("other fields regressed: %+v", ins[0])
	}
}

// TestParseInboundMissingMetadataIsFine: older/malformed payloads without
// metadata must not crash — PhoneNumberID just comes back empty (caller falls
// back to unscoped routing).
func TestParseInboundMissingMetadataIsFine(t *testing.T) {
	body := []byte(`{"entry":[{"changes":[{"value":{"messages":[{"from":"1","type":"text","text":{"body":"hi"}}]}}]}]}`)
	ins, err := ParseInbound(body)
	if err != nil {
		t.Fatalf("ParseInbound: %v", err)
	}
	if len(ins) != 1 || ins[0].PhoneNumberID != "" {
		t.Fatalf("expected empty PhoneNumberID, got %+v", ins)
	}
}
