// Package beliefs keeps legitimacy, knowledge and memory (ODD §6.12,
// draft).
package beliefs

import (
	"fmt"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/output"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

type Module struct {
	edu, eduWeight float64
	adultAge       world.Tick       // worker age in days: the base is fixed from then on
	young          []world.PersonID // alive and under worker age at the last step, in ID order
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "beliefs" }

// Requires network: knowledge passes in conversations (ODD §6.11).
func (m *Module) Requires() []string { return []string{"network"} }

// ODD §6.12: everyone alive on day 1 starts with base = L_edu and knows
// nothing.
func (m *Module) Init(w *world.World, cfg *config.Scenario, r *rng.Stream) error {
	if w.Network == nil {
		return fmt.Errorf("beliefs: no network state; list network before beliefs in modules")
	}
	m.edu, m.eduWeight = cfg.Beliefs.TaughtLegitimacy, cfg.Beliefs.EducationWeight
	m.adultAge = world.Tick(cfg.Work.WorkerAge * engine.DaysPerYear)
	w.Beliefs = &world.Beliefs{Cost: cfg.Beliefs.LegitimacyCostPerFlag}
	return m.step(w, r)
}

func (m *Module) Steps() []engine.Step {
	return []engine.Step{{Phase: engine.PhaseBeliefs, Run: m.step}}
}

// ODD §6.12 / §3 step 10.7.
func (m *Module) step(w *world.World, r *rng.Stream) error {
	m.newcomers(w, r)
	if w.Day%engine.DaysPerMonth == 0 {
		m.upbringing(w)
	}
	return nil
}

// ODD §6.12, State: newcomers, in ID order, get a uniform risk aversion R
// and the taught base, which upbringing then adjusts.
func (m *Module) newcomers(w *world.World, r *rng.Stream) {
	b := w.Beliefs
	for id := len(b.Base); id < len(w.People); id++ {
		b.Base = append(b.Base, m.edu)
		b.Known = append(b.Known, 0)
		b.Risk = append(b.Risk, r.Float64())
		m.young = append(m.young, world.PersonID(id))
	}
}

// ODD §6.12, Upbringing: every 30 days until worker age, the base is set
// to ε × L_edu plus (1 − ε) × the mean L of the living parents and, for a
// shadow, the master; L_edu alone with none of them alive. Those are all
// adults, whose bases are fixed, so the visiting order doesn't matter.
// Monthly, not daily: daily it cost about 3% of a run, since elders'
// records are scattered in memory.
func (m *Module) upbringing(w *world.World) {
	b := w.Beliefs
	keep := m.young[:0]
	for _, id := range m.young {
		p := &w.People[id]
		if !p.Alive || w.Day-p.BirthDay >= m.adultAge {
			continue
		}
		keep = append(keep, id)
		sum, n := 0.0, 0
		elders := [3]world.PersonID{p.Mother, p.Father, world.None}
		if p.Stage == world.Shadow {
			elders[2] = p.Master
		}
		for _, e := range elders {
			if e != world.None && w.People[e].Alive {
				sum += b.L(e)
				n++
			}
		}
		if n == 0 {
			b.Base[id] = m.edu
			continue
		}
		b.Base[id] = m.eduWeight*m.edu + (1-m.eduWeight)*sum/float64(n)
	}
	m.young = keep
}

// FrameMetrics: the mean L of the adults living on each floor (ODD §6.12,
// Outputs; 0 where no adult lives).
func (m *Module) FrameMetrics() []output.FrameMetric {
	return []output.FrameMetric{{Name: "legitimacy", Fill: func(w *world.World, v []float64) {
		adults := make([]int, len(v))
		for _, id := range w.Alive {
			if w.Day-w.People[id].BirthDay < m.adultAge {
				continue
			}
			f := w.Households[w.People[id].Household].Floor - 1
			v[f] += w.Beliefs.L(id)
			adults[f]++
		}
		for i, n := range adults {
			if n > 0 {
				v[i] /= float64(n)
			}
		}
	}}}
}
