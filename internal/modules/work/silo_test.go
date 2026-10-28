package work

import (
	"math"
	"testing"
	"time"

	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/initpop"
	"github.com/skrilexxx/silo-sim/internal/invariants"
	"github.com/skrilexxx/silo-sim/internal/modules/demography"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

func dayOne(t *testing.T) (*world.World, *Module) {
	t.Helper()
	cfg := baseline(t)
	r, _ := rng.New(cfg.Seed, "initpop")
	w, err := initpop.Build(cfg, r)
	if err != nil {
		t.Fatal(err)
	}
	m := New()
	if err := m.Init(w, cfg, stream(t)); err != nil {
		t.Fatal(err)
	}
	if err := invariants.Check(w, cfg); err != nil {
		t.Fatal(err)
	}
	return w, m
}

// ODD §5 steps 5–6: day-1 stages by age, workers only in departments of
// their zone, headcounts at target, shadows with masters, skill on the
// saturating curve.
func TestDayOneJobs(t *testing.T) {
	w, m := dayOne(t)
	c := m.c
	workers, masterless := 0, 0
	for _, id := range w.Alive {
		p := &w.People[id]
		a := age(w, id)
		want := world.Child
		switch {
		case a >= m.days(c.RetireAge):
			want = world.Retired
		case a >= m.days(c.WorkerAge):
			want = world.Worker
		case a >= m.days(c.ShadowAge):
			want = world.Shadow
		}
		if p.Stage != want {
			t.Fatalf("person %d aged %d days: stage %d, want %d", id, a, p.Stage, want)
		}
		switch p.Stage {
		case world.Worker:
			workers++
			if z := w.Depts[p.Department].Zone; z != world.AllZones && z != homeZone(w, id) {
				t.Fatalf("worker %d lives in zone %d but works in %s", id, homeZone(w, id), w.Depts[p.Department].Name)
			}
		case world.Shadow:
			if p.Master == world.None {
				masterless++
			}
		case world.Retired:
			if p.Department != world.None {
				t.Errorf("day-1 retiree %d has a department", id)
			}
		}
		if learned := min(a, m.days(c.RetireAge)) - m.days(c.ShadowAge); learned > 0 {
			if want := 1 - math.Pow(1-c.SkillGrowth, float64(learned)); math.Abs(p.Skill-want) > 1e-12 {
				t.Fatalf("person %d: skill %v, want %v", id, p.Skill, want)
			}
		}
	}
	for _, dep := range w.Depts {
		target := dep.TargetShare * float64(workers)
		t.Logf("%-15s %4d workers, target %6.1f, %3d shadows", dep.Name, dep.Headcount, target, dep.Shadows)
		if math.Abs(float64(dep.Headcount)-target) > 0.05*target+1 {
			t.Errorf("%s: %d workers on day 1, target %.1f", dep.Name, dep.Headcount, target)
		}
	}
	if masterless > 0 {
		t.Errorf("%d day-1 shadows without a master", masterless)
	}
}

// T2.1 done-when, over 50 years of the full silo with demography and work,
// invariants every tick (a master never has more than 2 shadows): every
// department within ±15% of its target at the end. The share of workers in
// a parent's department is an outcome, logged: it is higher than the 60%
// follow chance (TestFollowParent tests that) because the other 40% of
// placements often land in a parent's department too. It must at least
// reach the follow chance.
func TestFiftyYears(t *testing.T) {
	cfg := baseline(t)
	cfg.WarmupYears, cfg.Years = 0, 50
	r, _ := rng.New(cfg.Seed, "initpop")
	w, err := initpop.Build(cfg, r)
	if err != nil {
		t.Fatal(err)
	}
	e, err := engine.New(cfg, w, []engine.Module{demography.New(), New()})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if err := e.Run(); err != nil {
		t.Fatal(err)
	}

	workers := 0
	for _, d := range w.Depts {
		workers += d.Headcount
	}
	worst := 0.0
	for _, dep := range w.Depts {
		target := dep.TargetShare * float64(workers)
		dev := float64(dep.Headcount)/target - 1
		worst = max(worst, math.Abs(dev))
		t.Logf("%-15s %4d workers, target %6.1f (%+.1f%%), %3d shadows", dep.Name, dep.Headcount, target, 100*dev, dep.Shadows)
		if math.Abs(dev) > 0.15 {
			t.Errorf("%s: %d workers, target %.1f (%+.1f%%)", dep.Name, dep.Headcount, target, 100*dev)
		}
	}

	withParent, inParent := 0, 0
	for _, id := range w.Alive {
		p := &w.People[id]
		if p.Stage != world.Worker {
			continue
		}
		var pd []int
		for _, par := range []world.PersonID{p.Mother, p.Father} {
			if par != world.None && w.People[par].Department != world.None {
				pd = append(pd, w.People[par].Department)
			}
		}
		if len(pd) == 0 {
			continue
		}
		withParent++
		for _, d := range pd {
			if d == p.Department {
				inParent++
				break
			}
		}
	}
	share := float64(inParent) / float64(withParent)
	t.Logf("after 50 years: %d alive, %d workers; largest department deviation %.1f%%; %d of %d workers with a parent's trade on record work in it (%.1f%%); %.0fs",
		len(w.Alive), workers, 100*worst, inParent, withParent, 100*share, time.Since(start).Seconds())
	if share < cfg.Work.FollowParentChance {
		t.Errorf("%.1f%% of workers in a parent's department, below the %.0f%% follow chance", 100*share, 100*cfg.Work.FollowParentChance)
	}
}
