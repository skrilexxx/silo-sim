package demography_test

// An external test package, because initpop itself imports demography.

import (
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/initpop"
	"github.com/skrilexxx/silo-sim/internal/modules/demography"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// deathsOnly runs just the demography module's death step, so a test of
// the T1.2 rules is not mixed with partnering and housing.
type deathsOnly struct{ *demography.Module }

func (d deathsOnly) Steps() []engine.Step {
	var out []engine.Step
	for _, s := range d.Module.Steps() {
		if s.Phase == engine.PhaseDeaths {
			out = append(out, s)
		}
	}
	return out
}

// T1.2 done-when: over 20 years of the full day-1 silo with deaths only, every
// child under 18 always lives in a household with an adult, in the zone
// they started in. Invariants run every tick.
func TestOrphansStayHousedInZone(t *testing.T) {
	cfg, err := config.Load("../../../scenarios/baseline.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Output.InvariantEvery = 1
	cfg.WarmupYears, cfg.Years = 0, 20
	r, _ := rng.New(cfg.Seed, "initpop")
	w, err := initpop.Build(cfg, r)
	if err != nil {
		t.Fatal(err)
	}
	zone := func(id world.PersonID) world.Zone {
		return w.Floors[w.Households[w.People[id].Household].Floor-1].Zone
	}
	startZone := make([]world.Zone, len(w.People))
	startHH := make([]world.HouseholdID, len(w.People))
	for _, id := range w.Alive {
		startZone[id], startHH[id] = zone(id), w.People[id].Household
	}

	e, err := engine.New(cfg, w, []engine.Module{deathsOnly{demography.New()}})
	if err != nil {
		t.Fatal(err)
	}
	adult := world.Tick(cfg.Partnering.MinAge * engine.DaysPerYear)
	hasAdult := func(h *world.Household) bool {
		for _, id := range h.Members {
			if w.Day-w.People[id].BirthDay >= adult {
				return true
			}
		}
		return false
	}
	moved := 0
	for range cfg.Years {
		for range engine.DaysPerYear {
			if err := e.Tick(); err != nil {
				t.Fatal(err)
			}
		}
		for _, id := range w.Alive {
			if w.Day-w.People[id].BirthDay >= adult {
				continue
			}
			if !hasAdult(&w.Households[w.People[id].Household]) {
				t.Fatalf("day %d: child %d lives without an adult", w.Day, id)
			}
			if zone(id) != startZone[id] {
				t.Fatalf("day %d: child %d moved from zone %d to %d", w.Day, id, startZone[id], zone(id))
			}
		}
	}
	for i := range w.People {
		if w.People[i].Alive && w.People[i].Household != startHH[i] {
			moved++
		}
	}
	t.Logf("%d of 10000 died in 20 years; %d survivors were rehomed as orphans", 10000-len(w.Alive), moved)
	if moved == 0 {
		t.Error("no orphan was rehomed, so the test proved nothing")
	}
}
