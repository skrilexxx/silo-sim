package resources_test

import (
	"os"
	"testing"
	"time"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/initpop"
	"github.com/skrilexxx/silo-sim/internal/modules/demography"
	"github.com/skrilexxx/silo-sim/internal/modules/health"
	"github.com/skrilexxx/silo-sim/internal/modules/resources"
	"github.com/skrilexxx/silo-sim/internal/modules/work"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// T3.2a stability check: the baseline with demography, health, work and
// resources, 10 seeds × 100 years from day 1. Too slow for make check, so
// it runs only with SILOSIM_STABILITY=1 (make stability). Seeds run in
// parallel; each run is still one goroutine (CLAUDE.md, Determinism).
//
// Target (docs/tasks.md T3.2a): the silo survives every generator failure,
// i.e. average health is back above 0.9 within a year of each failure.
func TestStability(t *testing.T) {
	if os.Getenv("SILOSIM_STABILITY") == "" {
		t.Skip("set SILOSIM_STABILITY=1 (make stability)")
	}
	for seed := uint64(1); seed <= 10; seed++ {
		t.Run("seed", func(t *testing.T) {
			t.Parallel()
			runStability(t, seed)
		})
	}
}

func runStability(t *testing.T, seed uint64) {
	cfg, err := config.Load("../../../scenarios/baseline.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Seed, cfg.WarmupYears, cfg.Years = seed, 0, 100
	cfg.Modules = []string{"demography", "health", "work", "resources"}
	r, err := rng.New(seed, "initpop")
	if err != nil {
		t.Fatal(err)
	}
	w, err := initpop.Build(cfg, r)
	if err != nil {
		t.Fatal(err)
	}
	e, err := engine.New(cfg, w, []engine.Module{demography.New(), health.New(), work.New(), resources.New()})
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	end := world.Tick(cfg.Years * engine.DaysPerYear)
	failures, recovered, longest := 0, 0, 0
	minHealth, minAir := 1.0, 1.0
	var failDay world.Tick // last failure still waiting to recover; 0 if none
	for w.Day < end {
		down := w.Silo.GeneratorDown
		if err := e.Tick(); err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		if down == 0 && w.Silo.GeneratorDown > 0 {
			failures++
			longest = max(longest, w.Silo.GeneratorDown)
			if failDay == 0 {
				failDay = w.Day
			}
		}
		h := 0.0
		for _, id := range w.Alive {
			h += w.People[id].Health
		}
		h /= float64(len(w.Alive))
		minHealth, minAir = min(minHealth, h), min(minAir, w.Silo.AirQuality)
		if failDay > 0 && w.Silo.GeneratorDown == 0 && h > 0.9 {
			recovered++
			failDay = 0
		}
		if failDay > 0 && w.Day-failDay > engine.DaysPerYear {
			break // not back within a year: the silo did not survive
		}
	}
	ok := failDay == 0
	t.Logf("seed %2d: %d failures, longest outage %d days, lowest avg health %.3f, lowest air %.3f, condition %.3f, population %d %s | %.0fs",
		seed, failures, longest, minHealth, minAir, w.Silo.GeneratorCondition, len(w.Alive), mark(ok), time.Since(start).Seconds())
	if !ok {
		t.Errorf("seed %d: average health not back above 0.9 within a year of the failure on day %d", seed, failDay)
	}
}

func mark(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}
