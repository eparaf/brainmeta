# BrainMeta — Durum Raporu & Yol Haritası (2026-07-03)

## 1. Tamamlanmışlık — "%90" nüansı
Karar **çekirdeği** ~%90; **üretime hazır ürün** ~%65-70.

| Alan | Durum | % |
|------|-------|---|
| Karar motorları (scoring/budget/matching/noshow/sla) | Prod kalite, testli | ~95 |
| Senaryo motoru (Faz A) | Bitti | 100 |
| Entegrasyon kodları (WhatsApp/Meta/GoogleAds) | Gerçek HTTP, çalışıyor | ~85 |
| LLM ajan | Gemini/Claude gerçek, tool-calling text'te yok | ~75 |
| Persistence | Docker/prod path zaten Postgres kullanıyordu; artık gerçek DB'ye karşı test edilmiş + CI'de doğrulanıyor | ~90 |
| Auth/Güvenlik | Prod guard'lar + register privilege-escalation kapandı; cross-tenant scoping 8 testle kanıtlandı | ~80 |
| Canlı outcome loop | Otomatik PMS yok (vendor-spesifik, genelleştirilemez); manuel raporlama yolu (`/v1/outcomes`) dedup+scoping bug'ları düzeltilerek güvenli hale getirildi | ~60 |
| Onboarding (Meta Embedded Signup OAuth) | Store hazır, akış yok | ~40 |
| Frontend | nextjs-web canlı; 3 kopya var | ~70 |

## 2. Güvenlik açıkları
**P0:**
1. Secret sızıntısı (public repo): `brain.env` + `nextjs-web/.env.local` → GitHub PAT, Gemini key, AUTH_SECRET, META_APP_SECRET, GOOGLE_CLIENT_SECRET. → hepsini rotate. (İkisi de takipten çıkarıldı; `.env.local` commit'i push onayı bekliyor.)
2. Dev-open default: `BRAIN_REQUIRE_AUTH` kapalı → `/v1/*` açık, `CanAccessClinic(nil)=true` → herkes tüm klinik verisini görür.
3. Default admin `admin@disci.local / admin1234` seed'leniyor.

**Yüksek:** JWT secret yoksa rastgele · WhatsApp webhook imzası META_APP_SECRET boşsa doğrulanmıyor · logout revocation yok · `internal/api` clinic-scoping enforcement testi yok.

## 3. Açık kaynak keşfi — öğrenilenler
**Genel bulgu:** bandit+water-filling+PID ile gerçek bütçe yazan + SLA chance-constraint + Poisson-binom overbooking'i TEK beyin altında birleştiren OSS **yok**. Bileşenler tek tek var; entegrasyon + "nitelikli randevu garantisi" bizim moat.

- **Bütçe/bandit:** `fidelity/mabwiser` (politika-agnostik + simülasyon harness), `sony/ABA` (non-stationarity, change detection, aylık episode, GP-UCB baseline). Ders: **offline log-replay zorunlu**, **discounting/sliding-window** ekle, budgeted-MAB = reward/cost (bizde CPL — hizalı), **gecikmeli reward** posterior'u dönüşüm geldikçe güncelle.
- **No-show:** `TimKong21/...No-Show-Prediction` (uçtan uca, lead-time en güçlü sinyal), Featuretools, LightGBM örnekleri. Ders: **olasılık kalibrasyonu (Platt/isotonic)** — Poisson-binom overbooking'in doğruluğu buna bağlı; OSS'ler bunu atlıyor.
- **WhatsApp ajan:** `mjunaidca/appointment-agent` (diş kliniği, LangGraph tool-calling), `santifer/jacobo-workflows` (2 yıl prod). Ders: **LLM ajanı Python/LangGraph, tool'lar Go backend'e RPC**; conversation memory tenant-izole Postgres.
- **Multi-tenant + connection store:** **Chatwoot** en değerli referans — polimorfik provider adapter (Meta/Twilio/360Dialog aynı arayüz), credential **JSONB + encrypted at rest**, her şey tenant'a scope'lu. AWS SaaS örnekleri: tenant context **asla client'tan değil, doğrulanmış JWT'den**.
- **Embedded Signup tuzağı (kritik):** WABA-level `subscribed_apps` → o WABA'daki tüm numaralar **tek webhook + verify token paylaşır**; ikinci numarayı farklı token ile onboard etmek birincisini sessizce bozar. Her klinik = ayrı WABA/numara olacağı için birinci sınıf risk. + lifecycle event **retry/idempotency** şart, token **audit trail**.

## 4. To-do (önceliklendirilmiş)
**P0 — güvenlik:**
- [ ] Sızan tüm token'ları rotate et *(kullanıcı)*
- [ ] Secret temizlik commit'ini push et *(onay)*
- [ ] Prod auth enforcement + güvenli default'lar

**P1 — üretime hazırlık:**
- [x] API auth/clinic-scoping testleri — `internal/api/scoping_test.go`: 8 cross-tenant testi
  (leads/conversations/appointments/doctors/services/connections/widget). Klinik-A kullanıcısı
  klinik-B'nin verisini ne liste ne tekil GET'te ne de mutasyonda göremiyor/değiştiremiyor
  (403, sızıntı yok), admin her yere erişiyor, dev-open (auth kapalıyken) tam erişim tasarım
  gereği doğrulandı. Hiçbir handler'da açık bulunmadı — mevcut kod zaten doğru scoped'muş.
- [x] Postgres persistence — **düzeltme: default'u DEĞİŞTİRMEDİK** (CLAUDE.md kural #9
  ile çelişirdi). Gerçek durum: `Dockerfile`/`docker-compose.yml` zaten `-tags "pgx redis"`
  ile build edip Postgres'i prod'da varsayılan yapıyordu — bu daha önce zaten çözülmüştü.
  Asıl eksik: `postgres.go` **hiç test edilmemişti** (CI sadece derliyordu, SQL'i hiç
  çalıştırmıyordu). Kapatıldı: `internal/store/postgres_test.go` (9 entegrasyon testi,
  `-tags pgx`, `DATABASE_URL` yoksa/erişilemezse zarifçe skip eder) + CI'ye gerçek
  Postgres service container eklendi. Docker açıp izole bir test konteynerinde (port 5433)
  9/9 test + restart-sonrası kalıcılık (`serve` iki kez başlatıldı, admin reseed edilmedi,
  login DB'den çalıştı) canlı doğrulandı.
- [x] Faz B: Google Ads test hesabı kampanya CRUD + Keyword Planner — kod hazır, canlı test için gerçek test hesabı (test manager, normal hesap değil) gerekiyor
- [x] Canlı outcome/feedback loop — **gerçek bir generic PMS entegrasyonu YOK ve olamaz**
  (her klinik farklı pratik-yönetim yazılımı kullanıyor; sahte/mock bir PMS flywheel'i
  uydurma veriyle eğitir, otomasyonsuzluktan daha kötü). Bunun yerine gerçek bir bug
  düzeltildi: `/v1/outcomes` (`handleOutcome`) `engine.IngestOutcome`'un dedup korumasını
  ATLAYIP doğrudan `Loop.Ingest` çağırıyordu (tekrarlanan POST modelleri çift eğitirdi) VE
  hiç clinic-scoping kontrolü yoktu (herhangi bir kullanıcı başka klinik için outcome
  enjekte edebilirdi). İkisi de düzeltildi + testlerle kanıtlandı (dedup: ilk POST
  fresh=true, tekrar fresh=false; cross-tenant: 403). `serve` başlangıcına PMS yoksa
  manuel raporlama gerektiğini belirten net bir log satırı eklendi.
- [x] No-show olasılık kalibrasyonu (overbooking doğruluğu) — Platt scaling, identity başlangıç, feedback'e bağlı
- [x] Bandit offline-replay harness + non-stationarity — `internal/budget` sliding-window z-test change detection; `internal/sim` drift + bandit-vs-manuel karşılaştırma; `brain compare` CLI.
  **Önemli bulgu:** 20 seed ortalamasında mevcut ayarla bandit ROAS'ı manuel eşit-bölmeyi YENMİYOR (~-2 ila -3%). Tek seed yön değiştirebiliyor (seed=7: -30%). Bu harness'ın amacı tam olarak bunu canlıya çıkmadan yakalamak.
  **Kök neden bulundu ve KISMEN düzeltildi (#23, tamamlandı):** `budget.Allocate()` klinik başına GÜNDE TEK bir Thompson (Beta) örneği çekip greedy "kazanan hepsini alır" su doldurma yapıyordu (`budget.go`) — şanssız bir örnek günün %100'ünü yanlış arm'a kaydırabiliyordu. Düzeltme: `avgSampleBeta` — N=20 Beta örneğinin ortalaması, tek örnek yerine (varyansı ~√N azaltır, Thompson keşfini korur). Sonuç: 20-seed ortalama açık **%-2.2 → %-0.9**'a (üçte birinden azına), en kötü tek-seed (seed=7) **%-30.1 → %-8.5**'e düştü. Test edilen ama İYİLEŞTİRMEYEN ikincil hipotezler: N=60 (daha yüksek N monoton iyileşme vermedi — RNG akış kayması), pseudo-count 8→3 (daha KÖTÜLEŞTİRDİ, %-4.6). pseudo=8 + N=20'de bırakıldı. `TestAvgSampleBetaReducesVariance` varyans azalmasını doğruluyor. **Açık tam kapanmadı** (%-0.9 hâlâ negatif) — kalan fark muhtemelen kısa ufuk (20-30 gün) + düşük arm sayısı (klinik başı 2) nedeniyle bandit'in keşif maliyetini henüz tam amorti edememesi; daha fazla tuning yerine mevcut durumla bırakıldı (overfitting riski).

**P2/P3:**
- [ ] Meta Embedded Signup onboarding (Chatwoot kalıbı; per-phone webhook tuzağına dikkat)
- [ ] Meta Lead Ads webhook'u tamamla (leadgen fetch)
- [ ] LLM tool-calling'i text-agent'a bağla (appointment-agent referans)
- [ ] Frontend tekilleştirme (front/ arşivle)
- [ ] Panel senaryo/fizibilite kartı

## 5. Önerilen sıra
P0 güvenlik (rotate + push + auth enforcement) → Faz B (Google Ads) → P1 test+persistence → no-show kalibrasyon + bandit replay → onboarding/agent.
