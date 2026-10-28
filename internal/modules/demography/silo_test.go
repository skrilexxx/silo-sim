package demography_test

import (
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/initpop"
	"github.com/skrilexxx/silo-sim/internal/modules/demography"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// T1.3 done-when: the full day-1 silo runs 20 years with deaths,
// partnering and housing, invariants checked every tick (partnerships
// symmetric, no floor over capacity, every person in one household).
func TestSiloPartnersAndHouses(t *testing.T) {
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
	e, err := engine.New(cfg, w, []engine.Module{demography.New()})
	if err != nil {
		t.Fatal(err)
	}

	startZone := make([]world.Zone, len(w.People))
	for _, id := range w.Alive {
		startZone[id] = w.Floors[w.Households[w.People[id].Household].Floor-1].Zone
	}
	seen := map[[2]world.PersonID]bool{} // couples, only for counting in this test
	sameSex, maxQueue, queueSum, months := 0, 0, 0, 0
	for w.Day < world.Tick(cfg.Years*engine.DaysPerYear) {
		if err := e.Tick(); err != nil {
			t.Fatal(err)
		}
		for _, q := range w.HousingQueue {
			k := [2]world.PersonID{min(q.A, q.B), max(q.A, q.B)}
			if !seen[k] {
				seen[k] = true
				if w.People[q.A].Sex == w.People[q.B].Sex {
					sameSex++
				}
			}
		}
		if w.Day%30 == 0 {
			months++
			queueSum += len(w.HousingQueue)
			maxQueue = max(maxQueue, len(w.HousingQueue))
		}
	}
	moved := 0
	for _, id := range w.Alive {
		if w.Floors[w.Households[w.People[id].Household].Floor-1].Zone != startZone[id] {
			moved++
		}
	}
	t.Logf("20 years: %d new couples (%.1f%% same-sex); queue after housing averaged %.1f, max %d; %d of %d survivors changed zone",
		len(seen), 100*float64(sameSex)/float64(len(seen)), float64(queueSum)/float64(months), maxQueue, moved, len(w.Alive))
	if len(seen) == 0 {
		t.Error("no couple formed in 20 years")
	}
}
