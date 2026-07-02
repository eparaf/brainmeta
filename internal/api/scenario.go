package api

import (
	"encoding/json"
	"net/http"

	"disci/brain/internal/auth"
	"disci/brain/internal/domain"
	"disci/brain/internal/priors"
	"disci/brain/internal/scenario"
)

// handleScenario forecasts appointments for a hypothetical ad plan — the offline
// "with this budget, how many appointments per month?" what-if. It spends no money
// and calls no LLM; the math is the Monte-Carlo scenario engine over the funnel
// priors. When a ClinicID is supplied the caller must be able to access it, and
// blank fields (segment/budget/audience) default from that clinic.
func (s *Server) handleScenario(w http.ResponseWriter, r *http.Request) {
	var plan scenario.CampaignPlan
	if err := json.NewDecoder(r.Body).Decode(&plan); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	u, _ := auth.UserFrom(r.Context())
	if plan.ClinicID != "" && !auth.CanAccessClinic(u, plan.ClinicID) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	// Fill blanks from the clinic so the panel can POST just {clinicId, monthlyBudget}.
	if plan.ClinicID != "" {
		if c, ok := s.eng.Clinic(plan.ClinicID); ok {
			if plan.Segment == "" {
				plan.Segment = c.Segment
			}
			if plan.MonthlyBudget <= 0 {
				plan.MonthlyBudget = c.MonthlyAdBudget
			}
		}
	}
	if plan.Segment == "" {
		plan.Segment = domain.SegmentGeneral
	}
	if plan.Platform == "" {
		plan.Platform = domain.PlatformGoogle
	}
	if plan.Audience == "" {
		plan.Audience = priors.AudienceLocalTR
		if plan.Segment == domain.SegmentAesthetic {
			plan.Audience = priors.AudienceTourism
		}
	}
	res, err := scenario.New(nil).Simulate(plan)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}
