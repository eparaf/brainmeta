# Spec — Senaryo Motoru (`internal/scenario`)

## Amaç
Bir klinik için **para harcamadan**, "bu bütçeyle ayda kaç randevu beklerim?" sorusuna
istatistiksel bir aralıkla cevap veren simülasyon katmanı. Keyword Planner'dan gelen
(gelecekte) arama hacmi / CPC / rekabet verisini, koddaki funnel prior'larıyla birleştirip
**Monte Carlo** ile 1000 kez oynatır ve **pesimist / gerçekçi / optimist** çıktı verir.

Bu, satış konuşmasında "size ayda 30–45 nitelikli randevu getiririz" cümlesini
**sayıyla savunmayı** ve garanti (SLA) satmadan önce fizibiliteyi test etmeyi sağlar.

## Sistemdeki yeri
- Karar motorları (`scoring/budget/matching/noshow/sla`) **canlı hot-path**tir; bu motor
  onların yanında duran **çevrimdışı/planlama** aracıdır. Beynin posteriorlarını bozmaz.
- Deterministik matematik — **LLM devrede değil**. API key gerekmez.
- Tüm sayısal temel `internal/priors`'tan gelir (magic number yok).

## Girdi — `CampaignPlan`
| Alan | Açıklama |
|------|----------|
| `ClinicID` | ölçekleme/yetki için (opsiyonel) |
| `Segment` | `aesthetic/implant/ortho/general` — funnel'ı seçer |
| `Platform` | `meta/google` — CPC/CTR prior'ını seçer |
| `Audience` | `local_tr/tourism` — CPC'yi domine eder |
| `MonthlyBudget` | TRY, aylık reklam bütçesi |
| `Keywords` | opsiyonel; boşsa `KeywordSource`'tan çekilir |
| `Runs` | Monte Carlo tekrarı (default 1000) |
| `Seed` | determinizm için |

## Veri kaynağı — `KeywordSource` (interface)
```
Keywords(seg, aud) → []KeywordMetrics{ Keyword, MonthlySearches, CompetitionIndex, CPCLowTRY, CPCHighTRY }
```
- **Faz A (şimdi):** `PriorKeywordSource` — `internal/priors`'tan tohumlanan sentetik,
  açıkça "cold-start placeholder" etiketli veri. Kredi gerektirmez.
- **Faz B (test hesabı gelince):** `internal/googleads` içinde Keyword Planner
  (`KeywordPlanIdeaService`) aynı interface'i uygular; motor değişmeden gerçek veriyle beslenir.
  Bu, repodaki `datasource` interface deseninin (gerçek/fake takas) aynısıdır.

## Model (tek Monte Carlo turu)
Funnel: **impression → click → lead(temas) → qualified → booked randevu → showed**

Her turda dağılımdan örnek çek (seeded `mathx.SampleBeta`, üçgen CPC):
1. `cpc` ← keyword CPC aralığından üçgen dağılım.
2. `ctr` ← Beta(mean=platform CTR prior, κ).
3. Bütçe-kısıtlı tıklama: `clicksBudget = budget / cpc`.
   Arama-hacmi tavanı: `clicksSearch = Σsearches · ctr · maxImpressionShare`.
   `clicks = min(clicksBudget, clicksSearch)` — hem bütçe hem Keyword Planner hacmi anlamlı.
4. `clickToLead` ← Beta (kaynak [CONV], healthcare PPC lead conv %2.4–11).
5. `qualify/book/show` ← Beta(`priors.FunnelFor(seg)`, κ).
6. `bookedAppts = clicks · clickToLead · qualify · book`
   `keptAppts   = bookedAppts · show`

## Çıktı — `Result`
Her metrik için **Band{P10, P50, P90, Mean}**:
- `BookedAppointments` (ana metrik — "ayda X–Y randevu" = P10–P90)
- `KeptAppointments` (gösterim sonrası)
- `QualifiedLeads`, `Clicks`
- `CostPerAppointment`, `CostPerLead` (TRY)
- `AvgCPC`, `SearchVolume`, ve şeffaflık için kullanılan `Assumptions` (prior'lar).

**İsimlendirme:** pesimist = P10, gerçekçi = P50, optimist = P90 (randevu için).
Maliyet için tersi (P90 = pahalı = pesimist); JSON ham percentile döner, etiketleme UI'da.

## Determinizm & test (repo kuralı 10)
- Seeded RNG; aynı `Seed` → aynı sonuç.
- Bilinen-sonuç testleri:
  - Determinizm (iki koşu eşit).
  - Monotonluk: bütçe ↑ → beklenen randevu ↑.
  - Bant sıralaması: P10 ≤ P50 ≤ P90.
  - Segment: yüksek-funnel `general` > düşük-funnel `aesthetic` (eşit bütçe, aynı lead sayısı başına randevu oranı).
  - CPC ↑ → randevu ↓.

## Yüzeyler
- `POST /v1/scenario` — clinic-scoped; `CampaignPlan` alır, `Result` döner.
- `brain scenario` CLI — panel olmadan hızlı demo raporu.

## Kapsam dışı (Faz B)
- Gerçek Keyword Planner çağrısı (test MCC + developer token gelince).
- Test hesabında kampanya oluştur/oku (gerçek harcama yok).
- Panelde senaryo kartı (nextjs-web).
