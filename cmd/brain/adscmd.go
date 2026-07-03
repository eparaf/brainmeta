package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"disci/brain/internal/domain"
	"disci/brain/internal/googleads"
	"disci/brain/internal/scenario"
)

// runGoogleAdsTest exercises the LIVE Google Ads test-account flow end-to-end with
// ZERO spend: discover the customer, pull real Keyword Planner data, create a
// PAUSED search campaign, and read the campaigns back. Needs the OAuth refresh
// token (see `brain google-oauth`) + developer token in brain.env. Because the
// account is a Google Ads *test* account and the campaign is PAUSED, nothing ever
// serves or costs money.
func runGoogleAdsTest() {
	loadEnvFile()
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	secret := os.Getenv("GOOGLE_CLIENT_SECRET")
	refresh := os.Getenv("GOOGLE_ADS_REFRESH_TOKEN")
	devTok := os.Getenv("GOOGLE_ADS_DEVELOPER_TOKEN")
	loginCust := os.Getenv("GOOGLE_ADS_LOGIN_CUSTOMER_ID")
	customerID := os.Getenv("GOOGLE_ADS_CUSTOMER_ID")

	if clientID == "" || secret == "" || devTok == "" {
		log.Fatal("google-ads-test: GOOGLE_CLIENT_ID/SECRET + GOOGLE_ADS_DEVELOPER_TOKEN gerekli (brain.env)")
	}
	if refresh == "" {
		log.Fatal("google-ads-test: GOOGLE_ADS_REFRESH_TOKEN yok — önce `go run ./cmd/brain google-oauth` çalıştır")
	}

	c := googleads.New(clientID, secret, refresh, devTok, customerID, loginCust)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 1) Erişilebilir hesaplar (customer id verilmediyse ilkini kullan).
	fmt.Println("=== 1) Erişilebilir müşteri hesapları ===")
	ids, err := c.ListAccessibleCustomers(ctx)
	if err != nil {
		log.Fatalf("listAccessibleCustomers: %v", err)
	}
	fmt.Printf("   %v\n", ids)
	if c.CustomerID == "" {
		if len(ids) == 0 {
			log.Fatal("google-ads-test: hesap bulunamadı; test hesabı customer id'sini GOOGLE_ADS_CUSTOMER_ID'ye yaz")
		}
		c.CustomerID = ids[0]
		fmt.Printf("   (customer id otomatik seçildi: %s)\n", c.CustomerID)
	}

	// 2) Gerçek Keyword Planner verisi.
	fmt.Println("=== 2) Keyword Planner (gerçek arama hacmi / CPC) ===")
	kws, err := c.GenerateKeywordIdeas(ctx, scenario.SeedKeywords(domain.SegmentImplant), "", "")
	if err != nil {
		log.Fatalf("generateKeywordIdeas: %v", err)
	}
	for i, k := range kws {
		if i >= 8 {
			break
		}
		fmt.Printf("   %-28s aramalar/ay=%-7d CPC %.1f–%.1f TRY  rekabet=%.2f\n",
			k.Keyword, k.MonthlySearches, k.CPCLowTRY, k.CPCHighTRY, k.CompetitionIndex)
	}

	// 3) PAUSED kampanya oluştur (sıfır harcama).
	name := fmt.Sprintf("BrainMeta Test %d", time.Now().Unix())
	fmt.Printf("=== 3) PAUSED kampanya oluştur: %q (100 TRY/gün, yayınlanmaz) ===\n", name)
	res, err := c.CreateSearchCampaign(ctx, name, 100)
	if err != nil {
		log.Fatalf("createCampaign: %v", err)
	}
	fmt.Printf("   oluşturuldu: %s\n", res)

	// 4) Kampanyaları oku.
	fmt.Println("=== 4) Hesaptaki kampanyalar ===")
	camps, err := c.ListCampaigns(ctx)
	if err != nil {
		log.Fatalf("listCampaigns: %v", err)
	}
	for _, cm := range camps {
		fmt.Printf("   [%s] %-28s %s  %.0f TRY/gün\n", cm.ID, cm.Name, cm.Status, cm.DailyBudgetTRY)
	}
	fmt.Println("\nBitti — hiçbir reklam yayınlanmadı, hiç para harcanmadı (test hesabı + PAUSED).")
}
