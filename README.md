# Dişçi Beyni — Patient Acquisition & Guarantee Engine

A decision **brain** for a dental patient-acquisition business: it turns ad spend
into *qualified appointments*, routes the right patient to the right clinic,
keeps no-shows down, and honours a per-clinic monthly guarantee — learning the
whole time from real outcomes.

This is not a slide deck. It's a runnable, tested Go system. Run it:

```bash
go run ./cmd/brain sim     # 30-day end-to-end simulation + report
go run ./cmd/brain serve   # HTTP API for the production serving layer
go test ./...              # unit tests for every motor
```

The simulator invents a hidden reality (which ads truly work, who shows up) that
the brain cannot see, then proves the brain discovers it. Latest run: the brain
recovered every arm's hidden quality to within ~0.02, drove **~6x ROAS**, kept
**~62% show rate**, and pulled every clinic to **82–108% of its guarantee** —
flagging the one guarantee that's genuinely unachievable with current ad quality.

---

## 1. What the brain optimises

One constrained objective:

```
maximize  Σ_clinic Σ_lead  EV(lead, clinic)        (network margin)
s.t.      qualified_appts[k] ≥ N_k                 (1) the GUARANTEE  (SLA)
          ad_spend[k]        ≤ B_k                 (2) clinic budget
          appointments[k]    ≤ capacity_k          (3) clinic seats
```

Everything below is how we decompose and actually solve this online, with
incomplete information, in real time.

---

## 2. The four motors

| Motor | Package | Job | Core math |
|------|---------|-----|-----------|
| **1. Lead Value** | `internal/scoring` | EV of every lead | logistic funnel models + Bayesian cold-start priors |
| **2. Budget Allocation** | `internal/budget` | where ad money goes | Thompson sampling + water-filling + PID pacing |
| **3. Routing** | `internal/matching` | right patient → right clinic | Hungarian assignment under capacity |
| **4. Show-up** | `internal/noshow` | beat no-shows | logistic show model + overbooking (Poisson-binomial) |

Tied together by the **Guarantee Controller** (`internal/sla`) and the **feedback
loop** (`internal/feedback`).

### Motor 1 — Lead Value (`scoring`)

For each lead we estimate the revenue chain:

```
EV = P(qualify) · P(book|qualify) · P(show|book) · P(close|show) · ticket · margin
```

Each probability is an online logistic regression over the lead's features
(reply speed, engagement, distance, intent, stated budget, time-of-day, …).
**Cold start:** before we have data, each stage falls back to a per-segment Beta
prior seeded from dental-marketing benchmarks, blended out by a confidence
weight as real outcomes arrive. So the very first lead still gets a sane score.

### Motor 2 — Budget Allocation (`budget`) — *"some clinics advertise well, some don't"*

This is the answer to your core question, and **nothing here is hand-tuned**.

Each ad lever is an **arm** = (clinic, platform, campaign, creative, segment)
with an unknown qualify-rate θ and a learned cost-per-lead. Per cycle:

1. **Thompson sample** θ ~ Beta(α,β) for every arm — exploration ∝ uncertainty.
2. Compute each arm's sampled **appointments-per-TRY** = θ / CPL.
3. **Water-fill** the budget toward the most efficient arms until their marginal
   efficiency equalises — the Lagrangian optimum; **λ** is the shadow price of a
   TRY of budget.
4. **Capacity-aware cap:** never buy more qualified leads than a clinic can seat
   (leads beyond capacity are wasted money). The cap = `1.3 · seats / θ̂` leads.
5. **PID pacing** spreads daily spend so an arm doesn't blow its budget by noon.

A clinic that advertises well has high-θ, low-CPL arms → it wins budget
automatically. A poor one is throttled — unless its guarantee shadow price forces
a floor (below).

### Motor 3 — Routing (`matching`) — *"right patient to the right clinic"*

The marketplace vision. Leads land in a pool; the brain assigns each to a clinic:

```
maximize Σ x_ij · value(lead_i, clinic_j)
s.t. each lead → ≤1 clinic;  Σ_i x_ij ≤ capacity_j;  x_ij=0 if incompatible
```

Solved exactly with the **Hungarian (Kuhn–Munkres)** algorithm — clinics expand
into seats, value blends EV × close-rate × distance-decay × segment-fit ×
guarantee-priority.

### Motor 4 — Show-up (`noshow`)

A booked appointment isn't a kept one (dental no-show ≈ 20–30%). We:
- predict P(show) per appointment (logistic),
- pick an intervention (reminders → deposit → call) by risk × value,
- **overbook** correctly: accept appointments while P(arrivals > seats) stays
  under a risk budget, using a Poisson-binomial normal approximation. Because
  each patient shows with prob < 1, this safely books *more* than capacity.

### The Guarantee Controller (`sla`) — the SLA as a control problem

Each clinic's monthly guarantee is a chance constraint
`P(appts_month ≥ N) ≥ 90%`. The controller tracks each clinic's running deficit
vs a time-prorated target and converts it into one **shadow price λ_k ≥ 1**.
That single number is consumed by **three** motors:
- the **budget** motor (spend more on a lagging clinic's arms),
- the **routing** motor (send it more leads),
- the **booking gate** (book even low-ticket leads when the guarantee demands
  volume — this is what rescued the general-segment clinic in simulation).

Comfortably-ahead clinic → λ≈1 (no distortion). Dangerously-behind → large λ that
pulls resources toward it. Exactly how ad-delivery systems handle pacing SLAs.

### The flywheel (`feedback`)

Every outcome (reply / booking / show / close) updates every model. Data
compounds into the moat — competitors can copy the architecture, not the months
of accumulated posteriors.

---

## 3. Data & training strategy

> *On your question about public Turkish/European statistical data for training.*

> **These numbers are now in the code, with sources.** See `internal/priors/`:
> every funnel rate, CPL, ticket size and reminder-lift carries a 2025–2026
> citation (WordStream/LocaliQ ad benchmarks, First Page Sage / InfluxMD
> conversion data, Henry Schein One + Klara/MDPI no-show studies, Turkey dental-
> tourism price guides). The motors read from there — no magic numbers.

**Day-one (cold start): public priors, not trained models.** We don't need a
trained model to start — we need good priors. These come from publicly available
benchmarks and seed the Beta/logistic intercepts in `scoring` and `noshow`:

- **Funnel conversion benchmarks** — published dental/healthcare lead-gen
  conversion rates (lead→appointment→show→treatment) per treatment type.
- **Ad cost benchmarks** — Meta/Google CPL & CPC by vertical and geography
  (TR vs EU differ a lot — useful for the dental-tourism premium segment).
- **No-show literature** — peer-reviewed studies give baseline no-show rates and
  the lift from reminders/deposits (already encoded as `expectedLift`).
- **Treatment economics** — implant / veneer / ortho price ranges set `AvgTicket`.

These are *aggregate statistics*, not personal data — free to use, and they make
the system useful before our own data exists.

**Then: learn from first-party outcomes.** The moment real leads flow, the
feedback loop takes over and the public priors are blended out. Our own data
always beats generic benchmarks.

**Data-science methods your friend will actually use** (beyond what's coded here):
- **Uplift / causal modelling** — does an intervention *cause* a show, or would
  they have shown anyway? (Don't waste deposits on sure-shows.)
- **Survival analysis** — time-to-no-show / time-to-book hazard models.
- **Media-mix modelling (MMM) & geo-experiments** — attribute revenue across
  channels beyond last-click; run holdout regions to measure true incrementality.
- **CAC / LTV cohorts** — unit economics per segment and district.
- **Conversion-feedback to ad platforms** — send realised qualified-appointment
  events back to Meta/Google so *their* optimisers compound with ours.

**Compliance is not optional.** Any first-party patient data (the 6k-record
reactivation play, or anything identifiable) is subject to **KVKK** (and GDPR for
EU patients): lawful basis / consent, a data-processing agreement with each
clinic, opt-out on every message, and data minimisation. Public benchmarks sit
outside this; patient records sit squarely inside it.

---

## 4. Architecture (serving vs learning)

```
                 ┌───────────────────────── Go serving layer (this repo) ──────────────────────────┐
 WhatsApp/Meta → │  API → Engine ─┬─ Motor1 scoring ─┬─ Motor3 matching ── book ── Motor4 no-show   │
   webhook       │                ├─ Motor2 budget   └─ SLA controller                              │
                 │                └─ feedback loop ◄──────────────── outcomes (reply/show/close)     │
                 └──────────────────────────────────────────────────────────────────────────────────┘
                          │ online (ms)                              │ offline (nightly, Python)
                     Postgres + Redis                         model retraining, MMM, uplift
```

- **Online (Go, ms):** scoring, Thompson sampling, routing, booking — the hot
  path. Go's concurrency is the right tool; this is the high-scale part.
- **Offline (nightly, Python):** heavier retraining, MMM, uplift, reporting.
- **Storage:** Postgres for entities, Redis for the hot feature cache. The
  `store.Store` interface means swapping the in-memory store for Postgres touches
  no decision logic.

---

## 5. Package map

```
cmd/brain            entry point (sim | serve)
internal/domain      core entities (Clinic, Lead, Appointment, Arm, Outcome)
internal/mathx       sampling (Beta/Gamma), sigmoid, dot — dependency-free
internal/scoring     Motor 1 — lead value + cold-start priors
internal/budget      Motor 2 — Thompson sampling, water-filling, PID pacing
internal/matching    Motor 3 — Hungarian assignment routing
internal/noshow      Motor 4 — show prediction + overbooking solver
internal/sla         guarantee controller (shadow pricing)
internal/feedback    the learning flywheel
internal/engine      orchestrator — wires it all, owns HandleLead / PlanBudget
internal/store       persistence interface + in-memory impl (Postgres-ready)
internal/priors      REAL sourced 2025–2026 benchmarks (cold-start knowledge)
internal/datasource  adapters to the outside world + SyncService (see below)
internal/api         HTTP endpoints for the serving layer
internal/sim         hidden-world simulator that proves the brain works
internal/config      tunable policy knobs
```

### How the brain reaches real data (`internal/datasource`)

The motors are pure logic; `datasource` is the adapter ring that connects them to
real systems, each behind a narrow interface so production APIs and local test
fakes are interchangeable:

| Interface | Real backend | What flows |
|-----------|--------------|------------|
| `LeadSource` | WhatsApp Cloud API webhook / Meta lead ads | inbound prospects → `HandleLead` |
| `AdPlatform` | Meta Marketing API, Google Ads API | pull true CPL, **push budget decisions**, upload conversions |
| `ClinicPMS` | clinic practice-mgmt system / CRM | capacity, push appointments, pull real show/close outcomes |
| `Messenger` | WhatsApp Cloud API send | the no-show reminders Motor 4 picks |

`SyncService` runs three loops — **leads** (decide → book in PMS → send reminder),
**ads** (correct CPL → push budgets → upload conversions so the platforms'
optimisers compound with ours), and **outcomes** (pull real results → feedback
loop). `internal/datasource/memory_adapter.go` implements all four interfaces
in-memory, and `sync_test.go` runs the whole pipeline end-to-end with no
credentials. Swap in HTTP clients for production — the SyncService is unchanged.

---

## 6. Build roadmap (3–4 months)

- **Month 1 — 1 clinic, prove the funnel.** Wire the WhatsApp agent + this brain
  in single-tenant mode (clinic-pinned leads). Learn real funnel numbers; they
  calibrate every guarantee you sell.
- **Month 2 — harden + multi-tenant.** Postgres store, dashboard, reminders/
  overbooking live, 2–3 clinics.
- **Month 3 — routing + SLA at scale.** Turn on Motor 3 (marketplace routing) and
  the guarantee controller across clinics; conversion-feedback to ad platforms.
- **Month 4 — voice agent + packaging.** Add Turkish voice for no-show recovery;
  sell the guarantee package to 10 clinics with simulation-backed numbers.

The brain in this repo is the Month-1→3 core. It already runs and is tested.
```
