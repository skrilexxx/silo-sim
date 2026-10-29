// Package demography holds mortality (ODD §6.1), partnering and housing
// (§6.3), and the lottery and reproduction (§6.4).
package demography

import (
	"math"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
)

// ODD §6.1: daily death chance. The yearly Gompertz hazard A·e^(B·age),
// plus the infant hazard in the first year of life, is raised by poor
// health and turned into a one-day probability:
//
//	p = 1 − exp(−h(age) · (1 + k_h · (1 − health)) / 365)
//
// A person whose health has reached 0 dies for certain: the health step
// (§3 step 6) runs before deaths, so that is the same day.
//
// The day-1 population (initpop) draws ages from this same rule, so the
// starting age structure is the one mortality keeps in balance.
func DeathChance(ageYears, health float64, m config.Mortality) float64 {
	if health <= 0 {
		return 1
	}
	h := m.GompertzA * math.Exp(m.GompertzB*ageYears)
	if ageYears < 1 {
		h += m.InfantHazard
	}
	// -Expm1(-x) is 1 − e^(−x) without the rounding loss for small x.
	return -math.Expm1(-h * (1 + m.HealthMultiplier*(1-health)) / engine.DaysPerYear)
}
