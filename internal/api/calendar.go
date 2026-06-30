package api

import (
	"encoding/json"
	"net/http"
	"time"

	"disci/brain/internal/auth"
	"disci/brain/internal/domain"
	"disci/brain/internal/noshow"
)

// ---- protected: doctor CRUD --------------------------------------------------

func (s *Server) handleListDoctors(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	clinicID := r.URL.Query().Get("clinicId")
	if clinicID != "" && !auth.CanAccessClinic(u, clinicID) {
		writeJSON(w, 403, map[string]string{"error": "forbidden"})
		return
	}
	out := []domain.Doctor{}
	for _, d := range s.store.ListDoctors(clinicID) {
		if auth.CanAccessClinic(u, d.ClinicID) {
			out = append(out, d)
		}
	}
	writeJSON(w, 200, out)
}

func (s *Server) handleSaveDoctor(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var d domain.Doctor
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	if !auth.CanAccessClinic(u, d.ClinicID) {
		writeJSON(w, 403, map[string]string{"error": "forbidden"})
		return
	}
	if d.Name == "" {
		writeJSON(w, 400, map[string]string{"error": "name required"})
		return
	}
	if d.ID == "" {
		d.ID = "doc-" + newID()
	}
	if d.SlotMins <= 0 {
		d.SlotMins = 30
	}
	if d.EndHour <= d.StartHour {
		d.StartHour, d.EndHour = 9, 17
	}
	if len(d.Days) == 0 {
		d.Days = []int{1, 2, 3, 4, 5}
	}
	s.store.SaveDoctor(d)
	writeJSON(w, 200, d)
}

func (s *Server) handleDeleteDoctor(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	d, ok := s.store.GetDoctor(r.PathValue("id"))
	if !ok {
		writeJSON(w, 404, map[string]string{"error": "not_found"})
		return
	}
	if !auth.CanAccessClinic(u, d.ClinicID) {
		writeJSON(w, 403, map[string]string{"error": "forbidden"})
		return
	}
	s.store.DeleteDoctor(d.ID)
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// ---- protected: service CRUD -------------------------------------------------

func (s *Server) handleListServices(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	clinicID := r.URL.Query().Get("clinicId")
	if clinicID != "" && !auth.CanAccessClinic(u, clinicID) {
		writeJSON(w, 403, map[string]string{"error": "forbidden"})
		return
	}
	out := []domain.Service{}
	for _, svc := range s.store.ListServices(clinicID) {
		if auth.CanAccessClinic(u, svc.ClinicID) {
			out = append(out, svc)
		}
	}
	writeJSON(w, 200, out)
}

func (s *Server) handleSaveService(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var svc domain.Service
	if err := json.NewDecoder(r.Body).Decode(&svc); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	if !auth.CanAccessClinic(u, svc.ClinicID) {
		writeJSON(w, 403, map[string]string{"error": "forbidden"})
		return
	}
	if svc.Name == "" {
		writeJSON(w, 400, map[string]string{"error": "name required"})
		return
	}
	if svc.ID == "" {
		svc.ID = "svc-" + newID()
	}
	if svc.DurationMins <= 0 {
		svc.DurationMins = 30
	}
	s.store.SaveService(svc)
	writeJSON(w, 200, svc)
}

func (s *Server) handleDeleteService(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	svc, ok := s.store.GetService(r.PathValue("id"))
	if !ok {
		writeJSON(w, 404, map[string]string{"error": "not_found"})
		return
	}
	if !auth.CanAccessClinic(u, svc.ClinicID) {
		writeJSON(w, 403, map[string]string{"error": "forbidden"})
		return
	}
	s.store.DeleteService(svc.ID)
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// ---- public calendar flow (no auth, CORS *) ----------------------------------

func (s *Server) activeDoctorIDs(clinicID string) map[string]domain.Doctor {
	m := map[string]domain.Doctor{}
	for _, d := range s.store.ListDoctors(clinicID) {
		if d.Active {
			m[d.ID] = d
		}
	}
	return m
}

// GET /public/widget/services?key=
func (s *Server) handlePublicServices(w http.ResponseWriter, r *http.Request) {
	cfg, ok := s.widgetByKey(r)
	if !ok {
		writeJSON(w, 401, map[string]string{"error": "invalid_key"})
		return
	}
	docs := s.activeDoctorIDs(cfg.ClinicID)
	out := []map[string]any{}
	for _, svc := range s.store.ListServices(cfg.ClinicID) {
		if !svc.Active {
			continue
		}
		hasDoctor := false
		for _, id := range svc.DoctorIDs {
			if _, ok := docs[id]; ok {
				hasDoctor = true
				break
			}
		}
		if !hasDoctor {
			continue
		}
		out = append(out, map[string]any{
			"id": svc.ID, "name": svc.Name, "durationMins": svc.DurationMins,
		})
	}
	writeJSON(w, 200, out)
}

// GET /public/widget/doctors?key=&serviceId=
func (s *Server) handlePublicDoctors(w http.ResponseWriter, r *http.Request) {
	cfg, ok := s.widgetByKey(r)
	if !ok {
		writeJSON(w, 401, map[string]string{"error": "invalid_key"})
		return
	}
	svc, ok := s.store.GetService(r.URL.Query().Get("serviceId"))
	if !ok || svc.ClinicID != cfg.ClinicID {
		writeJSON(w, 404, map[string]string{"error": "not_found"})
		return
	}
	allow := map[string]bool{}
	for _, id := range svc.DoctorIDs {
		allow[id] = true
	}
	out := []map[string]any{}
	for _, d := range s.store.ListDoctors(cfg.ClinicID) {
		if d.Active && allow[d.ID] {
			out = append(out, map[string]any{
				"id": d.ID, "name": d.Name, "title": d.Title, "specialty": d.Specialty,
				"days": d.Days,
			})
		}
	}
	writeJSON(w, 200, out)
}

// firstAvailable returns the doctor's earliest free slot within the horizon, plus
// how many free slots they have in the next 7 days (a "how open is this doctor"
// signal). zero time + 0 if none.
func (s *Server) firstAvailable(d domain.Doctor, horizonDays int) (time.Time, int) {
	now := time.Now().UTC()
	var first time.Time
	free7 := 0
	for off := 0; off < horizonDays; off++ {
		day := now.AddDate(0, 0, off)
		slots := s.availability(d, day)
		if first.IsZero() && len(slots) > 0 {
			first = slots[0]
		}
		if off < 7 {
			free7 += len(slots)
		}
	}
	return first, free7
}

// handlePublicRecommend suggests the best doctor + earliest slot for a service.
// "Boşluğa göre öneri": picks the soonest available appointment, tie-broken by the
// most-open doctor, then annotates the brain's show-probability for that slot.
// GET /public/widget/recommend?key=&serviceId=
func (s *Server) handlePublicRecommend(w http.ResponseWriter, r *http.Request) {
	cfg, ok := s.widgetByKey(r)
	if !ok {
		writeJSON(w, 401, map[string]string{"error": "invalid_key"})
		return
	}
	svc, ok := s.store.GetService(r.URL.Query().Get("serviceId"))
	if !ok || svc.ClinicID != cfg.ClinicID {
		writeJSON(w, 404, map[string]string{"error": "not_found"})
		return
	}
	allow := map[string]bool{}
	for _, id := range svc.DoctorIDs {
		allow[id] = true
	}
	var best domain.Doctor
	var bestSlot time.Time
	bestFree := -1
	for _, d := range s.store.ListDoctors(cfg.ClinicID) {
		if !d.Active || !allow[d.ID] {
			continue
		}
		slot, free := s.firstAvailable(d, 14)
		if slot.IsZero() {
			continue
		}
		// Prefer the earliest slot; on a tie (same day) prefer the more-open doctor.
		better := bestSlot.IsZero() || slot.Before(bestSlot) ||
			(sameDay(slot, bestSlot) && free > bestFree)
		if better {
			best, bestSlot, bestFree = d, slot, free
		}
	}
	if bestSlot.IsZero() {
		writeJSON(w, 200, map[string]any{"available": false})
		return
	}
	// Integrate the brain's no-show motor: show probability for this slot.
	clinic, _ := s.eng.Clinic(cfg.ClinicID)
	lead := bestSlot.Sub(time.Now().UTC()).Hours() / 24.0
	pShow := s.eng.NoShow.PShow(noshow.Appt{
		LeadTimeDays: lead, HourOfDay: float64(bestSlot.Hour()), ConfirmedReply: true,
		Segment: clinic.Segment,
	})
	writeJSON(w, 200, map[string]any{
		"available": true,
		"doctor":    map[string]any{"id": best.ID, "name": best.Name, "title": best.Title, "specialty": best.Specialty},
		"slot":      map[string]string{"iso": bestSlot.Format(time.RFC3339), "label": bestSlot.Format("02 Jan, 15:04")},
		"pShow":     pShow,
		"reason":    "En erken uygun randevu",
	})
}

// GET /public/widget/availability?key=&doctorId=&date=YYYY-MM-DD
func (s *Server) handlePublicAvailability(w http.ResponseWriter, r *http.Request) {
	cfg, ok := s.widgetByKey(r)
	if !ok {
		writeJSON(w, 401, map[string]string{"error": "invalid_key"})
		return
	}
	d, ok := s.store.GetDoctor(r.URL.Query().Get("doctorId"))
	if !ok || d.ClinicID != cfg.ClinicID || !d.Active {
		writeJSON(w, 404, map[string]string{"error": "not_found"})
		return
	}
	date, err := time.Parse("2006-01-02", r.URL.Query().Get("date"))
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "bad date"})
		return
	}
	out := []map[string]string{}
	for _, t := range s.availability(d, date) {
		out = append(out, map[string]string{"iso": t.Format(time.RFC3339), "label": t.Format("15:04")})
	}
	writeJSON(w, 200, out)
}

// POST /public/widget/book?key=
func (s *Server) handlePublicCalendarBook(w http.ResponseWriter, r *http.Request) {
	cfg, ok := s.widgetByKey(r)
	if !ok {
		writeJSON(w, 401, map[string]string{"error": "invalid_key"})
		return
	}
	var b struct {
		ServiceID string `json:"serviceId"`
		DoctorID  string `json:"doctorId"`
		Slot      string `json:"slot"`
		Name      string `json:"name"`
		Phone     string `json:"phone"`
		Note      string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	if b.Name == "" || b.Phone == "" {
		writeJSON(w, 400, map[string]string{"error": "name and phone required"})
		return
	}
	d, ok := s.store.GetDoctor(b.DoctorID)
	if !ok || d.ClinicID != cfg.ClinicID {
		writeJSON(w, 400, map[string]string{"error": "invalid doctor"})
		return
	}
	slot, err := time.Parse(time.RFC3339, b.Slot)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid slot"})
		return
	}
	if !s.slotFree(d, slot) {
		writeJSON(w, 409, map[string]string{"error": "slot_taken"})
		return
	}
	svc, _ := s.store.GetService(b.ServiceID)
	clinic, _ := s.eng.Clinic(cfg.ClinicID)

	s.store.SaveAppointment(domain.Appointment{
		ID: "web-appt-" + newID(), ClinicID: cfg.ClinicID, DoctorID: d.ID, Service: svc.Name,
		Name: b.Name, Phone: b.Phone, When: slot, Segment: clinic.Segment, PShow: 0.9,
	})
	note := "Online randevu: " + svc.Name + " · " + d.Name
	if b.Note != "" {
		note += " — " + b.Note
	}
	s.store.SaveLead(domain.Lead{
		ID: "web-" + newID(), Phone: b.Phone, Name: b.Name, ClinicID: cfg.ClinicID,
		ArmID: cfg.ClinicID + ":webform:" + string(clinic.Segment), Segment: clinic.Segment,
		Platform: domain.PlatformOrganic, CreatedAt: time.Now(), Status: domain.LeadBooked,
		ApptTime: slot,
		Features: domain.LeadFeatures{
			IntentScore: 0.9, UrgencyScore: 0.6, MessagesExchanged: 1,
			FirstResponseSecs: 20, HourOfDay: float64(slot.Hour()),
		},
	})
	s.eng.SLA.RecordQualifiedAppt(cfg.ClinicID)
	writeJSON(w, 200, map[string]any{
		"ok": true, "apptTime": slot, "doctor": d.Name, "service": svc.Name,
	})
}

// availability returns the doctor's free slots on a date (working hours minus
// already-booked slots and past times). All times are UTC wall-clock.
func (s *Server) availability(d domain.Doctor, date time.Time) []time.Time {
	iso := int(date.Weekday())
	if iso == 0 {
		iso = 7 // Sunday
	}
	working := false
	for _, wd := range d.Days {
		if wd == iso {
			working = true
		}
	}
	if !working {
		return nil
	}
	step := d.SlotMins
	if step <= 0 {
		step = 30
	}
	start, end := d.StartHour, d.EndHour
	if end <= start {
		start, end = 9, 17
	}
	booked := map[string]bool{}
	for _, a := range s.eng.Appointments(d.ClinicID) {
		if a.DoctorID == d.ID && sameDay(a.When, date) {
			booked[a.When.UTC().Format("15:04")] = true
		}
	}
	now := time.Now().UTC()
	day := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	out := []time.Time{}
	for mins := start * 60; mins+step <= end*60; mins += step {
		t := day.Add(time.Duration(mins) * time.Minute)
		if t.Before(now) || booked[t.Format("15:04")] {
			continue
		}
		out = append(out, t)
	}
	return out
}

func (s *Server) slotFree(d domain.Doctor, slot time.Time) bool {
	for _, a := range s.eng.Appointments(d.ClinicID) {
		if a.DoctorID == d.ID && a.When.Equal(slot) {
			return false
		}
	}
	return true
}

func sameDay(a, b time.Time) bool {
	au, bu := a.UTC(), b.UTC()
	return au.Year() == bu.Year() && au.YearDay() == bu.YearDay()
}
