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
