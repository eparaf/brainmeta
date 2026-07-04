package store

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"log"
	"strings"
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

func (s *StorePG) CreateUser(u domain.User) error {
	u.Email = strings.ToLower(u.Email)
	b, _ := json.Marshal(u)
	_, err := s.db.Exec(`INSERT INTO users(id,email,data) VALUES($1,$2,$3)`, u.ID, u.Email, b)
	if isUniqueViolation(err) {
		return ErrDuplicate
	}
	return err
}

func (s *StorePG) GetUserByEmail(email string) (domain.User, bool) {
	var b []byte
	if err := s.db.QueryRow(`SELECT data FROM users WHERE email=$1`, strings.ToLower(email)).Scan(&b); err != nil {
		return domain.User{}, false
	}
	var u domain.User
	_ = json.Unmarshal(b, &u)
	return u, true
}

func (s *StorePG) GetUserByID(id string) (domain.User, bool) {
	var b []byte
	if err := s.db.QueryRow(`SELECT data FROM users WHERE id=$1`, id).Scan(&b); err != nil {
		return domain.User{}, false
	}
	var u domain.User
	_ = json.Unmarshal(b, &u)
	return u, true
}

func (s *StorePG) ListUsers() []domain.User {
	rows, err := s.db.Query(`SELECT data FROM users ORDER BY email`)
	if err != nil {
		log.Printf("pg ListUsers: %v", err)
		return nil
	}
	defer rows.Close()
	var out []domain.User
	for rows.Next() {
		var b []byte
		if rows.Scan(&b) == nil {
			var u domain.User
			if json.Unmarshal(b, &u) == nil {
				out = append(out, u)
			}
		}
	}
	return out
}

func (s *StorePG) ListLeads(f LeadFilter) []domain.Lead {
	q := `SELECT data FROM leads WHERE ($1='' OR clinic_id=$1) AND ($2='' OR status=$2) ORDER BY created_at DESC`
	rows, err := s.db.Query(q, f.ClinicID, string(f.Status))
	if err != nil {
		log.Printf("pg ListLeads: %v", err)
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

func (s *StorePG) UpsertConnection(c domain.Connection) {
	b, _ := json.Marshal(c)
	s.exec(`INSERT INTO connections(id,clinic_id,type,data) VALUES($1,$2,$3,$4)
	        ON CONFLICT(id) DO UPDATE SET clinic_id=EXCLUDED.clinic_id,type=EXCLUDED.type,data=EXCLUDED.data`,
		c.ID, c.ClinicID, c.Type, b)
}

func (s *StorePG) ListConnections(clinicID string) []domain.Connection {
	rows, err := s.db.Query(`SELECT data FROM connections WHERE $1='' OR clinic_id=$1 ORDER BY id`, clinicID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []domain.Connection
	for rows.Next() {
		var b []byte
		if rows.Scan(&b) == nil {
			var c domain.Connection
			if json.Unmarshal(b, &c) == nil {
				out = append(out, c)
			}
		}
	}
	return out
}

func (s *StorePG) UpsertOAuthToken(t domain.OAuthToken) {
	b, _ := json.Marshal(t)
	s.exec(`INSERT INTO oauth_tokens(id,clinic_id,provider,phone_number_id,data) VALUES($1,$2,$3,$4,$5)
	        ON CONFLICT(id) DO UPDATE SET clinic_id=EXCLUDED.clinic_id,provider=EXCLUDED.provider,phone_number_id=EXCLUDED.phone_number_id,data=EXCLUDED.data`,
		oauthTokenKey(t.ClinicID, t.Provider, t.Type), t.ClinicID, t.Provider, nullIfEmpty(t.PhoneNumberID), b)
}

func (s *StorePG) GetOAuthToken(clinicID, key string) (domain.OAuthToken, bool) {
	var b []byte
	if err := s.db.QueryRow(`SELECT data FROM oauth_tokens WHERE id=$1`, clinicID+":"+key).Scan(&b); err != nil {
		return domain.OAuthToken{}, false
	}
	var t domain.OAuthToken
	if json.Unmarshal(b, &t) != nil {
		return domain.OAuthToken{}, false
	}
	return t, true
}

func (s *StorePG) ListOAuthTokens(provider string) []domain.OAuthToken {
	rows, err := s.db.Query(`SELECT data FROM oauth_tokens WHERE $1='' OR provider=$1 ORDER BY id`, provider)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []domain.OAuthToken
	for rows.Next() {
		var b []byte
		if rows.Scan(&b) == nil {
			var t domain.OAuthToken
			if json.Unmarshal(b, &t) == nil {
				out = append(out, t)
			}
		}
	}
	return out
}

// ResolveClinicByPhoneNumberID uses the indexed phone_number_id column (see
// Memory's linear-scan doc comment for why Postgres gets a real index instead).
func (s *StorePG) ResolveClinicByPhoneNumberID(phoneNumberID string) (string, bool) {
	if phoneNumberID == "" {
		return "", false
	}
	var clinicID string
	if err := s.db.QueryRow(`SELECT clinic_id FROM oauth_tokens WHERE phone_number_id=$1 LIMIT 1`, phoneNumberID).Scan(&clinicID); err != nil {
		return "", false
	}
	return clinicID, true
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (s *StorePG) SaveTemplate(t domain.TemplateDraft) {
	b, _ := json.Marshal(t)
	s.exec(`INSERT INTO templates(id,clinic_id,status,data) VALUES($1,$2,$3,$4)
	        ON CONFLICT(id) DO UPDATE SET clinic_id=EXCLUDED.clinic_id,status=EXCLUDED.status,data=EXCLUDED.data`,
		t.ID, t.ClinicID, t.Status, b)
}

func (s *StorePG) ListTemplates(clinicID string) []domain.TemplateDraft {
	rows, err := s.db.Query(`SELECT data FROM templates WHERE $1='' OR clinic_id=$1 ORDER BY id`, clinicID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []domain.TemplateDraft
	for rows.Next() {
		var b []byte
		if rows.Scan(&b) == nil {
			var t domain.TemplateDraft
			if json.Unmarshal(b, &t) == nil {
				out = append(out, t)
			}
		}
	}
	return out
}

func (s *StorePG) SaveWidgetConfig(c domain.WidgetConfig) {
	b, _ := json.Marshal(c)
	s.exec(`INSERT INTO widgets(clinic_id,public_key,data) VALUES($1,$2,$3)
	        ON CONFLICT(clinic_id) DO UPDATE SET public_key=EXCLUDED.public_key,data=EXCLUDED.data`,
		c.ClinicID, c.PublicKey, b)
}

func (s *StorePG) GetWidgetConfig(clinicID string) (domain.WidgetConfig, bool) {
	var b []byte
	if err := s.db.QueryRow(`SELECT data FROM widgets WHERE clinic_id=$1`, clinicID).Scan(&b); err != nil {
		return domain.WidgetConfig{}, false
	}
	var c domain.WidgetConfig
	_ = json.Unmarshal(b, &c)
	return c, true
}

func (s *StorePG) GetWidgetConfigByKey(publicKey string) (domain.WidgetConfig, bool) {
	var b []byte
	if err := s.db.QueryRow(`SELECT data FROM widgets WHERE public_key=$1`, publicKey).Scan(&b); err != nil {
		return domain.WidgetConfig{}, false
	}
	var c domain.WidgetConfig
	_ = json.Unmarshal(b, &c)
	return c, true
}

func (s *StorePG) SaveDoctor(d domain.Doctor) {
	b, _ := json.Marshal(d)
	s.exec(`INSERT INTO doctors(id,clinic_id,data) VALUES($1,$2,$3)
	        ON CONFLICT(id) DO UPDATE SET clinic_id=EXCLUDED.clinic_id,data=EXCLUDED.data`,
		d.ID, d.ClinicID, b)
}

func (s *StorePG) ListDoctors(clinicID string) []domain.Doctor {
	rows, err := s.db.Query(`SELECT data FROM doctors WHERE $1='' OR clinic_id=$1 ORDER BY id`, clinicID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []domain.Doctor{}
	for rows.Next() {
		var b []byte
		if rows.Scan(&b) == nil {
			var d domain.Doctor
			if json.Unmarshal(b, &d) == nil {
				out = append(out, d)
			}
		}
	}
	return out
}

func (s *StorePG) GetDoctor(id string) (domain.Doctor, bool) {
	var b []byte
	if err := s.db.QueryRow(`SELECT data FROM doctors WHERE id=$1`, id).Scan(&b); err != nil {
		return domain.Doctor{}, false
	}
	var d domain.Doctor
	_ = json.Unmarshal(b, &d)
	return d, true
}

func (s *StorePG) DeleteDoctor(id string) { s.exec(`DELETE FROM doctors WHERE id=$1`, id) }

func (s *StorePG) SaveService(svc domain.Service) {
	b, _ := json.Marshal(svc)
	s.exec(`INSERT INTO services(id,clinic_id,data) VALUES($1,$2,$3)
	        ON CONFLICT(id) DO UPDATE SET clinic_id=EXCLUDED.clinic_id,data=EXCLUDED.data`,
		svc.ID, svc.ClinicID, b)
}

func (s *StorePG) ListServices(clinicID string) []domain.Service {
	rows, err := s.db.Query(`SELECT data FROM services WHERE $1='' OR clinic_id=$1 ORDER BY id`, clinicID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []domain.Service{}
	for rows.Next() {
		var b []byte
		if rows.Scan(&b) == nil {
			var svc domain.Service
			if json.Unmarshal(b, &svc) == nil {
				out = append(out, svc)
			}
		}
	}
	return out
}

func (s *StorePG) GetService(id string) (domain.Service, bool) {
	var b []byte
	if err := s.db.QueryRow(`SELECT data FROM services WHERE id=$1`, id).Scan(&b); err != nil {
		return domain.Service{}, false
	}
	var svc domain.Service
	_ = json.Unmarshal(b, &svc)
	return svc, true
}

func (s *StorePG) DeleteService(id string) { s.exec(`DELETE FROM services WHERE id=$1`, id) }

// isUniqueViolation reports whether err is a Postgres unique-constraint violation
// (SQLSTATE 23505). We match on the code in the error string rather than importing
// a pq/pgx error type, so this file keeps compiling in the dependency-free default
// build (rule #9).
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}

func (s *StorePG) exec(q string, args ...any) {
	if _, err := s.db.Exec(q, args...); err != nil {
		log.Printf("pg exec: %v", err)
	}
}

var _ Store = (*StorePG)(nil)
