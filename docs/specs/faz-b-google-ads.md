# Faz B — Google Ads test hesabı entegrasyonu (kaldığımız yer)

Bu doküman, senaryo motoru (Faz A) bittikten sonra Google Ads **test hesabı** ile
gerçek kampanya oluştur/oku akışına geçmek için kalan adımları tutar. **Test hesabı
= sıfır harcama** (Google test hesapları asla reklam yayınlamaz).

## Durum (2026-07-03)

**Bitti (Faz A):**
- `internal/scenario` — Monte-Carlo randevu tahmini (pesimist/gerçekçi/optimist).
  Kredisiz, `PriorKeywordSource` ile çalışır. Testler yeşil.
- `POST /v1/scenario` endpoint + `brain scenario` CLI demo.
- `brain google-oauth` — refresh token üreten tek seferlik OAuth (loopback) komutu.

**Toplanan Google Ads kimlikleri (hepsi `brain.env`'de, burada DEĞİL):**
- `GOOGLE_ADS_DEVELOPER_TOKEN` — test_disci manager hesabı, "Test Account" erişimli
- `GOOGLE_ADS_LOGIN_CUSTOMER_ID` — manager (MCC) id, tiresiz
- `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` — Cloud Console OAuth client

## Kalan adımlar

### 1. OAuth `redirect_uri_mismatch` düzeltmesi
OAuth client "Web uygulaması" tipinde oluşturulmuş; loopback adresini kayıtlı ister.
- Cloud Console → Kimlik bilgileri → OAuth client → **Yetkili yönlendirme URI'leri**
  → ekle: `http://127.0.0.1:8765/` (+ istersen `http://localhost:8765/`) → Kaydet
- **Alternatif:** client'ı "Masaüstü uygulaması" tipinde yeniden oluştur (o zaman
  hiçbir redirect kaydı gerekmez).

### 2. Refresh token üret
```
go run ./cmd/brain google-oauth
```
Çıkan linki `okan6226@gmail.com` ile aç → izin ver → komut `REFRESH_TOKEN=...` basar.
Bunu `brain.env`'e **`GOOGLE_ADS_REFRESH_TOKEN=`** olarak yaz.

### 3. Test hesabı oluştur (kampanya CRUD için hedef)
- Manager (test_disci) içinde → Hesaplar → yeni **test hesabı** oluştur.
- 10 haneli **customer id**'sini not al (kampanyalar burada kurulacak; sıfır harcama).

### 4. Kampanya oluştur/oku kodu (Faz B — YAPILDI, canlı run token bekliyor)
`internal/googleads/campaigns.go` eklendi (fake HTTP transport ile test edildi):
- **`GenerateKeywordIdeas`** (`KeywordPlanIdeaService`) → `scenario.KeywordMetrics` (TRY).
- **`LiveKeywordSource`** — `scenario.KeywordSource`'u uygular; `/v1/scenario` creds
  varken gerçek arama hacmi/CPC kullanır (main.go'da wire edildi).
- **`CreateSearchCampaign`** (PAUSED, Search, kendi bütçesi) + **`ListCampaigns`** —
  test hesabında, sıfır harcama.
- **CLI:** `go run ./cmd/brain google-ads-test` → erişilebilir hesap → keyword çek →
  PAUSED kampanya oluştur → oku. (Adım 2'deki refresh token gelince çalışır.)
- Web-site seed keyword'ü sonraya bırakıldı (segment default ile başlandı).

### 5. Panel kartı (opsiyonel)
`nextjs-web`'de "Senaryo / Fizibilite" sayfası → `/v1/scenario`'ya bağlan, X–Y
randevu bandını göster.

## Çalıştırma
```
go run ./cmd/brain scenario        # senaryo demosu (kredisiz)
go run ./cmd/brain google-oauth    # refresh token üret (Adım 2)
go test ./...                      # hepsi yeşil kalmalı
```

## ⚠️ GÜVENLİK — acil
`brain.env` bir ara **public repoya commit edilmiş** (GitHub token + Gemini key
sızdı). Bu commit'te `brain.env` takipten çıkarıldı ama **git geçmişi kalıcıdır**.
Yapılması gerekenler:
- **GitHub PAT'i iptal et** (Settings → Developer settings → Tokens) ve yenile.
- **Gemini API key'i iptal et** (Google AI Studio) ve yenile.
- Google OAuth `client_secret` ve developer token'ı da riskliyse **sıfırla**
  (Cloud Console / Ads API Center → "Jetonu sıfırla").
- İdeali: geçmişi temizlemek için repoyu **private** yap veya history purge (BFG).
