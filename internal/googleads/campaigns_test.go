package googleads

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

// fakeTransport routes Google Ads / OAuth calls to canned responses and records
// the request bodies so tests can assert the exact payloads we send — no
// credentials, no network.
type fakeTransport struct {
	bodies  map[string]string // path-substring → last request body
	replies map[string]string // path-substring → response JSON
}

func (f *fakeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	body := ""
	if r.Body != nil {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
	}
	// OAuth token endpoint.
	if strings.Contains(r.URL.Host, "oauth2.googleapis.com") {
		return jsonResp(`{"access_token":"tok","expires_in":3600}`), nil
	}
	for key, reply := range f.replies {
		if strings.Contains(r.URL.Path, key) {
			if f.bodies == nil {
				f.bodies = map[string]string{}
			}
			f.bodies[key] = body
			return jsonResp(reply), nil
		}
	}
	return jsonResp(`{}`), nil
}

func jsonResp(s string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(s)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func testClient(ft *fakeTransport) *Client {
	c := New("cid", "secret", "refresh", "devtok", "1234567890", "6306436427")
	c.HTTP = &http.Client{Transport: ft}
	return c
}

func TestGenerateKeywordIdeasMapping(t *testing.T) {
	ft := &fakeTransport{replies: map[string]string{
		":generateKeywordIdeas": `{"results":[
			{"text":"implant fiyatları","keywordIdeaMetrics":{"avgMonthlySearches":"5400","competitionIndex":"85","lowTopOfPageBidMicros":"12000000","highTopOfPageBidMicros":"48000000"}},
			{"text":"diş implantı","keywordIdeaMetrics":{"avgMonthlySearches":"3200","competitionIndex":"60","lowTopOfPageBidMicros":"8000000","highTopOfPageBidMicros":"20000000"}}
		]}`,
	}}
	c := testClient(ft)
	kws, err := c.GenerateKeywordIdeas(context.Background(), []string{"implant fiyatları"}, "", "")
	if err != nil {
		t.Fatalf("GenerateKeywordIdeas: %v", err)
	}
	if len(kws) != 2 {
		t.Fatalf("want 2 keywords, got %d", len(kws))
	}
	k := kws[0]
	if k.Keyword != "implant fiyatları" || k.MonthlySearches != 5400 {
		t.Errorf("bad first keyword: %+v", k)
	}
	if k.CompetitionIndex != 0.85 { // 85/100
		t.Errorf("competition: want 0.85, got %v", k.CompetitionIndex)
	}
	if k.CPCLowTRY != 12.0 || k.CPCHighTRY != 48.0 { // micros → TRY
		t.Errorf("cpc: want 12/48, got %v/%v", k.CPCLowTRY, k.CPCHighTRY)
	}
	// The request must carry the Turkey/Turkish defaults + GOOGLE_SEARCH network.
	sent := ft.bodies[":generateKeywordIdeas"]
	for _, want := range []string{GeoTurkey, LanguageTurkish, networkGoogleSrch, "implant fiyatları"} {
		if !strings.Contains(sent, want) {
			t.Errorf("request missing %q; body=%s", want, sent)
		}
	}
}

func TestCreateSearchCampaignPayloads(t *testing.T) {
	ft := &fakeTransport{replies: map[string]string{
		"campaignBudgets:mutate": `{"results":[{"resourceName":"customers/1234567890/campaignBudgets/555"}]}`,
		"campaigns:mutate":       `{"results":[{"resourceName":"customers/1234567890/campaigns/999"}]}`,
	}}
	c := testClient(ft)
	res, err := c.CreateSearchCampaign(context.Background(), "İmplant Test", 250)
	if err != nil {
		t.Fatalf("CreateSearchCampaign: %v", err)
	}
	if res != "customers/1234567890/campaigns/999" {
		t.Fatalf("unexpected campaign resource: %s", res)
	}
	// Budget op: amount in micros, standard delivery.
	bBody := ft.bodies["campaignBudgets:mutate"]
	if !strings.Contains(bBody, `"amountMicros":"250000000"`) {
		t.Errorf("budget amountMicros wrong; body=%s", bBody)
	}
	// Campaign op: PAUSED, SEARCH, references the created budget → safe, no spend.
	cBody := ft.bodies["campaigns:mutate"]
	for _, want := range []string{`"status":"PAUSED"`, `"advertisingChannelType":"SEARCH"`, "customers/1234567890/campaignBudgets/555"} {
		if !strings.Contains(cBody, want) {
			t.Errorf("campaign payload missing %q; body=%s", want, cBody)
		}
	}
}

func TestListCampaignsParse(t *testing.T) {
	ft := &fakeTransport{replies: map[string]string{
		"googleAds:searchStream": `[{"results":[
			{"campaign":{"id":"999","name":"İmplant Test","status":"PAUSED"},"campaignBudget":{"amountMicros":"250000000"}}
		]}]`,
	}}
	c := testClient(ft)
	camps, err := c.ListCampaigns(context.Background())
	if err != nil {
		t.Fatalf("ListCampaigns: %v", err)
	}
	if len(camps) != 1 {
		t.Fatalf("want 1 campaign, got %d", len(camps))
	}
	got := camps[0]
	if got.ID != "999" || got.Name != "İmplant Test" || got.Status != "PAUSED" || got.DailyBudgetTRY != 250 {
		t.Errorf("bad campaign parse: %+v", got)
	}
}

// LiveKeywordSource satisfies scenario.KeywordSource and returns real metrics.
func TestLiveKeywordSourceImplementsInterface(t *testing.T) {
	ft := &fakeTransport{replies: map[string]string{
		":generateKeywordIdeas": `{"results":[{"text":"diş teli","keywordIdeaMetrics":{"avgMonthlySearches":"1000","competitionIndex":"40","lowTopOfPageBidMicros":"5000000","highTopOfPageBidMicros":"9000000"}}]}`,
	}}
	src := LiveKeywordSource{Client: testClient(ft)}
	kws, err := src.Keywords("ortho", "")
	if err != nil {
		t.Fatalf("Keywords: %v", err)
	}
	if len(kws) != 1 || kws[0].Keyword != "diş teli" {
		t.Fatalf("unexpected: %+v", kws)
	}
}
