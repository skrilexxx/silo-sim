// Package beliefs keeps legitimacy, knowledge and memory (ODD §6.12,
// draft).
package beliefs

import (
	"fmt"
	"math"
	"math/bits"
	"slices"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/output"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

type Module struct {
	edu, eduWeight float64
	pass           float64      // π: chance a flag passes in a conversation, × (1 − L of the listener)
	forgetHeard    float64      // daily chance of losing a flag known as heard
	forgetLived    float64      // … as lived
	shadowAge      world.Tick   // in days: who sees a scripted reveal
	findChance     float64      // daily chance a hidden relic is found
	handChance     float64      // daily chance a holder hands a relic over
	cue            float64      // κ: chance a cue works
	used           []world.Tick // the last day a person's relics worked on them
	candidates     []world.PersonID
	seats          int              // N_F: Flamekeeper seats
	adultAge       world.Tick       // worker age in days: the base is fixed from then on
	young          []world.PersonID // alive and under worker age at the last step, in ID order

	hide        float64 // d: daily chance a known flag is hidden while dosed
	waterDays   int     // D_water: days a water dosing lasts
	it, medical int     // department indexes, or None

	tellers []world.PersonID // today's knowers, shuffled
	hiders  []world.PersonID // people with a hidden flag, and some who no longer have one
	fl      []world.Flag     // scratch
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
	m.findChance = 1 / (cfg.Beliefs.RelicHiddenYears * engine.DaysPerYear)
	m.handChance = 1 / (cfg.Beliefs.RelicKeptYears * engine.DaysPerYear)
	m.cue = cfg.Beliefs.CueChance
	m.seats = cfg.Beliefs.FlamekeeperSeats
	m.hide, m.waterDays = cfg.Beliefs.DoseHideChance, cfg.Beliefs.WaterDosingDays
	m.it, m.medical = world.None, world.None
	for d, dep := range w.Depts {
		switch dep.Name {
		case "it":
			m.it = d
		case "medical":
			m.medical = d
		}
	}
	w.Beliefs = &world.Beliefs{Cost: cfg.Beliefs.LegitimacyCostPerFlag, Water: world.None}
	m.newcomers(w, r)
	// ODD §6.12, Relics: on day 1, each hidden on a floor drawn uniformly.
	for range cfg.Beliefs.Relics {
		w.Beliefs.Relics = append(w.Beliefs.Relics, world.Relic{Floor: r.IntRange(1, len(w.Floors)), Holder: world.None})
	}
	if err := m.founders(w, r); err != nil {
		return err
	}
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

// EventTypes: ODD §6.12, Scripted events. Both fire just before the
// beliefs step.
//   - revealHistory stands in for a crushed rebellion everyone saw:
//     everyone alive aged 14 or more and not dosed learns the history as
//     lived, passed on from tomorrow like anything learned today.
//   - waterDosing puts Vitamin D+ in the water for D_water days from today,
//     dosing everyone alive and everyone born while it lasts, except those
//     the Flamekeepers shield if they are warned today.
func (m *Module) EventTypes() []engine.EventType {
	return []engine.EventType{
		{Name: "revealHistory", Phase: engine.PhaseBeliefs, Fire: func(w *world.World, _ *rng.Stream) error {
			for _, id := range w.Alive {
				// Age first: newborns have no beliefs until the step.
				if w.Day-w.People[id].BirthDay >= m.shadowAge && !w.Beliefs.Dosed(w, id) {
					m.teach(w.Beliefs, id, world.FlagHistory, true)
				}
			}
			return nil
		}},
		{Name: "waterDosing", Phase: engine.PhaseBeliefs, Fire: func(w *world.World, _ *rng.Stream) error {
			w.Beliefs.Water = max(w.Beliefs.Water, w.Day+world.Tick(m.waterDays)-1)
			w.Beliefs.Warned = m.warned(w)
			return nil
		}},
	}
}

// ODD §6.12 / §3 step 10.7.
func (m *Module) step(w *world.World, r *rng.Stream) error {
	m.newcomers(w, r)
	if w.Day%engine.DaysPerMonth == 0 {
		m.upbringing(w)
	}
	m.inherit(w, r)
	m.forget(w, r)
	m.relics(w, r)
	m.converse(w, r)
	m.seat(w, r)
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
		b.Hidden = append(b.Hidden, 0)
		b.DosedUntil = append(b.DosedUntil, world.None)
		b.Risk = append(b.Risk, r.Float64())
		m.young = append(m.young, world.PersonID(id))
		m.fresh = append(m.fresh, 0)
		m.used = append(m.used, -1)
		b.IsMember = append(b.IsMember, false)
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

// ODD §6.12, Relics: when a holder dies, each relic goes to a living
// member of their household aged 14 or more, drawn uniformly; with none,
// it stays hidden in the flat, on its floor.
func (m *Module) inherit(w *world.World, r *rng.Stream) {
	for i := range w.Beliefs.Relics {
		rel := &w.Beliefs.Relics[i]
		if rel.Holder == world.None || w.People[rel.Holder].Alive {
			continue
		}
		m.candidates = m.candidates[:0]
		for _, o := range w.Households[w.People[rel.Holder].Household].Members {
			if w.People[o].Alive && m.old(w, o) {
				m.candidates = append(m.candidates, o)
			}
		}
		rel.Holder = m.pick(r)
	}
}

// ODD §6.12, Relics: hidden relics are found by someone on their floor
// today; holders hand relics over to someone they talk with today; and
// relics work on their holders, once a day however many they hold. A
// held relic lies in its holder's flat.
func (m *Module) relics(w *world.World, r *rng.Stream) {
	b, n := w.Beliefs, w.Network
	for i := range b.Relics {
		rel := &b.Relics[i]
		switch {
		case rel.Holder == world.None:
			if r.Float64() >= m.findChance {
				continue
			}
			m.candidates = m.candidates[:0]
			for _, id := range w.Alive {
				if w.Space.Where(w, id) == rel.Floor && m.old(w, id) {
					m.candidates = append(m.candidates, id)
				}
			}
			rel.Holder = m.pick(r)
		case r.Float64() < m.handChance:
			m.candidates = m.candidates[:0]
			for _, t := range n.TiesOf(w, rel.Holder) {
				if m.old(w, t.Other) && n.Talk(w, rel.Holder, t.Other, t.Kind) {
					m.candidates = append(m.candidates, t.Other)
				}
			}
			if to := m.pick(r); to != world.None {
				if w.Control != nil { // ODD §6.13: a handover is an act by both
					w.Control.Report(world.Act{Person: rel.Holder, Other: to, Offence: world.TradeRelic})
				}
				rel.Holder = to
			}
		}
		if rel.Holder == world.None {
			continue
		}
		rel.Floor = w.Households[w.People[rel.Holder].Household].Floor
		if m.used[rel.Holder] == w.Day {
			continue
		}
		m.used[rel.Holder] = w.Day
		m.cueByRelic(w, r, rel.Holder)
	}
}

// cueByRelic: ODD §6.12, Relics and Vitamin D+. A relic works on a holder
// who isn't dosed like a cue: each hidden flag comes back with chance κ,
// and the history, if they neither know it nor have it hidden, is learned
// as lived with chance κ.
func (m *Module) cueByRelic(w *world.World, r *rng.Stream, id world.PersonID) {
	b := w.Beliefs
	if b.Dosed(w, id) || w.Control.Held(id) { // ODD §6.13: the relic is at home
		return
	}
	m.fl = flagsOf(b.Hidden[id], m.fl[:0])
	for _, f := range m.fl {
		if r.Float64() < m.cue {
			m.teach(b, id, f, false) // comes back as heard or lived, as before
		}
	}
	if (b.Known[id] | b.Hidden[id]).Has(world.FlagHistory) || r.Float64() >= m.cue {
		return
	}
	m.teach(b, id, world.FlagHistory, true)
}

// warned: ODD §6.12, Warning the Flamekeepers. On the day a water dosing
// starts, the group is warned if a member works in IT or Medical, or has a
// strong tie to a worker or shadow there who knows the history.
func (m *Module) warned(w *world.World) bool {
	b, n := w.Beliefs, w.Network
	for _, mem := range b.Members {
		if !w.People[mem].Alive { // died today, before the step drops them
			continue
		}
		if m.insider(w, mem) {
			return true
		}
		for _, t := range n.TiesOf(w, mem) {
			if !t.Kind.Weak() && m.insider(w, t.Other) && b.Known[t.Other].Has(world.FlagHistory) {
				return true
			}
		}
	}
	return false
}

// insider reports whether a person works in IT or Medical, as a worker or
// shadow: the few who know what Vitamin D+ is. The Head of IT never warns
// (ODD §6.13).
func (m *Module) insider(w *world.World, id world.PersonID) bool {
	p := &w.People[id]
	d := p.Department
	return (p.Stage == world.Worker || p.Stage == world.Shadow) && d != world.None && (d == m.it || d == m.medical) && !w.Control.IsHead(id)
}

// ODD §6.12, Flamekeepers: on day 1, N_F adults drawn uniformly take the
// seats and know the history as lived.
func (m *Module) founders(w *world.World, r *rng.Stream) error {
	m.candidates = m.candidates[:0]
	for _, id := range w.Alive {
		if w.Day-w.People[id].BirthDay >= m.adultAge {
			m.candidates = append(m.candidates, id)
		}
	}
	if m.seats > len(m.candidates) {
		return fmt.Errorf("beliefs: %d Flamekeeper seats, but only %d adults on day 1", m.seats, len(m.candidates))
	}
	for range m.seats {
		id := m.take(r)
		m.join(w, id)
		m.teach(w.Beliefs, id, world.FlagHistory, true)
	}
	return nil
}

// ODD §6.12, Flamekeepers: members who died or no longer know the history
// leave their seats; each empty seat goes to an adult who knows the
// history, isn't a member, and talks today with a member over a strong
// tie, drawn uniformly from all of them. With nobody like that, the seat
// stays empty; with no members left, the group is gone for good.
func (m *Module) seat(w *world.World, r *rng.Stream) {
	b, n := w.Beliefs, w.Network
	kept := b.Members[:0]
	for _, id := range b.Members {
		if w.People[id].Alive && b.Known[id].Has(world.FlagHistory) {
			kept = append(kept, id)
		} else {
			b.IsMember[id] = false
		}
	}
	b.Members = kept
	if len(b.Members) == m.seats {
		return
	}
	m.candidates = m.candidates[:0]
	for _, mem := range b.Members {
		for _, t := range n.TiesOf(w, mem) {
			c := t.Other
			if t.Kind.Weak() || b.IsMember[c] || !b.Known[c].Has(world.FlagHistory) || w.Control.IsHead(c) ||
				w.Day-w.People[c].BirthDay < m.adultAge || slices.Contains(m.candidates, c) || !n.Talk(w, mem, c, t.Kind) {
				continue
			}
			m.candidates = append(m.candidates, c)
		}
	}
	for len(b.Members) < m.seats && len(m.candidates) > 0 {
		m.join(w, m.take(r))
	}
}

// take removes one candidate, drawn uniformly, and returns them.
func (m *Module) take(r *rng.Stream) world.PersonID {
	i := r.IntRange(0, len(m.candidates)-1)
	id := m.candidates[i]
	m.candidates[i] = m.candidates[len(m.candidates)-1]
	m.candidates = m.candidates[:len(m.candidates)-1]
	return id
}

func (m *Module) join(w *world.World, id world.PersonID) {
	w.Beliefs.Members = append(w.Beliefs.Members, id)
	w.Beliefs.IsMember[id] = true
}

// old reports whether a person is of shadowing age (14) or more.
func (m *Module) old(w *world.World, id world.PersonID) bool {
	return w.Day-w.People[id].BirthDay >= m.shadowAge
}

// pick draws one of the candidates uniformly, or None if there are none.
func (m *Module) pick(r *rng.Stream) world.PersonID {
	if len(m.candidates) == 0 {
		return world.None
	}
	return m.candidates[r.IntRange(0, len(m.candidates)-1)]
}

// ODD §6.12, Forgetting: each known flag is lost for good with a daily
// chance set by its half-life, heard or lived. The knowers list loses the
// dead and those who now know nothing, in its own (deterministic) order.
// Losses are rare, so they are drawn by skipping (skip) at the higher of
// the two chances and kept, for a flag with the lower one, in proportion.
//
// Then Vitamin D+: for a dosed knower, each flag still known is hidden
// with chance d, the Flamekeepers' history too.
func (m *Module) forget(w *world.World, r *rng.Stream) {
	b := w.Beliefs
	most := max(m.forgetHeard, m.forgetLived)
	sk, dose := newSkip(r, most), newSkip(r, m.hide)
	// The list of people with hidden flags is cleaned before any are hidden
	// today, so someone who recalled everything and is hidden again is
	// listed once.
	hid := m.hiders[:0]
	for _, h := range m.hiders {
		if w.People[h].Alive && b.Hidden[h] != 0 {
			hid = append(hid, h)
		}
	}
	m.hiders = hid
	keep := b.Knowers[:0]
	for _, id := range b.Knowers {
		if !w.People[id].Alive {
			continue
		}
		if w.Control.IsHead(id) { // ODD §6.13: never forgets, never dosed
			keep = append(keep, id)
			continue
		}
		for known := b.Known[id]; known != 0; known &= known - 1 {
			f := world.Flag(bits.TrailingZeros8(uint8(known)))
			if f == world.FlagHistory && b.IsMember[id] || !sk.land(r) { // Flamekeepers never forget it
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
		if b.Dosed(w, id) {
			for known := b.Known[id]; known != 0; known &= known - 1 {
				if dose.land(r) && b.Hide(id, world.Flag(bits.TrailingZeros8(uint8(known)))) {
					m.hiders = append(m.hiders, id)
				}
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
// learn. A dosed listener learns nothing, and one who has the flag hidden
// may recall it instead (recall). Only knowers are visited, so the cost
// follows them, not the population. With the control module on, each
// telling (the slot lands, the listener doesn't know the flag, and the two
// talk) is reported as an act, believed or not (ODD §6.13), and the Head of
// IT tells nothing. Over a weak tie the teller dares to speak only with
// chance 1 − F, the silo's fear; one who doesn't commits no act.
//
// The draws are made in the cheapest order, with the same chances. Every
// pair of a teller's tie and a flag they tell is a slot; a slot is a real
// chance to pass the flag when the listener doesn't know it. First, which
// slots would pass at π is drawn by skipping (almost none do), so the
// step goes straight to them and a teller none of whose slots land is
// passed over without reading their ties; then, for a landed slot that is
// real, whether it passes at 1 − L of the listener; and only then whether
// the two talk today. Asking every tie whether it was used cost 1 ms a day
// with 80% of the silo knowing a flag (T10.3).
func (m *Module) converse(w *world.World, r *rng.Stream) {
	b, n := w.Beliefs, w.Network
	m.tellers = append(m.tellers[:0], b.Knowers...)
	if c := w.Control; c != nil { // ODD §6.13: the Head of IT never tells
		m.tellers = slices.DeleteFunc(m.tellers, c.IsHead)
	}
	r.Shuffle(len(m.tellers), func(i, j int) { m.tellers[i], m.tellers[j] = m.tellers[j], m.tellers[i] })

	fear := 0.0
	if w.Control != nil {
		fear = w.Control.Fear
	}
	sk := newSkip(r, m.pass)
	for _, k := range m.tellers {
		m.fl = flagsOf(b.Known[k]&^m.fresh[k], m.fl[:0])
		fl := m.fl
		ties := n.TiesOf(w, k)
		slots := len(fl) * len(ties)
		for s := sk.next(r, slots); s >= 0; s = next(r, &sk, s, slots) {
			t, f := ties[s/len(fl)], fl[s%len(fl)]
			j := t.Other
			if b.Known[j].Has(f) {
				continue
			}
			if fear > 0 && t.Kind.Weak() && r.Float64() < fear { // ODD §6.12: fear silences talk with acquaintances
				continue
			}
			pass := !b.Hidden[j].Has(f) && !b.Dosed(w, j) && r.Float64() < 1-b.L(j)
			if !pass && w.Control == nil || !n.Talk(w, k, j, t.Kind) {
				continue
			}
			if w.Control != nil { // ODD §6.13: a telling is an act, believed or not
				w.Control.Report(world.Act{Person: k, Other: world.None, Offence: world.SpeakForbidden})
			}
			if pass {
				m.learn(b, j, f)
			}
		}
	}
	m.recall(w, r)
	for _, id := range m.learned {
		m.fresh[id] = 0
	}
	m.learned = m.learned[:0]
}

// ODD §6.12, Vitamin D+: once the dose has ended, a hidden flag comes back,
// known as before, with chance κ in each conversation with someone who
// knew it before today: the person's own memory, so their L doesn't block
// it. The people with hidden flags are visited in the order they first
// had one hidden, and the draws skip to the slots (tie and hidden flag)
// that land, as in converse.
func (m *Module) recall(w *world.World, r *rng.Stream) {
	b, n := w.Beliefs, w.Network
	sk := newSkip(r, m.cue)
	for _, h := range m.hiders {
		if b.Dosed(w, h) {
			continue
		}
		m.fl = flagsOf(b.Hidden[h], m.fl[:0])
		fl := m.fl
		ties := n.TiesOf(w, h)
		slots := len(fl) * len(ties)
		for s := sk.next(r, slots); s >= 0; s = next(r, &sk, s, slots) {
			t, f := ties[s/len(fl)], fl[s%len(fl)]
			if !b.Hidden[h].Has(f) || !(b.Known[t.Other] &^ m.fresh[t.Other]).Has(f) || w.Control.IsHead(t.Other) || !n.Talk(w, h, t.Other, t.Kind) {
				continue
			}
			m.teach(b, h, f, false)
		}
	}
}

// flagsOf lists the flags in a set, appended to out.
func flagsOf(set world.Flags, out []world.Flag) []world.Flag {
	for ; set != 0; set &= set - 1 {
		out = append(out, world.Flag(bits.TrailingZeros8(uint8(set))))
	}
	return out
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

// next returns how many of the next n items miss before one lands, and
// moves past that one; or −1, moving past all n, if none lands.
func (s *skip) next(r *rng.Stream, n int) int {
	if s.left >= n {
		s.left -= n
		return -1
	}
	i := s.left
	s.draw(r)
	return i
}

// next, for a loop over slots 0 … n−1 whose slot at landed has just been
// handled, returns the next slot that lands, or −1.
func next(r *rng.Stream, sk *skip, landed, n int) int {
	i := sk.next(r, n-landed-1)
	if i < 0 {
		return -1
	}
	return landed + 1 + i
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
