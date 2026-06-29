// Package whatsapp holds the Meta-approved message templates. WhatsApp Business
// requires PRE-APPROVED templates for business-initiated messages (appointment
// confirmations, reminders, deposit requests, reactivation) — only replies
// inside the 24h customer-service window may be free-form. This registry is the
// single source of truth for those templates: their names, category, language,
// approval status, and body with {{n}} placeholders. The no-show interventions
// the brain chooses map to these template names.
package whatsapp

import (
	"fmt"
	"sort"
	"strings"
)

// Template mirrors a WhatsApp Cloud API message template.
type Template struct {
	Name     string   `json:"name"`     // exact template name registered with Meta
	Category string   `json:"category"` // UTILITY | MARKETING | AUTHENTICATION
	Language string   `json:"language"` // e.g. tr, en
	Status   string   `json:"status"`   // APPROVED | PENDING | REJECTED
	Body     string   `json:"body"`     // with {{1}}, {{2}} placeholders
	Vars     []string `json:"vars"`     // human labels for each placeholder
}

// registry — keyed by "name:language".
var registry = map[string]Template{
	"appointment_confirmation:tr": {Name: "appointment_confirmation", Category: "UTILITY", Language: "tr", Status: "APPROVED",
		Body: "Merhaba {{1}}, {{2}} tarihli randevunuz oluşturuldu. Onaylamak için EVET yazın. Değişiklik için bu mesaja yanıt verebilirsiniz.", Vars: []string{"isim", "randevu_zamani"}},
	"appointment_confirmation:en": {Name: "appointment_confirmation", Category: "UTILITY", Language: "en", Status: "APPROVED",
		Body: "Hi {{1}}, your appointment on {{2}} is reserved. Reply YES to confirm, or reply to this message to change it.", Vars: []string{"name", "appt_time"}},

	"reminder_24h:tr": {Name: "reminder_24h", Category: "UTILITY", Language: "tr", Status: "APPROVED",
		Body: "Merhaba {{1}}, yarın {{2}} randevunuzu hatırlatmak isteriz. Gelemeyecekseniz lütfen bildirin, yerinizi başka hastaya açalım.", Vars: []string{"isim", "randevu_zamani"}},
	"reminder_2h:tr": {Name: "reminder_2h", Category: "UTILITY", Language: "tr", Status: "APPROVED",
		Body: "Merhaba {{1}}, {{2}} randevunuza 2 saat kaldı. Görüşmek üzere! Konum: {{3}}", Vars: []string{"isim", "randevu_zamani", "konum"}},
	"deposit_request:tr": {Name: "deposit_request", Category: "UTILITY", Language: "tr", Status: "APPROVED",
		Body: "Merhaba {{1}}, {{2}} randevunuzu kesinleştirmek için iade edilebilir {{3}} TL ön ödeme alabilir miyiz? Güvenli link: {{4}}", Vars: []string{"isim", "randevu_zamani", "tutar", "link"}},
	"reactivation:tr": {Name: "reactivation", Category: "MARKETING", Language: "tr", Status: "APPROVED",
		Body: "Merhaba {{1}}, kontrol zamanınız geldi. Bu ay {{2}} için uygun yerlerimiz var, randevu için yanıtlayın. Çıkmak için DUR yazın.", Vars: []string{"isim", "tedavi"}},
}

// intervention (from the no-show motor) → WhatsApp template name.
var byIntervention = map[string]string{
	"standard_reminders": "reminder_24h",
	"enhanced_reminders": "reminder_24h",
	"request_deposit":    "deposit_request",
	"confirmation_call":  "appointment_confirmation",
}

// ForIntervention maps a no-show intervention to its approved template name.
func ForIntervention(iv string) string {
	if n, ok := byIntervention[iv]; ok {
		return n
	}
	return "appointment_confirmation"
}

// Get returns a template by name + language (falls back to tr).
func Get(name, lang string) (Template, bool) {
	if t, ok := registry[name+":"+lang]; ok {
		return t, true
	}
	t, ok := registry[name+":tr"]
	return t, ok
}

// Render substitutes {{1}}, {{2}}… with args in order.
func Render(name, lang string, args ...string) (string, error) {
	t, ok := Get(name, lang)
	if !ok {
		return "", fmt.Errorf("whatsapp: no template %q", name)
	}
	out := t.Body
	for i, a := range args {
		out = strings.ReplaceAll(out, fmt.Sprintf("{{%d}}", i+1), a)
	}
	return out, nil
}

// List returns all templates (for the dashboard), sorted by name.
func List() []Template {
	out := make([]Template, 0, len(registry))
	for _, t := range registry {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].Language < out[j].Language
		}
		return out[i].Name < out[j].Name
	})
	return out
}
