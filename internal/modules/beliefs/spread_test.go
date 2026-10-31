package beliefs

import (
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/initpop"
	"github.com/skrilexxx/silo-sim/internal/modules/network"
	"github.com/skrilexxx/silo-sim/internal/modules/space"
	"github.com/skrilexxx/silo-sim/internal/modules/work"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// still is a day-1 silo of the baseline scenario that doesn't age: only
// the network, beliefs and location steps run, so spread is all that
// changes.
type still struct {
	t    testing.TB
	w    *world.World
	cfg  *config.Scenario
	days []func(*world.World, *rng.Stream) error
	rs   []*rng.Stream

	beliefs *Module
}

// newStill builds it with pass chance π; with noPorters, every porter
// works in General services instead, so nobody has porter ties.
func newStill(t testing.TB, seed uint64, pass float64, noPorters bool) *still {
	cfg, err := config.Load("../../../scenarios/baseline.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Seed, cfg.Beliefs.PassChance = seed, pass
	r, _ := rng.New(seed, "initpop")
	w, err := initpop.Build(cfg, r)
	if err != nil {
		t.Fatal(err)
	}
	s := &still{t: t, w: w, cfg: cfg}
	s.beliefs = New()
	for _, m := range []engine.Module{work.New(), space.New(), network.New(), s.beliefs} {
		mr, _ := rng.New(seed, m.Name())
		if err := m.Init(w, cfg, mr); err != nil {
			t.Fatal(err)
		}
		if m.Name() == "work" && noPorters { // the departments exist from here on
			general, porters := deptIndex(w, "general"), deptIndex(w, "porters")
			for i := range w.People {
				if w.People[i].Department == porters {
					w.People[i].Department = general
				}
			}
		}
		for _, st := range m.Steps() {
			if st.Phase == engine.PhaseNetwork || st.Phase == engine.PhaseBeliefs || st.Phase == engine.PhaseLocation {
				s.days, s.rs = append(s.days, st.Run), append(s.rs, mr)
			}
		}
	}
	return s
}

func deptIndex(w *world.World, name string) int {
	for d, dep := range w.Depts {
		if dep.Name == name {
			return d
		}
	}
	return world.None
}

func (s *still) day() {
	s.t.Helper()
	s.w.Day++
	for i, run := range s.days {
		if err := run(s.w, s.rs[i]); err != nil {
			s.t.Fatal(err)
		}
	}
}

// share is the share of living adults in zone z (or every zone, for
// AllZones) who know the flag.
func (s *still) share(f world.Flag, z world.Zone) float64 {
	adult := world.Tick(s.cfg.Work.WorkerAge * engine.DaysPerYear)
	know, n := 0, 0
	for _, id := range s.w.Alive {
		p := &s.w.People[id]
		if s.w.Day-p.BirthDay < adult || z != world.AllZones && s.w.Floors[s.w.Households[p.Household].Floor-1].Zone != z {
			continue
		}
		n++
		if s.w.Beliefs.Known[id].Has(f) {
			know++
		}
	}
	return float64(know) / float64(n)
}

// seedIn teaches the history to one adult worker, not a porter, living in
// zone z, and returns them.
func (s *still) seedIn(z world.Zone) world.PersonID {
	porters := deptIndex(s.w, "porters")
	for _, id := range s.w.Alive {
		p := &s.w.People[id]
		if p.Stage == world.Worker && p.Department != porters && s.w.Floors[s.w.Households[p.Household].Floor-1].Zone == z {
			s.w.Beliefs.Learn(id, world.FlagHistory, false)
			return id
		}
	}
	s.t.Fatal("nobody to seed")
	return world.None
}

// T10.2 done-when: one seeded knower gives an S-shaped spread curve. With
// nothing forgotten, the share of adults who know rises slowly while
// knowers are few, fastest in the middle, and slowly again near
// saturation: reaching 10% takes longer than going from 10% to 50%, and
// the biggest 30-day rise falls between 10% and 90%.
func TestOneKnowerGivesSCurve(t *testing.T) {
	if testing.Short() {
		t.Skip("full-size silo")
	}
	s := newStill(t, 3, 0.05, false)
	s.seedIn(world.Mids)
	var curve []float64 // share of adults who know, every 30 days
	t10, t50, t90 := -1, -1, -1
	for d := 1; d <= 3000 && t90 < 0; d++ {
		s.day()
		x := s.share(world.FlagHistory, world.AllZones)
		for _, c := range []struct {
			at   *int
			over float64
		}{{&t10, 0.1}, {&t50, 0.5}, {&t90, 0.9}} {
			if *c.at < 0 && x >= c.over {
				*c.at = d
			}
		}
		if d%engine.DaysPerMonth == 0 {
			curve = append(curve, x)
		}
	}
	if t90 < 0 {
		t.Fatalf("90%% not reached in 3,000 days (10%% on day %d, 50%% on day %d)", t10, t50)
	}
	best, bestAt := 0.0, 0
	for i := 1; i < len(curve); i++ {
		if rise := curve[i] - curve[i-1]; rise > best {
			best, bestAt = rise, (i+1)*engine.DaysPerMonth
		}
	}
	t.Logf("10%% on day %d, 50%% on day %d, 90%% on day %d; biggest 30-day rise %.3f, by day %d", t10, t50, t90, best, bestAt)
	if t10 <= t50-t10 {
		t.Errorf("no slow start: 10%% took %d days, 10%%→50%% %d", t10, t50-t10)
	}
	if bestAt <= t10 || bestAt-engine.DaysPerMonth >= t90 {
		t.Errorf("fastest rise by day %d, not between 10%% (day %d) and 90%% (day %d)", bestAt, t10, t90)
	}
}

// T10.2 done-when: through Porters, knowledge crosses zones faster. A
// knower Down Deep; the days until 5% of Up Top's adults know, with and
// without porters (as General services workers instead), over 3 seeds.
func TestPortersCarryKnowledgeAcrossZones(t *testing.T) {
	if testing.Short() {
		t.Skip("full-size silo")
	}
	reach := func(seed uint64, noPorters bool) int {
		s := newStill(t, seed, 0.05, noPorters)
		s.seedIn(world.DownDeep)
		for d := 1; d <= 3000; d++ {
			s.day()
			if s.share(world.FlagHistory, world.UpTop) >= 0.05 {
				return d
			}
		}
		t.Fatalf("seed %d: 5%% of Up Top never reached", seed)
		return 0
	}
	with, without := 0, 0
	for seed := range uint64(3) {
		a, b := reach(seed+1, false), reach(seed+1, true)
		t.Logf("seed %d: 5%% of Up Top on day %d with porters, %d without", seed+1, a, b)
		with, without = with+a, without+b
	}
	if with >= without {
		t.Errorf("with porters %d days in total over 3 seeds, without %d; want faster with them", with, without)
	}
}
