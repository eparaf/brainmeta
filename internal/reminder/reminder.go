// Package reminder is the scheduled no-show defence: it scans upcoming
// appointments and sends Meta-APPROVED WhatsApp templates at the 24h and 2h
// marks, honouring opt-out. This is what turns the no-show motor's intent into
// real outbound nudges. Runs on a ticker from main.
package reminder

import (
	"context"
	"sync"
	"time"

	"disci/brain/internal/domain"
)

// Notifier sends an approved template message (whatsapp.Cloud satisfies this).
type Notifier interface {
	SendTemplate(ctx context.Context, to, name, lang string, params ...string) error
}

// Scheduler sends 24h/2h reminders. It tracks what it has already sent in-memory
// (keyed by appointment+window) so it never double-sends within a process.
type Scheduler struct {
	Clinics  func() []domain.Clinic
	Appts    func(clinicID string) []domain.Appointment
	Notifier Notifier
	Allowed  func(phone string) bool // consent gate; nil = allow all

	mu   sync.Mutex
	sent map[string]bool
}

func New(clinics func() []domain.Clinic, appts func(string) []domain.Appointment, n Notifier, allowed func(string) bool) *Scheduler {
	return &Scheduler{Clinics: clinics, Appts: appts, Notifier: n, Allowed: allowed, sent: map[string]bool{}}
}

// Run ticks every interval until ctx is done.
func (s *Scheduler) Run(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			s.Tick(ctx, now)
		}
	}
}

// Tick sends any due reminders at time `now`. Returns how many it sent.
func (s *Scheduler) Tick(ctx context.Context, now time.Time) int {
	if s.Notifier == nil {
		return 0
	}
	count := 0
	for _, c := range s.Clinics() {
		for _, a := range s.Appts(c.ID) {
			if a.Phone == "" || a.When.Before(now) {
				continue
			}
			if s.Allowed != nil && !s.Allowed(a.Phone) {
				continue
			}
			d := a.When.Sub(now)
			when := a.When.Format("02 Jan 15:04")
			name := a.Name
			if name == "" {
				name = "değerli hastamız"
			}
			// Exclusive windows: 2h reminder when ≤2h out; 24h reminder only in the
			// (2h, 24h] band — so a near appointment never also triggers the 24h one.
			switch {
			case d <= 2*time.Hour && !s.mark(a.ID, "2h"):
				if s.Notifier.SendTemplate(ctx, a.Phone, "reminder_2h", "tr", name, when, c.Name) == nil {
					count++
				}
			case d > 2*time.Hour && d <= 24*time.Hour && !s.mark(a.ID, "24h"):
				if s.Notifier.SendTemplate(ctx, a.Phone, "reminder_24h", "tr", name, when) == nil {
					count++
				}
			}
		}
	}
	return count
}

// mark returns true if (appt,window) was ALREADY sent, otherwise records it.
func (s *Scheduler) mark(apptID, window string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := apptID + ":" + window
	if s.sent[k] {
		return true
	}
	s.sent[k] = true
	return false
}
