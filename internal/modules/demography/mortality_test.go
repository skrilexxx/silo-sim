package demography

import (
	"math"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
)

var baseline = config.Mortality{GompertzA: 0.0002, GompertzB: 0.085, InfantHazard: 0.02, HealthMultiplier: 2}

// Hand-computed values of the ODD §6.1 formula.
func TestDeathChance(t *testing.T) {
	cases := []struct {
		age, health, want float64
	}{
		// h = 0.0002·e^(0.085·40) = 0.0059932; p = 1 − e^(−h/365)
		{40, 1, 1 - math.Exp(-0.0002*math.Exp(0.085*40)/365)},
		// infant: h = 0.0002 + 0.02
		{0.5, 1, 1 - math.Exp(-(0.0002*math.Exp(0.085*0.5)+0.02)/365)},
		// health 0.5 doubles the hazard: 1 + 2·(1 − 0.5) = 2
		{40, 0.5, 1 - math.Exp(-2*0.0002*math.Exp(0.085*40)/365)},
	}
	for _, c := range cases {
		if got := DeathChance(c.age, c.health, baseline); math.Abs(got-c.want) > 1e-15 {
			t.Errorf("DeathChance(%v, %v) = %v, want %v", c.age, c.health, got, c.want)
		}
	}
	if DeathChance(1, 1, baseline) >= DeathChance(0.99, 1, baseline) {
		t.Error("infant hazard should stop at age 1")
	}
}
