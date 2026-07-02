package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// runGoogleOAuth performs the one-time Google OAuth "installed app" (loopback)
// flow to capture a refresh token for the Google Ads API. It reads
// GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET from brain.env, opens a local callback
// server, prints the consent URL for the user to approve once, exchanges the
// returned code for a refresh token, and prints it (marked REFRESH_TOKEN=...).
//
// Desktop-app OAuth clients allow loopback redirects (http://127.0.0.1:PORT)
// without pre-registration, so no redirect URI needs configuring in Cloud Console.
func runGoogleOAuth() {
	loadEnvFile()
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		log.Fatal("google-oauth: set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET in brain.env first")
	}

	// Bind a loopback listener on a fixed port so the redirect URI is stable.
	const port = 8765
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("google-oauth: cannot bind %s: %v (is it already in use?)", addr, err)
	}
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/", port)

	authURL := "https://accounts.google.com/o/oauth2/v2/auth?" + url.Values{
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"scope":         {"https://www.googleapis.com/auth/adwords"},
		"access_type":   {"offline"},
		"prompt":        {"consent"}, // force a refresh_token even on re-consent
	}.Encode()

	fmt.Println("=== GOOGLE OAUTH ===")
	fmt.Println("Aşağıdaki URL'yi tarayıcıda aç ve izin ver (okan6226@gmail.com ile):")
	fmt.Println()
	fmt.Println(authURL)
	fmt.Println()
	fmt.Println("İzin verince bu pencere otomatik tamamlanır...")

	codeCh := make(chan string, 1)
	errCh := make(chan string, 1)
	srv := &http.Server{}
	http.DefaultServeMux = http.NewServeMux() // isolate from any other handlers
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if e := q.Get("error"); e != "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, "<h2>Hata: %s</h2><p>Bu pencereyi kapatabilirsiniz.</p>", e)
			errCh <- e
			return
		}
		code := q.Get("code")
		if code == "" {
			return // ignore favicon etc.
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<h2>Tamamlandı ✔</h2><p>Bu pencereyi kapatıp terminale dönebilirsiniz.</p>")
		codeCh <- code
	})
	go func() { _ = srv.Serve(ln) }()

	var code string
	select {
	case code = <-codeCh:
	case e := <-errCh:
		log.Fatalf("google-oauth: consent failed: %s", e)
	case <-time.After(5 * time.Minute):
		log.Fatal("google-oauth: timed out waiting for consent (5m)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)

	// Exchange the authorization code for tokens.
	resp, err := http.PostForm("https://oauth2.googleapis.com/token", url.Values{
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		log.Fatalf("google-oauth: token exchange: %v", err)
	}
	defer resp.Body.Close()
	var tok struct {
		RefreshToken string `json:"refresh_token"`
		AccessToken  string `json:"access_token"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		log.Fatalf("google-oauth: decode token response: %v", err)
	}
	if tok.Error != "" {
		log.Fatalf("google-oauth: %s — %s", tok.Error, tok.ErrorDesc)
	}
	if tok.RefreshToken == "" {
		log.Fatal("google-oauth: no refresh_token returned (revoke prior grant and retry; prompt=consent should force one)")
	}

	fmt.Println()
	fmt.Println("BAŞARILI. Refresh token (bunu brain.env'e GOOGLE_ADS_REFRESH_TOKEN olarak yaz):")
	fmt.Println("REFRESH_TOKEN=" + strings.TrimSpace(tok.RefreshToken))
}
