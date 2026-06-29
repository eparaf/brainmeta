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
- **internal/api** — HTTP (CORS + gömülü konsol + webhooks). **cmd/brain** — sim | chat | serve.

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

## Çalıştırma
```
go run ./cmd/brain sim     # 30 günlük simülasyon (beyni kanıtlar)
go run ./cmd/brain chat    # mock ajan demo
go run ./cmd/brain serve   # API + gömülü konsol :8080
go test ./...              # tüm testler
```
UI: `cd ui && npm install && npm run dev` (Vite+React, BACKEND_URL ile backend'e bağlanır).

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

## Yol haritası (sıradaki)
Per-clinic **Connection store** (env yerine UI'dan klinik-bazlı Meta/WhatsApp bağlama) +
Meta Embedded Signup OAuth. Beyin multi-tenant olduğu için bu bir resolver + tablo + onboarding işi.
