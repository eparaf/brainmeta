// Package googleads is the REAL Google Ads API integration: pull ad spend/CPL,
// push the brain's daily budgets back to campaigns, and upload offline
// conversions so Google's optimiser compounds with ours. It implements
// datasource.AdPlatform.
//
// No third-party deps — it speaks the Google Ads REST API over net/http, exactly
// like internal/meta talks to the Graph API. Credentials: an OAuth refresh token
// (per clinic, captured by the panel's OAuth flow) + an account-level developer
// token + the target customer id. Arms are matched by AD GROUP NAME (name your ad
// groups to match the brain's arm ids, same convention as Meta's ad sets).
package googleads

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"disci/brain/internal/datasource"
)

const apiVersion = "v18"

// Client talks to the Google Ads REST API on behalf of ONE clinic/customer.
type Client struct {
	ClientID       string // OAuth app client id   (GOOGLE_CLIENT_ID)
	ClientSecret   string // OAuth app client secret(GOOGLE_CLIENT_SECRET)
	RefreshToken   string // clinic's refresh token (captured at OAuth consent)
	DeveloperToken string // account-level dev token(GOOGLE_ADS_DEVELOPER_TOKEN)
	CustomerID     string // target customer id, digits only, no dashes
	LoginCustomer  string // MCC/manager id for login-customer-id header (optional)
	ConvAction     string // conversion action resource name (optional, for uploads)
	HTTP           *http.Client

	mu          sync.Mutex
	accessTok   string
	accessTokAt time.Time // when the cached access token expires
}

// New builds a client. CustomerID/LoginCustomer are normalised to digits only.
func New(clientID, clientSecret, refreshToken, devToken, customerID, loginCustomer string) *Client {
	return &Client{
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		RefreshToken:   refreshToken,
		DeveloperToken: devToken,
		CustomerID:     digits(customerID),
		LoginCustomer:  digits(loginCustomer),
		HTTP:           &http.Client{Timeout: 30 * time.Second},
	}
}

func digits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// accessToken returns a valid OAuth access token, refreshing (and caching) it via
// the refresh-token grant when the cached one is missing or near expiry.
func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.accessTok != "" && time.Now().Before(c.accessTokAt) {
		return c.accessTok, nil
	}
	form := url.Values{}
	form.Set("client_id", c.ClientID)
	form.Set("client_secret", c.ClientSecret)
	form.Set("refresh_token", c.RefreshToken)
	form.Set("grant_type", "refresh_token")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token refresh %d: %s", resp.StatusCode, string(body))
	}
	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tok); err != nil {
		return "", err
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("token refresh: empty access_token")
	}
	c.accessTok = tok.AccessToken
	// Refresh a minute early to avoid races at the boundary.
	c.accessTokAt = time.Now().Add(time.Duration(tok.ExpiresIn-60) * time.Second)
	return c.accessTok, nil
}

func (c *Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

// post issues an authenticated POST to a customer-scoped endpoint and decodes the
// JSON body into out.
func (c *Client) post(ctx context.Context, endpoint string, payload, out any) error {
	tok, err := c.accessToken(ctx)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&buf).Encode(payload); err != nil {
			return err
		}
	}
	u := fmt.Sprintf("https://googleads.googleapis.com/%s/customers/%s/%s", apiVersion, c.CustomerID, endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("developer-token", c.DeveloperToken)
	req.Header.Set("Content-Type", "application/json")
	if c.LoginCustomer != "" {
		req.Header.Set("login-customer-id", c.LoginCustomer)
	}
	resp, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("google ads %s %d: %s", endpoint, resp.StatusCode, truncate(string(body), 600))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(body, out)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// --- datasource.AdPlatform ------------------------------------------------

// gaqlResult is one row of a searchStream response (only the fields we read).
type gaqlResult struct {
	Campaign struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"campaign"`
	AdGroup struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"adGroup"`
	CampaignBudget struct {
		ResourceName string `json:"resourceName"`
	} `json:"campaignBudget"`
	Metrics struct {
		CostMicros  string  `json:"costMicros"`
		Impressions string  `json:"impressions"`
		Clicks      string  `json:"clicks"`
		Conversions float64 `json:"conversions"`
	} `json:"metrics"`
}

// search runs a GAQL query via searchStream and returns the flattened rows.
func (c *Client) search(ctx context.Context, query string) ([]gaqlResult, error) {
	var stream []struct {
		Results []gaqlResult `json:"results"`
	}
	if err := c.post(ctx, "googleAds:searchStream", map[string]string{"query": query}, &stream); err != nil {
		return nil, err
	}
	var out []gaqlResult
	for _, chunk := range stream {
		out = append(out, chunk.Results...)
	}
	return out, nil
}

// PullSpend reports realised spend/CPL per arm. The AD GROUP NAME is used as the
// arm id (name your ad groups to match the brain's arms). cost_micros is TRY×1e6.
func (c *Client) PullSpend(ctx context.Context, since time.Time) ([]datasource.ArmSpend, error) {
	days := int(time.Since(since).Hours()/24) + 1
	if days < 1 {
		days = 1
	}
	q := fmt.Sprintf(`SELECT ad_group.id, ad_group.name, metrics.cost_micros, `+
		`metrics.impressions, metrics.clicks, metrics.conversions `+
		`FROM ad_group WHERE segments.date DURING LAST_%d_DAYS`, clampDays(days))
	rows, err := c.search(ctx, q)
	if err != nil {
		return nil, err
	}
	// Aggregate per arm (ad group name) in case the date range spans multiple rows.
	agg := map[string]*datasource.ArmSpend{}
	for _, r := range rows {
		arm := r.AdGroup.Name
		if arm == "" {
			continue
		}
		s := agg[arm]
		if s == nil {
			s = &datasource.ArmSpend{ArmID: arm}
			agg[arm] = s
		}
		costMicros, _ := strconv.ParseFloat(r.Metrics.CostMicros, 64)
		s.Spend += costMicros / 1e6
		s.Impressions += atoi(r.Metrics.Impressions)
		s.Clicks += atoi(r.Metrics.Clicks)
		s.Leads += int(r.Metrics.Conversions + 0.5)
	}
	out := make([]datasource.ArmSpend, 0, len(agg))
	for _, s := range agg {
		if s.Leads > 0 {
			s.CostPerLead = s.Spend / float64(s.Leads)
		}
		out = append(out, *s)
	}
	return out, nil
}

// clampDays maps an arbitrary day count to a GAQL LAST_N_DAYS bucket Google
// supports (7, 14, or 30); anything else falls back to the nearest larger bucket.
func clampDays(d int) int {
	switch {
	case d <= 7:
		return 7
	case d <= 14:
		return 14
	default:
		return 30
	}
}

// SetDailyBudgets pushes the brain's per-arm allocation back to Google. Arms are
// ad groups; their spend is governed by the parent campaign's shared budget, so we
// roll each arm's allocation up to its campaign budget and mutate amount_micros.
func (c *Client) SetDailyBudgets(ctx context.Context, perArm map[string]float64) error {
	if len(perArm) == 0 {
		return nil
	}
	// Map ad-group name → campaign budget resource, and sum the arm allocations
	// that share a campaign budget.
	rows, err := c.search(ctx,
		`SELECT ad_group.name, campaign_budget.resource_name FROM ad_group`)
	if err != nil {
		return err
	}
	budgetTotal := map[string]float64{} // budget resource → summed TRY/day
	for _, r := range rows {
		alloc, ok := perArm[r.AdGroup.Name]
		if !ok || r.CampaignBudget.ResourceName == "" {
			continue
		}
		budgetTotal[r.CampaignBudget.ResourceName] += alloc
	}
	if len(budgetTotal) == 0 {
		return nil
	}
	type op struct {
		Update     map[string]any `json:"update"`
		UpdateMask string         `json:"updateMask"`
	}
	ops := make([]op, 0, len(budgetTotal))
	for res, try := range budgetTotal {
		ops = append(ops, op{
			Update: map[string]any{
				"resourceName": res,
				"amountMicros": strconv.FormatInt(int64(try*1e6), 10),
			},
			UpdateMask: "amount_micros",
		})
	}
	return c.post(ctx, "campaignBudgets:mutate", map[string]any{"operations": ops}, nil)
}

// UploadConversions sends realised qualified-appointment / closed events back to
// Google's optimiser via offline click-conversion upload (matched by gclid). A
// configured conversion action resource name is required; without it this no-ops.
func (c *Client) UploadConversions(ctx context.Context, convs []datasource.Conversion) error {
	if c.ConvAction == "" || len(convs) == 0 {
		return nil
	}
	type clickConv struct {
		GclID              string  `json:"gclid"`
		ConversionAction   string  `json:"conversionAction"`
		ConversionDateTime string  `json:"conversionDateTime"`
		ConversionValue    float64 `json:"conversionValue"`
		CurrencyCode       string  `json:"currencyCode"`
	}
	out := make([]clickConv, 0, len(convs))
	for _, cv := range convs {
		if cv.ExternalID == "" { // need a gclid to attribute
			continue
		}
		out = append(out, clickConv{
			GclID:            cv.ExternalID,
			ConversionAction: c.ConvAction,
			// Google wants "yyyy-mm-dd hh:mm:ss+00:00".
			ConversionDateTime: cv.At.UTC().Format("2006-01-02 15:04:05-07:00"),
			ConversionValue:    cv.Value,
			CurrencyCode:       "TRY",
		})
	}
	if len(out) == 0 {
		return nil
	}
	return c.post(ctx, ":uploadClickConversions", map[string]any{
		"conversions":    out,
		"partialFailure": true,
		"validateOnly":   false,
	}, nil)
}

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

// ListAccessibleCustomers returns the customer ids the refresh token can reach,
// digits only. Used to auto-discover the customer id when one isn't configured.
func (c *Client) ListAccessibleCustomers(ctx context.Context) ([]string, error) {
	tok, err := c.accessToken(ctx)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("https://googleads.googleapis.com/%s/customers:listAccessibleCustomers", apiVersion)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("developer-token", c.DeveloperToken)
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("listAccessibleCustomers %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	var r struct {
		ResourceNames []string `json:"resourceNames"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(r.ResourceNames))
	for _, rn := range r.ResourceNames {
		out = append(out, digits(rn)) // "customers/123" → "123"
	}
	return out, nil
}
