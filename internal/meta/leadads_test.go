package meta

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

// TestFetchLeadParsesFieldData confirms name/phone are pulled from the Graph
// API's field_data (which comes as a name/values array, not flat JSON), using
// the standard Lead Ads field names.
func TestFetchLeadParsesFieldData(t *testing.T) {
	body := `{
		"id": "lg123",
		"created_time": "2026-07-01T12:00:00+0000",
		"ad_id": "ad1",
		"form_id": "form1",
		"field_data": [
			{"name": "full_name", "values": ["Ayşe Yılmaz"]},
			{"name": "phone_number", "values": ["+905551234567"]},
			{"name": "some_custom_question", "values": ["ignored"]}
		]
	}`
	c := &LeadAdsClient{Token: "tok", HTTP: &http.Client{Transport: fakeRT{body: body}}}
	l, err := c.FetchLead(context.Background(), "lg123")
	if err != nil {
		t.Fatalf("FetchLead: %v", err)
	}
	if l.Name != "Ayşe Yılmaz" || l.Phone != "+905551234567" {
		t.Fatalf("field_data not parsed correctly: %+v", l)
	}
	if l.AdID != "ad1" || l.FormID != "form1" {
		t.Fatalf("ad/form ids not parsed: %+v", l)
	}
}

// TestFetchLeadHTTPError confirms a non-2xx Graph API response surfaces an error
// rather than silently returning a zero-value Lead.
func TestFetchLeadHTTPError(t *testing.T) {
	c := &LeadAdsClient{Token: "tok", HTTP: &http.Client{Transport: errRT{}}}
	if _, err := c.FetchLead(context.Background(), "lg123"); err == nil {
		t.Fatal("expected an error from a failing Graph API call")
	}
}

type errRT struct{}

func (errRT) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 400,
		Body:       io.NopCloser(strings.NewReader(`{"error":"bad token"}`)),
	}, nil
}
