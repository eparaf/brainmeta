package googleads

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"disci/brain/internal/domain"
	"disci/brain/internal/priors"
	"disci/brain/internal/scenario"
)

// Geo/language defaults for Keyword Planner lookups. Turkey + Turkish; override on
// the LiveKeywordSource for other markets. (Google's fixed constant ids.)
const (
	GeoTurkey         = "geoTargetConstants/2792"
	LanguageTurkish   = "languageConstants/1037"
	networkGoogleSrch = "GOOGLE_SEARCH"
)

// mutateResp is the shape of a *:mutate response — one resource name per op.
type mutateResp struct {
	Results []struct {
		ResourceName string `json:"resourceName"`
	} `json:"results"`
}

// GenerateKeywordIdeas calls Keyword Planner (KeywordPlanIdeaService) for real
// search volume / competition / top-of-page CPC on the given seed terms, mapped
// into scenario.KeywordMetrics (TRY). This is the LIVE data that replaces the
// cold-start PriorKeywordSource in the scenario engine. Read-only, no spend.
func (c *Client) GenerateKeywordIdeas(ctx context.Context, seeds []string, geo, lang string) ([]scenario.KeywordMetrics, error) {
	if len(seeds) == 0 {
		return nil, fmt.Errorf("generateKeywordIdeas: no seed keywords")
	}
	if geo == "" {
		geo = GeoTurkey
	}
	if lang == "" {
		lang = LanguageTurkish
	}
	payload := map[string]any{
		"keywordSeed":        map[string]any{"keywords": seeds},
		"geoTargetConstants": []string{geo},
		"language":           lang,
		"keywordPlanNetwork": networkGoogleSrch,
	}
	var resp struct {
		Results []struct {
			Text    string `json:"text"`
			Metrics struct {
				AvgMonthlySearches     string `json:"avgMonthlySearches"`
				CompetitionIndex       string `json:"competitionIndex"`
				LowTopOfPageBidMicros  string `json:"lowTopOfPageBidMicros"`
				HighTopOfPageBidMicros string `json:"highTopOfPageBidMicros"`
			} `json:"keywordIdeaMetrics"`
		} `json:"results"`
	}
	// Colon-on-customer form (not a sub-resource), so use postPath directly.
	if err := c.postPath(ctx, fmt.Sprintf("customers/%s:generateKeywordIdeas", c.CustomerID), payload, &resp); err != nil {
		return nil, err
	}
	out := make([]scenario.KeywordMetrics, 0, len(resp.Results))
	for _, r := range resp.Results {
		low := microsToTRY(r.Metrics.LowTopOfPageBidMicros)
		high := microsToTRY(r.Metrics.HighTopOfPageBidMicros)
		if high < low {
			high = low
		}
		out = append(out, scenario.KeywordMetrics{
			Keyword:          r.Text,
			MonthlySearches:  atoi(r.Metrics.AvgMonthlySearches),
			CompetitionIndex: atof(r.Metrics.CompetitionIndex) / 100.0, // 0..100 → 0..1
			CPCLowTRY:        low,
			CPCHighTRY:       high,
		})
	}
	return out, nil
}

// LiveKeywordSource adapts a Google Ads client to scenario.KeywordSource so the
// scenario engine forecasts on REAL Keyword Planner data. Seed terms per segment
// come from scenario.SeedKeywords (same terms the cold-start source uses).
type LiveKeywordSource struct {
	Client   *Client
	Geo      string // optional; defaults to Turkey
	Language string // optional; defaults to Turkish
}

// Keywords implements scenario.KeywordSource.
func (l LiveKeywordSource) Keywords(seg domain.Segment, _ priors.Audience) ([]scenario.KeywordMetrics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return l.Client.GenerateKeywordIdeas(ctx, scenario.SeedKeywords(seg), l.Geo, l.Language)
}

// Campaign is a read view of a Google Ads campaign.
type Campaign struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Status         string  `json:"status"`
	DailyBudgetTRY float64 `json:"dailyBudgetTRY"`
}

// CreateSearchCampaign creates a PAUSED Search campaign with its own daily budget
// in the (test) account and returns the campaign resource name. PAUSED + a test
// account means it never serves and never spends — safe to run end-to-end. It is a
// two-step mutate: create the budget, then the campaign that references it.
func (c *Client) CreateSearchCampaign(ctx context.Context, name string, dailyBudgetTRY float64) (string, error) {
	if name == "" {
		return "", fmt.Errorf("createCampaign: empty name")
	}
	if dailyBudgetTRY <= 0 {
		dailyBudgetTRY = 1
	}
	// 1) Budget. Names must be unique per account, so suffix with a timestamp.
	budgetName := fmt.Sprintf("%s Budget %d", name, time.Now().Unix())
	var bResp mutateResp
	err := c.post(ctx, "campaignBudgets:mutate", map[string]any{
		"operations": []any{map[string]any{"create": map[string]any{
			"name":             budgetName,
			"amountMicros":     strconv.FormatInt(int64(dailyBudgetTRY*1e6), 10),
			"deliveryMethod":   "STANDARD",
			"explicitlyShared": false,
		}}},
	}, &bResp)
	if err != nil {
		return "", fmt.Errorf("create budget: %w", err)
	}
	if len(bResp.Results) == 0 {
		return "", fmt.Errorf("create budget: no resource returned")
	}
	budgetRes := bResp.Results[0].ResourceName

	// 2) Campaign referencing that budget. PAUSED, Search-only, manual CPC.
	var cResp mutateResp
	err = c.post(ctx, "campaigns:mutate", map[string]any{
		"operations": []any{map[string]any{"create": map[string]any{
			"name":                   name,
			"status":                 "PAUSED",
			"advertisingChannelType": "SEARCH",
			"campaignBudget":         budgetRes,
			"manualCpc":              map[string]any{},
			"networkSettings": map[string]any{
				"targetGoogleSearch":         true,
				"targetSearchNetwork":        false,
				"targetContentNetwork":       false,
				"targetPartnerSearchNetwork": false,
			},
		}}},
	}, &cResp)
	if err != nil {
		return "", fmt.Errorf("create campaign: %w", err)
	}
	if len(cResp.Results) == 0 {
		return "", fmt.Errorf("create campaign: no resource returned")
	}
	return cResp.Results[0].ResourceName, nil
}

// ListCampaigns reads the account's campaigns (id, name, status, daily budget).
func (c *Client) ListCampaigns(ctx context.Context) ([]Campaign, error) {
	rows, err := c.search(ctx,
		`SELECT campaign.id, campaign.name, campaign.status, campaign_budget.amount_micros FROM campaign ORDER BY campaign.id`)
	if err != nil {
		return nil, err
	}
	out := make([]Campaign, 0, len(rows))
	for _, r := range rows {
		out = append(out, Campaign{
			ID:             r.Campaign.ID,
			Name:           r.Campaign.Name,
			Status:         r.Campaign.Status,
			DailyBudgetTRY: microsToTRY(r.CampaignBudget.AmountMicros),
		})
	}
	return out, nil
}

func microsToTRY(s string) float64 {
	if s == "" {
		return 0
	}
	m, _ := strconv.ParseFloat(s, 64)
	return m / 1e6
}

func atof(s string) float64 { f, _ := strconv.ParseFloat(s, 64); return f }
