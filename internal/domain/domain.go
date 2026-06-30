// Package domain holds the core entities the brain reasons about. These types
// are intentionally storage-agnostic: the same struct flows from the WhatsApp
// webhook, through the four motors, into Postgres, and back out to the clinic
// dashboard. Keeping them pure (no DB tags, no JSON quirks) keeps the decision
// logic testable in isolation.
package domain

import "time"

// Segment captures the commercial profile of a clinic / campaign. The whole
// system behaves differently for premium-aesthetic vs high-volume-implant, so
// the segment is a first-class routing dimension rather than a free-text label.
type Segment string

const (
	SegmentAesthetic Segment = "aesthetic" // Nişantaşı: smile design, veneers, dental tourism. High ticket, low volume.
	SegmentImplant   Segment = "implant"   // Ümraniye: implants, prosthetics. Mid ticket, high volume.
	SegmentOrtho     Segment = "ortho"     // Orthodontics: long treatment, recurring revenue.
	SegmentGeneral   Segment = "general"   // Checkups, fillings. Low ticket, funnel entry.
)

// AllSegments is the canonical ordering used wherever we iterate segments.
func AllSegments() []Segment {
	return []Segment{SegmentAesthetic, SegmentImplant, SegmentOrtho, SegmentGeneral}
}

// AvgTicket returns a prior expected revenue (TRY) for a segment. These seed the
// lead-value model before we have closed-deal data of our own.
func (s Segment) AvgTicket() float64 {
	switch s {
	case SegmentAesthetic:
		return 220_000
	case SegmentImplant:
		return 45_000
	case SegmentOrtho:
		return 60_000
	default:
		return 4_000
	}
}

// Platform is an advertising channel.
type Platform string

const (
	PlatformMeta    Platform = "meta"
	PlatformGoogle  Platform = "google"
	PlatformOrganic Platform = "organic"
)

// Clinic is a tenant. Each clinic carries its own funnel, budget, capacity, and
// — critically — its guaranteed appointment commitment (the SLA the brain must
// honour).
type Clinic struct {
	ID       string
	Name     string
	District string  // e.g. "Nişantaşı", "Ümraniye"
	Side     string  // "europe" | "asia" — matters for routing & travel distance
	Segment  Segment // primary commercial profile
	LatY     float64 // crude geo coords for distance scoring
	LatX     float64

	// Commercial parameters.
	MarginRate float64 // fraction of ticket the clinic keeps as margin (drives EV)
	CloseRate  float64 // historical P(close | patient shows up) — clinic-side skill

	// Capacity: how many *new-patient* appointment slots per day the clinic can
	// actually absorb. Guaranteeing more than this is how naïve agencies blow up.
	DailyCapacity int

	// SLA: the monthly qualified-appointment guarantee we sold them.
	GuaranteedApptsPerMonth int

	// Budget: monthly ad spend the clinic funds (passthrough). The budget motor
	// allocates *within* this ceiling; it never overspends it.
	MonthlyAdBudget float64
}

// LeadStatus tracks where a lead is in the funnel.
type LeadStatus string

const (
	LeadNew       LeadStatus = "new"
	LeadQualified LeadStatus = "qualified"
	LeadBooked    LeadStatus = "booked"
	LeadShowed    LeadStatus = "showed"
	LeadClosed    LeadStatus = "closed"
	LeadLost      LeadStatus = "lost"
	LeadNoShow    LeadStatus = "no_show"
)

// Lead is a prospective patient captured from an ad click / WhatsApp message.
type Lead struct {
	ID       string
	Phone    string // patient WhatsApp number (for reminders / outbound)
	Name     string
	ClinicID string // for the single-tenant regime; empty in marketplace regime
	ArmID    string // which ad arm produced this lead (for credit assignment)
	Segment  Segment
	Platform Platform

	CreatedAt time.Time

	// Behavioural / qualification features captured by the WhatsApp agent.
	Features LeadFeatures

	Status LeadStatus

	// Filled in as the brain processes the lead.
	Score    LeadScore
	BookedAt time.Time
	ApptTime time.Time
}

// LeadFeatures are the raw signals the scorer consumes. Kept as a typed struct
// (not a map) so the feature vector ordering is stable across train/serve.
type LeadFeatures struct {
	FirstResponseSecs float64 // how fast we replied (lower = higher conversion)
	MessagesExchanged float64 // engagement depth
	DistanceKm        float64 // patient -> clinic distance
	HourOfDay         float64 // 0..23, intent varies by time
	StatedBudgetTRY   float64 // 0 if unknown
	UrgencyScore      float64 // 0..1, pain/aesthetic urgency from NLU
	PastNoShows       float64 // count, if known
	IntentScore       float64 // 0..1 from the NLU classifier on the message
}

// LeadScore is the output of Motor 1.
type LeadScore struct {
	PQualify float64 // P(lead is genuinely qualified)
	PBook    float64 // P(books an appointment | qualified)
	PShow    float64 // P(shows up | booked)
	PClose   float64 // P(treatment closes | showed)
	Ticket   float64 // expected revenue if closed (TRY)
	Margin   float64 // expected margin (TRY)
	EV       float64 // expected value of this lead to the network (TRY)
}

// Appointment is a booked slot.
type Appointment struct {
	ID       string
	ClinicID string
	LeadID   string
	Phone    string // patient number, so the reminder scheduler can reach them
	Name     string
	When     time.Time
	Segment  Segment
	PShow    float64 // model's show probability at booking time
	Showed   *bool   // nil until the appointment time passes
	Overbook bool    // true if this was an intentional overbooking slot

	// Reminder bookkeeping (which approved templates have been sent).
	Reminded24h bool
	Reminded2h  bool

	// Set when booked via the calendar widget (doctor + service chosen by patient).
	DoctorID string
	Service  string
}

// Outcome is a feedback event that flows back into the learning loop. Every
// real-world result (a reply, a booking, a no-show, a closed sale) becomes one
// of these and updates the models. This is the flywheel.
type Outcome struct {
	// OutcomeID uniquely identifies this event so the sync loop can dedup and
	// never double-count an outcome into the models.
	OutcomeID string

	LeadID   string
	ClinicID string
	ArmID    string
	Segment  Segment

	Qualified *bool
	Booked    *bool
	Showed    *bool
	Closed    *bool
	Revenue   float64

	// AdCost is the spend the ad platform attributed to delivering this lead.
	// It feeds the bandit's cost-per-lead estimate.
	AdCost float64

	// Propensity is the brain's probability of booking this lead at decision time,
	// and DropReason records why a lead was not booked. These are logged (not yet
	// used) so the counterfactual data needed for future off-policy/IPW learning
	// is preserved rather than destroyed.
	Propensity float64
	DropReason string

	At time.Time
}

// Arm is a single advertising lever: a (clinic, platform, campaign, creative,
// segment) combination. The budget motor treats each arm as a bandit arm with
// an unknown conversion rate.
type Arm struct {
	ID       string
	ClinicID string
	Platform Platform
	Campaign string
	Creative string
	Segment  Segment

	// Cost model: what we pay per click/impression on this arm, learned online.
	AvgCostPerLead float64 // TRY per lead delivered (CPL)

	// ClinicCapacity is the owning clinic's daily new-patient seat count. The
	// budget motor uses it to avoid pouring spend into an arm whose clinic is
	// already full — leads beyond what the clinic can seat are wasted money.
	ClinicCapacity int

	// ExpectedValuePerAppt is the expected realised margin (TRY) of one qualified
	// appointment from this arm's segment. The budget motor optimises VALUE per
	// TRY, not appointments per TRY — otherwise it would always starve premium
	// (high-CPL, high-ticket) tourism arms in favour of cheap local leads.
	ExpectedValuePerAppt float64
}

// Role is a dashboard user's authorization level. Admins act across every clinic;
// clinic users are scoped to their ClinicIDs. Auth is a dashboard concern layered
// on top of the brain — it never touches the learned posterior state.
type Role string

const (
	RoleAdmin  Role = "admin"  // sees/acts on every clinic
	RoleClinic Role = "clinic" // scoped to ClinicIDs
)

// User is a dashboard account for the Next.js panel. Unlike the other domain types
// it carries json tags: it round-trips through the JSONB store AND out to the
// panel, where field names must be camelCase. PasswordHash is persisted (do NOT
// json:"-" it — that would drop it from JSONB storage); it is stripped at the API
// boundary instead (see api.publicUser).
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"` // unique, lowercased
	Name         string    `json:"name"`
	PasswordHash string    `json:"passwordHash"` // PHC string; never returned to clients
	Role         Role      `json:"role"`
	ClinicIDs    []string  `json:"clinicIds"` // clinic membership; ignored for RoleAdmin
	CreatedAt    time.Time `json:"createdAt"`
}

// Connection is a per-clinic integration status (WhatsApp / Meta / Google / web
// form). The dashboard's "Bağlantılar" page reads/writes these. Detail holds only
// a masked identifier or status string — NEVER a secret/token.
type Connection struct {
	ID        string    `json:"id"` // "<clinicID>:<type>"
	ClinicID  string    `json:"clinicId"`
	Type      string    `json:"type"` // whatsapp | meta_ads | google_ads | web_form
	Connected bool      `json:"connected"`
	Detail    string    `json:"detail"` // masked status only — no secrets
	UpdatedAt time.Time `json:"updatedAt"`
}

// OAuthToken holds the SECRET credentials for a clinic's ad-platform integration
// (the refresh token + ids needed to call the platform API on the clinic's behalf).
// It is deliberately SEPARATE from Connection — Connection carries only masked
// status and is safe to return to the panel; OAuthToken is never serialised back
// to any client. Persisted (snapshot / Postgres) so live sync survives restarts.
type OAuthToken struct {
	ClinicID     string    `json:"clinicId"`
	Provider     string    `json:"provider"` // "google" | "meta"
	RefreshToken string    `json:"refreshToken"`
	CustomerID   string    `json:"customerId"` // Google Ads customer id (no dashes)
	UpdatedAt    time.Time `json:"updatedAt"`
}

// WidgetField is one configurable field of the embeddable web form.
type WidgetField struct {
	Key      string `json:"key"` // name | phone | message | email
	Label    string `json:"label"`
	Required bool   `json:"required"`
	Enabled  bool   `json:"enabled"`
}

// WidgetConfig is a clinic's customization for the embeddable JS widgets. The form
// and calendar are SEPARATE widgets, each independently styled. PublicKey is a
// publishable key embedded on the clinic's site; it only allows submitting leads /
// booking — never reading data. Doctors and services are first-class entities (see
// Doctor / Service), not part of this config.
type WidgetConfig struct {
	ClinicID  string `json:"clinicId"`
	PublicKey string `json:"publicKey"`

	// Form (contact/lead) widget.
	PrimaryColor string        `json:"primaryColor"`
	FormTitle    string        `json:"formTitle"`
	FormSubtitle string        `json:"formSubtitle"`
	SuccessText  string        `json:"successText"`
	Fields       []WidgetField `json:"fields"`

	// Calendar (appointment) widget.
	CalendarColor    string `json:"calendarColor"`
	CalendarTitle    string `json:"calendarTitle"`
	CalendarSubtitle string `json:"calendarSubtitle"`
	ConfirmText      string `json:"confirmText"`

	// Shared appearance/behaviour.
	Theme     string `json:"theme"`     // "dark" | "light"
	Recommend bool   `json:"recommend"` // show the AI "best slot" suggestion

	UpdatedAt time.Time `json:"updatedAt"`
}

// DefaultWidgetConfig returns a sensible starting config for a clinic.
func DefaultWidgetConfig(clinicID, publicKey string) WidgetConfig {
	return WidgetConfig{
		ClinicID:     clinicID,
		PublicKey:    publicKey,
		PrimaryColor: "#30d158",
		FormTitle:    "Ücretsiz Muayene Randevusu",
		FormSubtitle: "Bilgilerinizi bırakın, kliniğimiz sizi arasın.",
		SuccessText:  "Teşekkürler! En kısa sürede sizinle iletişime geçeceğiz.",
		Fields: []WidgetField{
			{Key: "name", Label: "Ad Soyad", Required: true, Enabled: true},
			{Key: "phone", Label: "Telefon", Required: true, Enabled: true},
			{Key: "message", Label: "Mesaj / Şikayet", Required: false, Enabled: true},
			{Key: "email", Label: "E-posta", Required: false, Enabled: false},
		},
		CalendarColor:    "#30d158",
		CalendarTitle:    "Online Randevu",
		CalendarSubtitle: "Hizmet ve hekim seçerek uygun saatte randevunuzu oluşturun.",
		ConfirmText:      "Randevu talebiniz alındı! Kliniğimiz onay için sizinle iletişime geçecek.",
		Theme:            "dark",
		Recommend:        true,
	}
}

// Doctor is a clinic's practitioner. Working days/hours + slot length drive the
// calendar widget's availability computation.
type Doctor struct {
	ID        string `json:"id"`
	ClinicID  string `json:"clinicId"`
	Name      string `json:"name"`
	Title     string `json:"title"`     // "Dt.", "Uzm. Dt.", "Prof. Dr." ...
	Specialty string `json:"specialty"` // free text, e.g. "İmplantoloji"
	Active    bool   `json:"active"`
	Days      []int  `json:"days"`      // working weekdays, ISO 1=Mon..7=Sun
	StartHour int    `json:"startHour"` // e.g. 9
	EndHour   int    `json:"endHour"`   // e.g. 17
	SlotMins  int    `json:"slotMins"`  // appointment granularity, e.g. 30
}

// Service is a treatment/examination type offered by a clinic. DoctorIDs are the
// doctors who can perform it (drives the calendar's service → doctor step).
type Service struct {
	ID           string   `json:"id"`
	ClinicID     string   `json:"clinicId"`
	Name         string   `json:"name"`
	DurationMins int      `json:"durationMins"`
	DoctorIDs    []string `json:"doctorIds"`
	Active       bool     `json:"active"`
}

// TemplateDraft is a clinic-authored WhatsApp message template awaiting Meta
// approval. The static Meta-approved set lives in internal/whatsapp; these are the
// drafts the dashboard can create (Status defaults to PENDING — Meta approves).
type TemplateDraft struct {
	ID        string    `json:"id"`
	ClinicID  string    `json:"clinicId"`
	Name      string    `json:"name"`
	Category  string    `json:"category"` // UTILITY | MARKETING
	Language  string    `json:"language"`
	Body      string    `json:"body"`
	Status    string    `json:"status"` // PENDING | APPROVED | REJECTED
	CreatedAt time.Time `json:"createdAt"`
}
