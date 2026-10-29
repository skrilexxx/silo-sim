package health

import (
	"math"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/initpop"
	"github.com/skrilexxx/silo-sim/internal/modules/demography"
	"github.com/skrilexxx/silo-sim/internal/modules/work"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

const year = engine.DaysPerYear

// Department indexes in the baseline table.
const farms, general, mechanical, medical = 0, 1, 2, 7

func baseline(t *testing.T) *config.Scenario {
	t.Helper()
	cfg, err := config.Load("../../../scenarios/baseline.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Output.InvariantEvery = 1
	return cfg
}

// crowd builds n living people of one stage and department, with the
// baseline department table and Medical coverage cov.
func crowd(t *testing.T, n int, stage world.Stage, dept int, cov float64) (*world.World, *Module) {
	t.Helper()
	cfg := baseline(t)
	w := &world.World{
		Silo:       world.Silo{AirQuality: 1, WaterSupply: 1, Intake: [3]float64{1, 1, 1}},
		Floors:     []world.Floor{{Number: 1, Zone: world.UpTop}},
		Households: []world.Household{{Floor: 1}},
	}
	for _, d := range cfg.Work.Departments {
		w.Depts = append(w.Depts, world.Department{Name: d.Name})
	}
	w.Depts[medical].Coverage = cov
	for i := range n {
		w.People = append(w.People, world.Person{ID: world.PersonID(i), Alive: true, Stage: stage, Department: dept, Health: 1})
		w.Alive = append(w.Alive, world.PersonID(i))
	}
	m := New()
	m.Init(w, cfg, nil)
	return w, m
}

// hitRate runs one day for a crowd at health 1 with no recovery and
// returns the share whose health dropped.
func hitRate(t *testing.T, w *world.World, m *Module) float64 {
	t.Helper()
	r, _ := rng.New(1, "health")
	if err := m.health(w, r); err != nil {
		t.Fatal(err)
	}
	hit := 0
	for _, id := range w.Alive {
		if w.People[id].Health < 1 {
			hit++
		}
	}
	return float64(hit) / float64(len(w.Alive))
}

// Daily injury chance is injuryChance × the department factor (Mechanical
// ×2, Farms ×1, others ×0.5); illness adds illnessChance for everyone.
// Children are not injured. 200,000 people, one day, no Medical.
func TestInjuryAndIllnessRates(t *testing.T) {
	c := baseline(t).Health
	cases := []struct {
		what   string
		stage  world.Stage
		dept   int
		factor float64
	}{
		{"Mechanical worker", world.Worker, mechanical, c.InjuryFactor.Mechanical},
		{"Farms shadow", world.Shadow, farms, c.InjuryFactor.Farms},
		{"General worker", world.Worker, general, c.InjuryFactor.Other},
		{"child", world.Child, world.None, 0},
	}
	const n = 200_000
	for _, k := range cases {
		w, m := crowd(t, n, k.stage, k.dept, 0)
		got := hitRate(t, w, m)
		pi, pl := c.InjuryChance*k.factor, c.IllnessChance
		want := 1 - (1-pi)*(1-pl)
		sd := math.Sqrt(want * (1 - want) / n)
		t.Logf("%s: %.5f hurt or ill in a day (expected %.5f ± %.5f)", k.what, got, want, sd)
		if math.Abs(got-want) > 4*sd {
			t.Errorf("%s: rate %.5f, want %.5f", k.what, got, want)
		}
	}
}

// Recovery is recoveryRate × Medical coverage, capped at 1; injuries cost
// injuryCost; bad air costs airDamage × (1 − air).
func TestHealthArithmetic(t *testing.T) {
	c := baseline(t).Health
	w, m := crowd(t, 1, world.Retired, world.None, 0.5)
	m.c.IllnessChance = 0
	w.People[0].Health = 0.5
	w.Silo.AirQuality = 0.6
	r, _ := rng.New(1, "health")
	m.health(w, r)
	want := 0.5 - c.AirDamage*0.4 + c.RecoveryRate*0.5
	if got := w.People[0].Health; math.Abs(got-want) > 1e-12 {
		t.Errorf("health %v, want %v", got, want)
	}

	w, m = crowd(t, 1, world.Worker, mechanical, 1)
	m.c.IllnessChance, m.c.InjuryChance = 0, 1 // injured for sure
	w.People[0].Health = 0.2
	m.health(w, r)
	if got := w.People[0].Health; got != 0 {
		t.Errorf("health %v after an injury from 0.2, want 0 (clamped)", got)
	}
	w.People[0].Health = 0.995
	m.c.InjuryChance = 0
	m.health(w, r)
	if got := w.People[0].Health; got != 1 {
		t.Errorf("health %v after recovery from 0.995, want 1 (capped)", got)
	}
}

// noMedical runs the work module but forces Medical coverage to 0 after
// each work step: "Medical removed" without touching the config rules.
type noMedical struct{ *work.Module }

func (n noMedical) Steps() []engine.Step {
	steps := n.Module.Steps()
	for i, s := range steps {
		if s.Phase == engine.PhaseWork {
			run := s.Run
			steps[i].Run = func(w *world.World, r *rng.Stream) error {
				err := run(w, r)
				w.Depts[medical].Coverage = 0
				return err
			}
		}
	}
	return steps
}

type result struct {
	minHealth, endHealth float64
	deaths, sickRetired  int
}

// run plays the full silo for the given years with demography, health and
// the work module (wrapped or not), invariants every tick.
func run(t *testing.T, years int, workModule engine.Module) result {
	t.Helper()
	cfg := baseline(t)
	cfg.WarmupYears, cfg.Years = 0, years
	r, _ := rng.New(cfg.Seed, "initpop")
	w, err := initpop.Build(cfg, r)
	if err != nil {
		t.Fatal(err)
	}
	e, err := engine.New(cfg, w, []engine.Module{demography.New(), New(), workModule})
	if err != nil {
		t.Fatal(err)
	}
	avg := func() float64 {
		s := 0.0
		for _, id := range w.Alive {
			s += w.People[id].Health
		}
		return s / float64(len(w.Alive))
	}
	res := result{minHealth: 1}
	for w.Day < world.Tick(years*year) {
		if err := e.Tick(); err != nil {
			t.Fatal(err)
		}
		if w.Day%year == 0 {
			res.minHealth = min(res.minHealth, avg())
		}
	}
	res.endHealth = avg()
	for i := range w.People {
		p := &w.People[i]
		if !p.Alive {
			res.deaths++
		} else if p.Stage == world.Retired && w.Day-p.BirthDay < world.Tick(cfg.Work.RetireAge*year) {
			res.sickRetired++
		}
	}
	return res
}

// T2.3 done-when: with full Medical, average health stays above 0.9; with
// Medical removed, average health falls and deaths rise. Same seed, 20
// years each.
func TestMedicalMatters(t *testing.T) {
	full := run(t, 20, work.New())
	none := run(t, 20, noMedical{work.New()})
	t.Logf("full Medical: average health min %.3f, end %.3f; %d deaths; %d retired early for ill health", full.minHealth, full.endHealth, full.deaths, full.sickRetired)
	t.Logf("no Medical:   average health min %.3f, end %.3f; %d deaths; %d retired early for ill health", none.minHealth, none.endHealth, none.deaths, none.sickRetired)
	if full.minHealth <= 0.9 {
		t.Errorf("with full Medical average health fell to %.3f, want above 0.9", full.minHealth)
	}
	if none.endHealth >= full.endHealth || none.deaths <= full.deaths {
		t.Errorf("without Medical: health %.3f vs %.3f, deaths %d vs %d; want lower health and more deaths",
			none.endHealth, full.endHealth, none.deaths, full.deaths)
	}
}

// Hunger costs hungerDamage × (1 − intake/need) a day; food above need
// gives nothing.
func TestHunger(t *testing.T) {
	for _, tc := range []struct{ intake, want float64 }{{0.5, 1 - 0.5*baseline(t).Health.HungerDamage}, {0, 1 - baseline(t).Health.HungerDamage}, {1.5, 1}} {
		w, m := crowd(t, 10, world.Child, world.None, 0)
		m.c.IllnessChance = 0
		w.Silo.Intake[world.UpTop] = tc.intake
		r, _ := rng.New(1, "health")
		if err := m.health(w, r); err != nil {
			t.Fatal(err)
		}
		for _, id := range w.Alive {
			if h := w.People[id].Health; math.Abs(h-tc.want) > 1e-12 {
				t.Fatalf("intake/need %v: health %v, want %v", tc.intake, h, tc.want)
			}
		}
	}
}
