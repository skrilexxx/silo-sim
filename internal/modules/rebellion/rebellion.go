// Package rebellion is grievance and rebellion (ODD §6.14, draft).
package rebellion

import (
	"fmt"
	"math"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

type Module struct {
	loss      float64    // ℓ: hardship of a loss on its day
	lossDays  float64    // T_loss in days
	scapegoat float64    // s_DD
	adultAge  world.Tick // worker age in days (ODD §6.5)
	buf       []world.Tie
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "rebellion" }

// Requires control: grievance reads the losses it reports (cleanings,
// denied wins), and through it beliefs (L) and the network (family).
func (m *Module) Requires() []string { return []string{"control"} }

func (m *Module) Init(w *world.World, cfg *config.Scenario, r *rng.Stream) error {
	if w.Control == nil {
		return fmt.Errorf("rebellion: no control state; list control before rebellion in modules")
	}
	m.loss, m.scapegoat = cfg.Rebellion.LossHardship, cfg.Rebellion.ScapegoatHardship
	m.lossDays = cfg.Rebellion.LossHalfLifeYears * engine.DaysPerYear
	m.adultAge = world.Tick(cfg.Work.WorkerAge * engine.DaysPerYear)
	w.Rebellion = &world.Rebellion{}
	m.newcomers(w)
	return nil
}

func (m *Module) Steps() []engine.Step {
	return []engine.Step{{Phase: engine.PhaseRebellion, Run: m.step}}
}

// ODD §6.14 / §3 step 10.9.
func (m *Module) step(w *world.World, r *rng.Stream) error {
	m.newcomers(w)
	m.losses(w)
	m.hardship(w)
	return nil
}

func (m *Module) newcomers(w *world.World) {
	b := w.Rebellion
	for len(b.LossDay) < len(w.People) {
		b.LossDay, b.H, b.G = append(b.LossDay, world.None), append(b.H, 0), append(b.G, 0)
	}
}

// ODD §6.14, Hardship: losses. The family of each cleaner who went out
// today (partner, household and kin: the strong ties of §6.11 other than
// master–shadow), and both partners of each denied lottery win, have a
// loss today. The dead keep their records (ODD §6.1), so the cleaner's
// family still follows from them.
func (m *Module) losses(w *world.World) {
	b, c := w.Rebellion, w.Control
	for _, id := range c.Cleaned {
		m.buf = w.Network.Ties(w, id, m.buf[:0])
		for _, t := range m.buf {
			if t.Kind <= world.TieKin {
				b.LossDay[t.Other] = w.Day
			}
		}
	}
	for _, id := range c.DeniedToday {
		if !w.People[id].Alive {
			continue
		}
		b.LossDay[id] = w.Day
		if p := w.People[id].Partner; p != world.None {
			b.LossDay[p] = w.Day
		}
	}
}

// ODD §6.14, Hardship and Grievance: H combines food, health, loss and
// the Down Deep like independent chances, so it stays within 0–1 without
// a clamp; G = H × (1 − L). Children have neither.
func (m *Module) hardship(w *world.World) {
	b := w.Rebellion
	for _, id := range w.Alive {
		p := &w.People[id]
		if w.Day-p.BirthDay < m.adultAge {
			b.H[id], b.G[id] = 0, 0
			continue
		}
		zone := w.Floors[w.Households[p.Household].Floor-1].Zone
		calm := (1 - max(0, 1-w.Silo.Intake[zone])) * p.Health
		if d := b.LossDay[id]; d != world.None {
			calm *= 1 - m.loss*math.Exp2(-float64(w.Day-d)/m.lossDays)
		}
		if zone == world.DownDeep {
			calm *= 1 - m.scapegoat
		}
		b.H[id] = 1 - calm
		b.G[id] = b.H[id] * (1 - w.Beliefs.L(id))
	}
}
