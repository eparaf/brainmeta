# BrainMeta — Durum Raporu & Yol Haritası (2026-07-03)

## 1. Tamamlanmışlık — "%90" nüansı
Karar **çekirdeği** ~%90; **üretime hazır ürün** ~%65-70.

| Alan | Durum | % |
|------|-------|---|
| Karar motorları (scoring/budget/matching/noshow/sla) | Prod kalite, testli | ~95 |
| Senaryo motoru (Faz A) | Bitti | 100 |
| Entegrasyon kodları (WhatsApp/Meta/GoogleAds) | Gerçek HTTP, çalışıyor | ~85 |
| LLM ajan | Gemini/Claude gerçek, tool-calling text'te yok | ~75 |
| Persistence | Postgres var ama default değil (restart'ta veri uçar) | ~65 |
| Auth/Güvenlik | İyi primitive, ama dev-open default + sızıntı | ~55 |
| Canlı outcome/PMS loop | Sadece interface, wire edilmemiş | ~25 |
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
- [ ] API auth/clinic-scoping testleri
- [ ] Postgres'i default persistence yap + store CRUD testleri
- [ ] Faz B: Google Ads test hesabı kampanya CRUD + Keyword Planner
- [ ] Canlı outcome/feedback loop wire
- [x] No-show olasılık kalibrasyonu (overbooking doğruluğu) — Platt scaling, identity başlangıç, feedback'e bağlı
- [x] Bandit offline-replay harness + non-stationarity — `internal/budget` sliding-window z-test change detection; `internal/sim` drift + bandit-vs-manuel karşılaştırma; `brain compare` CLI.
  **Önemli bulgu:** 20 seed ortalamasında mevcut ayarla bandit ROAS'ı manuel eşit-bölmeyi YENMİYOR (~-2 ila -3%). Tek seed yön değiştirebiliyor (seed=7: -30%). Bu harness'ın amacı tam olarak bunu canlıya çıkmadan yakalamak.
  **Kök neden (analiz tamam, kod düzeltmesi #23'te):** `budget.Allocate()` klinik başına GÜNDE TEK bir Thompson (Beta) örneği çekip sonuca göre greedy "kazanan hepsini alır" su doldurma yapıyor (`budget.go:328-365`) — o günün TÜM bütçesi tek gürültülü örneğe göre dağıtılıyor. Şanssız bir örnek günün %100'ünü yanlış arm'a kaydırabiliyor; `EqualSplitAllocator` bu varyanstan tamamen bağışık (hep %50/50). İkincil katkı: segment-bazlı (platform-körü) prior'lar gerçek θ'lerden sistematik yüksek + pseudo-count=8 gerçek veriyi yavaş domine ediyor, kısa ufukta (20-30 gün) yanlış-tahsis penceresini büyütüyor. Küçük katkı: `qRemaining` tavanı dolunca kalan bütçe hiç harcanmıyor (EqualSplit hep tam harcıyor) — bu config'de nadiren tetikleniyor ama gerçek asimetri.

**P2/P3:**
- [ ] Meta Embedded Signup onboarding (Chatwoot kalıbı; per-phone webhook tuzağına dikkat)
- [ ] Meta Lead Ads webhook'u tamamla (leadgen fetch)
- [ ] LLM tool-calling'i text-agent'a bağla (appointment-agent referans)
- [ ] Frontend tekilleştirme (front/ arşivle)
- [ ] Panel senaryo/fizibilite kartı

## 5. Önerilen sıra
P0 güvenlik (rotate + push + auth enforcement) → Faz B (Google Ads) → P1 test+persistence → no-show kalibrasyon + bandit replay → onboarding/agent.
