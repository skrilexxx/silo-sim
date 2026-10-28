// Package work holds departments, life stages, shadowing and skill (ODD
// §6.5). Output, coverage and retraining arrive in T2.2.
package work

import (
	"math"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

type Module struct {
	cfg *config.Scenario
	c   config.Work

	// target headcount per department, from the monthly staffing step
	// (ODD §3 step 14). Placements in between compare it with live counts.
	target []float64

	// Reused buffers.
	load   []int // shadows per master, indexed by PersonID
	cands  []int
	ws     []float64
	people []world.PersonID
}

func New() *Module { return &Module{} }

func (m *Module) Name() string       { return "work" }
func (m *Module) Requires() []string { return nil }

func (m *Module) Steps() []engine.Step {
	return []engine.Step{
		{Phase: engine.PhaseWork, Run: m.skillGrowth},
		{Phase: engine.PhaseLifeStages, Run: m.lifeStages},
		{Phase: engine.PhaseStaffing, Run: m.staffing},
	}
}

// Init builds the department table and gives day-1 jobs and skill (ODD §5
// steps 5 and 6). It lives here, not in initpop, so a run without the work
// module has no jobs at all.
func (m *Module) Init(w *world.World, cfg *config.Scenario, r *rng.Stream) error {
	m.cfg, m.c = cfg, cfg.Work
	w.Depts = w.Depts[:0]
	for _, d := range m.c.Departments {
		w.Depts = append(w.Depts, world.Department{Name: d.Name, Zone: parseZone(d.Zone), TargetShare: d.Share})
	}
	m.dayOneWorkers(w, r)
	m.dayOneSkill(w)
	m.recount(w)
	_ = m.staffing(w, r)
	m.dayOneShadows(w, r)
	return nil
}

func parseZone(z string) world.Zone {
	switch z {
	case "upTop":
		return world.UpTop
	case "mids":
		return world.Mids
	case "downDeep":
		return world.DownDeep
	}
	return world.AllZones // config validation allows only these four
}

func (m *Module) days(years int) world.Tick { return world.Tick(years * engine.DaysPerYear) }

func age(w *world.World, id world.PersonID) world.Tick { return w.Day - w.People[id].BirthDay }

// homeZone is the zone of a living person's household.
func homeZone(w *world.World, id world.PersonID) world.Zone {
	return w.Floors[w.Households[w.People[id].Household].Floor-1].Zone
}

// ODD §5 step 5: in each zone, adults aged workerAge to retireAge fill the
// departments of that zone, plus the all-zone departments, by target share.
// An all-zone department's share is split over the zones by allZoneSplit.
// Quotas are rounded to whole people by largest remainder; who goes where
// is random. Older people are retired with no department.
func (m *Module) dayOneWorkers(w *world.World, r *rng.Stream) {
	split := [world.NumZones]float64{m.c.AllZoneSplit.UpTop, m.c.AllZoneSplit.Mids, m.c.AllZoneSplit.DownDeep}
	for z := range world.Zone(world.NumZones) {
		m.people = m.people[:0]
		for _, id := range w.Alive {
			a := age(w, id)
			switch {
			case a >= m.days(m.c.RetireAge):
				w.People[id].Stage = world.Retired
			case a >= m.days(m.c.WorkerAge) && homeZone(w, id) == z:
				m.people = append(m.people, id)
			}
		}
		r.Shuffle(len(m.people), func(i, j int) { m.people[i], m.people[j] = m.people[j], m.people[i] })

		weight := make([]float64, len(w.Depts))
		total := 0.0
		for d, dep := range w.Depts {
			switch dep.Zone {
			case z:
				weight[d] = dep.TargetShare
			case world.AllZones:
				weight[d] = dep.TargetShare * split[z]
			}
			total += weight[d]
		}
		quota := largestRemainder(weight, total, len(m.people))
		next := 0
		for d, q := range quota {
			for _, id := range m.people[next : next+q] {
				w.People[id].Stage, w.People[id].Department = world.Worker, d
			}
			next += q
		}
	}
}

// largestRemainder splits n into whole parts proportional to weight.
func largestRemainder(weight []float64, total float64, n int) []int {
	q := make([]int, len(weight))
	given := 0
	for i, x := range weight {
		q[i] = int(x / total * float64(n))
		given += q[i]
	}
	for ; given < n; given++ {
		best := 0
		for i := range weight {
			if weight[i]/total*float64(n)-float64(q[i]) > weight[best]/total*float64(n)-float64(q[best]) {
				best = i
			}
		}
		q[best]++
	}
	return q
}

// ODD §5 step 6: day-1 skill follows the same saturating curve as daily
// growth (ODD §6.5), counted from the 14th birthday because shadows learn
// too: skill = 1 − (1 − α)^(days since shadowAge), stopping at retireAge.
func (m *Module) dayOneSkill(w *world.World) {
	for _, id := range w.Alive {
		learned := min(age(w, id), m.days(m.c.RetireAge)) - m.days(m.c.ShadowAge)
		if learned > 0 {
			w.People[id].Skill = 1 - math.Pow(1-m.c.SkillGrowth, float64(learned))
		}
	}
}

// ODD §5 step 5: people aged shadowAge to workerAge on day 1 are placed by
// the same shadowing rule as a child turning 14 (ODD §6.5), in random order.
func (m *Module) dayOneShadows(w *world.World, r *rng.Stream) {
	m.people = m.people[:0]
	for _, id := range w.Alive {
		if a := age(w, id); a >= m.days(m.c.ShadowAge) && a < m.days(m.c.WorkerAge) {
			m.people = append(m.people, id)
		}
	}
	kids := append([]world.PersonID(nil), m.people...)
	r.Shuffle(len(kids), func(i, j int) { kids[i], kids[j] = kids[j], kids[i] })
	for _, id := range kids {
		m.place(w, r, id)
	}
}

// ODD §6.5: skill grows each working day by α × (1 − skill), for workers
// and shadows, so it approaches 1 and never passes it.
func (m *Module) skillGrowth(w *world.World, _ *rng.Stream) error {
	for _, id := range w.Alive {
		p := &w.People[id]
		if p.Stage == world.Shadow || p.Stage == world.Worker {
			p.Skill += m.c.SkillGrowth * (1 - p.Skill)
		}
	}
	return nil
}

// ODD §6.5 / §3 step 14: staffing, every 30 days. Each department's target
// is its share of the current workers. Placements until the next month
// use these targets with live counts (see shortfall), so every shadow
// placed shrinks the gap at once.
func (m *Module) staffing(w *world.World, _ *rng.Stream) error {
	m.recount(w)
	workers := 0
	for _, d := range w.Depts {
		workers += d.Headcount
	}
	m.target = m.target[:0]
	for _, d := range w.Depts {
		m.target = append(m.target, d.TargetShare*float64(workers))
	}
	return nil
}

// shortfall counts shadows as future workers: max(0, target − workers − shadows).
func (m *Module) shortfall(w *world.World, d int) float64 {
	return max(0, m.target[d]-float64(w.Depts[d].Headcount+w.Depts[d].Shadows))
}

// recount refreshes department headcounts and shadows per master from the
// living. Deaths are handled by demography, so a dead worker's job is
// vacated here, at the next count (ODD §6.1).
func (m *Module) recount(w *world.World) {
	for d := range w.Depts {
		w.Depts[d].Headcount, w.Depts[d].Shadows = 0, 0
	}
	m.load = append(m.load[:0], make([]int, len(w.People))...)
	for _, id := range w.Alive {
		p := &w.People[id]
		switch p.Stage {
		case world.Worker:
			w.Depts[p.Department].Headcount++
		case world.Shadow:
			w.Depts[p.Department].Shadows++
			if p.Master != world.None {
				m.load[p.Master]++
			}
		}
	}
}
