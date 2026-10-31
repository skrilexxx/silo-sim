package demography

import (
	"fmt"
	"math"
	"slices"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// Module is the demography module: deaths (§6.1), partnering and housing
// (§6.3), lottery and reproduction (§6.4).
type Module struct {
	cfg      *config.Scenario
	deaths30 int // deaths since the last lottery (D30 in ODD §6.4)

	// healthy[d] is DeathChance at health 1 for someone d days old, built
	// as ages are reached. Most people are at health exactly 1, and the
	// table gives them the very same number without the two exp calls.
	healthy []float64

	// Reused buffers, so the daily steps allocate nothing.
	pool    []world.PersonID
	poolZ   []world.Zone
	weights []float64
}

func New() *Module { return &Module{} }

func (m *Module) Name() string       { return "demography" }
func (m *Module) Requires() []string { return nil }

func (m *Module) Init(_ *world.World, cfg *config.Scenario, _ *rng.Stream) error {
	m.cfg = cfg
	return nil
}

func (m *Module) Steps() []engine.Step {
	return []engine.Step{
		{Phase: engine.PhaseDeaths, Run: m.deaths},
		{Phase: engine.PhaseReproduction, Run: m.reproduction},
		{Phase: engine.PhasePartnering, Run: m.partnering},
		{Phase: engine.PhaseLottery, Run: m.lottery},
		{Phase: engine.PhaseHousing, Run: m.housing},
	}
}

// EventTypes: ODD §6.1 (part 2, draft), cull. A share of the living
// people, or of a department's workers, drawn uniformly, die just before
// the day's deaths step, each handled as any death. It starts runs from the
// damaged states that measure the point of no return P_min (§1.2, T12.6).
func (m *Module) EventTypes() []engine.EventType {
	return []engine.EventType{{Name: "cull", Phase: engine.PhaseDeaths, Fire: m.cull}}
}

func (m *Module) cull(w *world.World, r *rng.Stream, ev config.Event) error {
	pool := slices.Clone(w.Alive)
	if ev.Department != "" {
		d := slices.IndexFunc(w.Depts, func(dep world.Department) bool { return dep.Name == ev.Department })
		if d < 0 {
			return fmt.Errorf("cull: no department %q", ev.Department)
		}
		pool = slices.DeleteFunc(pool, func(id world.PersonID) bool {
			p := &w.People[id]
			return p.Stage != world.Worker || p.Department != d
		})
	}
	before := m.deaths30
	for i := range int(math.Round(ev.Share * float64(len(pool)))) {
		j := r.IntRange(i, len(pool)-1)
		pool[i], pool[j] = pool[j], pool[i]
		if err := m.die(w, r, pool[i]); err != nil {
			return err
		}
	}
	m.removeDead(w, before)
	return nil
}

// ODD §6.1: mortality. Each living person, in this tick's shuffled order,
// dies with DeathChance for their age and health. Each death is handled at
// once (ODD §3, asynchronous updating), so a later death in the same tick
// sees its effects.
func (m *Module) deaths(w *world.World, r *rng.Stream) error {
	before := m.deaths30
	for _, id := range w.Alive {
		p := &w.People[id]
		if r.Bernoulli(m.deathChance(w.Day-p.BirthDay, p.Health)) {
			if err := m.die(w, r, id); err != nil {
				return err
			}
		}
	}
	// ODD §6.13: the condemned go out to clean today, after the day's
	// natural deaths, in the order they were sentenced.
	if c := w.Control; c != nil {
		for _, id := range c.Doomed {
			if w.People[id].Alive {
				if err := m.die(w, r, id); err != nil {
					return err
				}
			}
		}
		c.Doomed = c.Doomed[:0]
	}
	// Removed after the loop, not during it, so the loop's order is the
	// tick's shuffled order throughout.
	m.removeDead(w, before)
	// Part 2 (ODD §1.2): the last death ends the run, whether or not a
	// collapse ended it first.
	if len(w.Alive) == 0 && !w.Over {
		w.Events = append(w.Events, world.EventHeader{Day: w.Day, Type: "extinction"})
		w.Over = true
	}
	return nil
}

// removeDead drops the dead from Alive, if anyone died since deaths30 was
// before (die counts every death there).
func (m *Module) removeDead(w *world.World, before int) {
	if m.deaths30 != before {
		w.Alive = slices.DeleteFunc(w.Alive, func(id world.PersonID) bool { return !w.People[id].Alive })
	}
}

// deathChance is DeathChance for an age in days, from the table at
// health 1.
func (m *Module) deathChance(days world.Tick, health float64) float64 {
	if health != 1 {
		return DeathChance(float64(days)/engine.DaysPerYear, health, m.cfg.Mortality)
	}
	for int(days) >= len(m.healthy) {
		m.healthy = append(m.healthy, DeathChance(float64(len(m.healthy))/engine.DaysPerYear, 1, m.cfg.Mortality))
	}
	return m.healthy[days]
}

// ODD §6.1: on a death the partner becomes unpartnered and may pair again;
// a widow's open conception window closes, or her permit is lost if she is
// still waiting for the removal (ODD §6.4), though a pregnancy already
// under way goes on.
// If an adult still lives in the household, the children stay with them
// (their other parent, or a step-parent). Otherwise the children move
// together to a random household in the same zone with a living adult, or
// in any zone if their own has none; with no adult anywhere they stay in
// their home (part 2: a dead silo, which §1.2 ends). A household left empty
// is dissolved, which frees its home.
//
// The dead keep their Partner and Household fields as a record; the
// invariants only follow references from the living.
func (m *Module) die(w *world.World, r *rng.Stream, id world.PersonID) error {
	p := &w.People[id]
	p.Alive = false
	m.deaths30++
	if p.Partner != world.None {
		q := &w.People[p.Partner]
		q.Partner = world.None
		if q.Implant == world.WindowOpen || q.Implant == world.Queued {
			q.Implant = world.ImplantIn
		}
	}

	hh := &w.Households[p.Household]
	hh.Members = slices.DeleteFunc(hh.Members, func(x world.PersonID) bool { return x == id })
	if len(hh.Members) > 0 && !m.hasAdult(w, hh) {
		if to, ok := m.fosterHome(w, r, hh); ok {
			for _, kid := range hh.Members {
				w.People[kid].Household = to
			}
			w.Households[to].Members = append(w.Households[to].Members, hh.Members...)
			hh.Members = nil
		}
	}
	if len(hh.Members) == 0 {
		hh.Floor = world.None
	}
	return nil
}

func (m *Module) hasAdult(w *world.World, hh *world.Household) bool {
	adult := world.Tick(m.cfg.Partnering.MinAge * engine.DaysPerYear)
	for _, id := range hh.Members {
		if w.Day-w.People[id].BirthDay >= adult {
			return true
		}
	}
	return false
}

// fosterHome picks, uniformly, another household in from's zone with at
// least one living adult. If the zone has none (a mass death), it picks
// from every zone instead; false if there is no adult left at all.
func (m *Module) fosterHome(w *world.World, r *rng.Stream, from *world.Household) (world.HouseholdID, bool) {
	zone := w.Floors[from.Floor-1].Zone
	var homes []world.HouseholdID
	for _, anyZone := range []bool{false, true} {
		for i := range w.Households {
			h := &w.Households[i]
			if h.ID != from.ID && h.Floor != world.None && (anyZone || w.Floors[h.Floor-1].Zone == zone) && m.hasAdult(w, h) {
				homes = append(homes, h.ID)
			}
		}
		if len(homes) > 0 {
			return homes[r.IntRange(0, len(homes)-1)], true
		}
	}
	return 0, false
}
