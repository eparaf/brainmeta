-- BrainMeta entity store (Postgres). Learned model state lives in the snapshot
-- file; THIS holds the operational entities. Entities are stored as jsonb so the
-- domain structs can evolve without migrations for every field.
--
-- Apply:  psql "$DATABASE_URL" -f internal/store/schema.sql
-- Use:    add a driver (go get github.com/jackc/pgx/v5; import _ ".../stdlib"),
--         then store.NewPostgres("pgx", os.Getenv("DATABASE_URL")).

CREATE TABLE IF NOT EXISTS clinics (
  id   TEXT PRIMARY KEY,
  data JSONB NOT NULL
);

CREATE TABLE IF NOT EXISTS leads (
  id        TEXT PRIMARY KEY,
  clinic_id TEXT,
  status    TEXT,
  data      JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_leads_status ON leads(status);
CREATE INDEX IF NOT EXISTS idx_leads_clinic ON leads(clinic_id);

CREATE TABLE IF NOT EXISTS appointments (
  id        TEXT PRIMARY KEY,
  clinic_id TEXT,
  when_ts   TIMESTAMPTZ,
  overbook  BOOLEAN NOT NULL DEFAULT false,
  data      JSONB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_appt_clinic ON appointments(clinic_id);
CREATE INDEX IF NOT EXISTS idx_appt_when ON appointments(when_ts);

-- Consent / opt-out (KVKK). Phone is the contact key.
CREATE TABLE IF NOT EXISTS consent (
  phone     TEXT PRIMARY KEY,
  opted_out BOOLEAN NOT NULL DEFAULT false,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Dashboard users (Next.js panel auth). The brain itself is account-agnostic;
-- this is the login surface. Stored as jsonb like the other entities, with email
-- lifted into its own UNIQUE column for fast, case-insensitive login lookups.
CREATE TABLE IF NOT EXISTS users (
  id    TEXT PRIMARY KEY,
  email TEXT UNIQUE NOT NULL,
  data  JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Per-clinic integration status (dashboard "Bağlantılar"). No secrets stored.
CREATE TABLE IF NOT EXISTS connections (
  id         TEXT PRIMARY KEY,        -- "<clinicID>:<type>"
  clinic_id  TEXT,
  type       TEXT,
  data       JSONB NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_conn_clinic ON connections(clinic_id);

-- Ad-platform OAuth secrets (refresh tokens) for live spend sync. SEPARATE from
-- `connections` (which is status-only) so secrets never leak through that surface.
CREATE TABLE IF NOT EXISTS oauth_tokens (
  id         TEXT PRIMARY KEY,        -- "<clinicID>:<provider>"
  clinic_id  TEXT,
  provider   TEXT,                    -- google | meta
  data       JSONB NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_oauth_provider ON oauth_tokens(provider);

-- Clinic-authored WhatsApp template drafts (PENDING until Meta approves).
CREATE TABLE IF NOT EXISTS templates (
  id        TEXT PRIMARY KEY,
  clinic_id TEXT,
  status    TEXT,
  data      JSONB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_tmpl_clinic ON templates(clinic_id);

-- Embeddable widget config (web form + calendar). One per clinic; public_key is
-- the publishable embed key looked up by the public endpoints.
CREATE TABLE IF NOT EXISTS widgets (
  clinic_id  TEXT PRIMARY KEY,
  public_key TEXT UNIQUE NOT NULL,
  data       JSONB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_widget_key ON widgets(public_key);

-- Clinic practitioners + treatment/examination types for the appointment calendar.
CREATE TABLE IF NOT EXISTS doctors (
  id        TEXT PRIMARY KEY,
  clinic_id TEXT,
  data      JSONB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_doctor_clinic ON doctors(clinic_id);

CREATE TABLE IF NOT EXISTS services (
  id        TEXT PRIMARY KEY,
  clinic_id TEXT,
  data      JSONB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_service_clinic ON services(clinic_id);
