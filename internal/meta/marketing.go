// Package meta is the real Meta Marketing API integration: pull ad spend/CPL,
// push the brain's daily budgets back to ad sets, and upload conversions so
// Meta's optimiser compounds with ours. It implements datasource.AdPlatform.
package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"disci/brain/internal/datasource"
)

// Marketing talks to the Graph API. Configure with a long-lived access token,
// the ad account id (without the "act_" prefix), and a pixel id for the
// Conversions API.
type Marketing struct {
	Token        string
	AdAccountID  string
	PixelID      string
	GraphVersion string
	HTTP         *http.Client
}

func NewMarketing(token, adAccountID, pixelID string) *Marketing {
	return &Marketing{Token: token, AdAccountID: adAccountID, PixelID: pixelID,
		GraphVersion: "v21.0", HTTP: &http.Client{Timeout: 20 * time.Second}}
}

func (m *Marketing) ver() string {
	if m.GraphVersion == "" {
		return "v21.0"
	}
	return m.GraphVersion
}

// PullSpend reads ad-set level insights and maps them to per-arm spend/CPL. The
// ad-set id is used as the arm id (name your ad sets to match the brain's arms).
func (m *Marketing) PullSpend(ctx context.Context, since time.Time) ([]datasource.ArmSpend, error) {
	u := fmt.Sprintf("https://graph.facebook.com/%s/act_%s/insights", m.ver(), m.AdAccountID)
	q := url.Values{}
	q.Set("level", "adset")
	q.Set("fields", "adset_id,adset_name,spend,actions")
	q.Set("date_preset", "last_7d")
	q.Set("access_token", m.Token)
	var resp struct {
		Data []struct {
			AdsetID string `json:"adset_id"`
			Spend   string `json:"spend"`
			Actions []struct {
				ActionType string `json:"action_type"`
				Value      string `json:"value"`
			} `json:"actions"`
		} `json:"data"`
	}
	if err := m.getJSON(ctx, u+"?"+q.Encode(), &resp); err != nil {
		return nil, err
	}
	out := make([]datasource.ArmSpend, 0, len(resp.Data))
	for _, d := range resp.Data {
		spend, _ := strconv.ParseFloat(d.Spend, 64)
		leads := 0
		for _, a := range d.Actions {
			if a.ActionType == "lead" || a.ActionType == "onsite_conversion.messaging_first_reply" {
				n, _ := strconv.Atoi(a.Value)
				leads += n
			}
		}
		cpl := 0.0
		if leads > 0 {
			cpl = spend / float64(leads)
		}
		out = append(out, datasource.ArmSpend{ArmID: d.AdsetID, Spend: spend, Leads: leads, CostPerLead: cpl})
	}
	return out, nil
}

// SetDailyBudgets writes each arm's daily budget to its ad set. Budget is in the
// account currency's minor units (kuruş for TRY), so we multiply by 100.
func (m *Marketing) SetDailyBudgets(ctx context.Context, perArm map[string]float64) error {
	for adsetID, tl := range perArm {
		if tl <= 0 {
			continue
		}
		u := fmt.Sprintf("https://graph.facebook.com/%s/%s", m.ver(), adsetID)
		form := url.Values{}
		form.Set("daily_budget", strconv.Itoa(int(tl*100)))
		form.Set("access_token", m.Token)
		if err := m.postForm(ctx, u, form); err != nil {
			return fmt.Errorf("set budget %s: %w", adsetID, err)
		}
	}
	return nil
}

// UploadConversions sends offline/closed-deal events via the Conversions API.
func (m *Marketing) UploadConversions(ctx context.Context, convs []datasource.Conversion) error {
	if m.PixelID == "" || len(convs) == 0 {
		return nil
	}
	data := make([]map[string]any, 0, len(convs))
	for _, c := range convs {
		data = append(data, map[string]any{
			"event_name":    "Purchase",
			"event_time":    c.At.Unix(),
			"action_source": "business_messaging",
			"custom_data":   map[string]any{"value": c.Value, "currency": "TRY", "content_name": c.EventName},
		})
	}
	body, _ := json.Marshal(map[string]any{"data": data, "access_token": m.Token})
	u := fmt.Sprintf("https://graph.facebook.com/%s/%s/events", m.ver(), m.PixelID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return m.do(req, nil)
}

func (m *Marketing) getJSON(ctx context.Context, u string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	return m.do(req, out)
}
func (m *Marketing) postForm(ctx context.Context, u string, form url.Values) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return m.do(req, nil)
}
func (m *Marketing) do(req *http.Request, out any) error {
	resp, err := m.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("meta: status %d: %s", resp.StatusCode, string(data))
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

// compile-time check.
var _ datasource.AdPlatform = (*Marketing)(nil)
