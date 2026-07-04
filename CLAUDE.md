# CLAUDE.md — BrainMeta (dişçi randevu & reklam optimizasyon beyni)

Bu dosya, kod üzerinde çalışırken (özellikle AI asistanların) uyması gereken
mimari ve dikkat noktalarını özetler. Kısa tut, doğru tut.

## Ne bu?
Diş kliniklerine **garantili nitelikli randevu** sağlayan multi-tenant SaaS beyni.
Reklam → WhatsApp AI ajanı → niteleme → beyin karar → randevu → hatırlatma →
öğrenme (flywheel). Go backend + (ayrı) Next.js panel.

## Mimari (tek bakışta)
- **internal/domain** — çekirdek tipler (Clinic, Lead, Appointment, Arm, Outcome). Depo/JSON-agnostik.
- **internal/mathx** — sampling (Beta/Gamma), sigmoid, PoissonCDF. Bağımsız, deterministik.
- **internal/scoring** — Motor 1: lead EV (lojistik + Bayesian soğuk başlangıç).
- **internal/budget** — Motor 2: Thompson sampling + water-filling + PID pacing + per-clinic + değer-ağırlıklı.
- **internal/matching** — Motor 3: Hungarian atama (doğru hasta → doğru klinik).
- **internal/noshow** — Motor 4: gösterim tahmini + EXACT Poisson-binom overbooking.
- **internal/sla** — garanti kontrolcüsü: chance-constraint, gölge fiyat λ (booking gate + routing'i besler).
- **internal/engine** — orkestratör (HandleLead, PlanBudget). Tüm motorları + feedback'i bağlar.
- **internal/agent** — WhatsApp AI ajanı + **tools.go = agent'ın eylem yüzeyi** (beyin
  tool'ları: get_availability/book_appointment/escalate; LLM bunlarla eyleme geçer).
- **internal/voice** — sesli çağrı kanalı. İKİ yol:
  (1) **BEDAVA**: tarayıcı Web Speech API (Türkçe STT+TTS) → `/voice` sayfası → `/v1/whatsapp`
      (mevcut ajan). Telefoni/creds/maliyet YOK. Her zaman açık.
  (2) **ÜCRETLİ (PSTN)**: Twilio Voice + TwiML (`twilio.go`) — turn-based; `VOICE_PUBLIC_URL`
      + Twilio numara/creds ile açılır. Tool-loop, booking-durumu konuşması DETERMİNİSTİK.
  Gerçek-zamanlı (Gemini Live/OpenAI Realtime + media streaming) premium UX için sonraki yükseltme.
- **internal/feedback** — flywheel: her sonucu tüm modellere yayar.
- **internal/persist** — öğrenilen state snapshot'ı (restart'ta moat uçmaz).
- **internal/store** — store.Store arayüzü; Memory (default) + Postgres (StorePG, -tags pgx).
- **internal/session** — konuşma state'i arayüzü; Memory + Redis (-tags redis).
- **internal/whatsapp** — Meta-onaylı şablonlar + Cloud API client (gönder/webhook).
- **internal/meta** — Meta Marketing API (harcama/bütçe/Conversions).
- **internal/consent / auth** — KVKK opt-out + /v1 API-key.
- **internal/datasource** — dış dünya arayüzleri (LeadSource/AdPlatform/PMS/Messenger) + SyncService.
- **internal/reminder** — 24s/2s hatırlatma scheduler.
- **internal/api** — HTTP (CORS + gömülü konsol + webhooks). **cmd/brain** — sim | chat | serve |
  scenario | google-oauth | google-ads-test | compare.
- **internal/scenario** — çevrimdışı Monte-Carlo senaryo motoru: bütçe→randevu tahmini
  (pesimist/gerçekçi/optimist, P10/P50/P90). `KeywordSource` seam: kredisiz `PriorKeywordSource`
  (cold-start) ↔ canlı `googleads.LiveKeywordSource` (Keyword Planner). `POST /v1/scenario`.
- **internal/googleads/campaigns.go** — Faz B: `GenerateKeywordIdeas` (Keyword Planner, read-only),
  `CreateSearchCampaign` (PAUSED, sıfır harcama) + `ListCampaigns`. `brain google-ads-test` CLI ile
  uçtan uca doğrulanır (test hesabı gerekir — bkz. docs/specs/faz-b-google-ads.md).

## DİKKAT EDİLECEKLER (bunları bozma)
1. **LLM önerir, BEYİN karar verir.** Ajan; randevu durumunu (booked/iptal/slot)
   ASLA serbest metinle iddia etmez — bu cümleler deterministik şablon (`replyBooked`/
   `replyAlreadyBooked`). LLM yalnız niteleme + sıradaki soru. (Halüsinasyon randevu = yasak.)
2. **Niteleme oturumda BİRİKİR** (`mergeQual`), her turda sıfırlanmaz. Spesifik segment
   sonradan "general" ile ezilmez.
3. **Overbooking kovaları RANDEVU GÜNÜNE göre** (booking gününe değil). `reserveSlot`
   exact Poisson-binom riskini doğru popülasyona uygular. Atomik (bookMu).
4. **Outcome çift-sayımı yok:** `engine.IngestOutcome` `OutcomeKey` ile dedup eder; dedup
   seti + feature store'lar SNAPSHOT'a girer (restart'ta tekrar yedirilmez).
5. **Bütçe per-clinic (passthrough):** A kliniğinin parası B'ye gitmez. SLA gölge fiyatı
   bütçeyi DEĞİL, lead-routing + booking-gate'i etkiler.
6. **Öğrenen beyin TEK-YAZICI.** Yatay ölçek: konuşma front-end'i Redis ile çoğaltılır;
   beyni dikey ölçekle ya da **klinik-bazlı shard'la**. Aynı kliniğin posterioruna iki yazıcı koşturma.
7. **Para birimi TRY.** Tüm prior'lar `internal/priors`'ta kaynaklı (2025-2026). Magic number ekleme.
8. **Sırlar `brain.env`'de** (.gitignore'da, ASLA commit etme). API anahtarları, tokenlar.
9. **Default build BAĞIMSIZ** (third-party yok). Postgres/Redis kodu build-tag arkasında
   (`-tags pgx`, `-tags redis`); default'u kırma.
10. **Testler deterministik** (seeded RNG). Yeni motor → yeni test. `go test ./...` yeşil kalsın.
11. **Commit mesajları:** Claude/AI/araç ibaresi YOK. Co-author trailer ekleme. Bunun
    yerine mesajın sonuna kısa bir **`Yavuz notu: ...`** satırı düş.
12. **Auth prod-guard:** `BRAIN_REQUIRE_AUTH=true` iken `BRAIN_JWT_SECRET` boşsa VEYA admin
    parolası hâlâ default (`admin1234`) ise `serve` **başlamayı reddeder** (cmd/brain/main.go).
    `/v1/auth/register` her modda **admin-only** (dev-open dahil) — açık bırakma, privilege-
    escalation deliği açar.
13. **`budget.Allocate` sıralı iterasyon şart.** Klinik/arm map'leri ID'ye göre sort edilmeden
    range'lenirse (Go map sırası rastgele), paylaşılan Thompson-sampling RNG'si farklı sırada
    tüketilir → aynı state'te iki çağrı FARKLI sonuç verir. Bunu bozma; `internal/sim`'deki
    offline-replay/karşılaştırma harness'i buna bağımlı.
14. **No-show `PShow` KALİBRE olasılıktır** (Platt scaling, `internal/noshow`). Ham lojistik
    çıktısını DEĞİL, `PShow()`'u kullan — Poisson-binom overbooking'in doğruluğu buna bağlı.
    Yeni predictor ekleyeceksen aynı deseni (identity başlangıç + online kalibrasyon) uygula.
15. **`brain compare` bir GERÇEK bulgu taşıyor, gizleme.** 20-seed ortalamasında mevcut ayarla
    bandit (Thompson+water-filling) ROAS'ta naif eşit-bölmeyi YENMİYOR (~%-2/-3, tek seed'de
    %-30'a kadar). Kök neden araştırması sürüyor (bkz. docs/DURUM-RAPORU.md). Bu motoru
    "kanıtlanmış üstün" varsayma — `brain compare` ile doğrulamadan production tuning değiştirme.
16. **`OAuthToken` anahtarı `Type`'a göre, `Provider`'a göre DEĞİL.** whatsapp+meta_ads ikisi de
    Provider="meta" — Type'ı unutup Provider'a dönersen iki bağlantı türü birbirini eziyor (bkz.
    `internal/store/store.go`'daki `oauthTokenKey`).

## Çalıştırma
```
go run ./cmd/brain sim              # 30 günlük simülasyon (beyni kanıtlar)
go run ./cmd/brain chat             # mock ajan demo
go run ./cmd/brain serve            # API + gömülü konsol :8080
go run ./cmd/brain scenario         # bütçe→randevu Monte-Carlo demo (kredisiz)
go run ./cmd/brain google-oauth     # Google Ads refresh token üret (bir kez login)
go run ./cmd/brain google-ads-test  # test hesabında keyword+kampanya uçtan uca (sıfır harcama)
go run ./cmd/brain compare          # bandit vs manuel ROAS — canlıya çıkmadan doğrulama
go test ./...                       # tüm testler
DATABASE_URL=... go test -tags pgx ./internal/store/...  # Postgres entegrasyon testleri (gerçek DB)
```
UI: `cd ui && npm install && npm run dev` (Vite+React, BACKEND_URL ile backend'e bağlanır).
Panel: `cd nextjs-web && npm run dev` (Next.js, :3002). Takvim modülü: `/calendar` (slug İngilizce).

## Geliştirme notu (AI asistan)
- **Her düzenlemede `eslint` çalıştırma** (yavaş). Frontend için doğrulama olarak yalnızca
  `npx tsc --noEmit` kullan; lint'i ancak iş bitiminde bir kez (veya istenirse) çalıştır.

## Build matrisi
```
go build ./...                          # default (memory, bağımsız)
go build -tags pgx ./cmd/brain          # + Postgres (go get github.com/jackc/pgx/v5)
go build -tags redis ./cmd/brain        # + Redis   (go get github.com/redis/go-redis/v9)
go build -tags "pgx redis" ./cmd/brain  # production
```

## Env (brain.env)
`GEMINI_API_KEY`/`ANTHROPIC_API_KEY` (ajan), `WHATSAPP_TOKEN`+`WHATSAPP_PHONE_NUMBER_ID`+
`WHATSAPP_VERIFY_TOKEN`, `META_TOKEN`+`META_AD_ACCOUNT_ID`+`META_PIXEL_ID`, `BRAIN_API_KEY`,
`DATABASE_URL`, `REDIS_URL`, `BRAIN_ADDR`, `BRAIN_SNAPSHOT`. Boşsa o özellik mock/kapalı.
Google Ads (Faz B): `GOOGLE_CLIENT_ID`+`GOOGLE_CLIENT_SECRET` (OAuth client, Desktop app
tipi önerilir — Web tipinde `http://127.0.0.1:8765/` redirect'i eklenmeli),
`GOOGLE_ADS_DEVELOPER_TOKEN` (Test Account erişimli yeterli), `GOOGLE_ADS_REFRESH_TOKEN`
(`google-oauth` ile üretilir), `GOOGLE_ADS_LOGIN_CUSTOMER_ID` + `GOOGLE_ADS_CUSTOMER_ID`
(**gerçek bir TEST manager/client hesabı olmalı** — normal/production hesaplar
`CUSTOMER_NOT_ENABLED`/`PERMISSION_DENIED` döner; test manager: ads.google.com/nav/selectaccount?sf=mt).

## Durum & yol haritası
Güncel tamamlanmışlık + öncelikli to-do listesi: **`docs/DURUM-RAPORU.md`**. Tüm P0/P1/P2/P3
görevleri tamamlandı (14 madde). Özet:
- P0 auth sertleştirme (privilege-escalation kapatıldı, prod guard'lar) · senaryo motoru (Faz A+B)
  · no-show Platt kalibrasyonu · bandit change-detection + offline-replay harness + varyans
  azaltma (avgSampleBeta) · Postgres store gerçek-DB testleri · API clinic-scoping testleri (8
  cross-tenant) · outcome-loop dedup/scoping bug fix · Meta Lead Ads webhook (gerçek Graph API
  fetch + imza doğrulama) · voice kanalının gerçek LLM tool-calling'e bağlanması · `front/` arşivi
  · panelde senaryo/fizibilite kartı · **Meta Embedded Signup + per-clinic WhatsApp routing
  resolver** (`OAuthToken.Type` — whatsapp/meta_ads'in `provider="meta"` çakışma bug'ı düzeltildi;
  `ResolveClinicByPhoneNumberID` inbound webhook'u doğru kliniğe yönlendiriyor; frontend Embedded
  Signup JS SDK gerçek Meta App olmadan test edilemedi, koda dikkat).
16. **`OAuthToken` anahtarı `Type`'a göre, `Provider`'a göre DEĞİL.** whatsapp+meta_ads ikisi de
    Provider="meta" — Type'ı unutup Provider'a dönersen iki bağlantı türü birbirini eziyor (bkz.
    `internal/store/store.go`'daki `oauthTokenKey`).
