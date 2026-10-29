package demography

import (
	"math"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/world"
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

// Health 0 means death the same day, at any age.
func TestHealthZeroDies(t *testing.T) {
	m := baseline
	for _, age := range []float64{0.5, 20, 80} {
		if p := DeathChance(age, 0, m); p != 1 {
			t.Errorf("age %v at health 0: death chance %v, want 1", age, p)
		}
	}
	if p := DeathChance(20, 0.01, m); p >= 0.001 {
		t.Errorf("age 20 at health 0.01: death chance %v, want the ordinary small risk", p)
	}
}

// The health-1 table gives exactly DeathChance's number for every age in
// days, so using it changes no result (T5.4).
func TestDeathChanceTable(t *testing.T) {
	m := &Module{cfg: &config.Scenario{Mortality: baseline}}
	for days := world.Tick(120 * year); days >= 0; days-- { // longest first: the table grows once
		if got, want := m.deathChance(days, 1), DeathChance(float64(days)/year, 1, baseline); got != want {
			t.Fatalf("day %d: table %v, formula %v", days, got, want)
		}
	}
	if got, want := m.deathChance(400, 0.7), DeathChance(400.0/year, 0.7, baseline); got != want {
		t.Errorf("health 0.7: %v, want %v", got, want)
	}
}
