package demography

import (
	"fmt"
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

// ODD §6.1: mortality. Each living person, in this tick's shuffled order,
// dies with DeathChance for their age and health. Each death is handled at
// once (ODD §3, asynchronous updating), so a later death in the same tick
// sees its effects.
func (m *Module) deaths(w *world.World, r *rng.Stream) error {
	for _, id := range w.Alive {
		p := &w.People[id]
		age := float64(w.Day-p.BirthDay) / engine.DaysPerYear
		if r.Bernoulli(DeathChance(age, p.Health, m.cfg.Mortality)) {
			if err := m.die(w, r, id); err != nil {
				return err
			}
		}
	}
	// Removed after the loop, not during it, so the loop's order is the
	// tick's shuffled order throughout.
	w.Alive = slices.DeleteFunc(w.Alive, func(id world.PersonID) bool { return !w.People[id].Alive })
	return nil
}

// ODD §6.1: on a death the partner becomes unpartnered and may pair again;
// a widow's open conception window closes (ODD §6.4), though a pregnancy
// already under way goes on.
// If an adult still lives in the household, the children stay with them
// (their other parent, or a step-parent). Otherwise the children move
// together to a random household in the same zone with a living adult. A
// household left empty is dissolved, which frees its home.
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
		if q.Implant == world.WindowOpen {
			q.Implant = world.ImplantIn
		}
	}

	hh := &w.Households[p.Household]
	hh.Members = slices.DeleteFunc(hh.Members, func(x world.PersonID) bool { return x == id })
	if len(hh.Members) > 0 && !m.hasAdult(w, hh) {
		to, err := m.fosterHome(w, r, hh)
		if err != nil {
			return err
		}
		for _, kid := range hh.Members {
			w.People[kid].Household = to
		}
		w.Households[to].Members = append(w.Households[to].Members, hh.Members...)
		hh.Members = nil
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
// least one living adult.
func (m *Module) fosterHome(w *world.World, r *rng.Stream, from *world.Household) (world.HouseholdID, error) {
	zone := w.Floors[from.Floor-1].Zone
	var homes []world.HouseholdID
	for i := range w.Households {
		h := &w.Households[i]
		if h.ID != from.ID && h.Floor != world.None && w.Floors[h.Floor-1].Zone == zone && m.hasAdult(w, h) {
			homes = append(homes, h.ID)
		}
	}
	if len(homes) == 0 {
		return 0, fmt.Errorf("demography: orphans of household %d have no household with an adult in zone %d", from.ID, zone)
	}
	return homes[r.IntRange(0, len(homes)-1)], nil
}
