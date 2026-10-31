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
	pass           float64          // π: chance a flag passes in a conversation, × (1 − L of the listener)
	adultAge       world.Tick       // worker age in days: the base is fixed from then on
	young          []world.PersonID // alive and under worker age at the last step, in ID order

	tellers []world.PersonID // today's knowers, shuffled
	fresh   []world.Flags    // flags learned today, not passed on before tomorrow
	learned []world.PersonID // who has fresh flags
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
	m.pass = cfg.Beliefs.PassChance
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
	m.converse(w, r)
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
		m.fresh = append(m.fresh, 0)
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

// ODD §6.12, Telling and believing: the day's knowers, in shuffled order,
// ask of each tie whether it is used today (ODD §6.11); in a conversation
// each flag the teller knew before today and the listener doesn't know
// passes with chance π × (1 − L of the listener), whose L drops as they
// learn. Only knowers are visited, so the cost follows them, not the
// population.
func (m *Module) converse(w *world.World, r *rng.Stream) {
	b, n := w.Beliefs, w.Network
	m.tellers = m.tellers[:0]
	keep := b.Knowers[:0]
	for _, id := range b.Knowers {
		if w.People[id].Alive && b.Known[id] != 0 {
			keep = append(keep, id)
		}
	}
	b.Knowers = keep
	m.tellers = append(m.tellers, keep...)
	r.Shuffle(len(m.tellers), func(i, j int) { m.tellers[i], m.tellers[j] = m.tellers[j], m.tellers[i] })

	for _, k := range m.tellers {
		tells := b.Known[k] &^ m.fresh[k]
		for _, t := range n.TiesOf(w, k) {
			j := t.Other
			news := tells &^ b.Known[j]
			if news == 0 || !n.Talk(w, k, j, t.Kind) {
				continue
			}
			for f := range world.NumFlags {
				if news.Has(f) && r.Float64() < m.pass*(1-b.L(j)) {
					m.learn(b, j, f)
				}
			}
		}
	}
	for _, id := range m.learned {
		m.fresh[id] = 0
	}
	m.learned = m.learned[:0]
}

// learn teaches a flag in a conversation; it can be passed on from tomorrow.
func (m *Module) learn(b *world.Beliefs, id world.PersonID, f world.Flag) {
	b.Learn(id, f)
	if m.fresh[id] == 0 {
		m.learned = append(m.learned, id)
	}
	m.fresh[id] |= 1 << f
}

// FrameMetrics: per floor, the mean L of the adults living there and the
// share of them who know the history (ODD §6.12, Outputs; 0 where no
// adult lives).
func (m *Module) FrameMetrics() []output.FrameMetric {
	return []output.FrameMetric{
		{Name: "legitimacy", Fill: m.adultMean(func(b *world.Beliefs, id world.PersonID) float64 { return b.L(id) })},
		{Name: "knowsHistory", Fill: m.adultMean(func(b *world.Beliefs, id world.PersonID) float64 {
			if b.Known[id].Has(world.FlagHistory) {
				return 1
			}
			return 0
		})},
	}
}

// adultMean fills each floor with the mean of x over the adults living there.
func (m *Module) adultMean(x func(*world.Beliefs, world.PersonID) float64) func(*world.World, []float64) {
	return func(w *world.World, v []float64) {
		adults := make([]int, len(v))
		for _, id := range w.Alive {
			if w.Day-w.People[id].BirthDay < m.adultAge {
				continue
			}
			f := w.Households[w.People[id].Household].Floor - 1
			v[f] += x(w.Beliefs, id)
			adults[f]++
		}
		for i, n := range adults {
			if n > 0 {
				v[i] /= float64(n)
			}
		}
	}
}
