package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// LeadAdsClient fetches the full lead (name, phone, custom fields) for a Meta
// Lead Ads webhook's leadgen_id via the Graph API. The webhook payload itself
// carries only the id — Meta requires a follow-up GET to retrieve the answers,
// using a token with the `leads_retrieval` permission (typically a Page access
// token; falls back to the ad-account token if that's all that's configured).
type LeadAdsClient struct {
	Token        string
	GraphVersion string
	HTTP         *http.Client
}

func NewLeadAdsClient(token string) *LeadAdsClient {
	return &LeadAdsClient{Token: token, GraphVersion: "v21.0", HTTP: &http.Client{Timeout: 15 * time.Second}}
}

func (c *LeadAdsClient) ver() string {
	if c.GraphVersion == "" {
		return "v21.0"
	}
	return c.GraphVersion
}

func (c *LeadAdsClient) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

// Lead is the normalised result of a Graph API leadgen fetch.
type Lead struct {
	ID        string
	CreatedAt time.Time
	AdID      string
	FormID    string
	Name      string
	Phone     string
}

// FetchLead calls GET /{leadgen_id} and extracts name/phone from field_data.
// Lead ad forms let clinics customise field labels, but Meta's standard
// "full name" / "phone number" question types always report under these keys.
func (c *LeadAdsClient) FetchLead(ctx context.Context, leadgenID string) (Lead, error) {
	u := fmt.Sprintf("https://graph.facebook.com/%s/%s", c.ver(), leadgenID)
	q := url.Values{}
	q.Set("access_token", c.Token)
	q.Set("fields", "id,created_time,ad_id,form_id,field_data")

	var resp struct {
		ID          string `json:"id"`
		CreatedTime string `json:"created_time"`
		AdID        string `json:"ad_id"`
		FormID      string `json:"form_id"`
		FieldData   []struct {
			Name   string   `json:"name"`
			Values []string `json:"values"`
		} `json:"field_data"`
	}
	if err := c.getJSON(ctx, u+"?"+q.Encode(), &resp); err != nil {
		return Lead{}, err
	}
	l := Lead{ID: resp.ID, AdID: resp.AdID, FormID: resp.FormID}
	if t, err := time.Parse(time.RFC3339, resp.CreatedTime); err == nil {
		l.CreatedAt = t
	}
	for _, f := range resp.FieldData {
		if len(f.Values) == 0 {
			continue
		}
		switch f.Name {
		case "full_name", "name", "first_name":
			if l.Name == "" {
				l.Name = f.Values[0]
			}
		case "phone_number", "phone":
			l.Phone = f.Values[0]
		}
	}
	return l, nil
}

func (c *LeadAdsClient) getJSON(ctx context.Context, u string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("meta leadgen fetch %s: status %d: %s", u, resp.StatusCode, string(data))
	}
	return json.Unmarshal(data, out)
}
