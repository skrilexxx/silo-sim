// Package rebellion is grievance and rebellion (ODD §6.14, draft).
package rebellion

import (
	"fmt"
	"math"
	"slices"

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

	threshold, k float64 // T and k (Epstein 2002)
	violent      float64 // T_v
	prev         []world.RebelStage
	def          config.Rebellion // the definitions of unrest and rebellion (ODD §1.2)
	officers     []int            // Judicial's and the Sheriff's department indexes
	courts       []int            // Judicial's office floors
	reach        []bool           // per floor: connected to a court today
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
	m.threshold, m.k, m.violent = cfg.Rebellion.ActiveThreshold, cfg.Rebellion.ArrestConstant, cfg.Rebellion.ViolentThreshold
	m.def = cfg.Rebellion
	for d, dep := range w.Depts {
		if dep.Name == "judicial" || dep.Name == "sheriff" {
			m.officers = append(m.officers, d)
		}
		if dep.Name == "judicial" {
			m.courts = w.Space.DeptFloors[d] // control checked that there are some
		}
	}
	m.reach = make([]bool, len(w.Floors))
	w.Rebellion = &world.Rebellion{LastAction: world.None, UnrestStart: world.None, RebellionStart: world.None}
	m.newcomers(w)
	m.count(w)
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
	m.stages(w)
	m.track(w, m.activeShare(w), m.action(w))
	m.count(w)
	return nil
}

func (m *Module) newcomers(w *world.World) {
	b := w.Rebellion
	for len(b.LossDay) < len(w.People) {
		b.LossDay, b.H, b.G = append(b.LossDay, world.None), append(b.H, 0), append(b.G, 0)
		b.Stage = append(b.Stage, world.Calm)
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

// ODD §6.14, Arrest estimate and Stages. An eligible adult is active
// when G − N > T, with net risk N = R × (1 − (1 − P)(1 − F)) and P = 1 −
// exp(−k × O_z / (A_z + 1)) from yesterday's counts in the zone of their
// day location and today's officers. An active adult joins when at least
// a share R of their ties were active or beyond yesterday, and a joined
// one is violent when G − N > T_v. Nothing is drawn and nobody's stage
// depends on another's today, so the visiting order doesn't matter. Each
// violent person commits the offence violence, every other active one
// organise.
func (m *Module) stages(w *world.World) {
	b, c := w.Rebellion, w.Control
	m.reachable(w)
	officers := m.onDuty(w)
	reach := 0
	for _, n := range b.Reach {
		reach += n
	}
	var p [world.NumZones]float64 // P on floors connected to a court; 0 elsewhere
	for z := range p {
		if reach > 0 {
			patrol := float64(officers) * float64(b.Reach[z]) / float64(reach)
			p[z] = 1 - math.Exp(-m.k*patrol/float64(b.Actives[z]+1))
		}
	}
	m.prev = append(m.prev[:0], b.Stage...)
	for _, id := range w.Alive {
		b.Stage[id] = world.Calm
		floor := w.Space.Where(w, id)
		if b.G[id] <= m.threshold || floor == 0 || !m.eligible(w, id) {
			continue // G − N > T needs G > T, as N ≥ 0; children have G = 0
		}
		pz := 0.0
		if m.reach[floor-1] {
			pz = p[w.Floors[floor-1].Zone]
		}
		r := w.Beliefs.Risk[id]
		n := r * (1 - (1-pz)*(1-c.Fear))
		if b.G[id]-n <= m.threshold {
			continue
		}
		b.Stage[id] = world.Active
		if m.joins(w, id, r) {
			b.Stage[id] = world.Joined
			if b.G[id]-n > m.violent {
				b.Stage[id] = world.Violent
			}
		}
		act := world.Organise
		if b.Stage[id] == world.Violent {
			act = world.Violence
		}
		c.Report(world.Act{Person: id, Other: world.None, Offence: act})
	}
}

// joins: at least a share r of the person's ties were active or beyond
// yesterday (ODD §6.14, Granovetter 1978). Someone with no ties never
// joins.
func (m *Module) joins(w *world.World, id world.PersonID, r float64) bool {
	ties := w.Network.TiesOf(w, id)
	if len(ties) == 0 {
		return false
	}
	out := 0
	for _, t := range ties {
		if int(t.Other) < len(m.prev) && m.prev[t.Other] != world.Calm {
			out++
		}
	}
	return float64(out) >= r*float64(len(ties))
}

// activeShare is today's share of adults active or beyond.
func (m *Module) activeShare(w *world.World) float64 {
	adults, active := 0, 0
	for _, id := range w.Alive {
		if w.Day-w.People[id].BirthDay >= m.adultAge {
			adults++
			if w.Rebellion.Stage[id] != world.Calm {
				active++
			}
		}
	}
	if adults == 0 {
		return 0
	}
	return float64(active) / float64(adults)
}

// action reports whether a collective action is under way today (ODD
// §6.14): a critical department on strike (T12.4), a fight or a held zone
// (T12.5). None exist yet.
func (m *Module) action(w *world.World) bool { return false }

// ODD §1.2, Definitions: unrest and rebellion. A rebellion starts on the
// last of RebellionDays days in a row with at least RebellionShare of
// adults active, if a collective action was under way on one of them; its
// start day is the first of them. Unrest starts on the last of UnrestDays
// days in a row with at least UnrestShare active, its start day the first
// of them. Both end after QuietDays days in a row below UnrestShare, their
// end day the first of those. An unrest spell during which a rebellion
// starts is that rebellion's run-up, not unrest, and no unrest is tracked
// while a rebellion runs (Adam, 2026-10-31).
func (m *Module) track(w *world.World, share float64, action bool) {
	b, d := w.Rebellion, m.def
	if share >= d.UnrestShare {
		b.Busy, b.Quiet = b.Busy+1, 0
	} else {
		b.Busy, b.Quiet = 0, b.Quiet+1
	}
	if share >= d.RebellionShare {
		b.Rising++
	} else {
		b.Rising = 0
	}
	if action {
		b.LastAction = w.Day
	}
	head := func(kind string) world.EventHeader { return world.EventHeader{Day: w.Day, Type: kind} }
	quietSince := w.Day - world.Tick(d.QuietDays) + 1
	switch {
	case b.RebellionStart != world.None:
		if b.Quiet >= d.QuietDays {
			w.Events = append(w.Events, world.RebellionEvent{EventHeader: head("rebellionEnd"),
				Start: b.RebellionStart, End: quietSince, Ending: "suppressed"})
			b.RebellionStart = world.None
		}
	case b.Rising >= d.RebellionDays && b.LastAction != world.None && b.LastAction > w.Day-world.Tick(d.RebellionDays):
		b.RebellionStart, b.UnrestStart = w.Day-world.Tick(d.RebellionDays)+1, world.None
		b.Rebellions++
		w.Events = append(w.Events, world.RebellionEvent{EventHeader: head("rebellionStart"), Start: b.RebellionStart})
	case b.UnrestStart != world.None:
		if b.Quiet >= d.QuietDays {
			w.Events = append(w.Events, world.UnrestEvent{EventHeader: head("unrest"), Start: b.UnrestStart, End: quietSince})
			b.UnrestStart = world.None
			b.UnrestSpells++
		}
	case b.Busy >= d.UnrestDays:
		b.UnrestStart = w.Day - world.Tick(d.UnrestDays) + 1
	}
}

// eligible: not an officer on duty, the Head of IT or held by Judicial
// (ODD §6.14, Stages).
func (m *Module) eligible(w *world.World, id world.PersonID) bool {
	if w.Control.IsHead(id) || w.Control.Held(id) {
		return false
	}
	return !m.officer(w, id)
}

// officer: a Judicial or Sheriff worker not off work (ODD §6.13).
func (m *Module) officer(w *world.World, id world.PersonID) bool {
	p := &w.People[id]
	return p.Stage == world.Worker && slices.Contains(m.officers, p.Department) && !w.Space.Off(id)
}

// onDuty counts the officers: the workers of Judicial and the Sheriff not
// off work (ODD §6.13, Arrests).
func (m *Module) onDuty(w *world.World) int {
	n := 0
	for _, d := range m.officers {
		n += w.Depts[d].Headcount
	}
	return n
}

// reachable marks the floors connected to a court today, walking out
// from each court until a broken segment (segment s joins floors s and
// s + 1). Asking Space.Connected for every floor and court costs a
// floor-length walk each.
func (m *Module) reachable(w *world.World) {
	clear(m.reach)
	broken := w.Space.Broken
	for _, c := range m.courts {
		m.reach[c-1] = true
		for f := c; f > 1 && !broken[f-2]; f-- {
			m.reach[f-2] = true
		}
		for f := c; f < len(m.reach) && !broken[f-1]; f++ {
			m.reach[f] = true
		}
	}
}

// count keeps tomorrow's zone counts (ODD §6.14, Step 9): adults active
// or beyond, and adults on floors connected to a court, by the zone of
// their day location.
func (m *Module) count(w *world.World) {
	b := w.Rebellion
	m.reachable(w)
	b.Actives, b.Reach = [world.NumZones]int{}, [world.NumZones]int{}
	for _, id := range w.Alive {
		floor := w.Space.Where(w, id)
		if floor == 0 || w.Day-w.People[id].BirthDay < m.adultAge {
			continue
		}
		z := w.Floors[floor-1].Zone
		if b.Stage[id] != world.Calm {
			b.Actives[z]++
		}
		if m.reach[floor-1] {
			b.Reach[z]++
		}
	}
}
