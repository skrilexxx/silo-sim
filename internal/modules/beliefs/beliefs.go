// Package beliefs keeps legitimacy, knowledge and memory (ODD §6.12,
// draft).
package beliefs

import (
	"fmt"
	"math"
	"math/bits"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/output"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

type Module struct {
	edu, eduWeight float64
	pass           float64          // π: chance a flag passes in a conversation, × (1 − L of the listener)
	forgetHeard    float64          // daily chance of losing a flag known as heard
	forgetLived    float64          // … as lived
	shadowAge      world.Tick       // in days: who sees a scripted reveal
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
	m.forgetHeard = dailyLoss(cfg.Beliefs.HalfLifeHeardYears)
	m.forgetLived = dailyLoss(cfg.Beliefs.HalfLifeLivedYears)
	m.shadowAge = world.Tick(cfg.Work.ShadowAge * engine.DaysPerYear)
	m.adultAge = world.Tick(cfg.Work.WorkerAge * engine.DaysPerYear)
	w.Beliefs = &world.Beliefs{Cost: cfg.Beliefs.LegitimacyCostPerFlag}
	return m.step(w, r)
}

// dailyLoss is the daily chance of losing a flag with a half-life of
// years (ODD §6.12, Forgetting): 1 − 2^(−1/(365 T)).
func dailyLoss(years float64) float64 {
	return 1 - math.Exp2(-1/(years*engine.DaysPerYear))
}

func (m *Module) Steps() []engine.Step {
	return []engine.Step{{Phase: engine.PhaseBeliefs, Run: m.step}}
}

// EventTypes: ODD §6.12, Scripted events. revealHistory stands in for a
// crushed rebellion everyone saw: everyone alive aged 14 or more learns the
// history as lived. It fires just before the beliefs step, and, like
// anything learned today, is passed on from tomorrow.
func (m *Module) EventTypes() []engine.EventType {
	return []engine.EventType{{Name: "revealHistory", Phase: engine.PhaseBeliefs, Fire: func(w *world.World, _ *rng.Stream) error {
		for _, id := range w.Alive {
			if w.Day-w.People[id].BirthDay >= m.shadowAge { // checked first: newborns have no beliefs until the step
				m.teach(w.Beliefs, id, world.FlagHistory, true)
			}
		}
		return nil
	}}}
}

// ODD §6.12 / §3 step 10.7.
func (m *Module) step(w *world.World, r *rng.Stream) error {
	m.newcomers(w, r)
	if w.Day%engine.DaysPerMonth == 0 {
		m.upbringing(w)
	}
	m.forget(w, r)
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
		b.Lived = append(b.Lived, 0)
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

// ODD §6.12, Forgetting: each known flag is lost for good with a daily
// chance set by its half-life, heard or lived. The knowers list loses the
// dead and those who now know nothing, in its own (deterministic) order.
// Losses are rare, so they are drawn by skipping (skip) at the higher of
// the two chances and kept, for a flag with the lower one, in proportion.
func (m *Module) forget(w *world.World, r *rng.Stream) {
	b := w.Beliefs
	most := max(m.forgetHeard, m.forgetLived)
	sk := newSkip(r, most)
	keep := b.Knowers[:0]
	for _, id := range b.Knowers {
		if !w.People[id].Alive {
			continue
		}
		for known := b.Known[id]; known != 0; known &= known - 1 {
			f := world.Flag(bits.TrailingZeros8(uint8(known)))
			if !sk.land(r) {
				continue
			}
			p := m.forgetHeard
			if b.Lived[id].Has(f) {
				p = m.forgetLived
			}
			if p == most || r.Float64() < p/most {
				b.Forget(id, f)
			}
		}
		if b.Known[id] != 0 {
			keep = append(keep, id)
		}
	}
	b.Knowers = keep
}

// ODD §6.12, Telling and believing: the day's knowers, in shuffled order,
// ask of each tie whether it is used today (ODD §6.11); in a conversation
// each flag the teller knew before today and the listener doesn't know
// passes with chance π × (1 − L of the listener), whose L drops as they
// learn. Only knowers are visited, so the cost follows them, not the
// population.
//
// The draws are made in the cheapest order, with the same chances. Every
// pair of a teller's tie and a flag they tell is a slot; a slot is a real
// chance to pass the flag when the listener doesn't know it. First, which
// slots would pass at π is drawn by skipping (almost none do), so a
// teller none of whose slots land is passed over without reading their
// ties; then, for a landed slot that is real, whether it passes at 1 − L
// of the listener; and only then whether the two talk today. Asking every
// tie whether it was used cost 1 ms a day with 80% of the silo knowing a
// flag (T10.3).
func (m *Module) converse(w *world.World, r *rng.Stream) {
	b, n := w.Beliefs, w.Network
	m.tellers = append(m.tellers[:0], b.Knowers...)
	r.Shuffle(len(m.tellers), func(i, j int) { m.tellers[i], m.tellers[j] = m.tellers[j], m.tellers[i] })

	sk := newSkip(r, m.pass)
	for _, k := range m.tellers {
		tells := b.Known[k] &^ m.fresh[k]
		ties := n.TiesOf(w, k)
		if sk.miss(tells.Count() * len(ties)) {
			continue
		}
		for _, t := range ties {
			j := t.Other
			for f := range world.NumFlags {
				if !tells.Has(f) || !sk.land(r) || b.Known[j].Has(f) || r.Float64() >= 1-b.L(j) || !n.Talk(w, k, j, t.Kind) {
					continue
				}
				m.learn(b, j, f)
			}
		}
	}
	for _, id := range m.learned {
		m.fresh[id] = 0
	}
	m.learned = m.learned[:0]
}

// learn teaches a flag in a conversation.
func (m *Module) learn(b *world.Beliefs, id world.PersonID, f world.Flag) {
	m.teach(b, id, f, false)
}

// teach teaches a flag that can be passed on from tomorrow.
func (m *Module) teach(b *world.Beliefs, id world.PersonID, f world.Flag, lived bool) {
	b.Learn(id, f, lived)
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

// skip says which items of a run, each landing with chance p on its own,
// land. It draws the geometric gap to the next landing instead of one
// number per item: the same chances, far fewer draws when p is small.
type skip struct {
	logMiss float64 // ln(1 − p)
	left    int     // items still to miss before the next one lands
}

func newSkip(r *rng.Stream, p float64) skip {
	s := skip{logMiss: math.Log1p(-p)}
	s.draw(r)
	return s
}

// draw sets the gap: P(left ≥ n) = (1 − p)^n.
func (s *skip) draw(r *rng.Stream) {
	switch {
	case s.logMiss == 0: // p = 0: nothing lands
		s.left = math.MaxInt
	case math.IsInf(s.logMiss, -1): // p = 1: everything does
		s.left = 0
	default:
		s.left = int(math.Log1p(-r.Float64()) / s.logMiss)
	}
}

// miss reports whether none of the next n items lands, and if so skips
// them; otherwise it skips nothing.
func (s *skip) miss(n int) bool {
	if s.left < n {
		return false
	}
	s.left -= n
	return true
}

// land reports whether the next item lands.
func (s *skip) land(r *rng.Stream) bool {
	if s.left > 0 {
		s.left--
		return false
	}
	s.draw(r)
	return true
}
