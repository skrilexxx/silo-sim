package demography_test

import (
	"math"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/initpop"
	"github.com/skrilexxx/silo-sim/internal/modules/demography"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// T1.5 milestone check: the baseline with demography only, 10 seeds, 50
// warm-up years plus 200 measured years. Too slow for make check, so it
// runs only with SILOSIM_MILESTONE=1 (make milestone1). Seeds run in
// parallel; each run is still one goroutine (CLAUDE.md, Determinism).
//
// Targets (docs/tasks.md T1.5, ODD §8): population within 9,000–11,000 in
// at least 9 of 10 seeds; median age 30–40. The largest year-to-year
// change in births is logged as well (ODD §8: under 30%).
func TestMilestone1(t *testing.T) {
	if os.Getenv("SILOSIM_MILESTONE") == "" {
		t.Skip("set SILOSIM_MILESTONE=1 (make milestone1)")
	}
	for seed := uint64(1); seed <= 10; seed++ {
		t.Run("seed", func(t *testing.T) {
			t.Parallel()
			runMilestone(t, seed)
		})
	}
}

func runMilestone(t *testing.T, seed uint64) {
	cfg, err := config.Load("../../../scenarios/baseline.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Seed = seed
	cfg.Modules = []string{"demography"}
	r, err := rng.New(seed, "initpop")
	if err != nil {
		t.Fatal(err)
	}
	w, err := initpop.Build(cfg, r)
	if err != nil {
		t.Fatal(err)
	}
	e, err := engine.New(cfg, w, []engine.Module{demography.New()})
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	warmup := world.Tick(cfg.WarmupYears * engine.DaysPerYear)
	end := world.Tick((cfg.WarmupYears + cfg.Years) * engine.DaysPerYear)
	minPop, maxPop := math.MaxInt, 0
	minMed, maxMed := math.Inf(1), 0.0
	var births []int // per measured year
	bornBefore := len(w.People)
	ages := make([]int, 0, len(w.Alive))
	for w.Day < end {
		if err := e.Tick(); err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		if w.Day <= warmup {
			if w.Day == warmup {
				bornBefore = len(w.People)
			}
			continue
		}
		minPop, maxPop = min(minPop, len(w.Alive)), max(maxPop, len(w.Alive))
		if w.Day%engine.DaysPerYear == 0 {
			ages = ages[:0]
			for _, id := range w.Alive {
				ages = append(ages, int(w.Day-w.People[id].BirthDay))
			}
			slices.Sort(ages)
			med := float64(ages[len(ages)/2]) / engine.DaysPerYear
			minMed, maxMed = min(minMed, med), max(maxMed, med)
			births = append(births, len(w.People)-bornBefore)
			bornBefore = len(w.People)
		}
	}
	maxChange := 0.0
	for i := 1; i < len(births); i++ {
		maxChange = max(maxChange, math.Abs(float64(births[i]-births[i-1]))/float64(births[i-1]))
	}

	popOK := minPop >= 9000 && maxPop <= 11000
	t.Logf("seed %2d: population %d–%d (final %d) %s | median age %.1f–%.1f %s | births/year %d–%d, largest yearly change %.1f%% | %.0fs",
		seed, minPop, maxPop, len(w.Alive), mark(popOK), minMed, maxMed, mark(minMed >= 30 && maxMed <= 40),
		slices.Min(births), slices.Max(births), 100*maxChange, time.Since(start).Seconds())
}

func mark(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}
