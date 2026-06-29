package store

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"log"
	"time"

	"disci/brain/internal/domain"
)

//go:embed schema.sql
var schemaSQL string

// StorePG is a Postgres-backed store.Store. It uses the stdlib database/sql, so
// this file compiles with NO third-party dependency; to actually connect, add a
// driver in your main (e.g. `import _ "github.com/jackc/pgx/v5/stdlib"`,
// `go get github.com/jackc/pgx/v5`) and call NewPostgres with a DSN. Entities are
// stored as jsonb (see schema.sql) so the schema stays simple and forward-compatible.
//
// The store.Store methods don't return errors (interface parity with Memory), so
// DB errors are logged; wrap with metrics/alerts in production.
type StorePG struct{ db *sql.DB }

// NewPostgres opens a pool and verifies connectivity. Returns an error (e.g.
// "unknown driver") if the driver isn't registered — callers fall back to Memory.
func NewPostgres(driver, dsn string) (*StorePG, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	s := &StorePG{db: db}
	if err := s.Migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

// Migrate applies the embedded schema (idempotent: CREATE TABLE IF NOT EXISTS),
// so a fresh database is ready with no manual psql step.
func (s *StorePG) Migrate() error {
	_, err := s.db.Exec(schemaSQL)
	return err
}

func (s *StorePG) UpsertClinic(c domain.Clinic) {
	b, _ := json.Marshal(c)
	s.exec(`INSERT INTO clinics(id,data) VALUES($1,$2)
	        ON CONFLICT(id) DO UPDATE SET data=EXCLUDED.data`, c.ID, b)
}

func (s *StorePG) GetClinic(id string) (domain.Clinic, bool) {
	var b []byte
	if err := s.db.QueryRow(`SELECT data FROM clinics WHERE id=$1`, id).Scan(&b); err != nil {
		return domain.Clinic{}, false
	}
	var c domain.Clinic
	_ = json.Unmarshal(b, &c)
	return c, true
}

func (s *StorePG) ListClinics() []domain.Clinic {
	rows, err := s.db.Query(`SELECT data FROM clinics ORDER BY id`)
	if err != nil {
		log.Printf("pg ListClinics: %v", err)
		return nil
	}
	defer rows.Close()
	var out []domain.Clinic
	for rows.Next() {
		var b []byte
		if rows.Scan(&b) == nil {
			var c domain.Clinic
			if json.Unmarshal(b, &c) == nil {
				out = append(out, c)
			}
		}
	}
	return out
}

func (s *StorePG) SaveLead(l domain.Lead)   { s.upsertLead(l) }
func (s *StorePG) UpdateLead(l domain.Lead) { s.upsertLead(l) }
func (s *StorePG) upsertLead(l domain.Lead) {
	b, _ := json.Marshal(l)
	s.exec(`INSERT INTO leads(id,clinic_id,status,data) VALUES($1,$2,$3,$4)
	        ON CONFLICT(id) DO UPDATE SET clinic_id=EXCLUDED.clinic_id,status=EXCLUDED.status,data=EXCLUDED.data`,
		l.ID, l.ClinicID, string(l.Status), b)
}

func (s *StorePG) GetLead(id string) (domain.Lead, bool) {
	var b []byte
	if err := s.db.QueryRow(`SELECT data FROM leads WHERE id=$1`, id).Scan(&b); err != nil {
		return domain.Lead{}, false
	}
	var l domain.Lead
	_ = json.Unmarshal(b, &l)
	return l, true
}

func (s *StorePG) LeadsByStatus(st domain.LeadStatus) []domain.Lead {
	rows, err := s.db.Query(`SELECT data FROM leads WHERE status=$1`, string(st))
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []domain.Lead
	for rows.Next() {
		var b []byte
		if rows.Scan(&b) == nil {
			var l domain.Lead
			if json.Unmarshal(b, &l) == nil {
				out = append(out, l)
			}
		}
	}
	return out
}

func (s *StorePG) SaveAppointment(a domain.Appointment) {
	b, _ := json.Marshal(a)
	s.exec(`INSERT INTO appointments(id,clinic_id,when_ts,overbook,data) VALUES($1,$2,$3,$4,$5)
	        ON CONFLICT(id) DO UPDATE SET data=EXCLUDED.data`, a.ID, a.ClinicID, a.When, a.Overbook, b)
}

func (s *StorePG) AppointmentsForClinic(clinicID string) []domain.Appointment {
	rows, err := s.db.Query(`SELECT data FROM appointments WHERE clinic_id=$1`, clinicID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []domain.Appointment
	for rows.Next() {
		var b []byte
		if rows.Scan(&b) == nil {
			var a domain.Appointment
			if json.Unmarshal(b, &a) == nil {
				out = append(out, a)
			}
		}
	}
	return out
}

func (s *StorePG) SeatUsage(clinicID string) int {
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM appointments WHERE clinic_id=$1 AND NOT overbook AND when_ts::date=$2`,
		clinicID, time.Now().Format("2006-01-02")).Scan(&n)
	return n
}

func (s *StorePG) exec(q string, args ...any) {
	if _, err := s.db.Exec(q, args...); err != nil {
		log.Printf("pg exec: %v", err)
	}
}

var _ Store = (*StorePG)(nil)
